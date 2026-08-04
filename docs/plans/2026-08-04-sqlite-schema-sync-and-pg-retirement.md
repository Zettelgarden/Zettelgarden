# SQLite Schema Sync + Postgres Retirement

**Date:** 2026-08-04
**Status:** Implemented in code (commit `54046de9`). ⚠️ **Not deployable to the public instance on 192.168.0.93** — that instance is still on Postgres and must be migrated first (see "Remaining work — the public instance on 192.168.0.93" below).
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

## ⚠️ Remaining work — the public instance on 192.168.0.93 (NOT done; blocks deploying this commit there)

The changes above are **code-only** and are **not deployable to 192.168.0.93**.
That host runs the *public* instance (`zettelgarden.com`,
`/mnt/nas-2-fast-data/config/zettelgarden`), which is **still on Postgres** (and
still on B2 storage). Its Postgres is a live database:

- **Do not** stop Postgres on .93.
- **Do not** run `./build.sh` — it builds `:latest` from master (now
  Postgres-free) and deploys it straight to .93, which would take the public
  site down. `build.sh` now has a guard that aborts unless
  `ZG_PUBLIC_DEPLOY_CONFIRMED=1`.

Before this commit can reach .93, the public instance needs the **same ETL +
cutover `zg-internal` got** (tracked in its own bd issue), performed with the
pre-retirement tooling:

1. **Verify state on .93** — confirm the public instance's `DB_DRIVER` / whether
   a `data/*.db` exists / what image runs (believed still Postgres + B2 as of
   2026-07-29; not re-verified).
2. **Migrate the data** — `pg_dump` the public DB → import with
   `cmd/migrate-pg-to-sqlite` (checked out from `0a9a0599`, or built from the
   running image) into a SQLite file. The public DB has diverged from the
   2026-07-28 snapshot, so this must be a **fresh** dump.
3. **Migrate file storage** B2 → local (`STORAGE_DIR=/usr/src/app/data/files`)
   via `cmd/migrate-storage` — same as the server-3 cutover
   (`docs/plans/2026-07-29-s3-to-local-file-storage-status.md`).
4. **Boot + A/B verify** a pre-retirement binary against the new SQLite + local
   store on .93 (read-path diff vs the live PG/B2), then flip the public
   compose to SQLite.
5. **Only then** deploy the new (Postgres-free, local-storage) `:latest` to .93
   and retire .93's Postgres (final fresh `pg_dump -Fc` to the NAS, then
   stop/remove the service). Retain the `:pre-sqlite` tag + NAS dumps as the
   cold fallback of record.

`zg-internal` (server-3) is unaffected and already safe — this section is about
the public instance only.

## Follow-ups (filed / noted)

- **`Zettelgarden-2lk`** — repurposed: the drift test landed; the full
  numbered-migrations framework is **deferred** unless drift recurs. The
  hand-maintained self-heal list is now CI-guarded, which is the right scope
  for a single-engine single-user app.
- **Public-instance migration (new bd issue)** — migrate `zettelgarden.com` on
  192.168.0.93 from Postgres+B2 to SQLite+local storage. **Prerequisite** for
  deploying this commit to .93 and for retiring .93's Postgres. `build.sh` is
  guarded against an accidental deploy until this lands.
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
