# FileVault Document Management Enhancement Design

**Date:** March 9, 2026
**Status:** Approved
**Approach:** Balanced Enhancement (Approach 2)

## Overview

Transform FileVault into a full document organizer with full-text search, tags, and previews. Enable users to centralize personal and work documents in Zettelgarden alongside their notes.

## Goals

- **Find documents quickly** - Full-text search across file contents
- **Flexible organization** - Tags + card linking
- **Better usability** - In-browser previews, metadata editing
- **Centralized knowledge** - Documents and notes in one place

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│ FileVault.tsx (Frontend)                                │
│ - Upload files                                          │
│ - Link to cards                                         │
│ - Add tags to files                                     │
│ - Search (filename + full-text)                         │
│ - Preview documents                                     │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────┐
│ Go Backend (handlers/files.go)                          │
│ - Handle upload                                         │
│ - Enqueue text extraction jobs                          │
│ - Query Typesense for search                            │
│ - Manage tags                                           │
└─────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────┬───────────────────────────────────┐
│ PostgreSQL          │ Typesense (already in stack)      │
│ - File metadata     │ - Full-text index                 │
│ - Tags              │ - Search API                      │
│ - Card links        │                                   │
│ - Extracted text    │                                   │
└─────────────────────┴───────────────────────────────────┘
```

## Database Schema Changes

### New Tables

```sql
-- File tags (user-specific)
CREATE TABLE file_tags (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);

-- Files to tags junction (many-to-many)
CREATE TABLE files_tags (
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES file_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);
```

### Columns Added to `files` Table

```sql
ALTER TABLE files ADD COLUMN description TEXT;
ALTER TABLE files ADD COLUMN extracted_text TEXT;
```

## Typesense Collection

```javascript
{
  name: 'files',
  fields: [
    { name: 'id', type: 'int32' },
    { name: 'user_id', type: 'int32' },
    { name: 'name', type: 'string' },           // filename
    { name: 'description', type: 'string' },    // user notes
    { name: 'extracted_text', type: 'string' }, // text from PDF/doc
    { name: 'tags', type: 'string[]' },         // array of tag names
    { name: 'card_pk', type: 'int32' },         // linked card
    { name: 'created_at', type: 'int64' }
  ]
}
```

## Upload Flow with Async Processing

```
1. User uploads file → Go backend receives it
2. Save file to S3 (current behavior)
3. Create file record in DB (extracted_text = NULL)
4. Enqueue text extraction job:
   - JobType: "file_text_extraction"
   - Payload: {file_id: 123, s3_key: "..."}
5. Return success immediately (file visible but not yet searchable)

Background Job:
1. Worker picks up job
2. Download file from S3
3. Extract text based on file type
4. Update files SET extracted_text = "..."
5. Index in Typesense
6. Mark job complete
```

## Text Extraction

### Supported File Types

| File Type | Library | Notes |
|-----------|---------|-------|
| PDF | `github.com/ledongthuc/pdf` | Pure Go, no external deps |
| Word (.docx) | `github.com/nguyenthenguyen/docx` | Extracts text from DOCX |
| Excel (.xlsx) | `github.com/xuri/excelize/v2` | Read cell values |
| Plain text | Built-in | Direct read |
| Images | Skip | No OCR in this approach |

### Edge Cases

- **Password-protected PDF** → Skip extraction, mark as "not searchable"
- **Corrupted file** → Job fails after 3 retries, mark as "extraction_failed"
- **Very large file** → Truncate to first 10,000 characters

## Search Flow

```
User searches for "mortgage payment"
        ↓
Frontend: GET /files?search=mortgage+payment
        ↓
Backend checks Typesense:
  - QueryBy: name,description,extracted_text,tags
  - FilterBy: user_id
  - SortBy: _text_match:desc,created_at:desc
        ↓
Fallback to PostgreSQL if Typesense unavailable:
  - WHERE name ILIKE '%mortgage%'
  - OR extracted_text ILIKE '%mortgage%'
  - OR description ILIKE '%mortgage%'
