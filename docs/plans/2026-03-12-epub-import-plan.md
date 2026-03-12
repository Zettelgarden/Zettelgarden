# Epub Import Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable users to import epub files from FileVault, creating a parent book card and child chapter cards with automatic summarization.

**Architecture:** Backend parses epub using go-epub library, extracts metadata and chapters, converts HTML to Markdown, creates cards with parent/child relationships, and triggers async summarization. Frontend adds "Import as Cards" action button for epub files.

**Tech Stack:** Go (go-epub, html-to-markdown), React/TypeScript, existing card and summarization infrastructure.

---

## Task 1: Add go-epub Dependency

**Files:**
- Modify: `go-backend/go.mod`
- Modify: `go-backend/go.sum`

**Step 1: Add go-epub dependency**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go get github.com/bmaupin/go-epub
```

Expected: Dependency added successfully

**Step 2: Verify dependency**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go mod tidy
```

Expected: No errors

**Step 3: Commit**

```bash
git add go-backend/go.mod go-backend/go.sum
git commit -m "chore: add go-epub dependency for epub parsing"
```

---

## Task 2: Create Epub Models

**Files:**
- Create: `go-backend/models/epub.go`

**Step 1: Create epub models file**

```go
package models

// EpubMetadata contains extracted metadata from an epub file
type EpubMetadata struct {
	Title       string
	Author      string
	Publisher   string
	Year        string
	Description string
}

// EpubChapter represents a single chapter from an epub
type EpubChapter struct {
	Title string
	Body  string // Markdown content
}

// ImportEpubResponse is the API response for epub import
type ImportEpubResponse struct {
	ParentCardID int            `json:"parent_card_id"`
	ChildCardIDs []int          `json:"child_card_ids"`
	Metadata     EpubMetadata   `json:"metadata"`
}
```

**Step 2: Commit**

```bash
git add go-backend/models/epub.go
git commit -m "feat: add epub models for import feature"
```

---

## Task 3: Create Epub Service

**Files:**
- Create: `go-backend/services/epub.go`
- Create: `go-backend/services/epub_test.go`

**Step 1: Write failing test for epub parsing**

```go
package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEpub(t *testing.T) {
	// Create a minimal test epub file
	testEpubPath := filepath.Join("testdata", "test.epub")

	// Skip if test file doesn't exist (will be created separately)
	if _, err := os.Stat(testEpubPath); os.IsNotExist(err) {
		t.Skip("Test epub file not found, skipping")
	}

	metadata, chapters, err := ParseEpub(testEpubPath)
	if err != nil {
		t.Fatalf("ParseEpub failed: %v", err)
	}

	if metadata.Title == "" {
		t.Error("Expected non-empty title")
	}

	if len(chapters) == 0 {
		t.Error("Expected at least one chapter")
	}

	for i, chapter := range chapters {
		if chapter.Title == "" {
			t.Errorf("Chapter %d has empty title", i)
		}
		if chapter.Body == "" {
			t.Errorf("Chapter %d has empty body", i)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go test ./services -run TestParseEpub -v
```

