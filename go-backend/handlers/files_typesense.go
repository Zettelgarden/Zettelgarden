package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"time"

	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

// InitFilesCollection creates the files collection in Typesense if it doesn't exist
func (s *Handler) InitFilesCollection() error {
	if s.Server.TypesenseClient == nil {
		return fmt.Errorf("Typesense client not initialized")
	}

	collectionName := "files"

	// Check if collection already exists
	_, err := s.Server.TypesenseClient.Collection(collectionName).Retrieve(context.Background())
	if err == nil {
		log.Printf("Files collection already exists in Typesense")
		return nil
	}

	// Create the collection
	schema := &api.CollectionSchema{
		Name: collectionName,
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "file_id", Type: "int32"},
			{Name: "user_id", Type: "int32", Facet: pointer.True()},
			{Name: "name", Type: "string"},
			{Name: "description", Type: "string", Optional: pointer.True()},
			{Name: "extracted_text", Type: "string", Optional: pointer.True()},
			{Name: "tags", Type: "string[]", Optional: pointer.True(), Facet: pointer.True()},
			{Name: "card_pk", Type: "int32", Optional: pointer.True(), Facet: pointer.True()},
			{Name: "size", Type: "int32"},
			{Name: "content_type", Type: "string"},
			{Name: "created_at", Type: "int64"},
		},
	}

	_, err = s.Server.TypesenseClient.Collections().Create(context.Background(), schema)
	if err != nil {
		return fmt.Errorf("failed to create files collection: %w", err)
	}

	log.Printf("Created files collection in Typesense")
	return nil
}

// IndexFileInTypesense indexes a file in Typesense
func (s *Handler) IndexFileInTypesense(ctx context.Context, fileID int) error {
	if s.Server.TypesenseClient == nil {
		return fmt.Errorf("Typesense client not initialized")
	}

	// Get file data
	var description, extractedText sql.NullString
	var cardPK sql.NullInt32
	var file struct {
		ID          int
		UserID      int
		Name        string
		ContentType string
		Size        int
		CreatedAt   time.Time
	}

	query := `
		SELECT id, user_id, name, type, size, created_at,
		       description, extracted_text, card_pk
		FROM files WHERE id = $1
	`
	err := s.GetDB().QueryRowContext(ctx, query, fileID).Scan(
		&file.ID, &file.UserID, &file.Name, &file.ContentType, &file.Size, &file.CreatedAt,
		&description, &extractedText, &cardPK,
	)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// Get tags for this file
	rows, err := s.GetDB().QueryContext(ctx, `
		SELECT t.name
		FROM file_tags t
		JOIN files_tags ft ON t.id = ft.tag_id
		WHERE ft.file_id = $1
	`, fileID)
	if err != nil {
		return fmt.Errorf("failed to get file tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err == nil {
			tags = append(tags, tag)
		}
	}

	// Prepare document
	cardPKValue := int32(-1)
	if cardPK.Valid {
		cardPKValue = cardPK.Int32
	}

	document := map[string]interface{}{
		"id":             fmt.Sprintf("file-%d", file.ID),
		"file_id":        int32(file.ID),
		"user_id":        int32(file.UserID),
		"name":           file.Name,
		"description":    description.String,
		"extracted_text": extractedText.String,
		"tags":           tags,
		"card_pk":        cardPKValue,
		"size":           int32(file.Size),
		"content_type":   file.ContentType,
		"created_at":     file.CreatedAt.Unix(),
	}

	// Upsert document
	_, err = s.Server.TypesenseClient.Collection("files").Documents().Upsert(ctx, document)
	if err != nil {
		return fmt.Errorf("failed to index file: %w", err)
	}

	log.Printf("Indexed file %d in Typesense", fileID)
	return nil
}

// DeleteFileFromTypesense removes a file from Typesense index
func (s *Handler) DeleteFileFromTypesense(ctx context.Context, fileID int) error {
	if s.Server.TypesenseClient == nil {
		return nil // Not an error if Typesense not available
	}

	filterBy := fmt.Sprintf("file_id:%d", fileID)
	_, err := s.Server.TypesenseClient.Collection("files").Documents().Delete(ctx, &api.DeleteDocumentsParams{
		FilterBy: &filterBy,
	})
	if err != nil {
		log.Printf("Failed to delete file %d from Typesense: %v", fileID, err)
		// Don't return error - not critical
	}

	return nil
}

