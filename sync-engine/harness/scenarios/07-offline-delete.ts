/**
 * Scenario 7 — offline delete vs concurrent edit: A deletes a card offline
 * while B edits the same card offline. The server's LWW keeps the DELETE;
 * B's push conflicts and must adopt the server's deleted row — which means
 * DROPPING the row locally, not keeping a ghost. (The feed tombstone was
 * skipped while B's edit was pending, so nothing else heals it — this is
 * the is_deleted 0/1-vs-boolean bug class found in review.)
 */

import type { Scenario } from './context';
import { convergeAndAssert, settle, withDevices } from './context';

function card(title: string, cardId: string) {
  return { title, card_id: cardId, body: title };
}

export const offlineDeleteScenario: Scenario = {
  name: '07 offline delete vs concurrent edit: losing device must not keep a ghost row',
  run: async ({ backend }) => {
    await withDevices(backend, 'delete-race', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      // Seed the shared card through A; B bootstraps it.
      await a.engine.bootstrap();
      await a.engine.mutate('cards', { rowUuid: 'card-x', data: card('doomed', 'x1') });
      await a.engine.sync();
      await b.engine.bootstrap();
      if (!await b.engine.getRow('cards', 'card-x')) {
        throw new Error('device B should bootstrap the seeded card');
      }

      // Both offline: A deletes it, B edits it, both from base 1.
      a.engine.setOnline(false);
      await a.engine.deleteLocal('cards', 'card-x');
      b.engine.setOnline(false);
      await b.engine.mutate('cards', { rowUuid: 'card-x', data: card('edited by B', 'x1') });

      const summaries = await settle([a, b], 2);
      const bPush = summaries.get('dev-b')?.find((s) => s.pushed + s.conflicts > 0);
      if (!bPush || bPush.conflicts < 1) {
        throw new Error(`expected device B's edit to conflict with A's delete, got ${JSON.stringify(bPush)}`);
      }

      const server = await convergeAndAssert('offline delete vs edit', [a, b], auth, baseUrl);

      // The delete won: no live card-x anywhere (server + both devices).
      const serverCards = server.collections.cards ?? [];
      if (serverCards.some((r) => r.row_uuid === 'card-x')) {
        throw new Error('server still has the deleted card');
      }
      for (const dev of [a, b]) {
        if (await dev.engine.getRow('cards', 'card-x')) {
          throw new Error(`device ${dev.id} keeps a ghost card-x after the delete won the LWW race`);
        }
      }
    });
  },
};
