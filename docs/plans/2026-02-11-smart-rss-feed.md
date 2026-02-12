# Smart RSS Feed Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a "smart feed" feature to the RSS reader that prioritizes articles based on feed volume, user interactions, and manual priority flags.

**Architecture:** A new API endpoint `/api/rss/articles/smart` that calculates scores for articles based on three factors: volume score (inverse of daily article rate), interaction bonus (conversions to cards), and priority flag (manual override). The scoring is done in the service layer with SQL queries for data aggregation.

**Tech Stack:** Go (backend), PostgreSQL, React/TypeScript (frontend), existing RSS infrastructure

---

## Task 1: Add Priority Column to RSS Feeds Table

**Files:**
- Create: `go-backend/schema/0114-add-rss-priority.sql`
- Modify: `go-backend/models/rss.go:6-19`
- Modify: `go-backend/models/rss.go:54-61`

**Step 1: Create the database migration**

```sql
--- Migration: Add priority column to rss_feeds
--- Description: Add boolean flag for manually prioritized feeds
--- Created: 2026-02-11

ALTER TABLE rss_feeds ADD COLUMN priority BOOLEAN DEFAULT FALSE;

-- Add comment for documentation
COMMENT ON COLUMN rss_feeds.priority IS 'Manual priority flag for smart feed - feeds marked as priority always rank higher';
```

**Step 2: Add Priority field to RSSFeed model**

Find the `RSSFeed` struct in `go-backend/models/rss.go` (around line 6-19) and add the `Priority` field after `Enabled`:

```go
type RSSFeed struct {
    ID             int        `json:"id"`
    UserID         int        `json:"user_id"`
    URL            string     `json:"url"`
    Name           string     `json:"name"`
    Folder         *string    `json:"folder,omitempty"`
    AutoTags       string     `json:"auto_tags"`
    FetchInterval  int        `json:"fetch_interval"`
    LastFetchedAt  *time.Time `json:"last_fetched_at,omitempty"`
    LastError      *string    `json:"last_error,omitempty"`
    Enabled        bool       `json:"enabled"`
    Priority       bool       `json:"priority"`  // ADD THIS LINE
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}
```

**Step 3: Add Priority to update params**

Find the `UpdateRSSFeedParams` struct in `go-backend/models/rss.go` (around line 54-61) and add `Priority`:

```go
// UpdateRSSFeedParams represents parameters for updating an RSS feed
type UpdateRSSFeedParams struct {
    Name          *string `json:"name,omitempty"`
    Folder        *string `json:"folder,omitempty"`
    AutoTags      *string `json:"auto_tags,omitempty"`
    FetchInterval *int    `json:"fetch_interval,omitempty"`
    Enabled       *bool   `json:"enabled,omitempty"`
    Priority      *bool   `json:"priority,omitempty"`  // ADD THIS LINE
}
```

**Step 4: Update service to handle priority in scans**

Open `go-backend/services/rss.go` and find the `GetRSSFeedByID` function (around line 75). Add the priority column to the SELECT query and Scan:

```go
// In GetRSSFeedByID, update the query to include priority:
err := db.QueryRow(`
        SELECT id, user_id, url, name, folder, auto_tags, fetch_interval,
               last_fetched_at, last_error, enabled, priority, created_at, updated_at  -- ADD priority HERE
        FROM rss_feeds
        WHERE id = $1 AND user_id = $2
`, feedID, userID).Scan(
    &feed.ID, &feed.UserID, &feed.URL, &feed.Name, &folder, &feed.AutoTags,
    &feed.FetchInterval, &lastFetched, &lastError, &feed.Enabled, &feed.Priority,  -- ADD &feed.Priority HERE
    &feed.CreatedAt, &feed.UpdatedAt,
)
```

**Step 5: Update ListRSSFeeds to include priority**

In `go-backend/services/rss.go` find the `ListRSSFeeds` function (around line 110). Update the query and scan:

