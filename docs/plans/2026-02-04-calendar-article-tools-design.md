# Calendar and Article Tools for Go Backend

**Date:** 2026-02-04
**Status:** Implemented

## Overview

Add calendar and article parsing tools to the Go backend's LLM agent toolkit to achieve parity with the Python MCP server. The underlying services and HTTP handlers already exist; we only need to add tool definitions that call the service layer directly.

## Tools to Add

### Calendar Tools (3)

| Tool | Description | Service Function |
|------|-------------|------------------|
| `list_external_calendars` | List user's external calendar subscriptions | `GetCalendars(db, userID)` |
| `list_external_events` | Get events within a date range | `GetEventsInRange(db, userID, start, end, limit, 0)` |
| `link_event_to_card` | Link an event to a card | `LinkEventToCard(db, userID, eventID, cardPK)` |

### Article Tools (2)

| Tool | Description | Implementation |
|------|-------------|----------------|
| `parse_url` | Extract article content from URL for preview | Extract logic from `ParseURLRoute` handler |
| `create_article` | Create a card directly from a URL | Extract logic from `CreateArticleRoute` handler |

## Architecture

```
LLM Tool Call → ToolHandler → Service Layer → Database
                              ↓
                         StructToMap (response)
```

Tools call the existing service layer directly, avoiding HTTP overhead. This follows the user's preference for service-layer calls over HTTP handler invocation.

## Implementation Details

### Calendar Tool Handlers

All three calendar tools have existing service implementations in `services/external_events.go`:

- `GetCalendars` at line 472
- `GetEventsInRange` at line 352
- `LinkEventToCard` at line 773

Tool handlers will:
1. Extract parameters using existing helpers (`getStringParam`, `getIntParam`, etc.)
2. Call the appropriate service function
3. Format output to match Python MCP style

### Article Tool Handlers

Create new file `services/articles.go` with:

```go
func ParseURL(url string) (ParseResult, error)
func CreateArticle(db *sql.DB, userID int, url, cardID, tags string) (models.Card, error)
```

Extract the readability and card creation logic from:
- `handlers/cards.go:1112` (ParseURLRoute)
- `handlers/cards.go:1194` (CreateArticleRoute)

### Error Handling

Follow the existing pattern in `tools.go`:

```go
func handleXxx(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
    param, err := getStringParam(args, "param")
    if err != nil {
        return nil, err
    }

    result, err := ServiceXxx(ctx.DB, ctx.UserID, param)
    if err != nil {
        return nil, fmt.Errorf("failed to xxx: %v", err)
    }

    return StructToMap(result), nil
}
```

### Date Parsing

- Accept ISO 8601 format (e.g., `2026-01-01T00:00:00Z`)
- Use `time.Parse(time.RFC3339, dateStr)`
- Validate start < end before querying

### URL Validation

- Basic format validation
- 30-second timeout for fetching
- Graceful error handling for unreachable URLs

## Testing Strategy

### Unit Tests
- Create `services/calendar_tools_test.go`
- Create `services/article_tools_test.go`
- Mock database responses
- Test error paths

### Integration Tests
- Use existing test infrastructure in `tests/conftest.go`
- Test full tool call flow
- Verify output format

### Test Cases

**Calendar:**
- Empty calendar list
- Multiple calendars with sync errors
- Date range validation
- All-day events with linked cards
- Link/unlink event-card relationships

**Articles:**
- Valid article parsing
- Unreachable URL handling
- Timeout handling
- Tag customization
- Duplicate detection

## Files to Modify

1. `go-backend/services/tools.go` - Add 5 new tool registrations and handlers
2. `go-backend/services/articles.go` - New file for article services
3. `go-backend/services/calendar_tools_test.go` - New test file
4. `go-backend/services/article_tools_test.go` - New test file
