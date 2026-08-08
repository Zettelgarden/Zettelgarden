# Data Export & Account Deletion (6er.9)

A self-hosted Zettelgarden lets users get their data out and delete their
account without contacting anyone — no SaaS support address required.

## Data export

**UI:** Settings → Data & Privacy → "Export my data". Downloads
`zettelgarden-export-<user-id>-<date>.zip`.

**API:** `GET /api/user/export` (authenticated, self).

The zip contains:

- `user.json` — the profile (credentials stripped: password, api_key_hash,
  caldav_token are never exported).
- `tables/<name>.json` — one JSON file per user-owned table (~50 tables:
  cards, tasks, tags, entities, facts, RSS, chat, emails, api keys, …).
  Junction tables are scoped via a subquery on their owning parent.
- `cards.md` / `cards.csv` — human-readable renderings of your cards.
- `files/<id>_<name>` — the original uploaded file bytes (raw, from the local
  storage root).
- `export.json` — metadata (exported_at, user_id, format version).

Blob read failures are logged and skipped; the metadata still ships. The
export never mutates anything.

## Account deletion

**UI:** Settings → Data & Privacy → "Delete account" (asks for your password;
GitHub/OIDC accounts without a local password can leave it empty). Admins can
also delete any user from Admin → user detail → "Delete User".

**API:**

- `DELETE /api/user` (authenticated, self) — body `{"password": "..."}`.
- `DELETE /api/users/{id}` (admin only).

### Behavior

- **Password confirmation** is required for local accounts (`403` on mismatch).
  Users provisioned via GitHub/OIDC have no local password and are
  authenticated by their session token.
- **Last-admin guard**: the final remaining admin account cannot be deleted
  (`400 cannot delete the last admin account`) — otherwise the instance would
  lose access to the admin settings UI.
- **Cascade**: the `users` row delete cascades via SQLite `ON DELETE CASCADE`
  to all ~40 user-owned tables. A small set of tables that predate the FK
  (`task_saved_searches`, `starred_cards`, `inactive_cards`, `keywords`,
  `entity_card_junction`, `files`) is cleaned explicitly first.
- **File blobs**: the on-disk bytes in the storage root are removed after the
  DB commit, best-effort — a failure leaves harmless orphan bytes but never
  blocks deletion.
- The deleted user's JWTs stop working immediately (the user row is gone).

### Tests

`go-backend/handlers/account_test.go` covers the export zip contents, wrong
password, self-delete cascade (including non-cascade tables and blob removal),
the last-admin guard, and admin deletion. `go-backend/services/account.go`
holds the delete service (`DeleteUserData`) shared by both routes.
