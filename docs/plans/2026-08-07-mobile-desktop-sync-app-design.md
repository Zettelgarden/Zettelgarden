# Mobile + Desktop Sync App Design

**Date:** 2026-08-07
**Status:** Design Draft — Pending Decisions
**Author:** pi + User

## Overview

Zettelgarden ships today as a hosted web SPA. The frontend reads and writes the
Go backend directly on every interaction — there is no local data, no offline
mode, and no sync. This document plans the next step: **native desktop and
mobile apps with real local-first sync**, following the model popularized by
mindwtr-style apps (local SQLite as the source of truth the UI sees, the server
as a sync hub, offline writes queued and reconciled when online).

It extends the earlier desktop-only design
(`docs/plans/2026-07-25-desktop-sync-client-design.md`) with a **shared sync
layer that both desktop and mobile consume**, and makes the platform choices
concrete. The web app stays a thin client and is untouched.

## Why Sync Is the Hard Part (and Everything Else Is a Shell)

The current data flow is:

```
React UI (zettelkasten-front/src)             — components, contexts, hooks
        │  React Query (in-memory, stale 5m / gc 10m — nothing persisted)
        ▼
apiClient (src/api/client.ts)                 — fetch() + JWT from localStorage
        ▼
Go REST API (go-backend/)                     — ~60+ routes, SQLite, Typesense
```

"Native apps" are easy — a Tauri shell around the Vite build already exists in
Electron form. The real work is the **data layer underneath the UI**: a local
database, a change-capture mechanism, and a reconciliation loop. That layer is
transport-agnostic and should be written **once**, shared by desktop and mobile.

### What the Backend Has Today (verified)

