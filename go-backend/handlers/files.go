package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/models"
	"go-backend/services"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

// isImageType checks if the content type is a supported image format
func isImageType(contentType string) bool {
	supportedTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	contentType = strings.ToLower(contentType)
	for _, t := range supportedTypes {
		if contentType == t {
			return true
		}
	}
	return false
}

// generateThumbnail creates a 300x300 thumbnail from an image file
func (s *Handler) generateThumbnail(sourcePath, thumbnailPath string) error {
	// Open the source file
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to 300x300, maintaining aspect ratio
	thumbnail := resize.Thumbnail(300, 300, img, resize.Lanczos3)

	// Create the thumbnail file
	thumbFile, err := os.Create(thumbnailPath)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer thumbFile.Close()

	// Encode as JPEG with quality 85
	err = jpeg.Encode(thumbFile, thumbnail, &jpeg.Options{Quality: 85})
	if err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return nil
}

// generateAndUploadThumbnail generates a thumbnail and uploads it to S3
func (s *Handler) generateAndUploadThumbnail(userID int, fileID int, sourcePath, s3Key, contentType string) {
	// Only generate thumbnails for images
	if !isImageType(contentType) {
		return
	}

	// Create thumbnail file path
	thumbnailTempPath := sourcePath + ".thumb.jpg"
	defer os.Remove(thumbnailTempPath)

	// Generate the thumbnail
	err := s.generateThumbnail(sourcePath, thumbnailTempPath)
	if err != nil {
		log.Printf("Failed to generate thumbnail for file %d: %v", fileID, err)
		return
	}

	// Upload thumbnail to storage with _thumb suffix
	thumbnailS3Key := s3Key + "_thumb.jpg"
	thumbFile, err := os.Open(thumbnailTempPath)
	if err != nil {
		log.Printf("Failed to open thumbnail for upload (file %d): %v", fileID, err)
		return
	}
	if err := s.Server.Store.Upload(context.Background(), thumbnailS3Key, thumbFile); err != nil {
		log.Printf("Failed to upload thumbnail for file %d: %v", fileID, err)
	}
	thumbFile.Close()

	// Update database with thumbnail path
	_, err = s.GetDB().Exec("UPDATE files SET thumbnail_path = $1 WHERE id = $2", thumbnailS3Key, fileID)
	if err != nil {
		log.Printf("Failed to update thumbnail path for file %d: %v", fileID, err)
		return
	}

	log.Printf("Successfully generated thumbnail for file %d", fileID)
}

