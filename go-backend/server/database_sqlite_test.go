package server

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRunMigrationsSkipsSubdirectories guards the regression where a
// subdirectory under SchemaDir (e.g. schema/sqlite/ under the postgres scan
// root) was treated as a migration file and ioutil.ReadFile failed with "is a
// directory", fataling the whole server at boot. The runner must skip dirs.
func TestRunMigrationsSkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	// A legitimate migration file.
	if err := os.WriteFile(filepath.Join(dir, "0001-real.sql"),
		[]byte(`CREATE TABLE real (id INTEGER PRIMARY KEY AUTOINCREMENT);`), 0644); err != nil {
		t.Fatal(err)
	}
	// A subdirectory that must be ignored, not ReadFile'd.
	if err := os.Mkdir(filepath.Join(dir, "sqlite"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sqlite", "schema.sqlite.sql"),
		[]byte(`THIS IS NEVER READ; reading the dir would fatal the server`), 0644); err != nil {
		t.Fatal(err)
	}

	db := openMemSQLite(t)
	S := &Server{DB: db, Driver: "sqlite", SchemaDir: dir}
	RunMigrations(S) // must not log.Fatal on the "sqlite" subdir

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM real").Scan(&n); err != nil {
		t.Fatalf("real table not created (migration skipped?): %v", err)
	}
	var migCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&migCount); err != nil {
		t.Fatal(err)
	}
	if migCount != 1 {
		t.Fatalf("migrations recorded = %d, want 1 (the subdir must not be recorded)", migCount)
	}
}

// TestExecScriptSQLite exercises the SQLite branch of the migration runner's
// helper: a multi-statement script (including a string literal containing
// semicolons) is split and applied within a single transaction.
func TestExecScriptSQLite(t *testing.T) {
	db := openMemSQLite(t)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	script := `
		CREATE TABLE a (id INTEGER PRIMARY KEY, name TEXT);
		INSERT INTO a (name) VALUES ('x;y;z');  -- semicolons inside the string
		CREATE INDEX idx_a ON a(name);
	`
	if err := execScript(tx, "sqlite", script); err != nil {
		t.Fatalf("execScript: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM a").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("row count = %d, want 1", n)
	}
}

// TestRunMigrationsSQLite is the Phase 1 integration gate for the migration
// runner on SQLite: it bootstraps the migrations table, applies multi-statement
// migration files via the splitter, records applied migrations, and re-running
// is idempotent.
func TestRunMigrationsSQLite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0001-first.sql"), []byte(
		`CREATE TABLE cards (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL);`,
	), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0002-second.sql"), []byte(`
		CREATE TABLE tags (card_id INTEGER, tag TEXT);
		INSERT INTO cards (title) VALUES ('seed');
	`), 0644); err != nil {
		t.Fatal(err)
	}

	db := openMemSQLite(t)
	S := &Server{DB: db, Driver: "sqlite", SchemaDir: dir}

	RunMigrations(S)

	// Both migrations applied.
	var cardCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM cards").Scan(&cardCount); err != nil {
		t.Fatal(err)
	}
	if cardCount != 1 {
		t.Fatalf("cards = %d, want 1", cardCount)
	}
	var hasTags int
	if err := db.QueryRow(
		"SELECT 1 FROM sqlite_master WHERE type='table' AND name='tags'").Scan(&hasTags); err != nil {
		t.Fatal(err)
	}

	// Migrations recorded.
	var migCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&migCount); err != nil {
		t.Fatal(err)
	}
	if migCount != 2 {
		t.Fatalf("migrations = %d, want 2", migCount)
	}

	// Re-running is idempotent: the seed insert must not double.
	RunMigrations(S)
	if err := db.QueryRow("SELECT COUNT(*) FROM cards").Scan(&cardCount); err != nil {
		t.Fatal(err)
	}
	if cardCount != 1 {
		t.Fatalf("after re-run cards = %d, want 1 (runner not idempotent)", cardCount)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&migCount); err != nil {
		t.Fatal(err)
	}
	if migCount != 2 {
		t.Fatalf("after re-run migrations = %d, want 2", migCount)
	}
}
