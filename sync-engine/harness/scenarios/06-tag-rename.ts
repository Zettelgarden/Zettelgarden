/**
 * Scenario 6 — tag rename-vs-rename and rename-vs-create per the v1 policy
 * (LWW on the merged row by version; rename is a normal upsert of the new
 * name).
 *
 * rename-vs-rename: both devices rename the SAME tag row offline; LWW keeps
 * one name, the loser adopts the winner's row (lost edit counted) — one tag,
 * no split.
 *
 * rename-vs-create: A renames "work" away while B creates a NEW "work" tag
 * offline. The server keeps A's renamed row and B's fresh row as two live
 * tags; both devices converge to the identical set. (The v1 policy
 * refinement "renamed-away name soft-deleted" would have B's create
 * resurrect a tombstone instead — observable convergence is the same; the
 * tombstone is filed as a follow-up bead, see harness README.)
 */

import type { Scenario } from './context';
import { convergeAndAssert, pushSummary, settle, withDevices } from './context';

export const tagRenameScenario: Scenario = {
  name: '06 tag rename-vs-rename and rename-vs-create (v1 policy)',
  run: async ({ backend }) => {
    await withDevices(backend, 'tag-rename', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      // ---- rename-vs-rename -----------------------------------------------
      // Seed "work" through A; B bootstraps.
      await a.engine.bootstrap();
      const baselineTags = (await a.engine.query('tags')).length;
      await a.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'work', color: 'blue' } });
      await a.engine.sync();
      await b.engine.bootstrap();
      if ((await b.engine.getRow('tags', 'tag-w'))?.data.name !== 'work') {
        throw new Error('device B should bootstrap the seeded tag');
      }

      // Both rename the same row offline from base 1.
      a.engine.setOnline(false);
      await a.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'tasks', color: 'blue' } });
      b.engine.setOnline(false);
      await b.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'projects', color: 'blue' } });

      const summaries = await settle([a, b], 2);
      const bPush = pushSummary(summaries, 'dev-b');
      if (!bPush || bPush.lostEdits < 1) {
        throw new Error(`rename-vs-rename: expected B to lose, got ${JSON.stringify(bPush)}`);
      }
      const server = await convergeAndAssert('rename-vs-rename', [a, b], auth, baseUrl);
      const serverTags = server.collections.tags ?? [];
      if (serverTags.length !== baselineTags + 1) {
        throw new Error(`rename-vs-rename split the tag: server has ${serverTags.length} tags (expected ${baselineTags + 1})`);
      }
      const winRow = serverTags.find((r) => r.row_uuid === 'tag-w');
      if ((winRow?.data as { name?: string } | undefined)?.name !== 'tasks') {
        throw new Error(`rename-vs-rename: expected A's 'tasks' to win, got ${JSON.stringify(winRow?.data)}`);
      }
      for (const dev of [a, b]) {
        const tags = await dev.engine.query('tags');
        const mine = tags.filter((t) => t.rowUuid === 'tag-w');
        if (tags.length !== baselineTags + 1 || mine.length !== 1 || (mine[0]!.data as { name?: string }).name !== 'tasks') {
          throw new Error(`device ${dev.id} did not converge on the single winning tag`);
        }
      }

      // ---- rename-vs-create ------------------------------------------------
      // Fresh account: seed "work", then A renames it away while B creates a
      // brand-new "work" tag offline.
      await withDevices(backend, 'tag-rename-create', ['dev-a', 'dev-b'], async ([a2, b2], auth2, baseUrl2) => {
        await a2.engine.bootstrap();
        const baselineTags2 = (await a2.engine.query('tags')).length;
        await a2.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'work', color: 'blue' } });
        await a2.engine.sync();
        await b2.engine.bootstrap();

        a2.engine.setOnline(false);
        await a2.engine.mutate('tags', { rowUuid: 'tag-w', data: { name: 'tasks', color: 'blue' } });
        b2.engine.setOnline(false);
        await b2.engine.mutate('tags', { rowUuid: 'tag-w2', data: { name: 'work', color: 'blue' } });

        await settle([a2, b2], 2);
        const server2 = await convergeAndAssert('rename-vs-create', [a2, b2], auth2, baseUrl2);
        const tags2 = server2.collections.tags ?? [];
        const workTags = tags2.filter((t) => (t.data as { name?: string }).name === 'work');
        const tasksTags = tags2.filter((t) => (t.data as { name?: string }).name === 'tasks');
        // A's rename kept its row; B's create kept its row. Deterministic and
        // converged: exactly one live 'tasks' and one live 'work' each.
        if (tags2.length !== baselineTags2 + 2 || workTags.length !== 1 || tasksTags.length !== 1) {
          throw new Error(
            `rename-vs-create: expected exactly ${baselineTags2 + 2} tags with one 'work' and one 'tasks', got ${JSON.stringify(tags2.map((t) => (t.data as { name?: string }).name).sort())}`,
          );
        }
        for (const dev of [a2, b2]) {
          const local = (await dev.engine.query('tags')).map((t) => (t.data as { name?: string }).name);
          const lw = local.filter((n) => n === 'work').length;
          const lt = local.filter((n) => n === 'tasks').length;
          if (lw !== 1 || lt !== 1 || local.length !== baselineTags2 + 2) {
            throw new Error(`device ${dev.id} did not converge on both tags, got ${JSON.stringify(local.sort())}`);
          }
        }
      });
    });
  },
};
