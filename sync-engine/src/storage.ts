/**
 * StorageAdapter: the local-first store the engine reads/writes. Shells
 * implement this against their SQLite binding. All mutations MUST be
 * transactionally atomic — a local row write and its outbox enqueue happen
 * together, so a crash never leaves a row written with no pending sync.
 */

import type { Collection, MirrorRow, OutboxEntry } from './types';

export interface StorageAdapter {
  /** Runs fn atomically (SQLite tx; in-memory adapters may no-op). */
  transaction<T>(fn: () => T): T;

  getRow(collection: Collection, rowUuid: string): MirrorRow | undefined;
  putRow(collection: Collection, row: MirrorRow): void;
  deleteRow(collection: Collection, rowUuid: string): void;
  allRows(collection: Collection): MirrorRow[];

  /** Pending local mutations, oldest first. */
  outbox(): OutboxEntry[];
  hasPending(rowUuid: string): boolean;
  /** Replaces an existing pending entry for the same row (coalescing). */
  enqueue(entry: OutboxEntry): void;
  dropOutbox(rowUuid: string): void;

  getCursor(): number;
  setCursor(n: number): void;

  getMeta(key: string): string | null;
  setMeta(key: string, value: string): void;
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

  transaction<T>(fn: () => T): T {
    return fn();
  }

  getRow(collection: Collection, rowUuid: string): MirrorRow | undefined {
    return this.rows.get(this.key(collection, rowUuid));
  }

  putRow(collection: Collection, row: MirrorRow): void {
    this.rows.set(this.key(collection, rowUuidOf(row)), row);
  }

  deleteRow(collection: Collection, rowUuid: string): void {
    this.rows.delete(this.key(collection, rowUuid));
  }

  allRows(collection: Collection): MirrorRow[] {
    return [...this.rows.values()].filter((r) => r.collection === collection);
  }

  outbox(): OutboxEntry[] {
    return [...this.outboxList];
  }

  hasPending(rowUuid: string): boolean {
    return this.outboxList.some((e) => e.rowUuid === rowUuid);
  }

  enqueue(entry: OutboxEntry): void {
    const i = this.outboxList.findIndex((e) => e.rowUuid === entry.rowUuid);
    if (i >= 0) {
      this.outboxList[i] = entry;
    } else {
      this.outboxList.push(entry);
    }
  }

  dropOutbox(rowUuid: string): void {
    this.outboxList = this.outboxList.filter((e) => e.rowUuid !== rowUuid);
  }

  getCursor(): number {
    return this.cursor;
  }

  setCursor(n: number): void {
    this.cursor = n;
  }

  getMeta(key: string): string | null {
    return this.meta.get(key) ?? null;
  }

  setMeta(key: string, value: string): void {
    this.meta.set(key, value);
  }
}

function rowUuidOf(row: MirrorRow): string {
  return row.rowUuid;
}
