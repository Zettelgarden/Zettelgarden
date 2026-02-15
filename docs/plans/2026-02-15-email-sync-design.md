# Email Sync with Fastmail - Design Document

**Date:** 2026-02-15
**Status:** Approved
**Feature Tier:** Free (available to all users)

## Overview

Add email synchronization with Fastmail to Zettelgarden, enabling AI-powered inbox triage with "Inbox Zero" workflow. Users can review AI decisions to archive/delete emails, and selectively convert important emails to knowledge cards.

## User Goals

- Sync emails from Fastmail inbox
- AI automatically triages emails (archive/delete/keep/convert)
- Graduated trust: high-confidence actions auto-execute, others require review
- Convert valuable emails to cards with full content and metadata
- Manual trigger + scheduled hybrid approach

## Architecture

### Backend (Go)

| Component | Path | Purpose |
|-----------|------|---------|
| API handlers | `handlers/email_sync.go` | Email account & triage endpoints |
| Models | `models/email_sync.go` | Email storage queries |
| JMAP client | `services/email_fetcher.go` | Fastmail communication |
| Sync job | `jobs/email_sync_job.go` | Scheduled email fetch |
| Triage job | `jobs/email_triage_job.go` | Scheduled AI triage |
| Schema | `schema/0113-add-email-sync-tables.sql` | Database tables |

### Frontend (React/TypeScript)

| Component | Path | Purpose |
|-----------|------|---------|
| Inbox page | `src/pages/EmailInboxPage.tsx` | Main inbox UI |
| Email components | `src/components/email/*` | List, detail, modals |
| API client | `src/api/email.ts` | Backend communication |

## Database Schema

### `email_accounts`

```sql
id                  SERIAL PRIMARY KEY
user_id             INTEGER REFERENCES users(id)
email_address       TEXT NOT NULL
jmap_server_url     TEXT NOT NULL DEFAULT 'https://api.fastmail.com/jmap/session'
app_password_encrypted TEXT
is_active           BOOLEAN DEFAULT true
last_sync_at        TIMESTAMP
sync_status         TEXT DEFAULT 'active' -- active, error, disabled
created_at          TIMESTAMP DEFAULT NOW()
updated_at          TIMESTAMP DEFAULT NOW()
```

### `emails`

```sql
id                  SERIAL PRIMARY KEY
user_id             INTEGER REFERENCES users(id)
email_account_id    INTEGER REFERENCES email_accounts(id)
message_id          TEXT NOT NULL UNIQUE -- JMAP message ID
thread_id           TEXT -- JMAP thread ID
subject             TEXT
from_address        TEXT
from_name           TEXT
body_text           TEXT
body_html           TEXT
received_at         TIMESTAMP
status              TEXT DEFAULT 'unprocessed' -- unprocessed, triaged, reviewed, archived, deleted, converted
folder              TEXT DEFAULT 'Inbox'
created_at          TIMESTAMP DEFAULT NOW()
updated_at          TIMESTAMP DEFAULT NOW()
```

### `email_triage_decisions`

```sql
id                  SERIAL PRIMARY KEY
email_id            INTEGER REFERENCES emails(id)
decision            TEXT NOT NULL -- archive, delete, keep, convert_to_card
confidence          FLOAT NOT NULL -- 0-1
reasoning           TEXT
is_auto_executed    BOOLEAN DEFAULT false
created_at          TIMESTAMP DEFAULT NOW()
```

### `email_card_links`

```sql
id                  SERIAL PRIMARY KEY
email_id            INTEGER REFERENCES emails(id)
card_id             INTEGER REFERENCES cards(id)
created_at          TIMESTAMP DEFAULT NOW()
```

## API Endpoints

### Account Management
- `POST /api/email/accounts` - Add Fastmail account
- `GET /api/email/accounts` - List accounts
- `DELETE /api/email/accounts/:id` - Remove account
- `POST /api/email/accounts/:id/sync` - Manual sync trigger
- `PUT /api/email/accounts/:id` - Update settings

### Email Retrieval
- `GET /api/emails` - List with filters (status, folder, pagination)
- `GET /api/emails/:id` - Get single email with triage decision
- `GET /api/emails/stats` - Counts by status

