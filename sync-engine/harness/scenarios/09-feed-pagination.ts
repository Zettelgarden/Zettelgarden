/**
 * Scenario 9 — feed pagination + >500-change push batch (Zettelgarden-b13).
 *
 * The changes feed pages at syncFeedPageSize=500 (handlers/sync.go, has_more
 * + cursor contract). This scenario pushes 510 changes in ONE batch (also
 * exercising the >500-changes push path; the byte guard is a MaxBytesReader)
 * and then pulls them back, so the engine's pull loop must advance the cursor
 * across two pages and apply every row. A pagination regression (cursor
 * advance, has_more handling) would fail the `pulled === 510` assertion or
 * leave devices diverged.
 */

import type { Scenario } from './context';
import { convergeAndAssert, withDevices } from './context';

function card(title: string, cardId: string) {
  return { title, card_id: cardId, body: title };
}

const PAGE_COUNT = 510; // syncFeedPageSize (500) + a partial second page

export const feedPaginationScenario: Scenario = {
  name: '09 feed pagination: 510-row push + paginated pull (>500 page size)',
  run: async ({ backend }) => {
    await withDevices(backend, 'feed-pagination', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      await a.engine.bootstrap();

      // Create 510 cards locally without syncing: one batched push.
      for (let i = 0; i < PAGE_COUNT; i++) {
        await a.engine.mutate('cards', {
          rowUuid: `pg-${i}`,
          data: card(`pagination card ${i}`, `pgid${i}`),
        });
      }

      // First sync: the 510-change batch drains in one /push.
      const s1 = await a.engine.sync();
      if (s1.pushed !== PAGE_COUNT) {
        throw new Error(
          `expected ${PAGE_COUNT} changes pushed in one batch, got ${JSON.stringify(s1)}`,
        );
      }

      // Second sync: pull the feed back. 510 entries page at 500, so the pull
      // must loop on has_more and apply all of them.
      const s2 = await a.engine.sync();
      if (s2.pulled !== PAGE_COUNT) {
        throw new Error(
          `expected paginated pull of ${PAGE_COUNT} entries, got ${JSON.stringify(s2)}`,
        );
      }

      // B bootstraps the full server state and converges.
      await b.engine.bootstrap();
      const bCards = (await b.engine.query('cards')).length;
      if (bCards < PAGE_COUNT) {
        throw new Error(`device B bootstrapped ${bCards} cards, want >= ${PAGE_COUNT}`);
      }

      const server = await convergeAndAssert('feed pagination >500 rows', [a, b], auth, baseUrl);
      const serverCards = server.collections.cards ?? [];
      // Fresh accounts are seeded with a welcome card (see harness README).
      if (serverCards.length !== PAGE_COUNT + 1) {
        throw new Error(
          `server card count = ${serverCards.length}, want ${PAGE_COUNT + 1} (welcome + ${PAGE_COUNT})`,
        );
      }
    });
  },
};
