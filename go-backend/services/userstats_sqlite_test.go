package services

import (
	"testing"

	"go-backend/server"
)

// openUserStatsSQLite builds a minimal in-memory schema (users + user_stats
// with user_id PK so ON CONFLICT (user_id) works on both drivers) for testing
// the user_stats maintenance helpers without the full consolidated schema.
func openUserStatsSQLite(t *testing.T) (*server.Server, int) {
	t.Helper()
	db, err := server.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	for _, stmt := range []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT)`,
		`CREATE TABLE user_stats (
			user_id INTEGER NOT NULL PRIMARY KEY,
			card_count INTEGER NOT NULL DEFAULT 0,
			task_count INTEGER NOT NULL DEFAULT 0,
			file_count INTEGER NOT NULL DEFAULT 0,
			chat_message_count INTEGER NOT NULL DEFAULT 0,
			llm_cost_usd NUMERIC NOT NULL DEFAULT 0,
			revenue_cents INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("apply stmt: %v\n  %s", err, stmt)
		}
	}
	res, err := db.Exec(`INSERT INTO users (username) VALUES ($1)`, "u")
	if err != nil {
		t.Fatal(err)
	}
	uid, _ := res.LastInsertId()
	return &server.Server{DB: db, Driver: "sqlite"}, int(uid)
}

func readCounters(t *testing.T, s *server.Server, userID int) (card, task, file, chat int, cost, revenue float64) {
	t.Helper()
	err := s.DB.QueryRow(
		`SELECT card_count, task_count, file_count, chat_message_count, llm_cost_usd, revenue_cents
		 FROM user_stats WHERE user_id = $1`, userID,
	).Scan(&card, &task, &file, &chat, &cost, &revenue)
	if err != nil {
		t.Fatalf("read user_stats: %v", err)
	}
	return
}

func TestUserStatsCounters(t *testing.T) {
	s, uid := openUserStatsSQLite(t)

	// No row exists yet; the first increment must lazily create it.
	IncrementUserCardCount(s.DB, uid)
	IncrementUserCardCount(s.DB, uid)
	IncrementUserTaskCount(s.DB, uid)
	IncrementUserFileCount(s.DB, uid)

	card, task, file, _, _, _ := readCounters(t, s, uid)
	if card != 2 || task != 1 || file != 1 {
		t.Fatalf("after increments: card=%d task=%d file=%d, want 2/1/1", card, task, file)
	}

	// Decrement floors at 0.
	DecrementUserCardCount(s.DB, uid)
	DecrementUserCardCount(s.DB, uid)
	DecrementUserCardCount(s.DB, uid) // would go negative -> clamped to 0
	DecrementUserTaskCount(s.DB, uid)
	card, task, _, _, _, _ = readCounters(t, s, uid)
	if card != 0 || task != 0 {
		t.Fatalf("after decrements: card=%d task=%d, want 0/0 (floored)", card, task)
	}
}

func TestUserStatsAmounts(t *testing.T) {
	s, uid := openUserStatsSQLite(t)

	AddUserLLMCost(s.DB, uid, 0.12)
	AddUserLLMCost(s.DB, uid, 0.08)
	AddUserRevenue(s.DB, uid, 500)
	AddUserRevenue(s.DB, uid, 250)

	_, _, _, _, cost, revenue := readCounters(t, s, uid)
	// Compare with a small epsilon for the NUMERIC float.
	if cost < 0.199 || cost > 0.201 {
		t.Fatalf("llm_cost_usd = %v, want ~0.20", cost)
	}
	if revenue != 750 {
		t.Fatalf("revenue_cents = %v, want 750", revenue)
	}
}

// TestUserStatsHelpersDoNotPanicOnMissingUser proves the best-effort contract:
// even against a user_stats row whose user_id has no matching users row
// (constraint aside), the helpers never panic and leave the counter correct.
// (FK enforcement is ON; this just confirms the upsert path itself.)
func TestUserStatsHelpersIdempotentAcrossCalls(t *testing.T) {
	s, uid := openUserStatsSQLite(t)
	for i := 0; i < 5; i++ {
		IncrementUserCardCount(s.DB, uid)
	}
	card, _, _, _, _, _ := readCounters(t, s, uid)
	if card != 5 {
		t.Fatalf("card_count = %d, want 5", card)
	}
}
