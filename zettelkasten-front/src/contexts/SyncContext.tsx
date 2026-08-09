/**
 * SyncContext: mounts the desktop sync client and exposes engine progress
 * (online/offline, pending changes) to the UI (epic Zettelgarden-v5b, Phase
 * 2b — issue fv3). In the web app this renders children untouched (thin
 * client — no engine).
 */

import React, { createContext, useContext, useEffect, useState } from 'react';
import type { SyncProgress } from '@zettelgarden/sync-engine/types';
import { getSyncClient, type SyncClient } from '../data/syncClient';
import { SyncDataProvider } from '../data/syncProvider';
import { registerSyncProvider } from '../data/provider';

export interface SyncContextValue {
  /** Present only in the desktop app. */
  client: SyncClient | null;
  progress: SyncProgress;
  /** Whether the app is running inside the Tauri desktop shell. */
  desktop: boolean;
  syncNow(): Promise<void>;
}

const initialProgress: SyncProgress = { state: 'idle', pendingChanges: 0 };

const SyncContext = createContext<SyncContextValue>({
  client: null,
  progress: initialProgress,
  desktop: false,
  syncNow: async () => undefined,
});

export const useSync = () => useContext(SyncContext);

export function SyncProvider({ children }: { children: React.ReactNode }) {
  const [client, setClient] = useState<SyncClient | null>(null);
  const [progress, setProgress] = useState<SyncProgress>(initialProgress);

  useEffect(() => {
    let cancelled = false;
    getSyncClient().then((c) => {
      if (cancelled || !c) return;
      // Make the local-first provider the app's data layer.
      registerSyncProvider(new SyncDataProvider(c.engine));
      setClient(c);
      c.subscribe(() => setProgress({ ...c.progress }));
      setProgress({ ...c.progress });
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const value: SyncContextValue = {
    client,
    progress,
    desktop: !!client,
    syncNow: async () => {
      await client?.syncNow();
    },
  };

  return <SyncContext.Provider value={value}>{children}</SyncContext.Provider>;
}
