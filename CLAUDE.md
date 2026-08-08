# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Zettelgarden is a human-centric, open-source personal knowledge management system built on zettelkasten principles. It's a full-stack application with four main components:

- **Frontend**: React/TypeScript with Vite (`zettelkasten-front/`)
- **Backend**: Go API server (`go-backend/`), SQLite-only
- **Mail**: Transactional email sent directly from the Go backend over SMTP (optional)
- **Desktop + Sync**: Tauri v2 shell (`desktop/`) and TypeScript offline-first sync engine (`sync-engine/`) — see epic Zettelgarden-v5b

## Development Commands

### Frontend (zettelkasten-front/)
```bash
cd zettelkasten-front
npm start          # Start development server (Vite)
npm run build      # Build for production (TypeScript compilation + Vite build)
npm test           # Run tests with Vitest
npm run serve      # Preview production build
```

### Backend (go-backend/)
```bash
cd go-backend
go run main.go     # Start development server
source .env-bash && go test ./...      # Run all tests
go build -o main   # Build binary
```

### Full Stack Development
```bash
# Build and deploy all services
./build.sh         # Builds Docker images and deploys via SSH

# Local development with Docker
docker-compose up  # Start all services locally
```

## Architecture

### Backend (Go)
- **Main**: `main.go` - HTTP server setup with JWT middleware, CORS, and route definitions
- **Handlers**: `handlers/` - HTTP route handlers organized by feature (auth, cards, tasks, files, etc.)
- **Models**: `models/` - Database models and business logic
- **Server**: `server/` - Database connections and server configuration
- **Migrations**: `schema/` - SQL migration files for database schema
- **LLMs**: `llms/` - AI/ML integration for embeddings, chat, and entity processing

### Frontend (React/TypeScript)
- **Pages**: `src/pages/` - Main application routes and page components
- **Components**: `src/components/` - Reusable UI components organized by feature
- **Contexts**: `src/contexts/` - React context providers for state management
- **API**: `src/api/` - HTTP client functions for backend communication
- **Models**: `src/models/` - TypeScript type definitions

### Key Features
- **Cards**: Atomic notes with markdown support, backlinking, starring, and AI-powered summaries/analysis
  - Card hierarchies with parent-child relationships
  - Multiple view modes: normal, summary, and analysis views
  - Linked entities and references tracking
  - Tabbed interface for files, facts, and metadata
- **Tasks**: Task management with recurring capability, priorities, and scheduling
  - Today's task counter in sidebar
  - Task creation shortcuts and dialogs
  - Task tagging and filtering
- **Files**: File upload/storage on local disk under `STORAGE_DIR` with card attachment
- **RSS Feed Client**: Subscribe to RSS/Atom feeds with auto-tagging support
  - Feeds: Browse and manage RSS/Atom feed subscriptions
  - Articles: Reader-style inbox for fetched articles
  - Conversion: Selectively convert interesting articles to cards
  - Folders: Organize feeds into folders for better navigation
  - Scheduled Fetch: Background job fetches new articles every 60 minutes
  - Starring: Star/unstar articles for later reference
    - Star icon in article list and reader view
    - Dedicated Starred feed in sidebar
    - Filtered API endpoint for starred articles
- **Search**: Vector search with embeddings, traditional text search, and starred searches
  - Quick search functionality with keyboard shortcuts
  - Search result starring and management
- **Entities**: Named entity recognition, management, and linking (PRO feature)
  - Entity dialogs for viewing and editing
  - Entity-card relationship tracking
- **Facts**: Structured fact management and storage (PRO feature)
- **Starring**: Bookmark system for both cards and searches with sidebar management
- **Templates**: Card templates with variable substitution
- **Keyboard Shortcuts**: 'c' (create card), 't' (create task), 's' (search)
- **Subscription Features**: PRO gating for advanced features like entities and facts
- **Admin**: Administrative interface for managing users, jobs, and system operations
  - Job Queue monitoring and management
  - Scheduled Jobs Admin
    - View all registered scheduled jobs with schedules
    - Monitor job execution status and history
    - Per-job statistics (success rate, recent runs)
    - Expandable history with pagination
    - Manual refresh for current status
  - User management and details

### Database
- SQLite (file-based, WAL mode) — the only supported database; no external DB server
- Typesense as an optional search cache with graceful fallback when not configured
- Migration-based schema management in `go-backend/schema/sqlite/` (consolidated schema; historical Postgres migrations live under `go-backend/schema/archive/postgres/`)
- Models use database/sql with manual query construction

### Authentication & Authorization
- JWT-based authentication with middleware in `main.go`
- Admin-only routes protected by admin middleware
- User context passed through request context

### AI/ML Integration
- OpenAI-compatible LLM client for chat and embeddings
- Vector search available through Typesense
- Entity extraction and processing pipeline

### Testing
- Go: Standard `testing` package with test helpers in `handlers/test_helpers.go`
- Frontend: Vitest with React Testing Library setup
- Test data in `tests/` directory

## Environment Configuration

The application requires environment configuration for:
- Database connection (SQLITE_PATH, default `./data/zettelgarden.db`)
- Local file storage (STORAGE_DIR, default `./data/files`)
- JWT secret (SECRET_KEY)
- Calendar encryption (CALENDAR_ENCRYPTION_KEY)
  - Purpose: Encryption key for calendar passwords (AES-256-GCM)
  - Requirements: 32+ character random string
  - Generation command: `openssl rand -base64 32`
  - When required: For authenticated calendar feeds
- LLM integration (ZETTEL_LLM_KEY, ZETTEL_LLM_ENDPOINT)
- Stripe billing (optional — `STRIPE_ENABLED=false` disables it; PRO features unlocked)
- Uptime Kuma integration (UPTIME_KUMA_PUSH_URL)
  - Purpose: Push monitor URL for heartbeat signals to verify job scheduler is operational
  - Format: Full URL including monitor UUID, e.g.: https://uptime.example.com/api/push/YOUR_MONITOR_ID?status=up&msg=OK
  - When required: Optional - job will gracefully skip if not configured
- RSS feed fetching (RSS_FETCH_INTERVAL_MINUTES)
  - Purpose: Interval in minutes for RSS feed fetching
  - Default: 60 minutes
  - When required: Optional - uses default if not configured
- Mail service configuration (SMTP_HOST/SMTP_PORT/SMTP_USERNAME/SMTP_PASSWORD/SMTP_FROM — optional; mail is a no-op when unset)
- OIDC / SSO (OIDC_* vars, optional) — see `.env.example`

## Development Notes

- The frontend uses Vite for fast development builds
- Backend follows RESTful API conventions with consistent error handling
- All routes are logged via `handlers.LogRoute` middleware
- File uploads go to local disk under STORAGE_DIR (no external object store)
- The application supports both development and production logging configurations
- Desktop shell: `cd desktop && npm run dev` (Tauri v2, requires Rust toolchain; see `desktop/README.md`)
- Sync engine: TypeScript offline-first sync (local SQLite mirror + outbox); live spec is `docs/plans/2026-08-07-mobile-desktop-sync-app-design.md`

We track work in Beads instead of Markdown. Run `bd quickstart` to see how.
Use 'bd' for task tracking


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
