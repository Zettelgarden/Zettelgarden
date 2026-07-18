package sqliteparam

// Phase 1 scan/RETURNING probes: determine whether existing Go scan patterns
// (time.Time for timestamps, []byte for JSONB) work unchanged against SQLite
// TEXT columns, and whether RETURNING works. These outcomes govern Phase 3
// (query translation): if they work as-is, large classes of queries need no
// change beyond NOW() -> app-side time.

import (
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// (openMem is defined in sqlite_param_test.go; reused here.)

// TestTimestampScan documents WHY timestamp columns must be declared DATETIME,
// not TEXT (see D5): a TEXT-declared column is returned as a string and will
// NOT scan into *time.Time. The positive case (DATETIME columns scan fine for
// both RFC3339 and datetime('now') values) is covered by
// TestColumnTypeAffectsTimeScan in sqlite_columntype_probe_test.go.
func TestTimestampScan(t *testing.T) {
	db := openMem(t)
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE ts (id INTEGER PRIMARY KEY, created_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ts (id, created_at) VALUES (1, datetime('now'))`); err != nil {
		t.Fatal(err)
	}

	var dt time.Time
	err := db.QueryRow("SELECT created_at FROM ts WHERE id = 1").Scan(&dt)
	if err == nil {
		t.Fatal("expected TEXT column to FAIL scanning into time.Time, but it succeeded — D5 premise is wrong; revisit")
	}
	t.Logf("confirmed: TEXT-declared column does not scan into time.Time (%v) — declare DATETIME instead", err)
}

// Probe: can modernc scan a SQLite TEXT column holding JSON into []byte
// (the way the existing code reads JSONB columns)?
func TestJSONByteScan(t *testing.T) {
	db := openMem(t)
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE j (id INTEGER PRIMARY KEY, data TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO j (id, data) VALUES (1, '{"a":1,"b":"two"}')`); err != nil {
		t.Fatal(err)
	}

	var raw []byte
	if err := db.QueryRow("SELECT data FROM j WHERE id = 1").Scan(&raw); err != nil {
		t.Fatalf("scan JSON TEXT into []byte FAILED: %v", err)
	}
	t.Logf("JSON TEXT -> scanned OK into []byte: %s", string(raw))
	if string(raw) != `{"a":1,"b":"two"}` {
		t.Errorf("got %q, want the exact JSON bytes", string(raw))
	}
}

// Probe: does RETURNING work on modernc.org/sqlite (SQLite 3.35+)?
func TestReturning(t *testing.T) {
	db := openMem(t)
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE r (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`); err != nil {
		t.Fatal(err)
	}

	var id int64
	err := db.QueryRow(`INSERT INTO r (name) VALUES ('x') RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatalf("INSERT ... RETURNING id FAILED: %v", err)
	}
	t.Logf("RETURNING id -> OK, got %d", id)

	var id2 int64
	if err := db.QueryRow(`UPDATE r SET name = 'y' WHERE id = $1 RETURNING id`, id).Scan(&id2); err != nil {
		t.Errorf("UPDATE ... RETURNING with $1 FAILED: %v", err)
	} else {
		t.Logf("UPDATE ... RETURNING id -> OK, got %d", id2)
	}
}
