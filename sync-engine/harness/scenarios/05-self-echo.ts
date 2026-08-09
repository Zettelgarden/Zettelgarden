/**
 * Scenario 5 — interleaved push/pull (self-echo): a device's own pushed
 * changes echo back through the feed on the next pull; they must be applied
 * idempotently (same uuid + version, no double-apply, no version inflation).
 * Exercises mutate → sync → sync (echo) → mutate → sync → sync interleaving.
 */

import type { Scenario } from './context';
import { convergeAndAssert, settle, withDevices } from './context';

function card(title: string, cardId: string) {
  return { title, card_id: cardId, body: title };
}

export const selfEchoScenario: Scenario = {
  name: '05 interleaved push/pull: own echoed changes are not double-applied',
  run: async ({ backend }) => {
    await withDevices(backend, 'self-echo', ['dev-a', 'dev-b'], async ([a, b], auth, baseUrl) => {
      await a.engine.bootstrap();
      await b.engine.bootstrap();
      const baselineCards = (await a.engine.query('cards')).length;

      // A creates a card offline and syncs: push applies it (v1), then the
      // next sync's pull echoes the same row back. Version must stay 1 and
      // exactly one row must exist on the server.
      a.engine.setOnline(false);
      await a.engine.mutate('cards', { rowUuid: 'echo-card', data: card('echo v1', 'e1') });
      a.engine.setOnline(true);
      await a.engine.sync(); // push
      await a.engine.sync(); // pull echoes own change

      if ((await a.engine.getRow('cards', 'echo-card'))?.version !== 1) {
        throw new Error(`self-echo double-bumped version: ${(await a.engine.getRow('cards', 'echo-card'))?.version}`);
      }

      // A edits the same card and syncs twice again.
      a.engine.setOnline(false);
      await a.engine.mutate('cards', { rowUuid: 'echo-card', data: card('echo v2', 'e1') });
      a.engine.setOnline(true);
      await a.engine.sync();
      await a.engine.sync();
      if ((await a.engine.getRow('cards', 'echo-card'))?.version !== 2) {
        throw new Error(`second self-echo double-bumped version: ${(await a.engine.getRow('cards', 'echo-card'))?.version}`);
      }

      // B bootstraps after the fact and pulls the full history.
      await b.engine.bootstrap();
      await b.engine.sync();
      if ((await b.engine.getRow('cards', 'echo-card'))?.data.title !== 'echo v2') {
        throw new Error('device B should see the final state via the feed');
      }

      // Settle everything and verify convergence + no duplicates.
      await settle([a, b], 2);
      const server = await convergeAndAssert('self-echo', [a, b], auth, baseUrl);
      const serverCards = server.collections.cards ?? [];
      if (serverCards.length !== baselineCards + 1) {
        throw new Error(`expected exactly ${baselineCards + 1} cards on the server, got ${serverCards.length}`);
      }
      const row = serverCards.find((r) => r.row_uuid === 'echo-card');
      if (!row) throw new Error('server missing echo-card');
      if (row.version !== 2) {
        throw new Error(`expected server card version 2, got ${row.version}`);
      }
      if ((row.data as { title?: string }).title !== 'echo v2') {
        throw new Error(`expected server card 'echo v2', got ${JSON.stringify(row.data)}`);
      }
    });
  },
};
