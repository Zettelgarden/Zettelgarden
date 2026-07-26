package services

import (
	"database/sql"
	"log"

	"go-backend/models"
)

// This file ports the Postgres rss-notification trigger (schema/0124) and the
// rss half of the delete_notification helper (schema/0122) to Go. The triggers
// are dropped for BOTH drivers by migration 0146; Go maintains the behaviour on
// Postgres and SQLite alike (migration design doc Phase 5, decision 3b).
//
// A notification exists for an RSS article when it is starred (importance 10) or
// is an unread article in a priority feed (importance 5). When an article no
// longer qualifies (unstarrd, or a priority-feed article marked read) its
// notification is deleted. All calls are best-effort (log + continue): a stale
// notification is a minor display issue, never worth failing a request for.
//
// filter_tags: the old trigger emitted only {starred?, priority?}. The Go
// helper models.GetFilterTagsForRSS emits a superset ({"rss", starred?,
// priority?, folder?, feedName?}); the extras are additive and harmless to
// subset-based notification filtering, and CreateNotification already expects
// its output.

// SyncRSSArticleNotification upserts (or deletes) the notification for an RSS
// article to match its current state. Call after every insert/update of an
// rss_articles row.
func SyncRSSArticleNotification(db models.Database, article *models.RSSArticle) {
	if article == nil {
		return
	}

	// The trigger joined rss_feeds to read the priority flag.
	var feedPriority bool
	err := db.QueryRow(`SELECT priority FROM rss_feeds WHERE id = $1`, article.FeedID).Scan(&feedPriority)
	if err == sql.ErrNoRows {
		return // feed missing -> trigger skips notification processing
	} else if err != nil {
		log.Printf("[rss-article:%d] notification sync (feed lookup): %v", article.ID, err)
		return
	}

	if !(article.IsStarred || (feedPriority && !article.Read)) {
		// No longer qualifies -> remove any existing notification (matches the
		// trigger's "if not starred, delete" branch; also covers priority-read).
		if err := models.DeleteNotificationBySource(db, article.UserID, models.SourceTypeRSS, article.ID); err != nil {
			log.Printf("[rss-article:%d] delete notification: %v", article.ID, err)
		}
		return
	}

	importance := models.CalculateImportanceScore(models.SourceTypeRSS, article.IsStarred, feedPriority)
	tags := models.GetFilterTagsForRSS(article.IsStarred, feedPriority, "", "")

	// timestamp = COALESCE(published_at, fetched_at); preview = LEFT(content, 200).
	ts := article.FetchedAt
	if article.PublishedAt != nil {
		ts = *article.PublishedAt
	}
	content := ""
	if article.Content != nil {
		content = *article.Content
	}
	if r := []rune(content); len(r) > 200 {
		content = string(r[:200]) // rune-safe; Postgres LEFT() counts characters
	}

	if _, err := models.CreateNotification(db, article.UserID, models.SourceTypeRSS, article.ID, article.Title, &content, ts, importance, tags); err != nil {
		log.Printf("[rss-article:%d] create notification: %v", article.ID, err)
	}
}

// syncRSSArticleNotificationsByPredicate re-syncs notifications for every
// rss_articles row matching a predicate. Used by the bulk mark-as-read paths
// (feed/folder), where the old per-row trigger would have fired on each updated
// row. Best-effort.
func syncRSSArticleNotificationsByPredicate(db models.Database, userID int, predicate string, args ...interface{}) {
	rows, err := db.Query(
		`SELECT id FROM rss_articles WHERE user_id = $1 AND `+predicate,
		append([]interface{}{userID}, args...)...,
	)
	if err != nil {
		log.Printf("[rss] notification sync query (%s): %v", predicate, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		if art, err := GetRSSArticleByID(db, userID, id); err == nil {
			SyncRSSArticleNotification(db, art)
		}
	}
}
