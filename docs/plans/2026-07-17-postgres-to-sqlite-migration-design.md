# PostgreSQL → SQLite Migration Plan

**Date:** 2026-07-17
**Status:** Revised — Ready for Implementation
**Author:** Nick + Pi

## Revision history

- **v1 (2026-07-17):** Initial draft.
- **v2:** Re-assessed against the current codebase. The job-queue
  locking that was the principal SQLite blocker has been **removed** (the queue
  is now inline, see "Current-State Inventory"). Effort estimates and phase
  list corrected accordingly. ETL + cutover runbook expanded because there is
  real user data (~15k cards) that cannot be lost.
- **v3:** Accuracy pass — every PG-ism count was recounted
  against the codebase. Placeholder count corrected (649 → **1373** occurrences;
  file count unchanged at 74); `NOW()` revised 38 → **43 non-test**; `INTERVAL`
  revised 7 → **11 non-test**. Found **3** GIN indexes in schema, not 1 (FTS +
  JSONB + array) — Phase 2 now enumerates all three. Added missing Phase 2 item:
  **98** column-level `DEFAULT NOW()` → `DEFAULT (datetime('now'))`. Added
  `PRAGMA synchronous=NORMAL` to D4, a bulk-import timing check to the Phase 1
  spike, and switched the ETL idempotency recommendation away from
  `INSERT OR REPLACE` (cascade footgun under `foreign_keys=ON`). Corrected
  "EXCLUDE" (no such constraint exists in the schema).
- **v4 (this revision):** Second accuracy pass with grep verification against
  the live tree. Corrected overcounts: `NOW()` is actually **31 files / 103
  occurrences** non-test (not 43); `INTERVAL` is **6 files** non-test (not 11);
  schema files total **144**, not 139. The "98 `DEFAULT NOW()`" figure is really
  **58 `NOW()` + 39 `CURRENT_TIMESTAMP` = 97** — both forms must be translated.
  Confirmed via grep that **no Go code uses any PG JSONB/array operator** (`@>`,
  `&&`, `?|`) **or any FTS function** (`to_tsvector`, `@@`, `tsquery`) — so all
  three GIN indexes are **dead** (no query reads them) and can be dropped
  unconditionally. **Biggest addition:** SQLite's native parameter syntax
  supports `$1`/`$NNN`; `modernc.org/sqlite` may accept numbered params as-is,
  which would make the 1373-edit placeholder sweep unnecessary — this is now a
  Phase 1 investigation. Also added: the per-connection PRAGMA footgun (D4), the
  Phase 4 → Phase 5 correctness dependency, the 3 `sql.Open("postgres")` call
  sites, and WAL-safe backup wording (`VACUUM INTO` primary).

## Overview

Migrate the Go backend's primary data store from PostgreSQL (+ pgvector) to
SQLite. The goal is operational simplicity: we are moving away from a SaaS
deployment model toward something that is easy to self-host and operate.
PostgreSQL (and its pgvector container) adds meaningful overhead — a separate
server process, a connection-pool to tune, credentials to manage, and a
container to run — for an application that is fundamentally single-binary,
low-write, and single-user.

Replacing Postgres with SQLite (WAL mode) collapses the deployment to a single
static Go binary plus a database file on disk, and removes the `DB_*` env config
entirely.

## Goal

- One static `go build` binary with **no external database process** and **no
  CGO** toolchain requirement.
- Zero behavior changes for the end user.
- Existing test suite passes against SQLite.
- A documented, **verified** one-time path to import existing Postgres data
  (~15k cards + full entity/fact/flashcard/task graph) with zero loss.

## Non-Goals

- Dropping **Typesense** — it remains the search/embedding index. (pgvector is
  declared in `docker-zettel-run.yml` as `pgvector/pgvector:pg16` but the
  **extension is not actually used** — confirmed: no `vector()` columns exist
  anywhere in `schema/`, and every "vector" hit in Go code is a comment about
  Typesense. Note: there *are* three core-Postgres GIN indexes in the schema
  (one FTS `to_tsvector` + one JSONB containment + one array containment) —
  these do **not** require the pgvector extension and are handled in Phase 2,
  not here. See Phase 0.)
