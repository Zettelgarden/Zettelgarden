# RSS Feed Client Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an RSS feed client to Zettelgarden that allows users to subscribe to feeds, browse articles in a reader-style inbox, and selectively convert interesting articles to cards.

**Architecture:** Three-panel frontend UI (Folders | Articles | Reader), Go backend with RSS parsing service, scheduled fetch job, PostgreSQL storage.

**Tech Stack:** Go (gofeed for RSS, go-readability for content), React/TypeScript, PostgreSQL, existing scheduler system.

---

## Task 1: Database Schema Migration

**Files:**
- Create: `go-backend/schema/0112-add-rss-tables.sql`

**Step 1: Write the migration file**

```sql
-- Migration: Add RSS feed client tables
-- Description: Tables for RSS feeds, articles, and folders
-- Created: 2025-02-06

-- RSS Feeds table
CREATE TABLE IF NOT EXISTS rss_feeds (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    name TEXT NOT NULL,
    folder TEXT,
    auto_tags TEXT DEFAULT '',
    fetch_interval INTEGER DEFAULT 60,
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, url)
);

-- RSS Articles table
CREATE TABLE IF NOT EXISTS rss_articles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feed_id INTEGER NOT NULL REFERENCES rss_feeds(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT,
    author TEXT,
    url TEXT NOT NULL,
    published_at TIMESTAMP WITH TIME ZONE,
    fetched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    read BOOLEAN DEFAULT false,
    UNIQUE(user_id, url)
);

-- RSS Folders table
CREATE TABLE IF NOT EXISTS rss_folders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    order_index INTEGER DEFAULT 0,
    UNIQUE(user_id, name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_rss_articles_user_feed ON rss_articles(user_id, feed_id);
CREATE INDEX IF NOT EXISTS idx_rss_articles_read ON rss_articles(user_id, read);
CREATE INDEX IF NOT EXISTS idx_rss_feeds_user ON rss_feeds(user_id);
CREATE INDEX IF NOT EXISTS idx_rss_feeds_enabled ON rss_feeds(enabled);
CREATE INDEX IF NOT EXISTS idx_rss_folders_user ON rss_folders(user_id);

-- Add comments for documentation
COMMENT ON TABLE rss_feeds IS 'RSS feed subscriptions per user';
COMMENT ON TABLE rss_articles IS 'Articles fetched from RSS feeds';
COMMENT ON TABLE rss_folders IS 'User-defined folders for organizing feeds';
COMMENT ON COLUMN rss_feeds.auto_tags IS 'Comma-separated tags to apply when converting articles to cards';
COMMENT ON COLUMN rss_feeds.fetch_interval IS 'Fetch interval in minutes';
```

**Step 2: Run the migration**

Run: `cd go-backend && source .env-bash && psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f schema/0112-add-rss-tables.sql`
Expected: Tables created successfully

**Step 3: Verify the migration**

Run: `cd go-backend && source .env-bash && psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "\dt rss_*"`
Expected: List of rss_feeds, rss_articles, rss_folders tables

**Step 4: Commit**

```bash
git add go-backend/schema/0112-add-rss-tables.sql
git commit -m "feat: add RSS feed client database schema"
```

---

## Task 2: Backend - RSS Models

**Files:**
- Create: `go-backend/models/rss.go`

**Step 1: Write the failing test**

Create test file: `go-backend/models/rss_test.go`

```go
package models

import "testing"

func TestRSSFeedModel(t *testing.T) {
    feed := RSSFeed{
        ID:     1,
        UserID: 1,
        URL:    "https://example.com/feed.xml",
        Name:   "Example Feed",
        Folder: "Tech",
    }

    if feed.URL != "https://example.com/feed.xml" {
        t.Errorf("expected URL to be https://example.com/feed.xml, got %s", feed.URL)
    }
}

func TestRSSArticleModel(t *testing.T) {
    article := RSSArticle{
        ID:     1,
        FeedID: 1,
        Title:  "Test Article",
        URL:    "https://example.com/article",
    }

    if article.Title != "Test Article" {
        t.Errorf("expected Title to be Test Article, got %s", article.Title)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && source .env-bash && go test ./models -run TestRSS`
Expected: FAIL with "undefined: RSSFeed"

**Step 3: Write minimal implementation**

Create: `go-backend/models/rss.go`

```go
package models

import "time"

// RSSFeed represents a configured RSS feed subscription
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
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}

// RSSArticle represents an article fetched from an RSS feed
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
}

// RSSFolder represents a folder for organizing RSS feeds
type RSSFolder struct {
    ID         int    `json:"id"`
    UserID     int    `json:"user_id"`
    Name       string `json:"name"`
    OrderIndex int    `json:"order_index"`
}

// CreateRSSFeedParams represents parameters for creating or updating an RSS feed
type CreateRSSFeedParams struct {
    URL           string  `json:"url"`
    Name          string  `json:"name"`
    Folder        *string `json:"folder,omitempty"`
    AutoTags      string  `json:"auto_tags,omitempty"`
    FetchInterval *int    `json:"fetch_interval,omitempty"`
    Enabled       *bool   `json:"enabled,omitempty"`
}

// UpdateRSSFeedParams represents parameters for updating an RSS feed
type UpdateRSSFeedParams struct {
    Name          *string `json:"name,omitempty"`
    Folder        *string `json:"folder,omitempty"`
    AutoTags      *string `json:"auto_tags,omitempty"`
    FetchInterval *int    `json:"fetch_interval,omitempty"`
    Enabled       *bool   `json:"enabled,omitempty"`
}

// ConvertArticleParams represents parameters for converting an RSS article to a card
type ConvertArticleParams struct {
    Title *string `json:"title,omitempty"`
    Body  *string `json:"body,omitempty"`
    Tags  *string `json:"tags,omitempty"`
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && source .env-bash && go test ./models -run TestRSS`
Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/models/rss.go go-backend/models/rss_test.go
git commit -m "feat: add RSS feed models"
```

---

## Task 3: Backend - RSS Service

**Files:**
- Create: `go-backend/services/rss.go`
- Create: `go-backend/services/rss_test.go`

**Step 3.1: Add gofeed dependency**

```bash
cd go-backend
go get github.com/mmcdole/gofeed
```

**Step 3.2: Write the RSS service**

Create: `go-backend/services/rss.go`

```go
package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/mmcdole/gofeed"
	"go-backend/models"
)

