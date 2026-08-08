# Zettelgarden

**Build Understanding, Not Just Notes**

Zettelgarden is a self-hosted, open-source knowledge management system built on
zettelkasten methodology, with AI-assisted discovery. It combines atomic
notes, bidirectional linking, tasks, RSS, and a local-first sync engine — all
backed by a single SQLite database.

> **Note:** This project is actively evolving. While stable for personal use,
> features may change based on development priorities.

## Why Zettelgarden?

- **Zettelkasten methodology** — atomic cards, `[[wiki-links]]`, backlinks, and a
  living knowledge graph.
- **AI as an augmentation** — optional LLM integration (OpenAI-compatible
  endpoints) for summaries, entity extraction, and semantic search.
- **Self-hosted by default** — SQLite file database (no external DB server),
  local on-disk file storage, optional SMTP mail, optional Stripe billing.
  Everything runs in one Docker container or two dev processes.
- **No vendor lock-in** — full data export from Settings, self-serve account
  deletion, plain JSON/Markdown/CSV output, and your original files.

## Features

### Core knowledge management
- **Atomic Cards** — Markdown-supported notes with unique identifiers and
  `[[card-title]]` bidirectional linking with backlink display.
- **Tasks** — Integrated todo system with scheduling, priorities, subtasks,
  recurring patterns, and saved searches.
- **Files** — Upload and organize PDFs, images, and documents, stored locally
  on disk under `STORAGE_DIR` (backed up together with the SQLite DB).
- **Templates** — Reusable card templates with variable substitution.
- **Hierarchical Organization** — Parent-child card relationships with multiple
  view modes.
- **RSS Feed Client** — Subscribe to feeds, read articles, star for later, and
  convert interesting articles into cards.

### AI-powered intelligence (PRO features; unlocked by default when Stripe is disabled)
- **Vector Search** — Semantic similarity search via embeddings
  (OpenAI-compatible endpoints) in addition to full-text search (Typesense with
  a graceful fallback when Typesense is not configured).
- **Entity Recognition** — LLM-powered extraction and linking of people,
  places, organizations, and concepts.
- **Content Analysis** — Summaries, theme extraction, and insight generation
  with citation integrity.

### Local-first sync (in progress)
- **Desktop app** — a Tauri v2 shell (`desktop/`) wrapping the web app.
- **Sync engine** — a TypeScript sync engine (`sync-engine/`) providing
  offline-first sync: local SQLite mirror + outbox, with the server acting as a
  sync hub (snapshot/changes/push). See the
  [sync app design](docs/plans/2026-08-07-mobile-desktop-sync-app-design.md)
  for the live spec (epic Zettelgarden-v5b).

### Technical features
- **SQLite-only** — file-based, WAL mode; no external database server.
- **Web client** — React 18 + TypeScript + Vite, Tailwind CSS, PWA.
- **REST API** — Go backend with JWT auth; programmatic access.
- **Transactional email** — sent directly from the Go backend over SMTP
  (optional; no separate mail service).
- **Backup** — online SQLite snapshots via `VACUUM INTO` (see the
  [backup runbook](docs/runbooks/sqlite-backup.md)).
- **Data export & deletion** — self-serve from Settings; see the
  [data export runbook](docs/runbooks/data-export-and-account-deletion.md).
- **Keyboard shortcuts** — `c` create card, `s` search, `t` create task.

## Architecture

| Component | Path | Stack |
| --------- | ---- | ----- |
| Web frontend | `zettelkasten-front/` | React 18 + TypeScript, Vite, Tailwind, TanStack Query |
| API backend | `go-backend/` | Go (`net/http`), SQLite (WAL), JWT auth, Typesense (optional), local file storage |
| Desktop shell | `desktop/` | Tauri v2 (Rust) wrapping the web app |
| Sync engine | `sync-engine/` | TypeScript offline-first sync (local SQLite mirror + outbox) |
| CLI | `zg/` | Go CLI for card/task operations |
| MCP server | `zettelgarden-mcp/` | Model Context Protocol server for Claude |

### AI/ML stack
- Embeddings and chat via OpenAI-compatible endpoints (configurable base URL,
  key, model — works with local models too).
- Vector search via Typesense when configured; graceful full-text fallback
  otherwise.
- LLM-powered entity recognition and content analysis.

### Infrastructure
- Docker Compose manifests (`docker-compose.yml`) for self-hosting; a single
  data directory holds the SQLite DB and uploaded files.
- SQL migrations in `go-backend/schema/sqlite/` (consolidated schema;
  historical Postgres migrations archived under `go-backend/schema/archive/`).
- Optional: SMTP for transactional mail, Stripe for PRO billing,
  OIDC/SSO providers, Uptime Kuma push monitors.

## Getting Started (self-hosted)

### Quick start with Docker

```bash
cp .env.example .env        # review and edit
docker-compose up --build
```

The instance is then available on the configured port (see `.env.example`).
A fresh install auto-creates the SQLite database file under `./data/`.

### Manual development setup

Backend (Go):

```bash
cd go-backend
go run ./main.go            # REST API (reads root .env)
```

Frontend (React + Vite):

```bash
cd zettelkasten-front
npm install
npm run start               # Vite dev server on http://localhost:5173
```

Tests: `go test ./...` (backend) and `npm run test:coverage` (frontend).

### Configuration

All configuration comes from the root `.env` (template: `.env.example`).
Highlights:

- `SQLITE_PATH` — SQLite database file (default `./data/zettelgarden.db`).
- `STORAGE_DIR` — local file storage directory (default `./data/files`).
- `ZETTEL_LLM_KEY` / `ZETTEL_LLM_ENDPOINT` — optional AI integration.
- `TYPESENSE_*` — optional full-text/vector search; the app falls back to
  built-in search when Typesense is not configured.
- `SMTP_*` — optional transactional mail (mail is a no-op when unset).
- `STRIPE_ENABLED=false` (default) — disables billing; PRO features are
  unlocked. Set to `true` with Stripe keys to enable subscription billing.
- `OIDC_*` — optional OIDC/SSO (e.g. Pocket ID); see the
  [OIDC design](docs/plans/2026-08-03-oidc-authentication-design.md).

### Desktop app

```bash
cd desktop
npm install
npm run dev                # tauri dev; requires Rust/Tauri toolchain
```

See `desktop/README.md` for details.

## Documentation

- `docs/plans/` — design docs (dated; historical ones carry a
  `HISTORICAL`/`SUPERSEDED`/`EXECUTED` header).
- `docs/runbooks/` — operational runbooks (SQLite backup, data export).
- `docs/archive/` — archived one-off reports and marketing-era docs.
- `AGENTS.md` / `CLAUDE.md` — development guides and commands.

## License

MIT — see [LICENSE](LICENSE).
