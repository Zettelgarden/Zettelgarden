> **STATUS: HISTORICAL — pre-SQLite era.** This plan predates the PostgreSQL→SQLite cutover (2026-07-28, epic Zettelgarden-c7j) and the move to local on-disk file storage (epic Zettelgarden-yar). Zettelgarden now runs SQLite-only with local storage; this document is kept for design history.

# Unified Notification Inbox Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a unified notification inbox that consolidates important items from email and RSS feeds into a single filtered view with user-customizable toggles.

**Architecture:** Create a new `notifications` table that aggregates items from email and RSS sources via database triggers. Backend provides a unified API endpoint that applies user preference filters. Frontend displays notifications in a tabbed interface with source indicators.

**Tech Stack:** Go (backend), React/TypeScript (frontend), PostgreSQL (database with triggers)

---

## Task 1: Create notifications table migration

**Files:**
- Create: `go-backend/schema/0120-add-notifications-table.sql`

**Step 1: Write the migration file**

```sql
-- Notifications table: unified view of important items from email, RSS, tasks
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    source_type VARCHAR(20) NOT NULL,  -- 'email', 'rss', 'task'
    source_id INTEGER NOT NULL,         -- foreign key to source table
    title TEXT NOT NULL,
    preview TEXT,                       -- brief content preview (truncated content)
    timestamp TIMESTAMPTZ NOT NULL,     -- normalized timestamp from source
    importance_score INTEGER DEFAULT 0, -- computed score for sorting
    is_read BOOLEAN DEFAULT FALSE,
    is_archived BOOLEAN DEFAULT FALSE,
    filter_tags TEXT[],                 -- e.g., '{priority,unprocessed,starred}'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, source_type, source_id)
);

-- Index for timestamp-based queries (main list view)
CREATE INDEX idx_notifications_user_timestamp ON notifications(user_id, timestamp DESC);

-- Index for unread queries (badge counts)
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read, is_archived)
    WHERE is_read = FALSE AND is_archived = FALSE;

-- Index for filter tag queries (user preferences)
CREATE INDEX idx_notifications_filter_tags ON notifications USING GIN(filter_tags);
```

**Step 2: Verify the migration file exists**

Run: `ls -la go-backend/schema/0120-add-notifications-table.sql`
Expected: File exists with content above

**Step 3: Commit**

```bash
git add go-backend/schema/0120-add-notifications-table.sql
git commit -m "schema: add notifications table for unified inbox"
```

---

## Task 2: Create notification_preferences table migration

**Files:**
- Create: `go-backend/schema/0121-add-notification-preferences-table.sql`

**Step 1: Write the migration file**

```sql
-- User preferences for notification filtering
CREATE TABLE notification_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    -- Filter toggles - what to include in unified inbox
    show_unprocessed_emails BOOLEAN DEFAULT TRUE,
    show_starred_articles BOOLEAN DEFAULT TRUE,
    show_priority_tasks BOOLEAN DEFAULT TRUE,
    show_priority_feeds BOOLEAN DEFAULT TRUE,
    -- Display settings
    items_per_page INTEGER DEFAULT 50,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create default preferences for existing users
INSERT INTO notification_preferences (user_id)
SELECT id FROM users
ON CONFLICT (user_id) DO NOTHING;
```

**Step 2: Verify the migration file exists**

Run: `ls -la go-backend/schema/0121-add-notification-preferences-table.sql`
Expected: File exists with content above

**Step 3: Commit**

```bash
git add go-backend/schema/0121-add-notification-preferences-table.sql
git commit -m "schema: add notification_preferences table for user filter settings"
```

---

## Task 3: Create notification model and types

**Files:**
- Create: `go-backend/models/notifications.go`
- Create: `go-backend/models/notifications_test.go`

**Step 1: Write the failing test first**

```go
package models

import (
    "testing"
    "time"
)

func TestNotificationImportanceScore(t *testing.T) {
    tests := []struct {
        name           string
        sourceType     string
        isUnprocessed  bool
        isStarred      bool
        isPriorityFeed bool
        expectedScore  int
    }{
        {
            name:           "unprocessed email",
            sourceType:     "email",
            isUnprocessed:  true,
            expectedScore:  10,
        },
        {
            name:           "triaged email",
            sourceType:     "email",
            isUnprocessed:  false,
            expectedScore:  5,
        },
        {
            name:           "starred article",
            sourceType:     "rss",
            isStarred:      true,
            expectedScore:  10,
        },
        {
            name:           "priority feed article",
            sourceType:     "rss",
            isPriorityFeed: true,
            expectedScore:  5,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            score := CalculateImportanceScore(tt.sourceType, tt.isUnprocessed, tt.isStarred, tt.isPriorityFeed)
            if score != tt.expectedScore {
                t.Errorf("CalculateImportanceScore() = %d, want %d", score, tt.expectedScore)
            }
        })
    }
}

func TestFilterTagsForEmail(t *testing.T) {
    tags := GetFilterTagsForEmail("unprocessed", "test@example.com")
    if len(tags) == 0 {
        t.Error("Expected at least one filter tag")
    }

    // Check that status is included
    hasStatusTag := false
    for _, tag := range tags {
        if tag == "unprocessed" {
            hasStatusTag = true
        }
    }
    if !hasStatusTag {
        t.Error("Expected 'unprocessed' tag in filter tags")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./models -run TestNotification -v`
Expected: FAIL with "undefined: CalculateImportanceScore"

**Step 3: Write the implementation**

