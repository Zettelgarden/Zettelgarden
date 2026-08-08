> **STATUS: HISTORICAL — pre-SQLite era.** This plan predates the PostgreSQL→SQLite cutover (2026-07-28, epic Zettelgarden-c7j) and the move to local on-disk file storage (epic Zettelgarden-yar). Zettelgarden now runs SQLite-only with local storage; this document is kept for design history.

# FileVault Document Management Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add full-text search, tags, and previews to FileVault for better document organization.

**Architecture:** Extract text from uploaded files (PDF/Word/Excel) via background job, index in Typesense, add tags table, enhance frontend with search, tag management, and PDF preview.

**Tech Stack:** Go (backend), PostgreSQL (database), Typesense (search), React/TypeScript (frontend), pdf.js (preview)

---

## Phase 1: Database Schema & Models

### Task 1: Create database migrations for tags

**Files:**
- Create: `go-backend/migrations/026_file_tags.up.sql`
- Create: `go-backend/migrations/026_file_tags.down.sql`

**Step 1: Write migration to add tags tables**

```sql
-- go-backend/migrations/026_file_tags.up.sql

-- Add description and extracted_text columns to files
ALTER TABLE files ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE files ADD COLUMN IF NOT EXISTS extracted_text TEXT;

-- Create file_tags table
CREATE TABLE IF NOT EXISTS file_tags (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_tag UNIQUE(user_id, name)
);

-- Create files_tags junction table
CREATE TABLE IF NOT EXISTS files_tags (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES file_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_file_tags_user_id ON file_tags(user_id);
CREATE INDEX IF NOT EXISTS idx_files_tags_file_id ON files_tags(file_id);
CREATE INDEX IF NOT EXISTS idx_files_tags_tag_id ON files_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_files_extracted_text ON files USING gin(to_tsvector('english', extracted_text));
```

**Step 2: Write rollback migration**

```sql
-- go-backend/migrations/026_file_tags.down.sql

DROP TABLE IF EXISTS files_tags;
DROP TABLE IF EXISTS file_tags;
ALTER TABLE files DROP COLUMN IF EXISTS extracted_text;
ALTER TABLE files DROP COLUMN IF EXISTS description;
```

**Step 3: Run migration**

Run: `cd go-backend && goose postgres $DATABASE_URL up`
Expected: Migration applies successfully

**Step 4: Verify schema**

Run: `psql $DATABASE_URL -c "\d files"` and `psql $DATABASE_URL -c "\d file_tags"`
Expected: New columns and tables visible

**Step 5: Commit**

```bash
git add go-backend/migrations/026_file_tags.*
git commit -m "Add database schema for file tags and extracted text"
```

---

### Task 2: Update Go models for tags

**Files:**
- Modify: `go-backend/models/file.go`
- Create: `go-backend/models/file_tag.go`

**Step 1: Create FileTag model**

```go
// go-backend/models/file_tag.go

package models

import "time"

type FileTag struct {
    ID        int       `json:"id"`
    UserID    int       `json:"user_id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

type FileTagWithCount struct {
    FileTag
    FileCount int `json:"file_count"`
}
```

**Step 2: Update File model**

```go
// go-backend/models/file.go - Add to existing File struct

type File struct {
    // ... existing fields ...
    Description   *string   `json:"description,omitempty"`
    ExtractedText *string   `json:"extracted_text,omitempty"`
    Tags          []string  `json:"tags,omitempty"` // Populated on read
}

// Add new struct for file updates
type FileUpdateParams struct {
    Name        *string `json:"name,omitempty"`
    Description *string `json:"description,omitempty"`
    CardPK      *int    `json:"card_pk,omitempty"`
}
```

**Step 3: Commit**

```bash
git add go-backend/models/file.go go-backend/models/file_tag.go
git commit -m "Add FileTag model and update File model with tags"
```

---

### Task 3: Add file_text_extraction job type

**Files:**
- Modify: `go-backend/models/job.go`

**Step 1: Add new job type constant**

```go
// go-backend/models/job.go - Add to const block

const (
    JobTypeSummarization        JobType = "summarization"
    JobTypeEntityExtraction     JobType = "entity_extraction"
    JobTypeFactEntityExtraction JobType = "fact_entity_extraction"
    JobTypeChat                 JobType = "chat"
    JobTypeMemory               JobType = "memory"
    JobTypeEmail                JobType = "email"
    JobTypeFileTextExtraction   JobType = "file_text_extraction" // NEW
)
```

**Step 2: Update job type validation in handlers**

```go
// go-backend/handlers/jobs.go - Update isValidJobType function

func isValidJobType(jobType string) bool {
    validTypes := []string{
        "summarization",
        "entity_extraction",
        "fact_entity_extraction",
        "chat",
        "memory",
        "email",
        "file_text_extraction", // NEW
    }
    for _, t := range validTypes {
        if t == jobType {
            return true
        }
    }
    return false
}
```

**Step 3: Commit**

```bash
git add go-backend/models/job.go go-backend/handlers/jobs.go
git commit -m "Add file_text_extraction job type"
```

---

## Phase 2: Text Extraction Service

### Task 4: Install text extraction libraries

**Files:**
- Modify: `go-backend/go.mod`

**Step 1: Add dependencies**

Run: `cd go-backend && go get github.com/ledongthuc/pdf && go get github.com/nguyenthenguyen/docx && go get github.com/xuri/excelize/v2`

**Step 2: Tidy dependencies**

Run: `cd go-backend && go mod tidy`

**Step 3: Commit**

```bash
git add go-backend/go.mod go-backend/go.sum
git commit -m "Add text extraction libraries for PDF, Word, Excel"
```

---

### Task 5: Create text extraction service

**Files:**
- Create: `go-backend/services/text_extraction.go`
- Create: `go-backend/services/text_extraction_test.go`

**Step 1: Write test for PDF extraction**

```go
// go-backend/services/text_extraction_test.go

