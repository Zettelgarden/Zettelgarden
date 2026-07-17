package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Error definitions for epub import
var (
	ErrFileNotFound    = errors.New("file not found")
	ErrNotEpub         = errors.New("file is not an epub")
	ErrEpubParse       = errors.New("unable to parse epub file")
	ErrEpubImport      = errors.New("failed to import epub")
	ErrEpubNoChapters  = errors.New("no valid chapters found in epub")
	ErrEpubInvalid     = errors.New("invalid epub file format")
	ErrDownloadTimeout = errors.New("download timed out")
)

// maxConcurrentEntityProcessing limits how many chapters can be processed simultaneously
const maxConcurrentEntityProcessing = 5

// epubDownloadTimeout is the maximum time allowed for downloading an epub from S3
const epubDownloadTimeout = 5 * time.Minute

// epubMagicBytes are the first 4 bytes of a valid EPUB (ZIP file header)
var epubMagicBytes = []byte{0x50, 0x4B, 0x03, 0x04} // "PK\x03\x04"

// ImportEpubRequest represents the request body for epub import
type ImportEpubRequest struct {
	CardID string `json:"card_id"` // User-specified card ID for the book
}

// validateEpubHeader checks if the file starts with ZIP magic bytes (EPUB is a ZIP)
func validateEpubHeader(header []byte) error {
	if len(header) < 4 {
		return ErrEpubInvalid
	}
	for i := 0; i < 4; i++ {
		if header[i] != epubMagicBytes[i] {
			return ErrEpubInvalid
		}
	}
	return nil
}

// ImportEpubRoute handles POST /files/{id}/import-epub
func (h *Handler) ImportEpubRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
		return
	}

	// Create context with timeout for the entire operation
	ctx, cancel := context.WithTimeout(r.Context(), epubDownloadTimeout)
	defer cancel()

	// Parse request body
	var req ImportEpubRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get file ID from URL
	fileIDStr := mux.Vars(r)["id"]
	fileID, err := strconv.Atoi(fileIDStr)
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	// Get file from database
	var s3Key, fileName, mimetype string
	err = h.DB.QueryRow(`
		SELECT path, name, type
		FROM files
		WHERE id = $1 AND user_id = $2
	`, fileID, userID).Scan(&s3Key, &fileName, &mimetype)

	if err == sql.ErrNoRows {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Failed to retrieve file %d for user %d: %v", fileID, userID, err)
		http.Error(w, "Failed to retrieve file", http.StatusInternalServerError)
		return
	}

	// Validate epub mimetype
	if mimetype != "application/epub+zip" {
		http.Error(w, "File is not an epub", http.StatusBadRequest)
		return
	}

	// Download epub from S3 to temp file with context
	tempPath, err := h.downloadEpubToTemp(ctx, s3Key, fileID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrDownloadTimeout) {
			log.Printf("Epub download timed out for file %d, user %d", fileID, userID)
			http.Error(w, "Download timed out", http.StatusGatewayTimeout)
		} else {
			log.Printf("Failed to download epub from S3 for file %d: %v", fileID, err)
			http.Error(w, "Failed to retrieve epub file", http.StatusInternalServerError)
		}
		return
	}
	defer os.Remove(tempPath) // Clean up temp file

	// Validate epub structure before parsing
	if err := h.validateEpubFile(tempPath); err != nil {
		log.Printf("Invalid epub file %d for user %d: %v", fileID, userID, err)
		http.Error(w, "Invalid epub file format", http.StatusUnprocessableEntity)
		return
	}

	// Check if context is still valid before expensive parsing
	select {
	case <-ctx.Done():
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
		return
	default:
	}

	// Parse the epub
	metadata, chapters, err := services.ParseEpub(tempPath)
	if err != nil {
		log.Printf("Failed to parse epub for file %d, user %d: %v", fileID, userID, err)
		// Return generic error to client, log details server-side
		if errors.Is(err, services.ErrNoValidChapters) {
			http.Error(w, "No valid chapters found in epub", http.StatusUnprocessableEntity)
		} else {
			http.Error(w, "Unable to parse epub file", http.StatusUnprocessableEntity)
		}
		return
	}

	// Create cards in a transaction
	response, err := h.createEpubCards(ctx, userID, fileID, req.CardID, metadata, chapters, fileName)
	if err != nil {
		log.Printf("Failed to create cards for file %d, user %d: %v", fileID, userID, err)
		http.Error(w, "Failed to import epub", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// downloadEpubToTemp downloads an epub from S3 to a temporary file with context support
func (h *Handler) downloadEpubToTemp(ctx context.Context, s3Key string, fileID int) (string, error) {
	// Download epub from S3
	s3Output, err := h.downloadObject(h.Server.S3, s3Key, "")
	if err != nil {
		return "", fmt.Errorf("failed to download from S3: %w", err)
	}
	defer s3Output.Body.Close()

	// Create temp file for epub
	tempFile, err := os.CreateTemp("", "epub-*.epub")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Write epub content to temp file with context awareness
	// Use a goroutine to allow context cancellation
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(tempFile, s3Output.Body)
		tempFile.Close()
		done <- err
	}()

	select {
	case <-ctx.Done():
		// Context cancelled - clean up and return
		os.Remove(tempPath)
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			os.Remove(tempPath)
			return "", fmt.Errorf("failed to write epub to temp file: %w", err)
		}
	}

	return tempPath, nil
}

