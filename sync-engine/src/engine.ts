/**
 * SyncEngine: the local-first core (epic Zettelgarden-v5b, Phase 1a).
 *
 * Responsibilities:
 *  - Local-first API: mutate()/deleteLocal() write the mirror + outbox
 *    atomically; the UI reads through the engine and never hits the network.
 *  - bootstrap(): snapshot the server, set the cursor.
 *  - sync(): pull (apply remote changes, never clobbering rows with pending
 *    local edits), then push (drain the outbox, apply server mappings).
 *  - Reconcile: applied → adopt server version/id; conflict → adopt server
 *    row (LWW); merged (tags) → rewrite the local row to the surviving uuid.
 *  - Self-echo: a pushed change echoes back on the next pull; applying it is
 *    idempotent (same uuid+version+data), and rows with pending outbox edits
 *    are never clobbered — so own writes are never double-applied or lost.
 *  - Scheduler: online/offline flag, exponential backoff, periodic sync,
 *    progress events surfaced in the UI (lastSynced, pendingChanges).
 */

import type {
  ChangesResponse,
  Collection,
  MirrorRow,
  OutboxEntry,
  SyncProgress,
  SyncSummary,
  SyncTransport,
} from './types';
import { COLLECTIONS } from './types';
import type { StorageAdapter } from './storage';

/** Fallback collection inference for feeds that omit the field (defensive). */
function inferCollection(data?: Record<string, unknown>): Collection | undefined {
  if (!data) return undefined;
  if ('card_pk' in data || 'is_complete' in data) return 'tasks';
  if ('body' in data || 'card_id' in data) return 'cards';
  if ('color' in data || ('name' in data && !('title' in data))) return 'tags';
  return undefined;
}

export interface SyncEngineOptions {
  storage: StorageAdapter;
  transport: SyncTransport;
  deviceId?: string;
  /** Max backoff between failed auto-syncs (default 5 min). */
  maxBackoffMs?: number;
}

export class SyncEngine {
  private storage: StorageAdapter;
  private transport: SyncTransport;
  readonly deviceId: string;
  private maxBackoffMs: number;
  private intervalMs = 0;

  private listeners = new Set<(p: SyncProgress) => void>();
  private lastSynced?: Date;
  private lastError?: string;
  private online = true;
  private backoffMs = 1000;
  private timer: ReturnType<typeof setTimeout> | undefined;
  private syncing = false;
  private currentSync: Promise<SyncSummary> | null = null;

  constructor(opts: SyncEngineOptions) {
    this.storage = opts.storage;
    this.transport = opts.transport;
    this.maxBackoffMs = opts.maxBackoffMs ?? 5 * 60 * 1000;
    this.deviceId =
      opts.deviceId ?? this.storage.getMeta('device_id') ?? crypto.randomUUID();
    this.storage.setMeta('device_id', this.deviceId);
  }

  // ---- local-first API ----------------------------------------------------

  getRow(collection: Collection, rowUuid: string): MirrorRow | undefined {
    return this.storage.getRow(collection, rowUuid);
  }

  query(collection: Collection): MirrorRow[] {
    return this.storage.allRows(collection);
  }

  pendingChanges(): number {
    return this.storage.outbox().length;
  }

  /** Local upsert: writes the mirror row and queues the change atomically. */
  mutate(collection: Collection, row: { rowUuid: string; data: Record<string, unknown> }): void {
    const existing = this.storage.getRow(collection, row.rowUuid);
    const baseVersion = existing?.version ?? 0;
    this.storage.transaction(() => {
      this.storage.putRow(collection, {
        collection,
        rowUuid: row.rowUuid,
        version: baseVersion,
        data: row.data,
      });
      this.storage.enqueue({
        collection,
        rowUuid: row.rowUuid,
        op: 'upsert',
        baseVersion,
        data: row.data,
      });
    });
    this.emitProgress();
  }

  /** Local delete: queues the deletion and drops the mirror row. */
  deleteLocal(collection: Collection, rowUuid: string): void {
    const existing = this.storage.getRow(collection, rowUuid);
    const baseVersion = existing?.version ?? 0;
    this.storage.transaction(() => {
      this.storage.enqueue({ collection, rowUuid, op: 'delete', baseVersion });
      this.storage.deleteRow(collection, rowUuid);
    });
    this.emitProgress();
  }