package services

import (
    "bytes"
    "os"
    "testing"
)

func TestExtractTextFromPDF(t *testing.T) {
    // Load test PDF
    data, err := os.ReadFile("testdata/sample.pdf")
    if err != nil {
        t.Skip("Test PDF not found")
    }

    text, err := ExtractText("application/pdf", bytes.NewReader(data))
    if err != nil {
        t.Fatalf("ExtractText failed: %v", err)
    }

    if text == "" {
        t.Error("Expected non-empty text extraction")
    }
}

func TestExtractTextFromUnsupportedType(t *testing.T) {
    text, err := ExtractText("image/png", nil)
    if err != nil {
        t.Errorf("Expected no error for unsupported type, got: %v", err)
    }
    if text != "" {
        t.Error("Expected empty string for unsupported type")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services -run TestExtractText -v`
Expected: FAIL - function not defined

**Step 3: Implement text extraction**

```go
// go-backend/services/text_extraction.go

package services

import (
    "bytes"
    "io"
    "log"
    "mime"
    "path/filepath"
    "strings"

    "github.com/ledongthuc/pdf"
    "github.com/nguyenthenguyen/docx"
    "github.com/xuri/excelize/v2"
)

// ExtractText extracts text content from various file types
func ExtractText(contentType string, reader io.Reader) (string, error) {
    switch {
    case strings.Contains(contentType, "pdf"):
        return extractFromPDF(reader)
    case strings.Contains(contentType, "word") || strings.Contains(contentType, "document"):
        return extractFromDocx(reader)
    case strings.Contains(contentType, "excel") || strings.Contains(contentType, "spreadsheet"):
        return extractFromXlsx(reader)
    case strings.HasPrefix(contentType, "text/"):
        return extractFromPlainText(reader)
    default:
        // Unsupported type - return empty string
        return "", nil
    }
}

func extractFromPDF(reader io.Reader) (string, error) {
    // Convert io.Reader to []byte for pdf library
    data, err := io.ReadAll(reader)
    if err != nil {
        return "", err
    }

    // Limit size to prevent memory issues
    if len(data) > 50*1024*1024 { // 50MB
        data = data[:50*1024*1024]
    }

    pdfReader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
    if err != nil {
        return "", err
    }

    var textBuilder strings.Builder
    numPages := pdfReader.NumPage()

    // Extract text from each page
    for i := 1; i <= numPages; i++ {
        page := pdfReader.Page(i)
        if page.V.IsNull() {
            continue
        }

        text, err := page.GetPlainText(nil)
        if err != nil {
            log.Printf("Error extracting text from page %d: %v", i, err)
            continue
        }

        textBuilder.WriteString(text)
        textBuilder.WriteString("\n")

        // Limit extracted text to 100KB
        if textBuilder.Len() > 100*1024 {
            textBuilder.WriteString("\n[TRUNCATED]")
            break
        }
    }

    return textBuilder.String(), nil
}

func extractFromDocx(reader io.Reader) (string, error) {
    data, err := io.ReadAll(reader)
    if err != nil {
        return "", err
    }

    doc, err := docx.Read(bytes.NewReader(data), int64(len(data)))
    if err != nil {
        return "", err
    }

    var textBuilder strings.Builder
    for _, para := range doc.Paragraphs() {
        textBuilder.WriteString(para)
        textBuilder.WriteString("\n")

        if textBuilder.Len() > 100*1024 {
            textBuilder.WriteString("\n[TRUNCATED]")
            break
        }
    }

    return textBuilder.String(), nil
}

func extractFromXlsx(reader io.Reader) (string, error) {
    f, err := excelize.OpenReader(reader)
    if err != nil {
        return "", err
    }
    defer f.Close()

    var textBuilder strings.Builder
    sheets := f.GetSheetList()

    for _, sheet := range sheets {
        rows, err := f.GetRows(sheet)
        if err != nil {
            continue
        }

        for _, row := range rows {
            for _, cell := range row {
                textBuilder.WriteString(cell)
                textBuilder.WriteString(" ")
            }
            textBuilder.WriteString("\n")

            if textBuilder.Len() > 100*1024 {
                textBuilder.WriteString("\n[TRUNCATED]")
                return textBuilder.String(), nil
            }
        }
        textBuilder.WriteString("\n")
    }

    return textBuilder.String(), nil
}

func extractFromPlainText(reader io.Reader) (string, error) {
    data, err := io.ReadAll(reader)
    if err != nil {
        return "", err
    }

    // Limit to 100KB
    if len(data) > 100*1024 {
        return string(data[:100*1024]) + "\n[TRUNCATED]", nil
    }

    return string(data), nil
}

// GetFileExtension extracts extension from content type
func GetFileExtension(contentType string) string {
    exts, err := mime.ExtensionsByType(contentType)
    if err != nil || len(exts) == 0 {
        return ""
    }
    return exts[0]
}
```

**Step 4: Create test data file**

```bash
mkdir -p go-backend/services/testdata
# Add a small sample.pdf file for testing
```

**Step 5: Run tests**

Run: `cd go-backend && go test ./services -run TestExtractText -v`
Expected: PASS (or skip if no test file)

**Step 6: Commit**

```bash
git add go-backend/services/text_extraction.go go-backend/services/text_extraction_test.go
git commit -m "Add text extraction service for PDF, Word, Excel files"
```

---

### Task 6: Create file text extraction job processor

**Files:**
- Create: `go-backend/services/jobs/file_text_extraction_job.go`
- Create: `go-backend/services/jobs/file_text_extraction_job_test.go`

**Step 1: Write test for job processor**

```go
// go-backend/services/jobs/file_text_extraction_job_test.go

package jobs

import (
    "context"
    "testing"

    "go-backend/models"
)

func TestFileTextExtractionJob_ProcessJob(t *testing.T) {
    // This is an integration test that would need a real file
    // Skip in unit tests
    t.Skip("Integration test - requires real file and database")
}
```

**Step 2: Implement job processor**

```go
// go-backend/services/jobs/file_text_extraction_job.go

package jobs

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "os"

    "go-backend/models"
    "go-backend/services"
)

// FileTextExtractionJob processes text extraction for uploaded files
type FileTextExtractionJob struct {
    DB             *sql.DB
    Typesense      services.TypesenseClient
    S3Bucket       string
    S3Client       services.S3Client
}

// NewFileTextExtractionJob creates a new job processor
func NewFileTextExtractionJob(db *sql.DB, typesense services.TypesenseClient, s3Bucket string, s3Client services.S3Client) *FileTextExtractionJob {
    return &FileTextExtractionJob{
        DB:        db,
        Typesense: typesense,
        S3Bucket:  s3Bucket,
        S3Client:  s3Client,
    }
}

// ProcessJob extracts text from a file and indexes it
func (j *FileTextExtractionJob) ProcessJob(ctx context.Context, job *models.LLMJob) (map[string]interface{}, error) {
    // Extract file_id from payload
    fileIDFloat, ok := job.Payload["file_id"].(float64)
    if !ok {
        return nil, fmt.Errorf("missing or invalid file_id in payload")
    }
    fileID := int(fileIDFloat)

    s3Key, ok := job.Payload["s3_key"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid s3_key in payload")
    }

    log.Printf("[FileTextExtractionJob] Processing file %d from %s", fileID, s3Key)

    // Download file from S3
    tempFile, err := j.downloadFromS3(s3Key)
    if err != nil {
        return nil, fmt.Errorf("failed to download file: %w", err)
    }
    defer os.Remove(tempFile)
    defer tempFile.Close()

    // Get file metadata
    var contentType string
    err = j.DB.QueryRow("SELECT type FROM files WHERE id = $1", fileID).Scan(&contentType)
    if err != nil {
        return nil, fmt.Errorf("failed to get file metadata: %w", err)
    }

    // Extract text
    extractedText, err := services.ExtractText(contentType, tempFile)
    if err != nil {
        log.Printf("[FileTextExtractionJob] Text extraction failed for file %d: %v", fileID, err)
        // Update file with error status
        j.DB.Exec("UPDATE files SET extracted_text = $1 WHERE id = $2", "[EXTRACTION_FAILED]", fileID)
        return nil, err
    }

    // Update database with extracted text
    _, err = j.DB.Exec("UPDATE files SET extracted_text = $1 WHERE id = $2", extractedText, fileID)
    if err != nil {
        return nil, fmt.Errorf("failed to update extracted text: %w", err)
    }

    // Index in Typesense
    err = j.indexInTypesense(ctx, fileID, extractedText)
    if err != nil {
        log.Printf("[FileTextExtractionJob] Failed to index file %d in Typesense: %v", fileID, err)
        // Don't fail the job - text is still in DB
    }

    log.Printf("[FileTextExtractionJob] Successfully processed file %d", fileID)

    return map[string]interface{}{
        "file_id":         fileID,
        "text_length":     len(extractedText),
        "extraction_type": contentType,
    }, nil
}

func (j *FileTextExtractionJob) downloadFromS3(s3Key string) (*os.File, error) {
    // Create temp file
    tempFile, err := os.CreateTemp("", "file-extraction-*.tmp")
    if err != nil {
        return nil, err
    }

    // Download from S3
    err = j.S3Client.Download(j.S3Bucket, s3Key, tempFile)
    if err != nil {
        tempFile.Close()
        return nil, err
    }

    // Reset file pointer for reading
    _, err = tempFile.Seek(0, 0)
    if err != nil {
        tempFile.Close()
        return nil, err
    }

    return tempFile, nil
}

func (j *FileTextExtractionJob) indexInTypesense(ctx context.Context, fileID int, extractedText string) error {
    // Get full file data
    var file models.File
    var description sql.NullString
    var cardPK sql.NullInt32

    query := `
        SELECT id, user_id, name, type, created_at
        FROM files WHERE id = $1
    `
    err := j.DB.QueryRow(query, fileID).Scan(
        &file.ID, &file.UserID, &file.Name, &file.Filetype, &file.CreatedAt,
    )
    if err != nil {
        return err
    }

    // Get tags for this file
    rows, err := j.DB.Query(`
        SELECT t.name
        FROM file_tags t
        JOIN files_tags ft ON t.id = ft.tag_id
        WHERE ft.file_id = $1
    `, fileID)
    if err != nil {
        return err
    }
    defer rows.Close()

    var tags []string
    for rows.Next() {
        var tag string
        if err := rows.Scan(&tag); err == nil {
            tags = append(tags, tag)
        }
    }

    // Index in Typesense
    document := map[string]interface{}{
        "id":             fmt.Sprintf("%d", file.ID),
        "user_id":        file.UserID,
        "name":           file.Name,
        "description":    description.String,
        "extracted_text": extractedText,
        "tags":           tags,
        "card_pk":        cardPK.Int32,
        "created_at":     file.CreatedAt.Unix(),
    }

    return j.Typesense.IndexDocument(ctx, "files", document)
}
```

**Step 3: Commit**

```bash
git add go-backend/services/jobs/file_text_extraction_job.go go-backend/services/jobs/file_text_extraction_job_test.go
git commit -m "Add file text extraction background job processor"
```

---

## Phase 3: Backend API Endpoints

### Task 7: Add tag management endpoints

**Files:**
- Modify: `go-backend/handlers/files.go`

**Step 1: Add create tag endpoint**

```go
// Add to go-backend/handlers/files.go

// CreateFileTagRequest represents a request to create a tag
type CreateFileTagRequest struct {
    Name string `json:"name"`
}

// CreateFileTagRoute creates a new file tag
func (h *Handler) CreateFileTagRoute(w http.ResponseWriter, r *http.Request) {
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
    err := h.GetDB().QueryRow(
        "INSERT INTO file_tags (user_id, name) VALUES ($1, $2) ON CONFLICT (user_id, name) DO NOTHING RETURNING id",
        userID, name,
    ).Scan(&tagID)

    if err == sql.ErrNoRows {
        // Tag already exists, get its ID
        err = h.GetDB().QueryRow("SELECT id FROM file_tags WHERE user_id = $1 AND name = $2", userID, name).Scan(&tagID)
    }

    if err != nil {
        log.Printf("Error creating tag: %v", err)
        http.Error(w, "Failed to create tag", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "id":   tagID,
        "name": name,
    })
}
```

**Step 2: Add get tags endpoint**

```go
// Add to go-backend/handlers/files.go

// GetUserFileTagsRoute returns all tags for a user
func (h *Handler) GetUserFileTagsRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    rows, err := h.GetDB().Query(`
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

    json.NewEncoder(w).Encode(tags)
}
```

**Step 3: Add tag file endpoint**

```go
// Add to go-backend/handlers/files.go

// TagFileRequest represents a request to tag a file
type TagFileRequest struct {
    TagNames []string `json:"tag_names"`
}

// TagFileRoute adds tags to a file
func (h *Handler) TagFileRoute(w http.ResponseWriter, r *http.Request) {
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
    err = h.GetDB().QueryRow("SELECT user_id FROM files WHERE id = $1", fileID).Scan(&ownerID)
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
    tx, err := h.GetDB().Begin()
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

    // Reindex file in Typesense
    go h.reindexFile(fileID)

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// UntagFileRoute removes a tag from a file
func (h *Handler) UntagFileRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    fileIDStr := mux.Vars(r)["file_id"]
    tagName := mux.Vars(r)["tag_name"]

    fileID, err := strconv.Atoi(fileIDStr)
    if err != nil {
        http.Error(w, "Invalid file ID", http.StatusBadRequest)
        return
    }

    // Verify file belongs to user
    var ownerID int
    err = h.GetDB().QueryRow("SELECT user_id FROM files WHERE id = $1", fileID).Scan(&ownerID)
    if err != nil || ownerID != userID {
        http.Error(w, "Unauthorized", http.StatusNotFound)
        return
    }

    // Remove tag association
    _, err = h.GetDB().Exec(`
        DELETE FROM files_tags
        WHERE file_id = $1 AND tag_id = (
            SELECT id FROM file_tags WHERE user_id = $2 AND name = $3
        )
    `, fileID, userID, tagName)

    if err != nil {
        http.Error(w, "Failed to remove tag", http.StatusInternalServerError)
        return
    }

    // Reindex file
    go h.reindexFile(fileID)

    w.WriteHeader(http.StatusOK)
}
```

**Step 4: Add helper to reindex file**

```go
// Add to go-backend/handlers/files.go

func (h *Handler) reindexFile(fileID int) {
    // Implementation will use Typesense client to reindex
    // This will be implemented in Task 9
}
```

**Step 5: Commit**

```bash
git add go-backend/handlers/files.go
git commit -m "Add tag management API endpoints for files"
```

---

### Task 8: Update file upload to enqueue extraction job

**Files:**
- Modify: `go-backend/handlers/files.go`

**Step 1: Update upload handler to enqueue job**

```go
// Find the upload handler in go-backend/handlers/files.go
// Add after successful file creation:

// After line where file is saved to DB:
_, err = s.GetDB().Exec(
    `INSERT INTO files (user_id, name, type, path, filename, size, card_pk, created_by, updated_by)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
    userID, filename, contentType, s3Key, filename, size, cardPK, userID, userID,
)

// ADD THIS:
// Enqueue text extraction job
if s.JobQueue != nil {
    jobParams := models.CreateJobParams{
        UserID:  userID,
        JobType: models.JobTypeFileTextExtraction,
        Priority: 5, // Normal priority
        Payload: map[string]interface{}{
            "file_id": fileID,
            "s3_key":  s3Key,
        },
        MaxRetries: 3,
        TimeoutSecs: 300, // 5 minutes
    }

    _, err = s.JobQueue.Enqueue(r.Context(), jobParams)
    if err != nil {
        log.Printf("Failed to enqueue text extraction job: %v", err)
        // Don't fail the upload - extraction can be retried
    }
}
```

**Step 2: Commit**

```bash
git add go-backend/handlers/files.go
git commit -m "Enqueue text extraction job on file upload"
```

---

### Task 9: Integrate Typesense search

**Files:**
- Modify: `go-backend/handlers/files.go`

**Step 1: Update GetAllFilesRoute to use Typesense**

```go
// Modify go-backend/handlers/files.go - GetAllFilesRoute function

func (s *Handler) GetAllFilesRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    // Parse parameters (existing code)
    searchTerm := r.URL.Query().Get("search")
    // ... other params ...

    // If search term provided and Typesense available, use it
    if searchTerm != "" && s.TypesenseClient != nil {
        files, err := s.searchFilesInTypesense(r.Context(), userID, searchTerm, page, perPage)
        if err == nil {
            // Success - return results
            json.NewEncoder(w).Encode(files)
            return
        }
        log.Printf("Typesense search failed, falling back to DB: %v", err)
    }

    // Fallback to PostgreSQL search (existing code)
    // ... existing PostgreSQL query logic ...
}

