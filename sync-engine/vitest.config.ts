import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

/**
 * Unit-test config for the sync engine. Aliases let the engine's own tests
 * and the mobile-adapter matrix import the frontend's MobileStorageAdapter
 * (which references the engine package by its published name, exactly as the
 * web bundle does via vite).
 */
export default defineConfig({
  test: {
    include: ["test/**/*.test.ts"],
    environment: "node",
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
          new URL("./src/storage.ts", import.meta.url),
        ),
      },
      {
        find: "@zettelgarden/sync-engine/types",
        replacement: fileURLToPath(new URL("./src/types.ts", import.meta.url)),
      },
      {
        find: "@zettelgarden/sync-engine",
        replacement: fileURLToPath(new URL("./src/index.ts", import.meta.url)),
      },
    ],
  },
});
