package server

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
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
//
// For a file path, the parent directory is created (os.MkdirAll) if it does
// not already exist: modernc opens the DB file read/write and fails if the
// directory is missing, so the default SQLITE_PATH ("./data/zettelgarden.db")
// would otherwise fatal on a fresh checkout. In-memory paths have no parent
// dir and are left untouched.
//
// Write transactions BEGIN IMMEDIATE (_txlock=immediate): the write lock is
// acquired up front, so a read-then-write transaction can never hit
// SQLITE_BUSY_SNAPSHOT (517) — the "database is locked" production failure
// where busy_timeout cannot help because the stale snapshot can never be
// upgraded; the whole transaction must be retried. See IsSQLiteBusy and
// TestSQLiteBusySnapshotRetryable. Read-only transactions opt back into
// deferred BEGIN via &sql.TxOptions{ReadOnly: true} (see Handler.BeginReadTx)
// so they take no write lock at all.
func OpenSQLite(path string) (*sql.DB, error) {
	return openSQLite(path, "immediate")
}

// OpenSQLiteDeferred opens a SQLite database whose transactions stay deferred
// (BEGIN acquires no lock; the write lock is taken at the first write). The
// test harness uses this variant: the shared per-test transaction (Server.Tx)
// must NOT pin the write lock for the whole test, because handlers also write
// via the pool directly (s.DB) while it is open (see tests/conftest.go).
func OpenSQLiteDeferred(path string) (*sql.DB, error) {
	return openSQLite(path, "deferred")
}

func openSQLite(path, txlock string) (*sql.DB, error) {
	if err := ensureSQLiteParentDir(path); err != nil {
		return nil, fmt.Errorf("sqlite mkdir parent of %q: %w", path, err)
	}
	dsn := buildSQLiteDSN(path, txlock)
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

// ensureSQLiteParentDir creates the directory containing the SQLite file path
// (e.g. "./data" for "./data/zettelgarden.db"). MkdirAll is a no-op for an
// existing dir or for the current directory ("."), so bare filenames are fine.
func ensureSQLiteParentDir(path string) error {
	if path == "" || path == ":memory:" {
		return nil
	}
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

// buildSQLiteDSN constructs a modernc.org/sqlite DSN with the D4 pragmas and
// the requested _txlock mode. ":memory:" is mapped to a shared in-memory
// database so that all pooled connections see the same data (otherwise each
// connection gets its own private, empty in-memory DB — a classic SQLite
// gotcha).
func buildSQLiteDSN(path, txlock string) string {
	base := path
	if base == "" || base == ":memory:" {
		base = "file::memory:?cache=shared"
	}
	parts := make([]string, 0, len(sqlitePragmas)+1)
	for _, p := range sqlitePragmas {
		parts = append(parts, "_pragma="+p.key+"("+p.value+")")
	}
	if txlock != "" {
		parts = append(parts, "_txlock="+txlock)
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + strings.Join(parts, "&")
}

// IsSQLiteBusy reports whether err is a SQLite busy-class failure. modernc
// enables extended result codes on every connection, so a stale-snapshot write
// upgrade surfaces as SQLITE_BUSY_SNAPSHOT (517) rather than the primary
// SQLITE_BUSY (5) — a bare `== SQLITE_BUSY` comparison misses it (the bug that
// made sync pushes fail with 517 instead of retrying). All busy variants
// (BUSY=5, BUSY_RECOVERY=261, BUSY_SNAPSHOT=517, BUSY_TIMEOUT=773 — the code
// busy_timeout expiry surfaces as) share the low byte, so mask to the primary
// code; no other primary code has low byte 5, so the mask cannot over-match.
func IsSQLiteBusy(err error) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code()&0xff == sqlite3.SQLITE_BUSY
}
