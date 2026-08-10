/**
 * Mobile StorageAdapter: the sync engine's local mirror over the RN WebView
 * bridge (epic Zettelgarden-v5b, Phase 3a — issue c6l.2). The webview cannot
 * run op-sqlite directly — op-sqlite lives on the RN JS thread, so every
 * storage call crosses the postMessage bridge (webviewShim.js → src/bridge.ts
 * → src/sqlite.ts). The mobile SQLiteAdapter IS this bridge, exactly the
 * desktop shape (TauriStorageAdapter → Rust sync_db.rs).
 *
 * The mirror schema is created here (whenReady) so it lives in exactly one
 * place, mirroring TauriStorageAdapter. Params use plain `?` placeholders —
 * both op-sqlite and the Node loopback executor (tests/harness) bind them
 * positionally.
 */

import type { StorageAdapter } from '@zettelgarden/sync-engine/storage';
import type {
  Collection,
  MirrorRow,
  OutboxEntry,
} from '@zettelgarden/sync-engine/types';

/** Invokes a bridge command via the zgMobile shim installed by
 * mobile/webviewShim.js. Injectable so tests can loop the bridge back to a
 * real SQLite executor (sync-engine's adapter matrix + harness). */
export function mobileInvoke<T = unknown>(
  cmd: string,
  args?: Record<string, unknown>,
): Promise<T> {
  const zg = (window as any).zgMobile;
  if (!zg || typeof zg.invoke !== 'function') {
    return Promise.reject(new Error(`Mobile bridge unavailable for ${cmd}`));
  }
  return zg.invoke(cmd, args) as Promise<T>;
}

export class MobileStorageAdapter implements StorageAdapter {
  private ready: Promise<void>;
  /** Serializes whole transactions (each invoke releases the RN-side
   * executor between calls, so BEGIN…body…COMMIT must not interleave with
   * another transaction's BEGIN). */
  private txQueue: Promise<void> = Promise.resolve();

  constructor(private readonly invoke: typeof mobileInvoke = mobileInvoke) {
    this.ready = this.migrate();
  }

  private async migrate(): Promise<void> {
    await this.invoke('ping');
    // Each CREATE is its own sql_exec (matches the Tauri adapter; keeps the
    // bridge commands single-statement).
    for (const sql of [
      `CREATE TABLE IF NOT EXISTS sync_meta (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL
      )`,
      `CREATE TABLE IF NOT EXISTS mirror_rows (
        collection TEXT NOT NULL,
        row_uuid TEXT NOT NULL,
        version INTEGER NOT NULL,
        data TEXT NOT NULL,
        PRIMARY KEY (collection, row_uuid)
      )`,
      `CREATE TABLE IF NOT EXISTS sync_outbox (
        collection TEXT NOT NULL,
        row_uuid TEXT NOT NULL,
        op TEXT NOT NULL,
        base_version INTEGER NOT NULL,
        data TEXT,
        seq INTEGER PRIMARY KEY AUTOINCREMENT
      )`,
    ]) {
      await this.invoke('sql_exec', { sql, params: [] });
    }
  }

  async whenReady(): Promise<void> {
    await this.ready;
  }

  close(): void {
    // The RN-side connection lives for the app's lifetime; nothing to
    // release from the webview (db_reset handles logout — c6l.3).
  }

  private async exec(sql: string, params: unknown[] = []): Promise<void> {
    await this.invoke('sql_exec', { sql, params });
  }

  private async query(
    sql: string,
    params: unknown[] = [],
  ): Promise<Record<string, unknown>[]> {
    const rows = await this.invoke<Record<string, unknown>[] | null>(
      'sql_query',
      { sql, params },
    );
    return rows ?? [];
  }

  // The promise queue serializes concurrent callers (each invoke releases the
  // RN-side executor between calls, so BEGIN…body…COMMIT must not interleave
  // with another transaction's BEGIN). NOT reentrant — no engine caller
  // nests.
  async transaction<T>(fn: () => Promise<T>): Promise<T> {
    const prev = this.txQueue;
    let release!: () => void;
    this.txQueue = new Promise((r) => (release = r));
    await prev;
    try {
      await this.invoke('sql_begin');
      try {
        const result = await fn();
        await this.invoke('sql_commit');
        return result;
      } catch (err) {
        await this.invoke('sql_rollback').catch(() => undefined);
        throw err;
      }
    } finally {
      release();
    }
  }

