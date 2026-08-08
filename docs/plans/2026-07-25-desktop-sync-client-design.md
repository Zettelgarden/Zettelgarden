# Desktop Sync Client Design

**Date:** 2026-07-25
**Status:** Design Draft — Pending Decisions
**Author:** pi + User

## Overview

Today Zettelgarden ships as a hosted web app (React SPA talking to the Go REST
API) plus a thin Electron wrapper (`zettelkasten-front/electron/`) that is
effectively a branded browser window — it loads the SPA and the SPA still hits
the backend over HTTP on every interaction. There is no local database, no
offline behavior, and no data synchronization. Two other clients exist — the
`zg` Go CLI and the `zettelgarden-mcp` Python server — both also thin HTTP
clients.

This document proposes a **real desktop client with local data sync**, built on
Tauri, that keeps the Go backend as the canonical source of truth. The goal is a
client that works offline, opens instantly, and reconciles changes in the
background when network is available — without requiring a backend rewrite.

## Goals

- A desktop client that reads and writes locally with no perceived network latency.
- Graceful offline operation: read everything, write the common cases, sync later.
- Keep the existing React UI investment; swap the *data layer*, not the views.
- Keep the Go backend as canonical; only small, additive changes over time.
- Position the codebase to reach mobile (iOS/Android) later via the same stack.

## Non-Goals (for v1)