func (s *Handler) searchFilesInTypesense(ctx context.Context, userID int, query string, page, perPage int) (*FilesResponse, error) {
    searchParams := &api.SearchCollectionParams{
        Q:                query,
        QueryBy:          "name,description,extracted_text,tags",
        FilterBy:         fmt.Sprintf("user_id:%d", userID),
        SortBy:           "_text_match:desc,created_at:desc",
        Page:             &page,
        PerPage:          &perPage,
    }

    result, err := s.TypesenseClient.Collection("files").Documents().Search(searchParams)
    if err != nil {
        return nil, err
    }

    // Convert Typesense results to File objects
    var files []models.File
    for _, hit := range *result.Hits {
        doc := hit.Document

        var id int
        switch v := doc["id"].(type) {
        case string:
            id, _ = strconv.Atoi(v)
        case float64:
            id = int(v)
        }

        file := models.File{
            ID:        id,
            UserID:    int(doc["user_id"].(float64)),
            Name:      doc["name"].(string),
            CreatedAt: time.Unix(int64(doc["created_at"].(float64)), 0),
        }

        if desc, ok := doc["description"].(string); ok {
            file.Description = &desc
        }

        if tags, ok := doc["tags"].([]interface{}); ok {
            for _, t := range tags {
                if tagStr, ok := t.(string); ok {
                    file.Tags = append(file.Tags, tagStr)
                }
            }
        }

        files = append(files, file)
    }

    return &FilesResponse{
        Files:      files,
        Page:       page,
        PerPage:    perPage,
        Total:      *result.Found,
        TotalPages: (*result.Found + perPage - 1) / perPage,
    }, nil
}
```

**Step 2: Add Typesense collection initialization**

```go
// Add to go-backend/services/typesense.go or create new file

