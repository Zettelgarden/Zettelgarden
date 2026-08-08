package services

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"go-backend/models"
	"go-backend/server"
)

// loadSpikeSchema opens an in-memory SQLite DB (via the Phase 1 OpenSQLite
// helper, which applies the D4 pragmas pool-wide) and loads the Phase 1 spike
// mini-schema through the Phase 1 statement splitter — exercising the same
// migration-runner code path that will load the Phase 2 consolidated schema.
func loadSpikeSchema(t *testing.T) *server.Server {
	t.Helper()
	db := freshSQLiteDB(t)

	script, err := os.ReadFile("testdata/spike_cards.sqlite.sql")
	if err != nil {
		t.Fatalf("read spike schema: %v", err)
	}
	for _, stmt := range server.SplitSQL(string(script)) {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("apply spike schema statement: %v\n  statement: %s", err, stmt)
		}
	}
	return &server.Server{DB: db}
}

// seedUser inserts a user and returns its id. The cards FK requires it.
func seedUser(t *testing.T, s *server.Server) int {
	t.Helper()
	res, err := s.DB.Exec(
		`INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`,
		"spike", "spike@example.com", "x",
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return int(id)
}

// TestCreateCardSQLite is the Phase 1 end-to-end spike for the cards read+write
// path on SQLite. It drives the real services.CreateCard flow (INSERT with
// app-side time + RETURNING, then GetFullCard, then the tag/backlink/audit side
// effects) against an in-memory SQLite DB built from the spike mini-schema, and
// confirms the created card round-trips with correctly-typed timestamps.
//
// This validates, in one real handler-adjacent path:
//   - the Phase 1 OpenSQLite + pragma setup,
//   - the statement splitter loading a multi-statement schema,
//   - modernc.org/sqlite binding $1..$N params natively,
//   - RETURNING on INSERT,
//   - time.Time scanning from DATETIME columns (D5),
//   - the first NOW() -> app-side time.Now().UTC() translations, and
//   - JSONB-as-TEXT structured_data round-trip into *json.RawMessage.
//
// It requires no Postgres and no Typesense (UpsertCardToTypesense is skipped
// via ZETTEL_IS_TESTING=true).
func TestCreateCardSQLite(t *testing.T) {
	// Skip Typesense indexing (no collection running in unit tests).
	t.Setenv("ZETTEL_IS_TESTING", "true")

	s := loadSpikeSchema(t)
	userID := seedUser(t, s)

	params := models.EditCardParams{
		CardID: "testcard",
		Title:  "Spike Card",
		Body:   "A simple root card with no hashtags or links.",
		Link:   "",
	}

	created, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	// INSERT RETURNING populated the id and the app-side time bound created_at.
	if created.ID == 0 {
		t.Fatalf("created.ID = 0, want non-zero (RETURNING id failed)")
	}
	if created.Title != params.Title || created.Body != params.Body {
		t.Fatalf("created card fields = %+v, want title/body %q/%q", created, params.Title, params.Body)
	}
	if created.CardID != params.CardID {
		t.Fatalf("created.CardID = %q, want %q", created.CardID, params.CardID)
	}
	// A root card's parent_id is set to its own id by CreateCard.
	if created.ParentID == nil || *created.ParentID != created.ID {
		t.Fatalf("created.ParentID = %v, want %d (root self-parent)", created.ParentID, created.ID)
	}
	// Timestamps must come back as real time.Time values (D5: DATETIME, not TEXT).
	if created.CreatedAt.IsZero() {
		t.Fatalf("created.CreatedAt is zero — time.Time scan from DATETIME failed")
	}
	if created.UpdatedAt.IsZero() {
		t.Fatalf("created.UpdatedAt is zero — time.Time scan from DATETIME failed")
	}
	if created.CreatedAt.Location() != time.UTC {
		t.Fatalf("created.CreatedAt location = %v, want UTC", created.CreatedAt.Location())
	}

	// An audit event for the "create" action must have been recorded.
	var auditCount int
	if err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM audit_events WHERE entity_id = $1 AND entity_type = 'card' AND action = 'create'`,
		created.ID,
	).Scan(&auditCount); err != nil {
		t.Fatalf("count audit_events: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("audit_events for new card = %d, want 1", auditCount)
	}

	// CreateCard now also bumps user_stats.card_count (Phase 5: trigger logic
	// ported to Go). Confirms the wiring end-to-end, not just the unit helper.
	var cardCount int
	if err := s.DB.QueryRow(
		`SELECT card_count FROM user_stats WHERE user_id = $1`, userID,
	).Scan(&cardCount); err != nil {
		t.Fatalf("read user_stats.card_count: %v", err)
	}
	if cardCount != 1 {
		t.Fatalf("user_stats.card_count = %d, want 1", cardCount)
	}

	// Re-read via GetFullCard to prove the read path scans cleanly too.
	fetched, err := GetFullCard(s.DB, userID, created.ID)
	if err != nil {
		t.Fatalf("GetFullCard: %v", err)
	}
	if fetched.ID != created.ID || fetched.Title != created.Title || fetched.Body != created.Body {
		t.Fatalf("GetFullCard round-trip mismatch: got %+v, want %+v", fetched, created)
	}
	if !fetched.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("GetFullCard CreatedAt = %v, want %v", fetched.CreatedAt, created.CreatedAt)
	}
}

// TestCreateCardSQLiteStructuredData proves JSONB->TEXT structured_data
// round-trips through the real CreateCard/GetFullCard path into a
// *json.RawMessage (the spike probe already showed the scan works in
// isolation; this confirms it inside the handler-adjacent flow).
func TestCreateCardSQLiteStructuredData(t *testing.T) {
	t.Setenv("ZETTEL_IS_TESTING", "true")

	s := loadSpikeSchema(t)
	userID := seedUser(t, s)

	payload := map[string]any{"summary": "hello", "count": 3.0}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	sd := json.RawMessage(raw)

	params := models.EditCardParams{
		CardID:         "structured",
		Title:          "Structured Spike",
		Body:           "card with structured_data",
		StructuredData: &sd,
	}

	created, err := CreateCard(s.DB, userID, params)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	fetched, err := GetFullCard(s.DB, userID, created.ID)
	if err != nil {
		t.Fatalf("GetFullCard: %v", err)
	}
	if fetched.StructuredData == nil {
		t.Fatalf("StructuredData is nil; want the round-tripped JSON")
	}
	var got map[string]any
	if err := json.Unmarshal(*fetched.StructuredData, &got); err != nil {
		t.Fatalf("unmarshal structured_data %q: %v", string(*fetched.StructuredData), err)
	}
	if got["summary"] != "hello" || got["count"] != 3.0 {
		t.Fatalf("structured_data round-trip = %+v, want summary=hello count=3", got)
	}
}
