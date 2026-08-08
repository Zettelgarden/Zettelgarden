import { describe, it, expect, vi } from 'vitest';
import { SyncEngine } from '../src/engine';
import { InMemoryAdapter } from '../src/storage';
import type { StorageAdapter } from '../src/storage';
import { MockServer } from './mock-server';
import type { Collection } from '../src/types';

function makeEngine(server: MockServer, deviceId: string): {
  engine: SyncEngine;
  storage: StorageAdapter;
} {
  const storage = new InMemoryAdapter();
  const engine = new SyncEngine({ storage, transport: server, deviceId });
  return { engine, storage };
}

function cardData(title: string, extra: Record<string, unknown> = {}): Record<string, unknown> {
  return { title, body: title, card_id: title, ...extra };
}

describe('SyncEngine', () => {
  it('bootstrap snapshots the mirror and sets the cursor', async () => {
    const server = new MockServer();
    server.seed('cards', 'c1', cardData('existing'));
    server.seed('tasks', 't1', { title: 'task' });

    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();

    expect(storage.getRow('cards', 'c1')?.data.title).toBe('existing');
    expect(storage.getRow('tasks', 't1')).toBeDefined();
    expect(storage.getCursor()).toBeGreaterThan(0);
    expect(engine.pendingChanges()).toBe(0);
  });

  it('local mutate queues the outbox and push drains it, adopting server identity', async () => {
    const server = new MockServer();
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();

    engine.mutate('cards', { rowUuid: 'new-card', data: cardData('offline card') });
    expect(engine.pendingChanges()).toBe(1);

    const summary = await engine.sync();
    expect(summary.pushed).toBe(1);
    expect(engine.pendingChanges()).toBe(0);

    const row = storage.getRow('cards', 'new-card')!;
    expect(row.version).toBe(1);
    expect(row.data.id).toBeGreaterThan(0); // server-assigned PK adopted
  });

  it('offline gap: mutations queue while offline and converge on reconnect', async () => {
    const server = new MockServer();
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();

    engine.setOnline(false);
    engine.mutate('cards', { rowUuid: 'a1', data: cardData('card A') });
    engine.mutate('tasks', { rowUuid: 't1', data: { title: 'task B' } });
    engine.deleteLocal('cards', 'does-not-exist'); // server will ignore this
    expect(engine.pendingChanges()).toBe(3);

    engine.setOnline(true);
    await engine.sync();
    expect(engine.pendingChanges()).toBe(0);
    expect(storage.getRow('cards', 'a1')?.data.title).toBe('card A');
    expect(storage.getRow('tasks', 't1')?.data.title).toBe('task B');
  });

  it('idempotent retry: a dropped response does not duplicate or regress', async () => {
    const server = new MockServer();
    const storage = new InMemoryAdapter();

    // Transport that applies the push server-side, then drops the response.
    const failingTransport = {
      snapshot: (c) => server.snapshot(c),
      changes: (s) => server.changes(s),
      push: async (req) => {
        await server.push(req);
        throw new Error('network dropped the response');
      },
    };
    const engine1 = new SyncEngine({ storage, transport: failingTransport, deviceId: 'dev-a' });
    await engine1.bootstrap();
    engine1.mutate('cards', { rowUuid: 'r1', data: cardData('retry me') });
    await expect(engine1.sync()).rejects.toThrow('network dropped');

    // The server applied the create, but the outbox entry is still pending.
    expect(engine1.pendingChanges()).toBe(1);
    const snap1 = await server.snapshot(['cards']);
    expect(snap1.collections.cards!).toHaveLength(1);

    // A fresh engine on the same storage retries the same entry (base 0).
    const engine2 = new SyncEngine({ storage, transport: server, deviceId: 'dev-a' });
    await engine2.sync();
    expect(engine2.pendingChanges()).toBe(0);
    const snap2 = await server.snapshot(['cards']);
    expect(snap2.collections.cards!).toHaveLength(1); // no duplicate row
    expect(storage.getRow('cards', 'r1')!.version).toBe(1);
  });

  it('self-echo: our own pushed change is not double-applied on the next pull', async () => {
    const server = new MockServer();
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();

    engine.mutate('cards', { rowUuid: 'e1', data: cardData('echo') });
    await engine.sync(); // push lands, log entry emitted

    const cursorBefore = storage.getCursor();
    await engine.sync(); // pull echoes our own entry
    expect(storage.getCursor()).toBeGreaterThanOrEqual(cursorBefore);

    const row = storage.getRow('cards', 'e1')!;
    expect(row.data.title).toBe('echo'); // data intact
    expect(row.version).toBe(1); // not double-bumped
    expect(storage.allRows('cards').filter((r) => r.rowUuid === 'e1')).toHaveLength(1);
    expect(engine.pendingChanges()).toBe(0);
  });

  it('two devices converge through the server (cross-device edit)', async () => {
    const server = new MockServer();
    const devA = makeEngine(server, 'dev-a');
    const devB = makeEngine(server, 'dev-b');
    await devA.engine.bootstrap();
    await devB.engine.bootstrap();

    // A creates the card and syncs; B pulls it.
    devA.engine.mutate('cards', { rowUuid: 'shared', data: cardData('A v1') });
    await devA.engine.sync();
    await devB.engine.sync();
    expect(devB.storage.getRow('cards', 'shared')!.data.title).toBe('A v1');

    // Both edit the SAME row from the same base (v1).
    devA.engine.mutate('cards', { rowUuid: 'shared', data: cardData('A v2') });
    devB.engine.mutate('cards', { rowUuid: 'shared', data: cardData('B v2') });
    await devA.engine.sync(); // A lands first -> v2
    await devB.engine.sync(); // B is stale -> adopts A's v2

    const serverRow = (await server.snapshot(['cards'])).collections.cards!.find(
      (r) => r.row_uuid === 'shared',
    )!;
    expect(serverRow.data?.title).toBe('A v2');
    expect(devA.storage.getRow('cards', 'shared')!.data.title).toBe('A v2');
    expect(devB.storage.getRow('cards', 'shared')!.data.title).toBe('A v2');
    expect(devA.storage.getRow('cards', 'shared')!.version).toBe(
      devB.storage.getRow('cards', 'shared')!.version,
    );
  });

  it('conflict on stale base adopts the server row and counts the lost edit', async () => {
    const server = new MockServer();
    const devA = makeEngine(server, 'dev-a');
    const devB = makeEngine(server, 'dev-b');
    await devA.engine.bootstrap();
    await devB.engine.bootstrap();

    // A creates the card and syncs.
    devA.engine.mutate('cards', { rowUuid: 'c1', data: cardData('A v1') });
    await devA.engine.sync();

    // B pulls it, then edits from base v1 -> v2.
    await devB.engine.sync();
    devB.engine.mutate('cards', { rowUuid: 'c1', data: cardData('B v2') });
    await devB.engine.sync();

    // A edits from its stale base v1 (server is now v2) -> conflict; A adopts.
    devA.engine.mutate('cards', { rowUuid: 'c1', data: cardData('A stale') });
    const summary = await devA.engine.sync();
    expect(summary.conflicts).toBe(1);
    expect(summary.lostEdits).toBe(1);

    expect(devA.storage.getRow('cards', 'c1')!.data.title).toBe('B v2');
    expect(devA.engine.pendingChanges()).toBe(0);
  });

  it('tag name-merge: second device adopts the surviving row_uuid', async () => {
    const server = new MockServer();
    const devA = makeEngine(server, 'dev-a');
    const devB = makeEngine(server, 'dev-b');
    await devA.engine.bootstrap();
    await devB.engine.bootstrap();

    devA.engine.mutate('tags', { rowUuid: 'tag-a', data: { name: 'Work', color: 'black' } });
    devA.engine.mutate('cards', { rowUuid: 'card-a', data: { ...cardData('A'), tags: ['Work'] } });
    await devA.engine.sync();

    // B's offline edit differs (red vs black): the merge discards it, so the
    // engine must surface the lost edit (v5b.6).
    devB.engine.mutate('tags', { rowUuid: 'tag-b', data: { name: 'Work', color: 'red' } });
    const summary = await devB.engine.sync();
    expect(summary.conflicts).toBe(0);
    expect(summary.lostEdits).toBe(1);
    // B's local tag row now uses the surviving uuid (tag-a).
    expect(devB.storage.getRow('tags', 'tag-a')).toBeDefined();
    expect(devB.storage.getRow('tags', 'tag-b')).toBeUndefined();
    expect(devB.storage.getRow('tags', 'tag-a')!.version).toBe(1);

    // Exactly one server tag.
    const snap = await server.snapshot(['tags']);
    expect(snap.collections.tags!).toHaveLength(1);
  });

  it('tag name-merge with identical data reports no lost edit', async () => {
    const server = new MockServer();
    const devA = makeEngine(server, 'dev-a');
    const devB = makeEngine(server, 'dev-b');
    await devA.engine.bootstrap();
    await devB.engine.bootstrap();

    devA.engine.mutate('tags', { rowUuid: 'tag-x', data: { name: 'Home', color: 'green' } });
    await devA.engine.sync();

    // B pushes the same name AND the same color: nothing is discarded.
    devB.engine.mutate('tags', { rowUuid: 'tag-y', data: { name: 'Home', color: 'green' } });
    const summary = await devB.engine.sync();
    expect(summary.lostEdits).toBe(0);
    expect(devB.storage.getRow('tags', 'tag-x')).toBeDefined();
    expect(devB.storage.getRow('tags', 'tag-y')).toBeUndefined();
  });

  it('delete propagates to the other device', async () => {
    const server = new MockServer();
    const devA = makeEngine(server, 'dev-a');
    const devB = makeEngine(server, 'dev-b');
    await devA.engine.bootstrap();
    await devB.engine.bootstrap();

    devA.engine.mutate('cards', { rowUuid: 'gone', data: cardData('doomed') });
    await devA.engine.sync();
    await devB.engine.sync();
    expect(devB.storage.getRow('cards', 'gone')).toBeDefined();

    devA.engine.deleteLocal('cards', 'gone');
    await devA.engine.sync();
    await devB.engine.sync();

    expect(devB.storage.getRow('cards', 'gone')).toBeUndefined();
  });

  it('delete-then-recreate offline converges (coalesced outbox keeps original base)', async () => {
    const server = new MockServer();
    server.seed('cards', 'dtr', cardData('v1'));
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();
    expect(storage.getRow('cards', 'dtr')!.version).toBe(1);

    // Offline: delete the row, then re-create the same rowUuid.
    engine.setOnline(false);
    engine.deleteLocal('cards', 'dtr');
    engine.mutate('cards', { rowUuid: 'dtr', data: cardData('recreated') });

    // The outbox coalesced to ONE upsert entry that keeps the original base
    // version (1), not the recreate's local base 0.
    const outbox = storage.outbox();
    expect(outbox).toHaveLength(1);
    expect(outbox[0]!.op).toBe('upsert');
    expect(outbox[0]!.baseVersion).toBe(1);

    engine.setOnline(true);
    const summary = await engine.sync();
    expect(summary.conflicts).toBe(0);
    expect(summary.lostEdits).toBe(0);
    expect(engine.pendingChanges()).toBe(0);

    // Server row is active with the recreate's data.
    const serverRow = (await server.snapshot(['cards'])).collections.cards!.find(
      (r) => r.row_uuid === 'dtr',
    )!;
    expect(serverRow).toBeDefined();
    expect(serverRow.data?.title).toBe('recreated');

    // The client mirror converges too.
    const row = storage.getRow('cards', 'dtr')!;
    expect(row.data.title).toBe('recreated');
    expect(row.version).toBeGreaterThanOrEqual(2);
  });

  it('auto-sync backs off exponentially after repeated failures', async () => {
    vi.useFakeTimers();
    try {
      const server = new MockServer();
      const storage = new InMemoryAdapter();
      const delays: number[] = [];
      const origSetTimeout = globalThis.setTimeout;
      vi.spyOn(globalThis, 'setTimeout').mockImplementation((fn: () => void, delay?: number, ...args: unknown[]) => {
        delays.push(delay ?? 0);
        return origSetTimeout(fn, delay, ...args) as ReturnType<typeof setTimeout>;
      });

      // Transport that fails on every changes/push after bootstrap.
      const failingTransport = {
        snapshot: (c: Collection[]) => server.snapshot(c),
        changes: async () => {
          throw new Error('network down');
        },
        push: async () => {
          throw new Error('network down');
        },
      };
      const engine = new SyncEngine({ storage, transport: failingTransport, deviceId: 'dev-a' });
      await engine.bootstrap();

      engine.start(1000);
      expect(delays[0]).toBe(1000); // steady state = interval

      // First cycle fails -> backoff doubles to 2000.
      await vi.advanceTimersByTimeAsync(1000);
      expect(delays[1]).toBe(2000);

      // Second failure -> 4000; a 2000ms wait must NOT fire a sync.
      await vi.advanceTimersByTimeAsync(2000);
      expect(delays[2]).toBe(4000);
      engine.stop();

      // Backoff is capped at maxBackoffMs.
      const capped = new SyncEngine({ storage, transport: failingTransport, deviceId: 'dev-a', maxBackoffMs: 2500 });
      capped.start(1000);
      await vi.advanceTimersByTimeAsync(1000); // fail -> backoff min(2000,2500)
      await vi.advanceTimersByTimeAsync(2000); // fail -> backoff min(4000,2500)=2500
      const cappedDelays = delays.length;
      await vi.advanceTimersByTimeAsync(2500);
      expect(delays.length).toBe(cappedDelays + 1);
      expect(delays[delays.length - 1]).toBeLessThanOrEqual(2500);
      capped.stop();
    } finally {
      vi.useRealTimers();
    }
  });

  it('re-bootstraps when the server reports a reset (pruned feed)', async () => {
    const server = new MockServer();
    server.seed('cards', 'c1', cardData('v1'));
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();
    expect(storage.getRow('cards', 'c1')!.data.title).toBe('v1');

    // The server pruned sync_log past our cursor and adds a row that only
    // exists in the snapshot: the next changes() answers reset, and the
    // engine must re-bootstrap instead of applying an impossible delta.
    server.seed('cards', 'c2', cardData('remote add'));
    let resetSent = false;
    const resettingTransport = {
      snapshot: (c: Collection[]) => server.snapshot(c),
      changes: async (since: number) => {
        if (!resetSent) {
          resetSent = true;
          return { cursor: since, rows: [], hasMore: false, reset: true };
        }
        return server.changes(since);
      },
      push: (req) => server.push(req),
    };
    const e2 = new SyncEngine({ storage, transport: resettingTransport, deviceId: 'dev-a' });
    const summary = await e2.sync();
    expect(summary.cursor).toBeGreaterThanOrEqual(storage.getCursor());
    // Re-bootstrapped: c2 (only present in the snapshot) is now mirrored.
    expect(storage.getRow('cards', 'c2')).toBeDefined();
    expect(storage.getRow('cards', 'c1')).toBeDefined();
  });

  it('edited create-retry reports a conflict instead of silently dropping the edit', async () => {
    const server = new MockServer();
    const storage = new InMemoryAdapter();
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
    engine1.mutate('cards', { rowUuid: 'r1', data: cardData('v1 title') });
    await expect(engine1.sync()).rejects.toThrow('network dropped');

    // The user edits before the retry: the outbox entry is still pending at
    // base 0 with the EDITED payload. LWW must surface a conflict + lost
    // edit, not a silent applied-no-write that clobbers the edit on pull.
    engine1.mutate('cards', { rowUuid: 'r1', data: cardData('edited title') });
    const engine2 = new SyncEngine({ storage, transport: server, deviceId: 'dev-a' });
    const summary = await engine2.sync();
    expect(summary.conflicts).toBe(1);
    expect(summary.lostEdits).toBe(1);
    expect(engine2.pendingChanges()).toBe(0);
    // Server row wins (LWW); the mirror adopts it.
    expect(storage.getRow('cards', 'r1')!.data.title).toBe('v1 title');
  });

  it('recreates a row after its delete synced (cross-batch) instead of conflicting it away', async () => {
    const server = new MockServer();
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();
    engine.mutate('cards', { rowUuid: 'dtr', data: cardData('v1') });
    await engine.sync();
    engine.deleteLocal('cards', 'dtr');
    await engine.sync();
    expect(storage.getRow('cards', 'dtr')).toBeUndefined();
    expect((await server.snapshot(['cards'])).collections.cards!).toHaveLength(0);

    // Recreate the SAME rowUuid: the mirror row is gone so the engine pushes
    // base 0 — the server must resurrect, not conflict the recreate away.
    engine.mutate('cards', { rowUuid: 'dtr', data: cardData('recreated') });
    const summary = await engine.sync();
    expect(summary.conflicts).toBe(0);
    expect(summary.lostEdits).toBe(0);
    const serverRow = (await server.snapshot(['cards'])).collections.cards!.find(
      (r) => r.row_uuid === 'dtr',
    )!;
    expect(serverRow.data?.title).toBe('recreated');
    expect(storage.getRow('cards', 'dtr')!.data.title).toBe('recreated');
  });

  it('re-bootstrap clears ghost rows the snapshot no longer has', async () => {
    const server = new MockServer();
    server.seed('cards', 'c1', cardData('v1'));
    server.seed('cards', 'ghost', cardData('doomed'));
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();
    expect(storage.getRow('cards', 'ghost')).toBeDefined();

    // The server pruned past our cursor and 'ghost' no longer exists in the
    // snapshot: reset forces a re-bootstrap that must DROP the ghost row.
    const resettingTransport = {
      snapshot: async (c: Collection[]) => ({
        cursor: 1,
        collections: { cards: [{ row_uuid: 'c1', version: 1, op: 'upsert', data: cardData('v1') }] },
      }),
      changes: async (since: number) => ({ cursor: since, rows: [], hasMore: false, reset: true }),
      push: (req) => server.push(req),
    };
    const e2 = new SyncEngine({ storage, transport: resettingTransport, deviceId: 'dev-a' });
    await e2.sync();
    expect(storage.getRow('cards', 'c1')).toBeDefined();
    expect(storage.getRow('cards', 'ghost')).toBeUndefined();
  });

  it('stop() halts the scheduler even with a sync in flight', async () => {
    vi.useFakeTimers();
    try {
      const server = new MockServer();
      const storage = new InMemoryAdapter();
      const delays: number[] = [];
      const origSetTimeout = globalThis.setTimeout;
      vi.spyOn(globalThis, 'setTimeout').mockImplementation((fn: () => void, delay?: number, ...args: unknown[]) => {
        delays.push(delay ?? 0);
        return origSetTimeout(fn, delay, ...args) as ReturnType<typeof setTimeout>;
      });

      const failingTransport = {
        snapshot: (c: Collection[]) => server.snapshot(c),
        changes: async () => {
          throw new Error('down');
        },
        push: async () => {
          throw new Error('down');
        },
      };
      const engine = new SyncEngine({ storage, transport: failingTransport, deviceId: 'dev-a' });
      await engine.bootstrap();
      engine.start(1000);
      await vi.advanceTimersByTimeAsync(1000); // first cycle fires + fails, backoff re-arms
      engine.stop();
      const countBefore = delays.length;
      await vi.advanceTimersByTimeAsync(100_000);
      expect(delays.length).toBe(countBefore); // no sync re-armed after stop()
    } finally {
      vi.useRealTimers();
    }
  });

  it('tag delete of a pre-merge uuid drains the outbox (no infinite re-push)', async () => {
    const server = new MockServer();
    server.seed('tags', 'tag-a', { name: 'Work', color: 'black' });
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();

    // Device holds a pre-merge uuid 'tag-b' for the same name and deletes it.
    engine.deleteLocal('tags', 'tag-b');
    const summary = await engine.sync();
    expect(summary.conflicts).toBe(0);
    expect(engine.pendingChanges()).toBe(0); // drained, no re-push loop
    expect(storage.getRow('tags', 'tag-b')).toBeUndefined();
  });

  it('progress events fire with pendingChanges and lastSynced', async () => {
    const server = new MockServer();
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();

    const seen: string[] = [];
    const off = engine.onProgress((p) => seen.push(`${p.state}:${p.pendingChanges}`));
    engine.mutate('cards', { rowUuid: 'p1', data: cardData('progress') });
    await engine.sync();
    off();

    expect(seen.some((s) => s.startsWith('idle:1'))).toBe(true);
    expect(storage.getRow('cards', 'p1')?.data.id).toBeGreaterThan(0);
  });

  it('cursor advances correctly across snapshot -> incremental pulls', async () => {
    const server = new MockServer();
    server.seed('cards', 'c1', cardData('seed'));
    const { engine, storage } = makeEngine(server, 'dev-a');
    await engine.bootstrap();
    const bootstrapCursor = storage.getCursor();

    // Remote change after bootstrap must appear on the next pull.
    server.seed('cards', 'c2', cardData('remote add'));
    await engine.sync();
    expect(storage.getRow('cards', 'c2')).toBeDefined();
    expect(storage.getCursor()).toBeGreaterThan(bootstrapCursor);

    // Nothing new -> pull is a no-op that returns the same cursor.
    const cursorBefore = storage.getCursor();
    await engine.sync();
    expect(storage.getCursor()).toBeGreaterThanOrEqual(cursorBefore);
  });
});
