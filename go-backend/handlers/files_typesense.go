package handlers

import (
	"context"
	"database/sql"
	"errors"
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
// searchFilesInTypesense searches files in Typesense and returns a response
// shaped like the SQL path. When only a search term is present it uses
// Typesense's native ranking + pagination and honors the sort/order params via
// sort_by. When filetype/unlinked filters are active it collects the candidate
// file IDs from Typesense and delegates the filtering, sorting, and pagination
// to the shared SQL query path (Typesense has no LIKE/substring filter support).
// Both shapes include storage_used + max_storage so the quota bar keeps working
// (Zettelgarden-72f.3).
func (s *Handler) searchFilesInTypesense(ctx context.Context, userID int, query, filetypeFilter, tagFilter string, unlinkedOnly bool, sortBy, sortOrder string, page, perPage int) (interface{}, error) {
	if s.Server.TypesenseClient == nil {
		return nil, fmt.Errorf("Typesense client not initialized")
	}

	if filetypeFilter == "" && !unlinkedOnly {
		return s.searchFilesInTypesenseNative(ctx, userID, query, tagFilter, sortBy, sortOrder, page, perPage)
	}

	// Filtered path: Typesense finds candidates, SQL applies filetype/unlinked
	// filters + sort + pagination.
	fileIDs, err := s.collectFileIDsFromTypesense(ctx, userID, query, tagFilter)
	if err != nil {
		return nil, err
	}
	files, total, storageUsed, maxStorage, err := s.runFileListQuery(fileListQuery{
		UserID:         userID,
		FileIDs:        fileIDs,
		SearchTerm:     query,
		FiletypeFilter: filetypeFilter,
		TagFilter:      tagFilter,
		UnlinkedOnly:   unlinkedOnly,
		SortBy:         sortBy,
		SortOrder:      sortOrder,
		Page:           page,
		PerPage:        perPage,
	})
	if err != nil {
		return nil, err
	}
	return buildFileListResponse(files, total, page, perPage, query, storageUsed, maxStorage), nil
}

// searchFilesInTypesenseNative runs a plain Typesense search (no filetype or
// unlinked filters), paginates natively, and enriches each hit with the fields
// the frontend needs (filetype key, card, description, storage quota).
func (s *Handler) searchFilesInTypesenseNative(ctx context.Context, userID int, query, tagFilter, sortBy, sortOrder string, page, perPage int) (interface{}, error) {
	if s.Server.TypesenseClient == nil {
		return nil, fmt.Errorf("Typesense client not initialized")
	}

	filterBy := buildTypesenseFilterBy(userID, tagFilter)
	tsSortBy := buildTypesenseSortBy(sortBy, sortOrder)
	searchParams := &api.SearchCollectionParams{
		Q:        query,
		QueryBy:  "name,description,extracted_text,tags",
		FilterBy: &filterBy,
		SortBy:   &tsSortBy,
		Page:     &page,
		PerPage:  &perPage,
	}

	result, err := s.Server.TypesenseClient.Collection("files").Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, err
	}

	// Convert Typesense results to response format. The JSON keys mirror
	// models.File (filetype, card, description) so the frontend File type works.
	type FileSearchResult struct {
		ID            int                `json:"id"`
		UserID        int                `json:"user_id"`
		Name          string             `json:"name"`
		Filetype      string             `json:"filetype"`
		Path          string             `json:"path"`
		Filename      string             `json:"filename"`
		Size          int                `json:"size"`
		CardPK        *int               `json:"card_pk,omitempty"`
		CreatedAt     string             `json:"created_at"`
		UpdatedAt     string             `json:"updated_at"`
		ThumbnailPath *string            `json:"thumbnail_path,omitempty"`
		Tags          []string           `json:"tags,omitempty"`
		Description   *string            `json:"description,omitempty"`
		ExtractedText *string            `json:"extracted_text,omitempty"`
		Snippet       *string            `json:"snippet,omitempty"`
		SnippetField  *string            `json:"snippet_field,omitempty"`
		Card          models.PartialCard `json:"card"`
	}

	// Initialize as an empty slice (not nil) so the JSON response is `[]` rather
	// than `null` when Typesense returns no hits — the frontend expects an array.
	files := []FileSearchResult{}
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			doc := *hit.Document

			fileID, ok := fileIDFromDocument(doc)
			if !ok {
				// Malformed hit (missing file_id): skip instead of surfacing a
				// broken row with id 0.
				continue
			}

			var hitUserID int
			switch v := doc["user_id"].(type) {
			case float64:
				hitUserID = int(v)
			case int32:
				hitUserID = int(v)
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
				UserID:    hitUserID,
				Name:      name,
				Filetype:  contentType,
				Size:      size,
				CreatedAt: time.Unix(createdAt, 0).Format(time.RFC3339),
				Tags:      tags,
			}

			// Get additional data from database (path, filename, etc.)
			var path, filename string
			var cardPK sql.NullInt32
			var thumbnailPath sql.NullString
			var description sql.NullString
			var extractedText sql.NullString
			var updatedAt time.Time
			err := s.GetDB().QueryRowContext(ctx, `
				SELECT path, filename, card_pk, updated_at, thumbnail_path, description, extracted_text
				FROM files WHERE id = $1 AND is_deleted = FALSE
			`, fileID).Scan(&path, &filename, &cardPK, &updatedAt, &thumbnailPath, &description, &extractedText)
			if errors.Is(err, sql.ErrNoRows) {
				// The row is missing or soft-deleted (stale index entry): don't
				// surface it, matching the filtered/SQL path's is_deleted filter.
				continue
			}
			if err == nil {
				file.Path = path
				file.Filename = filename
				file.UpdatedAt = updatedAt.Format(time.RFC3339)
				// Mirror the SQL path: expose card_pk for any stored value
				// (including -1 for unlinked files) so the frontend renders the
				// correct menu actions and renames preserve the field.
				if cardPK.Valid {
					cardPKInt := int(cardPK.Int32)
					file.CardPK = &cardPKInt
					if cardPK.Int32 > 0 {
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
				} else {
					file.Card = models.PartialCard{}
				}
				if description.Valid {
					desc := description.String
					file.Description = &desc
				}
				if extractedText.Valid {
					file.ExtractedText = &extractedText.String
				}
				// Server-computed snippet around the match (Zettelgarden-72f.10)
				if snippet, field := buildFileSnippet(query, name, description.String, extractedText.String, tags); snippet != "" {
					file.Snippet = &snippet
					file.SnippetField = &field
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

	storageUsed, maxStorage := s.getStorageInfo(userID)

	totalPages := (total + perPage - 1) / perPage

	return map[string]interface{}{
		"files":        files,
		"page":         page,
		"per_page":     perPage,
		"total":        total,
		"total_pages":  totalPages,
		"search":       query,
		"storage_used": storageUsed,
		"max_storage":  maxStorage,
	}, nil
}

// collectFileIDsFromTypesense walks all result pages of a Typesense search and
// returns the matching file IDs (bounded to keep the IN clause + latency sane).
func (s *Handler) collectFileIDsFromTypesense(ctx context.Context, userID int, query, tagFilter string) ([]int, error) {
	if s.Server.TypesenseClient == nil {
		return nil, fmt.Errorf("Typesense client not initialized")
	}

	filterBy := buildTypesenseFilterBy(userID, tagFilter)
	sortBy := "_text_match:desc"
	fileIDs := []int{}
	const (
		perPage       = 250
		maxCandidates = 2000
		maxPages      = 20 // hard bound (20 * 250 = 5000) guaranteeing termination
	)
	page := 1
	for page <= maxPages {
		includeFields := "file_id"
		searchParams := &api.SearchCollectionParams{
			Q:             query,
			QueryBy:       "name,description,extracted_text,tags",
			FilterBy:      &filterBy,
			SortBy:        &sortBy,
			Page:          &page,
			PerPage:       pointer.Int(perPage),
			IncludeFields: &includeFields,
		}
		result, err := s.Server.TypesenseClient.Collection("files").Documents().Search(ctx, searchParams)
		if err != nil {
			return nil, err
		}
		if result.Hits != nil {
			for _, hit := range *result.Hits {
				if id, ok := fileIDFromDocument(*hit.Document); ok {
					fileIDs = append(fileIDs, id)
				}
			}
		}
		found := 0
		if result.Found != nil {
			found = *result.Found
		}
		if len(fileIDs) >= found || len(fileIDs) >= maxCandidates {
			break
		}
		page++
	}
	return fileIDs, nil
}

// fileIDFromDocument extracts the numeric file_id from a Typesense hit document.
func fileIDFromDocument(doc map[string]interface{}) (int, bool) {
	switch v := doc["file_id"].(type) {
	case float64:
		return int(v), true
	case int32:
		return int(v), true
	}
	return 0, false
}

// buildTypesenseFilterBy builds the Typesense filter_by string for a user,
// optionally narrowed to files carrying the given tag (exact array match).
func buildTypesenseFilterBy(userID int, tagFilter string) string {
	filterBy := fmt.Sprintf("user_id:%d", userID)
	if tagFilter != "" {
		filterBy += " && tags:=" + tagFilter
	}
	return filterBy
}

// buildTypesenseSortBy maps the UI sort/order params onto a Typesense sort_by
// string. The default (date) keeps relevance ranking first with recency as the
// tiebreaker, matching the pre-existing plain-search behavior.
func buildTypesenseSortBy(sortBy, sortOrder string) string {
	if sortOrder == "" {
		sortOrder = "desc"
	}
	switch sortBy {
	case "name":
		return "name:" + sortOrder + ",created_at:" + sortOrder
	case "size":
		return "size:" + sortOrder
	case "type":
		return "content_type:" + sortOrder + ",created_at:" + sortOrder
	case "card":
		return "card_pk:" + sortOrder + ",created_at:" + sortOrder
	default: // "date"
		if sortOrder == "asc" {
			return "created_at:asc"
		}
		return "_text_match:desc,created_at:desc"
	}
}