- Supporting high-throughput multi-tenant SaaS write patterns. SQLite's
  single-writer model is acceptable for the self-hosted, single-user target.
- Translating all 144 historical migrations. We will **consolidate** instead
  (see Decision D2).

## Current-State Inventory (re-verified 2026-07-17)

| Area | Finding (corrected) | Migration impact |
|---|---|---|
| DB driver | `github.com/lib/pq` imported in 11 files; `sql.Open("postgres", …)` hardcoded in **3 sites**: `server/database.go`, `scripts/addParentId.go`, `scripts/computeKeywords.go` | Swap to `modernc.org/sqlite` (after cutover — see D6); **all three** call sites must move behind the feature flag |
| Query placeholders | **1373** `$1,$2,…` occurrences across **74 files** (file count exact) | Mechanical: `$N` → `?` |
| `RETURNING` | 79 clauses, 31 files | ✅ SQLite 3.35+ supports as-is |
| `ON CONFLICT` upserts | 14 files | ✅ SQLite supports identical syntax |
| `JSONB` columns | 11 schema files, 9 Go files (stored as `[]byte`) | ✅ Store as `TEXT` + JSON1 |
| `NOW()` | **31 files / 103 occurrences** non-test (was est. 43) | → app-side `time.Now().UTC()` or `datetime('now')` |
| `INTERVAL '…'` | **6 files** non-test (was est. 11) | → `datetime('now','+1 day')` |
| `EXTRACT(EPOCH FROM …)` | **0 in Go** (was est. 42 — that count was schema-only) | None in Go; schema handled by D2 |
| `ILIKE` | 5 files | → `LIKE` (case-insensitive for ASCII) |
| `::` casts | 5 lines, 2 files | remove or `CAST()` |
| **Arrays `TEXT[]`** | **0 `ARRAY[` in Go**; 2 columns at schema layer: `notifications.filter_tags`, `chat_messages.referenced_cards` (JSONB) | Schema-layer only → JSON-in-TEXT (D2) |
| **Triggers / PL/pgSQL** | **7 schema files**: `0093` (heaviest, user-stats), `0067`, `0096`, `0102`, `0122`, `0123` (email notify), `0124` (rss notify) | Rewrite logic in Go |
| `COMMENT ON …` | scattered | SQLite errors — strip |
| GIN / tsvector | **3 GIN indexes** (none need pgvector): FTS `to_tsvector` on `files.extracted_text`, JSONB containment on `chat_messages.referenced_cards`, array containment on `notifications.filter_tags`. No `EXCLUDE` constraint exists. | Drop all three. **Verified dead** — grep confirms zero Go usage of `to_tsvector`/`@@`/`tsquery` and zero usage of JSONB/array operators (`@>`, `&&`, `?|`). No query reads any of these indexes, so no replacement (FTS5, JSON expression index, app-side filter) is needed. |
| **Job-queue locking** | **`SKIP LOCKED` / `FOR UPDATE`: 0 occurrences.** `services/jobqueue.go` deleted; replaced by inline `services/jobrunner.go` (no dequeue, no worker pool, no heartbeats, no retry). | ✅ **Already resolved.** Was the principal blocker; now gone. |
| Test infra | `TRUNCATE … RESTART IDENTITY CASCADE`, `pg_get_serial_sequence`, `setval('entities_id_seq', …)` in `tests/conftest.go` (898 lines) | Rewrite test reset path |
| Migration runner | `server/database.go:171` runs each `.sql` via single `tx.Exec(string(content))` — **relies on lib/pq multi-statement parsing** | Adapt: quote-aware `;` splitter (see D2, Phase 1) |
| Standalone cmd binaries + scripts | `cmd/{reminders,userMemoryMaintenance,deduplication}` plus `scripts/{addParentId,computeKeywords}.go` — separate processes that open the DB independently | Optionally fold into in-process scheduler (Phase 4); low priority for single-user. **Caveat:** once trigger logic moves to Go (Phase 5), any non-server writer bypasses it unless the logic lives in a shared package these binaries call. |

## Decision Log