- Peer-to-peer or multi-device mesh sync. The server remains the single merge point.
- Conflict-resolution UI with manual merge. We accept last-write-wins on
  `updated_at` for v1 (see [Reality Check](#reality-check-leave-the-backend-the-same)).
- Offline authoring of server-derived data (entities, facts, embeddings,
  summaries). These are produced by the LLM/backend and are pull-only on the client.
- Replacing the hosted web app. The web build remains a first-class target.

## Current Architecture (Baseline)

```
React UI (zettelkasten-front/src/components, contexts, hooks)
        │  React Query (in-memory cache only; staleTime 5m, gcTime 10m)
        ▼
apiClient (src/api/client.ts) — fetch() + JWT from localStorage
        ▼
Go REST API (go-backend/) — Postgres+pgvector, Typesense, S3, Z.AI LLM
```

- **Auth**: JWT bearer token stored in `localStorage`.
- **Cache**: React Query, in-memory, not persisted.
- **Domain types**: cards, tasks, tags, entities, facts, files, references,
  templates, schemas, RSS, audit events, structured data — a wide surface.
- **Existing Electron**: window shell only (`electron/main.ts`,
  `electron-builder.json`). No sync, no local DB.

## What "Sync" Means — The Spectrum

"Sync" is overloaded. Effort scales roughly 10× between rungs:

1. **Persistent cache (offline read).** Local SQLite mirrors server data; UI
   reads locally; a background job refreshes. Writes require network. *Lowest
   effort, largest perceived win.*
2. **Offline write queue.** Reads from local DB; writes go to a local outbox and
   replay on reconnect; last-write-wins by `updated_at`. *Medium effort.*
3. **True bidirectional sync with conflict resolution.** Per-row version
   vectors/etags, tombstones, merge policy, conflict UI. *High effort; ongoing
   maintenance.*

**Recommendation: target rung 2.** Rung 1 ships fast and proves the stack; rung 2
satisfies the "works on a plane" requirement; rung 3 is rarely justified for a
single-user, single-source-of-truth product like Zettelgarden.

## Reality Check: "Leave the Backend the Same"

Rung 1 is achievable with **zero** backend changes. Rung 2 — offline writes —
will need **small, additive** backend work. The current API lacks the primitives
sync requires:

- **No incremental sync endpoint.** List endpoints return full sets; there is no
  `?since=<updated_at>` / changes feed / cursor. "Sync" today means re-pulling
  everything, which does not scale.
- **No per-row versioning.** Cards have `updated_at` but no `version`/`etag`/
  `revision`, so conflicts cannot be *detected* — only blind last-write-wins is
  possible. Many junction tables (tags, entity↔card, fact↔card) have no
  `updated_at` at all.
- **Mixed delete semantics.** Cards have `is_deleted` (soft — good), but many
  resources use hard `DELETE`, which is un-syncable without tombstones (the
  client cannot distinguish "deleted" from "never existed").
- **IDs.** Cards have a stable string `card_id`; most other entities use DB
  auto-increment integers, which are awkward for offline creation.

**Plan**: ship Phase 1 (cache) with no backend changes; budget small additive
endpoints for Phase 2 (offline writes). The backend stays structurally the same
throughout.

## Derived vs User-Authored Data (Critical Distinction)

Not every table syncs the same way:

| Category | Examples | Sync treatment |
|---|---|---|
| **User-authored** | cards, tasks, tags, references, structured data, templates | Bidirectional; offline-writable |
| **Server-derived** | entities, facts, summaries, embeddings, Typesense index | Pull-only; overwritten on refresh; recomputed server-side after a card mutates |
| **Blobs** | files in S3 | Record syncs like user-authored; binary cached separately on demand |

The local DB may *read* derived data but must never *write or merge* it. After a
card changes offline and syncs, the server re-runs entity/fact/embedding
processing; the client pulls the refreshed derived rows on the next sync cycle.

## Tauri vs Existing Electron

The repo already has a working Electron shell. The real question is whether to
keep it or rebuild the shell on Tauri, given we are rewriting the data layer
regardless.

| | Electron (current) | Tauri |
|---|---|---|
| Bundle size | ~150 MB+ | ~10–20 MB |
| Memory | Bundled Chromium (heavier) | Native webview (lighter; varies by OS) |
| Local DB | Node sidecar (`better-sqlite3`); JS end-to-end | Rust core with bundled SQLite (`tauri-plugin-sql` or sidecar) |
| Reuse React app | Already works | Also works — wraps the same Vite build |
| Sync engine language | TS/JS (hand-rolled, or adapt Replicache/PowerSync patterns) | Rust (fast; new language for the team) |
| Mobile later | Possible but painful | First-class iOS/Android target |

**Recommendation: Tauri.** The product already markets "no desktop app required
/ web-native / lean"; a small footprint reinforces the brand, and Tauri's mobile
story is a future option essentially for free. The cost is one Rust sync engine,
which is a bounded, well-defined component. If minimizing new technology is the
priority, staying on Electron with a TS sync engine against `better-sqlite3` is a
defensible alternative — the architecture below is transport-agnostic either way.

## Target Architecture

Keep the React UI. Replace what sits *under* it.

```
React UI (largely unchanged components, contexts, hooks)
        │  reads/writes via React Query
        ▼
Local Data Adapter  ◄── replaces apiClient for reads;
        │                mutations write local DB + outbox
        ▼
Local SQLite (Tauri / Electron)  ◄── the source of truth the UI sees
        │
   ├──► Sync Engine (background)
   │      ├─ pull: GET /changes?since=<cursor>  → upsert local rows
   │      ├─ push: drain outbox → POST/PUT/DELETE → reconcile
   │      └─ backoff, online/offline, progress events
   ▼
Go Backend (canonical; mostly as-is, small additive sync endpoints later)
```

Key properties:

- **React Query stays** — it now queries local SQLite (instant, offline) instead
  of HTTP. Existing query-key factories (`queryKeys`) are reused.
- **`apiClient` becomes one transport**, not *the* data layer. A new
  `localClient` serves reads; the sync engine is the only component issuing HTTP
  for writes.
- **Outbox table** captures every offline mutation; the engine drains it on
  reconnect with idempotency keys and reconciles server responses back into the
  local DB.
- **Derived data** is pulled and overwritten, never merged or written offline.
- **Per-user local isolation**: the local DB is scoped to the authenticated
  user; logout wipes it. Auth token stored in the OS keychain, not localStorage.

### Suggested Repository Layout

A sibling `desktop/` directory consuming the shared UI, so the web build stays
untainted:

```
zettelkasten-front/   # existing web SPA — unchanged
desktop/              # NEW — Tauri app
├── src-tauri/        # Rust core: SQLite, sync engine, IPC commands
│   ├── sync/         # pull/push/reconcile logic
│   ├── db/           # schema, migrations, repositories
│   └── outbox/       # offline write queue
├── src/              # thin TS layer: localClient, Tauri IPC glue
└── …                 # consumes zettelkasten-front/src via workspace alias
```

(If we keep Electron, the equivalent lives under `zettelkasten-front/electron/`
extended with a `better-sqlite3` sidecar and a TS sync engine — same shape.)

## Phased Implementation Plan

### Phase 0 — Scaffold (no sync)

- New `desktop/` Tauri app wrapping the existing Vite build.
- Keychain-based auth (token in OS keychain, not localStorage).
- App settings (server URL, account).
- CI produces a signed build per OS.
- **Exit criteria**: app launches, authenticates, and renders the existing UI
  hitting the backend directly (same as web, but packaged).

### Phase 1 — Read Cache (rung 1; zero backend changes)

- Local SQLite mirror of user-authored + derived tables.
- `localClient` replaces `apiClient` for reads; React Query reads locally.
- Background "full refresh" poll per collection (rate-limited).
- Online/offline indicator; instant open.
- **Exit criteria**: UI fully usable offline for reading; background refresh
  keeps data fresh within a configurable window.

### Phase 2 — Offline Writes (rung 2; small additive backend)

- Outbox table; mutations write locally + enqueue.
- Sync engine drains outbox on reconnect; last-write-wins on `updated_at`.
- Idempotency keys for safe retry.
- **Small backend additions** (additive, behind versioned routes):
  - `GET /api/changes?since=<timestamp>&collections=…` changes feed.
  - `version` / `updated_at` columns on every mutable table.
  - Tombstones (or universal soft-delete) for currently hard-deleted resources.
  - Stable client-generated IDs for offline-created records.
- **Exit criteria**: create/edit/delete cards and tasks offline; changes
  reconcile correctly on reconnect; no silent data loss.

### Phase 3 — Polish

- Selective sync (which collections to mirror; useful for large accounts).
- Blob caching for S3 files (download on access, optional pre-cache).
- Conflict/late-write UX (surfaces when last-write-wins discards a local edit).
- Delta sync backoff and bandwidth budgets.
- Multi-account local isolation and encrypted-at-rest local DB.

## Open Decisions (need input before Phase 0)

1. **Electron (keep) vs Tauri (rewrite shell)?** Recommendation: Tauri.
2. **Target rung?** Assumption: rung 2 (offline read + write). Confirm.
3. **Which entity types must be offline-mutable first?** Proposal: cards + tasks
   in Phase 2, then tags. Entities/facts stay server-derived (read-only
   offline). Confirm.
4. **Repository layout?** Recommendation: new sibling `desktop/` consuming the
   shared UI, leaving `zettelkasten-front` as the pure web build. Alternative:
   evolve `zettelkasten-front/electron/` in place.
5. **Backend budget for Phase 2?** The changes feed + version columns + soft
   deletes are the real gate on "sync," not the client. Confirm willingness to
   spend that budget when we reach Phase 2.

## Risks

- **Surface area.** The domain is wide; an offline mode that covers only some
  types feels broken. Phasing must be explicit and communicated in the UI
  ("available offline: cards, tasks").
- **Derived-data lag.** After an offline card edit syncs, entities/facts/embeddings
  refresh asynchronously server-side; the client may show stale derived data
  briefly. Acceptable, but should be invisible or surfaced lightly.
- **Backend drift.** If Phase 2 backend additions are deferred indefinitely, the
  client is stuck at rung 1. Phase 2 scope must be owned, not optional.
- **Webview variance (Tauri).** Native webviews differ across OS (WebKit on
  macOS, WebView2 on Windows, WebKitGTK on Linux); some React libs (e.g.
  `react-pdf`) may need validation per platform.
- **Local data security.** A local copy of all user data needs at-rest encryption
  and clean per-user isolation; getting this wrong is a privacy regression vs
  the current thin client.

## Next Steps

Once the five Open Decisions are settled:

1. File `bd` issues for Phase 0 and Phase 1.
2. Stand up the `desktop/` Tauri scaffold (Phase 0).
3. Prototype the SQLite schema mirror and `localClient` against the cards
   collection only, end-to-end, to validate the read path before broadening.