func CreateFilesCollection(client *typesense.Client) error {
    schema := &api.CollectionSchema{
        Name: "files",
        Fields: []api.Field{
            {Name: "id", Type: "int32"},
            {Name: "user_id", Type: "int32", Facet: true},
            {Name: "name", Type: "string"},
            {Name: "description", Type: "string", Optional: true},
            {Name: "extracted_text", Type: "string", Optional: true},
            {Name: "tags", Type: "string[]", Optional: true, Facet: true},
            {Name: "card_pk", Type: "int32", Optional: true, Facet: true},
            {Name: "created_at", Type: "int64"},
        },
    }

    _, err := client.Collections().Create(schema)
    return err
}
```

**Step 3: Commit**

```bash
git add go-backend/handlers/files.go
git commit -m "Integrate Typesense search for file content"
```

---

### Task 10: Add routes for new endpoints

**Files:**
- Modify: `go-backend/handlers/routes.go` or wherever routes are defined

**Step 1: Add routes**

```go
// Add to route definitions

// File tags
router.HandleFunc("/files/tags", s.CreateFileTagRoute).Methods("POST")
router.HandleFunc("/files/tags", s.GetUserFileTagsRoute).Methods("GET")
router.HandleFunc("/files/{file_id}/tags", s.TagFileRoute).Methods("POST")
router.HandleFunc("/files/{file_id}/tags/{tag_name}", s.UntagFileRoute).Methods("DELETE")
```

**Step 2: Commit**

```bash
git add go-backend/handlers/routes.go
git commit -m "Add routes for file tag management endpoints"
```

---

## Phase 4: Frontend Implementation

### Task 11: Update File model and API client

**Files:**
- Modify: `zettelkasten-front/src/models/File.ts`
- Modify: `zettelkasten-front/src/api/files.ts`

**Step 1: Update File model**

```typescript
// zettelkasten-front/src/models/File.ts