```go
// Update the query:
rows, err := db.Query(`
        SELECT id, user_id, url, name, folder, auto_tags, fetch_interval,
               last_fetched_at, last_error, enabled, priority, created_at, updated_at  -- ADD priority
        FROM rss_feeds
        WHERE user_id = $1
        ORDER BY name ASC
`, userID)

// Update the scan:
err := rows.Scan(
    &feed.ID, &feed.UserID, &feed.URL, &feed.Name, &folder, &feed.AutoTags,
    &feed.FetchInterval, &lastFetched, &lastError, &feed.Enabled, &feed.Priority,  -- ADD &feed.Priority
    &feed.CreatedAt, &feed.UpdatedAt,
)
```

**Step 6: Update UpdateRSSFeed to handle priority parameter**

In `go-backend/services/rss.go` find the `UpdateRSSFeed` function (around line 159). Add the priority update logic:

```go
// In the updates loop, add after the Enabled check:
if params.Priority != nil {
    updates = append(updates, fmt.Sprintf("priority = $%d", argPos))
    args = append(args, *params.Priority)
    argPos++
}
```

**Step 7: Update FetchRSSFeedArticles to include priority**

In `go-backend/services/rss.go` find the `FetchRSSFeedArticles` function (around line 520). Update the query:

```go
err := db.QueryRow(`
        SELECT id, user_id, url, name, folder, auto_tags, fetch_interval,
               last_fetched_at, last_error, enabled, priority  -- ADD priority
        FROM rss_feeds
        WHERE id = $1
`, feedID).Scan(
    &feed.ID, &feed.UserID, &feed.URL, &feed.Name, &folder, &feed.AutoTags,
    &feed.FetchInterval, &lastFetched, &lastError, &feed.Enabled, &feed.Priority,  -- ADD &feed.Priority
)
```

**Step 8: Run tests**

```bash
cd go-backend
go test ./services -run TestRSS -v
```

Expected: Tests should still pass (backward compatible since we added a DEFAULT FALSE).

**Step 9: Commit**

```bash
git add go-backend/schema/0114-add-rss-priority.sql go-backend/models/rss.go go-backend/services/rss.go
git commit -m "feat(rss): add priority flag to rss_feeds table and model

Adds priority boolean column to rss_feeds for manual feed prioritization
in the smart feed feature. Default is FALSE for existing feeds.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Add Smart Score Model Types

**Files:**
- Create: `go-backend/models/smart_feed.go`
- Modify: `go-backend/services/rss.go` (import)

**Step 1: Create the smart feed model file**

```go
package models

// SmartFeedScore represents the scoring breakdown for an article in the smart feed
type SmartFeedScore struct {
    ArticleID        int     `json:"article_id"`
    Score            float64 `json:"score"`
    VolumeScore      float64 `json:"volume_score"`
    InteractionBonus float64 `json:"interaction_bonus"`
    IsPriority       bool    `json:"is_priority"`
    Reason           string  `json:"reason"`
}

// RSSArticleWithScore extends RSSArticle with smart scoring
type RSSArticleWithScore struct {
    RSSArticle
    SmartScore *SmartFeedScore `json:"smart_score,omitempty"`
}
```

**Step 2: Run tests**

```bash
cd go-backend
go test ./models -v
```

Expected: PASS (no tests to fail, just type definitions).

**Step 3: Commit**

```bash
git add go-backend/models/smart_feed.go
git commit -m "feat(rss): add smart feed scoring models

Adds SmartFeedScore and RSSArticleWithScore types for the
smart feed feature response.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Implement Volume Score Calculation

**Files:**
- Create: `go-backend/services/smart_feed.go`
- Test: `go-backend/services/smart_feed_test.go`

**Step 1: Write the failing test**

Create `go-backend/services/smart_feed_test.go`:

```go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    _ "github.com/lib/pq"
)

func TestCalculateFeedVolumeScores(t *testing.T) {
    // This requires a test database connection
    // For now, test the scoring logic directly
    t.Run("zero articles gets max score", func(t *testing.T) {
        score := calculateVolumeScore(0)
        assert.Equal(t, 100.0, score)
    })

    t.Run("1 article per day gets high score", func(t *testing.T) {
        score := calculateVolumeScore(30) // 30 articles in 30 days = 1/day
        assert.Equal(t, 90.0, score)
    })

    t.Run("5 articles per day gets medium score", func(t *testing.T) {
        score := calculateVolumeScore(150) // 150 articles in 30 days = 5/day
        assert.Equal(t, 50.0, score)
    })

    t.Run("10+ articles per day gets zero score", func(t *testing.T) {
        score := calculateVolumeScore(300) // 300 articles in 30 days = 10/day
        assert.Equal(t, 0.0, score)
    })

    t.Run("score floors at zero", func(t *testing.T) {
        score := calculateVolumeScore(600) // 20 articles per day
        assert.Equal(t, 0.0, score)
    })
}
```

**Step 2: Run test to verify it fails**

```bash
cd go-backend
go test ./services -run TestCalculateFeedVolumeScores -v
```

Expected: FAIL with "undefined: calculateVolumeScore"

**Step 3: Write minimal implementation**

Create `go-backend/services/smart_feed.go`:

```go
package services

import (
    "database/sql"
    "fmt"
)

// calculateVolumeScore converts article count to volume score (0-100)
// score = max(0, 100 - (daily_avg × 10))
func calculateVolumeScore(articleCount int) float64 {
    if articleCount <= 0 {
        return 100.0
    }
    dailyAvg := float64(articleCount) / 30.0
    score := 100.0 - (dailyAvg * 10.0)
    if score < 0 {
        return 0.0
    }
    return score
}

// calculateFeedVolumeScores gets article counts for each feed in the last 30 days
func calculateFeedVolumeScores(db Database, userID int) (map[int]float64, error) {
    query := `
        SELECT feed_id, COUNT(*) as article_count
        FROM rss_articles
        WHERE user_id = $1 AND published_at > NOW() - INTERVAL '30 days'
        GROUP BY feed_id
    `
    rows, err := db.Query(query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get feed volume scores: %w", err)
    }
    defer rows.Close()

    scores := make(map[int]float64)
    for rows.Next() {
        var feedID, count int
        if err := rows.Scan(&feedID, &count); err != nil {
            return nil, fmt.Errorf("failed to scan volume score: %w", err)
        }
        scores[feedID] = calculateVolumeScore(count)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating volume scores: %w", err)
    }

    return scores, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd go-backend
go test ./services -run TestCalculateFeedVolumeScores -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/smart_feed.go go-backend/services/smart_feed_test.go
git commit -m "feat(rss): add volume score calculation for smart feed

Implements calculateVolumeScore and calculateFeedVolumeScores
to compute feed volume scores based on article count over 30 days.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Implement Interaction Bonus Calculation

**Files:**
- Modify: `go-backend/services/smart_feed.go`
- Modify: `go-backend/services/smart_feed_test.go`

**Step 1: Write the failing test**

Add to `go-backend/services/smart_feed_test.go`:

```go
func TestCalculateInteractionBonus(t *testing.T) {
    t.Run("zero conversions gets zero bonus", func(t *testing.T) {
        bonus := calculateInteractionBonus(0)
        assert.Equal(t, 0.0, bonus)
    })

    t.Run("1 conversion gets 10 points", func(t *testing.T) {
        bonus := calculateInteractionBonus(1)
        assert.Equal(t, 10.0, bonus)
    })

    t.Run("5 conversions gets 50 points (max)", func(t *testing.T) {
        bonus := calculateInteractionBonus(5)
        assert.Equal(t, 50.0, bonus)
    })

    t.Run("10+ conversions caps at 50 points", func(t *testing.T) {
        bonus := calculateInteractionBonus(10)
        assert.Equal(t, 50.0, bonus)
    })
}
```

**Step 2: Run test to verify it fails**

```bash
cd go-backend
go test ./services -run TestCalculateInteractionBonus -v
```

Expected: FAIL with "undefined: calculateInteractionBonus"

**Step 3: Write minimal implementation**

Add to `go-backend/services/smart_feed.go`:

```go
// calculateInteractionBonus converts conversion count to bonus (0-50)
// bonus = min(50, conversion_count × 10)
func calculateInteractionBonus(conversionCount int) float64 {
    bonus := float64(conversionCount) * 10.0
    if bonus > 50.0 {
        return 50.0
    }
    return bonus
}