### D1 — Driver: `modernc.org/sqlite` (pure Go, no CGO)
**Chosen over** `mattn/go-sqlite3` (CGO, faster). Rationale: the whole point is
operational simplicity; a pure-Go driver keeps `go build` producing a single
static binary with no C toolchain. modernc is mature and supports WAL,
`RETURNING`, JSON1, and FTS5.

### D2 — Do NOT port 139 migrations; consolidate the schema
Translating 144 incremental migrations to SQLite is wasted effort and risk.
Instead:
1. Produce one consolidated `go-backend/schema/sqlite/schema.sqlite.sql`
   representing the **current final state** (derived from `pg_dump --schema-only`
   of the live dev DB, then adapted to SQLite syntax).
2. New installs build the DB from that single file.
3. Existing-Postgres data is imported via a one-time ETL that reads **live PG
   tables** (not migration history) → SQLite (see Phase 6).

The migration runner stays, but learns to (a) split multi-statement files on `;`
**respecting quotes and string literals** (this is the critical adaptation —
`modernc.org/sqlite` does not parse multiple statements in a single `Exec`,
unlike lib/pq), and (b) treat the SQLite schema as the source of truth. The old
`schema/*.sql` (PG) files are preserved for the data-import ETL only.

### D3 — ~~Job queue: atomic `UPDATE … RETURNING` claim~~ — RESOLVED, NO LONGER NEEDED
The original plan replaced `SELECT … FOR UPDATE SKIP LOCKED` with an atomic
`UPDATE … RETURNING … WHERE id = (SELECT … LIMIT 1)` claim. **This is no longer
required:** the job queue was refactored to an inline runner
(`services/jobrunner.go`) with no dequeue step, no worker pool, and no
heartbeats. Jobs are recorded in `llm_jobs` as an audit table and executed
immediately in a recovered goroutine. SQLite's single-writer model handles this
trivially. (This decision is retained for historical context.)

### D4 — Concurrency mode: WAL + `busy_timeout` + `synchronous=NORMAL`
Set on every connection: `PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;
PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;`. WAL allows concurrent
readers alongside a single writer; `busy_timeout` makes contending writers wait
instead of erroring; `synchronous=NORMAL` trades the (power-loss-only) safety of
`FULL` for much better write throughput — the recommended setting for
single-user WAL deployments, and safe against corruption (only the last
committed transaction is at risk on power loss). For a single-user deployment
this combination is more than sufficient; `busy_timeout` will realistically
never contend.

**Implementation note (footgun):** `PRAGMA foreign_keys=ON` (and the others)
are **connection-scoped, not database-scoped** in SQLite. `sql.Open` returns a
pooled `*sql.DB`, and a bare `db.Exec("PRAGMA …")` sets the pragma on only one
pooled connection — FK enforcement will then be **silently inconsistent** across
the pool. Set the pragmas inside a `database/sql/driver.Connector` wrapper
registered via `sql.OpenDB(connector)` so they run on every new connection.
(`journal_mode=WAL` is persistent / database-level; the rest must be
per-connection.)

### D5 — Time values stored as ISO-8601 TEXT
SQLite has no native timestamp type. Store as TEXT (ISO-8601, UTC). Both Go
drivers scan these into `time.Time` transparently. `timestamptz` semantics are
preserved because the app normalizes to UTC in most places (verify in Phase 3).

### D6 — `lib/pq` is retained through the cutover window
The ETL tool (`cmd/migrate-pg-to-sqlite/`) reads from Postgres, so `lib/pq`
cannot be removed from `go.mod` until the cutover is confirmed in production.
Therefore Phase 7 is split:
- **7a (at cutover):** remove Postgres/pgvector from deploy config (docker,
  env).
- **7b (~2 weeks after cutover):** delete the ETL tool and drop `lib/pq` from
  `go.mod`, once SQLite-in-prod is proven.

## Phased Work Breakdown

Each phase is independently mergeable. Acceptance criteria are explicit.

### Phase 0 — pgvector image swap (quick win, ~30 min)
**Independent of everything else. De-risks the "is pgvector actually unused?"
question in production before the migration.**

- [ ] In `docker-zettel-run.yml`, replace `image: pgvector/pgvector:pg16` with
      `image: postgres:16-alpine`.
