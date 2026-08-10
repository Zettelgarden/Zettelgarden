package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// openSQLiteAt opens a SQLite DB at path via OpenSQLite and registers cleanup.
func openSQLiteAt(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("OpenSQLite(%q): %v", path, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// openMemSQLite opens a shared in-memory SQLite DB.
func openMemSQLite(t *testing.T) *sql.DB {
	t.Helper()
	return openSQLiteAt(t, ":memory:")
}

// openTempSQLite opens a file-backed SQLite DB in a per-test temp dir. Use this
// when testing WAL behaviour, which does not apply to in-memory databases.
func openTempSQLite(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	return openSQLiteAt(t, path), path
}

// TestOpenSQLiteCreatesParentDir locks in that OpenSQLite creates the parent
// directory of the DB file path. modernc opens the file read/write and errors
// if the dir is missing, so the default SQLITE_PATH ("./data/zettelgarden.db")
// would otherwise fatal on a fresh checkout.
func TestOpenSQLiteCreatesParentDir(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "a", "b", "c") // does not exist yet
	path := filepath.Join(nested, "test.db")

	db := openSQLiteAt(t, path)
	if _, err := db.Exec("CREATE TABLE x (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("write to newly-opened db: %v", err)
	}
	if fi, err := os.Stat(nested); err != nil || !fi.IsDir() {
		t.Errorf("parent dir not created: stat err=%v", err)
	}
}

func TestSQLitePragmasApplied(t *testing.T) {
	db, _ := openTempSQLite(t)

	checks := []struct {
		query string
		want  int
	}{
		// synchronous: 0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA
		{"PRAGMA synchronous", 1},
		{"PRAGMA busy_timeout", 5000},
		{"PRAGMA foreign_keys", 1},
	}
	for _, c := range checks {
		var got int
		if err := db.QueryRow(c.query).Scan(&got); err != nil {
			t.Fatalf("%s: %v", c.query, err)
		}
		if got != c.want {
			t.Errorf("%s = %d, want %d", c.query, got, c.want)
		}
	}

	// journal_mode returns a string; WAL only sticks on a file-backed DB.
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want \"wal\"", mode)
	}
}

