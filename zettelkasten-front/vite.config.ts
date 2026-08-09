import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react-swc';

// The shared sync engine (sync-engine/) is imported by source (not built
// dist) so the frontend builds without a preceding sync-engine build step —
// required for CI/Docker where dist/ is gitignored. The engine's sqlite.ts
// (better-sqlite3) is never imported by the frontend, so no native module
// leaks into the web bundle.
const syncEngineSrc = fileURLToPath(
  new URL('../sync-engine/src', import.meta.url),
);

// https://vitejs.dev/config/
export default defineConfig({
  base: '/',
  plugins: [react()],
  resolve: {
    alias: {
      '@zettelgarden/sync-engine': syncEngineSrc,
    },
  },
});