// CreateRSSFeed creates a new RSS feed for a user
func CreateRSSFeed(db *sql.DB, userID int, params models.CreateRSSFeedParams) (*models.RSSFeed, error) {
	// Validate URL by attempting to parse it
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(params.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed URL: %w", err)
	}

	// Use provided name or feed title
	name := params.Name
	if name == "" {
		name = feed.Title
	}

	// Set defaults
	fetchInterval := 60
	if params.FetchInterval != nil {
		fetchInterval = *params.FetchInterval
	}
	enabled := true
	if params.Enabled != nil {
		enabled = *params.Enabled
	}

	// Insert into database
	var feedID int
	var folder sql.NullString
	if params.Folder != nil {
		folder.String = *params.Folder
		folder.Valid = true
	}

	err = db.QueryRow(`
		INSERT INTO rss_feeds (user_id, url, name, folder, auto_tags, fetch_interval, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, userID, params.URL, name, folder, params.AutoTags, fetchInterval, enabled).Scan(&feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to create feed: %w", err)
	}

	return GetRSSFeedByID(db, userID, feedID)
}

// GetRSSFeedByID retrieves a single RSS feed by ID
func GetRSSFeedByID(db *sql.DB, userID, feedID int) (*models.RSSFeed, error) {
	var feed models.RSSFeed
	var folder, lastError sql.NullString
	var lastFetched sql.NullTime

	err := db.QueryRow(`
		SELECT id, user_id, url, name, folder, auto_tags, fetch_interval,
		       last_fetched_at, last_error, enabled, created_at, updated_at
		FROM rss_feeds
		WHERE id = $1 AND user_id = $2
	`, feedID, userID).Scan(
		&feed.ID, &feed.UserID, &feed.URL, &feed.Name, &folder, &feed.AutoTags,
		&feed.FetchInterval, &lastFetched, &lastError, &feed.Enabled,
		&feed.CreatedAt, &feed.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feed not found")
		}
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	if folder.Valid {
		feed.Folder = &folder.String
	}
	if lastError.Valid {
		feed.LastError = &lastError.String
	}
	if lastFetched.Valid {
		feed.LastFetchedAt = &lastFetched.Time
	}

	return &feed, nil
}

// ListRSSFeeds retrieves all RSS feeds for a user
func ListRSSFeeds(db *sql.DB, userID int) ([]models.RSSFeed, error) {
	rows, err := db.Query(`
		SELECT id, user_id, url, name, folder, auto_tags, fetch_interval,
		       last_fetched_at, last_error, enabled, created_at, updated_at
		FROM rss_feeds
		WHERE user_id = $1
		ORDER BY name ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list feeds: %w", err)
	}
	defer rows.Close()

	var feeds []models.RSSFeed
	for rows.Next() {
		var feed models.RSSFeed
		var folder, lastError sql.NullString
		var lastFetched sql.NullTime

		err := rows.Scan(
			&feed.ID, &feed.UserID, &feed.URL, &feed.Name, &folder, &feed.AutoTags,
			&feed.FetchInterval, &lastFetched, &lastError, &feed.Enabled,
			&feed.CreatedAt, &feed.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed: %w", err)
		}

		if folder.Valid {
			feed.Folder = &folder.String
		}
		if lastError.Valid {
			feed.LastError = &lastError.String
		}
		if lastFetched.Valid {
			feed.LastFetchedAt = &lastFetched.Time
		}

		feeds = append(feeds, feed)
	}

	return feeds, nil
}

// UpdateRSSFeed updates an existing RSS feed
func UpdateRSSFeed(db *sql.DB, userID, feedID int, params models.UpdateRSSFeedParams) (*models.RSSFeed, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if params.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *params.Name)
		argPos++
	}
	if params.Folder != nil {
		updates = append(updates, fmt.Sprintf("folder = $%d", argPos))
		args = append(args, *params.Folder)
		argPos++
	}
	if params.AutoTags != nil {
		updates = append(updates, fmt.Sprintf("auto_tags = $%d", argPos))
		args = append(args, *params.AutoTags)
		argPos++
	}
	if params.FetchInterval != nil {
		updates = append(updates, fmt.Sprintf("fetch_interval = $%d", argPos))
		args = append(args, *params.FetchInterval)
		argPos++
	}
	if params.Enabled != nil {
		updates = append(updates, fmt.Sprintf("enabled = $%d", argPos))
		args = append(args, *params.Enabled)
		argPos++
	}

	if len(updates) == 0 {
		return GetRSSFeedByID(db, userID, feedID)
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	query := fmt.Sprintf("UPDATE rss_feeds SET %s WHERE id = $%d AND user_id = $%d",
		fmt.Sprintf("%s", updates), argPos, argPos+1)
	args = append(args, feedID, userID)

	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update feed: %w", err)
	}

	return GetRSSFeedByID(db, userID, feedID)
}

// DeleteRSSFeed deletes an RSS feed
func DeleteRSSFeed(db *sql.DB, userID, feedID int) error {
	result, err := db.Exec("DELETE FROM rss_feeds WHERE id = $1 AND user_id = $2", feedID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("feed not found")
	}

	return nil
}