// TestSQLiteForeignKeysEnforcedAcrossPool is the core D4 footgun test: it proves
// foreign_keys=ON is set on EVERY pooled connection, not just one. With the
// naive db.Exec("PRAGMA foreign_keys=ON") approach this test would fail for
// some goroutines (the insert would succeed on connections where FK is OFF).
func TestSQLiteForeignKeysEnforcedAcrossPool(t *testing.T) {
	db := openMemSQLite(t)

	if _, err := db.Exec(`CREATE TABLE parent (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE child (
		id INTEGER PRIMARY KEY,
		parent_id INTEGER NOT NULL,
		FOREIGN KEY (parent_id) REFERENCES parent(id)
	)`); err != nil {
		t.Fatal(err)
	}

	const workers = 32
	var enforced int64
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := db.Conn(ctx)
			if err != nil {
				t.Errorf("db.Conn: %v", err)
				return
			}
			defer conn.Close()
			// Insert a child referencing a parent id that does not exist.
			_, err = conn.ExecContext(ctx, "INSERT INTO child (parent_id) VALUES (999999)")
			if err != nil {
				atomic.AddInt64(&enforced, 1) // FK violation blocked it — correct
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt64(&enforced); got != workers {
		t.Fatalf("FK enforced on only %d/%d connections — per-connection pragma not applied",
			got, workers)
	}
}

// openTempSQLiteDeferred opens a file-backed SQLite DB via OpenSQLiteDeferred
// (deferred BEGIN — the test-harness variant).
func openTempSQLiteDeferred(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := OpenSQLiteDeferred(path)
	if err != nil {
		t.Fatalf("OpenSQLiteDeferred(%q): %v", path, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestSQLiteImmediateTakesWriteLockAtBegin locks in the production DSN's
// _txlock=immediate semantics: db.Begin() issues BEGIN IMMEDIATE, so the write
// lock is held from the very start of the transaction. This is what makes
// SQLITE_BUSY_SNAPSHOT (517) impossible — the snapshot can never go stale
// while this connection holds the write lock. WAL readers are NOT blocked.
func TestSQLiteImmediateTakesWriteLockAtBegin(t *testing.T) {
	db, _ := openTempSQLite(t) // OpenSQLite => _txlock=immediate
	ctx := context.Background()

	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	tx1, err := db.Begin() // BEGIN IMMEDIATE — acquires the write lock now
	if err != nil {
		t.Fatal(err)
	}
	defer tx1.Rollback()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Don't let the busy handler mask the lock state with a 5s wait.
	if _, err := conn.ExecContext(ctx, "PRAGMA busy_timeout=0"); err != nil {
		t.Fatal(err)
	}

	// WAL readers are never blocked by the held write lock.
	var n int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM t").Scan(&n); err != nil {
		t.Fatalf("read while write tx open: %v", err)
	}

	// A second connection's write transaction must fail busy immediately:
	// tx1's BEGIN IMMEDIATE already holds the write lock.
	_, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE")
	if !IsSQLiteBusy(err) {
		t.Fatalf("second BEGIN IMMEDIATE = %v, want SQLITE_BUSY (write lock must be held from Begin)", err)
	}
}

// TestSQLiteDeferredBeginTakesNoLock proves the OpenSQLiteDeferred variant
// (used by the test harness) keeps deferred BEGIN semantics: db.Begin() takes
// no lock, so another connection can still start a write transaction. The
// shared per-test transaction therefore never pins the write lock for the
// whole test, and pool-direct handler writes keep working.
func TestSQLiteDeferredBeginTakesNoLock(t *testing.T) {
	db := openTempSQLiteDeferred(t)
	ctx := context.Background()

	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	tx1, err := db.Begin() // deferred — no lock acquired
	if err != nil {
		t.Fatal(err)
	}
	defer tx1.Rollback()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Deferred tx1 holds nothing, so this must succeed.
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("BEGIN IMMEDIATE blocked by deferred tx1: %v", err)
	}
	if _, err := conn.ExecContext(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
}

// TestSQLiteBusySnapshotRetryable reproduces the production "517 database is
// locked" failure deterministically and verifies the two-part remedy:
//
//  1. IsSQLiteBusy classifies SQLITE_BUSY_SNAPSHOT (517) as busy — the old
//     `== SQLITE_BUSY` check compared against the primary code 5 and missed
//     it, so the sync push retry never fired and the push hard-failed.
//  2. Retrying the WHOLE transaction with a fresh snapshot succeeds. That is
//     the only remedy for a stale snapshot: busy_timeout cannot help because
//     waiting never makes the old snapshot upgradable.
//
// The scenario is manufactured on the deferred handle (a read snapshot is
// taken at first read, then another connection commits a write, then the first
// transaction tries to write). Production cannot hit it anymore — immediate
// txlock takes the write lock at BEGIN — but any deferred/legacy path must
// still recover via the retry.
func TestSQLiteBusySnapshotRetryable(t *testing.T) {
	db := openTempSQLiteDeferred(t)
	ctx := context.Background()

	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	tx1, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx1.Rollback()

	// First read pins the read snapshot for this transaction.
	var n int
	if err := tx1.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n); err != nil {
		t.Fatal(err)
	}

	// A concurrent writer commits while tx1's snapshot is open.
	conn2, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn2.ExecContext(ctx, `INSERT INTO t (v) VALUES ('concurrent')`); err != nil {
		t.Fatal(err)
	}
	conn2.Close()

	// tx1 tries to upgrade to a write: the snapshot is stale -> 517.
	_, err = tx1.Exec(`INSERT INTO t (v) VALUES ('stale')`)
	if err == nil {
		t.Fatal("expected SQLITE_BUSY_SNAPSHOT writing after a concurrent commit")
	}
	var se *sqlite.Error
	if !errors.As(err, &se) {
		t.Fatalf("error is %T, want *sqlite.Error", err)
	}
	if se.Code() != sqlite3.SQLITE_BUSY_SNAPSHOT {
		t.Fatalf("busy code = %d (%v), want %d (SQLITE_BUSY_SNAPSHOT)", se.Code(), err, sqlite3.SQLITE_BUSY_SNAPSHOT)
	}
	if !IsSQLiteBusy(err) {
		t.Fatalf("IsSQLiteBusy(%v) = false, want true: extended busy code must be classified", err)
	}

	// The fix: retry the whole transaction with a fresh snapshot.
	tx2, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx2.Rollback()
	if _, err := tx2.Exec(`INSERT INTO t (v) VALUES ('fresh')`); err != nil {
		t.Fatalf("fresh transaction after busy: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatal(err)
	}

	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != 2 { // 'concurrent' + 'fresh'; the stale insert was rolled back
		t.Fatalf("rows = %d, want 2", got)
	}
}

// TestSQLiteConcurrentWrites validates the D4 concurrency model under a
// realistic write mix (the Phase 1 spike deliverable): many goroutines writing
// concurrently must not produce "database is locked" errors, thanks to WAL +
// busy_timeout.
func TestSQLiteConcurrentWrites(t *testing.T) {
	db, _ := openTempSQLite(t)

	if _, err := db.Exec(`CREATE TABLE writes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		worker INTEGER NOT NULL,
		payload TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}

	const workers = 16
	const perWorker = 50
	var (
		wg     sync.WaitGroup
		errors int64
	)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				_, err := db.ExecContext(ctx,
					"INSERT INTO writes (worker, payload) VALUES ($1, $2)",
					worker, fmt.Sprintf("w%d-j%d", worker, j))
				if err != nil {
					atomic.AddInt64(&errors, 1)
					t.Logf("write error: %v", err)
				}
			}
		}(w)
	}
	wg.Wait()

	if errors != 0 {
		t.Fatalf("%d writes failed under concurrency (expected 0)", errors)
	}

	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM writes").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if want := workers * perWorker; n != want {
		t.Fatalf("row count = %d, want %d", n, want)
	}
}
