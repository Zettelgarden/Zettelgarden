package services

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"go-backend/models"
)

const (
	// DefaultFetchInterval is the default RSS feed fetch interval in minutes
	DefaultFetchInterval = 60
	// RSSTimeout is the HTTP timeout for RSS feed fetching
	RSSTimeout = 30 * time.Second
	// DefaultArticleLimit is the default number of articles to return
	DefaultArticleLimit = 100
)

// CreateRSSFeed creates a new RSS feed for a user
func CreateRSSFeed(db models.Database, userID int, params models.CreateRSSFeedParams) (*models.RSSFeed, error) {
	// Validate URL by attempting to parse it
	fp := gofeed.NewParser()
	fp.Client = &http.Client{Timeout: RSSTimeout}
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
	fetchInterval := DefaultFetchInterval
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
func GetRSSFeedByID(db models.Database, userID, feedID int) (*models.RSSFeed, error) {
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
func ListRSSFeeds(db models.Database, userID int) ([]models.RSSFeed, error) {
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feeds: %w", err)
	}

	return feeds, nil
}

// UpdateRSSFeed updates an existing RSS feed
func UpdateRSSFeed(db models.Database, userID, feedID int, params models.UpdateRSSFeedParams) (*models.RSSFeed, error) {
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
		strings.Join(updates, ", "), argPos, argPos+1)
	args = append(args, feedID, userID)

	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update feed: %w", err)
	}

	return GetRSSFeedByID(db, userID, feedID)
}

// DeleteRSSFeed deletes an RSS feed
func DeleteRSSFeed(db models.Database, userID, feedID int) error {
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
func ListRSSArticles(db models.Database, userID int, filters map[string]interface{}) ([]models.RSSArticle, error) {
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

	limit := DefaultArticleLimit
	if limitParam, ok := filters["limit"].(int); ok && limitParam > 0 {
		limit = limitParam
	}
	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, limit)

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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating articles: %w", err)
	}

	return articles, nil
}

// GetRSSArticleByID retrieves a single RSS article
func GetRSSArticleByID(db models.Database, userID, articleID int) (*models.RSSArticle, error) {
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
func MarkRSSArticleAsRead(db models.Database, userID, articleID int, read bool) error {
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
func ConvertRSSArticleToCard(db models.Database, userID, articleID int, params *models.ConvertArticleParams) (*models.Card, error) {
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
		Link:  article.URL,
	}

	card, err := CreateCard(db, userID, cardParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %w", err)
	}

	// Mark article as read
	if err := MarkRSSArticleAsRead(db, userID, articleID, true); err != nil {
		log.Printf("[rss-article:%d] failed to mark as read after conversion: %v", articleID, err)
	}

	return &card, nil
}

// FetchRSSFeedArticles fetches new articles from an RSS feed
func FetchRSSFeedArticles(db models.Database, feedID int) error {
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
	fp.Client = &http.Client{Timeout: RSSTimeout}
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
		if err != nil {
			log.Printf("[rss-feed:%d] failed to check article existence: %v", feedID, err)
			continue
		}
		if exists {
			continue
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
func ListRSSFolders(db models.Database, userID int) ([]models.RSSFolder, error) {
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating folders: %w", err)
	}

	return folders, nil
}

// CreateRSSFolder creates a new folder
func CreateRSSFolder(db models.Database, userID int, params models.CreateRSSFolderParams) (*models.RSSFolder, error) {
	// Set default order_index if not provided
	orderIndex := 0
	if params.OrderIndex != nil {
		orderIndex = *params.OrderIndex
	}

	var folderID int
	err := db.QueryRow(`
		INSERT INTO rss_folders (user_id, name, order_index)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, params.Name, orderIndex).Scan(&folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	return GetRSSFolderByID(db, userID, folderID)
}

// GetRSSFolderByID retrieves a folder by ID
func GetRSSFolderByID(db models.Database, userID, folderID int) (*models.RSSFolder, error) {
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

// UpdateRSSFolder updates an existing RSS folder
func UpdateRSSFolder(db models.Database, userID, folderID int, name *string, orderIndex *int) (*models.RSSFolder, error) {
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *name)
		argPos++
	}
	if orderIndex != nil {
		updates = append(updates, fmt.Sprintf("order_index = $%d", argPos))
		args = append(args, *orderIndex)
		argPos++
	}

	if len(updates) == 0 {
		return GetRSSFolderByID(db, userID, folderID)
	}

	query := fmt.Sprintf("UPDATE rss_folders SET %s WHERE id = $%d AND user_id = $%d",
		strings.Join(updates, ", "), argPos, argPos+1)
	args = append(args, folderID, userID)

	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update folder: %w", err)
	}

	return GetRSSFolderByID(db, userID, folderID)
}

// DeleteRSSFolder deletes an RSS folder
func DeleteRSSFolder(db models.Database, userID, folderID int) error {
	result, err := db.Exec("DELETE FROM rss_folders WHERE id = $1 AND user_id = $2", folderID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("folder not found")
	}

	return nil
}
