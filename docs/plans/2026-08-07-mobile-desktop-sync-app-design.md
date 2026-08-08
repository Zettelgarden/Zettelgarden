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
| Stable string IDs | Cards only (`card_id`). Tasks/tags use int PKs | Offline-created records need a stable client identity for everything but cards |
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
        GET /api/sync/changes?since=<cursor>
        POST /api/sync/push        (batch, idempotent)
```

**Core pieces:**

- **`StorageAdapter` interface** — `get/put/delete/query` over a collection.
  Implementations: SQLite (desktop + mobile) and IndexedDB (a future web/PWA
  bonus, free to add later).
- **Local mirror** — the syncable tables copied into local SQLite, same schema
  dialect as the backend.
- **Outbox** — every local mutation writes the local row + a queued change
  (collection, row uuid, op, version, idempotency key) in one transaction.
- **Pull** — `GET /api/sync/changes?since=<cursor>` returns rows changed after
  the cursor; apply, advance cursor.
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
| Junction rows | card↔tag, card↔parent, task↔tag | Bidirectional, with tombstones |
| Server-derived (pull-only) | entities, facts, summaries, embeddings | Pull + overwrite; never merged |
| Blobs | file contents | Metadata syncs; binary cached on demand (LRU) |

**v1 offline-writable scope:** cards, tasks, tags, and their junction rows.
Everything else syncs read-only until proven necessary. The UI must say what's
offline-capable ("Available offline: cards, tasks, tags").

### Conflict Policy

- **v1: row-level last-write-wins**, ordered by `version` (monotonic per row,
  bumped by the server on every accepted write), then `updated_at`, then a
  deterministic tie-breaker (lexicographic device id). Because we add a real
  `version` counter, the second-precision timestamp problem disappears.
- **v2 (later):** field-level merge for card title/body, and a "discarded
  change" inbox so a losing edit isn't silently lost. Not needed for v1.

### Offline Search

- Local full-text search via SQLite FTS5 over the mirrored cards/tasks (rebuild
  on pull). Typesense/vector/semantic search remains an online-only action with
  a clear "requires connection" affordance.

## Backend Work Required (the Real Gate)

Small, additive, behind versioned routes. **Nothing existing changes.**

1. **`sync_log` table** — `(id PK AUTOINCREMENT, user_id, collection, row_uuid,
   op, version, created_at)`. Every server-side mutation writes a row in the
   same transaction. This is the changes feed; the cursor is the max `id` the
   client has seen.
2. **`GET /api/sync/changes?since=<cursor>&collections=…`** — per-user
   incremental feed, ordered by `sync_log.id`.
3. **`POST /api/sync/push`** — batch upsert/delete with idempotency keys
   (`row_uuid` + `version`), applied transactionally, returns accepted versions
   and conflicts.
4. **`version INTEGER` column** on every syncable table, bumped on update
   (alongside existing `updated_at`).
5. **Stable client identities** — add `sync_uuid TEXT UNIQUE` to tables whose
   PK is a server int (tasks, tags, …). Cards already have `card_id`; reuse it
   as the sync identity for cards. Offline-created rows ship a fresh UUID.
6. **Tombstones** — soft-delete (or tombstone rows) for currently hard-deleted
   resources/junctions so deletes propagate.
7. **Auth follow-ups (small)** — a refresh-token endpoint so native clients
   don't force a login every 15 days; OAuth/GitHub/OIDC deep-link callbacks for
   native shells.
8. **Tests** — Go tests for feed/push/idempotency under the existing
   `_test.go` pattern; fixtures in `go-backend/tests/`.

Estimated surface: one migration file, two new handlers, version bumps in the
existing write handlers. It is **mechanical, not architectural** — the backend
stays the canonical source of truth with the same read endpoints intact.

## Phased Implementation Plan

### Phase 0 — Backend Sync API (the critical path)
Everything depends on this. Build the `sync_log` feed, push endpoint, version
columns, sync UUIDs, and tombstones.
**Exit criteria:** a script can pull a changes feed, push a batch, and observe
the feed advance; idempotent retry produces no duplicates; Go tests green.

### Phase 1 — Shared Sync Engine (TS)
Storage adapters, local mirror, outbox, pull/push/reconcile, cursor, conflict
policy, unit tests against a SQLite adapter.
**Exit criteria:** a headless harness (no UI) keeps two local databases
converged through a live backend — including offline gaps and concurrent edits.

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
   team's language) vs Rust (background sync with the app closed on desktop).
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
