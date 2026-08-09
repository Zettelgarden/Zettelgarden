/**
 * SyncDataProvider tests (Zettelgarden-fv3, Phase 2b): the desktop data-layer
 * swap's offline semantics — temp ids + alias map, mirror reads, outbox
 * writes, task FK translation (card_pk -> card_pk_uuid) — against an
 * in-memory engine + fake transport. No Tauri runtime needed.
 */

import { describe, it, expect } from 'vitest';
import { SyncEngine } from '@zettelgarden/sync-engine/engine';
import { InMemoryAdapter } from '@zettelgarden/sync-engine/storage';
import type {
  SyncTransport,
  SnapshotResponse,
  ChangesResponse,
  PushResponse,
} from '@zettelgarden/sync-engine/types';
import { SyncDataProvider } from '../data/syncProvider';
import { emptyTask } from '../models/Task';

let idCounter = 1000;

/** Fake transport: snapshot of a server with seeded rows; push assigns ids
 * and returns applied results (minimal, no LWW). */
function fakeTransport(
  seed: Record<string, Record<string, unknown>[]> = {},
): SyncTransport & {
  serverRows: () => Record<string, unknown>[];
} {
  const server: Record<
    string,
    { rowUuid: string; version: number; data: Record<string, unknown> }[]
  > = {};
  for (const [collection, rows] of Object.entries(seed)) {
    server[collection] = rows.map((data) => ({
      rowUuid: data.sync_uuid as string,
      version: 1,
      data,
    }));
  }
  let cursor = 1;

  return {
    serverRows: () =>
      Object.values(server)
        .flat()
        .map((r) => ({ ...r.data, sync_uuid: r.rowUuid })),
    async snapshot(): Promise<SnapshotResponse> {
      const collections = {} as SnapshotResponse['collections'];
      for (const [c, rows] of Object.entries(server)) {
        collections[c as keyof typeof collections] = rows.map((r) => ({
          row_uuid: r.rowUuid,
          version: r.version,
          op: 'upsert' as const,
          data: r.data,
        }));
      }
      return { cursor, collections };
    },
    async changes(): Promise<ChangesResponse> {
      return { cursor, rows: [], hasMore: false };
    },
    async push(
      req: PushResponse extends never ? never : any,
    ): Promise<PushResponse> {
      const results: PushResponse['results'] = [];
      for (const change of req.changes) {
        if (change.op === 'delete') {
          const i = server[change.collection]?.findIndex(
            (r) => r.rowUuid === change.row_uuid,
          );
          if (i !== undefined && i >= 0)
            server[change.collection]?.splice(i, 1);
          results.push({
            rowUuid: change.row_uuid,
            status: 'applied',
            serverVersion: change.base_version + 1,
          });
        } else {
          const id = ++idCounter;
          const data = {
            ...(change.data ?? {}),
            id,
            version: change.base_version + 1,
          };
          (server[change.collection] ??= []).push({
            rowUuid: change.row_uuid,
            version: change.base_version + 1,
            data,
          });
          results.push({
            rowUuid: change.row_uuid,
            status: 'applied',
            serverId: id,
            serverVersion: change.base_version + 1,
            data,
          });
        }
      }
      cursor++;
      return { results, cursor, lostEdits: 0 };
    },
  };
}

function makeProvider() {
  const storage = new InMemoryAdapter();
  const transport = fakeTransport();
  const engine = new SyncEngine({ storage, transport, deviceId: 'test-dev' });
  const provider = new SyncDataProvider(engine);
  return { storage, transport, engine, provider };
}

function cardData(overrides: Record<string, unknown> = {}) {
  return {
    title: 't',
    body: 'b',
    card_id: 't',
    link: '',
    is_deleted: false,
    ...overrides,
  };
}

