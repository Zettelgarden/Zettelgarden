package jobs

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"go-backend/server"
	"go-backend/settings"
)

// retentionManager builds a settings manager (config.yaml in a temp dir) with
// the cleanup retention keys set to the given days.
func retentionManager(t *testing.T, jobDays, rssDays int) *settings.Manager {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	m, err := settings.New(path)
	if err != nil {
		t.Fatalf("settings.New: %v", err)
	}
	if err := m.Set("job_retention_days", strconv.Itoa(jobDays)); err != nil {
		t.Fatalf("Set job_retention_days: %v", err)
	}
	if err := m.Set("rss_article_retention_days", strconv.Itoa(rssDays)); err != nil {
		t.Fatalf("Set rss_article_retention_days: %v", err)
	}
	return m
}

func TestCleanupJobRetentionFromSettings(t *testing.T) {
	db, err := server.OpenSQLite("file:cleanup_retention?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	// Minimal tables the cleanup handler touches: it deletes from
	// scheduled_job_runs first (hardcoded 90 days), then llm_jobs via
	// models.CleanupOldJobs with the settings-driven retention.
	stmts := []string{
		`CREATE TABLE scheduled_job_runs (id INTEGER PRIMARY KEY, created_at TEXT)`,
		`CREATE TABLE llm_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			job_type TEXT NOT NULL,
			status TEXT DEFAULT 'pending' NOT NULL,
			payload TEXT NOT NULL,
			completed_at TEXT
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	// One completed job from 10 days ago (beyond the 7-day retention) and one
	// from 2 days ago (kept).
	old := time.Now().UTC().AddDate(0, 0, -10)
	recent := time.Now().UTC().AddDate(0, 0, -2)
	for _, ts := range []time.Time{old, recent} {
		if _, err := db.Exec(
			`INSERT INTO llm_jobs (user_id, job_type, status, payload, completed_at) VALUES (1, 'summarize', 'completed', '{}', $1)`,
			ts,
		); err != nil {
			t.Fatalf("insert llm_job: %v", err)
		}
	}

	job := NewCleanupJob(db, retentionManager(t, 7, 30))
	if err := job.Handler(context.Background()); err != nil {
		t.Fatalf("Handler: %v", err)
	}

	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM llm_jobs`).Scan(&remaining); err != nil {
		t.Fatalf("count llm_jobs: %v", err)
	}
	if remaining != 1 {
		t.Errorf("llm_jobs remaining = %d, want 1 (10-day-old job purged by 7-day retention)", remaining)
	}
}

func TestRSSArticleCleanupJobRetentionFromSettings(t *testing.T) {
	db, err := server.OpenSQLite("file:rss_cleanup_retention?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE notifications (id INTEGER PRIMARY KEY, source_type TEXT, source_id INTEGER)`,
		`CREATE TABLE rss_articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			feed_id INTEGER NOT NULL,
			title TEXT,
			url TEXT NOT NULL,
			fetched_at TEXT,
			is_starred INTEGER DEFAULT 0,
			card_id INTEGER
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	// One unstarred, unconverted article fetched 10 days ago (beyond the
	// 7-day retention -> purged) and one fetched 2 days ago (kept).
	old := time.Now().UTC().AddDate(0, 0, -10)
	recent := time.Now().UTC().AddDate(0, 0, -2)
	if _, err := db.Exec(
		`INSERT INTO rss_articles (user_id, feed_id, title, url, fetched_at, is_starred, card_id) VALUES (1, 1, 'old', 'https://old', $1, 0, NULL)`,
		old,
	); err != nil {
		t.Fatalf("insert old article: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO rss_articles (user_id, feed_id, title, url, fetched_at, is_starred, card_id) VALUES (1, 1, 'recent', 'https://recent', $1, 0, NULL)`,
		recent,
	); err != nil {
		t.Fatalf("insert recent article: %v", err)
	}

	job := NewRSSArticleCleanupJob(db, retentionManager(t, 30, 7))
	if err := job.Handler(context.Background()); err != nil {
		t.Fatalf("Handler: %v", err)
	}

	var remaining int
	if err := db.QueryRow(`SELECT COUNT(*) FROM rss_articles`).Scan(&remaining); err != nil {
		t.Fatalf("count rss_articles: %v", err)
	}
	if remaining != 1 {
		t.Errorf("rss_articles remaining = %d, want 1 (10-day-old article purged by 7-day retention)", remaining)
	}
}
