/**
 * Scenario 2 — concurrent edits: both devices edit the same card offline from
 * the same base version; the server resolves by LWW deterministically (first
 * push wins) and reports the lost edit. Both devices adopt the winner.
 */

import type { Scenario } from './context';
import { convergeAndAssert, pushSummary, settle, withDevices } from './context';

function card(title: string, cardId: string) {
  return { title, card_id: cardId, body: title };
}

export const concurrentLwwScenario: Scenario = {
  name: '02 concurrent edits: LWW resolves deterministically, lost edit reported',
  run: async ({ backend }) => {
    await withDevices(backend, 'concurrent-lww', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      // Seed the shared card through A; B bootstraps the same state.
      await a.engine.bootstrap();
      await a.engine.mutate('cards', { rowUuid: 'card-1', data: card('original', 'c1') });
      await a.engine.sync();
      await b.engine.bootstrap();
      if ((await b.engine.getRow('cards', 'card-1'))?.data.title !== 'original') {
        throw new Error('device B should bootstrap the seeded card');
      }

      // Both edit the same row offline from base version 1.
      a.engine.setOnline(false);
      await a.engine.mutate('cards', { rowUuid: 'card-1', data: card('edited by A', 'c1') });
      b.engine.setOnline(false);
      await b.engine.mutate('cards', { rowUuid: 'card-1', data: card('edited by B', 'c1') });

      // A pushes first (wins the LWW race), then B pushes against a stale base.
      const summaries = await settle([a, b], 2);
      const bPush = pushSummary(summaries, 'dev-b');
      if (!bPush) throw new Error('device B never pushed');
      if (bPush.lostEdits < 1) {
        throw new Error(`expected device B to report a lost edit on push, got lostEdits=${bPush.lostEdits}`);
      }

      const server = await convergeAndAssert('concurrent LWW', [a, b], auth, baseUrl);

      // Deterministic winner: A pushed first, so "edited by A" must survive.
      const winner = (server.collections.cards ?? []).find((r) => r.row_uuid === 'card-1');
      if (!winner || (winner.data as { title?: string }).title !== 'edited by A') {
        throw new Error(`expected A's edit to win, got ${JSON.stringify(winner?.data)}`);
      }
      if ((await b.engine.getRow('cards', 'card-1'))?.data.title !== 'edited by A') {
        throw new Error('device B should have adopted the server (winning) row');
      }
    });
  },
};
