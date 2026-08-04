# SQLite Schema Sync + Postgres Retirement

**Date:** 2026-08-04
**Status:** Implemented (code); one ops step remains (decommission the standby Postgres)
**Author:** Nick + Pi
**Tracking:** `Zettelgarden-2lk` (schema sync), epic `Zettelgarden-c7j` Phase 7b (PG retirement)

## Context

Two things converged in one conversation:

1. **The 2026-08-04 OIDC prod outage.** OIDC login broke with
   `no such column: oidc_provider`. Root cause: the SQLite migration path applies
   the consolidated `schema/sqlite/schema.sqlite.sql` only on **fresh** builds.
   The numbered Postgres migration `0147` (which added the oidc columns) lives in
   `./schema` — never scanned on SQLite, and Postgres-only syntax anyway. So an
   *existing* prod SQLite DB never received the columns. The immediate hotfix
   (commit `3d19255e`) added `ensureSQLiteSchemaUpgrades()` to back-fill them
   idempotently on boot — but that left a **hand-maintained list** with no
   mechanical link to the consolidated schema. The next schema change would
   reproduce the outage unless a human remembered to update the list.

2. **Postgres is already dead in production.** The PG→SQLite cutover
   (epic `Zettelgarden-c7j`) completed 2026-07-28. Prod (`zg-internal`,
   server-3) runs on SQLite. Postgres on 192.168.0.93 sits untouched,
   read-only, as a standby. Nothing reads from it. The real rollback safety is
   the 116 MB insurance `pg_dump` on the NAS + the `:pre-sqlite` image tag, not
   the warm process. Phase 7b (decommission PG + drop `lib/pq` + archive the
   numbered migrations) was already scoped in the migration status doc; this
   doc pulls it forward ~1 week past the soft 2-week gate (the highest-risk
   boot/scan window cleared on day 1).

## The two sync problems (kept distinct)

1. **SQLite fresh-build ↔ SQLite existing-build** — the OIDC bug. *Fixed here.*
2. **Postgres schema ↔ SQLite schema** (cross-engine parity) — a *retiring*
   concern that self-resolves with Phase 7b. Out of scope; the ETL
   (`cmd/migrate-pg-to-sqlite`, now deleted) and `translate.py` (now deleted)
   were its only consumers.

## Decisions

- **Retire Postgres now.** It is Phase 7b, already scoped, and a net
  simplification. Deleting it removes the cross-engine parity surface entirely
  and a chunk of dead code (`lib/pq`, `ConnectToDatabase`, the PG bootstrap
  branch, the dead `cmd/migrate-pg-to-sqlite`, the stale `translate.py`
  lineage, and 148 archived numbered migrations).
- **Keep the self-heal list; add a drift test.** The full numbered-migrations
  framework (the earlier "Option 3") is overkill for a single-engine,
  single-user app with occasional schema changes. The right-sized guard is the
  existing `ensureSQLiteSchemaUpgrades` list **plus a drift test** that fails CI
  if the list and the consolidated schema diverge. Defer the framework unless
  drift recurs.
- **Retiring PG does not by itself fix the OIDC class of bug** — that bug is
  within-SQLite. The two efforts compose: retirement removes parity pressure;
  the drift test is the mechanism guard.

## What changed (this session)

### Schema sync guard — the actual fix for the OIDC class

- `server/database.go`: the self-heal upgrades list is hoisted to a
  package-level `sqliteSelfHealUpgrades` so the runner and the test share one
  definition.
- `server/database_sqlite_upgrade_test.go`: new
  `TestSelfHealListMatchesSchemaDelta`. It enforces, for the `users` table:
  `set(columns in fresh consolidated build) − set(frozen pre-self-heal baseline)
  == set(self-heal-managed columns)`. The baseline is a frozen constant (the 30
  users columns as of 2026-08-04, pre-oidc); it is never edited. If you add a
  column to `schema.sqlite.sql` without a matching self-heal entry (the exact
  OIDC gap), this test fails and names the missing column. (Indexes are still
  covered by the existing behavioral `..._IndexIsEnforced` test.)

### Postgres retirement (Phase 7b) — the simplification

