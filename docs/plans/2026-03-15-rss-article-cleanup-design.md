# RSS Article Cleanup Job Design

**Date**: 2026-03-15
**Status**: Approved

## Overview

Add a scheduled job to clean up RSS articles older than a configurable retention period, protecting starred articles and articles converted to cards.

## Requirements

1. Delete RSS articles older than the retention period (default 30 days)
2. Protect starred articles (`is_starred = true`)
3. Protect articles converted to cards (`card_id IS NOT NULL`)
4. Run daily at 3 AM
5. Configurable retention period via environment variable

## Design

### New Scheduled Job

Create `RSSArticleCleanupJob` in `services/jobs/rss_article_cleanup_job.go`:

| Property | Value |
|----------|-------|
| **Name** | `rss-article-cleanup` |
| **Schedule** | `0 0 3 * * *` (daily at 3 AM) |
| **Max Retries** | 3 |

### Configuration

- Environment variable: `RSS_ARTICLE_RETENTION_DAYS`
- Default: `30` days

### Cleanup Query

```sql
DELETE FROM rss_articles
WHERE fetched_at < NOW() - INTERVAL '1 day' * $1
  AND is_starred = false
  AND card_id IS NULL
```

### Side Effects

- Related notifications are automatically deleted via the existing `delete_notification()` trigger on `rss_articles`

## Files Changed

1. **New**: `go-backend/services/jobs/rss_article_cleanup_job.go`
2. **Modified**: `go-backend/main.go` - register the new job with scheduler

## Implementation Notes

- Follow existing patterns from `cleanup_job.go` and `rss_fetch_job.go`
- Use `fetched_at` (not `published_at`) for age calculation since it's always set
- Log number of articles deleted for monitoring
