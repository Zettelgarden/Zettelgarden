# SQLite Schema Sync + Postgres Retirement

**Date:** 2026-08-04
**Status:** Implemented in code (`54046de9`) and **deployed**. Both instances now run on SQLite: `zg-internal` (server-3) since 2026-07-28; the public instance (`zettelgarden.com`, 192.168.0.93) since 2026-08-04 via the `:sqlite-cutover-public` image. Only the final swap to the PG-free `:latest` + `postgres:16` decommission remain (see below).
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

2. **The *internal* instance is off Postgres; the *public* instance is not.**
   The PG→SQLite cutover (epic `Zettelgarden-c7j`) completed 2026-07-28 for
   `zg-internal` (server-3 / 192.168.0.20), now on SQLite. **But 192.168.0.93
   also hosts a second, *public* instance (`zettelgarden.com`,
   `/mnt/nas-2-fast-data/config/zettelgarden`) that is still on Postgres (and
   still on B2 file storage)** — it was never part of the cutover. Its Postgres
   is a *live* database, not a standby; nothing about it has been decommissioned.
   The 116 MB `pg_dump` on the NAS is a 2026-07-28 snapshot of the *internal*
   instance's source DB and is now stale relative to the public instance.

   **Consequence:** the code changes below are correct for the repo and for
   `zg-internal`, but the resulting binary **cannot run on .93** until the
   public instance is migrated to SQLite. `build.sh` builds `:latest` from
   master and deploys it straight to .93 — so running it now would push a
   Postgres-free binary at a Postgres-backed site and break `zettelgarden.com`.
   A guard has been added to `build.sh` to prevent this.

## The two sync problems (kept distinct)

1. **SQLite fresh-build ↔ SQLite existing-build** — the OIDC bug. *Fixed here.*
2. **Postgres schema ↔ SQLite schema** (cross-engine parity) — a *retiring*
   concern that self-resolves with Phase 7b. Out of scope; the ETL
   (`cmd/migrate-pg-to-sqlite`, now deleted) and `translate.py` (now deleted)
   were its only consumers.

## Decisions

- **Retire Postgres from the codebase now** (scope: the repo + `zg-internal`).
  It is Phase 7b, already scoped, and a net simplification — deleting it removes
  the cross-engine parity surface entirely and a chunk of dead code (`lib/pq`,
  `ConnectToDatabase`, the PG bootstrap branch, the dead
  `cmd/migrate-pg-to-sqlite`, the stale `translate.py` lineage, and 148 archived
  numbered migrations). **Caveat:** this is *code-only*. The public instance on
  .93 still needs Postgres until its own migration lands, so the new binary is
  not yet deployable there. The deleted ETL (`cmd/migrate-pg-to-sqlite`) and
  dual-driver support remain recoverable from `git` (`0a9a0599`) / the image
  currently running on .93 — which is exactly what the public migration will use.
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

## Public-instance migration on 192.168.0.93 — DONE 2026-08-04

The public instance (`zettelgarden.com`, `/mnt/nas-2-fast-data/config/zettelgarden`)
was migrated from Postgres to SQLite on 2026-08-04 (`Zettelgarden-f3r`).

What ran:
- Built the `:sqlite-cutover-public` image + the `migrate-pg-to-sqlite` ETL from
  `0a9a0599` (dual-driver: PG + SQLite + the ETL — recoverable from git since
  `54046de9` deleted it).
- ETL copied the live `zettelkasten` Postgres (the `postgres:16` container on
  .93, reached via its `192.168.10.3` NIC) into `./data/zettelgarden.db`: **67
  tables, 499,581 rows pg==sqlite exact** (a stop + final ETL cleared the
  `scheduled_job_runs` live-write race).
- Flipped the public backend to `:sqlite-cutover-public` with `DB_DRIVER=sqlite`
  + `SQLITE_PATH` + the `./data:/usr/src/app/data` volume. Boots clean; Typesense
  connected; auth live (401); scheduler writing (`scheduled_job_runs` growing);
  WAL active; 0 restarts. Downtime ~5.5 min.

Rollback safety: `:pre-sqlite-public` image tag (the old 2026-05-03 PG image), a
`pg_dump -Fc` insurance dump (57 MB) at `backups/`, and `.env.pre-sqlite-cutover`
+ `docker-compose.yml.pre-sqlite-cutover`. The `postgres:16` container is **left
running** as fallback — nothing reads it.

Data note: the public DB carried ~109k pre-existing FK-orphan rows (92k
`llm_query_log`, 8k `card_tags`, rest across log/junction tables — all
referencing long-deleted users; accumulated cruft `zg-internal` never had).
Harmless at runtime (FK enforcement is forward-looking on writes; the app only
writes valid refs), so they were left in place as a faithful copy. 1,723 orphan
`summarizations` rows were deleted (garbage referencing non-existent user 1 +
cards). File storage needed no migration (0 files; never on B2).

Final image swap — DONE 2026-08-04 13:42 UTC-4:
- Built `:latest` from master (`bf187fcd`) on .93 (`6854c18d0efe`, PG-free) and
  flipped the public backend to it. Boots clean; auth live (401); scheduler
  writing (a `scheduled_job_runs` row landed at 13:42:02, right after the
  13:40:55 start); WAL active; integrity `ok`; `filter_tags` reads fine as
  `{starred}` — the `pq.StringArray` → `models.StringArray` swap is
  byte-compatible, as designed. The cutover image (`:sqlite-cutover-public`)
  is retained on .93 as a one-line rollback.
- Pushed that `:latest` to Docker Hub from .93 (`sha256:e046f3e8…`) so the
  registry matches and a future `docker compose pull` can't regress. All
  instances are on SQLite now, so a PG-free `:latest` is safe everywhere.
- Removed the `ZG_PUBLIC_DEPLOY_CONFIRMED` guard from `build.sh` — its premise
  (public instance still on PG) no longer holds.

Remaining tail (low priority):
- **Retire the `postgres:16` container** after ~1 week green (take a final
  fresh `pg_dump -Fc` first). Cutover artifacts retained for re-ETL:
  `~/code/zg-cutover` worktree + `/tmp/zg-out/migrate-pg-to-sqlite`.

`zg-internal` (server-3) was already on SQLite and is unaffected.

## Follow-ups (filed / noted)

- **`Zettelgarden-2lk`** — repurposed: the drift test landed; the full
  numbered-migrations framework is **deferred** unless drift recurs. The
  hand-maintained self-heal list is now CI-guarded, which is the right scope
  for a single-engine single-user app.
- **`Zettelgarden-f3r` (DONE 2026-08-04)** — public instance migrated to SQLite
  via the `:sqlite-cutover-public` image (see "Public-instance migration" above).
  Only the final `:latest` swap + `postgres:16` decommission remain.
- **`Zettelgarden-gve` (closed)** — originally "decommission standby PG on .93";
  closed because .93's Postgres is the *public* instance's live database, not a
  standby. Subsumed by the public-instance migration issue above.
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