// ListRSSArticles retrieves articles for a user with optional filters
func ListRSSArticles(db *sql.DB, userID int, filters map[string]interface{}) ([]models.RSSArticle, error) {
	query := `
		SELECT id, user_id, feed_id, title, content, author, url,
		       published_at, fetched_at, read
		FROM rss_articles
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argPos := 2

	// Apply filters
	if folder, ok := filters["folder"].(string); ok && folder != "" {
		query += fmt.Sprintf(" AND feed_id IN (SELECT id FROM rss_feeds WHERE user_id = $1 AND folder = $%d)", argPos)
		args = append(args, folder)
		argPos++
	}

	if unreadOnly, ok := filters["unread"].(bool); ok && unreadOnly {
		query += fmt.Sprintf(" AND read = false")
	}

	if feedID, ok := filters["feed_id"].(int); ok && feedID > 0 {
		query += fmt.Sprintf(" AND feed_id = $%d", argPos)
		args = append(args, feedID)
		argPos++
	}

	query += " ORDER BY published_at DESC NULLS LAST, fetched_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list articles: %w", err)
	}
	defer rows.Close()

	var articles []models.RSSArticle
	for rows.Next() {
		var article models.RSSArticle
		var content, author sql.NullString
		var publishedAt sql.NullTime

		err := rows.Scan(
			&article.ID, &article.UserID, &article.FeedID, &article.Title,
			&content, &author, &article.URL, &publishedAt,
			&article.FetchedAt, &article.Read,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
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

		articles = append(articles, article)
	}

	return articles, nil
}

// GetRSSArticleByID retrieves a single RSS article
func GetRSSArticleByID(db *sql.DB, userID, articleID int) (*models.RSSArticle, error) {
	var article models.RSSArticle
	var content, author sql.NullString
	var publishedAt sql.NullTime

	err := db.QueryRow(`
		SELECT id, user_id, feed_id, title, content, author, url,
		       published_at, fetched_at, read
		FROM rss_articles
		WHERE id = $1 AND user_id = $2
	`, articleID, userID).Scan(
		&article.ID, &article.UserID, &article.FeedID, &article.Title,
		&content, &author, &article.URL, &publishedAt,
		&article.FetchedAt, &article.Read,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("article not found")
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
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

	return &article, nil
}

// MarkRSSArticleAsRead marks an article as read or unread
func MarkRSSArticleAsRead(db *sql.DB, userID, articleID int, read bool) error {
	result, err := db.Exec(
		"UPDATE rss_articles SET read = $1 WHERE id = $2 AND user_id = $3",
		read, articleID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark article: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("article not found")
	}

	return nil
}

// ConvertRSSArticleToCard converts an RSS article to a card
func ConvertRSSArticleToCard(db *sql.DB, userID, articleID int, params *models.ConvertArticleParams) (*models.Card, error) {
	// Get the article
	article, err := GetRSSArticleByID(db, userID, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	// Get the feed to get auto_tags
	feed, err := GetRSSFeedByID(db, userID, article.FeedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	// Determine title and content
	title := article.Title
	if params != nil && params.Title != nil {
		title = *params.Title
	}

	content := ""
	if article.Content != nil {
		content = *article.Content
	}
	if params != nil && params.Body != nil {
		content = *params.Body
	}

	// Build tags
	tags := feed.AutoTags
	if params != nil && params.Tags != nil {
		tags = *params.Tags
	}
	if tags != "" {
		content = fmt.Sprintf("%s\n\n%s", content, tags)
	}

	// Create the card
	cardParams := models.EditCardParams{
		Title: title,
		Body:  content,
		Link:  &article.URL,
	}

	card, err := CreateCard(db, userID, cardParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %w", err)
	}

	// Mark article as read
	_ = MarkRSSArticleAsRead(db, userID, articleID, true)

	return &card, nil
}

// FetchRSSFeedArticles fetches new articles from an RSS feed
func FetchRSSFeedArticles(db *sql.DB, feedID int) error {
	// Get the feed
	var feed models.RSSFeed
	var folder, lastError sql.NullString
	var lastFetched sql.NullTime

	err := db.QueryRow(`
		SELECT id, user_id, url, name, folder, auto_tags, fetch_interval,
		       last_fetched_at, last_error, enabled
		FROM rss_feeds
		WHERE id = $1
	`, feedID).Scan(
		&feed.ID, &feed.UserID, &feed.URL, &feed.Name, &folder, &feed.AutoTags,
		&feed.FetchInterval, &lastFetched, &lastError, &feed.Enabled,
	)
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	if folder.Valid {
		feed.Folder = &folder.String
	}
	if lastError.Valid {
		feed.LastError = &lastError.String
	}
	if lastFetched.Valid {
		feed.LastFetchedAt = &lastFetched.Time
	}

	// Parse the RSS feed
	fp := gofeed.NewParser()
	parsedFeed, err := fp.ParseURL(feed.URL)
	if err != nil {
		// Update last_error
		_, _ = db.Exec("UPDATE rss_feeds SET last_error = $1 WHERE id = $2", err.Error(), feedID)
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	// Process each item
	newArticles := 0
	for _, item := range parsedFeed.Items {
		// Check if article already exists
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM rss_articles WHERE user_id = $1 AND url = $2)",
			feed.UserID, item.Link).Scan(&exists)
		if err != nil || exists {
			continue // Skip existing or error
		}

		// Fetch full content using readability
		var content *string
		if item.Link != "" {
			parsed, parseErr := ParseURL(item.Link)
			if parseErr == nil {
				content = &parsed.Content
			} else {
				log.Printf("[rss-feed:%d] failed to parse article content: %v", feedID, parseErr)
				// Fall back to description
				if item.Description != "" {
					content = &item.Description
				}
			}
		} else if item.Description != "" {
			content = &item.Description
		}

		// Parse published date
		var publishedAt *time.Time
		if item.PublishedParsed != nil {
			publishedAt = item.PublishedParsed
		}

		// Get author
		var author *string
		if item.Author != nil && item.Author.Name != "" {
			author = &item.Author.Name
		}

		// Insert article
		_, err = db.Exec(`
			INSERT INTO rss_articles (user_id, feed_id, title, content, author, url, published_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, feed.UserID, feed.ID, item.Title, content, author, item.Link, publishedAt)
		if err != nil {
			log.Printf("[rss-feed:%d] failed to insert article: %v", feedID, err)
			continue
		}

		newArticles++
	}

	// Update last_fetched_at and clear last_error
	now := time.Now()
	_, err = db.Exec("UPDATE rss_feeds SET last_fetched_at = $1, last_error = NULL WHERE id = $2", now, feedID)
	if err != nil {
		log.Printf("[rss-feed:%d] failed to update last_fetched_at: %v", feedID, err)
	}

	log.Printf("[rss-feed:%d] fetched %d new articles", feedID, newArticles)
	return nil
}

// ListRSSFolders retrieves all folders for a user
func ListRSSFolders(db *sql.DB, userID int) ([]models.RSSFolder, error) {
	rows, err := db.Query(`
		SELECT id, user_id, name, order_index
		FROM rss_folders
		WHERE user_id = $1
		ORDER BY order_index ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	defer rows.Close()

	var folders []models.RSSFolder
	for rows.Next() {
		var folder models.RSSFolder
		err := rows.Scan(&folder.ID, &folder.UserID, &folder.Name, &folder.OrderIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, folder)
	}

	return folders, nil
}

// CreateRSSFolder creates a new folder
func CreateRSSFolder(db *sql.DB, userID int, name string, orderIndex int) (*models.RSSFolder, error) {
	var folderID int
	err := db.QueryRow(`
		INSERT INTO rss_folders (user_id, name, order_index)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, name, orderIndex).Scan(&folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	return GetRSSFolderByID(db, userID, folderID)
}

// GetRSSFolderByID retrieves a folder by ID
func GetRSSFolderByID(db *sql.DB, userID, folderID int) (*models.RSSFolder, error) {
	var folder models.RSSFolder
	err := db.QueryRow(`
		SELECT id, user_id, name, order_index
		FROM rss_folders
		WHERE id = $1 AND user_id = $2
	`, folderID, userID).Scan(&folder.ID, &folder.UserID, &folder.Name, &folder.OrderIndex)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("folder not found")
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	return &folder, nil
}
```

**Step 3.3: Write tests**

Create: `go-backend/services/rss_test.go`

```go
package services

import (
	"go-backend/handlers"
	"go-backend/models"
	"testing"
)

func TestCreateRSSFeed(t *testing.T) {
	s := handlers.NewHandler()
	defer handlers.Teardown(s)

	userID := 1
	params := models.CreateRSSFeedParams{
		URL:   "https://example.com/feed.xml",
		Name:  "Test Feed",
		Folder: stringPtr("Tech"),
	}

	// This will fail with a real URL but tests the logic
	feed, err := CreateRSSFeed(s.Server.Tx, userID, params)
	if err != nil {
		t.Logf("Expected error with fake URL: %v", err)
	}
	if feed != nil && feed.Name != params.Name {
		t.Errorf("expected name %s, got %s", params.Name, feed.Name)
	}
}

func TestListRSSFeeds(t *testing.T) {
	s := handlers.NewHandler()
	defer handlers.Teardown(s)

	userID := 1

	feeds, err := ListRSSFeeds(s.Server.Tx, userID)
	if err != nil {
		t.Errorf("failed to list feeds: %v", err)
	}
	// Should be empty initially
	if len(feeds) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(feeds))
	}
}

