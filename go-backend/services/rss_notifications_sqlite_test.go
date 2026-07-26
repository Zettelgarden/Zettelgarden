package services

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go-backend/models"
	"go-backend/server"
)

// loadConsolidatedSchemaForRSS opens an in-memory SQLite and applies the real
// consolidated schema (the source of truth) so the rss/notification logic is
// tested against actual table shapes, FKs, and the unique index that backs
// CreateNotification's ON CONFLICT.
func loadConsolidatedSchemaForRSS(t *testing.T) *server.Server {
	t.Helper()
	db, err := server.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
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
	return &server.Server{DB: db, Driver: "sqlite"}
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
	s := loadConsolidatedSchemaForRSS(t)
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
