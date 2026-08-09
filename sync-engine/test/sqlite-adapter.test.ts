import { describe, it, expect, beforeAll } from 'vitest';
import { SqliteStorageAdapter } from '../src/sqlite';

let adapter: SqliteStorageAdapter;

beforeAll(async () => {
  adapter = new SqliteStorageAdapter(':memory:');
  await adapter.whenReady();
});

describe('SqliteStorageAdapter', () => {
  it('mirror CRUD round-trips', async () => {
    await adapter.putRow('cards', {
      collection: 'cards',
      rowUuid: 'r1',
      version: 3,
      data: { id: 5, title: 'hello', body: 'world' },
    });
    const row = await adapter.getRow('cards', 'r1');
    expect(row).toBeDefined();
    expect(row!.version).toBe(3);
    expect(row!.data.title).toBe('hello');
    expect(row!.data.id).toBe(5);

    await adapter.putRow('cards', {
      collection: 'cards',
      rowUuid: 'r1',
      version: 4,
      data: { id: 5, title: 'updated' },
    });
    expect((await adapter.getRow('cards', 'r1'))!.version).toBe(4);

    await adapter.deleteRow('cards', 'r1');
    expect(await adapter.getRow('cards', 'r1')).toBeUndefined();
  });

  it('collections are isolated', async () => {
    await adapter.putRow('cards', { collection: 'cards', rowUuid: 'c1', version: 1, data: {} });
    await adapter.putRow('tasks', { collection: 'tasks', rowUuid: 't1', version: 1, data: {} });
    expect(await adapter.allRows('cards')).toHaveLength(1);
    expect(await adapter.allRows('tasks')).toHaveLength(1);
  });

  it('outbox coalesces by rowUuid, keeping the original base_version', async () => {
    await adapter.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'a' } });
    await adapter.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'b' } });
    await adapter.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'c' } });

    const outbox = await adapter.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.data?.title).toBe('c');
    expect(outbox[0]!.baseVersion).toBe(4);
    expect(await adapter.hasPending('c1')).toBe(true);

    await adapter.dropOutbox('c1');
    expect(await adapter.outbox()).toHaveLength(0);
  });

  it('delete-then-recreate coalesces to an upsert that keeps the original base', async () => {
    await adapter.enqueue({ collection: 'cards', rowUuid: 'dtr', op: 'delete', baseVersion: 1, data: undefined });
    await adapter.enqueue({ collection: 'cards', rowUuid: 'dtr', op: 'upsert', baseVersion: 0, data: { title: 'recreated' } });

    const outbox = await adapter.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.op).toBe('upsert');
    // The local mirror row was dropped by the delete, so the recreate starts
    // at base 0 — but the pending entry must keep base 1 (the version the
    // client last confirmed) so the server LWW applies the recreate instead
    // of misreading base 0 as a create-retry and silently dropping it.
    expect(outbox[0]!.baseVersion).toBe(1);
  });

  it('cursor and meta persist', async () => {
    expect(await adapter.getCursor()).toBe(0);
    await adapter.setCursor(42);
    expect(await adapter.getCursor()).toBe(42);
    await adapter.setMeta('device_id', 'dev-1');
    expect(await adapter.getMeta('device_id')).toBe('dev-1');
  });

  it('transaction rolls back on throw', async () => {
    const before = (await adapter.allRows('cards')).length;
    await expect(
      adapter.transaction(async () => {
        await adapter.putRow('cards', { collection: 'cards', rowUuid: 'tx-row', version: 1, data: {} });
        throw new Error('boom');
      }),
    ).rejects.toThrow('boom');
    expect(await adapter.getRow('cards', 'tx-row')).toBeUndefined();
    expect(await adapter.allRows('cards')).toHaveLength(before);
  });
});
