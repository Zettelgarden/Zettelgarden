/**
 * Sync client: the native shell's single SyncEngine instance and its
 * online/offline + progress wiring (epic Zettelgarden-v5b, Phase 2b — fv3;
 * Phase 3a — c6l.4).
 *
 * Initialized once when the app boots in a native shell (SyncProvider). Reads
 * and writes for cards/tasks/tags go through the engine's local mirror +
 * outbox; the server is reached only by the engine's HttpTransport.
 */

import { SyncEngine } from '@zettelgarden/sync-engine/engine';
import { HttpTransport } from '@zettelgarden/sync-engine/http';
import type { SyncProgress } from '@zettelgarden/sync-engine/types';
import {
  TauriStorageAdapter,
  tauriInvoke,
  isDesktopApp,
  isMobileApp,
  isNativeShell,
} from './tauriStorageAdapter';

export interface SyncClient {
  engine: SyncEngine;
  /** Last engine progress (state, pendingChanges, lastSynced, lastError). */
  progress: SyncProgress;
  /** True once bootstrap has completed at least once (or a cursor exists). */
  ready: boolean;
  /** Forces a sync cycle (e.g. a manual "Sync now" affordance). */
  syncNow(): Promise<void>;
  subscribe(cb: () => void): () => void;
}

/**
 * Reads the shell's non-secret settings (server URL) — Tauri invoke on
 * desktop, zgMobile bridge on the RN webview (c6l.4). Returns undefined when
 * no shell is present or the shell has no settings.
 */
async function loadShellSettings(): Promise<
  { serverUrl?: string } | undefined
> {
  if (isDesktopApp()) {
    try {
      return await tauriInvoke<{ serverUrl?: string }>('load_settings');
    } catch {
      // Settings unavailable (e.g. missing zgDesktop) — fall through.
    }
  }
  if (isMobileApp()) {
    try {
      return await (window as any).zgMobile?.loadSettings();
    } catch {
      // Mobile bridge not ready — fall through.
    }
  }
  return undefined;
}

/**
 * Normalizes a configured server URL to the server ROOT: strips a trailing
 * /api (web REST convention) and any trailing slash so the engine's
 * /api/sync/* paths resolve exactly once.
 */
export function normalizeServerUrl(base: string): string {
  return base.replace(/\/?api\/?$/, '').replace(/\/$/, '');
}

/** Resolves the API base URL for the SYNC transport (server ROOT — the engine
 * appends /api/sync/... itself; VITE_URL / settings may include the /api
 * suffix used by the web REST client). Never falls back to the webview
 * origin — in the bundled app that is tauri://localhost (or a file://
 * origin on mobile), which fetch() cannot reach. */
export async function resolveBaseUrl(): Promise<string> {
  const settings = await loadShellSettings();
  const base: string | undefined =
    settings?.serverUrl ||
    ((import.meta as any).env?.VITE_URL as string | undefined);
  if (!base) {
    throw new Error(
      'no server URL configured: set VITE_URL at build time or configure the server in the app settings',
    );
  }
  return normalizeServerUrl(base);
}

/**
 * Selects the storage adapter for the active shell. Desktop uses the Tauri
 * IPC adapter; the mobile adapter (over the RN postMessage bridge to
 * op-sqlite) lands in c6l.2 — until then the mobile webview stays a thin
 * client (getSyncClient returns null, web behavior).
 */
function buildStorageAdapter(): TauriStorageAdapter | null {
  if (isDesktopApp()) return new TauriStorageAdapter();
  // TODO(c6l.2): return new MobileStorageAdapter() when zgMobile is present.
  return null;
}

let client: SyncClient | null = null;
let initPromise: Promise<SyncClient | null> | null = null;

async function buildClient(): Promise<SyncClient | null> {
  const storage = buildStorageAdapter();
  if (!storage) return null; // web thin client (or mobile until c6l.2 lands)
  await storage.whenReady();

  const baseUrl = await resolveBaseUrl();
  const transport = new HttpTransport({
    baseUrl,
    // The preload shim redirects the token key to the OS keychain.
    token: () => localStorage.getItem('token'),
  });

  const engine = new SyncEngine({ storage, transport });
  const progress: SyncProgress = { state: 'idle', pendingChanges: 0 };

  // No explicit bootstrap here: pull() snapshots automatically when the
  // cursor is 0, so a pre-login failure self-heals once the token is set and
  // the network is back. Existing mirrors open instantly offline (cursor > 0
  // -> incremental pulls only).
  const bootstrapped = true;

  const onProgress = (p: SyncProgress) => {
    progress.state = p.state;
    progress.lastSynced = p.lastSynced;
    progress.pendingChanges = p.pendingChanges;
    progress.lastError = p.lastError;
    for (const cb of listeners) cb();
  };
  const listeners = new Set<() => void>();
  engine.onProgress(onProgress);

  // 30s periodic sync; engine's exponential backoff throttles failures.
  engine.start(30_000);
  // Kick an immediate sync so the app is fresh as soon as it opens (the
  // engine shares in-flight syncs, so this is safe alongside the timer).
  void engine.sync().catch(() => undefined);

  // Online/offline: drive the engine from the browser's connectivity signal.
  const applyOnline = (online: boolean) => engine.setOnline(online);
  applyOnline(navigator.onLine);
  window.addEventListener('online', () => applyOnline(true));
  window.addEventListener('offline', () => applyOnline(false));

  const syncClient: SyncClient = {
    engine,
    progress,
    ready: bootstrapped,
    syncNow: async () => {
      await engine.sync();
    },
    subscribe(cb: () => void) {
      listeners.add(cb);
      return () => listeners.delete(cb);
    },
  };
  client = syncClient;
  return syncClient;
}

/**
 * Returns the native-shell sync client, initializing it on first call.
 * Returns null in the web app (thin client — apiClient stays the data layer)
 * and in the mobile webview until c6l.2 wires the bridge storage adapter.
 */
export function getSyncClient(): Promise<SyncClient | null> {
  if (!isNativeShell()) return Promise.resolve(null);
  if (client) return Promise.resolve(client);
  if (!initPromise) {
    initPromise = buildClient().catch((err) => {
      console.error('sync client init failed:', err);
      initPromise = null;
      return null;
    });
  }
  return initPromise;
}