  async getRow(
    collection: Collection,
    rowUuid: string,
  ): Promise<MirrorRow | undefined> {
    const rows = await this.query(
      'SELECT row_uuid, version, data FROM mirror_rows WHERE collection = ? AND row_uuid = ?',
      [collection, rowUuid],
    );
    const row = rows[0];
    if (!row) return undefined;
    return {
      collection,
      rowUuid: row.row_uuid as string,
      version: row.version as number,
      data: JSON.parse(row.data as string) as Record<string, unknown>,
    };
  }

  async putRow(collection: Collection, row: MirrorRow): Promise<void> {
    await this.exec(
      `INSERT INTO mirror_rows (collection, row_uuid, version, data)
       VALUES (?, ?, ?, ?)
       ON CONFLICT (collection, row_uuid) DO UPDATE SET version = excluded.version, data = excluded.data`,
      [collection, row.rowUuid, row.version, JSON.stringify(row.data)],
    );
  }

  async deleteRow(collection: Collection, rowUuid: string): Promise<void> {
    await this.exec(
      'DELETE FROM mirror_rows WHERE collection = ? AND row_uuid = ?',
      [collection, rowUuid],
    );
  }

  async allRows(collection: Collection): Promise<MirrorRow[]> {
    const rows = await this.query(
      'SELECT row_uuid, version, data FROM mirror_rows WHERE collection = ?',
      [collection],
    );
    return rows.map((r) => ({
      collection,
      rowUuid: r.row_uuid as string,
      version: r.version as number,
      data: JSON.parse(r.data as string) as Record<string, unknown>,
    }));
  }

  async outbox(): Promise<OutboxEntry[]> {
    const rows = await this.query(
      'SELECT collection, row_uuid, op, base_version, data FROM sync_outbox ORDER BY seq',
    );
    return rows.map((r) => ({
      collection: r.collection as Collection,
      rowUuid: r.row_uuid as string,
      op: r.op as 'upsert' | 'delete',
      baseVersion: r.base_version as number,
      data: r.data
        ? (JSON.parse(r.data as string) as Record<string, unknown>)
        : undefined,
    }));
  }

  async hasPending(rowUuid: string): Promise<boolean> {
    const rows = await this.query(
      'SELECT 1 FROM sync_outbox WHERE row_uuid = ? LIMIT 1',
      [rowUuid],
    );
    return rows.length > 0;
  }

  async enqueue(entry: OutboxEntry): Promise<void> {
    // Coalesce: replace any existing pending entry for the same row, keeping
    // the first (original) base_version (same contract as the engine's
    // SqliteStorageAdapter / TauriStorageAdapter).
    const existing = await this.query(
      'SELECT base_version, seq FROM sync_outbox WHERE row_uuid = ?',
      [entry.rowUuid],
    );
    if (existing[0]) {
      await this.exec(
        'UPDATE sync_outbox SET collection = ?, op = ?, data = ? WHERE seq = ?',
        [
          entry.collection,
          entry.op,
          entry.data ? JSON.stringify(entry.data) : null,
          existing[0].seq,
        ],
      );
      return;
    }
    await this.exec(
      'INSERT INTO sync_outbox (collection, row_uuid, op, base_version, data) VALUES (?, ?, ?, ?, ?)',
      [
        entry.collection,
        entry.rowUuid,
        entry.op,
        entry.baseVersion,
        entry.data ? JSON.stringify(entry.data) : null,
      ],
    );
  }

  async dropOutbox(rowUuid: string): Promise<void> {
    await this.exec('DELETE FROM sync_outbox WHERE row_uuid = ?', [rowUuid]);
  }

  async getCursor(): Promise<number> {
    const rows = await this.query('SELECT value FROM sync_meta WHERE key = ?', [
      'cursor',
    ]);
    return rows[0] ? Number(rows[0].value) : 0;
  }

  async setCursor(n: number): Promise<void> {
    await this.exec(
      'INSERT OR REPLACE INTO sync_meta (key, value) VALUES (?, ?)',
      ['cursor', String(n)],
    );
  }

  async getMeta(key: string): Promise<string | null> {
    const rows = await this.query('SELECT value FROM sync_meta WHERE key = ?', [
      key,
    ]);
    return rows[0] ? (rows[0].value as string) : null;
  }

  async setMeta(key: string, value: string): Promise<void> {
    await this.exec(
      'INSERT OR REPLACE INTO sync_meta (key, value) VALUES (?, ?)',
      [key, value],
    );
  }
}
