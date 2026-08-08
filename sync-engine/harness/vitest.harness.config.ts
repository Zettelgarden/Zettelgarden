import { defineConfig } from 'vitest/config';

/**
 * Dedicated config for the live-backend convergence harness: sequential
 * (one backend, one process), long timeouts for the Go build + boot, and a
 * test pattern that excludes it from the normal unit-test run.
 */
export default defineConfig({
  test: {
    include: ['harness/**/*.test.ts'],
    environment: 'node',
    fileParallelism: false,
    pool: 'forks',
    poolOptions: { forks: { singleFork: true } },
    testTimeout: 120_000,
    // A cold go build of go-backend (module download + compile) can exceed
    // several minutes on a fresh runner; CI pre-warms it, but keep the hook
    // generous for local runs too.
    hookTimeout: 600_000,
  },
});
