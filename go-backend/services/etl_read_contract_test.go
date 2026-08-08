package services

import (
	"testing"
	"time"

	"go-backend/models"
)

// ETL->app read contract (epic Zettelgarden-c7j, issue c7j.5). The
// 2026-07-29 cutover shipped a latent bug: cmd/migrate-pg-to-sqlite's
// normalizeValue stored jsonb as TEXT (a Go string) and PG arrays as JSON,
// while the app reads jsonb as []byte (json.RawMessage, which cannot scan a Go
// string) and arrays as {a,b} array-literal text (models.StringArray, the
// pq.StringArray replacement). The fix (3d75fa94) made the ETL keep jsonb as
// []byte (BLOB storage class — reads return []byte, scan into
// *json.RawMessage) and pass PG-array text through verbatim.
//
// TestETLStorageIsAppReadable inserts rows using the POST-FIX ETL storage
// shapes and reads them through the real app read paths (services.GetFullCard,
// the starred-search models scan, models.GetNotificationsByUser).
// TestBuggyETLStorageFailsScan proves the PRE-FIX shapes fail to scan, so a
// normalizeValue regression to TEXT/JSON storage breaks this suite instead of
// production.

func seedETLUser(t *testing.T, db models.Database) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`,
		"etl", "etl@example.com", "x",
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	uid, _ := res.LastInsertId()
	return int(uid)
}

func TestETLStorageIsAppReadable(t *testing.T) {
	s := loadConsolidatedSchema(t) // fresh DB, real consolidated schema
	db := s.DB
	userID := seedETLUser(t, db)

	// cards.structured_data — ETL output is []byte (BLOB storage class).
	jsonb := []byte(`{"type":"literature","title":"ETL Card"}`)
	now := time.Now().UTC()
	res, err := db.Exec(
		`INSERT INTO cards (user_id, card_id, title, body, link, structured_data, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, "etl-1", "etl card", "body", "", jsonb, now, now,
	)
	if err != nil {
		t.Fatalf("insert card with ETL-shaped structured_data: %v", err)
	}
	cardPK, _ := res.LastInsertId()

	card, err := GetFullCard(db, userID, int(cardPK))
	if err != nil {
		t.Fatalf("GetFullCard on ETL-shaped structured_data: %v", err)
	}
	if card.StructuredData == nil || string(*card.StructuredData) != string(jsonb) {
		t.Fatalf("structured_data mismatch: got %v, want %s", card.StructuredData, jsonb)
	}

	// starred_searches.search_config — ETL output is []byte (BLOB). This is
	// the exact SELECT+Scan of handlers.GetStarredSearchesRoute.
	res, err = db.Exec(
		`INSERT INTO starred_searches (user_id, title, search_term, search_config) VALUES ($1, $2, $3, $4)`,
		userID, "saved", "query", jsonb,
	)
	if err != nil {
		t.Fatalf("insert starred search with ETL-shaped search_config: %v", err)
	}

	var ss models.StarredSearch
	err = db.QueryRow(
		`SELECT id, user_id, title, search_term, search_config, created_at
		 FROM starred_searches WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	).Scan(&ss.ID, &ss.UserID, &ss.Title, &ss.SearchTerm, &ss.SearchConfig, &ss.CreatedAt)
	if err != nil {
		t.Fatalf("scan starred_searches.search_config (ETL shape): %v", err)
	}
	if string(ss.SearchConfig) != string(jsonb) {
		t.Fatalf("search_config mismatch: got %s, want %s", ss.SearchConfig, jsonb)
	}

	// notifications.filter_tags — ETL output is the {a,b} array literal,
	// passed through verbatim (models.StringArray / pq.StringArray format).
	arr := "{rss,starred}"
	if _, err := db.Exec(
		`INSERT INTO notifications (user_id, source_type, source_id, title, timestamp, importance_score, filter_tags)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, models.SourceTypeRSS, 7, "n", time.Now().UTC(), 10, arr,
	); err != nil {
		t.Fatalf("insert notification with ETL-shaped filter_tags: %v", err)
	}

	notifs, err := models.GetNotificationsByUser(db, userID, models.NotificationListFilters{})
	if err != nil {
		t.Fatalf("GetNotificationsByUser on ETL-shaped filter_tags: %v", err)
	}
	if len(notifs) != 1 || len(notifs[0].FilterTags) != 2 ||
		notifs[0].FilterTags[0] != "rss" || notifs[0].FilterTags[1] != "starred" {
		t.Fatalf("filter_tags mismatch: got %+v", notifs)
	}
}

// TestBuggyETLStorageFailsScan asserts the PRE-FIX normalizeValue shapes
// (jsonb as TEXT string, arrays as JSON ["a","b"]) fail to scan — proving the
// read-path tests above would catch a regression to that storage.
func TestBuggyETLStorageFailsScan(t *testing.T) {
	s := loadConsolidatedSchema(t)
	db := s.DB
	userID := seedETLUser(t, db)

	// jsonb stored as TEXT (Go string) — the pre-fix ETL output.
	now := time.Now().UTC()
	res, err := db.Exec(
		`INSERT INTO cards (user_id, card_id, title, body, link, structured_data, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, "buggy-1", "buggy", "b", "", `{"type":"literature"}`, now, now,
	)
	if err != nil {
		t.Fatalf("insert card with TEXT structured_data: %v", err)
	}
	cardPK, _ := res.LastInsertId()

	if _, err := GetFullCard(db, userID, int(cardPK)); err == nil {
		t.Fatal("GetFullCard succeeded on TEXT-stored structured_data, want scan error")
	}

	// filter_tags stored as JSON array text (the pre-fix parsePGArray output).
	if _, err := db.Exec(
		`INSERT INTO notifications (user_id, source_type, source_id, title, timestamp, importance_score, filter_tags)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, models.SourceTypeRSS, 8, "n", time.Now().UTC(), 10, `["rss","starred"]`,
	); err != nil {
		t.Fatalf("insert notification with JSON filter_tags: %v", err)
	}

	if _, err := models.GetNotificationsByUser(db, userID, models.NotificationListFilters{}); err == nil {
		t.Fatal("GetNotificationsByUser succeeded on JSON-array filter_tags, want scan error")
	}
}
