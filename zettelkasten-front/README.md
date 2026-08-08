# zettelkasten-front

Web client for Zettelgarden — React 18 + TypeScript, built with Vite, styled
with Tailwind CSS. Server state is managed with TanStack Query.

See the [root README](../README.md) for the overall project and self-hosting
instructions.

## Development

```bash
npm install
npm run start        # Vite dev server on http://localhost:5173
```

## Scripts

| Script                                    | Purpose                                  |
| ----------------------------------------- | ---------------------------------------- |
| `npm run start`                           | Vite dev server                          |
| `npm run build`                           | Type-check + production build to `dist/` |
| `npm run serve`                           | Preview the production build             |
| `npm run test` / `npm run test:watch`     | Vitest (watch mode)                      |
| `npm run test:run`                        | Vitest, run once                         |
| `npm run test:coverage`                   | Vitest with coverage (CI-style)          |
| `npm run typecheck`                       | `tsc --noEmit`                           |
| `npm run format` / `npm run format:check` | Prettier (write / CI gate)               |

## Conventions

- Components in `src/components/` (PascalCase), state in `src/contexts/`,
  server-state hooks in `src/hooks/queries/`, API clients in `src/api/`.
- Unit tests are colocated as `*.test.ts(x)` and run with Vitest + Testing
  Library; prefer rendering components over shallow mocks. See `TESTING.md`.
- Format with Prettier (2-space indent, single quotes).

## Desktop shell

The Tauri v2 desktop wrapper lives in `../desktop/` and packages the build
output of this app (`npm run build` produces `dist/`, which
`desktop/src-tauri/tauri.conf.json` points at).
