import { describe, it, expect, beforeAll } from 'vitest';
import { SqliteStorageAdapter } from '../src/sqlite';
import type { OutboxEntry } from '../src/types';

let adapter: SqliteStorageAdapter;

beforeAll(async () => {
  adapter = new SqliteStorageAdapter(':memory:');
  await adapter.whenReady();
});

describe('SqliteStorageAdapter', () => {
  it('mirror CRUD round-trips', () => {
    adapter.putRow('cards', {
      collection: 'cards',
      rowUuid: 'r1',
      version: 3,
      data: { id: 5, title: 'hello', body: 'world' },
    });
    const row = adapter.getRow('cards', 'r1');
    expect(row).toBeDefined();
    expect(row!.version).toBe(3);
    expect(row!.data.title).toBe('hello');
    expect(row!.data.id).toBe(5);

    adapter.putRow('cards', {
      collection: 'cards',
      rowUuid: 'r1',
      version: 4,
      data: { id: 5, title: 'updated' },
    });
    expect(adapter.getRow('cards', 'r1')!.version).toBe(4);

    adapter.deleteRow('cards', 'r1');
    expect(adapter.getRow('cards', 'r1')).toBeUndefined();
  });

  it('collections are isolated', () => {
    adapter.putRow('cards', { collection: 'cards', rowUuid: 'c1', version: 1, data: {} });
    adapter.putRow('tasks', { collection: 'tasks', rowUuid: 't1', version: 1, data: {} });
    expect(adapter.allRows('cards')).toHaveLength(1);
    expect(adapter.allRows('tasks')).toHaveLength(1);
  });

  it('outbox coalesces by rowUuid, keeping the original base_version', () => {
    adapter.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'a' } });
    adapter.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'b' } });
    adapter.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'c' } });

    const outbox = adapter.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.data?.title).toBe('c');
    expect(outbox[0]!.baseVersion).toBe(4);
    expect(adapter.hasPending('c1')).toBe(true);

    adapter.dropOutbox('c1');
    expect(adapter.outbox()).toHaveLength(0);
  });

  it('delete-then-recreate coalesces to an upsert that keeps the original base', () => {
    adapter.enqueue({ collection: 'cards', rowUuid: 'dtr', op: 'delete', baseVersion: 1, data: undefined });
    adapter.enqueue({ collection: 'cards', rowUuid: 'dtr', op: 'upsert', baseVersion: 0, data: { title: 'recreated' } });

    const outbox = adapter.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.op).toBe('upsert');
    // The local mirror row was dropped by the delete, so the recreate starts
    // at base 0 — but the pending entry must keep base 1 (the version the
    // client last confirmed) so the server LWW applies the recreate instead
    // of misreading base 0 as a create-retry and silently dropping it.
    expect(outbox[0]!.baseVersion).toBe(1);
  });

  it('cursor and meta persist', () => {
    expect(adapter.getCursor()).toBe(0);
    adapter.setCursor(42);
    expect(adapter.getCursor()).toBe(42);
    adapter.setMeta('device_id', 'dev-1');
    expect(adapter.getMeta('device_id')).toBe('dev-1');
  });

  it('transaction rolls back on throw', () => {
    const before = adapter.allRows('cards').length;
    expect(() =>
      adapter.transaction(() => {
        adapter.putRow('cards', { collection: 'cards', rowUuid: 'tx-row', version: 1, data: {} });
        throw new Error('boom');
      }),
    ).toThrow('boom');
    expect(adapter.getRow('cards', 'tx-row')).toBeUndefined();
    expect(adapter.allRows('cards')).toHaveLength(before);
  });
});
