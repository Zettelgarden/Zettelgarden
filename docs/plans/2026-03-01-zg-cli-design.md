# zg CLI Design Document

**Date:** 2026-03-01
**Status:** Draft
**Purpose:** Standalone CLI tool for Zettelgarden, designed for AI agent usage

## Overview

A standalone Go-based CLI tool (`zg`) that provides card and task CRUD operations via the Zettelgarden REST API. Designed as a more stable alternative to the MCP server for AI agent integration.

**Key Goals:**
- Simple, predictable interface for AI agents
- Single binary distribution
- JSON-first output for easy parsing
- Easy to update and maintain

## Architecture

```
zg/
├── cmd/zg/              # Main entry point
│   └── main.go          # CLI setup, command routing
├── internal/
│   ├── api/             # HTTP client for backend API
│   ├── config/          # Config file handling
│   ├── cmd/             # Command implementations
│   │   ├── card.go      # Card commands
│   │   └── task.go      # Task commands
│   └── output/          # JSON formatting/output
├── go.mod
└── README.md
```

**Design decisions:**
- Standalone binary, no shared code with backend
- Cobra for CLI framework (standard for Go CLIs)
- Structured JSON output for AI parsing
- Zero exit codes (errors in JSON response)

## Configuration

**Location:** `~/.config/zettelgarden/config.json`

**Schema:**
```json
{
  "api_url": "http://localhost:8080",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "timeout_seconds": 30
}
```

**Environment override:**
- `ZETTELGARDEN_TOKEN` - Override token
- `ZETTELGARDEN_API_URL` - Override API URL

## Command Structure

Git-style command structure:

```bash
# Card operations
zg card get <id>
zg card list [--limit N] [--offset N] [--starred]
zg card create --title "Title" --body "Body"
zg card update <id> [--title "New"] [--body "New"]
zg card delete <id>
zg card search <query> [--full-text] [--limit N]

# Task operations
zg task get <id>
zg task list [--completed] [--scheduled-date YYYY-MM-DD] [--priority high|medium|low]
zg task create --title "Title" [--scheduled-date YYYY-MM-DD] [--priority high]
zg task update <id> [--title "New"] [--is-complete true]
zg task delete <id>
zg task complete <id>

# Global flags
--pretty           # Pretty-print JSON
--config <path>    # Custom config path
--url <url>        # Override API URL
--token <token>    # Override auth token
```

## Output Format

All commands return structured JSON:

**Success:**
```json
{
  "success": true,
  "data": { /* card or task object */ }
}
```

**Error (still exits 0):**
```json
{
  "success": false,
  "error": "Error message",
  "details": "Additional context"
}
```

**List response:**
```json
{
  "success": true,
  "data": [ /* array */ ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

## API Integration

Reuses existing backend REST endpoints with JWT authentication.

**Cards:**
- `GET /api/user/cards/{id}`
- `GET /api/user/cards`
- `POST /api/user/cards`
- `PUT /api/user/cards/{id}`
- `DELETE /api/user/cards/{id}`
- `GET /api/user/search`

**Tasks:**
- `GET /api/user/tasks/{id}`
- `GET /api/user/tasks`
- `POST /api/user/tasks`
- `PUT /api/user/tasks/{id}`
- `DELETE /api/user/tasks/{id}`

## Phase 1 Scope (MVP)

- Card CRUD operations
- Task CRUD operations
- Basic list filtering
- Search functionality

## Future Considerations (Out of Scope)

- Template commands
- Schema commands
- Article import
- Calendar integration
- Bulk operations
- Interactive mode

## Build and Distribution

```bash
# Build
go build -o zg ./cmd/zg

# Cross-platform builds
go build -o zg-linux-amd64 -GOOS=linux -GOARCH=amd64 ./cmd/zg
go build -o zg-darwin-arm64 -GOOS=darwin -GOARCH=arm64 ./cmd/zg

# Install
cp zg /usr/local/bin/
```
