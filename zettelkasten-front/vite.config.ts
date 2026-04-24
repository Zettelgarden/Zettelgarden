import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react-swc'
import electron from 'vite-plugin-electron'

// https://vitejs.dev/config/
export default defineConfig({
  base: '/',
  plugins: [
    react(),
    electron([
      {
        entry: 'electron/main.ts',
      },
      {
        // Preload script — runs in renderer context with Node access
        entry: 'electron/preload.ts',
        onstart(args) {
          // Notify renderer to reload when preload changes
          args.reload()
        },
      },
    ]),
  ],
  test: {
    globals: true,
    environment: 'happy-dom',
    setupFiles: ['./src/setupTests.js'],
  },
})
