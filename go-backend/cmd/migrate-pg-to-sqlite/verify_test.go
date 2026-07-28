package main

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestIsIntegerAffinity(t *testing.T) {
	cases := map[string]bool{
		"INTEGER":     true,
		"integer":     true,
		"BIGINT":      true,
		"INT":         true,
		"TEXT":        false, // UUID columns are declared TEXT in the consolidated schema
		"BLOB":        false,
		"REAL":        false,
		"DATETIME":    false,
		"":            false,
	}
	for decl, want := range cases {
		if got := isIntegerAffinity(decl); got != want {
			t.Errorf("isIntegerAffinity(%q) = %v, want %v", decl, got, want)
		}
	}
}

// TestHasIntegerPKID distinguishes integer-PK tables (id stats should be
// computed) from UUID/text-PK tables (they must be skipped — this is the exact
// regression from the first live import, where chat_conversations/messages/
// tool_calls have a TEXT id that Postgres cannot MIN/MAX as a uuid).
func TestHasIntegerPKID(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	schema := `
CREATE TABLE int_pk (id INTEGER PRIMARY KEY AUTOINCREMENT, v TEXT);
CREATE TABLE uuid_pk (id TEXT PRIMARY KEY, v TEXT);
CREATE TABLE no_id   (other_id INTEGER PRIMARY KEY, v TEXT);
CREATE TABLE text_id_nonpk (id TEXT, v TEXT);  -- named id but not the PK
`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		t.Fatal(err)
	}

	cases := map[string]bool{
		"int_pk":         true,
		"uuid_pk":        false, // UUID/text PK — must be excluded
		"no_id":          false,
		"text_id_nonpk":  false,
	}
	for table, want := range cases {
		if got := hasIntegerPKID(ctx, db, table); got != want {
			t.Errorf("hasIntegerPKID(%q) = %v, want %v", table, got, want)
		}
	}
}