```go
package models

import (
    "database/sql"
    "fmt"
)

// Notification represents a unified notification item
type Notification struct {
    ID              int
    UserID          int
    SourceType      string // 'email', 'rss', 'task'
    SourceID        int
    Title           string
    Preview         string
    Timestamp       string
    ImportanceScore int
    IsRead          bool
    IsArchived      bool
    FilterTags      []string
}

// NotificationPreferences represents user filter settings
type NotificationPreferences struct {
    UserID                int
    ShowUnprocessedEmails bool
    ShowStarredArticles   bool
    ShowPriorityTasks     bool
    ShowPriorityFeeds     bool
    ItemsPerPage          int
}

// CalculateImportanceScore computes the importance score for a notification
func CalculateImportanceScore(sourceType string, isUnprocessed, isStarred, isPriorityFeed bool) int {
    switch sourceType {
    case "email":
        if isUnprocessed {
            return 10
        }
        return 5 // triaged
    case "rss":
        if isStarred {
            return 10
        }
        if isPriorityFeed {
            return 5
        }
        return 0
    case "task":
        // Future: implement task scoring
        return 0
    default:
        return 0
    }
}

// GetFilterTagsForEmail generates filter tags for email notifications
func GetFilterTagsForEmail(status, fromAddress string) []string {
    tags := []string{status}
    // Future: add contact-based tags
    return tags
}

// GetFilterTagsForRSS generates filter tags for RSS article notifications
func GetFilterTagsForRSS(isStarred, isPriorityFeed bool) []string {
    tags := []string{}
    if isStarred {
        tags = append(tags, "starred")
    }
    if isPriorityFeed {
        tags = append(tags, "priority")
    }
    return tags
}

// CreateNotification creates a new notification in the database
func CreateNotification(db *sql.DB, n *Notification) error {
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
            updated_at = NOW()
        RETURNING id, created_at, updated_at
    `
    err := db.QueryRow(
        query,
        n.UserID, n.SourceType, n.SourceID, n.Title, n.Preview, n.Timestamp, n.ImportanceScore, n.FilterTags,
    ).Scan(&n.ID, new(string), new(string))
    return err
}

// GetNotificationsByUser retrieves notifications for a user with filters
func GetNotificationsByUser(db *sql.DB, userID int, sourceType string, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
    // Build WHERE clause
    whereClause := "WHERE user_id = $1 AND is_archived = FALSE"
    args := []interface{}{userID}
    argPos := 2

    if sourceType != "" {
        whereClause += fmt.Sprintf(" AND source_type = $%d", argPos)
        args = append(args, sourceType)
        argPos++
    }

    if unreadOnly {
        whereClause += fmt.Sprintf(" AND is_read = FALSE", argPos)
    }

    // Get count
    countQuery := "SELECT COUNT(*) FROM notifications " + whereClause
    var total int
    if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
        return nil, 0, err
    }

    // Get notifications
    query := `
        SELECT id, user_id, source_type, source_id, title, preview, timestamp,
               importance_score, is_read, is_archived, filter_tags
        FROM notifications
        ` + whereClause + `
        ORDER BY importance_score DESC, timestamp DESC
        LIMIT $` + fmt.Sprintf("%d", argPos) + ` OFFSET $` + fmt.Sprintf("%d", argPos+1)
    args = append(args, limit, offset)

    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()

    var notifications []Notification
    for rows.Next() {
        var n Notification
        if err := rows.Scan(
            &n.ID, &n.UserID, &n.SourceType, &n.SourceID, &n.Title, &n.Preview,
            &n.Timestamp, &n.ImportanceScore, &n.IsRead, &n.IsArchived, &n.FilterTags,
        ); err != nil {
            return nil, 0, err
        }
        notifications = append(notifications, n)
    }

    return notifications, total, nil
}

