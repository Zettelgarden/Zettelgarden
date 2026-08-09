/**
 * Scenario 10 — tag rename-vs-create WITH the renamed-away tombstone
 * (bead Zettelgarden-8g0).
 *
 * A renames "work" → "tasks"; the server keeps a soft-deleted tombstone row
 * for "work" (fresh uuid, never emitted). B — offline — creates a NEW "work"
 * tag; its push finds the tombstone and RESURRECTS it, so B's fresh uuid is
 * merged away and the "work" identity is stable across the rename+recreate
 * cycle (matching REST CreateTag). Before 8g0, B's create made a fresh row
 * and its uuid survived.
 *
 * Exit criterion: exactly one live "work" (the tombstone's uuid, NOT B's
 * fresh uuid) + one live "tasks", fully converged.
 */

import type { Scenario } from './context';
import { convergeAndAssert, settle, withDevices } from './context';

export const tagTombstoneScenario: Scenario = {
  name: '10 tag rename-vs-create resurrects the renamed-away tombstone (8g0)',
  run: async ({ backend }) => {
    await withDevices(backend, 'tag-tombstone', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      // Seed "work" through A; B bootstraps the shared row.
      await a.engine.bootstrap();
      const baselineTags = (await a.engine.query('tags')).length;
      await a.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'work', color: 'blue' } });
      await a.engine.sync();
      await b.engine.bootstrap();
      if ((await b.engine.getRow('tags', 'tag-w'))?.data.name !== 'work') {
        throw new Error('device B should bootstrap the seeded tag');
      }

      // A renames "work" → "tasks" offline; B creates a NEW "work" offline.
      a.engine.setOnline(false);
      await a.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'tasks', color: 'blue' } });
      b.engine.setOnline(false);
      await b.engine.mutate('tags', { rowUuid: 'tag-b-fresh', data: { name: 'work', color: 'blue' } });

      await settle([a, b], 2);
      const server = await convergeAndAssert('rename-vs-create with tombstone', [a, b], auth, baseUrl);
      const tags = server.collections.tags ?? [];
      const workRows = tags.filter((t) => (t.data as { name?: string }).name === 'work');
      const tasksRows = tags.filter((t) => (t.data as { name?: string }).name === 'tasks');

      if (tags.length !== baselineTags + 2 || workRows.length !== 1 || tasksRows.length !== 1) {
        throw new Error(
          `expected exactly ${baselineTags + 2} tags with one 'work' and one 'tasks', got ${JSON.stringify(tags.map((t) => (t.data as { name?: string }).name).sort())}`,
        );
      }
      // THE 8g0 ASSERTION: B's fresh uuid must NOT survive — B's create
      // resurrected the renamed-away tombstone, so the live 'work' row is the
      // tombstone's uuid, and B's local fresh row was merged onto it.
      if (tags.some((t) => t.row_uuid === 'tag-b-fresh')) {
        throw new Error('B\'s fresh uuid survived: the create made a new row instead of resurrecting the tombstone');
      }
      for (const dev of [a, b]) {
        const local = await dev.engine.query('tags');
        if (local.some((t) => t.rowUuid === 'tag-b-fresh')) {
          throw new Error(`device ${dev.id} still holds B's fresh uuid after the merge`);
        }
        const names = local.map((t) => (t.data as { name?: string }).name);
        if (local.length !== baselineTags + 2 || names.filter((n) => n === 'work').length !== 1 || names.filter((n) => n === 'tasks').length !== 1) {
          throw new Error(`device ${dev.id} did not converge on both tags, got ${JSON.stringify(names.sort())}`);
        }
      }
    });
  },
};
