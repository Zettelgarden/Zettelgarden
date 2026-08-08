> **ARCHIVED** — Historical document moved to `docs/archive/` on 2026-08-08 during the documentation audit (Zettelgarden-0ui). Does not describe the current app; kept for the record.

# PostgreSQL → SQLite Dual-Backend Feasibility Analysis

**Date:** 2026-07-02
**Goal:** Assess supporting **both** PostgreSQL (web/hosted) and SQLite (desktop/embedded) from one Go backend codebase.

## TL;DR

**Feasible, with medium effort.** The codebase is actually well-positioned for this because all DB access already flows through a single `models.Database` interface over `database/sql`. The single largest obstacle is mechanical (835 Postgres `$N` placeholders that must become SQLite `?`), and it can be solved **once** with a small dialect-shim rather than editing 835 call sites. A handful of genuinely Postgres-specific features (a few query idioms + 7 PL/pgSQL trigger migrations) need targeted rewrites. Search is already delegated to Typesense with an ILIKE fallback, so FTS is not a blocker.

Recommended path: a **dialect abstraction layer** that keeps PostgreSQL as the default and adds a pure-Go SQLite driver, validated by the existing transaction-rollback test harness. No need for a fork or a query-builder rewrite.

---

## 1. Current state of the data layer

### 1.1 Architecture (the good news)
- **One DB interface.** `go-backend/models/database.go` defines `Database` wrapping `*sql.DB` *and* `*sql.Tx`. Every handler/service already calls `GetDB()` / `BeginTx()` against this interface. This is the abstraction a dual-backend strategy plugs into.
- **One connection point.** `server/database.go::ConnectToDatabase()` is the only place the `postgres` driver and connection string are constructed. `server/server.go` holds the single `*sql.DB`.
- **Env-based config.** `pkg/config/database.go` reads `DB_HOST/PORT/USER/PASS/NAME`. Adding a `DB_DRIVER` env var (or a `DATABASE_URL`) is a localized change.
- **Migration system.** 144 sequential `.sql` files in `go-backend/schema/`, tracked in a `migrations` table, applied in `RunMigrations()`. Simple and dialect-neutral in mechanism — only the *contents* are Postgres-flavored.
- **Search is decoupled.** Typesense (optional, 322 references) carries real search; when it's absent the code falls back to `ILIKE`. SQLite can reuse the same fallback, so search is not a conversion problem.
- **Frontend is already Electron-aware.** `zettelkasten-front/package.json` has `"main": "dist-electron/main.js"`, so a desktop shell could ship the React UI today and embed the Go backend as a sidecar talking to a local SQLite file.

### 1.2 Scale of SQL in Go
| Metric | Count |
|---|---|
| Go files containing SQL | ~98–91 |
| `$N` placeholder sites | **835** |
| `RETURNING` clauses | 90 |
| `ON CONFLICT` (upsert) | 40 |
| `pq.Array()` call sites | 8 |
| `pq.StringArray` field types | 6 |
| `ILIKE` query sites | ~15 (4 files) |
| `AT TIME ZONE` / timestamptz sites | 22 (4 files: tasks, stats) |
| `NOW()`/`CURRENT_TIMESTAMP` in Go | 173 |
| `STRING_AGG` | 1 (`handlers/search.go`) |
| `generate_series` / `DATE_TRUNC` / `INTERVAL` / `GREATEST` in Go | **0** |

The last row is the most encouraging finding: the genuinely painful Postgres date/time machinery is **not** embedded in the Go query layer — only in schema triggers.

---

## 2. Postgres-specific features × SQLite compatibility

| Feature | Where it lives | SQLite status | Conversion effort |
|---|---|---|---|
| `$1,$2` placeholders | 835 Go sites | needs `?` | **Solve once** via dialect shim (see §3) |
| `SERIAL PRIMARY KEY` | 44 schema files | `INTEGER PRIMARY KEY` | Schema port (scriptable) |
| `RETURNING` | 90 Go sites | ✅ supported 3.35+ (2021) | none |
| `ON CONFLICT … DO UPDATE` | 40 Go + schema | ✅ supported 3.24+ (2018) | none |
| `NOW()` | 173 Go sites | `CURRENT_TIMESTAMP` / `datetime('now')` | dialect shim or app helper |
| `pq.Array` / `pq.StringArray` | 14 sites | no arrays — CSV/JSON/temp-table/per-row | medium, localized |
| `ILIKE` | ~15 sites, 4 files | no ILIKE → `LIKE` w/ `NOCASE` collation or `LOWER()` | low–medium |
| `AT TIME ZONE` / `timestamptz` | 22 sites, 4 files | no TZ — store UTC, filter in app | medium, localized |
| `JSONB` columns | 11 schema files | TEXT + JSON1. **No arrow operators (`->>`) used in Go** — JSON is parsed in application code | low |
| `tsvector`/`GIN` | schema triggers only | FTS5 | medium (only in triggers) |
| **PL/pgSQL triggers/functions** | **7 schema files** | SQLite triggers are SQL-only (no procedural flow) | **high — needs hand port** |
| `STRING_AGG` | 1 site | `GROUP_CONCAT` | low |
| `CASCADE` drops | `ResetDatabase` only | different mechanism | low |
| `::` casts | minimal in Go | manual | low |