| Primitive | Status | Implication for sync |
|---|---|---|
| Stable string IDs | None truly stable: `card_id` is user-editable + can be empty; tasks/tags use int PKs | Every syncable table needs an immutable `sync_uuid`; `id`/`card_id` sync as ordinary fields |
| `updated_at` | Most tables have it, but **second precision** (`datetime('now')`) | Two devices editing in the same second can tie; timestamps alone can't order changes |
| Soft delete | Cards, tasks, tags (`is_deleted`) | Good — syncable |
| Hard delete | Junction tables (`fact_card_junction`, `entity_card_junction`, `entity_fact_junction`) and several resources | Un-syncable without tombstones (client can't distinguish deleted from never-existed) |
| Incremental feed | **None** — list endpoints return full sets | No `?since=` cursor anywhere; sync today means re-pulling everything |
| Version/etag | None | Conflicts can't be detected, only blindly last-write-wins |
| Batch write API | None | Each mutation is a separate HTTP call; no transactional batch push |
| Auth | JWT, 15-day expiry, **no refresh token** | Native apps need keychain storage + a re-auth story every 15 days |
| Search | Server-side Typesense + FTS only | Offline search needs a local mirror (SQLite FTS5); vector search stays online-only |
| Files | Local disk storage (S3 retired) | Blob metadata syncs as rows; binary cached on demand |

The good news: **SQLite is already the backend database** (68 tables), so the
local client mirrors the same dialect — no Postgres/MySQL impedance mismatch.

## The mindwtr Idea, Applied

The pattern worth copying, in order of importance:

1. **Local-first reads.** The UI never talks to the network. It queries a local
   SQLite database and is instant, even on a plane.
2. **The server is a sync hub, not a read/write API.** Its job is to accept a
   batch of changes, persist them, and hand back a changes feed. This is a
   different shape than today's REST surface.
3. **Offline writes go to an outbox** and replay when online, with idempotency
   keys so retries are safe.
4. **Changes are versioned.** Each row carries a monotonic version; the client
   tracks a per-user cursor. Conflicts resolve deterministically (v1:
   last-write-wins with a tie-breaker; see [Conflict Policy](#conflict-policy)).
5. **Only user-authored data syncs both ways.** Server-derived data (entities,
   facts, summaries, embeddings) is pull-only and regenerated server-side after
   a card mutates.

## Platform Options

### Desktop

| | Tauri v2 (recommended) | Electron (current shell) |
|---|---|---|
| Bundle size | ~10–20 MB | ~150 MB |
| SQLite | Rust plugin / sidecar (`tauri-plugin-sql`) | Node sidecar (`better-sqlite3`) |
| Sync engine | TS in webview (team's existing skills) or Rust | TS only |
| Mobile later | Same codebase can target iOS/Android | Painful |
| Background sync | Rust task can run with UI closed; TS runs while app is open | Full process control |

**Recommendation: Tauri v2.** Small footprint matches the "web-native / lean"
brand, and the Rust core gives a future home for background sync. The sync
engine itself is written in TypeScript (see [Shared Sync Engine](#shared-sync-engine)),
so the Rust surface stays small (SQLite + keychain + window management).

### Mobile

| | A: RN shell + WebView (recommended v1) | B: RN native UI | C: PWA (what exists today) |
|---|---|---|---|
| Reuses existing React UI | Yes (100%) | No — full rewrite | Yes |
| Store distribution (App Store/Play) | Yes | Yes | No (installable via browser only) |
| Native feel | Webview; acceptable for a notes app | Best | Web |
| Offline + sync | Full (sync engine bridges to native SQLite) | Full | Full, but IndexedDB in a webview/browser |
| Effort | Medium (shell + bridge + store plumbing) | Very high | Low, but no store presence |
| Native APIs (files, keychain, notifications) | Via RN bridge | Native | Limited |

**Recommendation: Option A for v1** — a React Native shell that hosts the
existing responsive web build in a WebView, with the sync engine running inside
and bridging to **native SQLite** (`op-sqlite` or `react-native-quick-sqlite`)
plus keychain storage. This ships the full ZG experience (including the
AI/entity/RSS features already built) as a real store app with real offline
sync, without rebuilding the UI. If native UX later becomes a priority, the
sync engine, types, and domain logic are already shareable — only the views
would be rewritten (Option B).

Two mobile-specific components the design must not gloss over:
- **WebView → SQLite bridge.** `op-sqlite` runs on the RN JS thread, *not*
  inside the WebView. The sync engine inside the WebView needs a `postMessage`
  bridge (webview → RN native module → SQLite); the mobile `SQLiteAdapter` is
  this bridge, not a direct call.
- **Token injection.** The JWT must be injected into the WebView per navigation
  (and revoked on logout), not read from `localStorage` inside the webview.

> Note: the existing PWA manifest + responsive design work means a PWA is a
> legitimate zero-native-code fallback, but store distribution is the reason to
> prefer the RN shell.

## Shared Sync Engine

One engine, three adapters. Written in TypeScript (the team's language, and it
runs everywhere):

```
┌─ Desktop (Tauri) ─────────────┐   ┌─ Mobile (RN WebView) ────────────┐
│ React UI (existing Vite build)│   │ WebView: React UI (existing)     │
│        │                      │   │        │                         │
│  Sync Engine (TS)             │   │  Sync Engine (TS)                │
│        │                      │   │        │                         │
│  SQLiteAdapter                │   │  SQLiteAdapter (via RN bridge)   │
└────────┬──────────────────────┘   └────────┬─────────────────────────┘
         │                                  │
         └──────────────┬───────────────────┘
                        ▼
              Go Backend — NEW sync API (canonical)
        GET /api/sync/snapshot    (bootstrap: full state)
        GET /api/sync/changes?since=<cursor>
        POST /api/sync/push        (batch, idempotent, optimistic)
```

**Core pieces:**

- **`StorageAdapter` interface** — `get/put/delete/query` over a collection.
  Implementations: SQLite (desktop + mobile) and IndexedDB (a future web/PWA
  bonus, free to add later).
- **Local mirror** — the syncable tables copied into local SQLite, same schema
  dialect as the backend.
- **Outbox** — every local mutation writes the local row + a queued change
  (collection, `row_uuid`, op, `base_version`) in one transaction. `row_uuid`
  is the idempotency key (upsert-by-uuid); `base_version` is the concurrency
  check carried on push.
- **Bootstrap + Pull** — first sync pulls a snapshot
  (`GET /api/sync/snapshot`); thereafter `GET /api/sync/changes?since=<cursor>`
  returns rows changed after the cursor; apply, advance cursor.
- **Push** — drain the outbox via `POST /api/sync/push`; apply server
  responses; reconcile conflicts per policy.
- **Sync scheduler** — online/offline detection, exponential backoff, periodic
  pull, progress events (`lastSynced`, `pendingChanges`) surfaced in the UI.
- **Per-user isolation** — the local DB is scoped to one account; logout wipes
  it. Token lives in the OS keychain, never localStorage.

### Collections & Sync Treatment

| Category | Examples | Treatment |
|---|---|---|
| User-authored (offline-writable) | cards, tasks, tags, task statuses, templates, references, pins, starred searches, structured data, files metadata | Bidirectional |
| Junction rows | card↔tag, task↔tag | Bidirectional, with tombstones |
| Card parentage | `cards.parent_id` | **Server-authoritative** — derived from `card_id` on push; pull-only in v1 |
| Server-derived (pull-only) | entities, facts, summaries, embeddings | Pull + overwrite; never merged |
| Blobs | file contents | Metadata syncs; binary cached on demand (LRU) |

**v1 offline-writable scope:** cards, tasks, tags, and their junction rows.
Everything else syncs read-only until proven necessary. The UI must say what's
offline-capable ("Available offline: cards, tasks, tags").

### Conflict Policy

- **v1: optimistic concurrency + row-level last-write-wins.** The client sends
  the `base_version` it wrote from; the server bumps `version` on every
  accepted write. If the base is stale, the server applies LWW per the ordering
  below and returns the accepted result (or a conflict response for deletes) —
  the client re-pulls and reconciles per policy. Ordering is `version`
  (monotonic per row), then `updated_at`, then a deterministic tie-breaker
  (lexicographic device id). Because `version` is the primary ordering key, the
  second-precision timestamp problem disappears.
- **Lost-edit surfacing:** the v1 policy silently discards the losing edit on
  two-device concurrent edit. Acceptable for v1, but the client surfaces a
  count ("N edits discarded on other devices") in the UI.
- **v2 (later):** field-level merge for card title/body, and a "discarded
  change" inbox so a losing edit isn't silently lost. Not needed for v1.

### Offline Search

- Local full-text search via SQLite FTS5 over the mirrored cards/tasks (rebuild
  on pull). Typesense/vector/semantic search remains an online-only action with
  a clear "requires connection" affordance.

## Backend Work Required (the Real Gate)

Small, additive, behind versioned routes. **Nothing existing changes.** Phase 0
scope is pinned to exactly five tables — `cards`, `tasks`, `tags`, `card_tags`,
`task_tags` — plus tombstones for hard-deleted rows. Everything else
(`task_statuses`, templates, pins, …) stays read-only until proven necessary.

1. **`sync_log` table** — `(id PK AUTOINCREMENT, user_id, collection, row_uuid,
   op, version, created_at)`. Every server-side mutation writes a row in the
   **same transaction**. This is the changes feed; the cursor is the max `id`
   the client has seen. The log is **append-only**: it must never be pruned
   while any client cursor trails it (retention = keep rows ≥ N days past the
   oldest active cursor, else force that client to re-bootstrap via the
   snapshot endpoint, item 3).
2. **Write-path audit + centralized change capture.** The real cost is
   inserting transactional `sync_log` writes into **every existing write path**
   for the five pinned tables (several use `INSERT OR REPLACE` junctions that
   must also emit tombstones). Make this an explicit Phase 0 workstream and
   centralize capture in the **service layer** (`services/`) — a single
   `emitChange(userID, collection, rowUUID, op, version)` helper called from
   services — rather than sprinkling handler edits.
3. **`GET /api/sync/snapshot?collections=…` (bootstrap)** — serves full current
   state per collection. `sync_log` starts empty at migration time, so existing
   rows have no feed entries; a new device pulling `since=0` would otherwise
   get an empty database. Client bootstrap = snapshot, then incremental feed.
   This also solves reinstalls. (Rejected alternative: a one-time backfill
   writing `sync_log` rows for every pre-existing row.)
4. **`GET /api/sync/changes?since=<cursor>&collections=…`** — per-user
   incremental feed, ordered by `sync_log.id`. Cursor is per-user *max seen
   id*, not contiguous.
5. **`POST /api/sync/push`** — batch upsert/delete, applied transactionally.
   Each change carries `row_uuid` (the idempotency key — a retry replays the
   same op) and `base_version` (the concurrency check; see Conflict Policy).
   The handler **resolves `row_uuid → server int PK` inside the same
   transaction** (for offline-created rows and for junctions referencing them)
   and returns that mapping in the response so the client can rewrite local
   references.
6. **`version INTEGER` column** on all five pinned tables, bumped on every
   accepted update (alongside existing `updated_at`).
7. **Stable client identities — `sync_uuid TEXT UNIQUE` on *all* syncable
   tables, cards included.** Neither `id` (server-assigned; unknown until the
   first push) nor `card_id` (user-editable via `UpdateCard`, app-level
   uniqueness only, empty for unsorted cards) is safe as the sync identity.
   `id` and `card_id` sync as ordinary fields.
8. **Parent relationships are server-authoritative.** `UpdateCard` already
   derives `parent_id` from the `card_id` prefix (`DiscoverParentId`) and root
   cards self-parent. The client sends `card_id` and never `parent_id`; the
   server derives it on push. The card↔parent junction is **pull-only in v1** —
   no offline reparenting; offline-created cards stay roots until the server
   resolves them.
9. **Tombstones** — soft-delete (or tombstone rows) for the hard-deleted
   junction rows (`card_tags`, `task_tags`) so deletes propagate.
10. **Auth follow-ups** — refresh-token endpoint (lands in **Phase 0**: 15-day
    re-auth on mobile is a churn magnet); OAuth/GitHub/OIDC deep-link callbacks
    for native shells (the OIDC design's token-in-URL callback is awkward for
    native — the deep-link work is genuinely needed).
11. **Tests** — Go tests for snapshot/feed/push/idempotency/concurrency under
    the existing `_test.go` pattern; fixtures in `go-backend/tests/`.

This is a bounded but wide workstream: one migration file, three new handlers,
`version` + `sync_uuid` columns on five tables, and a capture helper woven
through the existing service-layer write paths. The backend stays the canonical
source of truth with the same read endpoints intact.

## Phased Implementation Plan

### Phase 0 — Backend Sync API (the critical path)
Everything depends on this. Workstreams: (a) write-path audit + centralized
`emitChange` capture in `services/` for the five pinned tables; (b) `sync_log`
+ snapshot + changes feed + push endpoints (with `row_uuid → PK` resolution
and optimistic concurrency); (c) `version` + `sync_uuid` columns and junction
tombstones; (d) server-authoritative `parent_id` derivation on push; (e)
refresh-token endpoint + native auth deep links.
**Exit criteria:** a script can (1) snapshot-bootstrap an empty client, (2)
pull the feed, (3) push a batch including an offline-created card, a tag, and a
junction linking them, and observe the feed advance with the returned
`row_uuid → PK` mapping applied; idempotent retry produces no duplicates; a
stale `base_version` push is rejected or LWW-resolved deterministically; Go
tests green.

### Phase 1 — Shared Sync Engine (TS)
Storage adapters, local mirror, outbox, pull/push/reconcile, cursor, conflict
policy, unit tests against a SQLite adapter.
**Exit criteria:** a headless harness (no UI) keeps two local databases
converged through a live backend — including offline gaps, concurrent edits,
**offline-created linked rows** (a card created offline on device A, a task
created offline on device B, and junctions tying them to shared tags), and
offline `card_id` rename on both devices (must not split one logical row).

### Phase 2 — Desktop App (Tauri)
Scaffold (keychain auth, server-URL setting), wire the engine, swap the UI's
data layer to the local client, offline read + write for cards/tasks/tags,
online/offline + pending-changes indicators, signed CI builds per OS.
**Exit criteria:** create/edit/delete cards and tasks with the network
disconnected; changes reconcile on reconnect; the app opens instantly offline.

### Phase 3 — Mobile App (React Native)
RN shell + WebView of the existing UI, native SQLite bridge, keychain, store
builds (iOS TestFlight + Play internal), same engine + same exit criteria as
Phase 2.

### Phase 4 — Polish
Offline search (FTS5), blob caching, selective sync (large accounts), conflict
inbox UX, refresh tokens, optional web/IndexedDB adapter so the hosted web app
gains offline reads.

## Open Decisions (before Phase 0)

1. **Sync rung** — confirm **rung 2** (offline read + write, LWW) as the
   target. Rung 3 (field-level merge + conflict UI) is deferred to v2.
2. **Desktop** — **Tauri** (recommended) vs keeping Electron.
3. **Mobile** — **RN shell + WebView** (recommended v1) vs RN native UI vs PWA.
4. **Sync engine language** — **TypeScript** (recommended; runs in every shell,
   team's language). Explicit tradeoff: a TS engine syncs only while the app is
   open (on launch, on focus, on a timer). Rust buys background sync with the
   app closed on desktop at the cost of a second engine language. Default: TS;
   revisit only if "sync while app closed" becomes a requirement.
5. **Offline-writable scope** — confirm **cards, tasks, tags + junctions** for
   v1.
6. **Backend budget** — the Phase 0 items are the gate on "sync". Confirm
   willingness to spend it; otherwise the whole effort stalls at a read cache.
7. **Web app** — stays thin-client (recommended) vs gains the IndexedDB adapter
   later.

## Risks

- **Phase 0 is load-bearing.** If the sync API is deferred, clients are stuck
  at offline-read-only. It must be owned, not optional.
- **Second-precision timestamps.** Mitigated by the new `version` counter;
  do not rely on `updated_at` alone for ordering.
- **Surface area.** The domain is wide; an offline mode covering only some
  types feels broken. Scope must be explicit in the UI ("Available offline:
  cards, tasks, tags").
- **Derived-data lag.** After an offline card edit syncs, entities/facts/
  embeddings regenerate server-side; the client shows stale derived data
  briefly. Acceptable; surface lightly.
- **WebView variance (Tauri + RN).** Native webviews differ per OS; some React
  libs need per-platform validation (already flagged for `react-pdf` in the
  earlier design).
- **Local data security.** A full local copy of user data needs at-rest
  encryption and clean per-user isolation; getting this wrong is a privacy
  regression vs today's thin client.
- **JWT 15-day expiry.** Without refresh tokens, native users re-auth
  fortnightly; budget the refresh-token work in Phase 0 or 4.

## Next Steps

1. Settle the seven Open Decisions.
2. File `bd` issues for Phase 0 (backend sync API).
3. Stand up the Phase 0 migration + handlers; write the feed/push tests.
4. Prototype Phase 1 engine against the cards collection end-to-end before
   broadening to tasks/tags.

## Review Feedback (2026-08-08)

Reviewed against the current codebase. Factual claims verified: 15-day JWT with
no refresh (`handlers/auth.go`), second-precision `datetime('now')`, hard-deleted
junctions, int PKs for tasks/tags, 68 tables, local-disk file storage. The
following are **must-fix items for Phase 0**, plus smaller notes.

### Resolutions (2026-08-08)

Each item below is now incorporated into the design above:

| # | Feedback | Resolved where |
|---|---|---|
| 1 | Bootstrap/backfill unspecified | Backend Work §3 — snapshot endpoint; client bootstrap = snapshot then incremental |
| 2 | `id` + `card_id` both unsafe as sync identity | Backend Work §7 — immutable `sync_uuid` on all syncable tables, cards included |
| 3 | Offline-created row references unresolved | Backend Work §5 — push resolves `row_uuid → PK` in-transaction and returns the mapping |
| 4 | Push must be optimistic concurrency | Conflict Policy — `base_version` on push; LWW ordering; lost-edit count surfaced in UI |
| 5 | `parent_id` derivable from `card_id` | Backend Work §8 — server-authoritative parentage; junction pull-only in v1 |
| 6 | "Mechanical" undersells Phase 0 | Backend Work §2 + Phase 0(a) — write-path audit, centralized `emitChange` in `services/` |
| 7 | Pin the table list | Backend Work intro — exactly `cards`, `tasks`, `tags`, `card_tags`, `task_tags` |
| 8 | WebView→SQLite bridge is real | Mobile section — `postMessage` bridge + token injection |
| 9 | Background sync tradeoff | Open Decision 4 — TS syncs while app open; Rust = background |
| 10 | `sync_log` lifecycle / idempotency | Backend Work §1 + §5 — append-only, retention, cursor = max seen id, idempotency key = `row_uuid` |
| 11 | Auth follow-ups | Backend Work §10 — refresh tokens land in Phase 0; deep links needed |
| 12 | Harness must cover junctions | Phase 1 exit criteria — offline-created linked rows + concurrent `card_id` rename |

Original review text preserved below and in git history (394f443b).

### Critical

1. **Initial bootstrap / backfill is unspecified — this is the biggest gap.**
   `sync_log` starts empty at migration time, so **existing rows have no feed
   entries**. A new device (or reinstall) pulling `since=0` gets an empty
   database. The design needs either (a) a snapshot endpoint that serves full
   current state per collection (cursor "0" = snapshot, then incremental), or
   (b) a one-time backfill that writes `sync_log` rows for all pre-existing
   rows, keyed by a "legacy" version. (a) is simpler and also solves
   re-installs; recommend it.

2. **Neither `card_id` nor `id` is safe as the sync identity for cards.**
   Cards have **both** columns (`id INTEGER PRIMARY KEY AUTOINCREMENT` and
   `card_id TEXT`), and both fail:
   - `id` is **server-assigned**: an offline-created card has no `id` until
     the first push assigns one — the same problem tasks/tags have, which is
     why the doc already plans `sync_uuid` for them. Cards are no different.
   - `card_id` is **user-editable**: `UpdateCard` in `services/cards.go`
     renames it (app-level uniqueness check only, no DB constraint), and it
     can be **empty** ("unsorted cards", `card_id = ''`). Renaming a card on
     device A while offline while editing the same card on device B would
     split one logical row into two identities.
   Use an immutable `sync_uuid` for **all** syncable tables, cards included.
   `id` and `card_id` sync as ordinary fields. Add an exit-criteria test for
   "rename card_id while offline on two devices" to prove this. Note
   `parent_id` references `id` (int), so card parentage needs the same
   `row_uuid → server id` resolution as the other junctions (see item 3).

3. **Offline-created row references are unresolved.** Junctions
   (`card_tags`, `task_tags`) and `cards.parent_id` reference **server int
   PKs**. Two offline-created rows linked to each other (and every junction
   pointing at them) can't be written until the server assigns PKs. The push
   endpoint must resolve `row_uuid → server id` inside the same transaction,
   and the push response must return that mapping so the client can rewrite
   local references. Say this explicitly; it's the subtle part of the batch
   push.

4. **Push must be optimistic concurrency, or the conflict policy is dead
   code.** The doc's ordering is "version → updated_at → device id", but if
   the server blindly bumps `version` on every accepted push, then order is
   server arrival order and the tie-breakers never fire. Specify: client sends
   its `base_version`; server rejects (conflict response) or applies LWW when
   the base is stale; the client re-pulls and reconciles per policy. Also note
   the v1 policy silently discards the losing edit on two-device concurrent
   edit — acceptable for v1, but surface a count in the UI.

5. **`parent_id` is derivable from `card_id` and must not fight the server.**
   Verified: root cards self-parent (`parent_id = id`), and `UpdateCard`
   recomputes parentage from the card_id prefix on every rename. Syncing
   `parent_id` bidirectionally invites conflicts with server-side
   recomputation. Recommend: server-authoritative parent relationships
   (client sends `card_id`; server derives `parent_id`), or treat the
   card↔parent junction as pull-only for v1.

### Medium

6. **"Mechanical, not architectural" undersells Phase 0.** The real cost is
   inserting transactional `sync_log` writes into **every existing write path**
   (~60+ routes), several of which use `INSERT OR REPLACE` junctions that must
   also emit tombstones. Audit the write paths as an explicit Phase 0
   workstream, and centralize change capture in the service layer
   (`services/`) rather than sprinkling handler edits.

7. **Pin the Phase 0 table list.** "Every syncable table" invites scope creep.
   v1 scope is exactly: `cards`, `tasks`, `tags`, `card_tags`, `task_tags`
   (+ tombstones for the junctions and hard-deleted rows). `task_statuses`,
   templates, pins, etc. stay read-only until proven necessary.

8. **Mobile: the WebView→SQLite bridge is a real component.** `op-sqlite`
   runs on the RN JS thread, *not* inside the WebView. The sync engine running
   in the WebView needs a `postMessage` bridge (webview → RN native module →
   SQLite). One line in the design so the adapter isn't assumed to "just
   work". Token injection into the WebView per navigation also needs a story.

9. **Desktop background sync tradeoff.** The recommended TS-in-WebView engine
   cannot sync with the UI closed. Open Decision 4 should state this explicitly
   (TS = sync while app open; Rust = background sync), not just list Rust's
   capability in a table.

10. **`sync_log` lifecycle.** It must be append-only and never pruned while any
    client cursor trails (or define a retention policy + force-rebootstrap).
    Cursor is per-user *max seen id*, not contiguous. Minor: idempotency key is
    really `row_uuid` (upsert-by-uuid); `version` is the concurrency check, not
    part of the idempotency key — a retry replays the same op, not the same
    version.

11. **Auth follow-ups are real, and already partly designed.** The OIDC design
    (`2026-08-03-oidc-authentication-design.md`) uses a token-in-URL callback
    — awkward for native shells; the deep-link work is genuinely needed.
    Decide now whether refresh tokens land in Phase 0 or Phase 4 (the doc lists
    both); recommend Phase 0 since 15-day re-auth on mobile is a churn
    magnet.

### Phase 1 harness suggestion

12. The two-DB convergence harness should include junction + offline-created-
    linked-row cases (item 3), not just scalar card/task edits — that's where
    the engine will actually break.

