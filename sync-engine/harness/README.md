# Phase 1b: headless two-DB convergence harness (Zettelgarden-xre)

Validates the shared TS sync engine (Phase 1a, `nqx`) **through a live Go
backend** — no UI, no mocks on the server side. Two real SQLite-backed devices
share one account; every scenario must converge all three states (device A
mirror == device B mirror == server snapshot) or the harness exits non-zero.

This is the Phase 1 exit criterion of
[`docs/plans/2026-08-07-mobile-desktop-sync-app-design.md`](../../docs/plans/2026-08-07-mobile-desktop-sync-app-design.md)
and is CI-runnable headlessly.

## Running

```bash
cd sync-engine
npm run harness        # builds the Go backend (cached) + boots it + runs all scenarios
```

Requirements: Go toolchain, Node, and the repo's `go-backend/` (the harness
builds it with `go build` into `harness/.cache/`, then spawns it against a
throwaway SQLite file in the OS temp dir; Typesense/SMTP/LLM are all optional
for this boot and degrade gracefully).

Exit code is non-zero on any divergence, so `npm run harness` works directly
as a CI step (sequential single-process vitest run — one backend for the whole
run).

## Architecture

```
harness/
├── convergence.test.ts        # boots ONE live backend, runs all scenarios
├── lib/
│   ├── backend.ts             # build+spawn Go backend, wait-ready, register/login, cleanup
│   ├── device.ts              # device = SqliteStorageAdapter + SyncEngine + HttpTransport
│   └── verify.ts              # convergence assertion (uuid set + version + data equality)
└── scenarios/
    ├── context.ts             # fresh account per scenario, settle() syncs, convergeAndAssert()
    ├── 01-offline-gap.ts      # A and B edit offline, both reconnect, all changes land
    ├── 02-concurrent-lww.ts   # same row edited offline on both → LWW winner, lost edit reported
    ├── 03-linked-rows.ts      # offline card (A) + task via card_pk_uuid (B) + same-named tag on both
    ├── 04-card-id-rename.ts   # offline card_id rename on both devices → no row split
    ├── 05-self-echo.ts        # interleaved push/pull: own echoed changes not double-applied
    ├── 06-tag-rename.ts       # rename-vs-rename (LWW, one tag) and rename-vs-create
    ├── 07-offline-delete.ts   # delete vs concurrent edit: no ghost row on the losing device
    ├── 08-delete-propagates.ts # plain offline delete reaches the other device via the feed
    └── 09-feed-pagination.ts  # 510-row push + paginated pull (>500 feed page size)
```

Each scenario gets a **fresh account** (new users are seeded with a welcome
card + 5 default tags, which the scenarios account for) and two devices
(`dev-a`/`dev-b`) with separate SQLite files and device ids. Mutations happen
offline (`setOnline(false)`), then both devices `sync()` (pull-then-push) for
several rounds until settled, and the harness asserts:

1. every device's outbox is drained (`pendingChanges() == 0`);
2. the row **uuid sets** match across server and both devices, per collection;
3. per-row **version and data are byte-identical** (semantic equality: key
   order/number formatting ignored) across server and both devices.

## Bugs this harness has caught (all fixed)

- **Transport snake_case parsing** (`src/http.ts`): the Go backend emits
  `row_uuid`/`server_id`/`server_version`/`mapped_to_row_uuid`/`lost_edits`,
  but the engine read camelCase — pushes applied server-side while the client
  never drained its outbox or adopted versions. The transport now normalizes
  the wire shape to the engine contract (and stays compatible with the
  camelCase mock used in unit tests).
- **`mappedToRowUuid` case typo** (`src/http.ts`): the merged-tag response was
  normalized with `mappedToRowUUID` while the engine reads `mappedToRowUuid`;
  tag name-merges never rewrote the losing device's local row (survived only
  because the merged row also arrived via the feed pull).
- **Global `sync_uuid` unique index** (`go-backend` schema + boot self-heal):
  uniqueness was global, not per-user — a create whose uuid collided with
  another account was **silently ignored**, leaving devices diverged. Now
  `(user_id, sync_uuid)`; existing DBs carrying the old global index are
  rebuilt per-user at boot. Regression tests:
  `TestSyncSelfHealPerUserSyncUUIDIndex` (server) and
  `TestSyncPushPerUserSyncUUIDSharing` (handlers).
- **`is_deleted` 0/1 vs boolean** (`src/engine.ts`): the Go backend emits
  SQLite BOOLEANs as `0`/`1` in row payloads; the engine compared `!== true`
  and never matched. When a DELETE won an LWW conflict, the losing device
  kept a permanent ghost row (the feed tombstone had been skipped while the
  edit was pending). Scenario 07 reproduces it; the engine now treats
  `is_deleted` truthiness and drops the mirror row when the adopted server
  data says deleted. Found in the independent subagent review.

## Known policy gap (follow-up)

Tag **rename-vs-create** (A renames `work`→`tasks` offline while B creates a
fresh `work` offline) converges deterministically today — one live `tasks` tag
(A's row) and one live `work` tag (B's fresh row), identical on every device
and the server. The v1 policy refinement from the design's third-pass review
("the renamed-away name's row soft-deleted", so B's create would resurrect a
tombstone instead of creating fresh) is **not implemented**; observable
convergence is identical either way. Filed as a follow-up bead
(see `Zettelgarden-8g0`).
