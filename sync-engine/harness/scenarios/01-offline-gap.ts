/**
 * Scenario 1 — offline gaps: device A edits while offline, device B edits
 * while offline, both reconnect and sync; every change lands on the server
 * and both mirrors converge to the server's state.
 */

import type { Scenario } from './context';
import { convergeAndAssert, settle, withDevices } from './context';

function card(title: string, cardId: string) {
  return { title, card_id: cardId, body: title };
}

export const offlineGapScenario: Scenario = {
  name: '01 offline gaps: A and B edit while offline, both converge on reconnect',
  run: async ({ backend }) => {
    await withDevices(backend, 'offline-gap', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      await a.engine.bootstrap();
      await b.engine.bootstrap();
      // New accounts seed a welcome card + 5 default tags; record the baseline
      // so expectations stay correct if the seed changes.
      const baselineCards = a.engine.query('cards').length;
      const baselineTags = a.engine.query('tags').length;

      // Both go offline and make unrelated edits.
      a.engine.setOnline(false);
      a.engine.mutate('cards', { rowUuid: 'card-a', data: card('card from A', 'a1') });
      a.engine.mutate('tasks', { rowUuid: 'task-a', data: { title: 'task from A' } });
      a.engine.mutate('tags', { rowUuid: 'tag-a', data: { name: 'work-a', color: 'blue' } });
      if (a.engine.pendingChanges() !== 3) {
        throw new Error(`device A should queue 3 offline changes, has ${a.engine.pendingChanges()}`);
      }

      b.engine.setOnline(false);
      b.engine.mutate('cards', { rowUuid: 'card-b', data: card('card from B', 'b1') });
      b.engine.mutate('tags', { rowUuid: 'tag-b', data: { name: 'home-b', color: 'red' } });

      // Reconnect: A first, then B, then settle.
      await settle([a, b], 2);

      const server = await convergeAndAssert('offline gaps', [a, b], auth, baseUrl);

      // Spot-check that every offline edit landed. New accounts start with a
      // welcome card + default tags (CreateUser seeds them), so totals are
      // baseline + our rows.
      const serverCards = server.collections.cards ?? [];
      const serverTasks = server.collections.tasks ?? [];
      const serverTags = server.collections.tags ?? [];
      if (serverCards.length !== baselineCards + 2 || serverTasks.length !== 1 || serverTags.length !== baselineTags + 2) {
        throw new Error(
          `expected ${baselineCards + 2} cards, 1 task, ${baselineTags + 2} tags; got ${serverCards.length}, ${serverTasks.length}, ${serverTags.length}`,
        );
      }
      const byUuid = (rows: typeof serverCards, uuid: string) => rows.find((r) => r.row_uuid === uuid);
      if (!byUuid(serverCards, 'card-a') || !byUuid(serverCards, 'card-b')) {
        throw new Error('server missing an offline-created card');
      }
      if (!byUuid(serverTasks, 'task-a')) throw new Error('server missing the offline task');
      if (!byUuid(serverTags, 'tag-a') || !byUuid(serverTags, 'tag-b')) {
        throw new Error('server missing an offline-created tag');
      }
      if (a.engine.getRow('cards', 'card-b')?.data.title !== 'card from B') {
        throw new Error('device A should have converged on device B\'s card');
      }
      if (b.engine.getRow('cards', 'card-a')?.data.title !== 'card from A') {
        throw new Error('device B should have converged on device A\'s card');
      }
    });
  },
};
