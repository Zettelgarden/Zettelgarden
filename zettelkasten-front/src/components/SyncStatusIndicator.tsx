/**
 * SyncStatusIndicator: online/offline + pending-changes affordance for the
 * desktop app (epic Zettelgarden-v5b, Phase 2b — issue fv3), driven by the
 * sync engine's progress events. Renders nothing in the web app.
 */

import React from 'react';
import { useSync } from '../contexts/SyncContext';
import type { SyncProgress } from '@zettelgarden/sync-engine/types';

function statusStyle(state: SyncProgress['state']) {
  switch (state) {
    case 'syncing':
      return { dot: 'bg-blue-500 animate-pulse', label: 'Syncing…' };
    case 'offline':
      return { dot: 'bg-red-500', label: 'Offline' };
    case 'error':
      return { dot: 'bg-orange-500', label: 'Sync error' };
    default:
      return { dot: 'bg-green-500', label: 'Synced' };
  }
}

export function SyncStatusIndicator({
  collapsed = false,
}: {
  collapsed?: boolean;
}) {
  const { desktop, progress } = useSync();
  if (!desktop) return null;

  const { dot, label } = statusStyle(progress.state);
  const pending = progress.pendingChanges ?? 0;
  const title =
    pending > 0
      ? `${label} · ${pending} pending change${pending === 1 ? '' : 's'}`
      : label;

  if (collapsed) {
    return (
      <div
        className="flex items-center justify-center py-1"
        title={title}
        aria-label={title}
      >
        <span className={`w-2.5 h-2.5 rounded-full ${dot}`} />
        {pending > 0 && (
          <span className="ml-1 text-xs text-gray-600">{pending}</span>
        )}
      </div>
    );
  }

  return (
    <div
      className="flex items-center gap-2 px-3 py-1.5 text-xs text-gray-600"
      title={title}
      aria-label={title}
    >
      <span className={`w-2 h-2 rounded-full ${dot}`} />
      <span className="truncate">
        {label}
        {pending > 0 && ` · ${pending} pending`}
      </span>
    </div>
  );
}
