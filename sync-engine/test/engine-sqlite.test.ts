import { describe, it, expect } from 'vitest';
import { SyncEngine } from '../src/engine';
import { SqliteStorageAdapter } from '../src/sqlite';
import type { StorageAdapter } from '../src/storage';
import { MockServer } from './mock-server';
import type { Collection } from '../src/types';

// Engine scenarios against the REAL SqliteStorageAdapter (v5b.9). The main
// engine suite runs on InMemoryAdapter; these prove the engine's offline gap,
// dropped-response retry, delete-then-recreate and tag-merge flows behave
// identically on the durable adapter (outbox coalescing, base_version
// preservation, cursor/meta persistence are adapter-level behaviors that only
// show up end-to-end here).

async function makeEngine(server: MockServer, deviceId: string): Promise<{
  engine: SyncEngine;
  storage: StorageAdapter;
}> {
  const storage = new SqliteStorageAdapter(':memory:');
  await storage.whenReady();
  const engine = new SyncEngine({ storage, transport: server, deviceId });
  return { engine, storage };
}

function cardData(title: string): Record<string, unknown> {
  return { title, body: title, card_id: title };
}

describe('SyncEngine on SqliteStorageAdapter', () => {
  it('offline gap: mutations queue while offline and converge on reconnect', async () => {
    const server = new MockServer();
    const { engine, storage } = await makeEngine(server, 'dev-a');
    await engine.bootstrap();

    engine.setOnline(false);
    engine.mutate('cards', { rowUuid: 'a1', data: cardData('card A') });
    engine.mutate('tasks', { rowUuid: 't1', data: { title: 'task B' } });
    engine.deleteLocal('cards', 'does-not-exist');
    expect(engine.pendingChanges()).toBe(3);

    engine.setOnline(true);
    await engine.sync();
    expect(engine.pendingChanges()).toBe(0);
    expect(storage.getRow('cards', 'a1')?.data.title).toBe('card A');
    expect(storage.getRow('tasks', 't1')?.data.title).toBe('task B');
  });

  it('dropped-response retry: re-push of the same entry is idempotent', async () => {
    const server = new MockServer();
    const storage = new SqliteStorageAdapter(':memory:');
    await storage.whenReady();

    const failingTransport = {
      snapshot: (c: Collection[]) => server.snapshot(c),
      changes: (s: number) => server.changes(s),
      push: async (req) => {
        await server.push(req);
        throw new Error('network dropped the response');
      },
    };
    const engine1 = new SyncEngine({ storage, transport: failingTransport, deviceId: 'dev-a' });
    await engine1.bootstrap();
    engine1.mutate('cards', { rowUuid: 'r1', data: cardData('retry me') });
    await expect(engine1.sync()).rejects.toThrow('network dropped');

    expect(engine1.pendingChanges()).toBe(1); // entry retained for retry
    expect((await server.snapshot(['cards'])).collections.cards!).toHaveLength(1);

    // Fresh engine on the same storage retries the same entry (base 0).
    const engine2 = new SyncEngine({ storage, transport: server, deviceId: 'dev-a' });
    await engine2.sync();
    expect(engine2.pendingChanges()).toBe(0);
    expect((await server.snapshot(['cards'])).collections.cards!).toHaveLength(1);
    expect(storage.getRow('cards', 'r1')!.version).toBe(1);
  });

  it('delete-then-recreate offline converges (coalesced outbox keeps original base)', async () => {
    const server = new MockServer();
    server.seed('cards', 'dtr', cardData('v1'));
    const { engine, storage } = await makeEngine(server, 'dev-a');
    await engine.bootstrap();
    expect(storage.getRow('cards', 'dtr')!.version).toBe(1);

    engine.setOnline(false);
    engine.deleteLocal('cards', 'dtr');
    engine.mutate('cards', { rowUuid: 'dtr', data: cardData('recreated') });

    const outbox = storage.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.op).toBe('upsert');
    expect(outbox[0]!.baseVersion).toBe(1); // original base preserved

    engine.setOnline(true);
    const summary = await engine.sync();
    expect(summary.conflicts).toBe(0);
    expect(summary.lostEdits).toBe(0);

    const serverRow = (await server.snapshot(['cards'])).collections.cards!.find(
      (r) => r.row_uuid === 'dtr',
    )!;
    expect(serverRow.data?.title).toBe('recreated');
    expect(storage.getRow('cards', 'dtr')!.data.title).toBe('recreated');
  });

  it('tag name-merge adopts the surviving uuid and counts the lost edit', async () => {
    const server = new MockServer();
    const devA = await makeEngine(server, 'dev-a');
    const devB = await makeEngine(server, 'dev-b');
    await devA.engine.bootstrap();
    await devB.engine.bootstrap();

    devA.engine.mutate('tags', { rowUuid: 'tag-a', data: { name: 'Work', color: 'black' } });
    await devA.engine.sync();

    devB.engine.mutate('tags', { rowUuid: 'tag-b', data: { name: 'Work', color: 'red' } });
    const summary = await devB.engine.sync();
    expect(summary.lostEdits).toBe(1); // B's differing color was discarded
    expect(devB.storage.getRow('tags', 'tag-a')).toBeDefined();
    expect(devB.storage.getRow('tags', 'tag-b')).toBeUndefined();
    expect((await server.snapshot(['tags'])).collections.tags!).toHaveLength(1);
  });
});
