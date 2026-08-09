/**
 * Regression stress test for the tag rename-vs-create flow (harness scenario
 * 06) against the deterministic MockServer + real SqliteStorageAdapter. The
 * live-backend flake (SQLITE_BUSY silently dropping a push as 'ignored') is
 * fixed server-side; this suite keeps the client-side merge semantics pinned
 * across 50 iterations.
 */

import { describe, it, expect } from 'vitest';
import { SyncEngine } from '../src/engine';
import { SqliteStorageAdapter } from '../src/sqlite';
import { MockServer } from './mock-server';
import type { Collection } from '../src/types';

async function makeEngine(server: MockServer, deviceId: string) {
  const storage = new SqliteStorageAdapter(':memory:');
  await storage.whenReady();
  const engine = new SyncEngine({ storage, transport: server, deviceId });
  return { engine, storage };
}

async function settle(devices: { engine: SyncEngine }[], rounds = 2) {
  for (let round = 0; round < rounds; round++) {
    for (const d of devices) {
      d.engine.setOnline(true);
      await d.engine.sync();
    }
  }
}

describe('flake repro: tag rename-vs-create', () => {
  it('A renames away, B creates same name — converges to two tags', async () => {
    for (let run = 0; run < 50; run++) {
      const server = new MockServer();
      const a = await makeEngine(server, 'dev-a');
      const b = await makeEngine(server, 'dev-b');
      await a.engine.bootstrap();
      await a.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'work', color: 'blue' } });
      await a.engine.sync();
      await b.engine.bootstrap();

      a.engine.setOnline(false);
      await a.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'tasks', color: 'blue' } });
      b.engine.setOnline(false);
      await b.engine.mutate('tags', { rowUuid: 'tag-w2', data: { name: 'work', color: 'blue' } });

      await settle([a, b], 2);

      const serverTags = (await server.snapshot(['tags'])).collections.tags ?? [];
      const names = serverTags.map((t) => (t.data as any).name).sort();
      if (JSON.stringify(names) !== JSON.stringify(['tasks', 'work'])) {
        const aRow = (await a.storage.getRow('tags', 'tag-w'))?.data;
        const bRow = (await b.storage.getRow('tags', 'tag-w2'))?.data;
        throw new Error(
          `run ${run}: server tags = ${JSON.stringify(names)}, a.tag-w=${JSON.stringify(aRow)}, b.tag-w2=${JSON.stringify(bRow)}, aPending=${await a.engine.pendingChanges()}, bPending=${await b.engine.pendingChanges()}`,
        );
      }
    }
  }, 60_000);
});