  // ---- sync lifecycle -----------------------------------------------------

  /** One-time bootstrap: snapshot the server into the mirror and set cursor. */
  async bootstrap(): Promise<void> {
    const snap = await this.transport.snapshot(COLLECTIONS);
    this.storage.transaction(() => {
      for (const collection of COLLECTIONS) {
        for (const row of snap.collections[collection] ?? []) {
          this.storage.putRow(collection, {
            collection,
            rowUuid: row.row_uuid,
            version: row.version,
            data: row.data ?? {},
          });
        }
      }
      this.storage.setCursor(snap.cursor);
    });
    this.lastSynced = new Date();
    this.emitProgress();
  }

  /**
   * Full sync cycle: pull remote changes, then push the outbox. Pull-before-
   * push lets the client see others' edits before its own are LWW-resolved.
   * Concurrent calls share the in-flight sync (they await the same result).
   */
  async sync(): Promise<SyncSummary> {
    if (!this.online) {
      this.emitProgress();
      throw new Error('offline');
    }
    if (this.currentSync) {
      return this.currentSync;
    }
    const run = this.runSync();
    this.currentSync = run;
    try {
      return await run;
    } finally {
      this.currentSync = null;
    }
  }

  private async runSync(): Promise<SyncSummary> {
    this.syncing = true;
    this.emitProgress();
    try {
      const pulled = await this.pull();
      const pushed = await this.push();
      this.backoffMs = 1000;
      this.lastSynced = new Date();
      this.lastError = undefined;
      const summary: SyncSummary = {
        pushed: pushed.applied,
        pulled: pulled.applied,
        conflicts: pushed.conflicts,
        lostEdits: pushed.lostEdits,
        cursor: this.storage.getCursor(),
        pending: this.pendingChanges(),
        at: this.lastSynced,
      };
      this.emitProgress();
      return summary;
    } catch (err) {
      this.lastError = err instanceof Error ? err.message : String(err);
      this.emitProgress();
      throw err;
    } finally {
      this.syncing = false;
    }
  }

  /** Apply feed entries since the cursor; advance the cursor. */
  async pull(): Promise<{ applied: number }> {
    let cursor = this.storage.getCursor();
    let applied = 0;
    for (;;) {
      const page: ChangesResponse = await this.transport.changes(cursor);
      this.storage.transaction(() => {
        for (const entry of page.rows) {
          const collection = entry.collection ?? inferCollection(entry.data);
          if (!collection) continue; // cannot apply without a collection
          const pending = this.storage.hasPending(entry.row_uuid);
          if (entry.op === 'delete') {
            if (!pending) {
              this.storage.deleteRow(collection, entry.row_uuid);
              applied++;
            }
          } else if (!pending) {
            this.storage.putRow(collection, {
              collection,
              rowUuid: entry.row_uuid,
              version: entry.version,
              data: entry.data ?? {},
            });
            applied++;
          }
        }
        this.storage.setCursor(page.cursor);
      });
      // Advance the pull cursor for the next page fetch. Page cursors are
      // monotonic and constant across the page (setCursor above persists it),
      // so this single hoisted update replaces any per-entry bookkeeping.
      cursor = page.cursor;
      if (!page.hasMore) break;
    }
    return { applied };
  }

  /** Drain the outbox through /push and apply the server's disposition. */
  async push(): Promise<{ applied: number; conflicts: number; lostEdits: number }> {
    const outbox = this.storage.outbox();
    if (outbox.length === 0) {
      return { applied: 0, conflicts: 0, lostEdits: 0 };
    }
    const resp = await this.transport.push({
      changes: outbox.map((e) => ({
        collection: e.collection,
        row_uuid: e.rowUuid,
        op: e.op,
        base_version: e.baseVersion,
        data: e.data,
      })),
      device_id: this.deviceId,
    });

    this.storage.transaction(() => {
      for (const result of resp.results) {
        this.storage.dropOutbox(result.rowUuid);
        this.reconcilePushResult(outbox, result);
      }
    });
    return {
      applied: resp.results.filter((r) => r.status === 'applied').length,
      conflicts: resp.results.filter((r) => r.status === 'conflict').length,
      lostEdits: resp.lostEdits,
    };
  }

