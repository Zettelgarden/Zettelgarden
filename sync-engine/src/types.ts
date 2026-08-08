/**
 * Core types for the Zettelgarden local-first sync engine
 * (epic Zettelgarden-v5b, Phase 1a — issue nqx).
 *
 * The engine mirrors the server's pinned offline-writable tables (cards,
 * tasks, tags) in a local store, queues local mutations in an outbox, and
 * reconciles with the server via the Phase 0b sync API (snapshot / changes /
 * push). It is storage- and transport-agnostic: shells provide a
 * StorageAdapter (Tauri: tauri-plugin-sql; RN: op-sqlite via the WebView
 * bridge; tests: better-sqlite3 or in-memory) and a SyncTransport (fetch or a
 * test mock).
 */

export type Collection = 'cards' | 'tasks' | 'tags';

export const COLLECTIONS: Collection[] = ['cards', 'tasks', 'tags'];

/** Server payload for one row in a snapshot or changes-feed entry. */
export interface ServerRow {
  row_uuid: string;
  version: number;
  op: 'upsert' | 'delete';
  /** Present on changes-feed entries (snapshot groups by collection). */
  collection?: Collection;
  data?: Record<string, unknown>;
}

/** A row in the local mirror. data is the raw server/client column map. */
export interface MirrorRow {
  collection: Collection;
  rowUuid: string;
  version: number;
  data: Record<string, unknown>;
}

export type ChangeOp = 'upsert' | 'delete';

/** A queued local mutation awaiting push. */
export interface OutboxEntry {
  collection: Collection;
  rowUuid: string;
  op: ChangeOp;
  /** Server version the client wrote from (0 = never synced). */
  baseVersion: number;
  /** Full row state for upserts; absent for deletes. */
  data?: Record<string, unknown>;
}

/** Per-change server disposition from a push. */
export interface PushResult {
  rowUuid: string;
  status: 'applied' | 'conflict' | 'merged' | 'ignored';
  serverId?: number;
  serverVersion: number;
  mappedToRowUuid?: string;
  data?: Record<string, unknown>;
}

export interface PushResponse {
  results: PushResult[];
  cursor: number;
  lostEdits: number;
}

export interface SnapshotResponse {
  cursor: number;
  collections: Partial<Record<Collection, ServerRow[]>>;
}

export interface ChangesResponse {
  cursor: number;
  rows: ServerRow[];
  hasMore: boolean;
  /** True when `since` predates the pruned sync_log boundary: the engine must re-bootstrap (v5b.5). */
  reset?: boolean;
}

export interface PushRequest {
  changes: Array<{
    collection: Collection;
    row_uuid: string;
    op: ChangeOp;
    base_version: number;
    data?: Record<string, unknown>;
  }>;
  device_id: string;
  /** Client's last-known local cursor (retention heartbeat; v5b.5). */
  cursor?: number;
}

/** Transport abstraction over the Phase 0b sync API. */
export interface SyncTransport {
  snapshot(collections: Collection[]): Promise<SnapshotResponse>;
  changes(since: number): Promise<ChangesResponse>;
  push(req: PushRequest): Promise<PushResponse>;
}

export interface SyncSummary {
  pushed: number;
  pulled: number;
  conflicts: number;
  lostEdits: number;
  cursor: number;
  pending: number;
  at: Date;
}

export interface SyncProgress {
  state: 'idle' | 'syncing' | 'offline' | 'error';
  lastSynced?: Date;
  pendingChanges: number;
  lastError?: string;
}
