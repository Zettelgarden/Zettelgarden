/**
 * Sync client: the desktop app's single SyncEngine instance and its
 * online/offline + progress wiring (epic Zettelgarden-v5b, Phase 2b — fv3).
 *
 * Initialized once when the app boots in desktop mode (SyncProvider). Reads
 * and writes for cards/tasks/tags go through the engine's local mirror +
 * outbox; the server is reached only by the engine's HttpTransport.
 */

import { SyncEngine } from '@zettelgarden/sync-engine/engine';
import { HttpTransport } from '@zettelgarden/sync-engine/http';
import type { SyncProgress } from '@zettelgarden/sync-engine/types';
import { TauriStorageAdapter, tauriInvoke, isDesktopApp } from './tauriStorageAdapter';

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

/** Resolves the API base URL: desktop settings -> build-time VITE_URL. */
export async function resolveBaseUrl(): Promise<string> {
  try {
    const settings = await tauriInvoke<{ serverUrl?: string }>('load_settings');
    if (settings?.serverUrl) return settings.serverUrl;
  } catch {
    // Settings unavailable (e.g. missing zgDesktop) — fall through.
  }
  const viteUrl = (import.meta as any).env?.VITE_URL as string | undefined;
  if (viteUrl) return viteUrl;
  return window.location.origin;
}

let client: SyncClient | null = null;
let initPromise: Promise<SyncClient | null> | null = null;

async function buildClient(): Promise<SyncClient> {
  const storage = new TauriStorageAdapter();
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
 * Returns the desktop sync client, initializing it on first call. Returns
 * null in the web app (thin client — apiClient stays the data layer).
 */
export function getSyncClient(): Promise<SyncClient | null> {
  if (!isDesktopApp()) return Promise.resolve(null);
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
