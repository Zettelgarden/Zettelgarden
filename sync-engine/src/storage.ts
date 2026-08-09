/**
 * StorageAdapter: the local-first store the engine reads/writes. Shells
 * implement this against their SQLite binding. All mutations MUST be
 * transactionally atomic — a local row write and its outbox enqueue happen
 * together, so a crash never leaves a row written with no pending sync.
 *
 * The interface is ASYNC so any host can implement it: the desktop shell
 * (Tauri) and the mobile shell (React Native WebView bridge) both talk to
 * their native SQLite over async IPC, and Node-based adapters (better-sqlite3
 * in tests/harness) trivially wrap synchronous calls. The engine awaits every
 * storage call; the UI already consumes the engine's local API through
 * async React Query queryFn/mutation handlers.
 */

import type { Collection, MirrorRow, OutboxEntry } from './types';

export interface StorageAdapter {
  /** Runs fn atomically (SQLite tx; in-memory adapters may no-op). */
  transaction<T>(fn: () => Promise<T>): Promise<T>;

  getRow(collection: Collection, rowUuid: string): Promise<MirrorRow | undefined>;
  putRow(collection: Collection, row: MirrorRow): Promise<void>;
  deleteRow(collection: Collection, rowUuid: string): Promise<void>;
  allRows(collection: Collection): Promise<MirrorRow[]>;

  /** Pending local mutations, oldest first. */
  outbox(): Promise<OutboxEntry[]>;
  hasPending(rowUuid: string): Promise<boolean>;
  /** Replaces an existing pending entry for the same row (coalescing). */
  enqueue(entry: OutboxEntry): Promise<void>;
  dropOutbox(rowUuid: string): Promise<void>;

  getCursor(): Promise<number>;
  setCursor(n: number): Promise<void>;

  getMeta(key: string): Promise<string | null>;
  setMeta(key: string, value: string): Promise<void>;
}

/**
 * In-memory adapter for engine logic tests. Not durable; single-user; the
 * transaction() boundary is a synchronous no-op (JS is single-threaded).
 */
export class InMemoryAdapter implements StorageAdapter {
  private rows = new Map<string, MirrorRow>();
  private outboxList: OutboxEntry[] = [];
  private cursor = 0;
  private meta = new Map<string, string>();

  private key(collection: Collection, rowUuid: string): string {
    return `${collection}:${rowUuid}`;
  }

  async transaction<T>(fn: () => Promise<T>): Promise<T> {
    return fn();
  }

  async getRow(collection: Collection, rowUuid: string): Promise<MirrorRow | undefined> {
    return this.rows.get(this.key(collection, rowUuid));
  }

  async putRow(collection: Collection, row: MirrorRow): Promise<void> {
    this.rows.set(this.key(collection, row.rowUuid), row);
  }

  async deleteRow(collection: Collection, rowUuid: string): Promise<void> {
    this.rows.delete(this.key(collection, rowUuid));
  }

  async allRows(collection: Collection): Promise<MirrorRow[]> {
    return [...this.rows.values()].filter((r) => r.collection === collection);
  }

  async outbox(): Promise<OutboxEntry[]> {
    return [...this.outboxList];
  }

  async hasPending(rowUuid: string): Promise<boolean> {
    return this.outboxList.some((e) => e.rowUuid === rowUuid);
  }

  async enqueue(entry: OutboxEntry): Promise<void> {
    const i = this.outboxList.findIndex((e) => e.rowUuid === entry.rowUuid);
    if (i >= 0) {
      // Coalesce: replace op/data but KEEP the first (original) base_version
      // so the server LWW compares against the version the client last
      // confirmed — not the latest local edit. Mirrors SqliteStorageAdapter.
      // In the delete-then-recreate case this keeps base 1 instead of 0, so
      // the server applies the recreate instead of misreading it as a
      // create-retry and silently dropping it.
      this.outboxList[i] = { ...entry, baseVersion: this.outboxList[i]!.baseVersion };
    } else {
      this.outboxList.push(entry);
    }
  }

  async dropOutbox(rowUuid: string): Promise<void> {
    this.outboxList = this.outboxList.filter((e) => e.rowUuid !== rowUuid);
  }

  async getCursor(): Promise<number> {
    return this.cursor;
  }

  async setCursor(n: number): Promise<void> {
    this.cursor = n;
  }

  async getMeta(key: string): Promise<string | null> {
    return this.meta.get(key) ?? null;
  }

  async setMeta(key: string, value: string): Promise<void> {
    this.meta.set(key, value);
  }
}
