package services

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go-backend/models"
	"go-backend/server"
)

// loadConsolidatedSchema opens an in-memory SQLite and applies the real
// consolidated schema (the source of truth) so services tests run against
// actual table shapes, FKs, and the unique index that backs
// CreateNotification's ON CONFLICT.
func loadConsolidatedSchema(t *testing.T) *server.Server {
	t.Helper()
	db := freshSQLiteDB(t)
	// services/rss_notifications_sqlite_test.go -> ../schema/sqlite/schema.sqlite.sql
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	p := filepath.Join(filepath.Dir(filename), "..", "schema", "sqlite", "schema.sqlite.sql")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read consolidated schema: %v", err)
	}
	for _, stmt := range server.SplitSQL(string(raw)) {
		if _, err := db.Exec(stmt); err != nil {
			head := strings.TrimSpace(stmt)
			if i := strings.IndexByte(head, '\n'); i >= 0 {
				head = head[:i]
			}
			t.Fatalf("apply consolidated schema statement: %v\n  %s", err, head)
		}
	}
	return &server.Server{DB: db}
}

// rssTestWorld seeds one user and two feeds (priority + non-priority) and
// returns the handles the scenarios need.
type rssTestWorld struct {
	s            *server.Server
	userID       int
	priorityFeed int // feed id
	plainFeed    int // feed id
}

func setupRSSWorld(t *testing.T) *rssTestWorld {
	t.Helper()
	s := loadConsolidatedSchema(t)
	res, err := s.DB.Exec(`INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`, "u", "u@e.com", "x")
	if err != nil {
		t.Fatal(err)
	}
	uid, _ := res.LastInsertId()
	mustFeed := func(url, name string, priority bool) int {
		r, e := s.DB.Exec(`INSERT INTO rss_feeds (user_id, url, name, priority) VALUES ($1, $2, $3, $4)`, uid, url, name, priority)
		if e != nil {
			t.Fatal(e)
		}
		id, _ := r.LastInsertId()
		return int(id)
	}
	return &rssTestWorld{s: s, userID: int(uid), priorityFeed: mustFeed("https://a", "pfeed", true), plainFeed: mustFeed("https://b", "plain", false)}
}

func rssArticle(userID, feedID, id int, starred, read bool) *models.RSSArticle {
	c := "body content"
	return &models.RSSArticle{
		ID:          id,
		UserID:      userID,
		FeedID:      feedID,
		Title:       "T",
		Content:     &c,
		URL:         "https://x",
		PublishedAt: nil,
		FetchedAt:   time.Now().UTC(),
		Read:        read,
		IsStarred:   starred,
	}
}

func notifImportance(t *testing.T, db models.Database, userID, sourceID int) (exists bool, importance int) {
	t.Helper()
	err := db.QueryRow(
		`SELECT importance_score FROM notifications WHERE user_id = $1 AND source_type = 'rss' AND source_id = $2`,
		userID, sourceID,
	).Scan(&importance)
	if err != nil {
		return false, 0
	}
	return true, importance
}

