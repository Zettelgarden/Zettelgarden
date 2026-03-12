# Epub Import to Cards Feature Design

**Date:** 2026-03-12
**Status:** Approved

## Overview

Allow users to import epub files from the FileVault, automatically creating a structured set of cards: one parent "book" card with metadata, and child cards for each chapter containing the full text. Each chapter card will be automatically summarized using the existing summarization pipeline.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (FileVault)                      │
│  ┌─────────────┐                                                 │
│  │ epub file   │──▶ "Import as Cards" button                    │
│  └─────────────┘           │                                     │
└─────────────────────────────┼───────────────────────────────────┘
                              │ POST /files/{id}/import-epub
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Go Backend                                │
│  ┌─────────────────┐                                            │
│  │ ImportEpubRoute │                                            │
│  └────────┬────────┘                                            │
│           │                                                      │
│           ▼                                                      │
│  ┌─────────────────┐     ┌──────────────────┐                   │
│  │ Parse EPUB      │────▶│ Extract metadata │                   │
│  │ (go-epub)       │     │ - Title, Author  │                   │
│  └─────────────────┘     │ - Publisher, Year│                   │
│           │              │ - Description     │                   │
│           ▼              └──────────────────┘                   │
│  ┌─────────────────┐                                           │
│  │ Extract chapters│                                           │
│  │ (TOC + content) │                                           │
│  └────────┬────────┘                                           │
│           │                                                      │
│           ▼                                                      │
│  ┌─────────────────────────────────────────┐                   │
│  │ Create Cards (transactional)             │                   │
│  │ 1. Parent card: Book metadata            │                   │
│  │ 2. Child cards: One per chapter          │                   │
│  │    - Title from chapter name              │                   │
│  │    - Body from chapter text (HTML→MD)     │                   │
│  │    - parent_id linked to book card        │                   │
│  └────────┬────────────────────────────────┘                   │
│           │                                                      │
│           ▼                                                      │
│  ┌─────────────────────────────────────────┐                   │
│  │ Trigger summarization per child card     │                   │
│  │ (ProcessEntitiesAndFacts - async)        │                   │
│  └─────────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────┘
```

## API Specification

### New Endpoint

```
POST /files/{id}/import-epub

Request: (empty body)

Response (200 OK):
{
  "parent_card_id": 123,
  "child_card_ids": [124, 125, 126],
  "metadata": {
    "title": "Book Title",
    "author": "Author Name",
    "chapters_imported": 12
  }
}

Error Responses:
- 400 Bad Request: File is not an epub (wrong mimetype)
- 404 Not Found: File not found or doesn't belong to user
- 500 Internal Server Error: Epub parsing failed
```

## Data Model

### Card Structure

| Card | Title | Body | parent_id |
|------|-------|------|-----------|
| Book (parent) | Book Title | Description + metadata | null |
| Chapter 1 | "Chapter 1: Introduction" | Full chapter text (markdown) | book_id |
| Chapter 2 | "Chapter 2: ..." | Full chapter text (markdown) | book_id |

### Book Card Body Format

```markdown
> Author: Author Name  
> Publisher: Publisher  
> Year: 2024

Description text from epub metadata...

## Chapters
- Chapter 1: Introduction
- Chapter 2: Background
- ...
```

## Frontend Changes

### FileVault.tsx

1. **New action button:** "Import as Cards" appears in FileListItem for files with mimetype `application/epub+zip`

2. **Import handler:**
   - Calls `importEpub(fileId)` API
   - Shows success toast with card count
   - Simple loading spinner during import (2-5 seconds typically)

3. **New API function in files.ts:**
   ```typescript
   export function importEpub(fileId: number): Promise<ImportEpubResponse>
   ```

## Backend Implementation

### New Handler: `handlers/epub.go`

```go
func (h *Handler) ImportEpubRoute(w http.ResponseWriter, r *http.Request)
```

**Processing steps:**
1. Validate file mimetype is `application/epub+zip`
2. Parse epub using `github.com/bmaupin/go-epub`
3. Extract metadata: title, author, publisher, year, description
4. Extract chapters from TOC (NCX/NAV file)
5. Convert chapter HTML to Markdown (existing `html-to-markdown` library)
6. Create parent book card with metadata
7. Batch create child chapter cards with parent_id linkage
8. Trigger `ProcessEntitiesAndFacts` for each child card (async)
9. Return response with card IDs

### Chapter Extraction Logic

- Use epub's NCX/NAV file to get chapter structure
- Fall back to scanning for `<h1>`, `<h2>` headings if no TOC
- Skip front matter (cover page, copyright, etc.)
- Unnamed chapters become "Chapter N"

### Route Registration

```go
router.HandleFunc("/files/{id}/import-epub", handler.ImportEpubRoute).Methods("POST")
```

## Dependencies

### New Go Dependency

- `github.com/bmaupin/go-epub` - For epub parsing and chapter extraction

### Existing Dependencies (Reused)

- `github.com/JohannesKaufmann/html-to-markdown/v2` - Already used in cards.go
- Existing summarization pipeline (`ProcessEntitiesAndFacts`)
- Existing card creation and parent/child relationships

## Out of Scope (Future Enhancements)

- Cover image extraction (can be added later)
- Configurable import options (include/exclude original text)
- Progress tracking for very large epubs
- Support for other ebook formats (mobi, azw3)

## User Flow

1. User uploads epub to FileVault (existing flow)
2. User sees epub file in the file list
3. User clicks "Import as Cards" button
4. Spinner shows briefly (2-5 seconds)
5. Success toast: "Created 13 cards from 'Book Title'"
6. User navigates to Cards page to see book card and chapter cards
7. Summaries appear on each chapter card as they complete (async, existing behavior)