- [ ] In `.github/workflows/go.yml`, replace the CI test-database `image:
      pgvector/pgvector:pg16` with `image: postgres:16-alpine` (same unused
      dependency; the test schema has no `vector()` columns).
- [ ] Run the app against the plain-postgres container in staging/dev; confirm
      no errors (there shouldn't be — no `vector()` columns exist).

**Acceptance:** App runs unchanged against plain Postgres (no pgvector), and CI
runs green against the plain-postgres service image. This removes the pgvector
dependency immediately and can ship before the SQLite work begins.

### Phase 1 — Spike / de-risk (1–2 days)
**Prove the approach end-to-end on one table before the big sweep.**

- [ ] Add `modernc.org/sqlite` dependency.
- [ ] Implement a thin SQLite connection helper (`server/sqlite.go`) returning a
      `*sql.DB` configured with D4 pragmas, behind a feature flag (`DB_DRIVER=sqlite`,
      `SQLITE_PATH=…`).
- [x] **Investigate native `$N` parameter support FIRST — RESOLVED: YES.**
      Spike (`go-backend/spike/sqlite_param_test.go`, `modernc.org/sqlite`
      v1.54.0) confirms the driver binds `$1`-style numbered params to
      positional `[]any` args through the standard `database/sql` interface
      (`Query`/`QueryRow`/`Exec`). Also accepts `?`, `?NNN`, and named `:id` /
      `@id` / `$id` (via `sql.Named`). **Consequence: the entire 1373-edit
      `$N → ?` placeholder sweep (Phase 3) is unnecessary** — existing queries
      run unchanged. The single largest mechanical phase is eliminated.
- [x] Fallback rewriter: **not needed** (driver accepts `$1`). No runtime
      rewriter and no source edits required.
- [ ] **Build a quote-aware `;` statement splitter** for the migration runner
      (Phase 1 priority — see D2). Must handle `'…'` string literals, `''`
      escapes, and `--` comments. This is the #1 spike deliverable; without it
      the consolidated schema cannot load.
- [ ] Wire the `cards` (or `users`) read+write path through SQLite using a
      consolidated mini-schema for that one table.
- [ ] One handler test runs green against an in-memory (`:memory:`) SQLite DB.
- [ ] Exercise a realistic concurrent-write mix under WAL + `busy_timeout`:
      HTTP handler + a scheduler tick + an inline JobRunner goroutine hitting
      the same DB concurrently. Confirm no `database is locked` errors.
- [ ] **Bulk-import timing check:** load a ~1k-card slice (with its
      entity/fact/tag sub-graph) through the ETL path and measure wall-clock.
      `modernc.org/sqlite` is materially slower than the CGO driver on bulk
      inserts; confirm the projected 15k-card import is tolerable *before*
      committing to modernc for the Phase 6b import path. (Worst case: use CGO
      `mattn/go-sqlite3` for the one-shot ETL tool only — it's deleted in 7b
      anyway, so it doesn't touch the runtime CGO-free goal.)

**Acceptance:** A single authenticated request (e.g. `GET /api/cards/:id`)
works against SQLite with no Postgres running; the statement splitter loads a
multi-statement schema file cleanly into `:memory:`. This validates driver,
pragmas, placeholder rewriting, `RETURNING`, **and the migration runner
adaptation** before committing to the full sweep.

### Phase 2 — Consolidated SQLite schema (2–3 days)
- [ ] `pg_dump --schema-only` the current dev DB.
- [ ] Translate to `schema/sqlite/schema.sqlite.sql`:
  - `SERIAL`/`BIGSERIAL` → `INTEGER PRIMARY KEY AUTOINCREMENT`
  - `TIMESTAMP[TZ]`/`VARCHAR`/`TEXT` → `TEXT` (timestamps) / `TEXT`/`BLOB`
  - `JSONB` → `TEXT`; the 2 array columns (`filter_tags`, `referenced_cards`)
    → `TEXT` (JSON)
  - **Timestamp column defaults → `DEFAULT (datetime('now'))`** — **97
    occurrences** in schema column definitions: **58** are `DEFAULT NOW()` and
    **39** are `DEFAULT CURRENT_TIMESTAMP`. **Translate both forms** — easy to
    miss the `CURRENT_TIMESTAMP` variant in a hand-translation, and silently
    breaks insert defaults if skipped.
  - drop `COMMENT ON` (SQLite errors on it)
  - drop all **3 GIN indexes** (SQLite has no GIN):
    `idx_files_extracted_text` (FTS `to_tsvector`),
    `idx_chat_messages_referenced_cards` (JSONB containment),
    `idx_notifications_filter_tags` (array containment). **No replacement
    needed** — verified that no Go code reads any of them (zero usage of
    `to_tsvector`/`@@`/`tsquery` and zero usage of `@>`/`&&`/`?|`). Typesense
    remains the file-text search path.
  - no `EXCLUDE` constraint exists in the schema — nothing to do there.
  - add `CREATE INDEX` equivalents for ordinary lookups (JSON expression
    indexes where needed).
- [ ] Validate the schema loads cleanly into a fresh `:memory:` DB **via the
      Phase 1 statement splitter** (not hand-fed statements).

**Acceptance:** A brand-new SQLite file can be created from the consolidated
schema with zero errors, and `PRAGMA foreign_key_check` passes.

### Phase 3 — Query translation sweep (2–3 days)
The mechanical bulk. Smaller than the original estimate (EXTRACT/ARRAY are
schema-only; INTERVAL/NOW counts revised down). **Phase 1 confirmed modernc
accepts `$1` natively, so the placeholder sweep is OFF the table** — this phase
is just the `NOW()`/`INTERVAL`/`ILIKE`/cast/operator fixes.

- [x] ~~`$N` → `?` across all 74 files~~ — **NOT NEEDED.** Phase 1 proved
      modernc binds `$1` to positional args. Existing queries run unchanged.
- [ ] `NOW()` → `?` bound to `time.Now().UTC()` (prefer app-side time; enables
      test determinism) — **31 files / 103 occurrences** (non-test).
- [ ] `INTERVAL '…'` → `datetime(?, '+N unit')` — **6 files** (non-test).
- [ ] `ILIKE` → `LIKE` — verify case behavior for non-ASCII (collation) — 5 files.
- [ ] Remove `::` casts — 2 files.
- [ ] Strip `COMMENT ON` references (none in Go; schema already handled in P2).
- [ ] Audit timestamp/UTC handling per D5.

**Acceptance:** Backend compiles; `go vet ./...` clean; unit tests that don't
touch DB-specific SQL pass.

### Phase 4 — Consolidate standalone cmd binaries into the scheduler (1 day, optional)
**Replaces the deleted "job queue redesign" phase.** Low priority for a
single-user deployment, but architecturally cleaner: one process = simplest
SQLite story and removes a class of cross-process write concerns entirely.

**Note (Phase 5 dependency):** once trigger logic moves into Go (Phase 5), any
non-server writer (`cmd/*` binaries, `scripts/*`) bypasses that logic unless it
lives in a shared package they import. So this phase is effectively a
**correctness prerequisite** for Phase 5, not merely optional cleanup. If you
defer Phase 4, the trigger logic must be factored into a shared `services/`
package that all writers call — do not inline it in the HTTP layer.

- [ ] Move `cmd/reminders`, `cmd/userMemoryMaintenance`, `cmd/deduplication`
      logic into ticks of the in-process `services/scheduler.go`.
- [ ] Remove the standalone binaries (or keep as thin wrappers that call into
      the scheduler package).

**Acceptance:** A single binary handles all background work; the standalone cmd
binaries are gone or trivial. (Can be deferred post-cutover if it proves
fiddly — `busy_timeout` already handles cross-process writes safely at this
scale.)

### Phase 5 — Trigger logic → Go (2–3 days)
- [ ] Port the email-notification trigger (`0123`) into Go (notification helper
      already exists via `0122`).
- [ ] Port the RSS-notification trigger (`0124`) into Go.
- [ ] Port user-stats triggers (`0093`) into Go (or a scheduled job — there's
      already a scheduler).
- [ ] Port `0096`/`0102` (llm_jobs `updated_at`/notify) — app-side.
- [ ] `0067` — review individually; likely trivial.

**Acceptance:** Notification + user-stats behavior unchanged; integration tests
for notifications/stats pass against SQLite.

### Phase 6 — Test infra + ETL + cutover (3–4 days) — **HIGHEST-STAKES PHASE**
This phase owns data continuity. Everything else is reversible; this is not.
Build and validate against a **copy** of Postgres, never the live DB.

#### 6a — Test infra
- [ ] Rewrite `tests/conftest.go`:
  - DB points at a fresh `:memory:` (or temp file) SQLite built from the
    consolidated schema.
  - **Lean on the transaction-per-test rollback that already exists**
    (`conftest.go` already wraps each test in `S.Tx` + `Rollback()` at
    lines ~233/251). This is dramatically cleaner than the PG path and needs
    ~no PG-ism translation: a rolled-back transaction never commits, so there
    is nothing to reset.
  - Only where rollback isn't feasible (suite-level shared setup), replace
    `TRUNCATE … RESTART IDENTITY CASCADE` + `setval(...)` with
    `DELETE FROM <tables>` (in FK order) + `DELETE FROM sqlite_sequence`.
  - Fix `setval('entities_id_seq', …)` and any other `*_seq` references
    (these become no-ops or `sqlite_sequence` resets).
- [ ] Run the **full** Go test suite green against SQLite.

#### 6b — ETL tool (`cmd/migrate-pg-to-sqlite/`)
- [ ] Reads live PG tables (via `lib/pq`) → writes SQLite (via modernc), in
      FK-dependency order, **preserving PKs verbatim** so the graph
      (entity_card_junction, backlinks, task_dependencies, …) stays intact.
- [ ] **Idempotent / re-runnable** — prefer `INSERT … ON CONFLICT (pk) DO
      NOTHING` (or a per-table wipe+reload) over `INSERT OR REPLACE`. Under
      `foreign_keys=ON`, `INSERT OR REPLACE` on a conflicting PK issues a
      DELETE+INSERT, which fires cascades across the 50+-table graph; for a
      fresh DB either approach works, but `ON CONFLICT DO NOTHING` is safer and
      faster. Alternative: disable FKs during the bulk load, re-enable, then
      run `PRAGMA foreign_key_check`. Will be run dozens of times during dev
      against the copy.
- [ ] Table classes:
  - **Domain (migrate verbatim):** `cards`, `card_tags`, `card_views`,
    `card_templates`, `pinned_cards`, `inactive_cards`, `keywords`, `backlinks`,
    `entities`, `entity_card_junction`, `entity_fact_junction`, `facts`,
    `fact_card_junction`, `email_fact_junction`, `tags`, `tasks`, `task_tags`,
    `task_dependencies`, `task_statuses`, `task_saved_searches`, `files`,
    `files_tags`, `file_tags`, `spreadsheets`, `flashcards` (FSRS),
    `flashcard_reviews`, `summary_theses`, `summary_sections`, `summary_arguments`,
    `summarizations`, `chat_conversations`, `chat_messages`, `chat_instructions`,
    `chat_tool_calls`, `chat_usage_quotas`, `notifications`,
    `notification_preferences`, `user_memories`, `user_stats`, `users`,
    `api_keys`, `agents`, `agent_activity_log`, `rss_feeds`, `rss_folders`,
    `rss_articles`, `rss_seen_articles`, `email_accounts`, `emails`,
    `email_attachments`, `email_card_links`, `email_triage_decisions`,
    `external_calendars`, `external_events`, `admin_audit_log`, `audit_events`,
    `llm_jobs`, `llm_query_log`, `llm_models`, `llm_providers`,
    `user_llm_configurations`, `scheduled_job_runs`, `revenue`, `stripe_plans`,
    `mailing_list`, `mailing_list_messages`, `mailing_list_recipients`,
    `habits`, `habit_logs`.
  - **System (rebuild, do NOT migrate):** `migrations`, `schema_definitions`
    → rebuilt by the consolidated schema + Phase-1 migration runner.

#### 6c — Verification gates (the actual "is it safe" checks)
These must all be green before cutover. Implement as part of the ETL tool's
output or a companion `verify` command:

- [ ] **Per-table row counts:** `SELECT COUNT(*)` PG vs SQLite for every
      domain table — must match exactly.
- [ ] **PK stats:** min / max / sum of primary key per table — catches
      off-by-one, truncation, or duplicate inserts.
- [ ] **Graph-integrity spot check:** pick ~20 cards spread across time,
      fetch each via the API running on SQLite, and diff the response against
      the same fetch against Postgres. Entity / fact / tag / flashcard /
      backlink sub-graphs must match.
- [ ] **Frontend smoke:** dashboard counts, one card detail, one task list,
      one chat history render without diffs against the PG-backed instance.

**Acceptance:** `go test ./...` green with no Postgres running. ETL successfully
migrates a copy of real dev data and **all verification gates pass**.

### Phase 7 — Cleanup & rollout (split per D6)

#### 7a — At cutover (1 day)
- [ ] Remove `DB_HOST/PORT/USER/PASS/NAME` config; replace with `SQLITE_PATH`
      (default `./data/zettelgarden.db`).
- [ ] Move **all three** `sql.Open("postgres", …)` call sites behind the driver
      abstraction: `server/database.go`, `scripts/addParentId.go`,
      `scripts/computeKeywords.go` (the `scripts/` pair is easy to forget).
- [ ] Remove postgres from `docker-compose.yml` / `docker-zettel-run.yml`
      (pgvector already swapped in Phase 0; now drop the `db` service entirely).
- [ ] Update `.env.example`, `README.md` (self-hosting section), `AGENTS.md`.
- [ ] Archive `schema/*.sql` (PG) — keep the ETL-readable copy until 7b.
- [ ] Document backup = **`VACUUM INTO 'snapshot.db'`** as the ongoing model
      (not raw `cp`, which is unsafe under WAL without a checkpoint).

#### 7b — ~2 weeks after confirmed cutover (½ day)
- [ ] Delete `cmd/migrate-pg-to-sqlite/`.
- [ ] Drop `github.com/lib/pq` from `go.mod` and remove the now-dead `scripts/`
      postgres wiring (addParentId, computeKeywords) if not already folded into
      the scheduler in Phase 4.
- [ ] Delete or archive `schema/*.sql` (PG).
- [ ] Decommission the Postgres container/image for good.

**Acceptance (7a):** Fresh clone → `go build` → `go run` → working app, with no
database to install. **Acceptance (7b):** no Postgres references remain in the
repo; `go.mod` is `lib/pq`-free.

## Cutover Runbook (the actual "can't be lost" protection)

The ETL being correct is necessary but not sufficient. What actually protects
the data is **never destroying the source until the new path is proven in
production.** Nick is the only user, so a short maintenance window is fine.

1. **Pre-cutover (days before):** all development and validation of the SQLite
   path runs against a **copy** of Postgres (`pg_dump`/restore to a throwaway
   DB). Never iterate the ETL against the live DB.
2. **Cutover day:**
   1. Stop the app (or set it read-only).
   2. **Take a final `pg_dump` of Postgres to a file and store it safely.**
      This is the ultimate insurance — even if everything else goes wrong, this
      file *is* the data. Keep it for ≥90 days.
   3. Run the ETL one final time against the now-static Postgres.
   4. **Run all verification gates (Phase 6c).** Do not proceed until green.
   5. Flip `DB_DRIVER=sqlite`, `SQLITE_PATH=...`, restart.
   6. Smoke-test in prod (dashboard, a card, a task list, a chat).
3. **Post-cutover:** **Keep Postgres running, read-only and untouched, for
   ≥2 weeks** as fallback. Only execute Phase 7b (drop lib/pq, decommission
   Postgres) once SQLite-in-prod has been used confidently.

**Two distinct backup concepts — don't conflate them:**
- **`pg_dump` file at cutover** = one-time insurance against a botched
  migration. Keep ≥90 days.
- **SQLite ongoing backup** = **`VACUUM INTO 'snapshot.db'` is the primary
  recommendation** — it produces a consistent single-file snapshot regardless
  of WAL state. Raw `cp` of the `.db` file is only safe after
  `PRAGMA wal_checkpoint(TRUNCATE)`, because in WAL mode the live data is split
  across `*.db` + `*.db-wal` (+ `.db-shm`); copying just the `.db` can miss
  uncommitted-to-main commits.

## Risk Register (updated)

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| **Migration runner multi-statement parsing** (`tx.Exec(string)` relies on lib/pq; modernc does not split on `;`) | **High** | High | Quote-aware statement splitter is the **#1 Phase 1 deliverable**; spike validates it before any sweep |
| **Data loss / corruption during ETL** (15k cards + graph) | Low | **Critical** | Verification gates (Phase 6c); `pg_dump` insurance at cutover; Postgres kept read-only ≥2 weeks as fallback; ETL developed against a copy only |
| Hidden PG-isms in queries not caught by grep (implicit casts, `COALESCE` on timestamps) | Med | Med | Full test suite is the safety net; Phase 1 spike surfaces classes early |
| Non-ASCII `ILIKE` case-folding differs | Low | Low | Use `COLLATE NOCASE` or `LOWER(x) LIKE LOWER(?)`; verify with tests |
| Cross-process write contention (standalone cmd binaries vs. main server) | **Low** (single user) | Low | WAL + `busy_timeout=5000`; optionally fold binaries into scheduler (Phase 4) |
| `BEGIN`/DDL interaction in migration runner | Low | Med | Statement splitter respects SQLite rules; consolidated schema is mostly DDL run outside tx |
| Timestamp precision/timezone regressions | Med | Med | Store UTC ISO-8601 everywhere; audit in Phase 3 |
| modernc driver edge cases (e.g. large `[]byte` JSON) | Low | Low | Spike in Phase 1; well-trodden driver |

Note: the original "write contention under concurrent workers (heartbeats)"
risk is **removed** — heartbeats no longer exist with the inline JobRunner.

## Open Questions

Most questions from v1 are now resolved by the codebase state or by Nick's
input. Remaining:

1. ~~Data continuity?~~ **Answered:** ~15k cards of real data, single user,
   cannot be lost. Phase 6 + Cutover Runbook designed around this.
2. **Audit-table pruning:** migrate `llm_jobs` / `llm_query_log` / `audit_events`
   wholesale (cheap, keeps history — current default), or prune to last 90 days
   to slim the DB? *Recommend: migrate wholesale.*
3. **Typesense:** keep as-is (recommended). Out of scope here; it remains the
   other external service and is the real search/embedding index.
4. **Backups:** confirm "copy the `.db` file" (or `VACUUM INTO`) is acceptable
   as the ongoing model, alongside the one-time `pg_dump` cutover insurance.

## Testing Strategy

- **Unit/integration:** existing `_test.go` suite against `:memory:` SQLite. This
  is the primary gate — the suite is large (handler + service tests) and covers
  the API contract.
- **New:** SQLite schema-load test (Phase 2); statement-splitter unit tests
  (Phase 1); ETL verification gates (Phase 6c).
- **Manual smoke:** run the full app against SQLite for a session before
  declaring done; diff API responses against the PG-backed instance for sampled
  cards.

## Effort Summary (revised)

| Phase | Effort | Change vs v1 |
|---|---|---|
| 0 — pgvector swap | ~30 min | New quick win |
| 1 — Spike | 1–2 days | + statement splitter as headline deliverable |
| 2 — Consolidated schema | 2–3 days | — |
| 3 — Query translation | ~1 day | **Confirmed small:** Phase 1 proved modernc accepts `$1` natively, so the 1373-placeholder sweep is gone. Remaining work is just `NOW()`/`INTERVAL`/`ILIKE`/cast/operator fixes. |
| 4 — Consolidate cmd binaries | 1 day (optional) | **Replaces** deleted job-queue phase; can defer |
| 5 — Triggers → Go | 2–3 days | — |
| 6 — Tests + ETL + cutover | 3–4 days | **Up** from 2–3; verification gates + runbook added |
| 7a/7b — Cleanup & rollout | 1–2 days | Split per D6 |
| **Total** | **~8–13 focused days (~2 weeks)** | **Down from ~12–19** |

The principal blocker (job-queue locking) is already removed. The principal
remaining risk (multi-statement migration runner parsing) is a Phase 1 spike
deliverable. The principal remaining stake (15k cards of data) is protected by
the Cutover Runbook and verification gates. Phase 0 is the cheap first step.
