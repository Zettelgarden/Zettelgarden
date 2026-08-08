/**
 * Phase 1b convergence harness entry (Zettelgarden-xre).
 *
 * Boots ONE live Go backend for the whole run (build + spawn + wait-ready),
 * then runs every scenario against its own fresh account with two real
 * SQLite-backed devices. Any divergence fails the test (non-zero exit), so
 * this doubles as a CI step: `npm run harness`.
 */

import { afterAll, beforeAll, describe, it } from 'vitest';
import { HarnessBackend } from './lib/backend';
import { scenarios } from './scenarios';

describe('two-DB convergence harness (live backend)', () => {
  let backend: HarnessBackend;

  beforeAll(async () => {
    backend = await HarnessBackend.start();
  }, 180_000);

  afterAll(async () => {
    await backend?.stop();
  }, 30_000);

  for (const scenario of scenarios) {
    it(scenario.name, async () => {
      try {
        await scenario.run({ backend });
      } catch (err) {
        // Attach the live backend's log tail so server-side failures are
        // diagnosable without a manual repro.
        const detail =
          err instanceof Error ? err.message : String(err);
        throw new Error(
          `${detail}\n\n--- backend log tail ---\n${backend.logs().slice(-60).join('\n')}`,
        );
      }
    }, 120_000);
  }
});