// validateEpubFile performs basic validation on the epub file structure
func (h *Handler) validateEpubFile(tempPath string) error {
	file, err := os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Check ZIP magic bytes (first 4 bytes)
	header := make([]byte, 4)
	n, err := file.Read(header)
	if err != nil || n < 4 {
		return ErrEpubInvalid
	}

	if err := validateEpubHeader(header); err != nil {
		return err
	}

	// Additional validation: check for mimetype file (EPUB requirement)
	// The mimetype file should be the first file in the ZIP and uncompressed
	// For simplicity, we just check magic bytes. Full validation would require
	// parsing the ZIP structure to find META-INF/container.xml

	return nil
}

// createEpubCards creates the parent book card and child chapter cards
func (h *Handler) createEpubCards(ctx context.Context, userID int, fileID int, cardID string, metadata models.EpubMetadata, chapters []models.EpubChapter, filename string) (models.ImportEpubResponse, error) {
	var response models.ImportEpubResponse

	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		return response, ctx.Err()
	default:
	}

	tx, err := h.DB.Begin()
	if err != nil {
		return response, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use filename as fallback title
	title := metadata.Title
	if title == "" {
		title = strings.TrimSuffix(filename, ".epub")
	}

	// Generate card_id from title if not provided
	if cardID == "" {
		var err error
		// Use transaction for consistency
		cardID, err = services.GetNextRootCardID(tx, userID)
		if err != nil {
			return response, fmt.Errorf("failed to generate card ID: %w", err)
		}
	}

	// Check for cancellation before creating cards
	select {
	case <-ctx.Done():
		return response, ctx.Err()
	default:
	}

	// Create parent book card using CreateCard - pass transaction for atomicity
	bookBody := formatBookCardBody(metadata, chapters)
	parentCard, err := services.CreateCard(tx, userID, models.EditCardParams{
		CardID: cardID,
		Title:  title,
		Body:   bookBody,
	})
	if err != nil {
		return response, fmt.Errorf("failed to create book card: %w", err)
	}

	response.ParentCardID = parentCard.ID

	// Update file record to link to parent card
	_, err = tx.Exec(`
		UPDATE files SET card_pk = $1 WHERE id = $2
	`, parentCard.ID, fileID)
	if err != nil {
		return response, fmt.Errorf("failed to link file to parent card: %w", err)
	}

	// Create child chapter cards - all within the same transaction
	childCards := make([]models.Card, 0, len(chapters))
	for i, chapter := range chapters {
		// Check for cancellation periodically
		if i%10 == 0 {
			select {
			case <-ctx.Done():
				return response, ctx.Err()
			default:
			}
		}

		// Generate card_id for chapter (parent.cardID + incrementing number)
		chapterCardID := fmt.Sprintf("%s.%d", cardID, i+1)

		childCard, err := services.CreateCard(tx, userID, models.EditCardParams{
			CardID: chapterCardID,
			Title:  chapter.Title,
			Body:   chapter.Body,
		})
		if err != nil {
			// Track failed chapters instead of silently skipping
			response.FailedChapters = append(response.FailedChapters, models.FailedChapter{
				Index: i,
				Title: chapter.Title,
				Error: err.Error(),
			})
			log.Printf("Failed to create chapter card %d (%s) for user %d: %v", i, chapter.Title, userID, err)
			continue
		}

		response.ChildCardIDs = append(response.ChildCardIDs, childCard.ID)
		childCards = append(childCards, childCard)
	}

	if err := tx.Commit(); err != nil {
		return response, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Trigger summarization for each child card (async) with rate limiting and panic recovery
	// Use a semaphore to limit concurrent processing
	semaphore := make(chan struct{}, maxConcurrentEntityProcessing)
	var wg sync.WaitGroup

	for _, card := range childCards {
		wg.Add(1)
		go func(c models.Card) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic recovered in ProcessEntitiesAndFacts for card %d, user %d: %v", c.ID, userID, r)
				}
			}()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			h.ProcessEntitiesAndFacts(userID, c)
		}(card)
	}

	// Wait for all goroutines to complete (or at least acquire semaphore)
	// Note: We don't block the response on this, but we limit concurrency
	go func() {
		wg.Wait()
	}()

	response.Metadata = metadata
	return response, nil
}

// formatBookCardBody creates the markdown body for the book card
func formatBookCardBody(metadata models.EpubMetadata, chapters []models.EpubChapter) string {
	var body strings.Builder

	// Add metadata block
	if metadata.Author != "" || metadata.Publisher != "" || metadata.Year != "" {
		body.WriteString("> ")
		parts := []string{}
		if metadata.Author != "" {
			parts = append(parts, fmt.Sprintf("Author: %s", metadata.Author))
		}
		if metadata.Publisher != "" {
			parts = append(parts, fmt.Sprintf("Publisher: %s", metadata.Publisher))
		}
		if metadata.Year != "" {
			parts = append(parts, fmt.Sprintf("Year: %s", metadata.Year))
		}
		body.WriteString(strings.Join(parts, " | "))
		body.WriteString("\n\n")
	}

	// Add description
	if metadata.Description != "" {
		body.WriteString(metadata.Description)
		body.WriteString("\n\n")
	}

	// Add chapter list
	if len(chapters) > 0 {
		body.WriteString("## Chapters\n")
		for _, chapter := range chapters {
			body.WriteString(fmt.Sprintf("- %s\n", chapter.Title))
		}
	}

	return body.String()
}
