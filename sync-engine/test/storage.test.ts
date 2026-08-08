import { describe, it, expect } from 'vitest';
import { InMemoryAdapter } from '../src/storage';

// InMemoryAdapter is the engine's test double, but it must stay faithful to
// SqliteStorageAdapter's coalescing contract (v5b.2): a pending entry for the
// same row_uuid is replaced by op/data while the ORIGINAL base_version is
// kept. If it ever diverges (e.g. base reset to 0 on delete-then-recreate),
// the engine tests stop exercising the real LWW path and a future adapter
// copying the pattern silently loses data.
describe('InMemoryAdapter', () => {
  it('outbox coalesces by rowUuid, keeping the original base_version', () => {
    const a = new InMemoryAdapter();
    a.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'a' } });
    a.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'b' } });
    a.enqueue({ collection: 'cards', rowUuid: 'c1', op: 'upsert', baseVersion: 4, data: { title: 'c' } });

    const outbox = a.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.data?.title).toBe('c');
    expect(outbox[0]!.baseVersion).toBe(4);
    expect(a.hasPending('c1')).toBe(true);

    a.dropOutbox('c1');
    expect(a.outbox()).toHaveLength(0);
  });

  it('delete-then-recreate coalesces to an upsert that keeps the original base', () => {
    const a = new InMemoryAdapter();
    a.enqueue({ collection: 'cards', rowUuid: 'dtr', op: 'delete', baseVersion: 1, data: undefined });
    a.enqueue({ collection: 'cards', rowUuid: 'dtr', op: 'upsert', baseVersion: 0, data: { title: 'recreated' } });

    const outbox = a.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.op).toBe('upsert');
    // The local mirror row was dropped by the delete, so the recreate starts
    // at base 0 — but the pending entry must keep base 1 (the version the
    // client last confirmed) so the server LWW applies the recreate instead
    // of misreading base 0 as a create-retry and silently dropping it.
    expect(outbox[0]!.baseVersion).toBe(1);
  });

  it('transaction boundary is synchronous and atomic in order', () => {
    const a = new InMemoryAdapter();
    a.transaction(() => {
      a.putRow('cards', { collection: 'cards', rowUuid: 't1', version: 1, data: {} });
      a.enqueue({ collection: 'cards', rowUuid: 't1', op: 'upsert', baseVersion: 0, data: {} });
    });
    expect(a.getRow('cards', 't1')).toBeDefined();
    expect(a.outbox()).toHaveLength(1);
  });
});
