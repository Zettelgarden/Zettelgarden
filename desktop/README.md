# Zettelgarden Desktop (Tauri v2)

Native desktop shell for Zettelgarden (epic Zettelgarden-v5b, Phase 2a — issue
c5b). Wraps the existing React web app (`zettelkasten-front`) in a Tauri v2
window. Today it packages the web app as-is; the local-first sync engine
(`sync-engine/`) gets wired in during Phase 2b.

## Layout

```
desktop/
├── package.json          # npm shell: tauri CLI scripts
├── src-tauri/
│   ├── Cargo.toml        # Rust: tauri 2, tauri-plugin-store, keyring
│   ├── tauri.conf.json   # points frontendDist at zettelkasten-front/dist
│   ├── preload.js        # localStorage->keychain shim + window.zgDesktop API
│   ├── capabilities/     # IPC permissions (window controls, store)
│   └── src/
│       ├── lib.rs        # builder: window + init script + commands
│       ├── main.rs
│       └── commands.rs   # keychain get/set/delete, settings, window controls
```

## What the scaffold does

- **Keychain auth** — the web app's `localStorage` `token`/`username` keys are
  intercepted by `preload.js` and stored in the OS keychain (macOS Keychain /
  Windows Credential Manager / Linux Secret Service via the `keyring` crate).
  The web app is untouched.
- **Settings** — server URL + account persisted to the app data dir
  (`settings.json` via `tauri-plugin-store` + a `save_settings` command).
- **Shell API** — `window.zgDesktop` exposes token access, settings, window
  controls, and (stubbed) sync status for Phase 2b's offline/pending-change
  indicators.
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
# Frontend dist first
cd zettelkasten-front && npm ci && npm run build
# Then the desktop bundle
cd desktop && npm run build
```

> Note: on this dev machine `/usr/bin/node` is `nsolid`, which crashes the
> `@tauri-apps/cli` native addon. `cargo build`/`cargo test` work fine; run the
> CLI from a standard Node (CI does).

## Verified

- `cargo build` (dev + release) clean; `cargo test` green (settings round-trip).
- App launches under `xvfb` with the embedded `dist`, event loop stays alive,
  no panics.
- Release binary embeds the web app; keyring/signing are exercised in the
  per-platform CI builds.
