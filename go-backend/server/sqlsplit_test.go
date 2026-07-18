package server

import (
	"reflect"
	"testing"
)

func TestSplitSQL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "simple multiple statements",
			in:   "SELECT 1; SELECT 2;",
			want: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name: "trailing statement without semicolon",
			in:   "SELECT 1; SELECT 2",
			want: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name: "whitespace-only chunks dropped",
			in:   "  ; ; SELECT 1 ; ; ",
			want: []string{"SELECT 1"},
		},
		{
			name: "semicolon inside single-quoted string is preserved",
			in:   "INSERT INTO t VALUES ('a;b;c'); SELECT 1;",
			want: []string{"INSERT INTO t VALUES ('a;b;c')", "SELECT 1"},
		},
		{
			name: "escaped doubled quote '' inside string",
			in:   "INSERT INTO t VALUES ('it''s;a;b'); SELECT 1;",
			want: []string{"INSERT INTO t VALUES ('it''s;a;b')", "SELECT 1"},
		},
		{
			name: "semicolon inside double-quoted identifier",
			in:   `CREATE TABLE "we;rd" (x INT); SELECT 1;`,
			want: []string{`CREATE TABLE "we;rd" (x INT)`, "SELECT 1"},
		},
		{
			name: "line comment with semicolon is ignored",
			in:   "SELECT 1 -- this; is; a comment\n; SELECT 2;",
			want: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name: "block comment with semicolon is ignored",
			in:   "SELECT 1 /* a;b;c */; SELECT 2;",
			want: []string{"SELECT 1", "SELECT 2"},
		},
		{
			name: "comment-only chunks dropped",
			in:   "-- leading comment\nSELECT 1; -- trailing comment",
			want: []string{"SELECT 1"},
		},
		{
			name: "comment preserves token separation",
			in:   "SELECT--c\n1;",
			want: []string{"SELECT 1"},
		},
		{
			name: "realistic create table with default string containing semicolon",
			in:   "CREATE TABLE cards (id INTEGER PRIMARY KEY, title TEXT DEFAULT 'none; here');\n" +
				"CREATE INDEX idx_cards_title ON cards(title);",
			want: []string{
				"CREATE TABLE cards (id INTEGER PRIMARY KEY, title TEXT DEFAULT 'none; here')",
				"CREATE INDEX idx_cards_title ON cards(title)",
			},
		},
		{
			name: "multiline statement preserved",
			in:   "CREATE TABLE x (\n  id INTEGER PRIMARY KEY,\n  name TEXT\n);",
			want: []string{"CREATE TABLE x (\n  id INTEGER PRIMARY KEY,\n  name TEXT\n)"},
		},
		{
			name: "empty input",
			in:   "",
			want: nil,
		},
		{
			name: "block comment runs to EOF without closing",
			in:   "SELECT 1; /* never closed",
			want: []string{"SELECT 1"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SplitSQL(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("SplitSQL(%q)\n  got:  %#v\n  want: %#v", c.in, got, c.want)
			}
		})
	}
}

// TestSplitSQLRoundTripLoadSchema loads a synthetic but representative
// consolidated schema (the kind Phase 2 will produce) and asserts that every
// split statement is individually executable by modernc.org/sqlite.
func TestSplitSQLRoundTripLoadSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB-backed splitter test in short mode")
	}
	// see sqlite_test.go for openSQLite / pragma setup reuse
	db := openMemSQLite(t)
	defer db.Close()

	schema := `
-- A representative consolidated SQLite schema fragment.
CREATE TABLE cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT 'untitled; card',
    data TEXT,                       -- JSON column stored as TEXT
    created_at TEXT DEFAULT (datetime('now'))
);
CREATE TABLE card_tags (
    card_id INTEGER NOT NULL,
    tag TEXT NOT NULL,
    FOREIGN KEY (card_id) REFERENCES cards(id) ON DELETE CASCADE
);
/* a block comment ; with semicolons ; that must not split */
CREATE INDEX idx_cards_title ON cards(title);
INSERT INTO cards (title) VALUES ('first'); -- seed
INSERT INTO cards (title) VALUES ('semi; colon');
`
	for _, stmt := range SplitSQL(schema) {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec split statement failed:\n  stmt: %s\n  err:  %v", stmt, err)
		}
	}

	// Verify the seed inserts landed and FK cascade works (proves statements
	// were split correctly, not concatenated).
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM cards").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 cards, got %d", n)
	}

	// Cascade delete: removing a card should remove its tags via FK.
	if _, err := db.Exec("INSERT INTO card_tags (card_id, tag) VALUES (1, 'x')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DELETE FROM cards WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	var tags int
	if err := db.QueryRow("SELECT COUNT(*) FROM card_tags").Scan(&tags); err != nil {
		t.Fatal(err)
	}
	if tags != 0 {
		t.Fatalf("FK cascade did not fire; expected 0 card_tags, got %d", tags)
	}
}
