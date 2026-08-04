-- Migration: Add OIDC identity columns for generic SSO login
-- Date: 2026-08-03
--
-- Supports the OIDC authentication feature
-- (docs/plans/2026-08-03-oidc-authentication-design.md). The IdP in production
-- is Pocket ID, but the columns are provider-agnostic.
--
-- Stores the IdP issuer label and the stable `sub` claim so returning OIDC
-- users re-authenticate deterministically regardless of email changes. We
-- resolve users in this order on each login:
--   1. (oidc_provider, oidc_sub)   -- stable per-user, per-IdP
--   2. email  (only when id_token email_verified == true; auto-link)
--   3. create new account
--
-- Schema-path notes:
--   * Postgres path: this migration runs in place via the numbered-migration
--     runner in server/database.go.
--   * SQLite path: the consolidated schema/sqlite/schema.sqlite.sql carries
--     these same columns + index for fresh builds. The numbered-migration
--     runner does NOT scan this file on SQLite (and the IF NOT EXISTS /
--     COMMENT syntax below is Postgres-only), so for an EXISTING SQLite DB the
--     columns are back-filled idempotently by ensureSQLiteSchemaUpgrades()
--     in server/database.go (runs on every boot). No manual ALTER is needed.

ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_provider TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS oidc_sub      TEXT;

-- Unique partial index: one local account per (provider, subject). NULL
-- oidc_sub rows (password / GitHub / API-key users) are excluded so the
-- uniqueness constraint only applies to OIDC-linked accounts.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_sub
  ON users (oidc_provider, oidc_sub) WHERE oidc_sub IS NOT NULL;

COMMENT ON COLUMN users.oidc_provider IS 'OIDC issuer/label (e.g. pocket-id, google)';
COMMENT ON COLUMN users.oidc_sub IS 'Subject (sub) claim from the IdP id_token; stable per-user, per-IdP';
