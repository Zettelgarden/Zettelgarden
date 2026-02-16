# Email Sync: JMAP to IMAP Migration

**Date:** 2026-02-15
**Status:** Proposed (Migration from JMAP)
**Reason:** JMAP authentication issues, prefer standard IMAP protocol

## Current Implementation (JMAP)

### What Was Built

A complete email sync feature using Fastmail's JMAP protocol was implemented in Phase 1 (Foundation):

**Backend (Go):**
- `schema/0115-add-email-sync-tables.sql` - Database tables for email sync
- `schema/0116-rename-app-password-to-api-token.sql` - Column rename for API tokens
- `models/email_sync.go` - Go model types (EmailAccount, Email, etc.)
- `services/email_accounts.go` - EmailAccount CRUD service with encryption
- `services/emails.go` - Email storage service with upsert logic
- `services/jmap_client.go` - JMAP client for Fastmail communication
- `services/jobs/email_sync_job.go` - Scheduled job for email fetching
- `handlers/email_sync.go` - HTTP handlers for accounts and emails
- `routes/email.go` - Route registration

**Frontend (React/TypeScript):**
- `src/api/email.ts` - API client functions
- `src/pages/EmailInboxPage.tsx` - Inbox page with filters and account setup
- `src/components/email/EmailList.tsx` - Email list component
- `src/components/email/EmailRow.tsx` - Individual email row component

### Current Issues

1. **Authentication Error:**
   ```
   JMAP API returned status 403: JMAP capabilities requested but urn:ietf:params:jmap:core
   is not present: urn:ietf:params:jmap:mail
   ```
   - Fixed by adding both core and mail capabilities to `Using` field
   - May have additional issues not yet debugged

2. **Authentication Method:**
   - JMAP now requires Bearer tokens (API tokens) instead of Basic auth with app passwords
   - API tokens must be generated from Fastmail → Settings → API
   - Old app passwords no longer work with JMAP

3. **Feature Hidden:**
   - Email link removed from sidebar (`src/components/sidebar/NavigationLinks.tsx`)
   - Still accessible via direct URL `/app/emails` but not discoverable

## Why Switch to IMAP?

### Advantages of IMAP

1. **Universal Standard** - Works with Fastmail, Gmail, Outlook, and all email providers
2. **App Passwords** - Uses traditional app passwords that users are familiar with
3. **Mature Protocol** - Well-documented, stable, widely supported
4. **No Fastmail-Specific** - Not tied to Fastmail's proprietary implementation

### Disadvantages of IMAP

1. **Older Protocol** - Binary protocol, less efficient than JSON/HTTP
2. **More Complex** - Stateful connection, requires careful connection management
3. **Slower** - One request per email vs batch operations in JMAP
4. **Different Data Model** - Uses UIDs and sequence numbers vs JMAP message IDs

## Migration Plan: JMAP to IMAP

### 1. Replace JMAP Client with IMAP Client

**File to modify:** `services/jmap_client.go` → Rename to `services/imap_client.go`

**Recommended library:** `github.com/emersion/go-imap` or `github.com/emersion/go-imap/v2`

**New structure:**
```go
type IMAPClient struct {
    conn   *imap.Client
    server string
    username string
    password string
    mailbox string
}

func (c *IMAPClient) Connect(ctx context.Context) error
func (c *IMAPClient) SelectInbox(ctx context.Context) error
func (c *IMAPClient) FetchRecentEmails(ctx context.Context, limit int) ([]models.Email, error)
func (c *IMAPClient) Close() error
```

### 2. Update Authentication

**Current (JMAP):**
- Uses Bearer token: `Authorization: Bearer fmu1-...`
- API token generated from Fastmail → Settings → API

**New (IMAP):**
- Uses app password with PLAIN or LOGIN authentication
- App password generated from Fastmail → Settings → Password & Security

### 3. Update Data Model

**JMAP message ID:** String, provided by Fastmail
**IMAP UID:** uint32, mailbox-specific

**Migration needed:**
- `emails.message_id` currently stores JMAP message ID (string)
- For IMAP, store IMAP UID (uint32) + mailbox name
- Consider adding `uid` and `uid_validity` columns to emails table

### 4. Update Scheduled Job

**File:** `services/jobs/email_sync_job.go`

**Changes:**
- Replace JMAP client instantiation with IMAP client
- Update sync logic to handle IMAP connection lifecycle
- Handle IMAP-specific errors (connection drops, IDLE support, etc.)

### 5. Update API Models

**File:** `models/email_sync.go`

**Changes:**
- Rename `api_token_encrypted` back to `app_password_encrypted`
- Or keep separate if supporting both JMAP and IMAP
- Update `CreateEmailAccountParams` to use `app_password`

### 6. Update Frontend

**Files:**
- `src/api/email.ts`
- `src/pages/EmailInboxPage.tsx`

**Changes:**
- Change "API Token" back to "App Password"
- Update help text to reference Password & Security settings
- Re-enable sidebar link when working

## Database Migration Needed

```sql
-- Add IMAP-specific columns
ALTER TABLE emails ADD COLUMN imap_uid BIGINT;
ALTER TABLE emails ADD COLUMN imap_uid_validity BIGINT;
ALTER TABLE email_accounts ADD COLUMN imap_server TEXT DEFAULT 'imap.fastmail.com:993';
ALTER TABLE email_accounts ADD COLUMN imap_server_type TEXT DEFAULT 'imap';

-- Update column naming for clarity
ALTER TABLE email_accounts RENAME COLUMN api_token_encrypted TO app_password_encrypted;

-- Add index for IMAP UID lookups
CREATE INDEX idx_emails_imap_uid ON emails(user_id, imap_uid);
```

## IMAP Server Settings (Fastmail)

- **Server:** `imap.fastmail.com`
- **Port:** `993` (TLS/SSL)
- **Authentication:** PLAIN or LOGIN
- **Username:** Full email address
- **Password:** App password

## Implementation Order

1. Create new IMAP client service
2. Write unit tests for IMAP client (mock IMAP server)
3. Update database schema (new migration file)
4. Update email sync job to use IMAP client
5. Update API handlers to expect app_password
6. Update frontend to use app password field
7. Integration testing with real Fastmail account
8. Re-enable sidebar link
9. Update documentation

## Testing Strategy

1. **Unit Tests:** Mock IMAP server responses
2. **Integration Tests:** Test with real Fastmail account
3. **Error Handling:** Test connection failures, auth failures, timeouts
4. **Incremental Sync:** Test UIDVALIDITY changes, mailbox state
5. **Performance:** Compare sync times with large mailboxes

## Rollback Plan

If IMAP implementation fails:
1. Keep JMAP code in separate branch for reference
2. Can revert by:
   - Restoring pre-migration database schema
   - Reverting `jmap_client.go` from git
   - Reverting frontend changes

## References

- [go-imap library](https://github.com/emersion/go-imap)
- [Fastmail IMAP settings](https://www.fastmail.help/hc/en-us/articles/1500000393171-IMAP-OAuth2-and-app-passwords)
- [IMAP protocol RFC 3501](https://tools.ietf.org/html/rfc3501)
- [Current JMAP implementation](./2026-02-15-email-sync-design.md)