func (s *Handler) GetAllFilesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse pagination parameters
	page := 1
	perPage := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	// Parse search/filter parameters
	searchTerm := r.URL.Query().Get("search")
	filetypeFilter := r.URL.Query().Get("filetype")
	unlinkedOnly := r.URL.Query().Get("unlinked") == "true"
	sortBy := r.URL.Query().Get("sort")     // name, date, size, type, card
	sortOrder := r.URL.Query().Get("order") // asc, desc

	// Default sort
	if sortBy == "" {
		sortBy = "date"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	var response interface{}

	if searchTerm != "" && s.Server.TypesenseClient != nil {
		// Typesense semantic search. The Typesense path honors the filetype /
		// unlinked / sort / order params and returns storage_used + max_storage
		// just like the SQL path (Zettelgarden-72f.3). On failure we fall back
		// to the SQL search below.
		if result, err := s.searchFilesInTypesense(r.Context(), userID, searchTerm, filetypeFilter, unlinkedOnly, sortBy, sortOrder, page, perPage); err == nil && result != nil {
			response = result
		} else {
			if err != nil {
				log.Printf("Typesense search failed, falling back to SQL: %v", err)
			}
			files, total, storageUsed, maxStorage, qErr := s.runFileListQuery(fileListQuery{
				UserID:         userID,
				SearchPattern:  "%" + searchTerm + "%",
				FiletypeFilter: filetypeFilter,
				UnlinkedOnly:   unlinkedOnly,
				SortBy:         sortBy,
				SortOrder:      sortOrder,
				Page:           page,
				PerPage:        perPage,
			})
			if qErr != nil {
				http.Error(w, qErr.Error(), http.StatusInternalServerError)
				return
			}
			response = buildFileListResponse(files, total, page, perPage, searchTerm, storageUsed, maxStorage)
		}
	} else {
		// SQL path (no search term, or Typesense unavailable)
		q := fileListQuery{
			UserID:         userID,
			FiletypeFilter: filetypeFilter,
			UnlinkedOnly:   unlinkedOnly,
			SortBy:         sortBy,
			SortOrder:      sortOrder,
			Page:           page,
			PerPage:        perPage,
		}
		if searchTerm != "" {
			q.SearchPattern = "%" + searchTerm + "%"
		}
		files, total, storageUsed, maxStorage, err := s.runFileListQuery(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = buildFileListResponse(files, total, page, perPage, searchTerm, storageUsed, maxStorage)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type fileListQuery struct {
	UserID         int
	SearchPattern  string // SQL LIKE pattern from the fallback search ("" = none)
	FileIDs        []int  // optional exact ID set (Typesense candidates); takes precedence over SearchPattern
	FiletypeFilter string
	UnlinkedOnly   bool
	SortBy         string
	SortOrder      string
	Page           int
	PerPage        int
}

func buildFileListResponse(files []models.File, total, page, perPage int, searchTerm string, storageUsed, maxStorage int) map[string]interface{} {
	return map[string]interface{}{
		"files":        files,
		"page":         page,
		"per_page":     perPage,
		"total":        total,
		"total_pages":  (total + perPage - 1) / perPage,
		"search":       searchTerm,
		"storage_used": storageUsed,
		"max_storage":  maxStorage,
	}
}

// getStorageInfo returns the user's total uploaded bytes and their max storage
// quota, degrading to zero on error (the route treats quota as best-effort).
func (s *Handler) getStorageInfo(userID int) (int, int) {
	var storageUsed int
	err := s.GetDB().QueryRow(`SELECT COALESCE(SUM(size), 0) FROM files WHERE is_deleted = FALSE AND created_by = $1`, userID).Scan(&storageUsed)
	if err != nil {
		log.Printf("Error calculating storage used: %v", err)
		storageUsed = 0
	}
	var maxStorage int
	err = s.GetDB().QueryRow(`SELECT max_file_storage FROM users WHERE id = $1`, userID).Scan(&maxStorage)
	if err != nil {
		log.Printf("Error getting max storage: %v", err)
		maxStorage = 0
	}
	return storageUsed, maxStorage
}

// runFileListQuery executes the shared files query with the given filters and
// returns the page of files (with card + description populated), the total
// count, the user's storage usage, and their max storage.
func (s *Handler) runFileListQuery(q fileListQuery) ([]models.File, int, int, int, error) {
	// A candidate set that came back empty (e.g. Typesense found no hits for
	// the term) means zero results — short-circuit before building SQL.
	if q.FileIDs != nil && len(q.FileIDs) == 0 {
		storageUsed, maxStorage := s.getStorageInfo(q.UserID)
		return []models.File{}, 0, storageUsed, maxStorage, nil
	}

	sortBy, sortOrder := q.SortBy, q.SortOrder
	if sortBy == "" {
		sortBy = "date"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// Build ORDER BY clause based on sort parameters
	var orderByClause string
	switch sortBy {
	case "name":
		orderByClause = "f.name"
	case "size":
		orderByClause = "f.size"
	case "type":
		orderByClause = "f.type"
	case "card":
		orderByClause = "f.card_pk"
	default: // "date"
		orderByClause = "f.created_at"
	}

	// Apply sort order
	if sortOrder == "asc" {
		orderByClause += " ASC"
	} else {
		orderByClause += " DESC"
	}

	// For name, type and card, add a secondary sort by created_at so ordering
	// is deterministic when keys collide.
	if sortBy == "name" || sortBy == "type" || sortBy == "card" {
		if sortOrder == "asc" {
			orderByClause += ", f.created_at ASC"
		} else {
			orderByClause += ", f.created_at DESC"
		}
	}

	offset := (q.Page - 1) * q.PerPage

	// Build the query with optional filters
	var whereConditions []string
	var queryArgs []interface{}
	var countArgs []interface{}
	argNum := 1

	// Always filter by user and not deleted
	whereConditions = append(whereConditions, "f.is_deleted = FALSE", "f.user_id = $"+strconv.Itoa(argNum))
	queryArgs = append(queryArgs, q.UserID)
	countArgs = append(countArgs, q.UserID)
	argNum++

	// Exact candidate set (Typesense hits) or SQL LIKE search
	if len(q.FileIDs) > 0 {
		placeholders := make([]string, len(q.FileIDs))
		idArgs := make([]interface{}, len(q.FileIDs))
		for i, id := range q.FileIDs {
			placeholders[i] = "$" + strconv.Itoa(argNum)
			idArgs[i] = id
			argNum++
		}
		whereConditions = append(whereConditions, "f.id IN ("+strings.Join(placeholders, ",")+")")
		queryArgs = append(queryArgs, idArgs...)
		countArgs = append(countArgs, idArgs...)
	} else if q.SearchPattern != "" {
		whereConditions = append(whereConditions, "(f.name LIKE $"+strconv.Itoa(argNum)+" OR f.type LIKE $"+strconv.Itoa(argNum)+")")
		queryArgs = append(queryArgs, q.SearchPattern)
		countArgs = append(countArgs, q.SearchPattern)
		argNum++
	}

	// Add filetype filter (searches in MIME type)
	if q.FiletypeFilter != "" {
		whereConditions = append(whereConditions, "f.type LIKE $"+strconv.Itoa(argNum))
		filetypePattern := "%" + q.FiletypeFilter + "%"
		queryArgs = append(queryArgs, filetypePattern)
		countArgs = append(countArgs, filetypePattern)
		argNum++
	}

	// Add unlinked filter (files not attached to any card)
	if q.UnlinkedOnly {
		whereConditions = append(whereConditions, "(f.card_pk IS NULL OR f.card_pk <= 0)")
	}

	// Build the full queries
	whereClause := " WHERE " + strings.Join(whereConditions, " AND ")

	query := `
		SELECT
		f.id, f.user_id, f.name, f.type, f.path, f.filename, f.size,
		f.created_by, f.updated_by, f.card_pk, f.is_deleted,
		f.created_at, f.updated_at, f.thumbnail_path, f.description
		FROM files as f
		` + whereClause + ` ORDER BY ` + orderByClause + ` LIMIT $` + strconv.Itoa(argNum) + ` OFFSET $` + strconv.Itoa(argNum+1)

	countQuery := `SELECT COUNT(*) FROM files f` + whereClause

	queryArgs = append(queryArgs, q.PerPage, offset)

	rows, err := s.GetDB().Query(query, queryArgs...)
	if err != nil {
		log.Printf("Error querying files: %v", err)
		return nil, 0, 0, 0, err
	}
	defer rows.Close()

	// Buffer all files first so we don't run queries (card lookups) on the same
	// connection while the result set is open. Initialize as an empty slice (not
	// nil) so the JSON response is `[]` rather than `null` when nothing matches.
	files := []models.File{}
	for rows.Next() {
		var file models.File
		var cardPK sql.NullInt32
		var description sql.NullString
		if err := rows.Scan(
			&file.ID,
			&file.UserID,
			&file.Name,
			&file.Filetype,
			&file.Path,
			&file.Filename,
			&file.Size,
			&file.CreatedBy,
			&file.UpdatedBy,
			&cardPK,
			&file.IsDeleted,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.ThumbnailPath,
			&description,
		); err != nil {
			log.Printf("Error scanning file: %v", err)
			return nil, 0, 0, 0, err
		}
		// Convert nullable cardPK
		if cardPK.Valid {
			pk := int(cardPK.Int32)
			file.CardPK = &pk
		}
		// Convert nullable description
		if description.Valid {
			desc := description.String
			file.Description = &desc
		}
		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, 0, 0, err
	}

	// Now fetch cards for each file after closing the result set
	for i := range files {
		if files[i].CardPK != nil {
			partialCard, err := s.QueryPartialCardByID(q.UserID, *files[i].CardPK)
			if err != nil {
				log.Printf("card %v", partialCard)
				files[i].Card = models.PartialCard{}
			} else {
				files[i].Card = partialCard
			}
		} else {
			files[i].Card = models.PartialCard{}
		}
	}

	// Get total count for pagination
	var total int
	err = s.GetDB().QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		log.Printf("Error counting files: %v", err)
		return nil, 0, 0, 0, err
	}

	storageUsed, maxStorage := s.getStorageInfo(q.UserID)
	return files, total, storageUsed, maxStorage, nil
}

func (s *Handler) queryFile(userID int, id int) (models.File, error) {

	row := s.GetDB().QueryRow(`
	SELECT files.id, files.user_id, files.name, files.type, files.path, files.filename, files.size, files.created_by, files.updated_by, files.card_pk, files.is_deleted,
	files.created_at, files.updated_at, files.thumbnail_path, files.description
FROM files
	WHERE files.is_deleted = FALSE and files.id = $1 AND files.user_id = $2`, id, userID)

	var file models.File
	var cardPK sql.NullInt32
	var description sql.NullString

	if err := row.Scan(
		&file.ID,
		&file.UserID,
		&file.Name,
		&file.Filetype,
		&file.Path,
		&file.Filename,
		&file.Size,
		&file.CreatedBy,
		&file.UpdatedBy,
		&cardPK,
		&file.IsDeleted,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.ThumbnailPath,
		&description,
	); err != nil {
		log.Printf("err id %v %v", id, err)
		return models.File{}, errors.New("unable to access file")
	}
	// Convert nullable cardPK
	if cardPK.Valid {
		pk := int(cardPK.Int32)
		file.CardPK = &pk
	}
	if description.Valid {
		desc := description.String
		file.Description = &desc
	}
	if file.CardPK != nil {
		card, err := s.QueryPartialCardByID(userID, *file.CardPK)
		if err != nil {
			file.Card = models.PartialCard{}
		} else {
			file.Card = card
		}
	} else {
		file.Card = models.PartialCard{}
	}
	return file, nil
}

func (s *Handler) getFilesFromCardPK(userID int, cardPK int) ([]models.File, error) {

	files := []models.File{}
	rows, err := s.GetDB().Query(`
	SELECT
	files.id, files.user_id, files.name, files.type, files.path, files.filename,
	files.size, files.created_by, files.updated_by, files.card_pk,
	files.is_deleted, files.created_at, files.updated_at, files.thumbnail_path
	FROM files
	WHERE files.is_deleted = FALSE and files.card_pk = $1 AND files.user_id = $2`, cardPK, userID)

	if err != nil {
		return files, err
	}

	defer rows.Close()

	for rows.Next() {
		var file models.File
		var cardPK sql.NullInt32
		if err := rows.Scan(
			&file.ID,
			&file.UserID,
			&file.Name,
			&file.Filetype,
			&file.Path,
			&file.Filename,
			&file.Size,
			&file.CreatedBy,
			&file.UpdatedBy,
			&cardPK,
			&file.IsDeleted,
			&file.CreatedAt,
			&file.UpdatedAt,
			&file.ThumbnailPath,
		); err != nil {
			return files, err
		}
		// Convert nullable cardPK
		if cardPK.Valid {
			pk := int(cardPK.Int32)
			file.CardPK = &pk
		}
		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return files, err
	}
	return files, nil

}

func (s *Handler) GetFileMetadataRoute(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("current_user").(int)

	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	file, err := s.queryFile(userID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

func (s *Handler) EditFileMetadataRoute(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("current_user").(int)
	filePK, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var data models.EditFileMetadataParams
	bodyBytes, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes)) // Reconstruct the body for further use

	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	_, err = s.queryFile(userID, filePK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Description is optional: only write the column when explicitly provided so
	// rename / link / unlink calls (which omit it) don't wipe an existing value.
	// An explicitly-empty description clears the column (NULL).
	if data.Description != nil {
		var descriptionValue interface{} = *data.Description
		if *data.Description == "" {
			descriptionValue = nil
		}
		_, err = s.GetDB().Exec("UPDATE files SET name = $1, card_pk = $2, description = $3 WHERE id = $4", data.Name, data.CardPK, descriptionValue, filePK)
	} else {
		_, err = s.GetDB().Exec("UPDATE files SET name = $1, card_pk = $2 WHERE id = $3", data.Name, data.CardPK, filePK)
	}

	if err != nil {
		http.Error(w, "Failed to update file metadata", http.StatusInternalServerError)
		return
	}

	file, err := s.queryFile(userID, filePK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)

}

func (s *Handler) userCanUploadFile(userID int, header *multipart.FileHeader) error {
	user, err := s.QueryUser(userID)
	if err != nil {
		return fmt.Errorf("unknown problem")
	}
	if !user.CanUploadFiles {
		return fmt.Errorf("user does not have permissions to upload files")
	}
	var alreadyUploaded int
	err = s.GetDB().QueryRow(`SELECT COALESCE(sum(size), 0) FROM files WHERE created_by = $1`, userID).Scan(&alreadyUploaded)
	if err != nil {
		return err
	}
	if alreadyUploaded+int(header.Size) > user.MaxFileStorage {
		return fmt.Errorf("out of storage")
	}
	return nil
}

func (s *Handler) UploadFileRoute(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("current_user").(int)

	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		log.Printf("1")
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file part", http.StatusBadRequest)
		return
	}
	defer file.Close()
	err = s.userCanUploadFile(userID, handler)
	if err != nil {
		log.Printf("e?")
		log.Printf("err %v", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	var cardPK int
	cardPKForm := r.FormValue("card_pk")
	if cardPKForm == "undefined" {
		cardPK = -1
	} else {
		cardPK, err = strconv.Atoi(cardPKForm)
		if err != nil {
			http.Error(w, "No PK given", http.StatusBadRequest)
			return
		}
	}

	tempFile, err := os.CreateTemp("/tmp", "upload-*.tmp")
	if err != nil {
		http.Error(w, "Unable to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	// Write the file content to the temporary file
	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, "Unable to seek file", http.StatusInternalServerError)
		return
	}
	if _, err := tempFile.ReadFrom(file); err != nil {
		http.Error(w, "Unable to read file", http.StatusInternalServerError)
		return
	}
	uuidKey := uuid.New().String()
	s3Key := fmt.Sprintf("%s/%s", strconv.Itoa(userID), uuidKey)

	// Store the uploaded file. Upload reads from tempFile, so rewind it to
	// the start first (ReadFrom left the cursor at the end).
	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Unable to seek file", http.StatusInternalServerError)
		return
	}
	if err := s.Server.Store.Upload(r.Context(), s3Key, tempFile); err != nil {
		http.Error(w, "Unable to upload file", http.StatusInternalServerError)
		return
	}

	fileSize, err := tempFile.Seek(0, io.SeekEnd)
	if err != nil {
		http.Error(w, "Unable to determine file size", http.StatusInternalServerError)
		return
	}
	var lastInsertId int
	query := `INSERT INTO files (name, user_id, type, path, filename,
		size, card_pk, created_by, updated_by, updated_at) VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP) RETURNING id;`
	contentType := handler.Header.Get("Content-Type")
	err = s.GetDB().QueryRow(query,
		handler.Filename,
		userID,
		contentType,
		s3Key,
		s3Key,
		fileSize,
		cardPK,
		userID,
		userID).Scan(&lastInsertId)
	if err != nil {
		http.Error(w, "Unable to execute query", http.StatusInternalServerError)
		return
	}

	// user_stats was trigger-maintained (0093); now maintained in Go (Phase 5).
	services.IncrementUserFileCount(s.GetDB(), userID)

	// Start text extraction inline (audited in llm_jobs)
	_, err = s.JobRunner.Run(r.Context(), models.CreateJobParams{
		UserID:      userID,
		JobType:     models.JobTypeFileTextExtraction,
		Priority:    5, // Normal priority
		Payload:     map[string]interface{}{"file_id": lastInsertId, "s3_key": s3Key},
		MaxRetries:  3,
		TimeoutSecs: 300, // 5 minutes
	})
	if err != nil {
		log.Printf("Failed to start text extraction job for file %d: %v", lastInsertId, err)
		// Don't fail the upload - extraction can be retried later
	} else {
		log.Printf("Started text extraction job for file %d", lastInsertId)
	}

	// Generate thumbnail asynchronously for images. NOTE: this skip is NOT
	// about storage (the store works fine in tests) — it's about the
	// fire-and-forget goroutine outliving the rolled-back test transaction and
	// racing teardown. Keep it for that reason only (design decision D8).
	if s.Server.Testing {
		// In testing mode, skip thumbnail generation
	} else {
		go s.generateAndUploadThumbnail(userID, lastInsertId, tempFile.Name(), s3Key, contentType)
	}

	newFile, err := s.queryFile(userID, lastInsertId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	output := models.UploadFileResponse{
		Message: "File successfully uploaded",
		File:    newFile,
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

func (s *Handler) DownloadFileRoute(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value("current_user").(int)
	cardPK, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, err := s.queryFile(userID, cardPK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if thumbnail is requested
	downloadThumbnail := r.URL.Query().Get("thumbnail") == "true"
	filePathToDownload := file.Filename

	if downloadThumbnail && file.ThumbnailPath != nil && *file.ThumbnailPath != "" {
		filePathToDownload = *file.ThumbnailPath
	}

	rc, err := s.Server.Store.Download(r.Context(), filePathToDownload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer rc.Close()

	// Set headers BEFORE io.Copy: Content-Disposition used to be set *after*
	// the copy, so the header was never actually sent (fix-while-here).
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.Filename))

	// Copy the file content to the response. The Server.Testing short-circuit
	// that used to skip streaming here is gone — tests now use a real
	// tempdir-backed store, so the route streams real bytes (design D8).
	if _, err := io.Copy(w, rc); err != nil {
		http.Error(w, "Unable to send file", http.StatusInternalServerError)
		return
	}

}

func (s *Handler) DeleteFileRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	cardPK, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	file, err := s.queryFile(userID, cardPK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	query := `UPDATE files SET is_deleted = true WHERE id = $1`
	_, err = s.GetDB().Exec(query, cardPK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.Server.Store.Delete(r.Context(), file.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		_, _ = s.GetDB().Exec(`UPDATE files SET is_deleted = false WHERE id = $1`, cardPK)
		return
	}

	// file_count was trigger-maintained (0093); now maintained in Go (Phase 5).
	// Files are only ever soft-deleted, so the decrement rides the soft-delete
	// path (beads Zettelgarden-y6s) — after the storage delete succeeds, to
	// stay consistent with the is_deleted rollback above.
	services.DecrementUserFileCount(s.GetDB(), userID)
}

// Tag management endpoints

// CreateFileTagRequest represents a request to create a tag
type CreateFileTagRequest struct {
	Name string `json:"name"`
}

// CreateFileTagRoute creates a new file tag
func (s *Handler) CreateFileTagRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var req CreateFileTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Sanitize tag name
	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "Tag name cannot be empty", http.StatusBadRequest)
		return
	}
	if len(name) > 50 {
		name = name[:50]
	}

	// Insert tag
	var tagID int
	err := s.GetDB().QueryRow(
		"INSERT INTO file_tags (user_id, name) VALUES ($1, $2) ON CONFLICT (user_id, name) DO NOTHING RETURNING id",
		userID, name,
	).Scan(&tagID)

	if err == sql.ErrNoRows {
		// Tag already exists, get its ID
		err = s.GetDB().QueryRow("SELECT id FROM file_tags WHERE user_id = $1 AND name = $2", userID, name).Scan(&tagID)
	}

	if err != nil {
		log.Printf("Error creating tag: %v", err)
		http.Error(w, "Failed to create tag", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   tagID,
		"name": name,
	})
}

// GetUserFileTagsRoute returns all tags for a user
func (s *Handler) GetUserFileTagsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	rows, err := s.GetDB().Query(`
		SELECT t.id, t.name, COUNT(ft.file_id) as file_count
		FROM file_tags t
		LEFT JOIN files_tags ft ON t.id = ft.tag_id
		WHERE t.user_id = $1
		GROUP BY t.id, t.name
		ORDER BY t.name
	`, userID)

	if err != nil {
		log.Printf("Error fetching tags: %v", err)
		http.Error(w, "Failed to fetch tags", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tags []models.FileTagWithCount
	for rows.Next() {
		var tag models.FileTagWithCount
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.FileCount); err != nil {
			log.Printf("Error scanning tag: %v", err)
			continue
		}
		tags = append(tags, tag)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// TagFileRequest represents a request to tag a file
type TagFileRequest struct {
	TagNames []string `json:"tag_names"`
}

// TagFileRoute adds tags to a file
func (s *Handler) TagFileRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	fileIDStr := mux.Vars(r)["file_id"]

	fileID, err := strconv.Atoi(fileIDStr)
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	var req TagFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify file belongs to user
	var ownerID int
	err = s.GetDB().QueryRow("SELECT user_id FROM files WHERE id = $1", fileID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if ownerID != userID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Process tags
	tx, err := s.BeginTx()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, tagName := range req.TagNames {
		// Sanitize
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}

		// Get or create tag
		var tagID int
		err := tx.QueryRow(
			"INSERT INTO file_tags (user_id, name) VALUES ($1, $2) ON CONFLICT (user_id, name) DO UPDATE SET name = EXCLUDED.name RETURNING id",
			userID, tagName,
		).Scan(&tagID)

		if err != nil {
			err = tx.QueryRow("SELECT id FROM file_tags WHERE user_id = $1 AND name = $2", userID, tagName).Scan(&tagID)
			if err != nil {
				log.Printf("Error getting tag ID: %v", err)
				continue
			}
		}

		// Link tag to file
		_, err = tx.Exec(
			"INSERT INTO files_tags (file_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			fileID, tagID,
		)
		if err != nil {
			log.Printf("Error linking tag to file: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save tags", http.StatusInternalServerError)
		return
	}

	// Reindex file in Typesense (placeholder - will be implemented in Task 9)
	// go s.reindexFile(fileID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// UntagFileRoute removes a tag from a file
func (s *Handler) UntagFileRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	vars := mux.Vars(r)
	fileIDStr := vars["file_id"]
	tagName := vars["tag_name"]

	fileID, err := strconv.Atoi(fileIDStr)
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	// Verify file belongs to user
	var ownerID int
	err = s.GetDB().QueryRow("SELECT user_id FROM files WHERE id = $1", fileID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		http.Error(w, "Unauthorized", http.StatusNotFound)
		return
	}

	// Remove tag association
	result, err := s.GetDB().Exec(`
		DELETE FROM files_tags
		WHERE file_id = $1 AND tag_id = (
			SELECT id FROM file_tags WHERE user_id = $2 AND name = $3
		)
	`, fileID, userID, tagName)

	if err != nil {
		http.Error(w, "Failed to remove tag", http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Tag not found on file", http.StatusNotFound)
		return
	}

	// Reindex file (placeholder - will be implemented in Task 9)
	// go s.reindexFile(fileID)

	w.WriteHeader(http.StatusOK)
}
