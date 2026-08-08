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

// rssArticleColumns is the canonical rss_articles projection used by both
// GetRSSArticleByID and the bulk predicate sync below. Kept in one place so
// the two read paths cannot drift apart.
const rssArticleColumns = `id, user_id, feed_id, title, content, author, url,
       published_at, fetched_at, read, card_id, is_starred`

// rowScanner is satisfied by both *sql.Rows and *sql.Row, so scanRSSArticle
// works for the single-row and multi-row read paths alike.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanRSSArticle hydrates an *models.RSSArticle from a row in the canonical
// rssArticleColumns projection (nullable content/author/published_at/card_id
// are lifted onto the model's pointer fields).
func scanRSSArticle(s rowScanner) (*models.RSSArticle, error) {
	var article models.RSSArticle
	var content, author sql.NullString
	var publishedAt sql.NullTime
	var cardID sql.NullInt64

	err := s.Scan(
		&article.ID, &article.UserID, &article.FeedID, &article.Title,
		&content, &author, &article.URL, &publishedAt,
		&article.FetchedAt, &article.Read, &cardID, &article.IsStarred,
	)
	if err != nil {
		return nil, err
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
	return &article, nil
}

// syncRSSArticleNotificationsByPredicate re-syncs notifications for every
// rss_articles row matching a predicate. Used by the bulk mark-as-read paths
// (feed/folder), where the old per-row trigger would have fired on each updated
// row. Best-effort. The full row is fetched in the single predicate query (no
// per-row GetRSSArticleByID round-trip) and hydrated via scanRSSArticle.
func syncRSSArticleNotificationsByPredicate(db models.Database, userID int, predicate string, args ...interface{}) {
	rows, err := db.Query(
		`SELECT `+rssArticleColumns+` FROM rss_articles WHERE user_id = $1 AND `+predicate,
		append([]interface{}{userID}, args...)...,
	)
	if err != nil {
		log.Printf("[rss] notification sync query (%s): %v", predicate, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		article, err := scanRSSArticle(rows)
		if err != nil {
			log.Printf("[rss] notification sync scan: %v", err)
			continue
		}
		SyncRSSArticleNotification(db, article)
	}
}
