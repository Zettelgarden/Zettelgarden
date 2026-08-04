# OIDC (OpenID Connect) Authentication Support — Design

**Created:** 2026-08-03
**Status:** Proposed (awaiting review)
**Tracker:** TBD (beads)

## Problem Statement

Zettelgarden supports three auth mechanisms today: email/password (JWT), GitHub
OAuth (hand-rolled, provider-specific), and API keys. There is no support for
**generic OIDC / SSO providers** (Google, Microsoft Entra, Okta, Auth0,
Keycloak, etc.). Users on enterprise or self-hosted IdPs cannot log in without
a password or a GitHub account.

We want to add **standards-based OIDC** so a single configurable provider (and
eventually multiple) can be used to sign in. As a bonus, this is an opportunity
to harden the existing OAuth flows, which currently lack CSRF `state` and
`nonce` protection.

## Goals / Non-Goals

**Goals**
- Add a generic OIDC Authorization Code flow driven by **issuer discovery**
  (`/.well-known/openid-configuration`), so any compliant provider works.
- Validate the `id_token` properly (JWKS signature, `aud`, `iss`, `exp`,
  `nonce`).
- Reuse the existing JWT issuance + frontend token-in-URL callback so the rest
  of the app (RBAC, API keys, subscription checks) is untouched.
- Create or link local users by verified email, storing the OIDC `sub` for
  stable re-authentication.
- Make the flow CSRF-safe via `state` (and retrofit `state` to the GitHub
  flow).

**Non-Goals (this phase)**
- Multiple simultaneous providers (single configurable provider first; the
  design leaves the door open).
- Linking/unlinking IdP accounts from a settings page (follow-up).
- Social provider-specific UX beyond a generic "Continue with SSO" button.
- Migrating the hand-rolled GitHub handler to the OIDC library (kept as-is,
  only hardened with `state`).

## Current State (context for the plan)

- **Backend auth core:** `go-backend/handlers/auth.go` — `JwtMiddleware`,
  `APIKeyOrJWTMiddleware`, `generateAccessToken`, `LoginRoute`, etc. JWTs are
  HS256 signed with `s.Server.JwtSecretKey` (from `SECRET_KEY`).
- **GitHub OAuth:** `go-backend/handlers/oauth.go` —
  `StartGitHubOAuthRoute` / `GitHubCallbackRoute`. Exchanges code → GH access
  token → fetches `/user` + `/user/emails`, finds-or-creates user by email,
  issues our JWT, redirects to `${ZETTEL_URL}/login?token=<jwt>`.
- **Routes:** `go-backend/routes/auth.go` registers the public auth routes via
  `addRoute` (no auth). `go-backend/routes/helpers.go` defines
  `addProtectedRoute` / `addRoute` / `addAdminRoute`.
- **Config:** `go-backend/pkg/config/services.go` — `GitHubConfig` struct +
  `loadGitHubConfig()` reading `GITHUB_CLIENT_ID/SECRET/REDIRECT_URI`.
- **Users table** (`go-backend/schema/sqlite/schema.sqlite.sql` ~L943): has
  `auth_provider TEXT DEFAULT 'local'` and `github_id TEXT`.
- **Frontend:** `zettelkasten-front/src/pages/LoginPage.tsx` (GitHub button +
  `?token=` callback handler), `src/contexts/AuthContext.tsx`
  (`loginUserFromToken`), `src/api/auth.ts`, `src/api/client.ts` (Bearer
  header from `localStorage.token`).
- **CORS:** `main.go` allows only `cfg.Server.URL` with credentials.

## Proposed Architecture

### Flow (Authorization Code + PKCE-optional)

1. User clicks **"Continue with SSO"** → frontend hits
   `GET /api/auth/oidc/start`.
2. Backend builds the auth URL against the provider's **discovered**
   authorization endpoint with:
   `client_id`, `redirect_uri`, `response_type=code`,
   `scope=openid email profile`, a random **`state`** and **`nonce`**, then
   302-redirects the browser. `state` + `nonce` are stored in a short-lived
   **signed cookie** (HMAC over `state|nonce|expiry` using `SECRET_KEY`) so no
   server-side session store is required.
