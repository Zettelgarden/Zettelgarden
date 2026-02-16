# RSS Article Starring Feature Design

**Date**: 2026-02-15
**Status**: Approved

## Overview

Add ability to star/unstar RSS articles with a separate "Starred" feed in the sidebar. Star toggles appear in both the article list and reader view.

## Architecture

### Database Schema

**Migration**: Add `is_starred` boolean column to `rss_articles` table

```sql
ALTER TABLE rss_articles ADD COLUMN is_starred BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_rss_articles_starred ON rss_articles(user_id, is_starred);
```

- `is_starred` defaults to `FALSE`
- Index on `(user_id, is_starred)` for efficient filtering
- Follows the same pattern as the existing `read` column

### Backend API

**New Routes** (`go-backend/routes/rss.go`):
- `POST /api/rss/articles/{id}/star` - Star an article
- `DELETE /api/rss/articles/{id}/star` - Unstar an article

**Modified Route**:
- `GET /api/rss/articles` - Add optional `?starred=true` query parameter

**Handlers** (`go-backend/handlers/` - new file or added to existing RSS handlers):
- `StarRSSArticleRoute(userID, articleID)` - Inserts/updates `is_starred = true`
- `UnstarRSSArticleRoute(userID, articleID)` - Sets `is_starred = false`
- Modify `ListRSSArticlesRoute` to accept `starred` filter parameter

**Models Update** (`go-backend/models/rss.go`):
- Add `IsStarred bool` field to `RSSArticle` struct

### Frontend API

**Type Update** (`zettelkasten-front/src/api/rss.ts`):
- Add `is_starred?: boolean` to `RSSArticle` interface

**New API Functions**:
```typescript
export async function starArticle(articleId: number): Promise<void>
export async function unstarArticle(articleId: number): Promise<void>
```

**Modified Function**:
- `ArticleFilters` interface: Add `starred?: boolean` option

### UI Components

**Type**: `RSSArticle` gets `is_starred?: boolean` field

**RssArticlesPanel** (`components/rss/RssArticlesPanel.tsx`):
- Add star icon next to each article (next to existing smart score/card indicators)
- Star icon toggles on click (stop propagation to prevent article selection)

**RssReaderPanel** (`components/rss/RssReaderPanel.tsx`):
- Add "Star" action button in toolbar (alongside Convert/Mark as Unread)
- Button shows filled/unfilled star based on current state

**RssFeedsPanel** (`components/rss/RssFeedsPanel.tsx`):
- Add new "Starred" feed item at top of feeds list (below Smart Feed)
- Star icon + "Starred" label
- Shows count of starred articles
- Clicking sets `isStarredFeedActive = true`

**RssPage** (`pages/RssPage.tsx`):
- Add state: `isStarredFeedActive: boolean`
- Add handlers: `handleStarArticle`, `handleUnstarArticle`
- Pass starred filter to articles hook when starred feed is active
- Add starred article count calculation

**Mobile Layout** (`components/rss/RssMobileLayout.tsx`):
- Mirror all desktop changes for mobile

## Data Flow

### Starring an Article

1. User clicks star icon in article list or reader
2. `handleStarArticle(articleId)` called
3. Call `starArticle(articleId)` API function
4. On success: `updateArticle(articleId, { is_starred: true })`
5. Article re-renders with filled star

### Unstarring

Same flow with `unstarArticle` and `is_starred: false`.

### Viewing Starred Feed

1. User clicks "Starred" in feeds sidebar
2. `isStarredFeedActive = true`, `selectedFolder = null`, `selectedFeedId = null`
3. Articles hook called with `filters: { starred: true }`
4. API request: `GET /rss/articles?starred=true`
5. Filtered articles displayed

## Error Handling

- API failures show toast message (same pattern as mark as read)
- Star toggle is optimistic: UI updates immediately, rolls back on error
- Duplicate star requests ignored (backend handles idempotently)

## Testing

- Unit tests for backend star/unstar handlers
- Integration test for starred filter
- Frontend: verify star toggle updates UI
- Frontend: verify starred feed shows only starred articles