describe('SyncDataProvider (offline data layer)', () => {
  it('saveNewCard assigns a negative temp id and writes the mirror + outbox', async () => {
    const { provider, engine, storage } = makeProvider();
    const saved = await provider.saveNewCard({
      ...(cardData() as any),
      id: -1,
    } as any);
    expect(saved.id).toBeLessThan(0); // temp id
    expect(await engine.pendingChanges()).toBe(1);

    const row =
      (await storage.getRow('cards', 'x')) ??
      (await storage.allRows('cards'))[0];
    expect(row!.data.title).toBe('t');
  });

  it('getCard resolves temp ids via the alias map and returns the card', async () => {
    const { provider } = makeProvider();
    const saved = await provider.saveNewCard(cardData() as any);
    const card = await provider.getCard(String(saved.id));
    expect(card.title).toBe('t');
    expect(card.id).toBe(saved.id);
  });

  it('saveExistingCard on a temp-id card updates the same row (no dupes)', async () => {
    const { provider, engine, storage } = makeProvider();
    const saved = await provider.saveNewCard(cardData() as any);
    await provider.saveExistingCard({
      ...saved,
      title: 'edited',
      body: 'b2',
    } as any);
    expect(await engine.pendingChanges()).toBe(1); // coalesced, not 2

    const row = (await storage.allRows('cards'))[0];
    expect(row!.data.title).toBe('edited');
  });

  it('sync adopts the server id but temp-id URLs keep resolving', async () => {
    const { provider, engine } = makeProvider();
    const saved = await provider.saveNewCard(cardData() as any);
    await engine.sync();

    // The mirror row now carries the real server id…
    const card = await provider.getCard(String(saved.id));
    expect(card.id).toBeGreaterThan(0);
    // …and the old temp-id URL still resolves through the alias map.
    const again = await provider.getCard(String(saved.id));
    expect(again.id).toBeGreaterThan(0);
    expect(again.title).toBe('t');
    expect(await engine.pendingChanges()).toBe(0);
  });

  it('saveNewTask translates card_pk into card_pk_uuid for offline FK resolution', async () => {
    const { provider, storage } = makeProvider();
    const card = await provider.saveNewCard(cardData() as any);
    const task = {
      ...emptyTask,
      title: 'offline task',
      card_pk: card.id as number,
    };
    const savedTask = await provider.saveNewTask(task);

    expect(savedTask.id).toBeLessThan(0);
    const row = (await storage.allRows('tasks'))[0];
    expect(row!.data.card_pk_uuid).toBeTruthy();
    expect(row!.data.card_pk).toBeUndefined(); // never push raw ints
    expect(row!.data.title).toBe('offline task');
  });

  it('offline task appears on its offline card via card_pk_uuid', async () => {
    const { provider } = makeProvider();
    const card = await provider.saveNewCard(cardData() as any);
    await provider.saveNewTask({
      ...emptyTask,
      title: 'offline task on offline card',
      card_pk: card.id as number,
    } as any);

    const loaded = await provider.getCard(String(card.id));
    expect(loaded.tasks).toHaveLength(1);
    expect(loaded.tasks[0].title).toBe('offline task on offline card');
    const tasks = await provider.getCardTasks(String(card.id));
    expect(tasks.map((t: any) => t.title)).toEqual([
      'offline task on offline card',
    ]);
  });

  it('fetchTasks filters the mirror offline', async () => {
    const { provider, engine } = makeProvider();
    const t1 = await provider.saveNewTask({
      ...emptyTask,
      title: 'open',
      is_complete: false,
    } as any);
    await provider.saveNewTask({
      ...emptyTask,
      title: 'done',
      is_complete: true,
    } as any);

    const open = await provider.fetchTasks({ showCompleted: false });
    expect(open.map((t) => t.id)).toEqual([t1.id]);
    const all = await provider.fetchTasks({ showCompleted: true });
    expect(all).toHaveLength(2);
    await engine.sync(); // drains; no throw
  });

  it('deleteCard queues a local delete and drops the row from the mirror', async () => {
    const { provider, engine, storage } = makeProvider();
    const saved = await provider.saveNewCard(cardData() as any);
    await provider.deleteCard(saved.id as number);
    expect(await engine.pendingChanges()).toBe(1);
    expect(await storage.allRows('cards')).toHaveLength(0);
  });

  it('fetchUserTags lists non-deleted tags sorted by name', async () => {
    const { provider, storage } = makeProvider();
    await storage.putRow('tags', {
      collection: 'tags',
      rowUuid: 't1',
      version: 1,
      data: { name: 'zeta', color: 'black', is_deleted: false },
    });
    await storage.putRow('tags', {
      collection: 'tags',
      rowUuid: 't2',
      version: 1,
      data: { name: 'alpha', color: 'red', is_deleted: false },
    });
    await storage.putRow('tags', {
      collection: 'tags',
      rowUuid: 't3',
      version: 1,
      data: { name: 'gone', color: 'blue', is_deleted: true },
    });
    const tags = await provider.fetchUserTags();
    expect(tags.map((t) => t.name)).toEqual(['alpha', 'zeta']);
  });
});