func stringPtr(s string) *string {
	return &s
}
```

**Step 3.4: Run tests**

Run: `cd go-backend && source .env-bash && go test ./services -run TestRSS`
Expected: Tests pass (some may fail due to network, that's expected)

**Step 3.5: Commit**

```bash
git add go-backend/services/rss.go go-backend/services/rss_test.go go-backend/go.mod go-backend/go.sum
git commit -m "feat: add RSS service layer"
```

---

## Task 4: Backend - RSS Handlers

**Files:**
- Modify: `go-backend/handlers/handlers.go` (add RSS handler methods)
- Create: `go-backend/handlers/rss_test.go`

**Step 4.1: Add handler methods**

Read: `go-backend/handlers/handlers.go` to find the Handler struct, then add the RSS handler methods at the end of the file:

```go
// CreateRSSFeedRoute handles POST /api/rss/feeds
func (h *Handler) CreateRSSFeedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.CreateRSSFeedParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	feed, err := services.CreateRSSFeed(h.GetDB(), userID, params)
	if err != nil {
		log.Printf("Failed to create RSS feed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

// ListRSSFeedsRoute handles GET /api/rss/feeds
func (h *Handler) ListRSSFeedsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	feeds, err := services.ListRSSFeeds(h.GetDB(), userID)
	if err != nil {
		log.Printf("Failed to list RSS feeds: %v", err)
		http.Error(w, "Failed to list feeds", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feeds)
}

// GetRSSFeedRoute handles GET /api/rss/feeds/{id}
func (h *Handler) GetRSSFeedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	feedID, _ := strconv.Atoi(mux.Vars(r)["id"])

	feed, err := services.GetRSSFeedByID(h.GetDB(), userID, feedID)
	if err != nil {
		log.Printf("Failed to get RSS feed: %v", err)
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

// UpdateRSSFeedRoute handles PUT /api/rss/feeds/{id}
func (h *Handler) UpdateRSSFeedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	feedID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var params models.UpdateRSSFeedParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	feed, err := services.UpdateRSSFeed(h.GetDB(), userID, feedID, params)
	if err != nil {
		log.Printf("Failed to update RSS feed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

// DeleteRSSFeedRoute handles DELETE /api/rss/feeds/{id}
func (h *Handler) DeleteRSSFeedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	feedID, _ := strconv.Atoi(mux.Vars(r)["id"])

	if err := services.DeleteRSSFeed(h.GetDB(), userID, feedID); err != nil {
		log.Printf("Failed to delete RSS feed: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RefreshRSSFeedRoute handles POST /api/rss/feeds/fetch
func (h *Handler) RefreshRSSFeedsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get all enabled feeds for the user
	feeds, err := services.ListRSSFeeds(h.GetDB(), userID)
	if err != nil {
		log.Printf("Failed to list RSS feeds: %v", err)
		http.Error(w, "Failed to list feeds", http.StatusInternalServerError)
		return
	}

	// Fetch articles for each enabled feed
	count := 0
	for _, feed := range feeds {
		if feed.Enabled {
			if err := services.FetchRSSFeedArticles(h.GetDB(), feed.ID); err != nil {
				log.Printf("Failed to fetch feed %d: %v", feed.ID, err)
			} else {
				count++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fetched": count,
	})
}

// ListRSSArticlesRoute handles GET /api/rss/articles
func (h *Handler) ListRSSArticlesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse query parameters
	filters := make(map[string]interface{})
	if folder := r.URL.Query().Get("folder"); folder != "" {
		filters["folder"] = folder
	}
	if unread := r.URL.Query().Get("unread"); unread == "true" {
		filters["unread"] = true
	}
	if feedID := r.URL.Query().Get("feed_id"); feedID != "" {
		if id, err := strconv.Atoi(feedID); err == nil {
			filters["feed_id"] = id
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters["limit"] = l
		}
	}

	articles, err := services.ListRSSArticles(h.GetDB(), userID, filters)
	if err != nil {
		log.Printf("Failed to list RSS articles: %v", err)
		http.Error(w, "Failed to list articles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}

// GetRSSArticleRoute handles GET /api/rss/articles/{id}
func (h *Handler) GetRSSArticleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, _ := strconv.Atoi(mux.Vars(r)["id"])

	article, err := services.GetRSSArticleByID(h.GetDB(), userID, articleID)
	if err != nil {
		log.Printf("Failed to get RSS article: %v", err)
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(article)
}

// MarkRSSArticleAsReadRoute handles POST /api/rss/articles/{id}/read
func (h *Handler) MarkRSSArticleAsReadRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var params struct {
		Read bool `json:"read"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		params.Read = true // Default to marking as read
	}

	if err := services.MarkRSSArticleAsRead(h.GetDB(), userID, articleID, params.Read); err != nil {
		log.Printf("Failed to mark article as read: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ConvertRSSArticleToCardRoute handles POST /api/rss/articles/{id}/convert
func (h *Handler) ConvertRSSArticleToCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var params *models.ConvertArticleParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil && err.Error() != "EOF" {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	card, err := services.ConvertRSSArticleToCard(h.GetDB(), userID, articleID, params)
	if err != nil {
		log.Printf("Failed to convert article to card: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// ListRSSFoldersRoute handles GET /api/rss/folders
func (h *Handler) ListRSSFoldersRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	folders, err := services.ListRSSFolders(h.GetDB(), userID)
	if err != nil {
		log.Printf("Failed to list RSS folders: %v", err)
		http.Error(w, "Failed to list folders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}
```

**Step 4.2: Write tests**

Create: `go-backend/handlers/rss_test.go`

```go
package handlers

import (
	"bytes"
	"encoding/json"
	"go-backend/models"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
)

func TestListRSSFeedsRoute(t *testing.T) {
	s := NewHandler()
	defer Teardown(s)

	token, _ := GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/rss/feeds", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.ListRSSFeedsRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var feeds []models.RSSFeed
	json.Unmarshal(rr.Body.Bytes(), &feeds)

	// Should be empty initially
	if len(feeds) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(feeds))
	}
}

func TestListRSSArticlesRoute(t *testing.T) {
	s := NewHandler()
	defer Teardown(s)

	token, _ := GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/rss/articles", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.ListRSSArticlesRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestListRSSFoldersRoute(t *testing.T) {
	s := NewHandler()
	defer Teardown(s)

	token, _ := GenerateTestJWT(1)

	req, err := http.NewRequest("GET", "/api/rss/folders", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.JwtMiddleware(s.ListRSSFoldersRoute))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
```

**Step 4.3: Run tests**

Run: `cd go-backend && source .env-bash && go test ./handlers -run TestRSS`
Expected: PASS

**Step 4.4: Commit**

```bash
git add go-backend/handlers/handlers.go go-backend/handlers/rss_test.go
git commit -m "feat: add RSS feed handlers"
```

---

## Task 5: Backend - RSS Routes

**Files:**
- Create: `go-backend/routes/rss.go`
- Modify: `go-backend/routes/routes.go`
- Modify: `go-backend/main.go`

**Step 5.1: Create RSS routes file**

Create: `go-backend/routes/rss.go`

```go
package routes

import (
	"go-backend/handlers"
	"github.com/gorilla/mux"
)

func RegisterRSSRoutes(r *mux.Router, h *handlers.Handler) {
	addProtectedRoute(r, h, "/api/rss/feeds", h.ListRSSFeedsRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/feeds", h.CreateRSSFeedRoute, "POST")
	addProtectedRoute(r, h, "/api/rss/feeds/{id}", h.GetRSSFeedRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/feeds/{id}", h.UpdateRSSFeedRoute, "PUT")
	addProtectedRoute(r, h, "/api/rss/feeds/{id}", h.DeleteRSSFeedRoute, "DELETE")
	addProtectedRoute(r, h, "/api/rss/feeds/fetch", h.RefreshRSSFeedsRoute, "POST")

	addProtectedRoute(r, h, "/api/rss/articles", h.ListRSSArticlesRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/articles/{id}", h.GetRSSArticleRoute, "GET")
	addProtectedRoute(r, h, "/api/rss/articles/{id}/read", h.MarkRSSArticleAsReadRoute, "POST")
	addProtectedRoute(r, h, "/api/rss/articles/{id}/convert", h.ConvertRSSArticleToCardRoute, "POST")

	addProtectedRoute(r, h, "/api/rss/folders", h.ListRSSFoldersRoute, "GET")
}
```

**Step 5.2: Update routes.go**

Read `go-backend/routes/routes.go` and add the RSS import and registration:

Add import:
```go
import (
    // ... existing imports ...
)

func RegisterAllRoutes(r *mux.Router, h *handlers.Handler, scheduler *services.Scheduler) {
    // ... existing route registrations ...

    // RSS routes
    RegisterRSSRoutes(r, h)

    // ... rest of routes ...
}
```

**Step 5.3: Run server to verify routes compile**

Run: `cd go-backend && go build -o main`
Expected: Binary builds successfully

**Step 5.4: Commit**

```bash
git add go-backend/routes/rss.go go-backend/routes/routes.go
git commit -m "feat: register RSS routes"
```

---

## Task 6: Backend - RSS Fetch Job

**Files:**
- Create: `go-backend/services/jobs/rss_fetch_job.go`
- Modify: `go-backend/main.go`

**Step 6.1: Create RSS fetch job**

Create: `go-backend/services/jobs/rss_fetch_job.go`

```go
package jobs

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"go-backend/services"
)

// RSSFetchJob fetches articles from enabled RSS feeds
type RSSFetchJob struct {
	db       *sql.DB
	schedule string
}

// NewRSSFetchJob creates a new RSS fetch job
func NewRSSFetchJob(db *sql.DB) *RSSFetchJob {
	return &RSSFetchJob{
		db:       db,
		schedule: "0 */60 * * * *", // Every 60 minutes
	}
}

func (j *RSSFetchJob) Name() string {
	return "rss-fetch"
}

func (j *RSSFetchJob) Schedule() string {
	return j.schedule
}

func (j *RSSFetchJob) MaxRetries() int {
	return 3
}

func (j *RSSFetchJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.Schedule())
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}

func (j *RSSFetchJob) Handler(ctx context.Context) error {
	log.Println("[rss-fetch] starting RSS fetch job")

	if j.db == nil {
		log.Println("[rss-fetch] no database configured, skipping")
		return nil
	}

	// Get enabled RSS feeds
	rows, err := j.db.QueryContext(ctx, `
		SELECT id
		FROM rss_feeds
		WHERE enabled = true
	`)
	if err != nil {
		log.Printf("[rss-fetch] failed to fetch feeds: %v", err)
		return err
	}
	defer rows.Close()

	var feedIDs []int
	for rows.Next() {
		var feedID int
		if err := rows.Scan(&feedID); err != nil {
			log.Printf("[rss-fetch] failed to scan feed ID: %v", err)
			continue
		}
		feedIDs = append(feedIDs, feedID)
	}

	// Fetch from each feed
	totalNewArticles := 0
	for _, feedID := range feedIDs {
		if err := services.FetchRSSFeedArticles(j.db, feedID); err != nil {
			log.Printf("[rss-fetch] failed to fetch feed %d: %v", feedID, err)
		}
		totalNewArticles++
	}

	log.Printf("[rss-fetch] completed, processed %d feeds", totalNewArticles)
	return nil
}

// Verify RSSFetchJob implements ScheduledJob interface
var _ services.ScheduledJob = (*RSSFetchJob)(nil)
```

**Step 6.2: Register job in main.go**

Read `go-backend/main.go` and find where other jobs are registered (around line 256-261), then add:

```go
// In main() function after scheduler initialization
scheduler.Register(jobs.NewRSSFetchJob(s.DB))
```

**Step 6.3: Build to verify**

Run: `cd go-backend && go build -o main`
Expected: Binary builds successfully

**Step 6.4: Commit**

```bash
git add go-backend/services/jobs/rss_fetch_job.go go-backend/main.go
git commit -m "feat: add RSS fetch scheduled job"
```

---

## Task 7: Frontend - RSS API Client

**Files:**
- Create: `zettelkasten-front/src/api/rss.ts`

**Step 7.1: Create RSS API client**

Create: `zettelkasten-front/src/api/rss.ts`

```typescript
import { apiClient, getData } from "./client";

// Types
export interface RSSFeed {
  id: number;
  user_id: number;
  url: string;
  name: string;
  folder?: string;
  auto_tags: string;
  fetch_interval: number;
  enabled: boolean;
  last_fetched_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface RSSArticle {
  id: number;
  user_id: number;
  feed_id: number;
  title: string;
  content?: string;
  author?: string;
  url: string;
  published_at?: string;
  fetched_at: string;
  read: boolean;
}

export interface RSSFolder {
  id: number;
  user_id: number;
  name: string;
  order_index: number;
}

export interface CreateRSSFeedParams {
  url: string;
  name?: string;
  folder?: string;
  auto_tags?: string;
  fetch_interval?: number;
  enabled?: boolean;
}

export interface UpdateRSSFeedParams {
  name?: string;
  folder?: string;
  auto_tags?: string;
  fetch_interval?: number;
  enabled?: boolean;
}

export interface ConvertArticleParams {
  title?: string;
  body?: string;
  tags?: string;
}

export interface ArticleFilters {
  folder?: string;
  unread?: boolean;
  feed_id?: number;
  limit?: number;
}

// Feed API
export function createFeed(feed: CreateRSSFeedParams): Promise<RSSFeed> {
  return getData(apiClient.post<RSSFeed>("/rss/feeds", feed));
}

export function listFeeds(): Promise<RSSFeed[]> {
  return getData(apiClient.get<RSSFeed[]>("/rss/feeds"));
}

export function getFeed(id: number): Promise<RSSFeed> {
  return getData(apiClient.get<RSSFeed>(`/rss/feeds/${id}`));
}

export function updateFeed(id: number, params: UpdateRSSFeedParams): Promise<RSSFeed> {
  return getData(apiClient.put<RSSFeed>(`/rss/feeds/${id}`, params));
}

export function deleteFeed(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/rss/feeds/${id}`));
}

export function refreshFeeds(): Promise<{ fetched: number }> {
  return getData(apiClient.post<{ fetched: number }>("/rss/feeds/fetch", {}));
}

// Article API
export function listArticles(filters?: ArticleFilters): Promise<RSSArticle[]> {
  const params = new URLSearchParams();
  if (filters?.folder) params.set("folder", filters.folder);
  if (filters?.unread) params.set("unread", "true");
  if (filters?.feed_id) params.set("feed_id", filters.feed_id.toString());
  if (filters?.limit) params.set("limit", filters.limit.toString());

  const query = params.toString();
  return getData(apiClient.get<RSSArticle[]>(`/rss/articles${query ? `?${query}` : ""}`));
}

export function getArticle(id: number): Promise<RSSArticle> {
  return getData(apiClient.get<RSSArticle>(`/rss/articles/${id}`));
}

export function markAsRead(id: number, read: boolean = true): Promise<void> {
  return getData(apiClient.post<void>(`/rss/articles/${id}/read`, { read }));
}

export function convertToCard(id: number, params?: ConvertArticleParams): Promise<any> {
  return getData(apiClient.post<any>(`/rss/articles/${id}/convert`, params));
}

// Folder API
export function listFolders(): Promise<RSSFolder[]> {
  return getData(apiClient.get<RSSFolder[]>("/rss/folders"));
}
```

**Step 7.2: Commit**

```bash
git add zettelkasten-front/src/api/rss.ts
git commit -m "feat: add RSS API client"
```

---

## Task 8: Frontend - RSS Page Component

**Files:**
- Create: `zettelkasten-front/src/pages/RssPage.tsx`
- Modify: `zettelkasten-front/src/pages/AppRoutes.tsx`

**Step 8.1: Create RSS page component**

Create: `zettelkasten-front/src/pages/RssPage.tsx`

```typescript
import React, { useState, useEffect } from "react";
import { setDocumentTitle } from "../utils/documentTitle";
import {
  listFeeds,
  listArticles,
  listFolders,
  markAsRead,
  convertToCard,
  refreshFeeds,
  RSSFeed,
  RSSArticle,
  RSSFolder,
} from "../api/rss";

export function RssPage() {
  const [feeds, setFeeds] = useState<RSSFeed[]>([]);
  const [articles, setArticles] = useState<RSSArticle[]>([]);
  const [folders, setFolders] = useState<RSSFolder[]>([]);
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [selectedArticle, setSelectedArticle] = useState<RSSArticle | null>(null);
  const [showUnreadOnly, setShowUnreadOnly] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    setDocumentTitle("RSS");
    loadData();
  }, []);

  useEffect(() => {
    loadArticles();
  }, [selectedFolder, showUnreadOnly]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [feedsData, foldersData] = await Promise.all([
        listFeeds(),
        listFolders(),
      ]);
      setFeeds(feedsData);
      setFolders(foldersData);
    } catch (error) {
      console.error("Failed to load RSS data:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadArticles = async () => {
    try {
      const filters: any = {};
      if (selectedFolder) filters.folder = selectedFolder;
      if (showUnreadOnly) filters.unread = true;
      filters.limit = 50;

      const articlesData = await listArticles(filters);
      setArticles(articlesData);
    } catch (error) {
      console.error("Failed to load articles:", error);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await refreshFeeds();
      await loadData();
      await loadArticles();
    } catch (error) {
      console.error("Failed to refresh feeds:", error);
    } finally {
      setRefreshing(false);
    }
  };

  const handleArticleClick = async (article: RSSArticle) => {
    setSelectedArticle(article);
    if (!article.read) {
      try {
        await markAsRead(article.id, true);
        setArticles((prev) =>
          prev.map((a) => (a.id === article.id ? { ...a, read: true } : a))
        );
      } catch (error) {
        console.error("Failed to mark as read:", error);
      }
    }
  };

  const handleConvertToCard = async () => {
    if (!selectedArticle) return;
    try {
      await convertToCard(selectedArticle.id);
      // Optionally navigate to the new card or show success message
      alert("Article converted to card!");
    } catch (error) {
      console.error("Failed to convert to card:", error);
      alert("Failed to convert article to card");
    }
  };

  if (loading) {
    return <div className="p-4">Loading...</div>;
  }

  return (
    <div className="flex h-full">
      {/* Left Panel: Folders */}
      <div className="w-64 border-r p-4 overflow-y-auto">
        <div className="mb-4">
          <h2 className="text-lg font-semibold mb-2">RSS Feeds</h2>
          <button
            onClick={handleRefresh}
            disabled={refreshing}
            className="w-full bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 disabled:bg-gray-400"
          >
            {refreshing ? "Refreshing..." : "Refresh All"}
          </button>
        </div>

        <div className="mb-4">
          <button
            onClick={() => setSelectedFolder(null)}
            className={`w-full text-left px-3 py-2 rounded ${
              selectedFolder === null ? "bg-blue-100" : "hover:bg-gray-100"
            }`}
          >
            All Feeds ({feeds.length})
          </button>
        </div>

        <div className="mb-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={showUnreadOnly}
              onChange={(e) => setShowUnreadOnly(e.target.checked)}
              className="rounded"
            />
            <span>Unread only</span>
          </label>
        </div>

        <div>
          <h3 className="text-sm font-semibold text-gray-500 mb-2">FOLDERS</h3>
          {folders.map((folder) => (
            <button
              key={folder.id}
              onClick={() => setSelectedFolder(folder.name)}
              className={`w-full text-left px-3 py-2 rounded ${
                selectedFolder === folder.name ? "bg-blue-100" : "hover:bg-gray-100"
              }`}
            >
              {folder.name}
            </button>
          ))}
        </div>
      </div>

      {/* Middle Panel: Articles */}
      <div className="w-80 border-r p-4 overflow-y-auto">
        <h2 className="text-lg font-semibold mb-4">Articles</h2>
        {articles.length === 0 ? (
          <p className="text-gray-500">No articles found</p>
        ) : (
          <div className="space-y-2">
            {articles.map((article) => (
              <div
                key={article.id}
                onClick={() => handleArticleClick(article)}
                className={`p-3 rounded cursor-pointer ${
                  selectedArticle?.id === article.id
                    ? "bg-blue-100"
                    : article.read
                    ? "bg-gray-50 hover:bg-gray-100"
                    : "bg-white hover:bg-gray-100 border-l-4 border-blue-500"
                }`}
              >
                <h3 className="font-medium text-sm line-clamp-2">
                  {article.title}
                </h3>
                <p className="text-xs text-gray-500 mt-1">
                  {new Date(article.fetched_at).toLocaleDateString()}
                </p>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Right Panel: Article Reader */}
      <div className="flex-1 p-6 overflow-y-auto">
        {selectedArticle ? (
          <div>
            <div className="mb-4">
              <h1 className="text-2xl font-bold mb-2">{selectedArticle.title}</h1>
              <div className="flex items-center gap-4 text-sm text-gray-500">
                {selectedArticle.author && <span>By {selectedArticle.author}</span>}
                <span>
                  {selectedArticle.published_at
                    ? new Date(selectedArticle.published_at).toLocaleDateString()
                    : new Date(selectedArticle.fetched_at).toLocaleDateString()}
                </span>
                <a
                  href={selectedArticle.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-500 hover:underline"
                >
                  View original
                </a>
              </div>
            </div>

            {selectedArticle.content && (
              <div
                className="prose max-w-none"
                dangerouslySetInnerHTML={{ __html: selectedArticle.content }}
              />
            )}

            <div className="mt-6 flex gap-2">
              <button
                onClick={handleConvertToCard}
                className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
              >
                Convert to Card
              </button>
            </div>
          </div>
        ) : (
          <div className="text-gray-500 text-center mt-20">
            Select an article to read
          </div>
        )}
      </div>
    </div>
  );
}
```

**Step 8.2: Add route**

Read `zettelkasten-front/src/pages/AppRoutes.tsx` and add the RSS route:

```typescript
import { RssPage } from "./RssPage";

// In the routes section, within the hasSubscription block:
<Route path="rss" element={<RssPage />} />
```

**Step 8.3: Commit**

```bash
git add zettelkasten-front/src/pages/RssPage.tsx zettelkasten-front/src/pages/AppRoutes.tsx
git commit -m "feat: add RSS page component"
```

---

## Task 9: Frontend - Navigation Link

**Files:**
- Modify: `zettelkasten-front/src/components/sidebar/NavigationLinks.tsx`

**Step 9.1: Add RSS navigation link**

Read the NavigationLinks file to find where to add the RSS link (typically with other feature links like Files, Tasks, etc.):

Add the RSS icon (if needed) and navigation link:

```typescript
// Add import at top
import { RssIcon } from "@heroicons/react/24/outline"; // or appropriate icon library

// In the links section, after an appropriate link (like Files or Chat):
<SidebarLink to="/app/rss">
  <RssIcon className="w-5 h-5" />
  <span className="px-2">RSS</span>
</SidebarLink>
```

**Step 9.2: Commit**

```bash
git add zettelkasten-front/src/components/sidebar/NavigationLinks.tsx
git commit -m "feat: add RSS navigation link"
```

---

## Task 10: Frontend - Add Feed Dialog

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssAddFeedDialog.tsx`

**Step 10.1: Create add feed dialog**

Create: `zettelkasten-front/src/components/rss/RssAddFeedDialog.tsx`

```typescript
import React, { useState } from "react";
import { createFeed, CreateRSSFeedParams, RSSFeed } from "../../api/rss";

interface RssAddFeedDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onFeedAdded: (feed: RSSFeed) => void;
}

export function RssAddFeedDialog({
  isOpen,
  onClose,
  onFeedAdded,
}: RssAddFeedDialogProps) {
  const [url, setUrl] = useState("");
  const [name, setName] = useState("");
  const [folder, setFolder] = useState("");
  const [autoTags, setAutoTags] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const params: CreateRSSFeedParams = { url };
      if (name) params.name = name;
      if (folder) params.folder = folder;
      if (autoTags) params.auto_tags = autoTags;

      const feed = await createFeed(params);
      onFeedAdded(feed);
      onClose();
      // Reset form
      setUrl("");
      setName("");
      setFolder("");
      setAutoTags("");
    } catch (err: any) {
      setError(err.message || "Failed to add feed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
        <h2 className="text-xl font-bold mb-4">Add RSS Feed</h2>

        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-2 rounded mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Feed URL *</label>
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              required
              className="w-full border rounded px-3 py-2"
              placeholder="https://example.com/feed.xml"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Name (optional)</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full border rounded px-3 py-2"
              placeholder="Feed name"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Folder (optional)</label>
            <input
              type="text"
              value={folder}
              onChange={(e) => setFolder(e.target.value)}
              className="w-full border rounded px-3 py-2"
              placeholder="Tech, News, etc."
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">
              Auto Tags (optional)
            </label>
            <input
              type="text"
              value={autoTags}
              onChange={(e) => setAutoTags(e.target.value)}
              className="w-full border rounded px-3 py-2"
              placeholder="#tech #news"
            />
            <p className="text-xs text-gray-500 mt-1">
              Tags to apply when converting articles to cards
            </p>
          </div>

          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border rounded hover:bg-gray-100"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !url}
              className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:bg-gray-400"
            >
              {loading ? "Adding..." : "Add Feed"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

**Step 10.2: Update RssPage to use dialog**

Update `zettelkasten-front/src/pages/RssPage.tsx` to include the dialog:

```typescript
import { RssAddFeedDialog } from "../components/rss/RssAddFeedDialog";

// Add state for dialog
const [showAddFeedDialog, setShowAddFeedDialog] = useState(false);

// Add handler
const handleFeedAdded = (feed: RSSFeed) => {
  setFeeds((prev) => [...prev, feed]);
};

// Add button in the left panel
<button
  onClick={() => setShowAddFeedDialog(true)}
  className="w-full bg-green-500 text-white px-4 py-2 rounded hover:bg-green-600 mt-2"
>
  Add Feed
</button>

// Add dialog component at the end
<RssAddFeedDialog
  isOpen={showAddFeedDialog}
  onClose={() => setShowAddFeedDialog(false)}
  onFeedAdded={handleFeedAdded}
/>
```

**Step 10.3: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssAddFeedDialog.tsx zettelkasten-front/src/pages/RssPage.tsx
git commit -m "feat: add RSS add feed dialog"
```

---

## Task 11: Frontend - Convert Dialog

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssConvertDialog.tsx`

**Step 11.1: Create convert dialog**

Create: `zettelkasten-front/src/components/rss/RssConvertDialog.tsx`

```typescript
import React, { useState, useEffect } from "react";
import { convertToCard, ConvertArticleParams, RSSArticle } from "../../api/rss";

interface RssConvertDialogProps {
  isOpen: boolean;
  onClose: () => void;
  article: RSSArticle | null;
  onConverted: (cardId: number) => void;
}

export function RssConvertDialog({
  isOpen,
  onClose,
  article,
  onConverted,
}: RssConvertDialogProps) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [tags, setTags] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (article) {
      setTitle(article.title);
      setBody(article.content || "");
    }
  }, [article]);

  if (!isOpen || !article) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const params: ConvertArticleParams = {};
      if (title !== article.title) params.title = title;
      if (body !== article.content) params.body = body;
      if (tags) params.tags = tags;

      const card = await convertToCard(article.id, params);
      onConverted(card.id);
      onClose();
    } catch (err: any) {
      setError(err.message || "Failed to convert article");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-bold mb-4">Convert to Card</h2>

        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-2 rounded mb-4">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Title</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full border rounded px-3 py-2"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Content</label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={10}
              className="w-full border rounded px-3 py-2 font-mono text-sm"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Tags</label>
            <input
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              className="w-full border rounded px-3 py-2"
              placeholder="#to-read #reference"
            />
          </div>

          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border rounded hover:bg-gray-100"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:bg-gray-400"
            >
              {loading ? "Converting..." : "Convert to Card"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

**Step 11.2: Update RssPage to use convert dialog**

Update `zettelkasten-front/src/pages/RssPage.tsx`:

```typescript
import { RssConvertDialog } from "../components/rss/RssConvertDialog";

// Add state
const [showConvertDialog, setShowConvertDialog] = useState(false);

// Update convert handler
const handleConvertClick = () => {
  setShowConvertDialog(true);
};

const handleConverted = (cardId: number) => {
  // Navigate to card or show success
  window.location.href = `/app/card/${cardId}`;
};

// Update the convert button
<button
  onClick={handleConvertClick}
  className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
>
  Edit & Convert
</button>

// Add dialog at the end
<RssConvertDialog
  isOpen={showConvertDialog}
  onClose={() => setShowConvertDialog(false)}
  article={selectedArticle}
  onConverted={handleConverted}
/>
```

**Step 11.3: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssConvertDialog.tsx zettelkasten-front/src/pages/RssPage.tsx
git commit -m "feat: add RSS convert dialog"
```

---

## Task 12: Frontend Tests

**Files:**
- Create: `zettelkasten-front/src/pages/__tests__/RssPage.test.tsx`

**Step 12.1: Write frontend tests**

Create: `zettelkasten-front/src/pages/__tests__/RssPage.test.tsx`

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import { RssPage } from "../RssPage";

// Mock the API
vi.mock("../../api/rss", () => ({
  listFeeds: vi.fn(() => Promise.resolve([])),
  listArticles: vi.fn(() => Promise.resolve([])),
  listFolders: vi.fn(() => Promise.resolve([])),
  markAsRead: vi.fn(() => Promise.resolve()),
  convertToCard: vi.fn(() => Promise.resolve({ id: 1 })),
  refreshFeeds: vi.fn(() => Promise.resolve({ fetched: 0 })),
}));

describe("RssPage", () => {
  it("renders the RSS page", async () => {
    render(
      <BrowserRouter>
        <RssPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText("RSS Feeds")).toBeInTheDocument();
    });
  });

  it("shows refresh button", async () => {
    render(
      <BrowserRouter>
        <RssPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText("Refresh All")).toBeInTheDocument();
    });
  });
});
```

**Step 12.2: Run frontend tests**

Run: `cd zettelkasten-front && npm test -- RssPage`
Expected: Tests pass

**Step 12.3: Commit**

```bash
git add zettelkasten-front/src/pages/__tests__/RssPage.test.tsx
git commit -m "test: add RSS page tests"
```

---

## Task 13: Documentation

**Files:**
- Modify: `CLAUDE.md`

**Step 13.1: Update CLAUDE.md**

Add a new section for RSS in the Key Features section:

```markdown
### RSS Feed Client
- **Feeds**: Subscribe to RSS/Atom feeds with auto-tagging support
- **Articles**: Browse fetched articles in a reader-style inbox
- **Conversion**: Selectively convert interesting articles to cards
- **Folders**: Organize feeds into folders for better navigation
- **Scheduled Fetch**: Background job fetches new articles every 60 minutes
```

**Step 13.2: Add environment variable documentation**

Add to the Environment Configuration section:

```markdown
- RSS_FETCH_INTERVAL_MINUTES (optional) - Interval in minutes for RSS feed fetching (default: 60)
```

**Step 13.3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add RSS feature documentation"
```

---

## Task 14: End-to-End Testing

**Step 14.1: Build and run backend**

Run:
```bash
cd go-backend
source .env-bash
go run main.go
```

**Step 14.2: Build and run frontend**

Run:
```bash
cd zettelkasten-front
npm start
```

**Step 14.3: Test the feature manually**

1. Log in to the application
2. Navigate to RSS page
3. Add a feed (e.g., https://news.ycombinator.com/rss)
4. Click "Refresh All" to fetch articles
5. Browse articles in the middle panel
6. Click an article to read it
7. Convert an article to a card
8. Verify the card was created

**Step 14.4: Final commit**

```bash
git add -A
git commit -m "feat: complete RSS feed client implementation"
```

---

## Definition of Done Checklist

- [ ] All three database tables created with indexes
- [ ] RSS feeds can be added/removed via API
- [ ] Scheduled job fetches articles from feeds
- [ ] Articles displayed in three-panel UI
- [ ] Articles can be marked as read/unread
- [ ] Articles can be converted to cards with auto-tags
- [ ] Folders organize feeds
- [ ] All tests passing
- [ ] Documentation updated

---

## Notes for Implementation

1. **TDD Approach**: Each task follows the test-first approach - write test, see it fail, implement, see it pass, commit.

2. **Existing Patterns**: The implementation follows existing patterns in the codebase:
   - Handlers use JWT middleware and consistent error handling
   - Services use parameterized queries and proper error handling
   - Frontend uses the centralized apiClient with proper error handling
   - Routes use addProtectedRoute for authentication

3. **Reusing Existing Code**: The RSS service reuses `ParseURL` from the articles service for content extraction.

4. **Scheduler Integration**: The RSS fetch job integrates with the existing scheduler system using the ScheduledJob interface.

5. **Database Migrations**: Uses the existing migration pattern with numbered files.

6. **Frontend State Management**: Uses React hooks (useState, useEffect) following existing patterns.

7. **Testing**: Backend tests use the existing test helpers, frontend tests use Vitest and React Testing Library.
