# Smart RSS Feed Design

**Date**: 2026-02-11
**Status**: Design Approved
**Author**: Design collaboration via brainstorming

## Overview

Create a "smart feed" feature for the RSS reader that prioritizes articles based on relevance, preventing low-volume feeds from being buried while learning from user behavior.

## Problem Statement

Users subscribing to both high-volume news sites and low-volume personal blogs face three issues:

1. **Missing content**: Articles from quiet feeds get buried under frequent publications
2. **Information overload**: Too many articles to reasonably review
3. **Relevance**: Want to see content most likely to be interesting

## Goals

- Prioritize articles from feeds with fewer daily articles
- Learn from user interactions (articles they convert to cards)
- Provide manual override via priority flags
- Keep the algorithm transparent and deterministic

## Scoring Algorithm

Each article receives a **smart score** (higher = more priority). Three components:

### 1. Volume Score (0-100 points)

Inverse of feed's daily article rate, calculated from last 30 days:

```
daily_average = article_count_last_30_days / 30
volume_score = max(0, 100 - (daily_average × 10))
```

| Daily Articles | Volume Score |
|---------------|--------------|
| 0-1           | 90-100       |
| 5             | 50           |
| 10+           | 0            |

### 2. Interaction Bonus (0-50 points)

Based on articles converted to cards in last 90 days:

```
interaction_bonus = min(50, conversion_count × 10)
```

Each conversion adds 10 points, capped at 50.

### 3. Priority Flag (+100 points)

Manual override - `priority` boolean on `rss_feeds` table.

### Final Calculation

```
smart_score = volume_score + interaction_bonus + (100 if priority else 0)
```

**Sort order**: `smart_score DESC, published_at DESC`

## Data Model Changes

### Database Schema

```sql
ALTER TABLE rss_feeds ADD COLUMN priority BOOLEAN DEFAULT FALSE;
```

### Response Model

```go
type SmartFeedScore struct {
    ArticleID        int     `json:"article_id"`
    Score            float64 `json:"score"`
    VolumeScore      float64 `json:"volume_score"`
    InteractionBonus float64 `json:"interaction_bonus"`
    IsPriority       bool    `json:"is_priority"`
    Reason           string  `json:"reason"` // Human-readable explanation
}
```

### API Response

```
GET /api/rss/articles/smart?limit=50&offset=0
```

```json
{
  "articles": [
    {
      "id": 123,
      "title": "Article Title",
      "feed_id": 5,
      "published_at": "2026-02-11T10:00:00Z",
      "read": false,
      "smart_score": {
        "score": 145,
        "volume_score": 90,
        "interaction_bonus": 0,
        "is_priority": false,
        "reason": "Low-volume feed (~1 article/day)"
      }
    }
  ],
  "total": 523
}
```

## Implementation Details

### Backend (go-backend/services/rss.go)

```go
func ListSmartRSSArticles(db models.Database, userID int, filters map[string]interface{}) ([]models.RSSArticleWithScore, int, error) {
    // 1. Calculate volume scores (last 30 days)
    volumeScores, err := calculateFeedVolumeScores(db, userID)

    // 2. Calculate interaction bonuses (last 90 days)
    interactionBonuses, err := calculateInteractionBonuses(db, userID)

    // 3. Get priority feeds
    priorityFeeds, err := getPriorityFeeds(db, userID)

    // 4. Query articles with scoring and sort
    articles, total, err := queryArticlesWithScoring(db, userID, filters, volumeScores, interactionBonuses, priorityFeeds)

    return articles, total, nil
}
```

### Key Database Queries

```sql
-- Volume scores by feed (last 30 days)
SELECT feed_id, COUNT(*) as article_count
FROM rss_articles
WHERE user_id = $1 AND published_at > NOW() - INTERVAL '30 days'
GROUP BY feed_id

-- Interaction bonuses (last 90 days)
SELECT f.id, COUNT(a.card_id) as conversion_count
FROM rss_feeds f
JOIN rss_articles a ON f.id = a.feed_id
WHERE f.user_id = $1 AND a.card_id IS NOT NULL
  AND a.published_at > NOW() - INTERVAL '90 days'
GROUP BY f.id

-- Priority feeds
SELECT id FROM rss_feeds WHERE user_id = $1 AND priority = TRUE
```

### Route Registration

`go-backend/routes/rss.go`:
```go
addProtectedRoute(r, h, "/api/rss/articles/smart", h.ListSmartRSSArticlesRoute, "GET")
```

## Frontend Integration

1. **Navigation**: Add "Smart Feed" option to RSS section
2. **API client**: Add `getSmartRSSArticles()` in `src/api/rss.ts`
3. **Feed management**: Add "Priority" toggle in feed edit dialog
4. **Optional visual indicator**: Small badge showing why article ranked high

## Edge Cases

| Situation | Behavior |
|-----------|----------|
| New feed (no history) | Volume score = 50 (neutral), interaction = 0 |
| No articles in 30 days | Use available data, floor at minimum score |
| All feeds have same score | Fall back to `published_at DESC` |
| Feed with 0 articles ever | Treated as low-volume (score = 100) |
| User has 0 feeds | Return empty array |

## Performance Considerations

- Score calculations cached per request (not persistent cache for simplicity)
- Volume/interaction queries run once per smart feed call
- For large accounts (1000+ articles), consider materialized views later

## Future Enhancements (Out of Scope)

- Folder-level priority (treat whole folder as priority)
- Negative signals (penalty for feeds user frequently skips)
- Time decay (older articles get slight penalty)
- User-configurable score weights

## Testing Strategy

1. Unit tests for score calculation functions
2. Integration tests for API endpoint with various data scenarios
3. Manual testing with real feed data to validate ranking feels intuitive
