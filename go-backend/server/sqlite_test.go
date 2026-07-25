package server

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