3. Provider authenticates the user → redirects to
   `GET /api/auth/oidc/callback?code=...&state=...`.
4. Backend verifies the cookie's `state`/`nonce` and expiry, then exchanges
   `code` for tokens at the provider's token endpoint → receives `id_token`
   (JWT) + `access_token`.
5. Backend verifies the `id_token`:
   - Signature via provider **JWKS** (`coreos/go-oidc` Verifier).
   - `iss == configured issuer`, `aud == client_id`, `exp` not passed,
     `nonce` matches.
6. Extract identity claims: `sub` (stable), `email`, `email_verified`,
   `name`/`preferred_username`.
7. **Find or create user:**
   - Prefer match by `(oidc_provider, oidc_sub)` (stable re-auth).
   - Else match by `email` **only if `email_verified == true`**.
   - Else create a new user (mark `email_validated = true` since IdP verified
     it; `auth_provider = 'oidc'`).
8. Issue our app JWT via existing `generateAccessToken` and redirect to
   `${ZETTEL_URL}/login?token=<jwt>` — identical to the GitHub path, so the
   frontend's existing `loginUserFromToken` handles the rest.

### Library choice

Use the idiomatic Go stack (no hand-rolling of JWKS/nonce):
- `github.com/coreos/go-oidc/v3/oidc` — provider discovery, ID-token
  verification, JWKS caching.
- `golang.org/x/oauth2` — authorization-code exchange.

`go.mod` currently has `golang.org/x/oauth2`? **No** — needs adding. Add:
```
go get github.com/coreos/go-oidc/v3/oidc
go get golang.org/x/oauth2
```

## Detailed Changes (file by file)

### Backend

**New file `go-backend/handlers/oidc.go`**
- `OIDCConfig` resolved once at startup (or lazily on first request) using
  `oidc.NewProvider(ctx, issuer)` → caches `provider`, `oauth2.Config`,
  `oidc.IDTokenVerifier`.
- `StartOIDCRoute(w, r)` — generates `state`+`nonce`, sets signed cookie
  `zg_oidc`, builds auth URL, 302.
- `CallbackOIDCRoute(w, r)` — validates cookie, exchanges code, verifies
  id_token, resolves/creates user, issues app JWT, redirects to frontend.
- Helper `signStateCookie` / `verifyStateCookie` (HMAC-SHA256 over
  `state|nonce|exp` using `s.Server.JwtSecretKey`; `HttpOnly`, `Secure`,
  `SameSite=Lax`, ~10 min TTL).

**New file `go-backend/handlers/oauth_state.go`** (shared)
- Move the signed-cookie `state` helpers here so the GitHub handler can reuse
  them for CSRF hardening (see follow-ups). Both GitHub and OIDC flows then
  share one well-tested primitive.

**`go-backend/routes/auth.go`** — register two public routes:
```go
addRoute(r, "/api/auth/oidc/start", h.StartOIDCRoute, "GET")
addRoute(r, "/api/auth/oidc/callback", h.CallbackOIDCRoute, "GET")
```
Document them in the public-route comment block in `routes/routes.go`.

**`go-backend/pkg/config/services.go`**
- Add `OIDCConfig` struct:
  ```go
  type OIDCConfig struct {
      Enabled      bool   // if false, start endpoint 404s
      Issuer       string // e.g. https://accounts.google.com
      ClientID     string
      ClientSecret string
      RedirectURI  string // e.g. https://app/api/auth/oidc/callback
      // Optional: explicit scopes (default "openid email profile")
  }
  ```
- Add `loadOIDCConfig()` reading `OIDC_ENABLED`, `OIDC_ISSUER`,
  `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URI`. Use optional
  getters so OIDC stays disabled when unset (don't `log.Fatal` like GitHub
  config does — OIDC is opt-in).
- Wire into `ServiceConfig` and into `main.go` (`s.OIDCConfig = ...`) the same
  way `GitHub` config flows.

