package server

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// sqlitePragmas are the D4 concurrency/safety settings. All of these are
// connection-scoped in SQLite (not database-scoped), so foreign_keys=ON in
// particular MUST be applied to every pooled connection — a bare
// db.Exec("PRAGMA foreign_keys=ON") only affects one pooled connection and
// leaves enforcement silently OFF on the rest of the pool. modernc.org/sqlite
// applies each _pragma=value DSN parameter on every new connection it opens,
// which is the correct, pool-safe mechanism.
var sqlitePragmas = []struct{ key, value string }{
	{"journal_mode", "WAL"},
	{"synchronous", "NORMAL"},
	{"busy_timeout", "5000"},
	{"foreign_keys", "ON"},
}

// SQLitePoolSize caps the number of open connections. WAL allows concurrent
// readers alongside a single writer, and busy_timeout makes contending writers
// wait rather than error. A small pool is plenty for the single-user target and
// keeps lock contention bounded; see TestSQLiteConcurrentWrites for the
// validation that this does not produce "database is locked" under a realistic
// handler+scheduler+jobrunner write mix.
const SQLitePoolSize = 8

// OpenSQLite opens a SQLite database file at path with the D4 pragmas applied
// to every pooled connection. Pass ":memory:" (or "") for an ephemeral
// in-process database that is shared across the pool (modernc shared-cache).
func OpenSQLite(path string) (*sql.DB, error) {
	dsn := buildSQLiteDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open %q: %w", path, err)
	}
	db.SetMaxOpenConns(SQLitePoolSize)
	db.SetMaxIdleConns(SQLitePoolSize)
	db.SetConnMaxLifetime(0) // SQLite file connections are cheap to keep

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping %q: %w", path, err)
	}
	return db, nil
}

// buildSQLiteDSN constructs a modernc.org/sqlite DSN with the D4 pragmas.
// ":memory:" is mapped to a shared in-memory database so that all pooled
// connections see the same data (otherwise each connection gets its own
// private, empty in-memory DB — a classic SQLite gotcha).
func buildSQLiteDSN(path string) string {
	base := path
	if base == "" || base == ":memory:" {
		base = "file::memory:?cache=shared"
	}
	parts := make([]string, 0, len(sqlitePragmas))
	for _, p := range sqlitePragmas {
		parts = append(parts, "_pragma="+p.key+"("+p.value+")")
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + strings.Join(parts, "&")
}
