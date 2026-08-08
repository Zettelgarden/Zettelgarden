# Zettelgarden Blog (extracted)

The blog (markdown posts + RSS feed), extracted from the self-hosted app
bundle (epic Zettelgarden-6er, ticket 6er.13). This is a **standalone
project** — it does not import anything from `zettelkasten-front/` and is
ready to be moved to its own repository.

## Why it exists

The self-hosted Zettelgarden app no longer ships the blog: `/blog/*` routes
are gone from the bundle. The blog lives here instead, with its own static
header/footer (copied from the marketing site) and its own newsletter stub.

## Run

```bash
npm install
npm run dev      # http://localhost:5175/blog
npm run build    # emits static site to dist/
```

The blog is mounted under `/blog/*` so post URLs (`/blog/<slug>`) and the RSS
feed (`/blog/rss.xml`) keep their original shape. `App.tsx` also routes
unknown paths to the blog for convenient static hosting at `/`.

## Env vars (`.env` or shell)

| Var | Purpose |
| --- | --- |
| `VITE_APP_URL` | The hosted app base URL. The header CTA links to `${VITE_APP_URL}/login`. |
| `VITE_NEWSLETTER_ENDPOINT` | Optional JSON endpoint accepting `{ "email": "..." }` for the post newsletter form. Without it the form is a no-op success. |
| `VITE_SUPPORT_EMAIL` | Optional `mailto:` address shown in the footer. |

## Posts

Markdown lives in `src/blog/posts/*.md` (gray-matter frontmatter: title,
date, author, excerpt, tags). Add a file, and `utils.ts` picks it up
automatically; `rss.ts` generates the feed.

## Moving to its own repo

```bash
git subtree split -P blog -b blog
git push <blog-repo-url> blog:master
```

(Or copy the directory and init a fresh repo — nothing references the parent.)
