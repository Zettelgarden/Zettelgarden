package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// ImportEpubRoute handles POST /files/{id}/import-epub
func (h *Handler) ImportEpubRoute(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(w, r)
	if !ok {
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
	var filePath, fileName, mimetype string
	err = h.DB.QueryRow(`
		SELECT path, name, mimetype
		FROM files
		WHERE id = $1 AND user_id = $2
	`, fileID, userID).Scan(&filePath, &fileName, &mimetype)

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

	// Parse the epub
	metadata, chapters, err := services.ParseEpub(filePath)
	if err != nil {
		log.Printf("Failed to parse epub: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse epub: %v", err), http.StatusInternalServerError)
		return
	}

	// Create cards in a transaction
	response, err := h.createEpubCards(userID, metadata, chapters, fileName)
	if err != nil {
		log.Printf("Failed to create cards: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create cards: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createEpubCards creates the parent book card and child chapter cards
func (h *Handler) createEpubCards(userID int, metadata models.EpubMetadata, chapters []models.EpubChapter, filename string) (models.ImportEpubResponse, error) {
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

	// Create parent book card
	bookBody := formatBookCardBody(metadata, chapters)
	var parentCardID int
	err = tx.QueryRow(`
		INSERT INTO cards (user_id, title, body, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`, userID, title, bookBody).Scan(&parentCardID)

	if err != nil {
		return response, fmt.Errorf("failed to create book card: %w", err)
	}

	response.ParentCardID = parentCardID

	// Create child chapter cards
	childCards := make([]models.Card, 0, len(chapters))
	for i, chapter := range chapters {
		var childCardID int
		err = tx.QueryRow(`
			INSERT INTO cards (user_id, title, body, parent_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			RETURNING id
		`, userID, chapter.Title, chapter.Body, parentCardID).Scan(&childCardID)

		if err != nil {
			log.Printf("Failed to create chapter card %d: %v", i, err)
			continue
		}

		response.ChildCardIDs = append(response.ChildCardIDs, childCardID)
		childCards = append(childCards, models.Card{
			ID:     childCardID,
			Title:  chapter.Title,
			Body:   chapter.Body,
			UserID: userID,
		})
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
