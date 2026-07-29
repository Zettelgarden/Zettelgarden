package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"go-backend/server"

	_ "modernc.org/sqlite"
)

func TestParsePGArray(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "[]"},   // empty input
		{"{}", "[]"}, // empty array
		{"{work,home}", `["work","home"]`},
		{"{single}", `["single"]`},
		{`{"a,b","c"}`, `["a,b","c"]`},               // comma inside a quoted element
		{`{"he said \"hi\""}`, `["he said \"hi\""]`}, // escaped quote
		{`{one,two,three}`, `["one","two","three"]`},
	}
	for _, c := range cases {
		got, err := parsePGArray(c.in)
		if err != nil {
			t.Errorf("parsePGArray(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parsePGArray(%q) = %s, want %s", c.in, got, c.want)
		}
	}
	if _, err := parsePGArray("not an array"); err == nil {
		t.Error("parsePGArray(missing braces) expected error, got nil")
	}
}

func TestNormalizeValue(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		val    interface{}
		isArr  bool
		want   interface{}
		expErr bool
	}{
		{"nil", nil, false, nil, false},
		{"int64", int64(42), false, int64(42), false},
		{"float64", float64(1.5), false, float64(1.5), false},
		{"bool true", true, false, int64(1), false},
		{"bool false", false, false, int64(0), false},
		{"string", "hello", false, "hello", false},
		{"bytes json", []byte(`{"a":1}`), false, []byte(`{"a":1}`), false}, // []byte stays []byte (BLOB; json.RawMessage-readable)
		{"time", now, false, now, false},
		{"pg array", "{a,b}", true, `{a,b}`, false}, // array text passed through verbatim (pq.StringArray format)
		{"array wrong type", int64(1), true, nil, true}, // arrays must arrive as string
		{"unsupported", uint(1), false, nil, true},
	}
	for _, c := range cases {
		got, err := normalizeValue(c.val, "", "col", c.isArr)
		if c.expErr {
			if err == nil {
				t.Errorf("%s: expected error, got %v", c.name, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		// time.Time compares by value, []byte already converted to string.
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", c.want) {
			t.Errorf("%s: got %v (%T), want %v (%T)", c.name, got, got, c.want, c.want)
		}
	}
}

// TestCopyTableSQLiteToSQLite exercises the core copy loop end-to-end against a
// small representative SQLite schema (covering INTEGER / TEXT / BOOLEAN / REAL /
// DATETIME / JSON-as-TEXT). It does NOT exercise the Postgres array path (that
// has no SQLite analogue and is covered by TestParsePGArray / TestNormalizeValue
// above). It also verifies idempotency: copying twice yields the same single
// row set (the per-table wipe+reload, not a doubling).
func TestCopyTableSQLiteToSQLite(t *testing.T) {
	ctx := context.Background()

	const schema = `
CREATE TABLE t (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  body TEXT,
  flag BOOLEAN,
  cost REAL,
  when_at DATETIME,
  meta TEXT,
  created_at DATETIME DEFAULT (datetime('now'))
);
CREATE TABLE u (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  t_id INTEGER REFERENCES t(id)
);
`
	src := freshDB(t)
	dst := freshDB(t)
	for _, db := range []*sql.DB{src, dst} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}

	// Seed the source with rows exercising every type.
	ts := time.Date(2026, 7, 28, 9, 30, 0, 0, time.UTC)
	_, err := src.Exec(`INSERT INTO t (id, body, flag, cost, when_at, meta) VALUES
		($1, $2, $3, $4, $5, $6),
		($7, $8, $9, $10, $11, $12)`,
		1, "first", true, 9.95, ts, []byte(`{"k":"v1"}`),
		2, "second", false, 0, time.Time{}, []byte(`{"k":"v2"}`))
	if err != nil {
		t.Fatalf("seed t: %v", err)
	}
	_, err = src.Exec(`INSERT INTO u (id, name, t_id) VALUES ($1,$2,$3), ($4,$5,$6)`,
		10, "alpha", 1, 20, "beta", 2)
	if err != nil {
		t.Fatalf("seed u: %v", err)
	}

	// Copy t and u. Use a dedicated dst connection with FK enforcement OFF —
	// this mirrors runCopy()'s production setup: the per-table wipe would
	// otherwise fail when re-copying a parent that an already-loaded child
	// still references. copyTable's contract is that the caller manages FKs.
	dstConn, err := dst.Conn(ctx)
	if err != nil {
		t.Fatalf("dst conn: %v", err)
	}
	defer dstConn.Close()
	if _, err := dstConn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		t.Fatalf("disable FKs: %v", err)
	}

	for _, tbl := range []string{"t", "u"} {
		n, err := copyTable(ctx, dstConn, src, "sqlite", tbl)
		if err != nil {
			t.Fatalf("copy %s: %v", tbl, err)
		}
		if n != 2 {
			t.Errorf("copy %s: copied %d rows, want 2", tbl, n)
		}
	}

	// Verify values survived the round-trip with correct types.
	var (
		id     int64
		body   string
		flag   int64
		cost   float64
		whenAt time.Time
		meta   string
	)
	row := dst.QueryRowContext(ctx, `SELECT id, body, flag, cost, when_at, meta FROM t WHERE id = 1`)
	if err := row.Scan(&id, &body, &flag, &cost, &whenAt, &meta); err != nil {
		t.Fatalf("scan row 1: %v", err)
	}
	if id != 1 || body != "first" || flag != 1 || cost != 9.95 {
		t.Errorf("row 1 scalars: id=%d body=%q flag=%d cost=%v", id, body, flag, cost)
	}
	if !whenAt.Equal(ts) {
		t.Errorf("row 1 when_at: got %v, want %v", whenAt, ts)
	}
	if meta != `{"k":"v1"}` {
		t.Errorf("row 1 meta (JSON as TEXT): got %q", meta)
	}

	// The zero-time row must round-trip too (modernc stores DATETIME values as
	// text; a zero time must not become NULL or error).
	var zBody string
	var zWhen time.Time
	if err := dst.QueryRowContext(ctx, `SELECT body, when_at FROM t WHERE id = 2`).Scan(&zBody, &zWhen); err != nil {
		t.Fatalf("scan row 2: %v", err)
	}
	if zBody != "second" {
		t.Errorf("row 2 body: got %q", zBody)
	}

	// Idempotency: copy t again -> still 2 rows (wipe+reload, not append).
	// FKs are OFF on dstConn, so the parent wipe does not trip the child refs.
	if n, err := copyTable(ctx, dstConn, src, "sqlite", "t"); err != nil {
		t.Fatalf("re-copy t: %v", err)
	} else if n != 2 {
		t.Errorf("re-copy t: %d rows, want 2", n)
	}
	var tCount int64
	if err := dst.QueryRowContext(ctx, `SELECT COUNT(*) FROM t`).Scan(&tCount); err != nil {
		t.Fatal(err)
	}
	if tCount != 2 {
		t.Errorf("after re-copy: t has %d rows, want 2", tCount)
	}

	// AUTOINCREMENT sequence must reflect the max copied PK so future inserts
	// (by the running server) do not collide. modernc/SQLite updates
	// sqlite_sequence automatically on explicit-PK inserts into AUTOINCREMENT
	// columns; assert it.
	var seq sql.NullInt64
	if err := dst.QueryRowContext(ctx, `SELECT seq FROM sqlite_sequence WHERE name = 't'`).Scan(&seq); err != nil {
		t.Fatalf("sqlite_sequence: %v", err)
	}
	if !seq.Valid || seq.Int64 < 2 {
		t.Errorf("sqlite_sequence for t = %v, want >= 2", seq)
	}

	// Re-enable FKs and confirm the load is referentially intact (the exact
	// gate the production ETL runs after every load).
	if _, err := dstConn.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("re-enable FKs: %v", err)
	}
	if viol, err := foreignKeyCheck(ctx, dst); err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	} else if len(viol) != 0 {
		t.Errorf("foreign_key_check reported %d violations: %v", len(viol), viol)
	}
}

