# Google Reader API Design

## Overview

Add Google Reader API compatibility to Zettelgarden so that NetNewsWire (and other RSS readers) can sync with the existing RSS feed system. The Google Reader API is an undocumented but reverse-engineered API from ~2005 that many RSS clients still support.

**Goal:** Enable NetNewsWire on iOS to sync subscriptions, read/unread status, and starred items with Zettelgarden's RSS system.

## Background

NetNewsWire supports the "FreshRSS" login option under its Self-Hosted settings. FreshRSS implements the Google Reader API protocol. By implementing this protocol, Zettelgarden can work with NetNewsWire and many other RSS clients including:

- NetNewsWire (iOS, macOS)
- Reeder (iOS, macOS)
- Read You (Android)
- Fluent Reader (Android, iOS)
- NewsFlash (Linux)

## Architecture

### New Package: `go-backend/greader/`

```
go-backend/greader/
├── handler.go      # HTTP handlers for all Google Reader API endpoints
├── auth.go         # Authentication middleware bridging Google Reader tokens to JWT
├── ids.go          # ID conversion between internal IDs and Google Reader formats
└── responses.go    # Response builders for API responses
```

### Routes

New route file: `go-backend/routes/greader.go`

All routes are prefixed with `/api/greader/`.

## Database Schema Changes

### Migration

```sql
-- Add starred column to rss_articles
ALTER TABLE rss_articles ADD COLUMN starred BOOLEAN NOT NULL DEFAULT false;

-- Add index for starred queries
CREATE INDEX idx_rss_articles_starred ON rss_articles(user_id, starred);
```

### Updated Model

```go
type RSSArticle struct {
    ID          int        `json:"id"`
    UserID      int        `json:"user_id"`
    FeedID      int        `json:"feed_id"`
    Title       string     `json:"title"`
    Content     *string    `json:"content,omitempty"`
    Author      *string    `json:"author,omitempty"`
    URL         string     `json:"url"`
    PublishedAt *time.Time `json:"published_at,omitempty"`
    FetchedAt   time.Time  `json:"fetched_at"`
    Read        bool       `json:"read"`
    Starred     bool       `json:"starred"`      // NEW
    CardID      *int       `json:"card_id,omitempty"`
}
```

### New Service Functions

```go
// services/rss.go
func MarkRSSArticleAsStarred(db models.Database, userID, articleID int, starred bool) error
func ListStarredRSSArticles(db models.Database, userID int, filters) ([]models.RSSArticle, error)
```

## ID Conversion

The Google Reader API uses multiple ID formats. We must support all three:

### Item ID Formats

1. **Long-form hex:** `tag:google.com,2005:reader/item/000000000000001F`
2. **Short-form hex:** `000000000000001F` (16 chars, zero-padded)
3. **Decimal:** `31`

### Feed ID Format

- `feed/{id}` where id is a decimal integer

### Conversion Functions

```go
// greader/ids.go

func ItemIDToGoogleReaderFormat(id int) string
func ItemIDToShortHex(id int) string
func ParseItemID(id string) (int, error)
func FeedIDToGoogleReaderFormat(id int) string
func ParseFeedID(id string) (int, error)
```

## Authentication

### Google Reader Token Model

1. **Auth token** (long-lived) - Returned from `/accounts/ClientLogin`
2. **Session token** (short-lived) - Returned from `/reader/api/0/token`

### Implementation Strategy

Since Zettelgarden uses JWT internally:
- Generate a Google Reader-compatible auth token containing the user's JWT
- Accept `Authorization: GoogleLogin auth={token}` header
- Validate embedded JWT and extract user ID via middleware

```go
// greader/auth.go

func GenerateAuthToken(jwtToken string) string
func ParseAuthToken(authToken string) (string, error)
func GReaderAuthMiddleware(next http.Handler) http.Handler
```

## API Endpoints

### Authentication

#### `POST /api/greader/accounts/ClientLogin`

Client login endpoint. Returns auth token.

**Input:**
- `Email` (form or JSON)
- `Passwd` (form or JSON)
- `accountType`, `service`, `client` (optional, ignored)

**Response:** `text/plain`
```
SID={token}
Auth={token}
expires_in=604800
```

