# Repository Guidelines

## Project Structure & Module Organization
- `go-backend/`: Go API with `handlers/`, `services/`, and SQL `migrations/`; environment config comes from the root `.env` and expects PostgreSQL, Typesense, and AI provider keys.
- `zettelkasten-front/`: React 18 + TypeScript client; core UI lives in `src/components/`, state in `src/contexts/`, and shared helpers in `src/utils/` with colocated `*.test.ts(x)` specs.
- `python-mail/`: Minimal Flask mailer for transactional email; keep requirements in sync with `requirements.txt`.
- Supporting assets include `docs/` for design notes, `tickets/` for planning, and Docker manifests (`docker-compose.yml`, `docker-zettel-run.yml`, `build.sh`) for local orchestration.

## Build, Test, and Development Commands
- Frontend: `npm install` then `npm run start` (Vite dev server on http://localhost:5173); `npm run build` emits production assets in `dist/`.
- Backend: `go run ./main.go` boots the REST API; `go test ./...` exercises the full Go test suite.
- Frontend tests: `npm run test` (watch), `npm run test:coverage` for CI-style runs.
- Docker workflow: export the root `.env`, then `./build.sh` or `docker-compose up --build` to recreate images and services.

## Coding Style & Naming Conventions
- TypeScript: rely on Prettier defaults (2-space indentation, single quotes) and Tailwind utility classes; components stay `PascalCase`, hooks use the `useX` prefix.
- Go: run `go fmt ./...` before committing; packages stay lowercase, request handlers use verb-based names (`HandleCreateCard`).
- Python mailer: stick to Black-compatible formatting (4-space indentation) and descriptive function names.

## Testing Guidelines
- Frontend unit tests live alongside source files as `*.test.ts` or `*.test.tsx` and use Vitest with Testing Library; prefer rendering components over shallow mocks.
- Backend tests follow Go’s `_test.go` pattern under `handlers/` and `services/`; keep fixtures in `go-backend/tests/` and reset database state per test case.
- Add integration tests when touching API contracts so both client and server stay in sync.

## Commit & Pull Request Guidelines
- Follow the existing history: short, imperative subjects (`Add card_id tooltip`, `Move use template`) and one change per commit.
- Before opening a PR, run relevant test commands, note any schema changes, and update docs when behavior shifts.
- PR descriptions should cover context, screenshots for UI changes, and links to issues or tickets; call out any migrations or feature flags that reviewers must enable.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
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

<!-- BEGIN BEADS INTEGRATION -->
## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
bd create "Issue title" --description="What this issue is about" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" --description="Details about what was found" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`

### Auto-Sync

bd automatically syncs with git:

- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

For more details, see README.md and docs/QUICKSTART.md.

<!-- END BEADS INTEGRATION -->