Expected: FAIL or SKIP (function doesn't exist)

**Step 3: Create testdata directory and sample epub**

Run:
```bash
mkdir -p /home/nick/code/Zettelgarden/go-backend/services/testdata
```

Note: For testing, we'll create a minimal epub programmatically or download a public domain epub. For now, tests will skip if no test file exists.

**Step 4: Implement ParseEpub function**

```go
package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/bmaupin/go-epub"
	"go-backend/models"
)

// ParseEpub reads an epub file and extracts metadata and chapters
func ParseEpub(filePath string) (models.EpubMetadata, []models.EpubChapter, error) {
	var metadata models.EpubMetadata
	var chapters []models.EpubChapter

	// Open the epub file
	book, err := epub.Open(filePath)
	if err != nil {
		return metadata, chapters, fmt.Errorf("failed to open epub: %w", err)
	}
	defer book.Close()

	// Extract metadata
	metadata = extractMetadata(book)

	// Extract chapters from TOC
	chapters, err = extractChapters(book)
	if err != nil {
		return metadata, chapters, fmt.Errorf("failed to extract chapters: %w", err)
	}

	return metadata, chapters, nil
}

// extractMetadata pulls book metadata from the epub
func extractMetadata(book *epub.Epub) models.EpubMetadata {
	return models.EpubMetadata{
		Title:       book.Metadata().Title,
		Author:      strings.Join(book.Metadata().Creators, ", "),
		Publisher:   book.Metadata().Publisher,
		Year:        extractYear(book.Metadata().Date),
		Description: book.Metadata().Description,
	}
}

// extractYear parses year from date string
func extractYear(date string) string {
	if date == "" {
		return ""
	}
	// Try to extract 4-digit year
	re := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	return re.FindString(date)
}

// extractChapters gets all chapters from the epub's TOC
func extractChapters(book *epub.Epub) ([]models.EpubChapter, error) {
	var chapters []models.EpubChapter

	// Get the table of contents
	toc := book.Toc()
	if len(toc) == 0 {
		// Fall back to scanning all spine items
		return extractChaptersFromSpine(book)
	}

	// Create markdown converter
	mdConverter := htmltomarkdown.NewConverter(
		htmltomarkdown.WithHeadingStyle(htmltomarkdown.ATXHeading),
	)

	for _, tocItem := range toc {
		chapter, err := extractChapter(book, tocItem, mdConverter)
		if err != nil {
			// Log error but continue with other chapters
			continue
		}
		if chapter.Body != "" {
			chapters = append(chapters, chapter)
		}
	}

	// If no chapters found from TOC, try spine
	if len(chapters) == 0 {
		return extractChaptersFromSpine(book)
	}

	return chapters, nil
}

// extractChapter extracts a single chapter's content
func extractChapter(book *epub.Epub, tocItem epub.TocItem, mdConverter *htmltomarkdown.Converter) (models.EpubChapter, error) {
	var chapter models.EpubChapter
	chapter.Title = tocItem.Title

	// Get the content file
	content, err := book.GetContent(tocItem.Href)
	if err != nil {
		return chapter, fmt.Errorf("failed to get chapter content: %w", err)
	}

	// Convert HTML to Markdown
	markdown, err := mdConverter.ConvertString(content)
	if err != nil {
		// If conversion fails, use raw content
		chapter.Body = content
	} else {
		chapter.Body = markdown
	}

	// Clean up the body
	chapter.Body = strings.TrimSpace(chapter.Body)

	return chapter, nil
}

// extractChaptersFromSpine is a fallback when TOC is not available
func extractChaptersFromSpine(book *epub.Epub) ([]models.EpubChapter, error) {
	var chapters []models.EpubChapter

	mdConverter := htmltomarkdown.NewConverter(
		htmltomarkdown.WithHeadingStyle(htmltomarkdown.ATXHeading),
	)

	// Get all spine items
	spine := book.Spine()
	chapterNum := 0

	for _, spineItem := range spine {
		content, err := book.GetContent(spineItem.Href)
		if err != nil {
			continue
		}

		// Try to extract title from content
		title := extractTitleFromHTML(content)
		if title == "" {
			chapterNum++
			title = fmt.Sprintf("Chapter %d", chapterNum)
		}

		// Convert to markdown
		markdown, err := mdConverter.ConvertString(content)
		body := markdown
		if err != nil {
			body = content
		}

		body = strings.TrimSpace(body)
		if body != "" && len(body) > 100 { // Skip very short sections (likely front matter)
			chapters = append(chapters, models.EpubChapter{
				Title: title,
				Body:  body,
			})
		}
	}

	return chapters, nil
}

// extractTitleFromHTML tries to find a heading in HTML content
func extractTitleFromHTML(htmlContent string) string {
	// Look for h1 or h2 tags
	h1Regex := regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	h2Regex := regexp.MustCompile(`<h2[^>]*>([^<]+)</h2>`)

	if matches := h1Regex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	if matches := h2Regex.FindStringSubmatch(htmlContent); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}
```

**Step 5: Run tests**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go test ./services -run TestParseEpub -v
```

Expected: PASS (or SKIP if no test epub)

**Step 6: Commit**

```bash
git add go-backend/services/epub.go go-backend/services/epub_test.go
git commit -m "feat: add epub parsing service"
```

---

## Task 4: Create Import Epub Handler

**Files:**
- Create: `go-backend/handlers/epub.go`

**Step 1: Create the import handler**

```go
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

// ImportEpubRequest is the request body for epub import (empty for now)
type ImportEpubRequest struct{}

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
	var file models.File
	var mimetype string
	err = h.DB.QueryRow(`
		SELECT id, name, filepath, mimetype
		FROM files
		WHERE id = $1 AND user_id = $2
	`, fileID, userID).Scan(&file.ID, &file.Name, &file.Filepath, &mimetype)

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
	metadata, chapters, err := services.ParseEpub(file.Filepath)
	if err != nil {
		log.Printf("Failed to parse epub: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse epub: %v", err), http.StatusInternalServerError)
		return
	}

	// Create cards in a transaction
	response, err := h.createEpubCards(userID, metadata, chapters, file.Name)
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
	}

	if err := tx.Commit(); err != nil {
		return response, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Trigger summarization for each child card (async)
	for _, cardID := range response.ChildCardIDs {
		card := models.Card{
			ID:    cardID,
			Title: chapters[0].Title, // Title will be fetched properly in ProcessEntitiesAndFacts
		}
		// Find the matching chapter for this card
		for i, id := range response.ChildCardIDs {
			if id == cardID && i < len(chapters) {
				card.Title = chapters[i].Title
				card.Body = chapters[i].Body
				break
			}
		}
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
```

**Step 2: Verify compilation**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go build ./...
```

Expected: No errors

**Step 3: Commit**

```bash
git add go-backend/handlers/epub.go
git commit -m "feat: add import epub handler"
```

---

## Task 5: Register Route

**Files:**
- Modify: `go-backend/routes/routes.go`

**Step 1: Add route registration**

Find the existing file routes section and add the epub import route. Look for where other `/files` routes are registered.

Add after other file routes:
```go
router.HandleFunc("/files/{id}/import-epub", handler.ImportEpubRoute).Methods("POST")
```

**Step 2: Verify compilation**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go build ./...
```

Expected: No errors

**Step 3: Commit**

```bash
git add go-backend/routes/routes.go
git commit -m "feat: register epub import route"
```

---

## Task 6: Add Frontend API Function

**Files:**
- Modify: `zettelkasten-front/src/api/files.ts`
- Modify: `zettelkasten-front/src/models/File.ts`

**Step 1: Add response type to File.ts**

Add to `zettelkasten-front/src/models/File.ts`:
```typescript
export interface ImportEpubResponse {
  parent_card_id: number;
  child_card_ids: number[];
  metadata: {
    title: string;
    author: string;
    publisher: string;
    year: string;
    description: string;
  };
}
```

**Step 2: Add API function to files.ts**

Add to `zettelkasten-front/src/api/files.ts`:
```typescript
import { ImportEpubResponse } from "../models/File";

export function importEpub(fileId: number): Promise<ImportEpubResponse> {
  return getData(apiClient.post<ImportEpubResponse>(`/files/${fileId}/import-epub`));
}
```

**Step 3: Verify TypeScript compilation**

Run:
```bash
cd /home/nick/code/Zettelgarden/zettelkasten-front && npm run build
```

Expected: No TypeScript errors

**Step 4: Commit**

```bash
git add zettelkasten-front/src/api/files.ts zettelkasten-front/src/models/File.ts
git commit -m "feat: add importEpub API function"
```

---

## Task 7: Add Import Button to FileListItem

**Files:**
- Modify: `zettelkasten-front/src/components/files/FileListItem.tsx`

**Step 1: Add import button for epub files**

The FileListItem component needs to:
1. Check if file mimetype is `application/epub+zip`
2. Show "Import as Cards" button
3. Handle loading state and API call
4. Show success/error toast

Add import state and handler:
```typescript
// Add to imports
import { importEpub } from "../../api/files";

// Add state in component
const [isImporting, setIsImporting] = useState(false);

// Add handler
const handleImportEpub = async () => {
  setIsImporting(true);
  try {
    const result = await importEpub(file.id);
    showToast("success", "Epub Imported",
      `Created ${result.child_card_ids.length + 1} cards from "${result.metadata.title || file.name}"`);
  } catch (error) {
    showToast("error", "Import Failed",
      error instanceof Error ? error.message : "Failed to import epub");
  } finally {
    setIsImporting(false);
  }
};

// Add button in the actions area (conditionally for epub files)
{file.mimetype === "application/epub+zip" && (
  <button
    onClick={handleImportEpub}
    disabled={isImporting}
    className="text-sm text-blue-600 hover:text-blue-800 disabled:opacity-50"
  >
    {isImporting ? "Importing..." : "Import as Cards"}
  </button>
)}
```

**Step 2: Verify TypeScript compilation**

Run:
```bash
cd /home/nick/code/Zettelgarden/zettelkasten-front && npm run build
```

Expected: No TypeScript errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/files/FileListItem.tsx
git commit -m "feat: add Import as Cards button for epub files"
```

---

## Task 8: Integration Testing

**Files:**
- No new files (manual testing)

**Step 1: Start the backend**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go run ./main.go
```

**Step 2: Start the frontend**

Run:
```bash
cd /home/nick/code/Zettelgarden/zettelkasten-front && npm run start
```

**Step 3: Test the feature manually**

1. Upload an epub file to FileVault
2. Verify "Import as Cards" button appears
3. Click the button
4. Verify success toast shows
5. Navigate to Cards page
6. Verify parent book card exists with metadata
7. Verify child chapter cards exist with parent linkage
8. Wait for summaries to generate on chapter cards

**Step 4: Document any issues found**

If issues are found, create fix-up commits as needed.

---

## Task 9: Final Cleanup and Documentation

**Files:**
- Modify: `docs/plans/2026-03-12-epub-import-design.md` (if needed)

**Step 1: Run full test suite**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go test ./...
cd /home/nick/code/Zettelgarden/zettelkasten-front && npm run test:coverage
```

Expected: All tests pass

**Step 2: Run linters**

Run:
```bash
cd /home/nick/code/Zettelgarden/go-backend && go fmt ./...
cd /home/nick/code/Zettelgarden/zettelkasten-front && npm run lint
```

**Step 3: Final commit (if any changes)**

```bash
git add -A
git commit -m "chore: cleanup and formatting for epub import feature"
```

---

## Summary

This plan creates:
- Backend epub parsing service using go-epub
- API endpoint for importing epub files
- Frontend "Import as Cards" button in FileVault
- Parent/child card structure with automatic summarization

Total estimated tasks: 9
Estimated time: 2-3 hours