export interface File {
  id: number;
  user_id: number;
  name: string;
  type: string;
  path: string;
  filename: string;
  size: number;
  card_pk: number | null;
  created_at: string;
  updated_at: string;
  description?: string;
  extracted_text?: string;
  tags?: string[];
  thumbnail_path?: string;
}

export interface FileTag {
  id: number;
  user_id: number;
  name: string;
  file_count?: number;
}

export interface FileUpdateParams {
  name?: string;
  description?: string;
  card_pk?: number;
}
```

**Step 2: Add tag API functions**

```typescript
// Add to zettelkasten-front/src/api/files.ts

export interface FileTag {
  id: number;
  name: string;
  file_count?: number;
}

export function createFileTag(name: string): Promise<FileTag> {
  return getData(apiClient.post<FileTag>("/files/tags", { name }));
}

export function getFileTags(): Promise<FileTag[]> {
  return getData(apiClient.get<FileTag[]>("/files/tags"));
}

export function tagFile(fileId: number, tagNames: string[]): Promise<void> {
  return getData(apiClient.post(`/files/${fileId}/tags`, { tag_names: tagNames }));
}

export function untagFile(fileId: number, tagName: string): Promise<void> {
  return getData(apiClient.delete(`/files/${fileId}/tags/${encodeURIComponent(tagName)}`));
}
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/models/File.ts zettelkasten-front/src/api/files.ts
git commit -m "Update File model and add tag API functions"
```

---

### Task 12: Create FileTags component

**Files:**
- Create: `zettelkasten-front/src/components/files/FileTags.tsx`
- Create: `zettelkasten-front/src/components/files/FileTags.test.tsx`

**Step 1: Write test**

```typescript
// zettelkasten-front/src/components/files/FileTags.test.tsx

import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { FileTags } from './FileTags';

