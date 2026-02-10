# Per-Feed Refresh Feature Design

**Date:** 2026-02-09
**Status:** Design Approved

## Overview

Add per-feed refresh functionality to the RSS page, allowing users to refresh individual feeds instead of all feeds at once.

## Requirements

### Functional Requirements
1. User can refresh a single feed via the feed menu (three dots)
2. Toast notification shows refresh status (loading, success, error)
3. Unread counts update immediately after successful refresh
4. Works on both desktop and mobile layouts
5. Returns number of new articles fetched

### Non-Functional Requirements
- Reuses existing feed fetching logic
- Consistent UI patterns with existing menu system
- Non-blocking - user can continue using app during refresh

## Architecture

### Backend (Go)

**New API Endpoint:**
```
POST /api/rss/feeds/{id}/fetch
Response: {
  "fetched": 5,           // number of new articles
  "unread_count": 12      // new unread count for this feed
}
```

**Handler Function:** `RefreshSingleFeedRoute` in `handlers/rss.go`

**Request/Response Types:**
```go
type RefreshFeedResponse struct {
    Fetched      int `json:"fetched"`
    UnreadCount  int `json:"unread_count"`
}
```

**Implementation Notes:**
- Validate user owns the feed (403 if not)
- Return 404 if feed doesn't exist
- Reuse existing feed fetching logic
- Return updated unread count to avoid extra API call

### Frontend (React/TypeScript)

**New API Function:** `src/api/rss.ts`
```typescript
export interface RefreshFeedResponse {
  fetched: number;
  unread_count: number;
}

export function refreshFeed(feedId: number): Promise<RefreshFeedResponse> {
  return getData(apiClient.post<RefreshFeedResponse>(`/rss/feeds/${feedId}/fetch`, {}));
}
```

**Components to Modify:**
1. `src/components/rss/RssFeedItem.tsx` - Add "Refresh" menu item
2. `src/components/rss/RssFeedsPanel.tsx` - Pass handler, show toasts
3. `src/components/rss/RssMobileLayout.tsx` - Same changes for mobile
4. `src/pages/RssPage.tsx` - Add `handleRefreshFeed` function

## UI Design

### Feed Menu

```
┌─────────────────────────────────┐
│ My Tech Blog             (12) ⋮ │
├─────────────────────────────────┤
│ ▼ Mark as read                   │
│   Edit feed                      │
│   Refresh          ← NEW        │
│   Delete feed                    │
└─────────────────────────────────┘
```

### Toast Notifications

**Loading:**
```
ℹ️ Refreshing My Tech Blog...
```

**Success:**
```
✅ Refreshed My Tech Blog - 5 new articles
```

**Error:**
```
❌ Failed to refresh My Tech Blog: Connection timeout
```

## Data Flow

```
User Action                    Frontend                    Backend
─────────────────────────────────────────────────────────────────────
Click "Refresh"    →    Call refreshFeed(id)    →    Validate ownership
                         Show loading toast           Fetch feed
                         Wait for response            Parse articles
                                                      Store in DB
                                                      Return counts
                      ←    Update unread count   ←
                      ←    Show success toast    ←
                      ←    Update UI state       ←
```

## Error Handling

| Error Case | Status Code | Toast Message |
|------------|-------------|---------------|
| Feed not found | 404 | "Feed not found" |
| Not owned by user | 403 | "You don't have permission to refresh this feed" |
| Network timeout | 500 | "Connection timeout while fetching feed" |
| Parse error | 500 | "Failed to parse feed content" |
| Other error | 500 | "Failed to refresh feed: {details}" |

## Testing

### Backend Tests (`handlers/rss_test.go`)
- Valid feed ID owned by user → returns fetched count and unread count
- Non-existent feed ID → 404 Not Found
- Feed ID owned by different user → 403 Forbidden
- Network error fetching feed → 500 with error message
- No new articles → returns 0 fetched, current unread count

### Frontend Tests
- Click refresh with valid feed → shows loading toast
- Successful refresh → success toast with article count
- Failed refresh → error toast with message
- Unread count updates in UI after successful refresh
- Refresh works on mobile layout

## Edge Cases

- **Duplicate requests:** Ignore or debounce if user clicks refresh multiple times
- **Concurrent refreshes:** Backend should handle gracefully
- **Feed disabled during refresh:** Handle without errors
- **Very slow feeds:** Reuse existing timeout from bulk refresh
- **Mobile toast display:** Ensure toasts are visible on small screens

## Implementation Notes

1. Route registration: Add to `routes/rss.go` after existing feed routes
2. Auth middleware: Same as other RSS routes
3. Feed fetching: May need to refactor bulk refresh logic for reusability
4. Toast system: Use existing toast/notification system in app
5. Mobile parity: Ensure feature works identically on mobile
