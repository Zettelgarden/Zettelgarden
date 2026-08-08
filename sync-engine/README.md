# @zettelgarden/sync-engine

Local-first sync engine for Zettelgarden desktop + mobile clients (epic
Zettelgarden-v5b, Phase 1a). It mirrors the server's `cards`, `tasks`, and
`tags` tables into a local SQLite database with an outbox of pending writes,
and syncs via the server's snapshot/changes/push endpoints
(optimistic-concurrency, last-write-wins). `card_tags`/`task_tags` junctions
are server-derived and pull-only.

The live spec is
[`docs/plans/2026-08-07-mobile-desktop-sync-app-design.md`](../docs/plans/2026-08-07-mobile-desktop-sync-app-design.md).
The Tauri v2 desktop shell (`desktop/`) wires this engine in during Phase 2b.

## Layout

```
src/
├── engine.ts     # sync engine: pull/push loops, cursor, merge, retries
├── http.ts       # server sync API client (snapshot/changes/push)
├── sqlite.ts     # SQLite adapter (better-sqlite3)
├── storage.ts    # outbox + local mirror storage
└── types.ts      # shared sync types
```

## Commands

```bash
npm run build       # tsc → dist/
npm test            # vitest run
npm run test:watch  # vitest watch
```