**`go-backend/handlers/handlers.go`** — add `OIDCConfig` (and a cached
`*oidc.Provider`) to the `Handler` struct.

**`go-backend/models/user.go`** — add JSON fields:
```go
OIDCProvider string `json:"oidc_provider"`
OIDCSub      string `json:"oidc_sub"`
```
Update the scan in `QueryUser`/`QueryUserByEmail` (in `models/` or
`services/`) to read the new columns.

### Database / Schema

Two places, matching the repo's dual-path history (SQLite primary, Postgres
legacy):

**SQLite (primary):** edit the consolidated
`go-backend/schema/sqlite/schema.sqlite.sql` `users` table — add:
```sql
oidc_provider TEXT,
oidc_sub      TEXT,
```
plus an index for the stable-lookup path:
```sql
CREATE UNIQUE INDEX idx_users_oidc_sub ON users (oidc_provider, oidc_sub) WHERE oidc_sub IS NOT NULL;
```
> Note: SQLite applies `schema.sqlite.sql` once on a fresh DB (tracked in the
> `migrations` table by filename). For **existing** SQLite DBs the consolidated
> file won't re-run, so also add the `ALTER TABLE` to a numbered migration (see
> below) and drop it into the sqlite scan dir, OR accept that the cutover-era
> workflow rebuilds from the consolidated file. **Confirm with Nick which
> applies to prod before shipping.**

**Numbered migration (Postgres legacy + incremental SQLite):**
`go-backend/schema/0147-add-oidc-subject.sql`:
```sql
-- Migration: Add OIDC identity columns for generic SSO login
ALTER TABLE users ADD COLUMN oidc_provider TEXT;
ALTER TABLE users ADD COLUMN oidc_sub      TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_sub
  ON users (oidc_provider, oidc_sub) WHERE oidc_sub IS NOT NULL;
COMMENT ON COLUMN users.oidc_provider IS 'OIDC issuer/label, e.g. google, okta';
COMMENT ON COLUMN users.oidc_sub      IS 'Subject (`sub`) claim from the IdP id_token';
```
Next free number is **0147** (latest is `0146-drop-notification-triggers.sql`).

### Frontend

**`zettelkasten-front/src/pages/LoginPage.tsx`**
- Add a **"Continue with SSO"** button (conditionally rendered when
  `VITE_OIDC_ENABLED === 'true'`) that navigates to
  `${import.meta.env.VITE_URL}/auth/oidc/start`.
- The existing `?token=` callback effect already handles the post-redirect JWT
  via `loginUserFromToken` — **no new callback code needed.**

**`zettelkasten-front/.env` / docs** — document `VITE_OIDC_ENABLED` and an
optional `VITE_OIDC_LABEL` (e.g. "Continue with Google") for button text.

No changes to `AuthContext`, `api/auth.ts`, or `api/client.ts` — the OIDC path
funnels through the same JWT-in-localStorage mechanism as GitHub.

### Config / Env (new)

| Var | Required | Example | Notes |
|---|---|---|---|
| `OIDC_ENABLED` | no | `true` | If unset/`false`, OIDC routes 404 and frontend hides the button. |
| `OIDC_ISSUER` | if enabled | `https://accounts.google.com` | Used for discovery. |
| `OIDC_CLIENT_ID` | if enabled | `xxxx.apps.googleusercontent.com` | |
| `OIDC_CLIENT_SECRET` | if enabled | `GOCSPX-...` | Sensitive. |
| `OIDC_REDIRECT_URI` | if enabled | `https://app.zettelgarden.com/api/auth/oidc/callback` | Registered at the provider. |
| `VITE_OIDC_ENABLED` | no | `true` | Frontend flag to show the button. |
| `VITE_OIDC_LABEL` | no | `Google` | Optional button label. |

Add to `.env.example` (if present), `docker-compose.yml`, and the README config
section.

## Security Considerations

- **CSRF `state`:** current GitHub flow omits it. OIDC must use it; plan also
  retrofits GitHub to the shared signed-cookie helper.