func TestSyncRSSArticleNotification(t *testing.T) {
	w := setupRSSWorld(t)

	// 1. Priority feed, unread, not starred -> importance 5.
	a := rssArticle(w.userID, w.priorityFeed, 100, false, false)
	SyncRSSArticleNotification(w.s.DB, a)
	exists, imp := notifImportance(t, w.s.DB, w.userID, 100)
	if !exists || imp != 5 {
		t.Fatalf("priority-unread: exists=%v importance=%d, want exists=true importance=5", exists, imp)
	}

	// 2. Mark it read -> notification removed.
	a.Read = true
	SyncRSSArticleNotification(w.s.DB, a)
	exists, _ = notifImportance(t, w.s.DB, w.userID, 100)
	if exists {
		t.Fatalf("priority-read: notification still present, want deleted")
	}

	// 3. Non-priority feed, starred -> importance 10.
	b := rssArticle(w.userID, w.plainFeed, 200, true, false)
	SyncRSSArticleNotification(w.s.DB, b)
	exists, imp = notifImportance(t, w.s.DB, w.userID, 200)
	if !exists || imp != 10 {
		t.Fatalf("starred: exists=%v importance=%d, want exists=true importance=10", exists, imp)
	}

	// 4. Unstar -> notification removed.
	b.IsStarred = false
	SyncRSSArticleNotification(w.s.DB, b)
	exists, _ = notifImportance(t, w.s.DB, w.userID, 200)
	if exists {
		t.Fatalf("unstarred: notification still present, want deleted")
	}

	// 5. Non-priority, not starred, unread -> no notification created.
	c := rssArticle(w.userID, w.plainFeed, 300, false, false)
	SyncRSSArticleNotification(w.s.DB, c)
	exists, _ = notifImportance(t, w.s.DB, w.userID, 300)
	if exists {
		t.Fatalf("plain unread non-starred: notification created, want none")
	}

	// 6. Starred overrides read: a starred, read article still notifies (imp 10).
	d := rssArticle(w.userID, w.plainFeed, 400, true, true)
	SyncRSSArticleNotification(w.s.DB, d)
	exists, imp = notifImportance(t, w.s.DB, w.userID, 400)
	if !exists || imp != 10 {
		t.Fatalf("starred+read: exists=%v importance=%d, want exists=true importance=10", exists, imp)
	}
}

// TestSyncRSSArticleNotificationMissingFeed proves the best-effort contract:
// an article whose feed was deleted is a no-op (no panic, no notification).
func TestSyncRSSArticleNotificationMissingFeed(t *testing.T) {
	w := setupRSSWorld(t)
	a := rssArticle(w.userID, 999999, 500, true, false) // feed does not exist
	SyncRSSArticleNotification(w.s.DB, a)               // must not panic
	exists, _ := notifImportance(t, w.s.DB, w.userID, 500)
	if exists {
		t.Fatalf("missing feed: notification created, want none")
	}
}

// countingDatabase wraps models.Database and tallies SELECTs that read
// rss_articles, so the N+1 regression (one GetRSSArticleByID per matched
// article in the bulk mark-as-read path) is caught by an exact-count assert.
type countingDatabase struct {
	models.Database
	rssArticleSelects int
}

func (c *countingDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if strings.Contains(strings.ToLower(query), "from rss_articles") {
		c.rssArticleSelects++
	}
	return c.Database.Query(query, args...)
}

func (c *countingDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	if strings.Contains(strings.ToLower(query), "from rss_articles") {
		c.rssArticleSelects++
	}
	return c.Database.QueryRow(query, args...)
}

// TestMarkRSSFeedAsReadSingleArticleSelect pins the bn2 fix: the bulk
// mark-as-read paths must read the article set with exactly ONE rss_articles
// SELECT (the predicate query hydrating full rows), not one per article.
func TestMarkRSSFeedAsReadSingleArticleSelect(t *testing.T) {
	w := setupRSSWorld(t)
	const n = 25
	for i := 0; i < n; i++ {
		if _, err := w.s.DB.Exec(
			`INSERT INTO rss_articles (user_id, feed_id, title, url, fetched_at) VALUES ($1, $2, $3, $4, $5)`,
			w.userID, w.priorityFeed, fmt.Sprintf("a%d", i), fmt.Sprintf("https://a/%d", i), time.Now().UTC(),
		); err != nil {
			t.Fatalf("insert article %d: %v", i, err)
		}
	}

	db := &countingDatabase{Database: w.s.DB}
	if err := MarkRSSFeedAsRead(db, w.userID, w.priorityFeed); err != nil {
		t.Fatalf("MarkRSSFeedAsRead: %v", err)
	}
	if db.rssArticleSelects != 1 {
		t.Fatalf("rss_articles SELECTs = %d, want exactly 1 (N+1 regression)", db.rssArticleSelects)
	}

	// Behaviour unchanged: every priority-feed article is now read, so none
	// qualifies for a notification (priority && !read) and all are removed.
	var remaining int
	if err := w.s.DB.QueryRow(
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND source_type = 'rss'`, w.userID,
	).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("notifications remaining = %d, want 0 (all priority-read notifications removed)", remaining)
	}
}
