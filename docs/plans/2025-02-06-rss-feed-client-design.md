# RSS Feed Client Design

**Date:** 2025-02-06
**Status:** Design Approved

## Overview

Add an RSS feed client to Zettelgarden that allows users to subscribe to feeds, browse articles in a reader-style inbox, and selectively convert interesting articles to cards. This follows an "inbox flow" pattern where articles are staged first, then promoted to cards.

## Key Features

1. **Pull articles** - Scheduled polling of RSS feeds at configurable intervals
2. **Render articles** - Reader view using readability parsing (reuses existing article infrastructure)
3. **Turn into cards** - Selective conversion with auto-tagging per feed

## Workflow

```
Add RSS Feed → Scheduled Fetch → Browse RSS Inbox → Convert to Card
                    (background)       (read, filter)    (selective)
```

## Architecture

### Database Tables

#### Feeds Table
```sql
CREATE TABLE rss_feeds (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    url TEXT NOT NULL,
    name TEXT NOT NULL,
    folder TEXT,
    auto_tags TEXT DEFAULT '',  -- comma-separated
    fetch_interval INTEGER DEFAULT 60,  -- minutes
    last_fetched_at TIMESTAMP,
    last_error TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, url)
);
```

#### Articles Table
```sql
CREATE TABLE rss_articles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    feed_id INTEGER REFERENCES rss_feeds(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT,
    author TEXT,
    url TEXT NOT NULL,
    published_at TIMESTAMP,
    fetched_at TIMESTAMP DEFAULT NOW(),
    read BOOLEAN DEFAULT false,
    UNIQUE(user_id, url)
);
```

#### Folders Table
```sql
CREATE TABLE rss_folders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    name TEXT NOT NULL,
    order_index INTEGER DEFAULT 0,
    UNIQUE(user_id, name)
);
```

## Backend API Routes

**File:** `go-backend/handlers/rss.go`

### Feed Management
- `POST /api/rss/feeds` - Add new feed
- `GET /api/rss/feeds` - List user's feeds, grouped by folder
- `PUT /api/rss/feeds/{id}` - Update feed (name, folder, auto_tags, enabled)
- `DELETE /api/rss/feeds/{id}` - Remove feed
- `POST /api/rss/feeds/fetch` - Manual refresh trigger

### Article Management
- `GET /api/rss/articles` - List with filters (`?folder=tech&unread=true&limit=50`)
- `GET /api/rss/articles/{id}` - Get single article
- `POST /api/rss/articles/{id}/read` - Mark as read/unread
- `POST /api/rss/articles/{id}/convert` - Convert to card

### Folders
- `GET /api/rss/folders` - List user's folders
- `PUT /api/rss/folders/reorder` - Change folder order

## Frontend UI

**Page:** `zettelkasten-front/src/pages/RssPage.tsx`

### Three-Panel Layout
```
┌─────────────┬────────────────────┬────────────────────┐
│  Folders    │   Article List     │  Article Reader    │
│             │                    │                    │
│  Tech       │  Feed: All ▼       │  [Parsed Content]  │
│  News       │  Filter: Unread ▼  │                    │
│  Personal   │                    │  [Convert]         │
│             │  ────────────────  │  [Edit & Convert]  │
│  [Add Feed] │  Article Title     │  [Mark Read]       │
│             │  Excerpt...        │                    │
│             │  2 hours ago       │                    │
└─────────────┴────────────────────┴────────────────────┘
```

### Components
- **RssFeedList** - Sidebar with folders
- **RssArticleList** - Middle panel with filters
- **RssArticleReader** - Right panel, readability view
- **RssConvertDialog** - Edit before converting modal

### Navigation
Add "RSS" link to main sidebar (below Files/Chat)

## Data Flow

### Scheduled Fetch
```
scheduler (every 60min)
  → rss_fetch_job
    → For each enabled feed:
        1. Fetch RSS XML (gofeed library)
        2. Parse feed items
        3. For each new article (by URL):
           - Fetch full content (readability)
           - Convert to markdown
           - Insert to rss_articles
```

### Convert to Card
```
POST /api/rss/articles/{id}/convert
  → Get article + feed.auto_tags
  → Create card with article content
  → Mark article as read
  → Return new card_id
```

## Dependencies

### Backend
- `github.com/mmcdole/gofeed` - RSS/Atom/JSON Feed parser
- Existing `github.com/go-shiori/go-readability` - Article extraction
- Existing `github.com/JohannesKaufmann/html-to-markdown/v2` - HTML to markdown

### Frontend
- No new dependencies (reuses existing patterns)

## Testing

### Backend
- **`handlers/rss_test.go`** - CRUD operations, convert endpoint, deduplication
- **`services/rss_test.go`** - RSS parsing, content extraction, error handling

### Frontend
- **`pages/__tests__/RssPage.test.tsx`** - Render, filters, convert dialog

### Fixtures
- `tests/fixtures/rss/valid-rss.xml`
- `tests/fixtures/rss/atom-feed.xml`
- `tests/fixtures/rss/json-feed.json`

## Migration

Database migration file: `go-backend/schema/XXX_add_rss_tables.sql`

## Future Considerations

- Feed discovery/opml import
- Full-text search across RSS articles
- Per-feed custom fetch intervals
- Feed-specific filtering rules
