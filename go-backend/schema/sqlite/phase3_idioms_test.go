package sqlite_schema_test

import (
	"os"
	"testing"

	"go-backend/models"
	"go-backend/server"
)

// The four translated idioms Phase 3 introduced/normalized must execute on
// SQLite against the consolidated schema. This guards the "compiles but fails at
// runtime" class of error for the mechanical NOW()->CURRENT_TIMESTAMP and
// ILIKE->LIKE sweeps, plus the hand-rewritten stats day-bucket expression and
// the ON CONFLICT upsert. See docs/plans/2026-07-17-postgres-to-sqlite-... P3.
func TestPhase3TranslatedIdiomsExecuteOnSQLite(t *testing.T) {
	raw, err := os.ReadFile(findSchema(t))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	db, err := server.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, s := range server.SplitSQL(string(raw)) {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("apply %s: %v", firstLine(s), err)
		}
	}

	// Seed user.
	ures, err := db.Exec(`INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`,
		"u", "u@example.com", "x")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	uid, _ := ures.LastInsertId()

	// 1. INSERT with CURRENT_TIMESTAMP (the NOW() sweep target).
	cres, err := db.Exec(`INSERT INTO cards (user_id, title, created_at, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, uid, "hi")
	if err != nil {
		t.Fatalf("INSERT ... CURRENT_TIMESTAMP: %v", err)
	}
	cid, _ := cres.LastInsertId()

	// 2. UPDATE ... SET updated_at = CURRENT_TIMESTAMP
	if _, err := db.Exec(`UPDATE cards SET title = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		"hi2", cid); err != nil {
		t.Fatalf("UPDATE ... CURRENT_TIMESTAMP: %v", err)
	}

	// 3. LIKE search (converted from ILIKE). SQLite LIKE is case-insensitive for
	// ASCII by default, matching Postgres ILIKE for this (English) data.
	var found int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cards WHERE user_id = $1 AND (title LIKE $2 OR body LIKE $2)`,
		uid, "%HI%").Scan(&found); err != nil {
		t.Fatalf("LIKE search: %v", err)
	}
	if found != 1 {
		t.Errorf("LIKE (ASCII case-insensitive) matched %d rows, want 1", found)
	}

	// 4. stats.go day-bucket expression: substr(cast(created_at as text),1,10).
	var days int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT substr(cast(created_at as text), 1, 10))
		FROM cards WHERE user_id = $1`, uid).Scan(&days); err != nil {
		t.Fatalf("substr day-bucket: %v", err)
	}
	if days != 1 {
		t.Errorf("day buckets = %d, want 1", days)
	}

	// 5. ON CONFLICT ... DO UPDATE SET updated_at = CURRENT_TIMESTAMP (the upsert
	// pattern used across facts/entities/etc.). Seed a tag first to satisfy the
	// card_tags FK, then upsert.
	if _, err := db.Exec(`INSERT INTO tags (user_id, name) VALUES ($1, $2)`, uid, "t1"); err != nil {
		t.Fatalf("seed tag: %v", err)
	}
	var tagID int64
	if err := db.QueryRow(`SELECT id FROM tags WHERE user_id = $1 AND name = $2`, uid, "t1").Scan(&tagID); err != nil {
		t.Fatalf("fetch tag id: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO card_tags (card_pk, tag_id)
		VALUES ($1, $2)
		ON CONFLICT (card_pk, tag_id) DO UPDATE SET tag_id = EXCLUDED.tag_id`,
		cid, tagID); err != nil {
		t.Fatalf("ON CONFLICT upsert: %v", err)
	}
}

// TestSimilarityInListAndAppSideReorder guards the Zettelgarden-amt translation:
// handlers/facts.go + handlers/entity.go used Postgres "col = ANY($k)" bound with
// pq.Array, plus "ORDER BY array_position($k, col)" to preserve a similarity-
// ranked input order. SQLite has neither construct. The fix expands the array
// to an IN-list via models.InList (positional $N placeholders bind on both
// drivers) and reorders the scanned rows app-side. This test proves (a) the
// IN-list query executes on SQLite (no "no such function: ANY"), and (b) the
// app-side reorder restores the input/ranked order that array_position gave.
func TestSimilarityInListAndAppSideReorder(t *testing.T) {
	raw, err := os.ReadFile(findSchema(t))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	db, err := server.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, s := range server.SplitSQL(string(raw)) {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("apply %s: %v", firstLine(s), err)
		}
	}

	ures, err := db.Exec(`INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`,
		"u", "u@example.com", "x")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	uid, _ := ures.LastInsertId()

	cres, err := db.Exec(`INSERT INTO cards (user_id, title) VALUES ($1, $2)`, uid, "c")
	if err != nil {
		t.Fatalf("seed card: %v", err)
	}
	cid, _ := cres.LastInsertId()

	// Seed three facts; remember their auto-incremented ids.
	factText := map[int]string{}
	for _, txt := range []string{"alpha", "beta", "gamma"} {
		var id int64
		if err := db.QueryRow(`INSERT INTO facts (user_id, card_pk, fact) VALUES ($1, $2, $3) RETURNING id`,
			uid, cid, txt).Scan(&id); err != nil {
			t.Fatalf("seed fact %q: %v", txt, err)
		}
		factText[int(id)] = txt
	}

	// Build a deliberately non-sorted, non-id "ranked" input order (mimics the
	// similarity-ranked id slice FindSimilarFacts returns): gamma (3rd), then
	// alpha (1st), then beta (2nd).
	byText := map[string]int{}
	for id, txt := range factText {
		byText[txt] = id
	}
	ranked := []int{byText["gamma"], byText["alpha"], byText["beta"]}
	ids := ranked

	// The translated query: IN-list via models.InList, NO ORDER BY array_position.
	query := `SELECT id, fact FROM facts WHERE id IN ` + models.InList(1, len(ids))
	rows, err := db.Query(query, models.IntArgs(ids)...)
	if err != nil {
		t.Fatalf("IN-list query: %v", err)
	}
	byID := map[int]string{}
	for rows.Next() {
		var id int
		var fact string
		if err := rows.Scan(&id, &fact); err != nil {
			t.Fatalf("scan: %v", err)
		}
		byID[id] = fact
	}
	rows.Close()

	// App-side reorder: emit in the ranked input order.
	var got []string
	for _, id := range ids {
		if txt, ok := byID[id]; ok {
			got = append(got, txt)
		}
	}

	want := []string{"gamma", "alpha", "beta"} // matches ranked
	if len(got) != len(want) {
		t.Fatalf("reorder returned %d rows, want %d (got=%v want=%v)", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q (got=%v want=%v)", i, got[i], want[i], got, want)
		}
	}
}
