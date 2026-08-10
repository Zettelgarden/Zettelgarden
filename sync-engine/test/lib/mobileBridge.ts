/**
 * Loopback bridge for MobileStorageAdapter tests/harness (Zettelgarden-c6l.2).
 *
 * The adapter (zettelkasten-front/src/data/mobileStorageAdapter.ts) calls
 * zgMobile.invoke(cmd, args) and expects promise results. On-device the RN
 * shell (mobile/src/bridge.ts) dispatches sql_* to op-sqlite. In Node tests
 * this module stands in for BOTH sides: the executor is better-sqlite3 (the
 * same engine the sync engine itself tests against), and the invoke function
 * mirrors the RN bridge handler's command surface — so the adapter's SQL,
 * parameter binding, and transaction semantics are exercised against real
 * SQLite over the exact protocol the device bridge implements.
 */

import Database from "better-sqlite3";
import type { StorageAdapter } from "../../src/storage";

export interface SqlExecutor {
  begin(): void;
  commit(): void;
  rollback(): void;
  exec(sql: string, params: unknown[]): { rowsAffected: number };
  query(sql: string, params: unknown[]): Record<string, unknown>[];
  close(): void;
}

/** Opens a better-sqlite3 executor (real SQLite, mirror of the RN side). */
export function createSqlExecutor(dbPath = ":memory:"): SqlExecutor {
  const db = new Database(dbPath);
  return {
    begin() {
      db.exec("BEGIN IMMEDIATE");
    },
    commit() {
      db.exec("COMMIT");
    },
    rollback() {
      db.exec("ROLLBACK");
    },
    exec(sql, params) {
      return {
        rowsAffected: db.prepare(sql).run(...(params as never[])).changes,
      };
    },
    query(sql, params) {
      return db.prepare(sql).all(...(params as never[])) as Record<
        string,
        unknown
      >[];
    },
    close() {
      db.close();
    },
  };
}

/**
 * Wires an executor to the adapter's invoke signature. Command surface must
 * match mobile/src/bridge.ts (sql_begin/exec/query/commit/rollback + ping).
 */
export function createLoopbackInvoke(
  executor: SqlExecutor,
): (cmd: string, args?: Record<string, unknown>) => Promise<unknown> {
  return async (cmd, args = {}) => {
    switch (cmd) {
      case "ping":
        return "pong";
      case "sql_begin":
        executor.begin();
        return null;
      case "sql_commit":
        executor.commit();
        return null;
      case "sql_rollback":
        executor.rollback();
        return null;
      case "sql_exec":
        return executor.exec(
          args.sql as string,
          (args.params as unknown[]) ?? [],
        );
      case "sql_query":
        return executor.query(
          args.sql as string,
          (args.params as unknown[]) ?? [],
        );
      default:
        throw new Error(`unknown bridge command: ${cmd}`);
    }
  };
}

/** Convenience: adapter bound to a fresh executor (open + migrate + close). */
export function openMobileAdapter(
  dbPath = ":memory:",
): Promise<{ adapter: StorageAdapter; close: () => void }> {
  const executor = createSqlExecutor(dbPath);
  const invoke = createLoopbackInvoke(executor);
  const { MobileStorageAdapter } = requireMobileStorageAdapter();
  const adapter = new MobileStorageAdapter(invoke);
  return adapter.whenReady().then(() => ({
    adapter,
    close: () => executor.close(),
  }));
}

/**
 * Loads the frontend's MobileStorageAdapter via a relative dynamic import
 * (vitest processes the .ts through its module runner; the adapter's own
 * @zettelgarden/sync-engine/* imports resolve through the vitest aliases in
 * vitest.config.ts / harness/vitest.harness.config.ts).
 */
export async function loadMobileStorageAdapter(): Promise<{
  MobileStorageAdapter: new (
    invoke?: (cmd: string, args?: Record<string, unknown>) => Promise<unknown>,
  ) => StorageAdapter;
}> {
  return (await import("../../../zettelkasten-front/src/data/mobileStorageAdapter.ts")) as {
    MobileStorageAdapter: new (
      invoke?: (
        cmd: string,
        args?: Record<string, unknown>,
      ) => Promise<unknown>,
    ) => StorageAdapter;
  };
}
