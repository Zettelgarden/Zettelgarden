/**
 * Scenario 4 — offline card_id rename on both devices: both devices rename
 * the SAME logical card offline from the same base. LWW must pick one rename
 * and the card must NOT split into two logical rows (sync_uuid identity is
 * immutable; card_id is just a user-editable field).
 */

import type { Scenario } from './context';
import { convergeAndAssert, pushSummary, settle, withDevices } from './context';

function card(cardId: string, title: string) {
  return { card_id: cardId, title, body: title };
}

export const cardIdRenameScenario: Scenario = {
  name: '04 offline card_id rename on both devices must not split the row',
  run: async ({ backend }) => {
    await withDevices(backend, 'card-id-rename', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      // Seed a root card; both devices bootstrap it.
      await a.engine.bootstrap();
      const baselineCards = a.engine.query('cards').length;
      a.engine.mutate('cards', { rowUuid: 'card-root', data: card('root', 'root card') });
      await a.engine.sync();
      await b.engine.bootstrap();

      // Both devices rename the SAME card offline from base version 1.
      a.engine.setOnline(false);
      a.engine.mutate('cards', { rowUuid: 'card-root', data: card('root.alpha', 'root card') });
      b.engine.setOnline(false);
      b.engine.mutate('cards', { rowUuid: 'card-root', data: card('root.beta', 'root card') });

      const summaries = await settle([a, b], 2);
      const bPush = pushSummary(summaries, 'dev-b');
      if (!bPush || bPush.lostEdits < 1) {
        throw new Error(`expected device B to lose its concurrent rename, got ${JSON.stringify(bPush)}`);
      }

      const server = await convergeAndAssert('card_id rename', [a, b], auth, baseUrl);

      // Exactly ONE logical row survives (plus the seeded welcome card), with
      // one deterministic card_id.
      const serverCards = server.collections.cards ?? [];
      if (serverCards.length !== baselineCards + 1) {
        throw new Error(`card split: expected ${baselineCards + 1} rows, server has ${serverCards.length}`);
      }
      const winnerRow = serverCards.find((r) => r.row_uuid === 'card-root');
      const winner = (winnerRow?.data as { card_id?: string } | undefined)?.card_id;
      if (winner !== 'root.alpha') {
        throw new Error(`expected A's rename (root.alpha) to win, got '${winner}'`);
      }
      for (const dev of [a, b]) {
        const rows = dev.engine.query('cards').filter((r) => r.rowUuid === 'card-root');
        if (rows.length !== 1) {
          throw new Error(`device ${dev.id} has ${rows.length} rows for card-root; the rename split the logical row`);
        }
        if ((rows[0]!.data as { card_id?: string }).card_id !== 'root.alpha') {
          throw new Error(`device ${dev.id} did not adopt the winning card_id`);
        }
      }
    });
  },
};