#### `GET /api/greader/reader/api/0/token`

Returns session token for write operations.

**Response:** `text/plain` (57 chars)
```
{40-char-hash}{17-z-chars}
```

#### `GET /api/greader/reader/api/0/user-info`

Returns user information.

**Query:** `output=json`

**Response:** `application/json`
```json
{
  "userId": "1",
  "userName": "nick",
  "userProfileId": "1",
  "userEmail": "nick@example.com"
}
```

### Subscriptions

#### `GET /api/greader/reader/api/0/subscription/list`

Returns list of subscribed feeds.

**Query:** `output=json`

**Response:** `application/json`
```json
{
  "subscriptions": [
    {
      "id": "feed/1",
      "title": "Feed Name",
      "categories": [
        {"id": "user/-/label/Tech", "label": "Tech"}
      ],
      "url": "https://feed.url",
      "htmlUrl": "https://site.url",
      "iconUrl": ""
    }
  ]
}
```

#### `GET /api/greader/reader/api/0/tag/list`

Returns available tags/streams.

**Response:** `application/json`
```json
{
  "tags": [
    {"id": "user/-/state/com.google/starred"},
    {"id": "user/-/state/com.google/read"},
    {"id": "user/-/state/com.google/reading-list"},
    {"id": "user/-/state/com.google/kept-unread"}
  ]
}
```

### Content Retrieval

#### `GET /api/greader/reader/api/0/stream/contents/{streamId}`

Returns feed items for a stream.

**Supported stream IDs:**
- `user/-/state/com.google/starred` - Starred articles
- `user/-/state/com.google/read` - All articles (read + unread)
- `user/-/state/com.google/reading-list` - All articles
- `user/-/state/com.google/kept-unread` - All articles (same as reading-list)
- `feed/{id}` - Specific feed articles

**Query/Body:**
- `n` - Number of items to return
- `ot` - Older than (UNIX timestamp)
- `c` - Continuation token (offset)
- `xt` - Exclude target (stream ID to exclude)

**Response:** `application/json`
```json
{
  "id": "user/-/state/com.google/reading-list",
  "updated": 1738896000,
  "continuation": "20",
  "items": [
    {
      "id": "tag:google.com,2005:reader/item/0000000000000001",
      "title": "Article Title",
      "published": 1738896000,
      "crawlTimeMsec": "1738896000000",
      "timestampUsec": "1738896000000000",
      "alternate": [{"href": "https://article.url"}],
      "canonical": [{"href": "https://article.url"}],
      "summary": {"content": "Article content..."},
      "categories": [
        "user/-/state/com.google/reading-list",
        "user/-/state/com.google/read"
      ],
      "origin": {
        "streamId": "feed/1",
        "title": "Feed Name",
        "htmlUrl": "https://feed.url"
      }
    }
  ]
}
```

#### `POST /api/greader/reader/api/0/stream/contents/{streamId}`

Same as GET but accepts POST body parameters (some clients use POST).

#### `GET /api/greader/reader/api/0/stream/items/contents`

Batch load items by ID (used by Reeder).

**Body:** Multiple `i` parameters with item IDs
```
i=0000000000000001&i=0000000000000002&i=0000000000000003
```

**Response:** Same format as `/stream/contents/{streamId}`

#### `GET /api/greader/reader/api/0/stream/items/ids`

Returns list of item IDs for a stream.

**Query:**
- `s` - Stream ID (required)
- `n` - Number of items (up to 10000)
- `ot` - Older than timestamp
- `xt` - Exclude target
- `r` - Reverse sort (`o`)

**Response:** `application/json`
```json
{
  "itemRefs": [
    {"id": "1"},
    {"id": "2"},
    {"id": "3"}
  ]
}
```

#### `GET /api/greader/reader/api/0/unread-count`

Returns unread counts per feed.

**Query:** `all=1`, `output=json`

**Response:** `application/json`
```json
{
  "max": 10,
  "unreadcounts": [
    {
      "id": "feed/1",
      "count": 5,
      "newestItemTimestampUsec": "1738896000000000"
    }
  ]
}
```

### State Manipulation

#### `POST /api/greader/reader/api/0/edit-tag`