### Triage & Actions
- `POST /api/emails/:id/triage` - Request AI triage
- `POST /api/emails/batch-triage` - Batch AI triage
- `POST /api/emails/:id/approve` - Approve AI decision
- `POST /api/emails/:id/reject` - Reject AI decision
- `POST /api/emails/batch-approve` - Batch approve
- `POST /api/emails/:id/convert` - Convert to card
- `POST /api/emails/:id/archive` - Manual archive
- `POST /api/emails/:id/delete` - Manual delete

## JMAP Integration

**Fastmail endpoints:**
- Session: `https://api.fastmail.com/jmap/session`
- Operations: `Email/get`, `Email/query`, `Email/changes`, `Email/set`, `Mailbox/get`

**Sync strategy:**
1. Initial: Fetch all Inbox emails via `Email/query`
2. Incremental: `Email/changes` with state token
3. Store state token in `email_accounts` table

**Go library:** `github.com/bradenaw/jmap` or custom HTTP/JSON client

## AI Triage Logic

### Categories
| Decision | Description |
|----------|-------------|
| **Archive** | Low-value: receipts, confirmations, shipping notices |
| **Delete** | Spam, acted-upon marketing, duplicates |
| **Keep** | Requires attention, personal, important |
| **Convert** | Knowledge worth preserving: articles, ideas, references |

### Graduated Trust
| Confidence | Action |
|------------|--------|
| ≥ 0.9 | Auto-execute archive/delete |
| ≥ 0.7 | Show pre-selected in review queue |
| < 0.7 | Flag "unsure", require manual |

### Prompt Structure
```
You are an email triage assistant. Analyze this email:
1. Action: archive/delete/keep/convert_to_card
2. Confidence: 0-1
3. Reasoning: brief explanation

Context: User wants inbox zero, values knowledge retention.
Consider: sender relationship, content type, action required, reference value.

Email: [subject, from, body preview]
```

## Frontend UI

### Pages & Components
- `EmailInboxPage` - Main inbox with filters (All, Unprocessed, Triaged)
- `EmailRow` - Sender, subject, date, AI badge, confidence
- `EmailDetailModal` - Full email + reasoning + action buttons
- `EmailAccountSetup` - Fastmail credential form

### Keyboard Shortcuts
- `e` - Open email inbox
- `a` - Approve decision
- `r` - Reject decision
- `c` - Convert to card

### Sidebar
- "Email Inbox" with unprocessed count badge

## Scheduled Jobs

Using existing `services.ScheduledJob` interface:

| Job | Schedule | Purpose |
|-----|----------|---------|
| `email-sync` | Every 60 min | Fetch new emails from Fastmail |
| `email-triage` | After sync | Run AI triage on unprocessed |

Configurable via `EMAIL_SYNC_INTERVAL_MINUTES` env var (default: 60).

Both visible in Admin > Scheduled Jobs with history and success rates.

## Error Handling

| Scenario | Handling |
|----------|----------|
| Auth invalid | Disable account, notify user |
| Network timeout | Retry 3x with backoff |
| Rate limited | Respect Retry-After header |
| Email deleted externally | Reconcile on next sync |
| Large emails (>5MB) | Store text, truncate preview |
| Permission denied | Log error, mark status=error |

## Card Conversion

When converting email to card:
- **Content:** Full email body (subject + body)
- **Metadata:** From, to, date, subject stored as facts
- **Link:** `email_card_links` table maintains relationship
- **Source:** Link back to original Fastmail email via message ID

## Implementation Phases

### Phase 1: Foundation (MVP)
- Database schema
- JMAP fetch client
- Email sync job
- List APIs + read-only inbox UI

### Phase 2: AI Triage
- LLM integration
- Triage decisions storage
- Inbox UI with recommendations
- Manual approve/reject

### Phase 3: Actions & Cards
- JMAP archive/delete execution
- Convert to card functionality
- Email-card metadata linking
- Graduated trust auto-execution

### Phase 4: Polish
- Batch actions
- Keyboard shortcuts
- Admin monitoring
- Edge case handling

## Testing

- **Unit:** Mock JMAP/LLM responses
- **Integration:** Full flow with test DB
- **Fixtures:** Sample emails (newsletter, personal, receipt)
- **Manual:** Fastmail test account for E2E