```

## Reindexing Triggers

| Trigger | Action |
|---------|--------|
| File uploaded | Enqueue job → index after extraction |
| File description edited | Immediately reindex |
| Tags added/removed | Immediately reindex |
| File deleted | Remove from index |
| Card link changed | Immediately reindex |

## Frontend Components

### New Components

1. **FileTags.tsx** - Tag management UI
   - Tag input with autocomplete
   - Add/remove tags on files
   - Click tag to filter files
   - Bulk tag editing

2. **FileMetadataEditor.tsx** - Edit file details
   - Edit description/notes
   - Add/remove tags
   - Link/unlink from card

3. **FilePreview.tsx** - PDF viewer
   - Use pdf.js via react-pdf
   - In-browser rendering
   - Mobile-friendly

### Enhanced Components

1. **FileVault.tsx**
   - Enhanced search bar ("Search filename, content, tags...")
   - Processing status indicator on file items
   - Tag chips on file list items

2. **FileListItem.tsx**
   - Show tags
   - Show description preview
   - Show processing status badge

## UI Layout

```
┌─────────────────────────────────────────────────────────┐
│ Files                                    [Upload File] │
├─────────────────────────────────────────────────────────┤
│ [Search filename, content, tags...] [?]                │
│                                                         │
│ Filter: [All] [PDF] [Images] [Unlinked] [Processing]   │
│                                                         │
│ ☐ Select all                                            │
├─────────────────────────────────────────────────────────┤
│ ☐ 📄 tax-return-2024.pdf        Processing...          │
│    2.3 MB • Mar 10, 2024                               │
│    #taxes #2024 • Linked to: House                      │
│                                                         │
│ ☐ 📄 mortgage-statement.pdf     ✓ Ready                │
│    156 KB • Mar 5, 2024                                │
│    #mortgage • No card link                            │
│                                                         │
│ ☐ 📄 contractor-contract.docx   ✓ Ready                │
│    89 KB • Feb 28, 2024                                │
│    #house #renovation • Linked to: Kitchen Reno        │
└─────────────────────────────────────────────────────────┘
```

## Error Handling

### Upload Errors

| Scenario | Handling |
|----------|----------|
| File too large (>10MB) | Reject immediately, show error toast |
| Unsupported file type | Upload succeeds, skip extraction |
| S3 upload fails | Return error, don't create DB record |

### Extraction Errors

| Scenario | Handling |
|----------|----------|
| Corrupted file | Job fails, retry up to 3 times, mark "extraction_failed" |
| Password-protected PDF | Skip extraction, mark "not searchable" |
| Empty document | Extract empty string, index anyway |

### Search Errors

| Scenario | Handling |
|----------|----------|
| Typesense down | Fallback to PostgreSQL LIKE search |
| No results | Show "No files found" with suggestions |
| Slow query (>2s) | Show "Searching..." indicator |

### Tag Errors

| Scenario | Handling |
|----------|----------|
| Duplicate tag | Ignore, don't create duplicate |
| Special characters | Sanitize (letters, numbers, hyphens only) |
| Tag too long | Truncate to 50 characters |
| Delete tag with files | Remove tag from all files, delete tag |

## Implementation Files

### Backend (Go)

- `go-backend/models/job.go` - Add `JobTypeFileTextExtraction`
- `go-backend/services/jobs/file_text_extraction_job.go` - New worker
- `go-backend/handlers/files.go` - Enhanced upload, search, tag endpoints
- `go-backend/migrations/` - Schema migrations for tags

### Frontend (React/TypeScript)

- `zettelkasten-front/src/components/files/FileTags.tsx` - New
- `zettelkasten-front/src/components/files/FileMetadataEditor.tsx` - New
- `zettelkasten-front/src/components/files/FilePreview.tsx` - New
- `zettelkasten-front/src/pages/FileVault.tsx` - Enhanced
- `zettelkasten-front/src/api/files.ts` - Enhanced API client
- `zettelkasten-front/src/models/File.ts` - Updated model

## Success Criteria

- [ ] Can upload PDF/Word/Excel files and have them searchable within seconds
- [ ] Can add tags to files and filter by tags
- [ ] Can search across filename, content, and tags
- [ ] Can view PDFs in browser without downloading
- [ ] Can add notes/descriptions to files
- [ ] Processing status visible on file list
- [ ] Search falls back gracefully if Typesense unavailable

## Future Enhancements (Out of Scope)

- OCR for scanned documents/images
- File versioning
- Collections/folders hierarchical organization
- Document signing
- Advanced metadata extraction (dates, amounts, parties)
