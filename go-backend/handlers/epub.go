package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// ImportEpubRequest represents the request body for epub import
type ImportEpubRequest struct {
	CardID string `json:"card_id"` // User-specified card ID for the book
}

// ImportEpubRoute handles POST /files/{id}/import-epub
func (h *Handler) ImportEpubRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
		return
	}

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
		http.Error(w, "Failed to retrieve file", http.StatusInternalServerError)
		return
	}

	// Validate epub mimetype
	if mimetype != "application/epub+zip" {
		http.Error(w, "File is not an epub", http.StatusBadRequest)
		return
	}

	// Download epub from S3 to temp file
	s3Output, err := h.downloadObject(h.Server.S3, s3Key, "")
	if err != nil {
		log.Printf("Failed to download epub from S3: %v", err)
		http.Error(w, "Failed to retrieve epub file", http.StatusInternalServerError)
		return
	}
	defer s3Output.Body.Close()

	// Create temp file for epub
	tempFile, err := os.CreateTemp("", "epub-*.epub")
	if err != nil {
		log.Printf("Failed to create temp file: %v", err)
		http.Error(w, "Failed to process epub", http.StatusInternalServerError)
		return
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up temp file

	// Write epub content to temp file
	_, err = io.Copy(tempFile, s3Output.Body)
	tempFile.Close()
	if err != nil {
		log.Printf("Failed to write epub to temp file: %v", err)
		http.Error(w, "Failed to process epub", http.StatusInternalServerError)
		return
	}

	// Parse the epub
	metadata, chapters, err := services.ParseEpub(tempPath)
	if err != nil {
		log.Printf("Failed to parse epub: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse epub: %v", err), http.StatusInternalServerError)
		return
	}

	// Create cards in a transaction
	response, err := h.createEpubCards(userID, fileID, req.CardID, metadata, chapters, fileName)
	if err != nil {
		log.Printf("Failed to create cards: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create cards: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createEpubCards creates the parent book card and child chapter cards
func (h *Handler) createEpubCards(userID int, fileID int, cardID string, metadata models.EpubMetadata, chapters []models.EpubChapter, filename string) (models.ImportEpubResponse, error) {
	var response models.ImportEpubResponse

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
		cardID, err = services.GetNextRootCardID(h.DB, userID)
		if err != nil {
			return response, fmt.Errorf("failed to generate card ID: %w", err)
		}
	}

	// Create parent book card using CreateCard
	bookBody := formatBookCardBody(metadata, chapters)
	parentCard, err := services.CreateCard(h.DB, userID, models.EditCardParams{
		CardID: cardID,
		Title:  title,
		Body:  bookBody,
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
		log.Printf("Warning: Failed to link file to parent card: %v", err)
		// Continue anyway, cards were created
	}

	// Create child chapter cards
	childCards := make([]models.Card, 0, len(chapters))
	for i, chapter := range chapters {
		// Generate card_id for chapter (parent.cardID + incrementing number)
		chapterCardID := fmt.Sprintf("%s.%d", cardID, i+1)

		childCard, err := services.CreateCard(h.DB, userID, models.EditCardParams{
			CardID: chapterCardID,
			Title:  chapter.Title,
			Body:   chapter.Body,
		})
		if err != nil {
			log.Printf("Failed to create chapter card %d: %v", i, err)
			continue
		}

		response.ChildCardIDs = append(response.ChildCardIDs, childCard.ID)
		childCards = append(childCards, childCard)
	}

	if err := tx.Commit(); err != nil {
		return response, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Trigger summarization for each child card (async)
	for _, card := range childCards {
		go h.ProcessEntitiesAndFacts(userID, card)
	}

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