### 2.1 The two real blockers

**(A) Placeholder syntax — 835 sites, but solved once.**
This is the headline cost *if* you try to edit each site. You should not. Postgres `$1,$2…` and SQLite `?,?,?` differ only in spelling, and the mapping is a trivial, order-preserving regex (`\$(\d+)` → `?`). Put the rewrite behind the `Database` interface (or in a tiny wrapper driver) and **zero** of the 835 call sites change.

**(B) PL/pgSQL triggers — 7 schema files, must be hand-ported.**
These are the schema migrations that use procedural logic (no SQL equivalent in SQLite without rewriting):

- `0093-user-stats-triggers.sql` — maintains a denormalized `user_stats` counter table (card/task/file/chat/cost/revenue counts) via `AFTER INSERT/UPDATE/DELETE` triggers with `TG_OP`, `NEW/OLD`, `IF/ELSIF`, `GREATEST`, `ON CONFLICT`.
- `0122-add-notification-helper-functions.sql` — `delete_notification()` function using `TG_TABLE_NAME` + `CASE`.
- `0123-add-email-notification-trigger.sql`, `0124-add-rss-notification-trigger.sql` — fire insert/delete on notifications.
- `0096-llm-jobs-updated-at.sql`, `0102-llm-jobs-notify.sql` — auto-maintain `updated_at` / `notify` channel.
- `0067-chat-system.sql` — (partial) trigger for tsvector on messages.

SQLite triggers cannot express `IF/ELSIF`, `GREATEST`, `ON CONFLICT`, or `TG_OP` branching. Options:
1. **Rewrite as simpler SQLite triggers** where the logic is just insert/update (the notification inserts).
2. **Move to application code** where the logic is conditional (the `user_stats` counters are a classic case for an app-level `UpdateUserStats()` hook, or even a lazy recompute).
3. **Skip the optimization** in the desktop build (recompute `user_stats` on read) — acceptable for single-user SQLite.

This is the one piece that cannot be automated.

---

## 3. Recommended strategy: dialect abstraction (support both)

Keep **one** query layer, branch on dialect at the boundary. Do **not** fork, and do **not** rewrite into a query builder.

### 3.1 Driver choice
Use **`modernc.org/sqlite`** (pure Go, no CGO). This is the decisive choice for a desktop target:
- No CGO ⇒ trivial cross-compilation (Windows/macOS/Linux, incl. arm64) from any host.
- Plays well with bundling the backend as a desktop sidecar (Electron already present) or behind a Wails/Tauri shell.
- Performance is good enough for a single-user desktop DB; the web/hosted path stays on Postgres.

(`mattn/go-sqlite3` is faster but CGO-bound — avoid unless profiling later demands it.)

### 3.2 The shim (the keystone change)
Introduce a `Dialect` (postgres | sqlite) and route through a wrapper:

```
// pseudocode
type Dialect int
const ( DialectPostgres Dialect = iota; DialectSQLite )

// At the driver boundary, for SQLite:
func (d *sqliteDB) Query(q string, args ...) {
    q = translatePlaceholders(q)   // $1,$2.. -> ?,?..
    q = translateBuiltins(q)        // NOW() -> CURRENT_TIMESTAMP (simple cases)
    return inner.Query(q, args...)
}
```

- Placeholder rewrite: one regex, covers all 835 sites for free.
- `NOW()` → `CURRENT_TIMESTAMP`: cheap global rewrite; `CURRENT_TIMESTAMP` works on **both** engines, so this can be a straight find/replace in code and avoided in the shim.
- The 8 `pq.Array` sites: these can't be shimmed cleanly (array semantics differ). Convert to per-row `IN (?,?,?)` expansion or a small JSON/temp-table helper — localized to those 8 call sites.

The `models.Database` interface is **already** the right seam; add a SQLite implementation of it next to the Postgres one.