Mark items as read/unread or starred/unstarred.

**Body:** `application/x-www-form-urlencoded`
- `i` - Item ID(s), multiple allowed
- `a` - Tag to add (URL-encoded)
- `r` - Tag to remove (URL-encoded)

**Tags:**
- `user/-/state/com.google/read` - Mark as read
- `user/-/state/com.google/starred` - Mark as starred

**Response:** `text/plain` `OK`

#### `POST /api/greader/reader/api/0/mark-all-as-read`

Mark all items in a stream as read.

**Body:**
- `s` - Stream ID
- `ts` - UNIX timestamp to start from

**Response:** `text/plain` `OK`

## Important Implementation Notes

### Response Content-Type is Critical

Some clients (especially Reeder) require specific content-types:
- `/edit-tag` and `/mark-all-as-read` must return `text/plain`
- `/token` must return `text/plain`
- Most other endpoints return `application/json`

### Timestamp Formats

The API is picky about timestamp representations:

| Field | Type | Unit |
|-------|------|------|
| `published` | int | seconds |
| `updated` | int | seconds |
| `crawlTimeMsec` | string | milliseconds |
| `timestampUsec` | string | microseconds |
| `newestItemTimestampUsec` | string | microseconds |

### Continuation Tokens

Simple offset-based pagination:
1. Store offset number in token
2. Client returns token to get next page
3. Token is just the offset as a string

### Client Behavior

**NetNewsWire sync flow:**
1. Fetch subscriptions via `/subscription/list`
2. Fetch unread counts via `/unread-count`
3. Fetch article IDs via `/stream/items/ids`
4. Batch load contents via `/stream/items/contents`
5. Send `/edit-tag` when user marks items read/starred

**Reeder sync flow:**
1. Uses `/stream/contents/{streamId}` directly instead of IDs endpoint
2. Requires `text/plain` responses for edit endpoints
3. May send items in long hex format without prefix

### Categories/Folders

Zettelgarden's existing folder system maps to Google Reader's "categories":

```go
// Map folder to category format
"categories": [{"id": "user/-/label/" + folder, "label": folder}]
```

## Route Registration

```go
// routes/greader.go

func RegisterGReaderRoutes(r *mux.Router, h *handlers.Handler) {
    api := r.PathPrefix("/api/greader").Subrouter()

    // Auth endpoint (no middleware)
    api.HandleFunc("/accounts/ClientLogin", h.GReaderClientLogin).Methods("POST")

    // All other endpoints require auth
    authAPI := api.PathPrefix("/reader/api/0").Subrouter()
    authAPI.Use(handlers.GReaderAuthMiddleware)

    // User info
    authAPI.HandleFunc("/user-info", h.GReaderUserInfo).Methods("GET")
    authAPI.HandleFunc("/token", h.GReaderToken).Methods("GET")

    // Subscriptions
    authAPI.HandleFunc("/subscription/list", h.GReaderSubscriptionList).Methods("GET")
    authAPI.HandleFunc("/tag/list", h.GReaderTagList).Methods("GET")

    // Content
    authAPI.HandleFunc("/stream/contents/{streamId}", h.GReaderStreamContents).Methods("GET", "POST")
    authAPI.HandleFunc("/stream/items/contents", h.GReaderStreamItemsContents).Methods("GET", "POST")
    authAPI.HandleFunc("/stream/items/ids", h.GReaderStreamItemsIds).Methods("GET")
    authAPI.HandleFunc("/unread-count", h.GReaderUnreadCount).Methods("GET")

    // State manipulation
    authAPI.HandleFunc("/edit-tag", h.GReaderEditTag).Methods("POST")
    authAPI.HandleFunc("/mark-all-as-read", h.GReaderMarkAllAsRead).Methods("POST")
}
```

## References

- [FreshRSS Google Reader API Documentation](https://freshrss.github.io/FreshRSS/en/developers/06_GoogleReader_API.html)
- [Re-Implementing the Google Reader API in 2025](https://www.davd.io/posts/2025-02-05-reimplementing-google-reader-api-in-2025/)
- [Google Reader API Documentation (archived)](https://inessential.com/2013/03/14/google_reader_api_documentation.html)
