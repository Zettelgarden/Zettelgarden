# PostgreSQL → SQLite Migration — Status

**Last updated:** 2026-07-17
**Plan:** [`2026-07-17-postgres-to-sqlite-migration-design.md`](./2026-07-17-postgres-to-sqlite-migration-design.md)
**Tracking:** epic `Zettelgarden-c7j` · Phase 0 `Zettelgarden-bw1` (closed) · Phase 1 `Zettelgarden-2u2` (in progress)

## TL;DR

Implementation is underway. **Phase 0 done; Phase 1 ≈80% done.** Every
high-risk *unknown* has been resolved empirically, and the findings **shrank
the plan**: the 1373-edit placeholder sweep is gone, and a latent timestamp
landmine was caught and fixed (schema-side, no code changes). Remaining Phase 1
work is execution (cards wiring + bulk-import timing), not investigation.

## Phase status

| Phase | Status | Notes |
|---|---|---|
| 0 — pgvector image swap | ✅ **Done** | `docker-zettel-run.yml` + `.github/workflows/go.yml` → `postgres:16-alpine` |
| 1 — Spike / de-risk | 🟡 **~80%** | Driver, splitter, conn helper, migration runner, concurrency, **and** compatibility probes all done & tested. Cards wiring + bulk-import timing remain. |
| 2 — Consolidated SQLite schema | ⬜ Not started | Needs a live `pg_dump --schema-only` of the dev DB. **Must declare timestamp columns `DATETIME` (see D5).** |
| 3 — Query translation | ⬜ Not started | **Shrunk to ~1 day** — placeholder sweep eliminated; remaining: `NOW()`/`INTERVAL`/`ILIKE`/casts only |
| 4 — Consolidate cmd binaries | ⬜ Not started | Optional; correctness prerequisite for Phase 5 trigger porting |
| 5 — Triggers → Go | ⬜ Not started | 7 trigger files; logic must reach all write paths |
| 6 — Tests + ETL + cutover | ⬜ Not started | Highest-stakes phase; data-continuity protections in runbook |
| 7a/7b — Cleanup & rollout | ⬜ Not started | Split per D6 |

## Key findings (resolved this session)

These changed the shape of the plan. All verified by passing tests in
`go-backend/server/` and `go-backend/spike/`.

1. **`modernc.org/sqlite` accepts `$1`-style numbered params natively** (via
   standard positional `[]any` args). → **The 1373-edit `$N → ?` sweep is
   unnecessary.** Existing queries run unchanged. *(Phase 3: 2–3 days → ~1 day.)*
   Probe: `spike/sqlite_param_test.go`.

2. **Timestamps must be declared `DATETIME`, not `TEXT`.** A `TEXT`-declared
   column is returned as `string` and `Scan` into `*time.Time` fails; a
   `DATETIME`-declared column returns `time.Time` for both RFC3339 *and*
   `datetime('now')` values. SQLite's loose affinity still stores text either
   way, so this is a pure schema-declaration fix — **no scan-site code changes**.
   This corrected D5 (which wrongly claimed TEXT scans transparently). Probes:
   `spike/sqlite_scan_probe_test.go`, `spike/sqlite_columntype_probe_test.go`.

3. **`RETURNING` and JSON→`[]byte` scanning work unchanged.** No translation
   needed for those query classes.

4. **Per-connection PRAGMA footgun solved.** `foreign_keys=ON` etc. are
   connection-scoped; applied via modernc's `_pragma=` DSN params (run on every
   new pooled connection), validated by `TestSQLiteForeignKeysEnforcedAcrossPool`.

5. **WAL concurrency is fine for this workload.** `TestSQLiteConcurrentWrites`
   (16 goroutines × 50 writes) produced zero `database is locked` errors.

## What's been built

Code (all tested, all on `master`):

| File | Purpose |
|---|---|
| `server/sqlsplit.go` (+test) | Quote-aware `;` statement splitter for SQLite migration loading |
| `server/sqlite.go` (+test) | `OpenSQLite` — D4 pragmas via DSN, pool config |
| `server/database.go` | `execScript` helper; `RunMigrations` SQLite path (migrations-table bootstrap, `SELECT 1` existence check, splitter integration) |
| `server/server.go` | `Driver` field on `Server` |
| `bootstrap/bootstrap.go` | Branches on `DB_DRIVER` (postgres vs sqlite) |
| `pkg/config/database.go` | `Driver`/`SQLitePath` config; PG fields now optional in sqlite mode |
| `server/database_sqlite_test.go` | Migration-runner integration test (splitter + `$1` + idempotency) |
| `spike/*` | Throwaway probes documenting driver behavior (kept as evidence) |

Driver added: `modernc.org/sqlite v1.54.0`. Config flags: `DB_DRIVER`
(default `postgres`), `SQLITE_PATH` (default `./data/zettelgarden.db`).

## What's next (Phase 1 remainder)

1. **Wire the `cards` read/write path through SQLite** with a consolidated
   mini-schema for that table, and get **one handler test green on `:memory:`**.
   This surfaces the first real `NOW()`/model translations and validates the
   end-to-end spike. Pairs naturally with Phase 2 (the consolidated schema).
   Note: the existing test infra (`tests/conftest.go`) is PG-specific, so a
   SQLite handler test needs a small standalone setup (full conftest rewrite is
   Phase 6a).
2. **Bulk-import timing check** — load ~1k cards through the ETL path to confirm
   modernc's insert throughput is tolerable for the 15k-card cutover (fallback:
   CGO `mattn/go-sqlite3` for the one-shot ETL tool only).

## Open questions for Nick

- **Phase 2 input:** can you provide a `pg_dump --schema-only` of the live dev
  DB? That's the source for the consolidated SQLite schema.
- **`DB_PORT`/`DB_HOST` in CI:** CI sets these via the postgres service; once
  SQLite is the default, the CI workflow's `services: postgres` block can be
  removed (Phase 6a/7a).

## Revision cadence

This doc tracks execution progress; the **design doc** remains the source of
truth for the plan, decisions, and phase detail. Update this status doc at the
end of each working session.
