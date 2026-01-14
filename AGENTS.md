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
