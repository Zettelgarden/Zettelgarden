/**
 * Tauri StorageAdapter: the sync engine's local mirror over the Rust shell's
 * SQLite commands (epic Zettelgarden-v5b, Phase 2b — issue fv3). The webview
 * cannot run better-sqlite3, so every storage call crosses the IPC bridge to
 * `sync_db.rs` (single connection in WAL, origin-gated).
 *
 * The mirror schema is created here (whenReady) so it lives in exactly one
 * place — mirroring the engine's SqliteStorageAdapter.migrate(). Params use
 * numbered `?1` placeholders (rusqlite has no anonymous `?`).
 */

import type { StorageAdapter } from '@zettelgarden/sync-engine/storage';
import type { Collection, MirrorRow, OutboxEntry, SyncProgress } from '@zettelgarden/sync-engine/types';

/** Invokes a Tauri command via the internals bridge (the preload shim's
 * own invoke path; window.__TAURI__ is disabled with withGlobalTauri:false). */
export function tauriInvoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const internals = (window as any).__TAURI_INTERNALS__;
  if (!internals || typeof internals.invoke !== 'function') {
    return Promise.reject(new Error(`Tauri bridge unavailable for ${cmd}`));
  }
  return internals.invoke(cmd, args) as Promise<T>;
}

export function isDesktopApp(): boolean {
  return (
    typeof window !== 'undefined' &&
    !!(window as any).zgDesktop &&
    !!((window as any).__TAURI_INTERNALS__ ?? {}).invoke
  );
}

export class TauriStorageAdapter implements StorageAdapter {
  private ready: Promise<void>;
  /** Serializes whole transactions (each invoke releases the Rust mutex
   * between calls, so BEGIN…body…COMMIT must not interleave with another
   * transaction's BEGIN). */
  private txQueue: Promise<void> = Promise.resolve();

  constructor() {
    this.ready = this.migrate();
  }

  private async migrate(): Promise<void> {
    await tauriInvoke('sync_ping');
    await tauriInvoke('sync_exec', {
      sql: `
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
      `,
      params: [],
    });
  }

  async whenReady(): Promise<void> {
    await this.ready;
  }

  close(): void {
    // The Rust connection lives for the app's lifetime; nothing to release.
  }

  private async exec(sql: string, params: unknown[] = []): Promise<void> {
    await tauriInvoke('sync_exec', { sql, params });
  }

  private async query(sql: string, params: unknown[] = []): Promise<Record<string, unknown>[]> {
    const res = await tauriInvoke<Record<string, unknown>[]>('sync_query', { sql, params });
    return res ?? [];
  }

  async transaction<T>(fn: () => Promise<T>): Promise<T> {
    const prev = this.txQueue;
    let release!: () => void;
    this.txQueue = new Promise((r) => (release = r));
    await prev;
    try {
      await tauriInvoke('sync_begin');
      try {
        const result = await fn();
        await tauriInvoke('sync_commit');
        return result;
      } catch (err) {
        await tauriInvoke('sync_rollback').catch(() => undefined);
        throw err;
      }
    } finally {
      release();
    }
  }

  async getRow(collection: Collection, rowUuid: string): Promise<MirrorRow | undefined> {
    const rows = await this.query(
      'SELECT row_uuid, version, data FROM mirror_rows WHERE collection = ?1 AND row_uuid = ?2',
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
       VALUES (?1, ?2, ?3, ?4)
       ON CONFLICT (collection, row_uuid) DO UPDATE SET version = excluded.version, data = excluded.data`,
      [collection, row.rowUuid, row.version, JSON.stringify(row.data)],
    );
  }

  async deleteRow(collection: Collection, rowUuid: string): Promise<void> {
    await this.exec('DELETE FROM mirror_rows WHERE collection = ?1 AND row_uuid = ?2', [
      collection,
      rowUuid,
    ]);
  }

  async allRows(collection: Collection): Promise<MirrorRow[]> {
    const rows = await this.query(
      'SELECT row_uuid, version, data FROM mirror_rows WHERE collection = ?1',
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
      data: r.data ? (JSON.parse(r.data as string) as Record<string, unknown>) : undefined,
    }));
  }

  async hasPending(rowUuid: string): Promise<boolean> {
    const rows = await this.query('SELECT 1 FROM sync_outbox WHERE row_uuid = ?1 LIMIT 1', [
      rowUuid,
    ]);
    return rows.length > 0;
  }

  async enqueue(entry: OutboxEntry): Promise<void> {
    // Coalesce: replace any existing pending entry for the same row, keeping
    // the first (original) base_version (same contract as the engine's
    // SqliteStorageAdapter).
    const existing = await this.query(
      'SELECT base_version, seq FROM sync_outbox WHERE row_uuid = ?1',
      [entry.rowUuid],
    );
    if (existing[0]) {
      await this.exec(
        'UPDATE sync_outbox SET collection = ?1, op = ?2, data = ?3 WHERE seq = ?4',
        [entry.collection, entry.op, entry.data ? JSON.stringify(entry.data) : null, existing[0].seq],
      );
      return;
    }
    await this.exec(
      'INSERT INTO sync_outbox (collection, row_uuid, op, base_version, data) VALUES (?1, ?2, ?3, ?4, ?5)',
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
    await this.exec('DELETE FROM sync_outbox WHERE row_uuid = ?1', [rowUuid]);
  }

  async getCursor(): Promise<number> {
    const rows = await this.query('SELECT value FROM sync_meta WHERE key = ?1', ['cursor']);
    return rows[0] ? Number(rows[0].value) : 0;
  }

  async setCursor(n: number): Promise<void> {
    await this.exec('INSERT OR REPLACE INTO sync_meta (key, value) VALUES (?1, ?2)', [
      'cursor',
      String(n),
    ]);
  }

  async getMeta(key: string): Promise<string | null> {
    const rows = await this.query('SELECT value FROM sync_meta WHERE key = ?1', [key]);
    return rows[0] ? (rows[0].value as string) : null;
  }

  async setMeta(key: string, value: string): Promise<void> {
    await this.exec('INSERT OR REPLACE INTO sync_meta (key, value) VALUES (?1, ?2)', [key, value]);
  }
}

// Re-export the progress type so consumers can subscribe without importing
// the engine package's internals.
export type { SyncProgress };