describe('FileTags', () => {
  it('renders existing tags', () => {
    const tags = ['taxes', '2024'];
    render(<FileTags tags={tags} onAddTag={() => {}} onRemoveTag={() => {}} />);

    expect(screen.getByText('#taxes')).toBeInTheDocument();
    expect(screen.getByText('#2024')).toBeInTheDocument();
  });

  it('calls onAddTag when tag added', () => {
    const onAddTag = vi.fn();
    render(<FileTags tags={[]} onAddTag={onAddTag} onRemoveTag={() => {}} />);

    const input = screen.getByPlaceholderText('Add tag...');
    fireEvent.change(input, { target: { value: 'mortgage' } });
    fireEvent.keyPress(input, { key: 'Enter', charCode: 13 });

    expect(onAddTag).toHaveBeenCalledWith('mortgage');
  });

  it('calls onRemoveTag when tag removed', () => {
    const onRemoveTag = vi.fn();
    render(<FileTags tags={['taxes']} onAddTag={() => {}} onRemoveTag={onRemoveTag} />);

    const removeButton = screen.getByLabelText('Remove tag taxes');
    fireEvent.click(removeButton);

    expect(onRemoveTag).toHaveBeenCalledWith('taxes');
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test FileTags.test.tsx`
Expected: FAIL - component doesn't exist

**Step 3: Implement component**

```typescript
// zettelkasten-front/src/components/files/FileTags.tsx

import React, { useState, KeyboardEvent } from 'react';

interface FileTagsProps {
  tags: string[];
  onAddTag: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  editable?: boolean;
}

export function FileTags({ tags, onAddTag, onRemoveTag, editable = true }: FileTagsProps) {
  const [inputValue, setInputValue] = useState('');

  const handleKeyPress = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && inputValue.trim()) {
      e.preventDefault();
      onAddTag(inputValue.trim());
      setInputValue('');
    }
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      {tags.map((tag) => (
        <span
          key={tag}
          className="inline-flex items-center gap-1 px-2 py-1 text-sm bg-blue-100 text-blue-800 rounded-full"
        >
          #{tag}
          {editable && (
            <button
              onClick={() => onRemoveTag(tag)}
              className="text-blue-600 hover:text-blue-800"
              aria-label={`Remove tag ${tag}`}
            >
              ×
            </button>
          )}
        </span>
      ))}
      {editable && (
        <input
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder="Add tag..."
          className="px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      )}
    </div>
  );
}
```

**Step 4: Run tests**

Run: `cd zettelkasten-front && npm test FileTags.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/files/FileTags.tsx zettelkasten-front/src/components/files/FileTags.test.tsx
git commit -m "Add FileTags component for tag management"
```

---

### Task 13: Create FileMetadataEditor component

**Files:**
- Create: `zettelkasten-front/src/components/files/FileMetadataEditor.tsx`

**Step 1: Create component**

```typescript
// zettelkasten-front/src/components/files/FileMetadataEditor.tsx

import React, { useState, useEffect } from 'react';
import { File, FileTag } from '../../models/File';
import { editFile, getFileTags, tagFile, untagFile } from '../../api/files';
import { FileTags } from './FileTags';
import { BacklinkInput } from '../cards/BacklinkInput';
import { defaultCard, PartialCard } from '../../models/Card';

interface FileMetadataEditorProps {
  file: File;
  onUpdate: () => void;
  onClose: () => void;
}

export function FileMetadataEditor({ file, onUpdate, onClose }: FileMetadataEditorProps) {
  const [description, setDescription] = useState(file.description || '');
  const [tags, setTags] = useState<string[]>(file.tags || []);
  const [saving, setSaving] = useState(false);

  const handleSaveDescription = async () => {
    setSaving(true);
    try {
      await editFile(file.id.toString(), { description });
      onUpdate();
    } catch (error) {
      console.error('Failed to save description:', error);
    } finally {
      setSaving(false);
    }
  };

  const handleAddTag = async (tagName: string) => {
    try {
      await tagFile(file.id, [tagName]);
      setTags([...tags, tagName]);
      onUpdate();
    } catch (error) {
      console.error('Failed to add tag:', error);
    }
  };

  const handleRemoveTag = async (tagName: string) => {
    try {
      await untagFile(file.id, tagName);
      setTags(tags.filter((t) => t !== tagName));
      onUpdate();
    } catch (error) {
      console.error('Failed to remove tag:', error);
    }
  };

  const handleLinkCard = async (card: PartialCard) => {
    try {
      await editFile(file.id.toString(), { card_pk: card.id });
      onUpdate();
    } catch (error) {
      console.error('Failed to link card:', error);
    }
  };

  return (
    <div className="p-4 bg-white border border-gray-200 rounded-lg shadow-lg">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-lg font-semibold">Edit File Details</h3>
        <button onClick={onClose} className="text-gray-500 hover:text-gray-700">
          ×
        </button>
      </div>

      {/* Description */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          onBlur={handleSaveDescription}
          placeholder="Add notes about this file..."
          className="w-full p-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          rows={3}
        />
      </div>

      {/* Tags */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Tags</label>
        <FileTags tags={tags} onAddTag={handleAddTag} onRemoveTag={handleRemoveTag} />
      </div>

      {/* Card Link */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">Linked Card</label>
        {file.card_pk ? (
          <div className="text-sm text-gray-600">Linked to card #{file.card_pk}</div>
        ) : (
          <BacklinkInput addBacklink={handleLinkCard} />
        )}
      </div>

      {saving && <div className="text-sm text-gray-500">Saving...</div>}
    </div>
  );
}
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/components/files/FileMetadataEditor.tsx
git commit -m "Add FileMetadataEditor component for editing file details"
```

---

### Task 14: Add PDF preview component

**Files:**
- Modify: `zettelkasten-front/package.json`
- Create: `zettelkasten-front/src/components/files/FilePreview.tsx`

**Step 1: Install pdf.js library**

Run: `cd zettelkasten-front && npm install react-pdf`

**Step 2: Create preview component**

```typescript
// zettelkasten-front/src/components/files/FilePreview.tsx

import React, { useState } from 'react';
import { Document, Page, pdfjs } from 'react-pdf';
import 'react-pdf/dist/esm/Page/AnnotationLayer.css';
import 'react-pdf/dist/esm/Page/TextLayer.css';

// Configure PDF.js worker
pdfjs.GlobalWorkerOptions.workerSrc = `//unpkg.com/pdfjs-dist@${pdfjs.version}/build/pdf.worker.min.js`;

interface FilePreviewProps {
  fileUrl: string;
  filename: string;
  onClose: () => void;
}

export function FilePreview({ fileUrl, filename, onClose }: FilePreviewProps) {
  const [numPages, setNumPages] = useState<number | null>(null);
  const [pageNumber, setPageNumber] = useState(1);
  const [scale, setScale] = useState(1.0);

  const onDocumentLoadSuccess = ({ numPages }: { numPages: number }) => {
    setNumPages(numPages);
  };

  const goToPrevPage = () => setPageNumber(Math.max(1, pageNumber - 1));
  const goToNextPage = () => setPageNumber(Math.min(numPages || 1, pageNumber + 1));
  const zoomIn = () => setScale(Math.min(2.0, scale + 0.1));
  const zoomOut = () => setScale(Math.max(0.5, scale - 0.1));

  return (
    <div className="fixed inset-0 z-50 bg-black bg-opacity-75 flex items-center justify-center">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-full overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <h3 className="text-lg font-semibold truncate">{filename}</h3>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 text-2xl">
            ×
          </button>
        </div>

        {/* Controls */}
        <div className="flex items-center justify-center gap-4 p-2 bg-gray-100 border-b">
          <button
            onClick={goToPrevPage}
            disabled={pageNumber <= 1}
            className="px-3 py-1 bg-white border rounded disabled:opacity-50"
          >
            Previous
          </button>
          <span className="text-sm">
            Page {pageNumber} of {numPages || '?'}
          </span>
          <button
            onClick={goToNextPage}
            disabled={numPages === null || pageNumber >= numPages}
            className="px-3 py-1 bg-white border rounded disabled:opacity-50"
          >
            Next
          </button>
          <div className="border-l pl-4 flex items-center gap-2">
            <button onClick={zoomOut} className="px-3 py-1 bg-white border rounded">
              −
            </button>
            <span className="text-sm">{Math.round(scale * 100)}%</span>
            <button onClick={zoomIn} className="px-3 py-1 bg-white border rounded">
              +
            </button>
          </div>
        </div>

        {/* PDF Viewer */}
        <div className="flex-1 overflow-auto p-4 bg-gray-200">
          <Document file={fileUrl} onLoadSuccess={onDocumentLoadSuccess}>
            <Page pageNumber={pageNumber} scale={scale} />
          </Document>
        </div>
      </div>
    </div>
  );
}
```

**Step 3: Commit**

```bash
git add zettelkasten-front/package.json zettelkasten-front/package-lock.json zettelkasten-front/src/components/files/FilePreview.tsx
git commit -m "Add PDF preview component using react-pdf"
```

---

### Task 15: Update FileVault page to integrate new features

**Files:**
- Modify: `zettelkasten-front/src/pages/FileVault.tsx`
- Modify: `zettelkasten-front/src/components/files/FileListItem.tsx`

**Step 1: Update FileListItem to show tags and processing status**

```typescript
// Add to zettelkasten-front/src/components/files/FileListItem.tsx

// Add tags and status to the component
interface FileListItemProps {
  file: File;
  onDelete: (file_id: number) => void;
  // ... existing props
}

// In the render:
<div className="flex flex-col">
  <div className="font-medium">{file.name}</div>
  <div className="text-sm text-gray-500">
    {formatBytes(file.size)} • {formatDate(file.created_at)}
  </div>
  {/* Add tags display */}
  {file.tags && file.tags.length > 0 && (
    <div className="flex gap-1 mt-1">
      {file.tags.map((tag) => (
        <span key={tag} className="text-xs text-blue-600">#{tag}</span>
      ))}
    </div>
  )}
  {/* Add processing status */}
  {file.extracted_text === undefined && (
    <span className="text-xs text-yellow-600 mt-1">Processing...</span>
  )}
</div>
```

**Step 2: Update FileVault search placeholder**

```typescript
// In zettelkasten-front/src/pages/FileVault.tsx

// Update search input placeholder
<input
  type="text"
  value={filterString}
  onChange={handleFilter}
  placeholder="Search filename, content, tags..."
  className="..."
/>
```

**Step 3: Add tag filter chips**

```typescript
// Add to filter section in FileVault.tsx

const [availableTags, setAvailableTags] = useState<FileTag[]>([]);
const [selectedTags, setSelectedTags] = useState<string[]>([]);

// Load tags on mount
useEffect(() => {
  getFileTags().then(setAvailableTags).catch(console.error);
}, []);

// Add tag filter UI
<div className="flex flex-wrap gap-2">
  {availableTags.map((tag) => (
    <button
      key={tag.id}
      onClick={() => {
        if (selectedTags.includes(tag.name)) {
          setSelectedTags(selectedTags.filter((t) => t !== tag.name));
        } else {
          setSelectedTags([...selectedTags, tag.name]);
        }
      }}
      className={`px-3 py-1 text-sm rounded-full border ${
        selectedTags.includes(tag.name)
          ? 'bg-blue-600 text-white border-blue-600'
          : 'bg-white text-gray-700 border-gray-300'
      }`}
    >
      #{tag.name} ({tag.file_count})
    </button>
  ))}
</div>
```

**Step 4: Commit**

```bash
git add zettelkasten-front/src/pages/FileVault.tsx zettelkasten-front/src/components/files/FileListItem.tsx
git commit -m "Integrate tags, search enhancements, and processing status in FileVault"
```

---

### Task 16: Add Typesense collection initialization

**Files:**
- Create: `go-backend/scripts/init_typesense_files.go`

**Step 1: Create initialization script**

```go
// go-backend/scripts/init_typesense_files.go

package main

import (
    "log"
    "os"

    "github.com/typesense/typesense-go/typesense"
    "github.com/typesense/typesense-go/typesense/api"
)

func main() {
    client := typesense.NewClient(
        typesense.WithServer(os.Getenv("TYPESENSE_URL")),
        typesense.WithAPIKey(os.Getenv("TYPESENSE_API_KEY")),
    )

    schema := &api.CollectionSchema{
        Name: "files",
        Fields: []api.Field{
            {Name: "id", Type: "int32"},
            {Name: "user_id", Type: "int32", Facet: true},
            {Name: "name", Type: "string"},
            {Name: "description", Type: "string", Optional: true},
            {Name: "extracted_text", Type: "string", Optional: true},
            {Name: "tags", Type: "string[]", Optional: true, Facet: true},
            {Name: "card_pk", Type: "int32", Optional: true, Facet: true},
            {Name: "created_at", Type: "int64"},
        },
    }

    _, err := client.Collections().Create(schema)
    if err != nil {
        log.Fatalf("Failed to create files collection: %v", err)
    }

    log.Println("Successfully created files collection in Typesense")
}
```

**Step 2: Run initialization**

Run: `cd go-backend && go run scripts/init_typesense_files.go`
Expected: "Successfully created files collection in Typesense"

**Step 3: Commit**

```bash
git add go-backend/scripts/init_typesense_files.go
git commit -m "Add Typesense files collection initialization script"
```

---

## Phase 5: Testing & Documentation

### Task 17: Add integration tests

**Files:**
- Create: `go-backend/handlers/files_tags_test.go`

**Step 1: Write integration tests**

```go
// go-backend/handlers/files_tags_test.go

package handlers

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestCreateFileTag(t *testing.T) {
    // Setup test server
    router, db := setupTestRouter(t)
    defer db.Close()

    // Create tag
    body := map[string]string{"name": "taxes"}
    jsonBody, _ := json.Marshal(body)

    req, _ := http.NewRequest("POST", "/files/tags", bytes.NewBuffer(jsonBody))
    req.Header.Set("Content-Type", "application/json")
    setTestUserContext(req, 1)

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.StatusCode)

    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "taxes", response["name"])
}

func TestTagFile(t *testing.T) {
    // Similar test for tagging files
}
```

**Step 2: Run tests**

Run: `cd go-backend && go test ./handlers -run TestFileTag -v`
Expected: PASS

**Step 3: Commit**

```bash
git add go-backend/handlers/files_tags_test.go
git commit -m "Add integration tests for file tag endpoints"
```

---

### Task 18: Update API documentation

**Files:**
- Modify: `docs/api/README.md` or equivalent

**Step 1: Document new endpoints**

```markdown
# File Tags API

## POST /files/tags
Create a new file tag.

**Request:**
```json
{
  "name": "taxes"
}
```

**Response:**
```json
{
  "id": 1,
  "name": "taxes"
}
```

## GET /files/tags
Get all tags for current user with file counts.

**Response:**
```json
[
  {
    "id": 1,
    "name": "taxes",
    "file_count": 5
  }
]
```

## POST /files/:file_id/tags
Add tags to a file.

**Request:**
```json
{
  "tag_names": ["taxes", "2024"]
}
```

## DELETE /files/:file_id/tags/:tag_name
Remove a tag from a file.
```

**Step 2: Commit**

```bash
git add docs/api/README.md
git commit -m "Update API documentation with file tag endpoints"
```

---

## Final Steps

### Task 19: Run full test suite

**Step 1: Run backend tests**

Run: `cd go-backend && go test ./...`
Expected: All tests pass

**Step 2: Run frontend tests**

Run: `cd zettelkasten-front && npm test`
Expected: All tests pass

**Step 3: Commit if any fixes needed**

```bash
git add .
git commit -m "Fix failing tests"
```

---

### Task 20: Manual testing checklist

**Step 1: Test upload and extraction**

- [ ] Upload PDF file → verify "Processing..." status appears
- [ ] Wait 5-10 seconds → verify status changes to "Ready"
- [ ] Search for text inside PDF → verify file appears in results
- [ ] Upload Word doc → verify extraction works
- [ ] Upload Excel file → verify extraction works

**Step 2: Test tag management**

- [ ] Add tag to file → verify tag appears
- [ ] Click tag chip → verify files are filtered
- [ ] Remove tag → verify tag disappears
- [ ] Add duplicate tag → verify no error

**Step 3: Test search**

- [ ] Search by filename → verify results
- [ ] Search by file content → verify results
- [ ] Search by tag name → verify results
- [ ] Combine search + tag filter → verify results

**Step 4: Test PDF preview**

- [ ] Click PDF file → verify preview opens
- [ ] Navigate pages → verify page changes
- [ ] Zoom in/out → verify zoom works
- [ ] Close preview → verify modal closes

**Step 5: Commit final verification**

```bash
git tag -a v1.0-filevault -m "FileVault document management enhancement complete"
git push origin master --tags
```

---

## Success Criteria Checklist

- [ ] Database migrations applied successfully
- [ ] Text extraction works for PDF, Word, Excel
- [ ] Background job processes files within seconds
- [ ] Typesense search returns relevant results
- [ ] Tags can be added/removed from files
- [ ] File filtering by tags works
- [ ] PDF preview displays correctly
- [ ] Processing status indicator shows correctly
- [ ] Search falls back to PostgreSQL if Typesense unavailable
- [ ] All tests pass
- [ ] Manual testing checklist complete

---

## Notes for Implementation

- **Typesense availability:** Ensure Typesense is running before starting
- **Job queue:** Verify job queue is processing jobs
- **File size limits:** Current 10MB limit may need adjustment for large PDFs
- **Async processing:** Users see files immediately, searchability comes seconds later
- **Error handling:** Extraction failures don't block file access
- **Performance:** Monitor Typesense query performance with large document sets
