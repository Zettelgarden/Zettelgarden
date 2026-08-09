/**
 * Scenario 8 — plain offline delete propagates via the feed (Zettelgarden-b13).
 *
 * Scenario 07 covered delete-vs-concurrent-edit (the losing device's pull
 * SKIPS the tombstone while its edit is pending, and the conflict reconciles
 * it). This one is the dedicated no-concurrent-edit case: A deletes a card
 * offline, pushes the tombstone, and B — with NO pending edit on that row —
 * must see it disappear purely from pulling the feed. Regression guard for
 * the plain delete path that unit tests only cover with a mock transport.
 */

import type { Scenario } from './context';
import { convergeAndAssert, withDevices } from './context';

function card(title: string, cardId: string) {
  return { title, card_id: cardId, body: title };
}

export const deletePropagatesScenario: Scenario = {
  name: '08 delete propagates via feed: plain offline delete reaches the other device',
  run: async ({ backend }) => {
    await withDevices(backend, 'delete-propagate', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      // Seed through A; B bootstraps the live row.
      await a.engine.bootstrap();
      await a.engine.mutate('cards', { rowUuid: 'card-x', data: card('to be deleted', 'x1') });
      await a.engine.sync();
      await b.engine.bootstrap();
      if (!(await b.engine.getRow('cards', 'card-x'))) {
        throw new Error('device B should bootstrap the seeded card');
      }

      // A deletes it OFFLINE (no concurrent edit on B), then reconnects and
      // pushes the delete tombstone.
      a.engine.setOnline(false);
      await a.engine.deleteLocal('cards', 'card-x');
      a.engine.setOnline(true);
      const sA = await a.engine.sync();
      if (sA.pushed !== 1) {
        throw new Error(`expected A to push 1 delete, got ${JSON.stringify(sA)}`);
      }

      // B pulls with NO pending edit on card-x: the feed tombstone must remove
      // the row locally (not via a conflict).
      await b.engine.sync();
      if (await b.engine.getRow('cards', 'card-x')) {
        throw new Error('device B still has card-x after pulling the delete tombstone');
      }
      if (await a.engine.getRow('cards', 'card-x')) {
        throw new Error('device A keeps a ghost card-x after the delete');
      }

      const server = await convergeAndAssert('delete propagates via feed', [a, b], auth, baseUrl);
      const serverCards = server.collections.cards ?? [];
      if (serverCards.some((r) => r.row_uuid === 'card-x')) {
        throw new Error('server still has the deleted card');
      }
    });
  },
};
