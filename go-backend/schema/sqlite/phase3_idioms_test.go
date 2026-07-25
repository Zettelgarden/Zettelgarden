package sqlite_schema_test

import (
	"os"
	"testing"

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
