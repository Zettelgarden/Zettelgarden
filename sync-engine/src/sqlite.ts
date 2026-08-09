/**
 * SQLite StorageAdapter backed by better-sqlite3 (synchronous, transactional).
 * The desktop/mobile shells swap this for their own binding (Tauri invoke
 * commands, op-sqlite) while keeping the same async StorageAdapter contract.
 */

import Database from 'better-sqlite3';
import type { Collection, MirrorRow, OutboxEntry } from './types';
import type { StorageAdapter } from './storage';

export class SqliteStorageAdapter implements StorageAdapter {
  private db: Database.Database;
  private ready: Promise<void>;
  /** Serializes concurrent transactions (better-sqlite3 is sync, but an
   * async transaction body yields between awaits, so two overlapping
   * transaction() calls could interleave BEGIN/COMMIT without this). */
  private txQueue: Promise<void> = Promise.resolve();

  constructor(path: string) {
    this.db = new Database(path);
    this.db.pragma('journal_mode = WAL');
    this.ready = this.migrate();
  }

  private async migrate(): Promise<void> {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS sync_meta (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL
      );
      CREATE TABLE IF NOT EXISTS mirror_rows (
        collection TEXT NOT NULL,
        row_uuid TEXT NOT NULL,
        version INTEGER NOT NULL,
        data TEXT NOT NULL,
        PRIMARY KEY (collection, row_uuid)
      );
      CREATE TABLE IF NOT EXISTS sync_outbox (
        collection TEXT NOT NULL,
        row_uuid TEXT NOT NULL,
        op TEXT NOT NULL,
        base_version INTEGER NOT NULL,
        data TEXT,
        seq INTEGER PRIMARY KEY AUTOINCREMENT
      );
    `);
  }

  async whenReady(): Promise<void> {
    await this.ready;
  }

  close(): void {
    this.db.close();
  }

  /** Runs fn inside BEGIN IMMEDIATE…COMMIT (ROLLBACK on throw). The promise
   * queue serializes concurrent callers (a scheduler sync overlapping a UI
   * mutation waits for the previous transaction). NOT reentrant — a
   * transaction nested inside a transaction body would deadlock; no engine
   * caller nests. */
  async transaction<T>(fn: () => Promise<T>): Promise<T> {
    const prev = this.txQueue;
    let release!: () => void;
    this.txQueue = new Promise((r) => (release = r));
    await prev;
    try {
      this.db.exec('BEGIN IMMEDIATE');
      try {
        const result = await fn();
        this.db.exec('COMMIT');
        return result;
      } catch (err) {
        this.db.exec('ROLLBACK');
        throw err;
      }
    } finally {
      release();
    }
  }

  async getRow(collection: Collection, rowUuid: string): Promise<MirrorRow | undefined> {
    const row = this.db
      .prepare('SELECT row_uuid, version, data FROM mirror_rows WHERE collection = ? AND row_uuid = ?')
      .get(collection, rowUuid) as
      | { row_uuid: string; version: number; data: string }
      | undefined;
    if (!row) return undefined;
    return {
      collection,
      rowUuid: row.row_uuid,
      version: row.version,
      data: JSON.parse(row.data) as Record<string, unknown>,
    };
  }

  async putRow(collection: Collection, row: MirrorRow): Promise<void> {
    this.db
      .prepare(
        `INSERT INTO mirror_rows (collection, row_uuid, version, data)
         VALUES (?, ?, ?, ?)
         ON CONFLICT (collection, row_uuid) DO UPDATE SET version = excluded.version, data = excluded.data`,
      )
      .run(collection, row.rowUuid, row.version, JSON.stringify(row.data));
  }

  async deleteRow(collection: Collection, rowUuid: string): Promise<void> {
    this.db
      .prepare('DELETE FROM mirror_rows WHERE collection = ? AND row_uuid = ?')
      .run(collection, rowUuid);
  }

  async allRows(collection: Collection): Promise<MirrorRow[]> {
    const rows = this.db
      .prepare('SELECT row_uuid, version, data FROM mirror_rows WHERE collection = ?')
      .all(collection) as Array<{ row_uuid: string; version: number; data: string }>;
    return rows.map((r) => ({
      collection,
      rowUuid: r.row_uuid,
      version: r.version,
      data: JSON.parse(r.data) as Record<string, unknown>,
    }));
  }

  async outbox(): Promise<OutboxEntry[]> {
    const rows = this.db
      .prepare('SELECT collection, row_uuid, op, base_version, data FROM sync_outbox ORDER BY seq')
      .all() as Array<{
      collection: Collection;
      row_uuid: string;
      op: 'upsert' | 'delete';
      base_version: number;
      data: string | null;
    }>;
    return rows.map((r) => ({
      collection: r.collection,
      rowUuid: r.row_uuid,
      op: r.op,
      baseVersion: r.base_version,
      data: r.data ? (JSON.parse(r.data) as Record<string, unknown>) : undefined,
    }));
  }

  async hasPending(rowUuid: string): Promise<boolean> {
    const row = this.db
      .prepare('SELECT 1 FROM sync_outbox WHERE row_uuid = ? LIMIT 1')
      .get(rowUuid);
    return row !== undefined;
  }

  async enqueue(entry: OutboxEntry): Promise<void> {
    // Coalesce: replace any existing pending entry for the same row, keeping
    // the first (original) base_version so the server LWW compares against the
    // version the client last confirmed, not the latest local edit.
    const existing = this.db
      .prepare('SELECT base_version, seq FROM sync_outbox WHERE row_uuid = ?')
      .get(entry.rowUuid) as { base_version: number; seq: number } | undefined;
    if (existing) {
      this.db
        .prepare(
          'UPDATE sync_outbox SET collection = ?, op = ?, data = ? WHERE seq = ?',
        )
        .run(entry.collection, entry.op, entry.data ? JSON.stringify(entry.data) : null, existing.seq);
      return;
    }
    this.db
      .prepare(
        'INSERT INTO sync_outbox (collection, row_uuid, op, base_version, data) VALUES (?, ?, ?, ?, ?)',
      )
      .run(
        entry.collection,
        entry.rowUuid,
        entry.op,
        entry.baseVersion,
        entry.data ? JSON.stringify(entry.data) : null,
      );
  }

  async dropOutbox(rowUuid: string): Promise<void> {
    this.db.prepare('DELETE FROM sync_outbox WHERE row_uuid = ?').run(rowUuid);
  }

  async getCursor(): Promise<number> {
    const row = this.db.prepare('SELECT value FROM sync_meta WHERE key = ?').get('cursor') as
      | { value: string }
      | undefined;
    return row ? Number(row.value) : 0;
  }

  async setCursor(n: number): Promise<void> {
    this.db
      .prepare('INSERT OR REPLACE INTO sync_meta (key, value) VALUES (?, ?)')
      .run('cursor', String(n));
  }

  async getMeta(key: string): Promise<string | null> {
    const row = this.db.prepare('SELECT value FROM sync_meta WHERE key = ?').get(key) as
      | { value: string }
      | undefined;
    return row ? row.value : null;
  }

  async setMeta(key: string, value: string): Promise<void> {
    this.db
      .prepare('INSERT OR REPLACE INTO sync_meta (key, value) VALUES (?, ?)')
      .run(key, value);
  }
}
