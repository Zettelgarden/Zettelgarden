# RSS Feed Discovery Feature Design

**Date:** 2026-02-09
**Status:** Design Approved

## Overview

Add a feed discovery feature to the RSS page that allows users to enter a website URL (e.g., `https://example.com`) and auto-discover the RSS/Atom feed from the HTML headers, rather than requiring the direct feed URL.

## Requirements

### Functional Requirements
1. User enters a website URL in the Add Feed dialog
2. Click "Discover Feed" button to fetch and parse the webpage
3. Auto-populate the feed URL and name fields based on discovery
4. Handle errors gracefully with helpful messages
5. Prefer RSS feeds over Atom, use first found feed

### Non-Functional Requirements
- Server-side discovery to avoid CORS issues
- 10-second timeout for network requests
- Graceful handling of malformed HTML
- Support for common feed path fallbacks

## Architecture

### Backend (Go)

**New API Endpoint:**
```
POST /api/rss/discover
Request: { "url": "https://example.com" }
Response: {
  "feed_url": "https://example.com/feed.xml",
  "title": "Example Blog"
}
```

**Handler Function:** `DiscoverFeedRoute` in `handlers/rss.go`

**Request/Response Types:**
```go
type DiscoverFeedRequest struct {
    URL string `json:"url" validate:"required,url"`
}

type DiscoverFeedResponse struct {
    FeedURL string `json:"feed_url"`
    Title   string `json:"title"`
}
```

**Discovery Logic:**
1. Validate URL format
2. Fetch HTML with 10-second timeout
3. Parse HTML for `<link rel="alternate" type="application/rss+xml|atom+xml">`
4. Prefer RSS, fallback to Atom
5. If no feed in headers, try `/feed`, `/rss`, `/atom` paths
6. Extract `<title>` for feed name
7. Return 404 with message if no feed found

### Frontend (React/TypeScript)

**New API Function:** `src/api/rss.ts`
```typescript
export interface DiscoverFeedRequest {
  url: string;
}

export interface DiscoverFeedResponse {
  feed_url: string;
  title: string;
}

export function discoverFeed(url: string): Promise<DiscoverFeedResponse> {
  return getData(apiClient.post<DiscoverFeedResponse>("/rss/discover", { url }));
}
```

**Modified Component:** `src/components/rss/RssAddFeedDialog.tsx`

Add:
- `discovering` state for loading indicator
- "Discover Feed" button next to URL input
- `handleDiscover` function to call API and populate fields

## UI Design

```
┌─────────────────────────────────────────────────────────┐
│ Add RSS Feed                                           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ Feed URL *              [https://example.com] [Discover]│
│                                                         │
│ Name (optional)        [Example Blog]                   │
│                                                         │
│ Folder (optional)      [No folder ▼]                   │
│                                                         │
│ Auto Tags (optional)   [tech, ai, machine-learning]    │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ No RSS/Atom feed found. Try checking /feed, /rss,   │ │
│ │ or /atom                                           │ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│                    [Cancel] [Add Feed]                  │
└─────────────────────────────────────────────────────────┘
```

**Button States:**
- Idle: "Discover Feed"
- Loading: Spinner + "Discovering..."
- Disabled: When URL is empty

## Error Handling

| Error Case | Status Code | Message |
|------------|-------------|---------|
| Invalid URL format | 400 | "Invalid URL format" |
| Network timeout | 504 | "Request timed out. Please try again." |
| No feed found | 404 | "No RSS/Atom feed found. Try checking /feed, /rss, or /atom" |
| Non-HTML response | 404 | "No RSS/Atom feed found. The URL may not be a webpage." |

## Edge Cases

- **Relative URLs in `<link>` tags:** Resolve to absolute URLs
- **HTML entities in title:** Decode (`&amp;` → `&`)
- **Redirects:** Follow up to 3 redirects
- **Malformed HTML:** Best-effort parsing
- **Internationalized domain names:** Properly encode

## Testing

### Backend Tests (`handlers/rss_test.go`)
- Valid URL with RSS feed → returns feed URL
- Valid URL with Atom feed only → returns Atom feed URL
- Multiple feeds → returns first RSS feed
- No feed in headers, `/feed` works → discovers fallback
- No feeds at all → 404 with error
- Invalid URL → 400
- Network timeout → 504

### Frontend Tests (`RssAddFeedDialog.test.tsx`)
- Empty URL → button disabled
- Valid URL → shows loading state
- Success → URL and name populated
- Failure → error message shown
- Fields editable after discovery

## Implementation Notes

1. Use standard library `net/http` and `golang.org/x/net/html` for parsing (no new dependencies)
2. Add route to `handlers/handlers.go` after other RSS routes
3. Reuse existing auth middleware from RSS routes
4. Follow existing error response patterns
