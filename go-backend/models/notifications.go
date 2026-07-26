package models

import (
	"database/sql"
	"strconv"

	"github.com/lib/pq"
	"time"
)

// Source type constants
const (
	SourceTypeRSS  = "rss"
	SourceTypeTask = "task"
)

// Notification represents a unified notification from email, RSS, or tasks
type Notification struct {
	ID              int            `json:"id"`
	UserID          int            `json:"user_id"`
	SourceType      string         `json:"source_type"`       // "rss", "task"
	SourceID        int            `json:"source_id"`         // ID of the source record
	Title           string         `json:"title"`             // Email subject, article title, or task title
	Preview         *string        `json:"preview,omitempty"` // Preview text (email body, article content, task notes)
	Timestamp       time.Time      `json:"timestamp"`         // When the source item was created/received
	ImportanceScore int            `json:"importance_score"`  // Computed score for sorting
	IsRead          bool           `json:"is_read"`           // Whether the user has read this notification
	IsArchived      bool           `json:"is_archived"`       // Whether the user has archived this notification
	FilterTags      pq.StringArray `json:"filter_tags"`       // Tags for filtering (e.g., ["unprocessed", "starred", "priority"])
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// NotificationPreferences represents user preferences for the unified inbox
type NotificationPreferences struct {
	UserID                int       `json:"user_id"`
	ShowUnprocessedEmails bool      `json:"show_unprocessed_emails"`
	ShowStarredArticles   bool      `json:"show_starred_articles"`
	ShowPriorityTasks     bool      `json:"show_priority_tasks"`
	ShowPriorityFeeds     bool      `json:"show_priority_feeds"`
	ItemsPerPage          int       `json:"items_per_page"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// NotificationListFilters represents filters for listing notifications
type NotificationListFilters struct {
	SourceType *string `json:"source_type,omitempty"` // Filter by source type
	IsRead     *bool   `json:"is_read,omitempty"`     // Filter by read status
	IsArchived *bool   `json:"is_archived,omitempty"` // Filter by archived status
	Limit      *int    `json:"limit,omitempty"`       // Max results to return
	Offset     *int    `json:"offset,omitempty"`      // Results offset for pagination
}

// CalculateImportanceScore computes the importance score for a notification
// Higher scores indicate more important items
// Scores:
// - 10: Starred articles
// - 5: Priority feed articles, Priority tasks
// - 0: Everything else
func CalculateImportanceScore(sourceType string, isStarred, isPriorityFeed bool) int {
	switch sourceType {
	case SourceTypeRSS:
		if isStarred {
			return 10
		}
		if isPriorityFeed {
			return 5
		}
		return 0
	case SourceTypeTask:
		if isPriorityFeed {
			return 5
		}
		return 0
	default:
		return 0
	}
}

// GetFilterTagsForRSS generates filter tags for RSS article notifications
func GetFilterTagsForRSS(isStarred, isPriorityFeed bool, folder string, feedName string) pq.StringArray {
	tags := pq.StringArray{"rss"}

	if isStarred {
		tags = append(tags, "starred")
	}

	if isPriorityFeed {
		tags = append(tags, "priority")
	}

	if folder != "" {
		tags = append(tags, folder)
	}

	if feedName != "" {
		tags = append(tags, feedName)
	}

	return tags
}

// GetFilterTagsForTask generates filter tags for task notifications
func GetFilterTagsForTask(isPriority bool, tags string) pq.StringArray {
	filterTags := pq.StringArray{"task"}

	if isPriority {
		filterTags = append(filterTags, "priority")
	}

	// Parse task tags and add them as filter tags
	// This is a simple implementation - could be more sophisticated
	if tags != "" {
		// For now, just add the raw tags string as one tag
		// In production, you might want to parse comma-separated tags
		filterTags = append(filterTags, tags)
	}

	return filterTags
}

// CreateNotification creates or updates a notification in the unified inbox
func CreateNotification(db Database, userID int, sourceType string, sourceID int, title string, preview *string, timestamp time.Time, importanceScore int, filterTags pq.StringArray) (*Notification, error) {
	query := `
		INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, source_type, source_id)
		DO UPDATE SET
			title = EXCLUDED.title,
			preview = EXCLUDED.preview,
			timestamp = EXCLUDED.timestamp,
			importance_score = EXCLUDED.importance_score,
			filter_tags = EXCLUDED.filter_tags,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, user_id, source_type, source_id, title, preview, timestamp,
			importance_score, is_read, is_archived, filter_tags, created_at, updated_at
	`

	var notification Notification
	err := db.QueryRow(query, userID, sourceType, sourceID, title, preview, timestamp, importanceScore, filterTags).Scan(
		&notification.ID,
		&notification.UserID,
		&notification.SourceType,
		&notification.SourceID,
		&notification.Title,
		&notification.Preview,
		&notification.Timestamp,
		&notification.ImportanceScore,
		&notification.IsRead,
		&notification.IsArchived,
		&notification.FilterTags,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &notification, nil
}

// GetNotificationsByUser retrieves notifications for a user with optional filters
func GetNotificationsByUser(db Database, userID int, filters NotificationListFilters) ([]Notification, error) {
	// Build the base query
	baseQuery := `
		SELECT id, user_id, source_type, source_id, title, preview, timestamp,
			importance_score, is_read, is_archived, filter_tags, created_at, updated_at
		FROM notifications
		WHERE user_id = $1
	`

	// Build the filter conditions
	var conditions []interface{}
	conditions = append(conditions, userID)
	argCount := 1

	if filters.SourceType != nil {
		argCount++
		baseQuery += " AND source_type = $" + strconv.Itoa(argCount)
		conditions = append(conditions, *filters.SourceType)
	}

	if filters.IsRead != nil {
		argCount++
		baseQuery += " AND is_read = $" + strconv.Itoa(argCount)
		conditions = append(conditions, *filters.IsRead)
	}

	if filters.IsArchived != nil {
		argCount++
		baseQuery += " AND is_archived = $" + strconv.Itoa(argCount)
		conditions = append(conditions, *filters.IsArchived)
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY importance_score DESC, timestamp DESC"

	if filters.Limit != nil {
		argCount++
		baseQuery += " LIMIT $" + strconv.Itoa(argCount)
		conditions = append(conditions, *filters.Limit)
	}

	if filters.Offset != nil {
		argCount++
		baseQuery += " OFFSET $" + strconv.Itoa(argCount)
		conditions = append(conditions, *filters.Offset)
	}

	// Execute the query
	rows, err := db.Query(baseQuery, conditions...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan the results
	var notifications []Notification
	for rows.Next() {
		var notification Notification
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.SourceType,
			&notification.SourceID,
			&notification.Title,
			&notification.Preview,
			&notification.Timestamp,
			&notification.ImportanceScore,
			&notification.IsRead,
			&notification.IsArchived,
			&notification.FilterTags,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return notifications, nil
}

// GetUnreadCount retrieves the count of unread, unarchived notifications for a user
func GetUnreadCount(db Database, userID int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1 AND is_read = FALSE AND is_archived = FALSE
	`

	var count int
	err := db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// MarkNotificationAsRead marks a notification as read
func MarkNotificationAsRead(db Database, notificationID, userID int) error {
	query := `
		UPDATE notifications
		SET is_read = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
	`

	_, err := db.Exec(query, notificationID, userID)
	return err
}

// MarkNotificationAsArchived marks a notification as archived
func MarkNotificationAsArchived(db Database, notificationID, userID int) error {
	query := `
		UPDATE notifications
		SET is_archived = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
	`

	_, err := db.Exec(query, notificationID, userID)
	return err
}

// MarkAllAsRead marks all notifications for a user as read
func MarkAllAsRead(db Database, userID int) error {
	query := `
		UPDATE notifications
		SET is_read = TRUE, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND is_read = FALSE
	`

	_, err := db.Exec(query, userID)
	return err
}

// GetNotificationPreferences retrieves notification preferences for a user
func GetNotificationPreferences(db Database, userID int) (*NotificationPreferences, error) {
	query := `
		SELECT user_id, show_unprocessed_emails, show_starred_articles,
			show_priority_tasks, show_priority_feeds, items_per_page, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`

	var prefs NotificationPreferences
	err := db.QueryRow(query, userID).Scan(
		&prefs.UserID,
		&prefs.ShowUnprocessedEmails,
		&prefs.ShowStarredArticles,
		&prefs.ShowPriorityTasks,
		&prefs.ShowPriorityFeeds,
		&prefs.ItemsPerPage,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &prefs, nil
}

// UpdateNotificationPreferences upserts notification preferences for a user.
//
// This is an INSERT ... ON CONFLICT DO UPDATE (not a plain UPDATE) because the
// row may not exist yet — there is no DB trigger that auto-creates it
// (migration 0121 only backfills rows once at migration time, so users added
// later — including all test users — have no prefs row until first write).
// A plain UPDATE would silently affect 0 rows and the subsequent read would
// return "no rows". The upsert works identically on Postgres and SQLite.
func UpdateNotificationPreferences(db Database, userID int, prefs NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences
			(user_id, show_unprocessed_emails, show_starred_articles,
			 show_priority_tasks, show_priority_feeds, items_per_page,
			 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			show_unprocessed_emails = $2,
			show_starred_articles = $3,
			show_priority_tasks = $4,
			show_priority_feeds = $5,
			items_per_page = $6,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.Exec(query,
		userID,
		prefs.ShowUnprocessedEmails,
		prefs.ShowStarredArticles,
		prefs.ShowPriorityTasks,
		prefs.ShowPriorityFeeds,
		prefs.ItemsPerPage,
	)

	return err
}

// DeleteNotification removes a notification from the unified inbox
func DeleteNotification(db Database, notificationID, userID int) error {
	query := `
		DELETE FROM notifications
		WHERE id = $1 AND user_id = $2
	`

	_, err := db.Exec(query, notificationID, userID)
	return err
}

// DeleteNotificationBySource removes the notification for a given source
// (user_id + source_type + source_id). Mirrors the Postgres delete_notification()
// trigger helper (schema/0122) used by the rss/email AFTER DELETE triggers,
// now ported to Go (Phase 5). Runs on both drivers.
func DeleteNotificationBySource(db Database, userID int, sourceType string, sourceID int) error {
	_, err := db.Exec(
		`DELETE FROM notifications WHERE user_id = $1 AND source_type = $2 AND source_id = $3`,
		userID, sourceType, sourceID,
	)
	return err
}

// GetNotificationByID retrieves a specific notification by ID
func GetNotificationByID(db Database, notificationID, userID int) (*Notification, error) {
	query := `
		SELECT id, user_id, source_type, source_id, title, preview, timestamp,
			importance_score, is_read, is_archived, filter_tags, created_at, updated_at
		FROM notifications
		WHERE id = $1 AND user_id = $2
	`

	var notification Notification
	err := db.QueryRow(query, notificationID, userID).Scan(
		&notification.ID,
		&notification.UserID,
		&notification.SourceType,
		&notification.SourceID,
		&notification.Title,
		&notification.Preview,
		&notification.Timestamp,
		&notification.ImportanceScore,
		&notification.IsRead,
		&notification.IsArchived,
		&notification.FilterTags,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &notification, nil
}