// GetUnreadCount returns the count of unread notifications for a user
func GetUnreadCount(db *sql.DB, userID int) (int, error) {
    var count int
    err := db.QueryRow(
        "SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE AND is_archived = FALSE",
        userID,
    ).Scan(&count)
    return count, err
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./models -run TestNotification -v`
Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/models/notifications.go go-backend/models/notifications_test.go
git commit -m "feat: add notification model with importance scoring"
```

---

## Task 4: Create database trigger for email notifications sync

**Files:**
- Create: `go-backend/schema/0122-add-email-notification-trigger.sql`

**Step 1: Write the trigger migration**

```sql
-- Function to sync email to notifications table
CREATE OR REPLACE FUNCTION sync_email_notification()
RETURNS TRIGGER AS $$
DECLARE
    notification_title TEXT;
    notification_preview TEXT;
    notification_importance INT;
    notification_tags TEXT[];
BEGIN
    -- Only create notifications for unprocessed and triaged emails (not archived)
    IF NEW.status NOT IN ('unprocessed', 'triaged') THEN
        -- If archived, delete the notification
        IF NEW.status = 'archived' THEN
            DELETE FROM notifications WHERE user_id = NEW.user_id AND source_type = 'email' AND source_id = NEW.id;
            RETURN NEW;
        END IF;
        RETURN NEW;
    END IF;

    -- Build notification title and preview
    notification_title := COALESCE(NEW.subject, '(No subject)');
    notification_preview := LEFT(COALESCE(NEW.body_text, ''), 200);

    -- Calculate importance score
    IF NEW.status = 'unprocessed' THEN
        notification_importance := 10;
    ELSE
        notification_importance := 5;  -- triaged
    END IF;

    -- Build filter tags
    notification_tags := ARRAY[NEW.status];

    -- Insert or update notification
    INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
    VALUES (NEW.user_id, 'email', NEW.id, notification_title, notification_preview,
            COALESCE(NEW.received_at, NEW.created_at), notification_importance, notification_tags)
    ON CONFLICT (user_id, source_type, source_id)
    DO UPDATE SET
        title = EXCLUDED.title,
        preview = EXCLUDED.preview,
        timestamp = EXCLUDED.timestamp,
        importance_score = EXCLUDED.importance_score,
        filter_tags = EXCLUDED.filter_tags,
        updated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on emails table
DROP TRIGGER IF EXISTS email_notification_trigger ON emails;
CREATE TRIGGER email_notification_trigger
    AFTER INSERT OR UPDATE ON emails
    FOR EACH ROW
    EXECUTE FUNCTION sync_email_notification();

-- Trigger to delete notification when email is deleted
DROP TRIGGER IF EXISTS email_delete_notification_trigger ON emails;
CREATE TRIGGER email_delete_notification_trigger
    AFTER DELETE ON emails
    FOR EACH ROW
    EXECUTE FUNCTION delete_notification();
```

**Step 2: Add helper delete function**

Create: `go-backend/schema/0123-add-notification-helper-functions.sql`

```sql
-- Helper function to delete notification when source is deleted
CREATE OR REPLACE FUNCTION delete_notification()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM notifications WHERE source_type = TG_TABLE_NAME AND source_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
```

**Step 3: Verify migration files exist**

Run: `ls -la go-backend/schema/0122-*.sql go-backend/schema/0123-*.sql`
Expected: Both files exist with content above

**Step 4: Commit**

```bash
git add go-backend/schema/0122-add-email-notification-trigger.sql go-backend/schema/0123-add-notification-helper-functions.sql
git commit -m "schema: add email to notification sync trigger"
```

---

## Task 5: Create database trigger for RSS article notifications sync

**Files:**
- Create: `go-backend/schema/0124-add-rss-notification-trigger.sql`

**Step 1: Write the trigger migration**

```sql
-- Function to sync RSS articles to notifications table
CREATE OR REPLACE FUNCTION sync_rss_notification()
RETURNS TRIGGER AS $$
DECLARE
    feed_record RECORD;
    notification_title TEXT;
    notification_preview TEXT;
    notification_importance INT;
    notification_tags TEXT[];
BEGIN
    -- Only create notifications for starred articles or priority feed articles
    SELECT * INTO feed_record FROM rss_feeds WHERE id = NEW.feed_id;

    IF NOT (NEW.is_starred = TRUE OR (feed_record.priority = TRUE AND NEW.read = FALSE)) THEN
        -- If no longer starred, delete the notification
        IF NEW.is_starred = FALSE THEN
            DELETE FROM notifications WHERE user_id = NEW.user_id AND source_type = 'rss' AND source_id = NEW.id;
        END IF;
        RETURN NEW;
    END IF;

    -- Build notification title and preview
    notification_title := NEW.title;
    notification_preview := LEFT(COALESCE(NEW.content, ''), 200);

    -- Calculate importance score
    IF NEW.is_starred = TRUE THEN
        notification_importance := 10;
    ELSIF feed_record.priority = TRUE THEN
        notification_importance := 5;
    ELSE
        notification_importance := 0;
    END IF;

    -- Build filter tags
    notification_tags := ARRAY[]::TEXT[];
    IF NEW.is_starred = TRUE THEN
        notification_tags := array_append(notification_tags, 'starred');
    END IF;
    IF feed_record.priority = TRUE THEN
        notification_tags := array_append(notification_tags, 'priority');
    END IF;

    -- Insert or update notification
    INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
    VALUES (NEW.user_id, 'rss', NEW.id, notification_title, notification_preview,
            COALESCE(NEW.published_at, NEW.fetched_at), notification_importance, notification_tags)
    ON CONFLICT (user_id, source_type, source_id)
    DO UPDATE SET
        title = EXCLUDED.title,
        preview = EXCLUDED.preview,
        timestamp = EXCLUDED.timestamp,
        importance_score = EXCLUDED.importance_score,
        filter_tags = EXCLUDED.filter_tags,
        updated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger on rss_articles table
DROP TRIGGER IF EXISTS rss_article_notification_trigger ON rss_articles;
CREATE TRIGGER rss_article_notification_trigger
    AFTER INSERT OR UPDATE ON rss_articles
    FOR EACH ROW
    EXECUTE FUNCTION sync_rss_notification();

-- Trigger to delete notification when article is deleted
DROP TRIGGER IF EXISTS rss_article_delete_notification_trigger ON rss_articles;
CREATE TRIGGER rss_article_delete_notification_trigger
    AFTER DELETE ON rss_articles
    FOR EACH ROW
    EXECUTE FUNCTION delete_notification();
```

**Step 2: Verify migration file exists**

Run: `ls -la go-backend/schema/0124-add-rss-notification-trigger.sql`
Expected: File exists with content above

**Step 3: Commit**

```bash
git add go-backend/schema/0124-add-rss-notification-trigger.sql
git commit -m "schema: add RSS article to notification sync trigger"
```

---

## Task 6: Create notifications API handler

**Files:**
- Create: `go-backend/handlers/notifications.go`
- Create: `go-backend/handlers/notifications_test.go`

**Step 1: Write the failing test**

```go
package handlers

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestListNotificationsHandler(t *testing.T) {
    // Setup test DB and handler
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    handler := &NotificationHandler{DB: db}

    // Create test user
    user := createTestUser(t, db)

    // Create test notifications
    createTestNotification(t, db, user.ID, "email", 1, "Test Email", "unprocessed")
    createTestNotification(t, db, user.ID, "rss", 1, "Test Article", "starred")

    // Test request
    req := httptest.NewRequest("GET", "/notifications?limit=10", nil)
    req.Header.Set("X-User-ID", string(rune(user.ID)))

    w := httptest.NewRecorder()
    handler.ListNotifications(w, req)

    // Assert response
    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.NewDecoder(w.Body).Decode(&response)
    assert.NoError(t, err)

    notifications := response["notifications"].([]interface{})
    assert.Greater(t, len(notifications), 0)
}

func TestMarkNotificationAsRead(t *testing.T) {
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    handler := &NotificationHandler{DB: db}
    user := createTestUser(t, db)
    notif := createTestNotification(t, db, user.ID, "email", 1, "Test", "unprocessed")

    body := map[string]bool{"read": true}
    jsonBody, _ := json.Marshal(body)

    req := httptest.NewRequest("PATCH", "/notifications/"+string(rune(notif.ID))+"/read", bytes.NewBuffer(jsonBody))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-User-ID", string(rune(user.ID)))

    w := httptest.NewRecorder()
    handler.MarkAsRead(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./handlers -run TestNotification -v`
Expected: FAIL with handler not defined

**Step 3: Write the implementation**

```go
package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "strings"

    "github.com/gorilla/mux"
    "zettelgarden/models"
)

type NotificationHandler struct {
    DB *sql.DB
}

type NotificationListRequest struct {
    SourceType string `json:"source_type"`
    UnreadOnly bool   `json:"unread_only"`
    Limit      int    `json:"limit"`
    Offset     int    `json:"offset"`
}

type NotificationListResponse struct {
    Notifications []models.Notification `json:"notifications"`
    Total         int                    `json:"total"`
    UnreadCount   int                    `json:"unread_count"`
}

type MarkReadRequest struct {
    Read bool `json:"read"`
}

// ListNotifications handles GET /notifications
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)

    // Parse query parameters
    sourceType := r.URL.Query().Get("source_type")
    unreadOnly := r.URL.Query().Get("unread_only") == "true"

    limit := 50
    if l := r.URL.Query().Get("limit"); l != "" {
        if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
            limit = parsed
        }
    }

    offset := 0
    if o := r.URL.Query().Get("offset"); o != "" {
        if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
            offset = parsed
        }
    }

    // Get user preferences for filtering
    prefs, err := models.GetNotificationPreferences(h.DB, userID)
    if err != nil {
        http.Error(w, "Failed to get preferences", http.StatusInternalServerError)
        return
    }

    // Apply user preference filters via a filter clause
    // For now, we'll get all and the trigger handles what gets inserted
    notifications, total, err := models.GetNotificationsByUser(h.DB, userID, sourceType, unreadOnly, limit, offset)
    if err != nil {
        http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
        return
    }

    // Get unread count
    unreadCount, err := models.GetUnreadCount(h.DB, userID)
    if err != nil {
        unreadCount = 0
    }

    respondJSON(w, NotificationListResponse{
        Notifications: notifications,
        Total:         total,
        UnreadCount:   unreadCount,
    })
}

// GetUnreadCount handles GET /notifications/unread-count
func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)

    count, err := models.GetUnreadCount(h.DB, userID)
    if err != nil {
        http.Error(w, "Failed to get unread count", http.StatusInternalServerError)
        return
    }

    respondJSON(w, map[string]int{"unread_count": count})
}

// MarkAsRead handles PATCH /notifications/{id}/read
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)

    vars := mux.Vars(r)
    idStr := vars["id"]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid notification ID", http.StatusBadRequest)
        return
    }

    var req MarkReadRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if err := models.MarkNotificationAsRead(h.DB, id, userID, req.Read); err != nil {
        http.Error(w, "Failed to update notification", http.StatusInternalServerError)
        return
    }

    respondJSON(w, map[string]string{"status": "ok"})
}

// ArchiveNotification handles PATCH /notifications/{id}/archive
func (h *NotificationHandler) ArchiveNotification(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)

    vars := mux.Vars(r)
    idStr := vars["id"]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid notification ID", http.StatusBadRequest)
        return
    }

    if err := models.ArchiveNotification(h.DB, id, userID); err != nil {
        http.Error(w, "Failed to archive notification", http.StatusInternalServerError)
        return
    }

    respondJSON(w, map[string]string{"status": "ok"})
}

// GetPreferences handles GET /notifications/preferences
func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)

    prefs, err := models.GetNotificationPreferences(h.DB, userID)
    if err != nil {
        http.Error(w, "Failed to get preferences", http.StatusInternalServerError)
        return
    }

    respondJSON(w, prefs)
}

// UpdatePreferences handles PATCH /notifications/preferences
func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)

    var prefs models.NotificationPreferences
    if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    prefs.UserID = userID
    if err := models.UpdateNotificationPreferences(h.DB, &prefs); err != nil {
        http.Error(w, "Failed to update preferences", http.StatusInternalServerError)
        return
    }

    respondJSON(w, prefs)
}

// Helper: get user ID from request context (reuse existing auth middleware)
func getUserID(r *http.Request) int {
    // This should pull from context set by auth middleware
    // For now, placeholder
    if userID := r.Context().Value("user_id"); userID != nil {
        if id, ok := userID.(int); ok {
            return id
        }
    }
    return 0
}

// Helper: respond with JSON
func respondJSON(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(data)
}
```

**Step 4: Add missing model functions**

Add to `go-backend/models/notifications.go`:

```go
// MarkNotificationAsRead marks a notification as read or unread
func MarkNotificationAsRead(db *sql.DB, id, userID int, read bool) error {
    query := `UPDATE notifications SET is_read = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3`
    _, err := db.Exec(query, read, id, userID)
    return err
}

// ArchiveNotification archives a notification
func ArchiveNotification(db *sql.DB, id, userID int) error {
    query := `UPDATE notifications SET is_archived = TRUE, updated_at = NOW() WHERE id = $1 AND user_id = $2`
    _, err := db.Exec(query, id, userID)
    return err
}

// GetNotificationPreferences retrieves user preferences
func GetNotificationPreferences(db *sql.DB, userID int) (*NotificationPreferences, error) {
    query := `
        SELECT user_id, show_unprocessed_emails, show_starred_articles,
               show_priority_tasks, show_priority_feeds, items_per_page
        FROM notification_preferences WHERE user_id = $1
    `
    var prefs NotificationPreferences
    err := db.QueryRow(query, userID).Scan(
        &prefs.UserID,
        &prefs.ShowUnprocessedEmails,
        &prefs.ShowStarredArticles,
        &prefs.ShowPriorityTasks,
        &prefs.ShowPriorityFeeds,
        &prefs.ItemsPerPage,
    )
    if err == sql.ErrNoRows {
        // Return default preferences
        return &NotificationPreferences{
            UserID:                userID,
            ShowUnprocessedEmails: true,
            ShowStarredArticles:   true,
            ShowPriorityTasks:     true,
            ShowPriorityFeeds:     true,
            ItemsPerPage:          50,
        }, nil
    }
    return &prefs, err
}

// UpdateNotificationPreferences updates user preferences
func UpdateNotificationPreferences(db *sql.DB, prefs *NotificationPreferences) error {
    query := `
        INSERT INTO notification_preferences
        (user_id, show_unprocessed_emails, show_starred_articles, show_priority_tasks, show_priority_feeds, items_per_page)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (user_id)
        DO UPDATE SET
            show_unprocessed_emails = EXCLUDED.show_unprocessed_emails,
            show_starred_articles = EXCLUDED.show_starred_articles,
            show_priority_tasks = EXCLUDED.show_priority_tasks,
            show_priority_feeds = EXCLUDED.show_priority_feeds,
            items_per_page = EXCLUDED.items_per_page,
            updated_at = NOW()
    `
    _, err := db.Exec(query,
        prefs.UserID,
        prefs.ShowUnprocessedEmails,
        prefs.ShowStarredArticles,
        prefs.ShowPriorityTasks,
        prefs.ShowPriorityFeeds,
        prefs.ItemsPerPage,
    )
    return err
}
```

**Step 5: Run test to verify it passes**

Run: `cd go-backend && go test ./handlers -run TestNotification -v`
Expected: PASS (after fixing any test helper functions)

**Step 6: Commit**

```bash
git add go-backend/handlers/notifications.go go-backend/handlers/notifications_test.go go-backend/models/notifications.go
git commit -m "feat: add notifications API handler"
```

---

## Task 7: Add notifications routes

**Files:**
- Modify: `go-backend/main.go`

**Step 1: Add routes to main.go**

Find the section where routes are registered (look for `router.HandleFunc("/rss/",` or similar) and add:

```go
// Notifications routes
notificationHandler := &handlers.NotificationHandler{DB: db}
apiRouter.HandleFunc("/notifications", notificationHandler.ListNotifications).Methods("GET")
apiRouter.HandleFunc("/notifications/unread-count", notificationHandler.GetUnreadCount).Methods("GET")
apiRouter.HandleFunc("/notifications/{id}/read", notificationHandler.MarkAsRead).Methods("PATCH")
apiRouter.HandleFunc("/notifications/{id}/archive", notificationHandler.ArchiveNotification).Methods("PATCH")
apiRouter.HandleFunc("/notifications/preferences", notificationHandler.GetPreferences).Methods("GET")
apiRouter.HandleFunc("/notifications/preferences", notificationHandler.UpdatePreferences).Methods("PATCH")
```

**Step 2: Verify routes are added**

Run: `grep -n "notifications" go-backend/main.go`
Expected: See the new routes

**Step 3: Commit**

```bash
git add go-backend/main.go
git commit -m "feat: add notifications API routes"
```

---

## Task 8: Create frontend notifications API client

**Files:**
- Create: `zettelkasten-front/src/api/notifications.ts`

**Step 1: Write the API client**

```typescript
import { apiClient, getData } from "./client";

// Types
export interface Notification {
  id: number;
  user_id: number;
  source_type: 'email' | 'rss' | 'task';
  source_id: number;
  title: string;
  preview: string;
  timestamp: string;
  importance_score: number;
  is_read: boolean;
  is_archived: boolean;
  filter_tags: string[];
}

export interface NotificationPreferences {
  user_id: number;
  show_unprocessed_emails: boolean;
  show_starred_articles: boolean;
  show_priority_tasks: boolean;
  show_priority_feeds: boolean;
  items_per_page: number;
}

export interface NotificationListFilters {
  source_type?: string;
  unread_only?: boolean;
  limit?: number;
  offset?: number;
}

export interface NotificationListResponse {
  notifications: Notification[];
  total: number;
  unread_count: number;
}

export interface MarkReadRequest {
  read: boolean;
}

// Notification API
export function listNotifications(filters?: NotificationListFilters): Promise<NotificationListResponse> {
  const params = new URLSearchParams();
  if (filters?.source_type) params.set("source_type", filters.source_type);
  if (filters?.unread_only) params.set("unread_only", "true");
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());

  const query = params.toString();
  return getData(apiClient.get<NotificationListResponse>(`/notifications${query ? `?${query}` : ""}`));
}

export function getUnreadCount(): Promise<{ unread_count: number }> {
  return getData(apiClient.get<{ unread_count: number }>("/notifications/unread-count"));
}

export function markAsRead(id: number, read: boolean): Promise<void> {
  return getData(apiClient.patch<void>(`/notifications/${id}/read`, { read }));
}

export function archiveNotification(id: number): Promise<void> {
  return getData(apiClient.patch<void>(`/notifications/${id}/archive`, {}));
}

export function getPreferences(): Promise<NotificationPreferences> {
  return getData(apiClient.get<NotificationPreferences>("/notifications/preferences"));
}

export function updatePreferences(prefs: Partial<NotificationPreferences>): Promise<NotificationPreferences> {
  return getData(apiClient.patch<NotificationPreferences>("/notifications/preferences", prefs));
}
```

**Step 2: Verify file exists**

Run: `ls -la zettelkasten-front/src/api/notifications.ts`
Expected: File exists with content above

**Step 3: Commit**

```bash
git add zettelkasten-front/src/api/notifications.ts
git commit -m "feat: add notifications API client"
```

---

## Task 9: Create NotificationInboxPage component

**Files:**
- Create: `zettelkasten-front/src/pages/NotificationInboxPage.tsx`

**Step 1: Write the page component**

```typescript
import React, { useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { setDocumentTitle } from "../utils/title";
import {
  listNotifications,
  markAsRead,
  archiveNotification,
  getUnreadCount,
  Notification,
  NotificationListResponse,
} from "../api/notifications";

type TabType = 'all' | 'email' | 'rss' | 'task';

export function NotificationInboxPage() {
  const navigate = useNavigate();

  // State
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [total, setTotal] = useState(0);
  const [unreadCount, setUnreadCount] = useState(0);
  const [activeTab, setActiveTab] = useState<TabType>('all');
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(false);

  // Fetch notifications
  const fetchNotifications = useCallback(async (loadMore = false) => {
    if (loadMore) setLoadingMore(true);
    else setLoading(true);

    try {
      const offset = loadMore ? notifications.length : 0;
      const response: NotificationListResponse = await listNotifications({
        source_type: activeTab === 'all' ? undefined : activeTab,
        unread_only: true, // Default to showing unread
        limit: 50,
        offset,
      });

      if (loadMore) {
        setNotifications(prev => [...prev, ...response.notifications]);
      } else {
        setNotifications(response.notifications);
      }
      setTotal(response.total);
      setHasMore(response.notifications.length + offset < response.total);
    } catch (error) {
      console.error("Failed to fetch notifications:", error);
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  }, [activeTab, notifications.length]);

  // Fetch unread count
  const fetchUnreadCount = useCallback(async () => {
    try {
      const response = await getUnreadCount();
      setUnreadCount(response.unread_count);
    } catch (error) {
      console.error("Failed to fetch unread count:", error);
    }
  }, []);

  // Initial load
  useEffect(() => {
    fetchNotifications();
    fetchUnreadCount();
  }, [activeTab, fetchNotifications, fetchUnreadCount]);

  // Update document title
  useEffect(() => {
    if (unreadCount > 0) {
      setDocumentTitle(`Inbox (${unreadCount})`);
    } else {
      setDocumentTitle("Inbox");
    }
  }, [unreadCount]);

  // Handle notification click
  const handleNotificationClick = async (notification: Notification) => {
    // Mark as read
    if (!notification.is_read) {
      try {
        await markAsRead(notification.id, true);
        setNotifications(prev =>
          prev.map(n => n.id === notification.id ? { ...n, is_read: true } : n)
        );
        fetchUnreadCount();
      } catch (error) {
        console.error("Failed to mark as read:", error);
      }
    }

    // Navigate to source detail page
    switch (notification.source_type) {
      case 'email':
        navigate(`/app/emails/${notification.source_id}`);
        break;
      case 'rss':
        // Navigate to RSS page - would need to handle article selection
        navigate('/app/rss');
        break;
      case 'task':
        // Future: navigate to task detail
        navigate('/app');
        break;
    }
  };

  // Handle archive
  const handleArchive = async (notification: Notification, event: React.MouseEvent) => {
    event.stopPropagation();
    try {
      await archiveNotification(notification.id);
      setNotifications(prev => prev.filter(n => n.id !== notification.id));
      fetchUnreadCount();
    } catch (error) {
      console.error("Failed to archive:", error);
    }
  };

  // Render source icon
  const renderSourceIcon = (sourceType: string) => {
    switch (sourceType) {
      case 'email': return '📧';
      case 'rss': return '📰';
      case 'task': return '✓';
      default: return '📄';
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      {/* Header */}
      <div style={{
        borderBottom: '1px solid #e5e7eb',
        backgroundColor: '#ffffff',
        padding: '16px 24px',
      }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '16px',
        }}>
          <h1 style={{ fontSize: '24px', fontWeight: '700', color: '#111827', margin: 0 }}>
            Inbox
          </h1>
          {unreadCount > 0 && (
            <div style={{ fontSize: '14px', color: '#6b7280' }}>
              {unreadCount} {unreadCount === 1 ? 'item' : 'items'}
            </div>
          )}
        </div>

        {/* Tabs */}
        <div style={{ display: 'flex', gap: '8px' }}>
          <TabButton active={activeTab === 'all'} onClick={() => setActiveTab('all')}>
            All
          </TabButton>
          <TabButton active={activeTab === 'email'} onClick={() => setActiveTab('email')}>
            📧 Email
          </TabButton>
          <TabButton active={activeTab === 'rss'} onClick={() => setActiveTab('rss')}>
            📰 RSS
          </TabButton>
          <TabButton active={activeTab === 'task'} onClick={() => setActiveTab('task')}>
            ✓ Tasks
          </TabButton>
        </div>
      </div>

      {/* Notification List */}
      <div style={{ flex: 1, overflowY: 'auto', backgroundColor: '#ffffff' }}>
        {loading ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#6b7280' }}>
            Loading...
          </div>
        ) : notifications.length === 0 ? (
          <div style={{ padding: '48px', textAlign: 'center', color: '#6b7280' }}>
            No notifications
          </div>
        ) : (
          <div>
            {notifications.map(notification => (
              <NotificationItem
                key={notification.id}
                notification={notification}
                onClick={() => handleNotificationClick(notification)}
                onArchive={(e) => handleArchive(notification, e)}
              />
            ))}
            {hasMore && (
              <button
                onClick={() => fetchNotifications(true)}
                disabled={loadingMore}
                style={{
                  width: '100%',
                  padding: '16px',
                  border: 'none',
                  backgroundColor: '#f9fafb',
                  color: '#6b7280',
                  cursor: loadingMore ? 'not-allowed' : 'pointer',
                }}
              >
                {loadingMore ? 'Loading...' : 'Load more'}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

interface TabButtonProps {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}

function TabButton({ active, onClick, children }: TabButtonProps) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '8px 16px',
        fontSize: '14px',
        fontWeight: '500',
        borderRadius: '8px',
        border: 'none',
        cursor: 'pointer',
        transition: 'all 0.15s ease',
        backgroundColor: active ? '#3b82f6' : '#f3f4f6',
        color: active ? '#ffffff' : '#374151',
      }}
    >
      {children}
    </button>
  );
}

interface NotificationItemProps {
  notification: Notification;
  onClick: () => void;
  onArchive: (event: React.MouseEvent) => void;
}

function NotificationItem({ notification, onClick, onArchive }: NotificationItemProps) {
  const sourceIcon = notification.source_type === 'email' ? '📧' :
                     notification.source_type === 'rss' ? '📰' : '📄';

  return (
    <div
      onClick={onClick}
      style={{
        padding: '16px 24px',
        borderBottom: '1px solid #f3f4f6',
        cursor: 'pointer',
        backgroundColor: notification.is_read ? '#ffffff' : '#f9fafb',
        opacity: notification.is_read ? 0.7 : 1,
        transition: 'background-color 0.15s ease',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = '#f3f4f6';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = notification.is_read ? '#ffffff' : '#f9fafb';
      }}
    >
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px' }}>
        {/* Source icon */}
        <div style={{ fontSize: '20px', flexShrink: 0 }}>
          {sourceIcon}
        </div>

        {/* Content */}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            marginBottom: '4px',
          }}>
            <span style={{
              fontWeight: notification.is_read ? '400' : '600',
              color: '#111827',
              fontSize: '15px',
            }}>
              {notification.title}
            </span>
            {!notification.is_read && (
              <span style={{
                width: '8px',
                height: '8px',
                borderRadius: '50%',
                backgroundColor: '#3b82f6',
                flexShrink: 0,
              }} />
            )}
          </div>
          <div style={{
            color: '#6b7280',
            fontSize: '14px',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}>
            {notification.preview}
          </div>
          <div style={{
            color: '#9ca3af',
            fontSize: '12px',
            marginTop: '4px',
          }}>
            {new Date(notification.timestamp).toLocaleString()}
          </div>
        </div>

        {/* Archive button */}
        <button
          onClick={onArchive}
          style={{
            padding: '6px 12px',
            fontSize: '13px',
            borderRadius: '6px',
            border: '1px solid #e5e7eb',
            backgroundColor: '#ffffff',
            color: '#6b7280',
            cursor: 'pointer',
            flexShrink: 0,
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.backgroundColor = '#f3f4f6';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.backgroundColor = '#ffffff';
          }}
        >
          Archive
        </button>
      </div>
    </div>
  );
}
```

**Step 2: Verify file exists**

Run: `ls -la zettelkasten-front/src/pages/NotificationInboxPage.tsx`
Expected: File exists with content above

**Step 3: Commit**

```bash
git add zettelkasten-front/src/pages/NotificationInboxPage.tsx
git commit -m "feat: add NotificationInboxPage component"
```

---

## Task 10: Add Inbox route to router

**Files:**
- Modify: `zettelkasten-front/src/App.tsx` (or wherever routes are defined)

**Step 1: Add the route**

Find the routes section and add:

```typescript
import { NotificationInboxPage } from "./pages/NotificationInboxPage";

// In the routes, add:
<Route path="/app/inbox" element={<NotificationInboxPage />} />
```

**Step 2: Verify route is added**

Run: `grep -n "NotificationInboxPage\|/app/inbox" zettelkasten-front/src/App.tsx`
Expected: See the new route

**Step 3: Commit**

```bash
git add zettelkasten-front/src/App.tsx
git commit -m "feat: add inbox route"
```

---

## Task 11: Add Inbox link to sidebar navigation

**Files:**
- Modify: `zettelkasten-front/src/components/Sidebar.tsx` (or similar sidebar component)

**Step 1: Find the sidebar component**

Run: `grep -r "DashboardPage\|RssPage" zettelkasten-front/src/components/ | grep -v node_modules`
Expected: Find the sidebar/navigation component

**Step 2: Add the Inbox link**

Add between Dashboard and RSS links:

```typescript
import { useNavigate } from "react-router-dom";
import { getUnreadCount } from "../api/notifications";
import { useEffect, useState } from "react";

// In the sidebar, add:
const [unreadCount, setUnreadCount] = useState(0);

useEffect(() => {
  getUnreadCount().then(response => setUnreadCount(response.unread_count));
}, []);

// In the nav links, add:
<div
  onClick={() => navigate('/app/inbox')}
  style={{
    padding: '8px 12px',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  }}
>
  📬 Inbox
  {unreadCount > 0 && (
    <span style={{
      backgroundColor: '#ef4444',
      color: 'white',
      fontSize: '11px',
      padding: '2px 6px',
      borderRadius: '10px',
      marginLeft: 'auto',
    }}>
      {unreadCount}
    </span>
  )}
</div>
```

**Step 3: Verify sidebar has Inbox link**

Run: `grep -n "Inbox\|/app/inbox" zettelkasten-front/src/components/Sidebar.tsx`
Expected: See the new Inbox link

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/Sidebar.tsx
git commit -m "feat: add Inbox link to sidebar with unread badge"
```

---

## Task 12: Backfill existing data into notifications table

**Files:**
- Create: `go-backend/schema/0125-backfill-notifications.sql`

**Step 1: Write the backfill migration**

```sql
-- Backfill existing unprocessed/triaged emails as notifications
INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
SELECT
    user_id,
    'email'::VARCHAR,
    id,
    COALESCE(subject, '(No subject)'),
    LEFT(COALESCE(body_text, ''), 200),
    COALESCE(received_at, created_at),
    CASE WHEN status = 'unprocessed' THEN 10 ELSE 5 END,
    ARRAY[status]
FROM emails
WHERE status IN ('unprocessed', 'triaged')
ON CONFLICT (user_id, source_type, source_id) DO NOTHING;

-- Backfill existing starred articles and priority feed articles
INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
SELECT
    a.user_id,
    'rss'::VARCHAR,
    a.id,
    a.title,
    LEFT(COALESCE(a.content, ''), 200),
    COALESCE(a.published_at, a.fetched_at),
    CASE
        WHEN a.is_starred = TRUE THEN 10
        WHEN f.priority = TRUE THEN 5
        ELSE 0
    END,
    CASE
        WHEN a.is_starred = TRUE THEN ARRAY['starred']
        WHEN f.priority = TRUE THEN ARRAY['priority']
        ELSE ARRAY[]::TEXT[]
    END
FROM rss_articles a
JOIN rss_feeds f ON a.feed_id = f.id
WHERE a.is_starred = TRUE OR (f.priority = TRUE AND a.read = FALSE)
ON CONFLICT (user_id, source_type, source_id) DO NOTHING;
```

**Step 2: Verify migration file exists**

Run: `ls -la go-backend/schema/0125-backfill-notifications.sql`
Expected: File exists with content above

**Step 3: Commit**

```bash
git add go-backend/schema/0125-backfill-notifications.sql
git commit -m "schema: backfill existing data into notifications table"
```

---

## Task 13: Run migrations and verify

**Files:**
- None (verification task)

**Step 1: Check if migrations are auto-run**

Run: `grep -r "migration\|schema" go-backend/main.go | head -20`
Note: Check how migrations are run in your project

**Step 2: Run migrations (if manual)**

If your project runs migrations automatically on startup, just start the server. Otherwise:
Run: `cd go-backend && [your migration command]`

**Step 3: Verify triggers work**

Run: `psql [your-db] -c "\d+ notifications"` and check triggers on emails/rss_articles

**Step 4: Create test email and verify notification appears**

Run: `[your email sync command or API call to create test email]`

Run: `psql [your-db] -c "SELECT * FROM notifications LIMIT 5;"`

**Step 5: Commit any fixes**

```bash
git add .
git commit -m "fix: any migration or trigger issues found during testing"
```

---

## Task 14: Write end-to-end tests for notifications

**Files:**
- Create: `go-backend/handlers/notifications_e2e_test.go`

**Step 1: Write integration tests**

```go
package handlers

import (
    "testing"
    "time"
)

func TestNotificationWorkflow(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    defer teardownTestDB(t, db)
    handler := &NotificationHandler{DB: db}

    user := createTestUser(t, db)

    // Test 1: Email creates notification
    email := createTestEmail(t, db, user.ID, "test@example.com", "Test Subject", "unprocessed")

    notifications, _, err := models.GetNotificationsByUser(db, user.ID, "email", true, 10, 0)
    if err != nil {
        t.Fatalf("Failed to get notifications: %v", err)
    }
    if len(notifications) == 0 {
        t.Error("Expected notification after email creation")
    }

    // Test 2: Mark as read
    req := createMarkReadRequest(t, notifications[0].ID, true)
    // ... execute request and verify

    // Test 3: Archive
    // ... execute archive and verify notification is excluded from queries

    // Test 4: RSS article creates notification
    feed := createTestFeed(t, db, user.ID, true) // priority feed
    article := createTestArticle(t, db, user.ID, feed.ID, "Test Article", true) // starred

    notifications, _, err = models.GetNotificationsByUser(db, user.ID, "rss", true, 10, 0)
    if err != nil {
        t.Fatalf("Failed to get RSS notifications: %v", err)
    }
    if len(notifications) == 0 {
        t.Error("Expected notification after starred article creation")
    }
}

func TestNotificationPreferences(t *testing.T) {
    // Test default preferences
    // Test updating preferences
    // Test preferences affect filtering
}
```

**Step 2: Run tests**

Run: `cd go-backend && go test ./handlers -run TestNotification -v`

**Step 3: Commit**

```bash
git add go-backend/handlers/notifications_e2e_test.go
git commit -m "test: add end-to-end tests for notifications"
```

---

## Task 15: Manual testing and bug fixes

**Files:**
- Varies based on bugs found

**Step 1: Create a test checklist**

- [ ] Email appears in unified inbox when unprocessed
- [ ] Email disappears when archived
- [ ] Starred RSS article appears in unified inbox
- [ ] Clicking notification navigates to correct source page
- [ ] Mark as read works correctly
- [ ] Archive button works
- [ ] Unread count badge updates
- [ ] Tabs filter correctly (All, Email, RSS)
- [ ] Load more pagination works
- [ ] Preferences can be updated

**Step 2: Run through checklist manually**

Start the dev server and test each item

**Step 3: Fix any bugs found**

Create individual commits for each fix

**Step 4: Final commit**

```bash
git add .
git commit -m "fix: bugs found during manual testing of unified inbox"
```

---

## Summary

This implementation plan creates a unified notification inbox through:

1. **Database layer**: New `notifications` and `notification_preferences` tables with sync triggers
2. **Backend API**: Go handlers for listing, filtering, and managing notifications
3. **Frontend**: React page with tabbed interface, source indicators, and navigation

The architecture is extensible - adding tasks as a notification source later only requires:
- Adding a sync trigger on the tasks table
- Adding task route handling in the frontend click handler

All work follows TDD with tests written before implementation, and frequent commits.
