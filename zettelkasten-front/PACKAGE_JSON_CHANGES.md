# Package.json Changes for React Query Migration

## Current Dependencies

Relevant sections from current `package.json`:

```json
{
  "dependencies": {
    "@tanstack/match-sorter-utils": "^8.19.4",
    "@tanstack/react-table": "^8.20.6",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.22.3"
  },
  "devDependencies": {
    "vitest": "^2.0.5"
  }
}
```

## Required Additions

Add these dependencies to enable React Query:

```json
{
  "dependencies": {
    "@tanstack/match-sorter-utils": "^8.19.4",
    "@tanstack/react-query": "^5.60.0",
    "@tanstack/react-table": "^8.20.6",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.22.3"
  },
  "devDependencies": {
    "@tanstack/react-query-devtools": "^5.60.0",
    "vitest": "^2.0.5"
  }
}
```

## Installation Commands

```bash
cd zettelkasten-front
npm install @tanstack/react-query@^5.60.0
npm install -D @tanstack/react-query-devtools@^5.60.0
```

## Bundle Size Impact

| Package                        | Minified         | Gzipped         |
| ------------------------------ | ---------------- | --------------- |
| @tanstack/react-query          | ~45KB            | ~13KB           |
| @tanstack/react-query-devtools | ~15KB (dev only) | ~4KB (dev only) |
| **Total**                      | **~45KB**        | **~13KB**       |

Note: Devtools are only included in development builds.

## Version Compatibility

React Query v5 is compatible with:

- React 18+ (Zettelgarden uses React 18.2.0)
- TypeScript 5+ (Zettelgarden uses TypeScript 5.4.5)
- Vite (Zettelgarden uses Vite 5.3.4)

No breaking changes to existing dependencies.

## Peer Dependencies

React Query has no additional peer dependencies beyond React 18+.

## Existing TanStack Dependencies

Zettelgarden already uses TanStack packages:

- `@tanstack/react-table` - Table component library
- `@tanstack/match-sorter-utils` - Utilities

Adding `@tanstack/react-query` aligns with the existing ecosystem.

## Updated Package.json

Here is the complete updated `package.json` for reference:

```json
{
  "name": "zettelkasten-front",
  "version": "0.1.0",
  "private": true,
  "dependencies": {
    "@headlessui/react": "^2.2.0",
    "@hello-pangea/dnd": "^18.0.1",
    "@llamaindex/chat-ui": "^0.6.1",
    "@radix-ui/react-slot": "^1.0.2",
    "@stripe/stripe-js": "^7.9.0",
    "@tanstack/match-sorter-utils": "^8.19.4",
    "@tanstack/react-query": "^5.60.0",
    "@tanstack/react-table": "^8.20.6",
    "@testing-library/jest-dom": "^5.17.0",
    "@testing-library/react": "^13.4.0",
    "@testing-library/user-event": "^13.5.0",
    "@types/react": "^18.3.3",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react-swc": "^3.7.0",
    "bootstrap-icons": "^1.11.3",
    "buffer": "^6.0.3",
    "date-fns": "^4.1.0",
    "date-fns-tz": "^3.2.0",
    "diff": "^8.0.3",
    "feed": "^4.2.2",
    "framer-motion": "^11.15.0",
    "gray-matter": "^4.0.3",
    "happy-dom": "^15.7.3",
    "js-yaml": "^4.1.0",
    "linkify-html": "^4.1.3",
    "linkify-react": "^4.1.3",
    "linkifyjs": "^4.1.3",
    "react": "^18.2.0",
    "react-date-picker": "^11.0.0",
    "react-dom": "^18.2.0",
    "react-icons": "^5.5.0",
    "react-markdown": "^9.0.0",
    "react-router-dom": "^6.22.3",
    "rehype-raw": "^7.0.0",
    "rehype-sanitize": "^6.0.0",
    "remark-gfm": "^4.0.1",
    "typescript": "^5.4.5",
    "vite": "^5.3.4",
    "vite-plugin-svgr": "^4.2.0",
    "vite-tsconfig-paths": "^4.3.2",
    "web-vitals": "^2.1.4"
  },
  "scripts": {
    "start": "vite",
    "build": "tsc && vite build",
    "serve": "vite preview",
    "test": "vitest",
    "test:run": "vitest run",
    "test:watch": "vitest",
    "test:coverage": "vitest run --coverage",
    "test:ui": "vitest --ui"
  },
  "eslintConfig": {
    "extends": ["react-app", "react-app/jest"]
  },
  "browserslist": {
    "production": [">0.2%", "not dead", "not op_mini all"],
    "development": [
      "last 1 chrome version",
      "last 1 firefox version",
      "last 1 safari version"
    ]
  },
  "devDependencies": {
    "@tanstack/react-query-devtools": "^5.60.0",
    "@tailwindcss/typography": "^0.5.15",
    "@testing-library/dom": "^10.4.1",
    "@vitest/coverage-v8": "^2.1.9",
    "autoprefixer": "^10.4.19",
    "postcss": "^8.4.39",
    "prettier": "^3.0.3",
    "tailwindcss": "^3.4.6",
    "vitest": "^2.0.5"
  }
}
```

## Verification

After installation, verify the installation:

```bash
# Check installed versions
npm list @tanstack/react-query
npm list @tanstack/react-query-devtools

# Should show:
# @tanstack/react-query@5.x.x
# @tanstack/react-query-devtools@5.x.x
```