// calculateInteractionBonuses gets conversion counts for each feed in last 90 days
func calculateInteractionBonuses(db Database, userID int) (map[int]float64, error) {
    query := `
        SELECT f.id, COUNT(a.card_id) as conversion_count
        FROM rss_feeds f
        JOIN rss_articles a ON f.id = a.feed_id
        WHERE f.user_id = $1 AND a.card_id IS NOT NULL
          AND a.published_at > NOW() - INTERVAL '90 days'
        GROUP BY f.id
    `
    rows, err := db.Query(query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get interaction bonuses: %w", err)
    }
    defer rows.Close()

    bonuses := make(map[int]float64)
    for rows.Next() {
        var feedID, count int
        if err := rows.Scan(&feedID, &count); err != nil {
            return nil, fmt.Errorf("failed to scan interaction bonus: %w", err)
        }
        bonuses[feedID] = calculateInteractionBonus(count)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating interaction bonuses: %w", err)
    }

    return bonuses, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd go-backend
go test ./services -run TestCalculateInteractionBonus -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/smart_feed.go go-backend/services/smart_feed_test.go
git commit -m "feat(rss): add interaction bonus calculation for smart feed

Implements calculateInteractionBonus and calculateInteractionBonuses
to compute feed interaction bonuses based on articles converted to cards.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Implement Priority Feed Lookup

**Files:**
- Modify: `go-backend/services/smart_feed.go`
- Modify: `go-backend/services/smart_feed_test.go`

**Step 1: Write the failing test**

Add to `go-backend/services/smart_feed_test.go`:

```go
func TestGetPriorityFeeds(t *testing.T) {
    // This would require DB setup, so we'll test the function structure
    // For now, just ensure it compiles
    t.Run("function exists", func(t *testing.T) {
        // This is a compile-time check
        var _ func(Database, int) (map[int]bool, error) = getPriorityFeeds
    })
}
```

**Step 2: Run test to verify it fails**

```bash
cd go-backend
go test ./services -run TestGetPriorityFeeds -v
```

Expected: FAIL with "undefined: getPriorityFeeds"

**Step 3: Write minimal implementation**

Add to `go-backend/services/smart_feed.go`:

```go
// getPriorityFeeds returns a map of feed IDs that have priority=true
func getPriorityFeeds(db Database, userID int) (map[int]bool, error) {
    query := `SELECT id FROM rss_feeds WHERE user_id = $1 AND priority = TRUE`
    rows, err := db.Query(query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get priority feeds: %w", err)
    }
    defer rows.Close()

    priorityFeeds := make(map[int]bool)
    for rows.Next() {
        var feedID int
        if err := rows.Scan(&feedID); err != nil {
            return nil, fmt.Errorf("failed to scan priority feed: %w", err)
        }
        priorityFeeds[feedID] = true
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating priority feeds: %w", err)
    }

    return priorityFeeds, nil
}
```

**Step 4: Run test to verify it passes**

```bash
cd go-backend
go test ./services -run TestGetPriorityFeeds -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/smart_feed.go go-backend/services/smart_feed_test.go
git commit -m "feat(rss): add priority feed lookup for smart feed

Implements getPriorityFeeds to retrieve feeds marked with priority=true.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Implement Smart Score Reasoning

**Files:**
- Modify: `go-backend/services/smart_feed.go`

**Step 1: Add reason generation function**

Add to `go-backend/services/smart_feed.go`:

```go
// generateScoreReason creates a human-readable explanation for the score
func generateScoreReason(volumeScore, interactionBonus float64, isPriority bool, dailyAvg float64) string {
    reasons := []string{}

    if isPriority {
        reasons = append(reasons, "Priority feed")
    }

    if volumeScore >= 80 {
        reasons = append(reasons, fmt.Sprintf("Low-volume feed (~%.1f article/day)", dailyAvg))
    } else if volumeScore >= 50 {
        reasons = append(reasons, fmt.Sprintf("Medium-volume feed (~%.1f articles/day)", dailyAvg))
    } else if volumeScore > 0 {
        reasons = append(reasons, fmt.Sprintf("High-volume feed (~%.1f articles/day)", dailyAvg))
    }

    if interactionBonus > 0 {
        reasons = append(reasons, fmt.Sprintf("You convert %.0f%% of articles", interactionBonus/5))
    }

    if len(reasons) == 0 {
        return "New feed"
    }

    return reasons[0]
}
```

**Step 2: Run tests**

```bash
cd go-backend
go test ./services -v
```

Expected: PASS (existing tests still pass)

**Step 3: Commit**

```bash
git add go-backend/services/smart_feed.go
git commit -m "feat(rss): add score reason generation for smart feed

Implements generateScoreReason to create human-readable explanations
for why articles are ranked in the smart feed.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Implement ListSmartRSSArticles Service Function

**Files:**
- Modify: `go-backend/services/rss.go`
- Modify: `go-backend/services/smart_feed.go`

**Step 1: Write the main smart feed function**

Add to `go-backend/services/smart_feed.go`:

```go
import (
    "database/sql"
    "fmt"
    "go-backend/models"
    "time"
)

// ... existing functions ...

// ListSmartRSSArticles returns articles ranked by smart scoring
func ListSmartRSSArticles(db Database, userID int, filters map[string]interface{}) ([]models.RSSArticleWithScore, int, error) {
    // 1. Calculate volume scores (last 30 days)
    volumeScores, err := calculateFeedVolumeScores(db, userID)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to calculate volume scores: %w", err)
    }

    // 2. Calculate interaction bonuses (last 90 days)
    interactionBonuses, err := calculateInteractionBonuses(db, userID)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to calculate interaction bonuses: %w", err)
    }

    // 3. Get priority feeds
    priorityFeeds, err := getPriorityFeeds(db, userID)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to get priority feeds: %w", err)
    }

    // 4. Query articles with scoring and sort
    articles, total, err := queryArticlesWithScoring(db, userID, filters, volumeScores, interactionBonuses, priorityFeeds)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to query articles: %w", err)
    }

    return articles, total, nil
}

// queryArticlesWithScoring fetches articles and calculates scores
func queryArticlesWithScoring(db Database, userID int, filters map[string]interface{}, volumeScores map[int]float64, interactionBonuses map[int]float64, priorityFeeds map[int]bool) ([]models.RSSArticleWithScore, int, error) {
    // Build base query
    query := `
        SELECT id, user_id, feed_id, title, content, author, url,
               published_at, fetched_at, read, card_id
        FROM rss_articles
        WHERE user_id = $1
    `
    args := []interface{}{userID}
    argPos := 2

    // Apply filters (same as ListRSSArticles)
    if folder, ok := filters["folder"].(string); ok && folder != "" {
        query += fmt.Sprintf(" AND feed_id IN (SELECT id FROM rss_feeds WHERE user_id = $1 AND folder = $%d)", argPos)
        args = append(args, folder)
        argPos++
    }

    if unreadOnly, ok := filters["unread"].(bool); ok && unreadOnly {
        query += " AND read = false"
    }

    // Get count first
    countQuery := "SELECT COUNT(*) FROM rss_articles WHERE " + query[38:] // Strip "SELECT ... FROM "
    var total int
    countArgs := make([]interface{}, len(args))
    copy(countArgs, args)
    err := db.QueryRow(countQuery, countArgs...).Scan(&total)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to count articles: %w", err)
    }

    // Get all articles (we'll sort in memory)
    query += " ORDER BY published_at DESC NULLS LAST, fetched_at DESC"

    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to query articles: %w", err)
    }
    defer rows.Close()

    // Collect articles with scores
    var scoredArticles []models.RSSArticleWithScore
    for rows.Next() {
        var article models.RSSArticleWithScore
        var content, author sql.NullString
        var publishedAt sql.NullTime
        var cardID sql.NullInt64

        err := rows.Scan(
            &article.ID, &article.UserID, &article.FeedID, &article.Title,
            &content, &author, &article.URL, &publishedAt,
            &article.FetchedAt, &article.Read, &cardID,
        )
        if err != nil {
            return nil, 0, fmt.Errorf("failed to scan article: %w", err)
        }

        if content.Valid {
            article.Content = &content.String
        }
        if author.Valid {
            article.Author = &author.String
        }
        if publishedAt.Valid {
            article.PublishedAt = &publishedAt.Time
        }
        if cardID.Valid {
            cardIDInt := int(cardID.Int64)
            article.CardID = &cardIDInt
        }

        // Calculate scores
        volumeScore := volumeScores[article.FeedID]
        interactionBonus := interactionBonuses[article.FeedID]
        isPriority := priorityFeeds[article.FeedID]

        priorityBonus := 0.0
        if isPriority {
            priorityBonus = 100.0
        }

        totalScore := volumeScore + interactionBonus + priorityBonus

        // Calculate daily average for reason
        dailyAvg := 0.0
        if volumeScore < 100 {
            // Reverse engineer from score
            dailyAvg = (100.0 - volumeScore) / 10.0
        }

        article.SmartScore = &models.SmartFeedScore{
            ArticleID:        article.ID,
            Score:            totalScore,
            VolumeScore:      volumeScore,
            InteractionBonus: interactionBonus,
            IsPriority:       isPriority,
            Reason:           generateScoreReason(volumeScore, interactionBonus, isPriority, dailyAvg),
        }

        scoredArticles = append(scoredArticles, article)
    }

    if err = rows.Err(); err != nil {
        return nil, 0, fmt.Errorf("error iterating articles: %w", err)
    }

    // Sort by score DESC, then published_at DESC
    sort.Slice(scoredArticles, func(i, j int) bool {
        si := scoredArticles[i].SmartScore
        sj := scoredArticles[j].SmartScore
        if si.Score != sj.Score {
            return si.Score > sj.Score
        }
        // Tie-break by published date
        pi, pj := scoredArticles[i].PublishedAt, scoredArticles[j].PublishedAt
        if pi == nil {
            return false
        }
        if pj == nil {
            return true
        }
        return pi.After(*pj)
    })

    // Apply limit/offset after sorting
    limit := 100
    if limitParam, ok := filters["limit"].(int); ok && limitParam > 0 {
        limit = limitParam
    }
    offset := 0
    if offsetParam, ok := filters["offset"].(int); ok && offsetParam > 0 {
        offset = offsetParam
    }

    if offset >= len(scoredArticles) {
        return []models.RSSArticleWithScore{}, total, nil
    }

    end := offset + limit
    if end > len(scoredArticles) {
        end = len(scoredArticles)
    }

    return scoredArticles[offset:end], total, nil
}
```

**Step 2: Add sort import**

Add to imports in `go-backend/services/smart_feed.go`:

```go
import (
    "database/sql"
    "fmt"
    "sort"
    "go-backend/models"
    "time"
)
```

**Step 3: Run tests**

```bash
cd go-backend
go test ./services -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add go-backend/services/smart_feed.go
git commit -m "feat(rss): implement ListSmartRSSArticles service function

Implements the main smart feed service function that:
1. Calculates volume scores from last 30 days
2. Calculates interaction bonuses from last 90 days
3. Gets priority feeds
4. Queries articles and sorts by score

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: Add Smart Feed Route Handler

**Files:**
- Modify: `go-backend/handlers/handlers.go`
- Modify: `go-backend/routes/rss.go`

**Step 1: Add the handler function**

Add to `go-backend/handlers/handlers.go` (after `ListRSSArticlesRoute` around line 344):

```go
// ListSmartRSSArticlesRoute handles GET /api/rss/articles/smart
func (h *Handler) ListSmartRSSArticlesRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    // Parse query parameters (same as ListRSSArticlesRoute)
    filters := make(map[string]interface{})
    if folder := r.URL.Query().Get("folder"); folder != "" {
        filters["folder"] = folder
    }
    if unread := r.URL.Query().Get("unread"); unread == "true" {
        filters["unread"] = true
    }
    if limit := r.URL.Query().Get("limit"); limit != "" {
        if l, err := strconv.Atoi(limit); err == nil {
            filters["limit"] = l
        }
    }
    if offset := r.URL.Query().Get("offset"); offset != "" {
        if o, err := strconv.Atoi(offset); err == nil {
            filters["offset"] = o
        }
    }

    articles, total, err := services.ListSmartRSSArticles(h.GetDB(), userID, filters)
    if err != nil {
        log.Printf("Failed to list smart RSS articles: %v", err)
        http.Error(w, "Failed to list smart feed articles", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Total-Count", strconv.Itoa(total))

    result := map[string]interface{}{
        "articles": articles,
        "total":    total,
    }

    json.NewEncoder(w).Encode(result)
}
```

**Step 2: Register the route**

Add to `go-backend/routes/rss.go` (after the `/api/rss/articles` routes around line 23):

```go
addProtectedRoute(r, h, "/api/rss/articles/smart", h.ListSmartRSSArticlesRoute, "GET")
```

**Step 3: Run tests**

```bash
cd go-backend
go test ./handlers -run TestRSS -v 2>&1 | grep -v "dial tcp"
```

Expected: PASS (may have DB connection warnings but code compiles)

**Step 4: Commit**

```bash
git add go-backend/handlers/handlers.go go-backend/routes/rss.go
git commit -m "feat(rss): add smart feed API route and handler

Adds GET /api/rss/articles/smart endpoint that returns articles
ranked by the smart scoring algorithm (volume + interaction + priority).

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 9: Frontend API Client

**Files:**
- Modify: `zettelkasten-front/src/api/rss.ts`

**Step 1: Add the API function**

Find `src/api/rss.ts` and add the function after the existing `getRSSArticles`:

```typescript
export interface SmartFeedScore {
  article_id: number;
  score: number;
  volume_score: number;
  interaction_bonus: number;
  is_priority: boolean;
  reason: string;
}

export interface RSSArticleWithScore extends RSSArticle {
  smart_score?: SmartFeedScore;
}

export async function getSmartRSSArticles(params: {
  folder?: string;
  unread?: boolean;
  limit?: number;
  offset?: number;
}): Promise<{ articles: RSSArticleWithScore[]; total: number }> {
  const queryParams = new URLSearchParams();
  if (params.folder) queryParams.append('folder', params.folder);
  if (params.unread) queryParams.append('unread', 'true');
  if (params.limit) queryParams.append('limit', params.limit.toString());
  if (params.offset) queryParams.append('offset', params.offset.toString());

  const response = await api.get(`/api/rss/articles/smart?${queryParams.toString()}`);
  return response.data;
}
```

**Step 2: Run TypeScript check**

```bash
cd zettelkasten-front
npm run type-check
```

Expected: PASS (no type errors)

**Step 3: Commit**

```bash
git add zettelkasten-front/src/api/rss.ts
git commit -m "feat(rss): add smart feed API client function

Adds getSmartRSSArticles function with types for SmartFeedScore
and RSSArticleWithScore.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 10: Add Priority Toggle to Feed Edit Dialog

**Files:**
- Modify: `zettelkasten-front/src/components/rss/EditFeedDialog.tsx` (or similar)

**Step 1: Find the feed edit dialog component**

```bash
cd zettelkasten-front
find src -name "*eed*ialog*" -o -name "*eed*dit*" | head -5
```

Based on existing patterns, add a checkbox for the priority field. The exact implementation depends on the existing component structure.

**Step 2: Add priority checkbox to form**

In the feed edit form, add:

```tsx
<FormControlLabel
  control={
    <Checkbox
      checked={feed.priority || false}
      onChange={(e) => setFeed({ ...feed, priority: e.target.checked })}
    />
  }
  label="Priority feed (always show in smart feed)"
/>
```

**Step 3: Update feed update handler**

Ensure the `priority` field is included when calling `updateRSSFeed`.

**Step 4: Test in browser**

```bash
cd zettelkasten-front
npm start
```

Manual: Open feed edit dialog, verify priority checkbox appears and saves.

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/rss/EditFeedDialog.tsx
git commit -m "feat(rss): add priority toggle to feed edit dialog

Allows users to mark feeds as priority which gives them +100 score
in the smart feed ranking.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 11: Add Smart Feed Navigation Item

**Files:**
- Modify: `zettelkasten-front/src/pages/ArticlesPage.tsx` (or similar RSS page)

**Step 1: Find the RSS articles page**

```bash
cd zettelkasten-front
find src -name "*rticle*" -o -name "*RSS*" | head -10
```

**Step 2: Add "Smart Feed" option to navigation/filters**

Add a filter option for "Smart Feed" alongside "All Articles", "Unread", etc.:

```tsx
<MenuItem onClick={() => setFeedFilter('smart')}>
  <ListItemIcon><AutoAwesomeIcon /></ListItemIcon>
  <ListItemText>Smart Feed</ListItemText>
</MenuItem>
```

**Step 3: Handle smart feed selection**

When `feedFilter === 'smart'`, call `getSmartRSSArticles` instead of `getRSSArticles`.

**Step 4: Display smart score indicator**

Optionally show a small badge or tooltip indicating why an article is ranked high:

```tsx
{article.smart_score && (
  <Tooltip title={article.smart_score.reason}>
    <AutoAwesome fontSize="small" color="action" />
  </Tooltip>
)}
```

**Step 5: Test in browser**

Manual: Navigate to Smart Feed, verify articles load and show indicators.

**Step 6: Commit**

```bash
git add zettelkasten-front/src/pages/ArticlesPage.tsx
git commit -m "feat(rss): add Smart Feed navigation option and UI

Adds 'Smart Feed' filter option in RSS articles page that shows
articles ranked by the smart scoring algorithm with visual
indicators for score reasoning.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 12: Manual Testing and Verification

**Step 1: Run full backend test suite**

```bash
cd go-backend
go test ./... 2>&1 | tail -30
```

Expected: All non-DB tests pass

**Step 2: Run frontend tests**

```bash
cd zettelkasten-front
npm test
```

Expected: PASS

**Step 3: Manual testing checklist**

1. Create a feed and mark it as priority
2. Wait for articles to be fetched or manually refresh
3. Navigate to Smart Feed
4. Verify priority feed articles appear at top
5. Mark a feed as non-priority, verify ranking changes
6. Convert an article to card, verify that feed gets boost
7. Check low-volume feeds appear higher than high-volume feeds

**Step 4: Final commit if needed**

```bash
git commit --allow-empty -m "feat(rss): complete smart feed implementation

All tasks completed and manual testing passed.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Summary

This implementation plan breaks down the smart feed feature into 12 bite-sized tasks:

1. Database migration and model updates for priority flag
2. Smart score model types
3-6. Core scoring logic (volume, interaction, priority, reasoning)
7. Main service function
8. API route and handler
9-11. Frontend integration (API, UI, navigation)
12. Testing and verification

Each task follows TDD: write failing test, implement, verify passing, commit.
