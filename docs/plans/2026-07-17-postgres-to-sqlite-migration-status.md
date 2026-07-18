# PostgreSQL → SQLite Migration — Status

**Last updated:** 2026-07-18
**Plan:** [`2026-07-17-postgres-to-sqlite-migration-design.md`](./2026-07-17-postgres-to-sqlite-migration-design.md)
**Tracking:** epic `Zettelgarden-c7j` · Phase 0 `Zettelgarden-bw1` (closed) · Phase 1 `Zettelgarden-2u2` (in progress)

## TL;DR

Implementation is underway. **Phase 0 done; Phase 1 done.** Every
high-risk *unknown* has been resolved empirically, and the findings **shrank
the plan**: the 1373-edit placeholder sweep is gone, and a latent timestamp
landmine was caught and fixed (schema-side, no code changes). The cards
read+write path now runs end-to-end on `:memory:` SQLite through the real
`services.CreateCard` flow, and bulk-insert timing confirms modernc handles the
full 15k-card import in **under a second** — no CGO fallback needed for the
ETL. **Phase 2 (consolidated schema) is next** and is gated on a live
`pg_dump --schema-only` of the dev DB.

## Phase status

| Phase | Status | Notes |
|---|---|---|
| 0 — pgvector image swap | ✅ **Done** | `docker-zettel-run.yml` + `.github/workflows/go.yml` → `postgres:16-alpine` |
| 1 — Spike / de-risk | ✅ **Done** | Driver, splitter, conn helper, migration runner, concurrency, compatibility probes, **cards read+write wiring on `:memory:`**, and **bulk-import timing** all done & tested. |
| 2 — Consolidated SQLite schema | ⬜ Not started | Needs a live `pg_dump --schema-only` of the dev DB (still open). **Must declare timestamp columns `DATETIME` (see D5).** A `services/testdata/spike_cards.sqlite.sql` mini-schema already demonstrates the translation rules for the cards sub-graph. |
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
6. **The cards read+write path runs end-to-end on SQLite.** `services.CreateCard`
   → `GetFullCard` (plus the tag / backlink / audit side effects) round-trips
   on an `:memory:` DB built from a hand-derived mini-schema
   (`services/testdata/spike_cards.sqlite.sql`) via the Phase 1 splitter. This
   is the first path that actually exercises the driver, splitter, `RETURNING`,
   `time.Time` scan from `DATETIME`, and JSONB-as-TEXT scan together. Covered by
   `services/cards_sqlite_test.go` (`TestCreateCardSQLite`, incl. a
   structured-data round-trip sub-test).
7. **modernc is fast enough for the ETL — CGO fallback unnecessary.** The bulk
   probe (`spike/sqlite_bulk_timing_test.go`) inserts **15,000 cards in 780ms**
   (≈52µs/card) on a file-backed WAL DB, *with one transaction per insert*
   (the slowest pattern the real CreateCard flow uses). The Phase 6b ETL tool,
   which will batch inserts in fewer transactions, will be faster still. So
   `mattn/go-sqlite3` is **not** needed for the one-shot import — modernc stays
   for both runtime and ETL, preserving the pure-Go / no-CGO goal.
8. **`NOW()` translations are mechanical and driver-neutral.** The first four
   sites in `services/cards.go` (CreateCard INSERT, UpdateCard, DeleteCard
   soft-delete, UpdateCardStructuredData) now bind `time.Now().UTC()` as a
   parameter. Works identically on Postgres and SQLite, and makes tests
   deterministic. The remaining `NOW()` sites (tags, backlink, etc.) are Phase 3
   and follow the same pattern.

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
| `spike/*` | Throwaway probes documenting driver behavior (kept as evidence), incl. the bulk-insert timing probe |
| `services/testdata/spike_cards.sqlite.sql` | Phase 1 spike mini-schema (users + cards + tags + card_tags + backlinks + audit_events); demonstrates the Phase 2 translation rules (SERIAL→AUTOINCREMENT, TIMESTAMP→DATETIME, JSONB→TEXT, NOW()/CURRENT_TIMESTAMP→datetime('now')). Superseded by Phase 2's consolidated schema. |
| `services/cards_sqlite_test.go` | End-to-end cards read+write spike on `:memory:` SQLite (`TestCreateCardSQLite` + structured-data round-trip sub-test) |
| `services/cards.go` | First 4 `NOW()` sites → app-side `time.Now().UTC()` (CreateCard, UpdateCard, DeleteCard, UpdateCardStructuredData) |

Driver added: `modernc.org/sqlite v1.54.0`. Config flags: `DB_DRIVER`
(default `postgres`), `SQLITE_PATH` (default `./data/zettelgarden.db`).

## What's next (Phase 2)

Phase 1 is **done** — the spike has de-risked every unknown that could be
de-risked without the full schema. The next blocker is Phase 2 (consolidated
SQLite schema), which is gated on the open question below.

1. **Phase 2 input needed from Nick:** a `pg_dump --schema-only` of the live dev
   DB. That is the source for `schema/sqlite/schema.sqlite.sql`. The hand-derived
   `services/testdata/spike_cards.sqlite.sql` mini-schema already proves the
   translation rules on the cards sub-graph and can seed the consolidated file.
2. **Phase 3 is now ~1 day:** placeholder sweep already eliminated; `NOW()`
   translation pattern proven (first 4 sites done in Phase 1, ~99 remaining
   non-test occurrences across 31 files). Remaining: `INTERVAL` (6 files),
   `ILIKE` (5 files), `::` casts (2 files), and the rest of `NOW()`.
3. **Phase 6b ETL:** no CGO fallback needed — modernc inserts 15k cards in
   ~0.8s (Phase 1 probe). ETL can be pure Go using modernc for both read-from-PG
   (`lib/pq`) and write-to-SQLite.

## Open questions for Nick

- **Phase 2 input (blocking):** can you provide a `pg_dump --schema-only` of
  the live dev DB? That's the source for the consolidated SQLite schema. Until
  this lands, Phase 2 can only proceed table-by-table from the migration files
  (as the spike mini-schema did for cards).
- **`DB_PORT`/`DB_HOST` in CI:** CI sets these via the postgres service; once
  SQLite is the default, the CI workflow's `services: postgres` block can be
  removed (Phase 6a/7a).

This doc tracks execution progress; the **design doc** remains the source of
truth for the plan, decisions, and phase detail. Update this status doc at the
end of each working session.
