/**
 * Scenario 3 — offline-created linked rows: a card is created offline on A, a
 * task linked to it via tasks.card_pk is created offline on B, and the
 * same-named tag is created offline on BOTH devices. The server must end with
 * ONE card, ONE task whose card_pk points at the card, and ONE tag — with both
 * devices' references remapped to the surviving tag row.
 */

import type { Scenario } from './context';
import { convergeAndAssert, pushSummary, settle, withDevices } from './context';

export const linkedRowsScenario: Scenario = {
  name: '03 offline-created linked rows: card + task via card_pk + same-named tag merge',
  run: async ({ backend }) => {
    await withDevices(backend, 'linked-rows', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      await a.engine.bootstrap();
      await b.engine.bootstrap();
      const baselineCards = a.engine.query('cards').length;
      const baselineTags = a.engine.query('tags').length;

      // A goes offline: creates the card (body tags it #work) and the tag.
      a.engine.setOnline(false);
      a.engine.mutate('cards', {
        rowUuid: 'card-live',
        data: { card_id: 'live', title: 'live card', body: 'notes #work' },
      });
      a.engine.mutate('tags', { rowUuid: 'tag-a', data: { name: 'work', color: 'blue' } });

      // B goes offline: creates the tag (same name, different color) and a
      // task linked to A's not-yet-created card via its sync_uuid.
      b.engine.setOnline(false);
      b.engine.mutate('tags', { rowUuid: 'tag-b', data: { name: 'work', color: 'red' } });
      b.engine.mutate('tasks', {
        rowUuid: 'task-off',
        data: {
          title: 'offline task',
          card_pk_uuid: 'card-live',
          status: 'todo',
        },
      });

      // A pushes (card + tag in one batch; the server derives the tag from the
      // card body and merges A's tag onto it), then B pushes (task FK resolved
      // to the card's server PK; B's tag merges onto the surviving row).
      const summaries = await settle([a, b], 2);
      const bPush = pushSummary(summaries, 'dev-b');
      if (!bPush || bPush.lostEdits < 1) {
        throw new Error(`expected device B to lose its differing tag color, got ${JSON.stringify(bPush)}`);
      }

      const server = await convergeAndAssert('offline linked rows', [a, b], auth, baseUrl);

      const serverCards = server.collections.cards ?? [];
      const serverTasks = server.collections.tasks ?? [];
      const serverTags = server.collections.tags ?? [];

      // Exactly one of each NEW thing, linked. (Welcome card + default tags
      // are seeded per account, so totals are baseline + ours.)
      if (serverCards.length !== baselineCards + 1 || serverTasks.length !== 1 || serverTags.length !== baselineTags + 1) {
        throw new Error(
          `expected ${baselineCards + 1} cards, 1 task, ${baselineTags + 1} tags; got ${serverCards.length}, ${serverTasks.length}, ${serverTags.length}`,
        );
      }
      const cardRow = serverCards.find((r) => r.row_uuid === 'card-live');
      const taskRow = serverTasks[0]!;
      const workTags = serverTags.filter((r) => (r.data as { name?: string }).name === 'work');
      const tagRow = workTags[0];
      if (!cardRow || workTags.length !== 1 || !tagRow) {
        throw new Error(`expected exactly one 'work' tag, got ${JSON.stringify(serverTags.map((t) => (t.data as { name?: string }).name))}`);
      }
      const cardId = cardRow.data?.id;
      const taskCardPk = taskRow.data?.card_pk;
      if (typeof cardId !== 'number' || taskCardPk !== cardId) {
        throw new Error(`task.card_pk=${taskCardPk} must equal the card's server id ${cardId}`);
      }
      if ((tagRow.data as { name?: string }).name !== 'work') {
        throw new Error(`expected a single 'work' tag, got ${JSON.stringify(tagRow.data)}`);
      }

      // Both devices must reference the surviving tag uuid (no duplicate
      // push, no stale tag-b row), and both must have the server-form task
      // (card_pk int, not the client's card_pk_uuid placeholder).
      const survivingTagUuid = tagRow.row_uuid;
      for (const dev of [a, b]) {
        if (dev.engine.getRow('tags', 'tag-b') || dev.engine.getRow('tags', 'tag-a')) {
          throw new Error(`device ${dev.id} still holds a pre-merge tag uuid`);
        }
        const merged = dev.engine.getRow('tags', survivingTagUuid);
        if (!merged || merged.data.name !== 'work') {
          throw new Error(`device ${dev.id} does not hold the surviving tag ${survivingTagUuid}`);
        }
        const task = dev.engine.getRow('tasks', 'task-off');
        if (task?.data.card_pk !== cardId) {
          throw new Error(`device ${dev.id} task.card_pk=${task?.data.card_pk} not remapped to ${cardId}`);
        }
      }
    });
  },
};
