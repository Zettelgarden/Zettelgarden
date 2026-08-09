# Zettelgarden Desktop (Tauri v2)

Native desktop shell for Zettelgarden (epic Zettelgarden-v5b, Phase 2a — issue
c5b; Phase 2b — issue fv3). Wraps the existing React web app
(`zettelkasten-front`) in a Tauri v2 window with the local-first sync engine
(`sync-engine/`) as the data layer: cards/tasks/tags read and write a local
SQLite mirror + outbox (offline-capable) and reconcile with the server on
reconnect. Everything else (entities, files, RSS, search, …) is available
online-only and degrades gracefully offline.

## Layout

```
desktop/
├── package.json          # npm shell: tauri CLI scripts
├── src-tauri/
│   ├── Cargo.toml        # Rust: tauri 2, tauri-plugin-store, keyring, rusqlite
│   ├── tauri.conf.json   # points frontendDist at zettelkasten-front/dist
│   ├── preload.js        # localStorage->keychain shim + window.zgDesktop API
│   ├── capabilities/     # IPC permissions (window controls, store)
│   └── src/
│       ├── lib.rs        # builder: window + init script + commands + SyncDb
│       ├── main.rs
│       ├── commands.rs   # keychain get/set/delete, settings, window controls
│       └── sync_db.rs    # local mirror SQLite (rusqlite): sync_exec/query/begin/commit/rollback
```

## What the shell does

- **Keychain auth** — the web app's `localStorage` `token`/`username` keys are
  intercepted by `preload.js` and stored in the OS keychain (macOS Keychain /
  Windows Credential Manager / Linux Secret Service via the `keyring` crate).
- **Settings** — server URL + account persisted to the app data dir
  (`settings.json` via `tauri-plugin-store` + a `save_settings` command). The
  sync transport uses the configured server URL (fallback: build-time
  `VITE_URL`).
- **Local mirror** — `sync_db.rs` opens `app_data_dir/sync/mirror.db` (WAL,
  single connection behind a mutex) and exposes the engine's storage surface
  as origin-gated commands. The mirror schema lives in the TS adapter
  (`zettelkasten-front/src/data/tauriStorageAdapter.ts`), which implements
  the engine's async `StorageAdapter` over those commands.
- **Data layer** — the web UI's card/task/tag reads and writes route through
  `src/data/provider.ts`: in the desktop app the `SyncDataProvider` serves
  the local mirror + outbox; online-only reads degrade to empty on network
  failure (`src/data/offline.ts`). React Query hooks and `queryKeys` are
  unchanged — they now query local SQLite (instant, offline).
- **Indicators** — `SyncContext` exposes engine progress; the
  `SyncStatusIndicator` (sidebar footer) shows online/offline + pending
  changes. Offline startup keeps the keychain session and restores the cached
  user profile (`AuthContext`).
- **CI** — `.github/workflows/desktop.yml` builds per OS; dispatch/tag builds
  bundle + sign (secrets commented in for macOS/Windows).

## Development

```bash
# Terminal 1: web app dev server (tauri.conf.json devUrl)
cd zettelkasten-front && npm run start

# Terminal 2: tauri dev (needs a display)
cd desktop && npm install && npm run dev
```

## Build

```bash
# Frontend dist first (sync-engine is imported by source — no separate build)
cd zettelkasten-front && npm ci && npm run build
# Then the desktop bundle
cd desktop && npm run build
```

> Note: on this dev machine `/usr/bin/node` is `nsolid`, which crashes the
> `@tauri-apps/cli` native addon. `cargo build`/`cargo test` work fine; run the
> CLI from a standard Node (CI does).

## Verified

- `cargo build` (dev + release) clean; `cargo test` green (settings round-trip
  + sync_db exec/query + transaction isolation + file-keychain round-trip).
- App launches under `xvfb` with the embedded `dist`, event loop stays alive,
  no panics.
- Release binary embeds the web app; keyring/signing are exercised in the
  per-platform CI builds.

## E2E smoke (Zettelgarden-77j)

`desktop/e2e/smoke.sh` runs the REAL release binary against a live Go backend
under `xvfb`, driving a scripted scenario inside the webview via an env-gated
bridge (`ZG_E2E=1`; the bridge is never injected in normal runs). It verifies:

1. **fresh** — register + real login form; fresh-mirror bootstrap through the
   real IPC (Rust sync_db + TauriStorageAdapter); online create+sync; then
   OFFLINE create/edit/delete of a card and create/edit of a task, asserting
   the sidebar indicator shows "Offline · N pending" and **zero `window.fetch`
   calls on the hot path**; reconnect + reconciliation (server spot-check);
   indicator returns to "Synced".
2. **session** — relaunch with the same app-data dir + keychain file: the app
   must boot authenticated from the keychain (no login form) and render the
   offline-created card from the local mirror instantly.

Run with `bash desktop/e2e/smoke.sh` (needs Go, Node, cargo, xvfb-run, and
python3 — the script dumps the mirror.db via python3's sqlite3; uses port
18131 and a temp run dir). The file keychain (`ZG_KEYCHAIN_FILE`) stands in
for the OS Secret Service daemon, which needs a desktop session.