- **Dropped `github.com/lib/pq`** from `go.mod`. The three runtime usages:
  - `models/notifications.go`: `pq.StringArray` (for `filter_tags`, stored as
    PG-array text `{a,b}`) → new `models.StringArray`
    (`models/stringarray.go`) implementing the same PostgreSQL array-literal
    format byte-for-byte. Covered by encode/decode/round-trip tests + a
    2000-iteration fuzz round-trip over an alphabet including commas, braces,
    quotes, backslashes, and whitespace.
  - `handlers/schemas.go`: the `*pq.Error` type-assertion in
    `isDuplicateKeyError` removed; the case-insensitive string fallback already
    handles modernc's `UNIQUE constraint failed: ...` (the only live driver).
  - `server/database.go`: `ConnectToDatabase` + the blank pq import deleted.
- **Collapsed the runtime to SQLite-only.** `bootstrap.InitServer` is
  unconditional `OpenSQLite`; `pkg/config` dropped the Postgres fields,
  `DB_DRIVER` env read, and the unused `ConnectionString`/`TestConnectionString`
  methods (`Driver` retained as a constant `"sqlite"` tag for the Server/test
  wiring). `tests/conftest.go` dropped the Postgres `else` branch, the PG env
  defaults, and the dead Postgres `setval` block.
- **Deleted dead tools/lineage.** `cmd/migrate-pg-to-sqlite/` (one-shot ETL,
  job done) and `schema/sqlite/source/` (`translate.py` + `dev_schema.sql` —
  the consolidated schema has been hand-evolved past the pg_dump snapshot for
  oidc, `task_saved_searches`, and the `habits` drop, so the lineage is stale).
- **`cmd/migrate-storage/`**: its Postgres read branch removed (the B2→local
  ETL is done; the tool is now SQLite-only and a deletion candidate for the
  S3 epic).
- **Archived 148 numbered Postgres migrations** `schema/*.sql` →
  `schema/archive/postgres/`. Nothing scans them (prod + CI are SQLite; the PG
  runner is gone).
- **Operator docs/config** updated: `.env.example`, `docker-zettel-run.yml`,
  `AGENTS.md`, and the stale `lib/pq`/`DB_DRIVER` code comments.

### Verification

- `go build ./...` clean.
- `go vet ./...` clean except 2 **pre-existing, unrelated** warnings in
  `handlers/oauth.go` (documented in the cutover notes; not touched here to
  keep the retirement diff out of the auth path).
- `go test ./...` — **all packages green**, including `models` (StringArray),
  `server` (drift test), `handlers` (the full 30s suite), `schema/sqlite`, and
  `cmd/migrate-storage`.

## Remaining ops step (NOT done by this change — Nick's action)

The code no longer references Postgres, but the **standby Postgres process on
192.168.0.93 is still running**. Decommission it:

1. Confirm ≥1 green day on SQLite post-this-deploy (smoke: login, card
   read/write, notifications read — the `filter_tags` path that the
   `pq.StringArray` → `StringArray` swap touched).
2. Take a final `pg_dump -Fc` to the NAS alongside the existing 2026-07-28
   insurance dump (belt-and-suspenders).
3. Stop + remove the Postgres container/service on 192.168.0.93. Retain the
   `:pre-sqlite` image tag and the NAS dumps as the cold fallback of record.
4. Remove any `DB_*` / `DB_HOST`/`DB_PORT`/etc. from the prod `.env`/compose
   (no longer read; tidy only).

This is reversible for as long as the NAS dumps + `:pre-sqlite` image are kept.

## Follow-ups (filed / noted)

- **`Zettelgarden-2lk`** — repurposed: the drift test landed; the full
  numbered-migrations framework is **deferred** unless drift recurs. The
  hand-maintained self-heal list is now CI-guarded, which is the right scope
  for a single-engine single-user app.
- **Vestigial `Server.Driver` field** — now always `"sqlite"`. The
  `if S.Driver == "sqlite"` checks in `database.go`/`conftest.go` are
  always-true. A future cleanup can remove the field + collapse the checks; left
  as-is here to keep the retirement diff bounded.
- **`cmd/migrate-storage` deletion** — its job (B2→local) is done; candidate
  for removal under the S3 epic (separate from this retirement).
- **`handlers/oauth.go` vet warnings** — 2 pre-existing (`userRes`/`emailRes`
  used before error check); left untouched here.
- **NOT NULL-on-add / type-change / data-backfill columns** — SQLite `ADD
  COLUMN` cannot express these (needs add-nullable → backfill → table rebuild).
  None exist today; if one is needed, the self-heal list + drift test are
  insufficient and the deferred numbered-migrations framework becomes warranted.
