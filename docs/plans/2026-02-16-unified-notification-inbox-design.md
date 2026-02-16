# Unified Notification Inbox Design

**Date:** 2026-02-16
**Status:** Approved

## Overview

A unified inbox that consolidates important notifications from email, RSS feeds, and eventually tasks. The goal is to filter the "firehose" of information to what really matters, with user-customizable filter toggles.

## Requirements

1. **Consolidate important items** from email and RSS sources (tasks later)
2. **Keep existing pages intact** - EmailInboxPage and RssPage remain unchanged
3. **Flexible architecture** for adding future notification types
4. **Simple user customization** via toggle filters (no complex rule builder yet)
5. **Hybrid importance model** - smart signals + manual curation (starred items)

## Design Decisions

### UI Organization
- **Combined approach**: Unified chronological list with source-specific tabs
- Tabs: All | Email | RSS | Tasks (future)
- Visual indicators (icons/badges) show source type
- Clicking item navigates to original detail page

### Importance Criteria
- **Smart signals**: Unprocessed emails, priority RSS feeds, smart scores
- **Manual curation**: Starred articles, flagged emails
- **User toggles**: Simple on/off per category

### Customization
- Simple toggles per source (show_unprocessed_emails, show_starred_articles, etc.)
- Predefined filter presets considered for future enhancement

## Architecture

### Approach: Unified Notification Table

A new `notifications` table aggregates items from all sources. This provides:
- Fast single-table queries
- Easy extensibility for new sources
- Pre-computed importance scores
- Clean API endpoints

Trade-off: Requires sync logic (triggers) to keep table aligned with source tables.

## Database Schema

### notifications Table

```sql
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    source_type VARCHAR(20) NOT NULL,  -- 'email', 'rss', 'task'
    source_id INTEGER NOT NULL,         -- foreign key to source table
    title TEXT NOT NULL,
    preview TEXT,                       -- brief content preview
    timestamp TIMESTAMPTZ NOT NULL,     -- normalized timestamp from source
    importance_score INTEGER DEFAULT 0,
    is_read BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    filter_tags TEXT[],                 -- e.g., '{priority,unprocessed,starred}'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, source_type, source_id)
);

CREATE INDEX idx_notifications_user_timestamp ON notifications(user_id, timestamp DESC);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read, is_archived)
    WHERE is_read = FALSE AND is_archived = FALSE;
CREATE INDEX idx_notifications_filter_tags ON notifications USING GIN(filter_tags);
```

### notification_preferences Table

```sql
CREATE TABLE notification_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    -- Filter toggles
    show_unprocessed_emails BOOLEAN DEFAULT TRUE,
    show_starred_articles BOOLEAN DEFAULT TRUE,
    show_priority_tasks BOOLEAN DEFAULT TRUE,
    show_priority_feeds BOOLEAN DEFAULT TRUE,
    -- Other settings
    items_per_page INTEGER DEFAULT 50,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Sync Triggers

Database triggers on `emails` and `rss_articles` tables automatically maintain the notifications table:

- **Email trigger**: Fires on INSERT/UPDATE, syncs unprocessed/triaged emails
- **RSS trigger**: Fires on INSERT/UPDATE, syncs starred and priority feed articles
- **Task trigger** (future): Similar pattern for tasks

## Backend API

### Routes

```
GET  /notifications                    # List notifications (paginated)
GET  /notifications/unread-count       # Get count of unread notifications
PATCH /notifications/:id/read          # Mark as read/unread
PATCH /notifications/:id/archive       # Archive/unarchive
GET  /notifications/preferences        # Get user preferences
PATCH /notifications/preferences       # Update user preferences
POST /notifications/sync               # Manual sync trigger (dev/debug)
```

### Handler Structure

`go-backend/handlers/notifications.go`:

```go
type Notification struct {
    ID              int
    UserID          int
    SourceType      string    // 'email', 'rss', 'task'
    SourceID        int
    Title           string
    Preview         string
    Timestamp       time.Time
    ImportanceScore int
    IsRead          bool
    IsArchived      bool
    FilterTags      []string
}

type NotificationListFilters struct {
    SourceType  string   // optional: filter by source
    UnreadOnly  bool     // default: true
    Limit       int      // default: 50
    Offset      int      // for pagination
}
```

### Sync Service

`go-backend/services/notifications.go`:

Importance scoring logic:
- **Email**: unprocessed=10, triaged=5, archived=0
- **RSS**: starred=10, priority feed=5 + smart score bonus
- **Task** (future): due soon=10, high priority=5, overdue=15

## Frontend Components

### New Page Component

`zettelkasten-front/src/pages/NotificationInboxPage.tsx`:
- Tabbed interface: All | Email | RSS | Tasks
- Unified list display with source indicators
- Navigate to source detail pages on click
- Quick actions: mark read, archive

### API Client

`zettelkasten-front/src/api/notifications.ts`:
```typescript
export interface Notification {
    id: number;
    source_type: 'email' | 'rss' | 'task';
    source_id: number;
    title: string;
    preview: string;
    timestamp: string;
    importance_score: number;
    is_read: boolean;
    is_archived: boolean;
}

export interface NotificationPreferences {
    show_unprocessed_emails: boolean;
    show_starred_articles: boolean;
    show_priority_tasks: boolean;
    show_priority_feeds: boolean;
    items_per_page: number;
}
```

### Navigation

Add "Inbox" link to sidebar navigation between Dashboard and RSS, with unread count badge.

## Data Flow

```
┌─────────────────┐     ┌─────────────────┐
│  Email Source   │     │   RSS Source    │
│   (IMAP sync)   │     │   (fetch job)   │
└────────┬────────┘     └────────┬────────┘
         │                       │
         │ INSERT/UPDATE         │ INSERT/UPDATE
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│   emails table  │     │ rss_articles    │
└────────┬────────┘     └────────┬────────┘
         │                       │
         │ TRIGGER               │ TRIGGER
         │                       │
         └───────────┬───────────┘
                     ▼
         ┌───────────────────────┐
         │   notifications table │
         └───────────┬───────────┘
                     │ API call
                     ▼
         ┌───────────────────────┐
         │  NotificationInboxPage │
         └───────────────────────┘
```

## Future Enhancements

1. **Task notifications** - Add tasks as a notification source
2. **Filter presets** - "Minimal", "Balanced", "Everything" modes
3. **Advanced rules** - Per-sender email filters, per-feed RSS filters
4. **Inline preview** - Expandable rows instead of navigation (user preference)
5. **Push notifications** - Real-time updates when new items arrive

## Migration Plan

1. Create new tables (`notifications`, `notification_preferences`)
2. Create sync triggers for email and RSS
3. Backfill existing data into notifications table
4. Add backend handlers and routes
5. Add frontend components and API client
6. Add sidebar navigation link
7. Testing and verification