  private reconcilePushResult(outbox: OutboxEntry[], result: {
    rowUuid: string;
    status: 'applied' | 'conflict' | 'merged' | 'ignored';
    serverId?: number;
    serverVersion: number;
    mappedToRowUuid?: string;
    data?: Record<string, unknown>;
  }): void {
    const entry = outbox.find((e) => e.rowUuid === result.rowUuid);
    if (!entry) return;

    switch (result.status) {
      case 'applied': {
        if (entry.op === 'upsert' && result.serverId !== undefined) {
          // Adopt the server identity + version; carry the client's data.
          const existing = this.storage.getRow(entry.collection, entry.rowUuid);
          const data = {
            ...(existing?.data ?? entry.data ?? {}),
            id: result.serverId,
          };
          this.storage.putRow(entry.collection, {
            collection: entry.collection,
            rowUuid: entry.rowUuid,
            version: result.serverVersion,
            data,
          });
        } else if (entry.op === 'delete') {
          this.storage.deleteRow(entry.collection, entry.rowUuid);
        }
        break;
      }
      case 'conflict': {
        // LWW kept the server row; adopt it (server data is authoritative).
        if (entry.op === 'upsert' && result.data) {
          this.storage.putRow(entry.collection, {
            collection: entry.collection,
            rowUuid: entry.rowUuid,
            version: result.serverVersion,
            data: result.data,
          });
        } else {
          this.storage.deleteRow(entry.collection, entry.rowUuid);
        }
        break;
      }
      case 'merged': {
        // Tag name-merge: the server reused another row. Rewrite THIS local
        // row to the surviving uuid, adopting the server's canonical state.
        const existing = this.storage.getRow(entry.collection, entry.rowUuid);
        if (existing && result.mappedToRowUuid) {
          this.storage.deleteRow(entry.collection, entry.rowUuid);
          this.storage.putRow(entry.collection, {
            collection: entry.collection,
            rowUuid: result.mappedToRowUuid,
            version: result.serverVersion,
            data: result.data ?? { ...existing.data, id: result.serverId },
          });
        }
        break;
      }
      case 'ignored':
        break;
    }
  }

  // ---- scheduler / events -------------------------------------------------

  onProgress(cb: (p: SyncProgress) => void): () => void {
    this.listeners.add(cb);
    return () => this.listeners.delete(cb);
  }

  private emitProgress(): void {
    const progress: SyncProgress = {
      state: this.syncing ? 'syncing' : this.online ? 'idle' : 'offline',
      lastSynced: this.lastSynced,
      pendingChanges: this.pendingChanges(),
      lastError: this.lastError,
    };
    for (const cb of this.listeners) cb(progress);
  }

  setOnline(online: boolean): void {
    if (online === this.online) return;
    this.online = online;
    if (online) {
      this.backoffMs = 1000;
      void this.sync().catch(() => undefined);
    }
    this.emitProgress();
  }

  /** Start periodic auto-sync with exponential backoff on failure. */
  start(intervalMs: number): void {
    if (this.timer) clearTimeout(this.timer);
    this.timer = undefined;
    this.intervalMs = intervalMs;
    this.backoffMs = 1000;
    this.scheduleNext();
  }

  stop(): void {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = undefined;
    }
  }

  /**
   * Schedules the next auto-sync. Steady state runs at intervalMs; after a
   * failure the next cycle waits backoffMs (doubles per failure, capped at
   * maxBackoffMs, reset to 1000 on success in runSync), so a flaky network
   * does not hammer the server at constant frequency.
   */
  private scheduleNext(): void {
    if (this.timer) return; // one in-flight schedule (or a sync running)
    const delay = Math.max(this.backoffMs, this.intervalMs);
    this.timer = setTimeout(() => {
      this.timer = undefined;
      void this.sync()
        .catch(() => {
          // Exponential backoff on repeated failure (errors surface via
          // progress.lastError).
          this.backoffMs = Math.min(this.backoffMs * 2, this.maxBackoffMs);
          this.emitProgress();
        })
        .finally(() => this.scheduleNext());
    }, delay);
    this.timer.unref?.();
  }
}
