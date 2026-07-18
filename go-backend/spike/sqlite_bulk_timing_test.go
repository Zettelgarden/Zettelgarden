package sqliteparam

// Spike probe (Phase 1): bulk-insert timing for modernc.org/sqlite.
//
// The Phase 6b ETL tool will import ~15k cards (+ entity/fact/tag sub-graph)
// from Postgres into SQLite in one shot. modernc.org/sqlite (pure Go, no CGO)
// is known to be materially slower than mattn/go-sqlite3 (CGO) on bulk inserts,
// so this probe measures real throughput on the *file-backed* path (not
// :memory:, which hides I/O cost) to decide:
//   * if modernc does ~15k simple card inserts in well under a minute or two,
//     keep modernc for the ETL tool and avoid pulling in CGO at all;
//   * if it is painfully slow, use CGO mattn/go-sqlite3 for the one-shot ETL
//     tool only (deleted in Phase 7b, so it never threatens the CGO-free
//     runtime goal).
//
// This is a throwaway spike artifact, not a load test. The 1k-card figure comes
// from the migration design doc Phase 1 checklist; the extrapolation to 15k is
// reported via b.N scaling and a log line.

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const bulkCardsSchema = `
CREATE TABLE cards (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	card_id TEXT,
	user_id INT,
	title TEXT,
	body TEXT,
	link TEXT,
	is_deleted BOOLEAN DEFAULT FALSE,
	created_at DATETIME,
	updated_at DATETIME,
	parent_id INT,
	structured_data TEXT
);
CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT);
`

// BenchmarkBulkInsertCards measures per-insert cost for the cards INSERT shape
// (the actual RETURNING-id query CreateCard uses) against a file-backed SQLite
// DB with the D4 pragmas. Run with:
//
//	go test ./spike/ -bench BenchmarkBulkInsertCards -benchtime=1000x
//
// 1000x => inserts 1000 cards; -benchtime=15000x would simulate the full cut.
func BenchmarkBulkInsertCards(b *testing.B) {
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "bulk.db")
	dsn := "file:" + dbPath +
		"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)" +
		"&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		b.Fatalf("ping: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(bulkCardsSchema); err != nil {
		b.Fatalf("schema: %v", err)
	}
	var userID int
	if err := db.QueryRow(`INSERT INTO users (username) VALUES ($1) RETURNING id`, "bulk").Scan(&userID); err != nil {
		b.Fatalf("seed user: %v", err)
	}

	const insert = `INSERT INTO cards
		(card_id, user_id, title, body, link, is_deleted, created_at, updated_at, parent_id, structured_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`

	b.ResetTimer()
	start := time.Now()
	for i := 0; i < b.N; i++ {
		now := time.Now().UTC()
		var id int
		// One transaction per insert mirrors the CreateCard path (each card is
		// its own committed write in the real flow). If this is too slow, the
		// ETL tool can wrap batches in a single tx — see the assertion below.
		tx, err := db.Begin()
		if err != nil {
			b.Fatalf("begin: %v", err)
		}
		if err := tx.QueryRow(insert,
			fmt.Sprintf("card-%d", i), userID, "title", "body", "", false, now, now, i, nil,
		).Scan(&id); err != nil {
			_ = tx.Rollback()
			b.Fatalf("insert: %v", err)
		}
		if err := tx.Commit(); err != nil {
			b.Fatalf("commit: %v", err)
		}
	}
	elapsed := time.Since(start)
	b.StopTimer()

	// Report an extrapolation so the 15k-card projection is explicit.
	perInsert := elapsed / time.Duration(b.N)
	b.Logf("inserted %d cards in %s (%s/card); 15k projection ≈ %s",
		b.N, elapsed.Round(time.Millisecond), perInsert,
		(perInsert * 15000).Round(10*time.Millisecond))
}
