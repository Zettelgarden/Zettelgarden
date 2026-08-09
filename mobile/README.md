# Zettelgarden Mobile

React Native WebView shell for Zettelgarden (epic `Zettelgarden-v5b`, Phase 3a —
issue `Zettelgarden-c6l`). Hosts the existing `zettelkasten-front` Vite build —
the same responsive web UI the desktop Tauri shell (`desktop/`) wraps — inside a
`react-native-webview`, with the local-first sync engine running inside the
WebView and bridging to native SQLite (op-sqlite) and the OS keychain.

Design: `docs/plans/2026-08-07-mobile-desktop-sync-app-design.md` (Phase 3).

## Status (c6l.1 scaffold)

- RN 0.86.2 (new architecture), `react-native-webview`, app id
  `com.zettelgarden.mobile`, minSdk 24 / target+compileSdk 36.
- `App.tsx` renders a WebView; dev default points at the Android-emulator host
  (`http://10.0.2.2:5173`) — real server-URL config lands with c6l.4.
- Builds an installable debug APK: `npm run build:android:debug`.
- Android toolchain lives outside the repo: JDK 21 (Temurin) at
  `~/tools/jdk-21.0.12+8`, Android SDK at `~/Android/Sdk` (env in `~/.bashrc`).

## Commands

```bash
npm install
npm run start          # Metro dev server
npm run android        # install + launch on connected device/emulator
npm run ios            # iOS (requires macOS)
npm run build:android:debug   # ./gradlew assembleDebug
npm run build:android         # ./gradlew assembleRelease
npm run typecheck      # tsc --noEmit
npm run lint           # eslint
npm run test           # jest
npm run format:check   # prettier
```

## Phases

- c6l.1 — RN shell scaffold (this milestone): WebView shell, Android build, CI.
- c6l.2 — WebView↔RN postMessage bridge + `MobileStorageAdapter` (op-sqlite).
- c6l.3 — keychain JWT + token injection/revocation (webview shim).
- c6l.4 — shell detection (`isNativeShell`) + server-URL config.
- c6l.5 — simulator verification + desktop↔mobile convergence smoke.

iOS run-verification is deliberately deferred (Linux dev box cannot run iOS
simulators); iOS compiles via CI (`mobile.yml`, macos-latest) and run-verify
lands with Phase 3b (store builds).
