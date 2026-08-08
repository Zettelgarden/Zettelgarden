# Zettelgarden Marketing Site (extracted)

The SaaS marketing landing page, extracted from the self-hosted app bundle
(epic Zettelgarden-6er, ticket 6er.14). This is a **standalone project** — it
does not import anything from `zettelkasten-front/` and is ready to be moved
to its own repository.

## Why it exists

The self-hosted Zettelgarden app no longer ships marketing chrome: `/` serves
login/app, never a landing page. The landing page lives here instead, so it
can be deployed as its own static site pointing at your hosted app.

## Run

```bash
npm install
npm run dev      # http://localhost:5174
npm run build    # emits static site to dist/
```

## Env vars (`.env` or shell)

| Var | Purpose |
| --- | --- |
| `VITE_APP_URL` | The hosted app base URL (e.g. `https://app.zettelgarden.com`). The header CTA and "Get Started" buttons link to `${VITE_APP_URL}/login`. |
| `VITE_NEWSLETTER_ENDPOINT` | Optional JSON endpoint that accepts `{ "email": "..." }` (Buttondown/Formspree-style). Without it the newsletter form is a no-op success. |
| `VITE_SUPPORT_EMAIL` | Optional `mailto:` address shown in the footer. |
| `VITE_ENV` | Optional title suffix. |

## Content

- Components: `src/landing/` (LandingPage + Header/Footer/SocialLinks + 10
  section components), `src/data/landingContent.ts` (copy), `src/types/landing.ts`.
- Copy docs that moved with the marketing project: `docs/marketing-copy.md`,
  `docs/product-overview.md`, `docs/competitive-analysis.md` (in the main repo
  until this project gets its own repo; move them here when you do).

## Moving to its own repo

```bash
git subtree split -P marketing-site -b marketing-site
git push <marketing-repo-url> marketing-site:master
```

(Or copy the directory and init a fresh repo — nothing references the parent.)