- **`nonce`:** included in the auth request and validated inside the id_token
  to prevent replay. Stored alongside `state` in the signed cookie.
- **`email_verified`:** only auto-link an existing local account to an OIDC
  login when the IdP asserts `email_verified=true`. Otherwise create a new
  account (avoids account takeover via an unverified-email IdP).
- **Stable identity:** prefer `(provider, sub)` over email for matching —
  emails change; `sub` is immutable per IdP.
- **Token validation:** rely on `go-oidc`'s verifier (JWKS, `iss`, `aud`,
  `exp`); do not hand-verify. Pin `aud` to our `client_id`.
- **Cookie flags:** `HttpOnly`, `Secure` (prod), `SameSite=Lax` so the IdP
  redirect carries it back but JS can't read it.
- **Secrets:** reuse `SECRET_KEY` (already required) for the state-cookie
  HMAC; no new secret needed.
- **No password for OIDC users:** created users get a random, unusable
  password hash (mirror the GitHub `"github_oauth_<id>"` pattern but stronger
  — a random bcrypt of high entropy).

## Phased Implementation Plan

**Phase 1 — Backend core (no UI)**
1. Add deps to `go.mod` (`go-oidc/v3`, `oauth2`).
2. `OIDCConfig` + `loadOIDCConfig()` + wire into `Handler`/`Server`.
3. `handlers/oauth_state.go` signed-cookie helpers (with unit tests).
4. `handlers/oidc.go` start + callback (discovery, exchange, verify,
   find-or-create, issue JWT).
5. Register routes.
6. `models` + schema/migration (`0147` + `schema.sqlite.sql` edit).

**Phase 2 — Frontend**
7. Conditional "Continue with SSO" button + env flag.
8. Manual end-to-end against a test IdP (e.g. Google or a local Dex/Keycloak).

**Phase 3 — Hardening + tests**
9. Backend tests: callback happy path, bad state, expired cookie, bad nonce,
   email_verified=false, new vs existing user, `(provider,sub)` re-auth.
10. Retrofit GitHub flow to the shared `state` cookie (CSRF fix).
11. Docs: README config section, update `routes.go` public-route comment, add
    `.env.example`.

## Testing

- **Unit (`go test ./...`):**
  - signed state-cookie sign/verify/tamper/expiry.
  - `findOrCreateOIDCUser`: new user, match-by-sub, match-by-verified-email,
    reject-unverified-email-link.
  - Callback handler with a mocked provider/verifier (table-driven; inject a
    test `*oidc.Provider` or stub the verifier).
- **Frontend (`npm run test`):** render `LoginPage` with
  `VITE_OIDC_ENABLED=true`/`false` and assert button presence.
- **Integration/manual:** run against Google (or a local Dex container) — full
  login, account creation, re-login (sub match), and that the issued app JWT
  works on a protected route (`GET /api/auth`).
- **Regression:** GitHub login + email/password login unaffected.

## Open Decisions (need Nick's input)

1. **Single provider vs. multi-provider now?** Plan assumes one configurable
   provider (simplest correct step). Multi-provider is a clean follow-up
   (table/enum of providers + `provider` query param on start/callback).
2. **Account-linking UX:** should an existing email/password user logging in
   via OIDC for the first time be auto-linked (when `email_verified`) or
   prompted to confirm? Plan auto-links when verified; confirm this is desired.
3. **SQLite migration path for existing prod DBs:** edit consolidated
   `schema.sqlite.sql` only, or also ship `0147` into the sqlite scan dir for
   in-place upgrade? (See Database section.)
4. **Provider for initial testing:** Google, Okta, Auth0, or self-hosted
   Dex/Keycloak?

## Risks

- **Hand-rolled validation bugs** → mitigated by using `coreos/go-oidc`.
- **Clock skew** on `exp` → go-oidc defaults handle small skew; document.
- **Discovery fetch on cold start** adds latency to first login → cache the
  provider after first successful discovery; fall back gracefully.
- **`coreos/go-oidc` maintenance** → still the de-facto Go standard; monitor,
  but low risk for this scope.
