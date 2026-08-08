/**
 * Device helpers: a "device" is a real local SQLite mirror (SqliteStorageAdapter)
 * wired to a real SyncEngine talking to the live backend over HTTP. Two devices
 * share one account but keep separate DB files and device ids — exactly the
 * Phase 1b "two local DBs + live backend" shape.
 */

import { HttpTransport } from '../../src/http';
import { SyncEngine } from '../../src/engine';
import { SqliteStorageAdapter } from '../../src/sqlite';
import type { SyncTransport } from '../../src/types';

export interface Device {
  /** e.g. 'dev-a' */
  id: string;
  engine: SyncEngine;
  storage: SqliteStorageAdapter;
  /** Transport bound to the harness account; handy for raw API assertions. */
  transport: SyncTransport;
  close(): void;
}

export async function makeDevice(
  baseUrl: string,
  auth: string,
  id: string,
  dbPath: string,
): Promise<Device> {
  const storage = new SqliteStorageAdapter(dbPath);
  await storage.whenReady();
  const transport = new HttpTransport({ baseUrl, token: () => auth });
  const engine = new SyncEngine({ storage, transport, deviceId: id });
  return {
    id,
    engine,
    storage,
    transport,
    close: () => storage.close(),
  };
}
