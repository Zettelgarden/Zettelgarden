/**
 * Native SQLite executor for the mobile shell (Zettelgarden-c6l.2).
 *
 * Runs on the RN JS thread via op-sqlite — NOT inside the WebView. The
 * webview's sync engine reaches it over the postMessage bridge
 * (webviewShim.js → src/bridge.ts → this module). The command surface
 * mirrors the desktop shell's sync_db.rs (begin/exec/query/commit/rollback)
 * so the webview-side MobileStorageAdapter is a near copy of
 * TauriStorageAdapter with a different transport.
 *
 * Transactions are explicit BEGIN IMMEDIATE / COMMIT / ROLLBACK across
 * execute calls (op-sqlite keeps a single native connection per open()).
 * The webview-side adapter serializes concurrent transactions with a promise
 * queue, exactly like the desktop adapter.
 */

import { open, type DB } from '@op-engineering/op-sqlite';

export const MIRROR_DB_NAME = 'zettelgarden.db';

let db: DB | null = null;

function getDb(): DB {
  if (!db) {
    db = open({ name: MIRROR_DB_NAME });
  }
  return db;
}

/** BEGIN IMMEDIATE — the write lock waits, so the tx queue wins the lock. */
export async function sqlBegin(): Promise<void> {
  await getDb().execute('BEGIN IMMEDIATE');
}

export async function sqlCommit(): Promise<void> {
  await getDb().execute('COMMIT');
}

export async function sqlRollback(): Promise<void> {
  await getDb().execute('ROLLBACK');
}

export async function sqlExec(
  sql: string,
  params: unknown[] = [],
): Promise<{ rowsAffected: number }> {
  const result = await getDb().execute(sql, params as never[]);
  return { rowsAffected: result.rowsAffected };
}

export async function sqlQuery(
  sql: string,
  params: unknown[] = [],
): Promise<Record<string, unknown>[]> {
  const result = await getDb().execute(sql, params as never[]);
  return result.rows as unknown as Record<string, unknown>[];
}

/**
 * Wipes the mirror for logout / account switch (c6l.3 calls this on logout).
 * The per-user isolation contract says logout clears the local copy.
 */
export async function sqlReset(): Promise<void> {
  if (!db) return;
  const current = db;
  db = null;
  current.close();
  current.delete();
}