### 3.3 Schema (two flavors, generated + hand-finished)
Maintain a `schema/` (Postgres) and a new `schema-sqlite/` set. Most of the 144 files port mechanically:
- `SERIAL PRIMARY KEY` → `INTEGER PRIMARY KEY`
- `TIMESTAMP`/`TIMESTAMPTZ` → `TEXT` (ISO-8601 UTC) — store UTC everywhere
- `BOOLEAN` → `INTEGER` (0/1) or keep `BOOLEAN` (SQLite is flexible)
- `JSONB` → `TEXT`
- `VARCHAR(N)` → `TEXT`

A script (`scripts/schemato-sqlite`) can do the bulk, then the 7 trigger files get hand-authored equivalents (or app-level hooks). Track SQLite migrations in the same `migrations` table; `RunMigrations` picks the dir from the dialect.

### 3.4 The four PG-specific query idioms (targeted branches)
Small `sqlcompat` helpers, branched by dialect:
- **`ILIKE`** (4 files: search, facts, files, cards/services) → `LIKE` with `NOCASE` collation or `LOWER(col) LIKE LOWER(?)`.
- **`AT TIME ZONE`** (tasks, stats) → app stores UTC; the "user-local day" filtering becomes a Go-side date computation passed in as a bound value.
- **`STRING_AGG`** → `GROUP_CONCAT`.
- **`pq.Array`** → expand to `IN (?, ?, …)` or use a small temp-table/junction.

These are ~6–8 files total, all identified.

### 3.5 Concurrency
- **Desktop/SQLite:** enable `PRAGMA journal_mode=WAL; PRAGMA busy_timeout=...; PRAGMA foreign_keys=ON;` on open. WAL gives concurrent readers + one writer — ideal for single-user. Limit pool to 1 writer.
- **Web/Postgres:** unchanged (`SetMaxOpenConns(25)` etc.).
- Decision: the hosted multi-tenant server stays on Postgres; SQLite is the local mode. One writer is fine for desktop.

---

## 4. Phased plan

| Phase | Outcome | Risk |
|---|---|---|
| **0. Spike** (~1–2 days) | Add `modernc.org/sqlite`; port `0001-initial.sql`; get one `SELECT` round-tripping via the shim against an in-memory DB. | Low — proves the core idea. |
| **1. Shim + tests** | `Dialect` + placeholder/`NOW()` rewriter behind `Database`. Run the existing `*_test.go` (which already use transaction rollback) against SQLite. | Medium — tests surface incompatibilities fast. |
| **2. Schema port** | `schema-sqlite/` generated + 7 trigger files hand-authored or converted to app hooks. | Medium-high — triggers are the manual work. |
| **3. PG idioms** | `sqlcompat` for `ILIKE`/`AT TIME ZONE`/`GROUP_CONCAT`/`pq.Array` (~8 files). | Low-medium. |
| **4. Desktop shell** | Wire backend as sidecar behind the existing Electron `main.js`, pointing at `~/.zettelgarden/zettel.db`. Decouple Typesense/S3/Stripe/mail as optional services. | Medium — external service toggling. |
| **5. CI parity** | Run full suite against **both** drivers to prevent drift. | Low — guardrails. |

## 5. Key decisions to make
1. **Driver:** `modernc.org/sqlite` (recommended, pure Go) vs `mattn/go-sqlite3` (CGO). Affects desktop packaging.
2. **Triggers:** port to SQLite triggers vs move to app-level hooks (recommend app-level for `user_stats`, trigger for notification inserts).
3. **Timestamps:** standardize on **UTC TEXT** in SQLite; confirm the `AT TIME ZONE` code's user-local-day semantics are reproduced in Go.
4. **Search on desktop:** ship without Typesense (ILIKE fallback), or bundle a local Typesense. Recommend ILIKE initially.
5. **Optional services:** S3 (files), Stripe, mail, Typesense, CalDAV/JMAP/IMAP sync — these need feature-flag toggles for an offline desktop build. The `bootstrap`/`config` layer is the place.

## 6. What I would *not* do
- Rewrite all 835 query sites by hand (the shim removes that need).
- Adopt a full query builder (sqlc/squirrel/ent) as a prerequisite — it's a nice-to-have later, not a blocker now.
- Fork the codebase into separate PG/SQLite repos — the `Database` interface already invites one codebase.

## 7. Bottom line
The conversion is **economically justified** precisely because the codebase already centralizes DB access behind `models.Database` and already treats search/Typesense as optional. The cost is concentrated in (a) one shim, (b) a schema port with ~7 hand-finished trigger migrations, and (c) ~8 files of PG-specific query idioms. Everything else is either already ANSI-SQL compatible (`RETURNING`, `ON CONFLICT`, `JSONB`-as-text) or automatable (`$N`→`?`, `NOW()`→`CURRENT_TIMESTAMP`). Supporting both backends from one Go binary is realistic in a few weeks of focused work, with a low-risk spike possible in days.