// searchFilesInTypesense searches files using Typesense
func (s *Handler) searchFilesInTypesense(ctx context.Context, userID int, query string, page, perPage int) (interface{}, error) {
	if s.Server.TypesenseClient == nil {
		return nil, fmt.Errorf("Typesense client not initialized")
	}

	filterBy := fmt.Sprintf("user_id:%d", userID)
	sortBy := "_text_match:desc,created_at:desc"
	searchParams := &api.SearchCollectionParams{
		Q:        query,
		QueryBy:  "name,description,extracted_text,tags",
		FilterBy: &filterBy,
		SortBy:   &sortBy,
		Page:     &page,
		PerPage:  &perPage,
	}

	result, err := s.Server.TypesenseClient.Collection("files").Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, err
	}

	// Convert Typesense results to response format
	type FileSearchResult struct {
		ID            int                `json:"id"`
		UserID        int                `json:"user_id"`
		Name          string             `json:"name"`
		Type          string             `json:"type"`
		Path          string             `json:"path"`
		Filename      string             `json:"filename"`
		Size          int                `json:"size"`
		CardPK        *int               `json:"card_pk,omitempty"`
		CreatedAt     string             `json:"created_at"`
		UpdatedAt     string             `json:"updated_at"`
		ThumbnailPath *string            `json:"thumbnail_path,omitempty"`
		Tags          []string           `json:"tags,omitempty"`
		Description   *string            `json:"description,omitempty"`
		Card          models.PartialCard `json:"card"`
	}

	// Initialize as an empty slice (not nil) so the JSON response is `[]` rather
	// than `null` when Typesense returns no hits — the frontend expects an array.
	files := []FileSearchResult{}
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			doc := *hit.Document

			var fileID int
			switch v := doc["file_id"].(type) {
			case float64:
				fileID = int(v)
			case int32:
				fileID = int(v)
			}

			var userID int
			switch v := doc["user_id"].(type) {
			case float64:
				userID = int(v)
			case int32:
				userID = int(v)
			}

			var size int
			switch v := doc["size"].(type) {
			case float64:
				size = int(v)
			case int32:
				size = int(v)
			}

			var createdAt int64
			switch v := doc["created_at"].(type) {
			case float64:
				createdAt = int64(v)
			case int64:
				createdAt = v
			}

			var tags []string
			if t, ok := doc["tags"].([]interface{}); ok {
				for _, tag := range t {
					if tagStr, ok := tag.(string); ok {
						tags = append(tags, tagStr)
					}
				}
			}

			name, _ := doc["name"].(string)
			contentType, _ := doc["content_type"].(string)

			file := FileSearchResult{
				ID:        fileID,
				UserID:    userID,
				Name:      name,
				Type:      contentType,
				Size:      size,
				CreatedAt: time.Unix(createdAt, 0).Format(time.RFC3339),
				Tags:      tags,
			}

			// Get additional data from database (path, filename, etc.)
			var path, filename string
			var cardPK sql.NullInt32
			var thumbnailPath sql.NullString
			var description sql.NullString
			var updatedAt time.Time
			err := s.GetDB().QueryRowContext(ctx, `
				SELECT path, filename, card_pk, updated_at, thumbnail_path, description
				FROM files WHERE id = $1
			`, fileID).Scan(&path, &filename, &cardPK, &updatedAt, &thumbnailPath, &description)
			if err == nil {
				file.Path = path
				file.Filename = filename
				file.UpdatedAt = updatedAt.Format(time.RFC3339)
				if cardPK.Valid && cardPK.Int32 > 0 {
					cardPKInt := int(cardPK.Int32)
					file.CardPK = &cardPKInt
					// Populate the linked card so the UI can render a link without
					// crashing on a missing card object (Zettelgarden-72f.2).
					if card, cardErr := s.QueryPartialCardByID(userID, cardPKInt); cardErr == nil {
						file.Card = card
					} else {
						file.Card = models.PartialCard{}
					}
				} else {
					file.Card = models.PartialCard{}
				}
				if description.Valid {
					desc := description.String
					file.Description = &desc
				}
				if thumbnailPath.Valid {
					file.ThumbnailPath = &thumbnailPath.String
				}
			}

			files = append(files, file)
		}
	}

	total := 0
	if result.Found != nil {
		total = *result.Found
	}

	totalPages := (total + perPage - 1) / perPage

	return map[string]interface{}{
		"files":       files,
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
	}, nil
}