// TestForeignKeyCheckClean confirms the verify helper reports no violations on
// a well-formed DB (and would catch a broken one).
func TestForeignKeyCheck(t *testing.T) {
	ctx := context.Background()
	db := freshDB(t)
	if _, err := db.Exec(`
CREATE TABLE parent (id INTEGER PRIMARY KEY);
CREATE TABLE child (id INTEGER PRIMARY KEY, pid INTEGER REFERENCES parent(id));
PRAGMA foreign_keys = OFF;
INSERT INTO child (id, pid) VALUES (1, 999); -- dangling
`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	viol, err := foreignKeyCheck(ctx, db)
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	if len(viol) == 0 {
		t.Fatal("expected an FK violation from the dangling child row, got none")
	}
}

// freshDB returns a uniquely-named in-memory SQLite DB so multiple tests (and
// multiple DBs within one test) do not collide on the process-wide
// ":memory:" shared cache.
func freshDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", uniqueMemoryDSN(t.Name()))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("ping sqlite: %v", err)
	}
	return db
}

var dsnCounter int

// uniqueMemoryDSN builds a per-call shared-cache in-memory DSN so that source
// and destination DBs in one test process never share state.
func uniqueMemoryDSN(tag string) string {
	dsnCounter++
	// _pragma foreign_keys(on) mirrors server.OpenSQLite's per-connection setup.
	return fmt.Sprintf("file:etltest_%d_%s?mode=memory&cache=shared&_pragma=foreign_keys(on)", dsnCounter, tag)
}

// Ensure server.SplitSQL is referenced (the migration runner uses it to load
// the consolidated schema; importing server here documents that dependency for
// readers of this test file).
var _ = server.SplitSQL
