import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

/**
 * Dedicated config for the live-backend convergence harness: sequential
 * (one backend, one process), long timeouts for the Go build + boot, and a
 * test pattern that excludes it from the normal unit-test run. The aliases
 * match vitest.config.ts so scenario 11 (mobile bridge) can load the
 * frontend's MobileStorageAdapter.
 */
export default defineConfig({
  test: {
    include: ["harness/**/*.test.ts"],
    environment: "node",
    fileParallelism: false,
    pool: "forks",
    poolOptions: { forks: { singleFork: true } },
    testTimeout: 120_000,
    // A cold go build of go-backend (module download + compile) can exceed
    // several minutes on a fresh runner; CI pre-warms it, but keep the hook
    // generous for local runs too.
    hookTimeout: 600_000,
  },
  resolve: {
    alias: [
      {
        find: "@zg-adapter/mobile",
        replacement: fileURLToPath(
          new URL(
            "../../zettelkasten-front/src/data/mobileStorageAdapter.ts",
            import.meta.url,
          ),
        ),
      },
      {
        find: "@zettelgarden/sync-engine/storage",
        replacement: fileURLToPath(
          new URL("../src/storage.ts", import.meta.url),
        ),
      },
      {
        find: "@zettelgarden/sync-engine/types",
        replacement: fileURLToPath(new URL("../src/types.ts", import.meta.url)),
      },
      {
        find: "@zettelgarden/sync-engine",
        replacement: fileURLToPath(new URL("../src/index.ts", import.meta.url)),
      },
    ],
  },
});
