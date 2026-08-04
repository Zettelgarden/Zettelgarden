package server

import (
	"database/sql"
	"testing"
)

// createUsersWithoutOIDC simulates a pre-cutover SQLite database: a users
// table that predates the oidc_provider/oidc_sub columns (the exact state that
// produced the 2026-08-04 "no such column: oidc_provider" prod outage).
// openMemSQLite is provided by sqlite_test.go (shared in-memory helper).
func createUsersWithoutOIDC(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		email TEXT,
		auth_provider TEXT DEFAULT 'local'
	)`); err != nil {
		t.Fatalf("create users: %v", err)
	}
}

func TestSQLiteColumnExists(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)

	if got, err := sqliteColumnExists(db, "users", "email"); err != nil || !got {
		t.Fatalf("expected email column to exist (got=%v err=%v)", got, err)
	}
	if got, err := sqliteColumnExists(db, "users", "oidc_provider"); err != nil || got {
		t.Fatalf("expected oidc_provider to be ABSENT before upgrade (got=%v err=%v)", got, err)
	}
}

func TestEnsureSQLiteSchemaUpgrades_AddsMissingColumnsAndIndex(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)

	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("first upgrade: %v", err)
	}

	for _, col := range []string{"oidc_provider", "oidc_sub"} {
		got, err := sqliteColumnExists(db, "users", col)
		if err != nil {
			t.Fatalf("check %s: %v", col, err)
		}
		if !got {
			t.Fatalf("expected %s to exist after upgrade", col)
		}
	}

	// The partial unique index must be present.
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_users_oidc_sub'`,
	).Scan(&n); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected idx_users_oidc_sub to exist, got count=%d", n)
	}
}

func TestEnsureSQLiteSchemaUpgrades_Idempotent(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)

	// First run adds the columns; the second must be a clean no-op (this runs
	// on every app start, so idempotency is required).
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("first upgrade: %v", err)
	}
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("second upgrade (idempotency): %v", err)
	}
}

func TestEnsureSQLiteSchemaUpgrades_NoopWhenAlreadyPresent(t *testing.T) {
	db := openMemSQLite(t)
	// Simulate a fresh build from the consolidated schema: the columns already
	// exist. The upgrade must not error and must not duplicate anything.
	if _, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		email TEXT,
		oidc_provider TEXT,
		oidc_sub TEXT
	)`); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade on fresh schema: %v", err)
	}
	got, err := sqliteColumnExists(db, "users", "oidc_provider")
	if err != nil || !got {
		t.Fatalf("oidc_provider must still exist after noop upgrade (got=%v err=%v)", got, err)
	}
}

// TestEnsureSQLiteSchemaUpgrades_IndexIsEnforced guards the index definition:
// two distinct users with the same (provider, sub) must collide, while NULL
// oidc_sub rows (password/GitHub/API-key users) are excluded by the partial
// index.
func TestEnsureSQLiteSchemaUpgrades_NoopWhenTableAbsent(t *testing.T) {
	// A DB with no users table at all (e.g. a partial/test schema, or one
	// where users will be created fresh later). The self-heal must skip
	// cleanly rather than fatal on ALTER of a missing table — a fresh create
	// carries the columns, so there is nothing to back-fill.
	db := openMemSQLite(t)
	if _, err := db.Exec(`CREATE TABLE unrelated (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create unrelated: %v", err)
	}
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade with no users table: %v", err)
	}
	hasUsers, err := sqliteTableExists(db, "users")
	if err != nil {
		t.Fatalf("check users: %v", err)
	}
	if hasUsers {
		t.Fatal("upgrade must not create the users table")
	}
}

func TestEnsureSQLiteSchemaUpgrades_IndexIsEnforced(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// First OIDC-linked row is fine.
	if _, err := db.Exec(
		`INSERT INTO users (email, oidc_provider, oidc_sub) VALUES ('a@x.com', 'pocket-id', 'sub-1')`,
	); err != nil {
		t.Fatalf("insert first oidc user: %v", err)
	}
	// Duplicate (provider, sub) must be rejected by the unique partial index.
	_, err := db.Exec(
		`INSERT INTO users (email, oidc_provider, oidc_sub) VALUES ('b@x.com', 'pocket-id', 'sub-1')`,
	)
	if err == nil {
		t.Fatal("expected unique violation on duplicate (oidc_provider, oidc_sub)")
	}
	// NULL oidc_sub rows must NOT collide (partial index excludes them).
	for _, email := range []string{"c@x.com", "d@x.com"} {
		if _, err := db.Exec(
			`INSERT INTO users (email, oidc_provider, oidc_sub) VALUES ($1, 'local', NULL)`, email,
		); err != nil {
			t.Fatalf("insert non-oidc user %s: %v", email, err)
		}
	}
}
