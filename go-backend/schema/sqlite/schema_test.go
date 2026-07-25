package sqlite_schema_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-backend/server"
)

// schemaFile is the consolidated SQLite schema produced by translate.py.
const schemaFile = "schema.sqlite.sql"

// findSchema locates schema.sqlite.sql relative to this test file
// (.../go-backend/schema/sqlite/schema.sqlite.sql).
func findSchema(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Tests run from the package dir, so the file is right here.
	p := filepath.Join(cwd, schemaFile)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("cannot find %s at %s: %v", schemaFile, p, err)
	}
	return p
}

// TestConsolidatedSchemaLoads is the Phase 2 acceptance gate:
//
//   - the consolidated schema splits cleanly via server.SplitSQL (the Phase 1
//     statement splitter — no hand-fed statements),
//   - every statement applies against a fresh :memory: SQLite DB with the D4
//     pragmas (foreign_keys=ON applied per-connection by OpenSQLite),
//   - PRAGMA foreign_key_check passes (no dangling FK references), and
//   - PRAGMA integrity_check reports 'ok'.
//
// See docs/plans/2026-07-17-postgres-to-sqlite-migration-design.md Phase 2.
func TestConsolidatedSchemaLoads(t *testing.T) {
	raw, err := os.ReadFile(findSchema(t))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	stmts := server.SplitSQL(string(raw))
	if len(stmts) == 0 {
		t.Fatal("SplitSQL returned no statements")
	}
	t.Logf("schema split into %d statements", len(stmts))

	db, err := server.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Apply each statement individually so a failure reports exactly which one.
	// All are DDL (CREATE TABLE / CREATE INDEX) plus the leading PRAGMA.
	var (
		tables, indexes int
		failed          int
	)
	for i, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			failed++
			t.Errorf("statement %d failed: %v\n  %s", i, err, firstLine(s))
		}
		head := strings.TrimSpace(s)
		switch {
		case strings.HasPrefix(strings.ToUpper(head), "CREATE TABLE "):
			tables++
		case strings.HasPrefix(strings.ToUpper(head), "CREATE INDEX"),
			strings.HasPrefix(strings.ToUpper(head), "CREATE UNIQUE INDEX"):
			indexes++
		}
	}
	if failed > 0 {
		t.Fatalf("%d statements failed to apply", failed)
	}

	t.Logf("created %d tables, %d indexes", tables, indexes)
	if tables == 0 {
		t.Fatal("no tables were created")
	}

	// foreign_key_check returns one row per FK violation (empty = clean).
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	violations := 0
	for rows.Next() {
		violations++
		var cols []any // best-effort: we only care that a row exists
		_ = rows.Scan(&cols)
	}
	rows.Close()
	if violations > 0 {
		t.Errorf("PRAGMA foreign_key_check reported %d FK violations", violations)
	}

	var integrity string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		t.Fatalf("integrity_check: %v", err)
	}
	if integrity != "ok" {
		t.Errorf("integrity_check = %q, want \"ok\"", integrity)
	}
}

// TestConsolidatedSchemaUUIDDefaultAndBooleanDefaults sanity-checks the two
// translated defaults most likely to silently misbehave: the gen_random_uuid()
// -> SQLite UUIDv4 expression on chat_conversations.id, and a BOOLEAN DEFAULT
// round-trip. These exercised end-to-end in the cards spike; here they guard
// the consolidated schema directly.
func TestConsolidatedSchemaUUIDDefaultAndBooleanDefaults(t *testing.T) {
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

	// Need a parent user row first (chat_conversations.user_id FK -> users).
	res, err := db.Exec(`INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`,
		"u", "u@example.com", "x")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	uid, _ := res.LastInsertId()

	// chat_conversations.id is a uuid PK defaulted via the translated expression.
	var convID string
	if err := db.QueryRow(
		`INSERT INTO chat_conversations (user_id, title) VALUES ($1, $2) RETURNING id`,
		uid, "test").Scan(&convID); err != nil {
		t.Fatalf("insert chat_conversation relying on uuid default: %v", err)
	}
	// Expect 8-4-4-4-12 hyphenated hex.
	if len(convID) != 36 || strings.Count(convID, "-") != 4 {
		t.Errorf("generated id %q is not a hyphenated UUID", convID)
	}

	// Boolean default: a fresh card has is_deleted=false stored as 0.
	var isDeleted bool
	if err := db.QueryRow(
		`INSERT INTO cards (user_id, title) VALUES ($1, $2) RETURNING is_deleted`,
		uid, "c").Scan(&isDeleted); err != nil {
		t.Fatalf("insert card: %v", err)
	}
	if isDeleted {
		t.Errorf("cards.is_deleted default = true, want false")
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
