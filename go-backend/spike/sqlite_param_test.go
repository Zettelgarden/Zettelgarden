package sqliteparam

// Spike probe (Phase 1): determine which SQL parameter styles modernc.org/sqlite
// accepts when called through the standard database/sql positional-args path
// (db.Query(sql, args...)), which is how every query in this repo is invoked.
//
// The pivotal question: does the driver bind positional []any args to $1-style
// numbered placeholders? If yes, the entire 1373-edit $N -> ? sweep (Phase 3) is
// unnecessary. This file is a throwaway spike artifact.

import (
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"
)

func openMem(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Mirror the production pragmas from the design doc (D4).
	for _, p := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(p); err != nil {
			t.Fatalf("pragma %q: %v", p, err)
		}
	}
	return db
}

func setup(t *testing.T) *sql.DB {
	t.Helper()
	db := openMem(t)
	if _, err := db.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO probe (id, name) VALUES (1, 'alpha'), (2, 'beta')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

// tryQuery runs `query` with positional args and reports whether it succeeds.
func tryQuery(db *sql.DB, query string, args ...any) (got string, err error) {
	row := db.QueryRow(query, args...)
	err = row.Scan(&got)
	return
}

func TestParamStyles(t *testing.T) {
	db := setup(t)
	defer db.Close()

	cases := []struct {
		name  string
		query string
		args  []any
	}{
		{"question-mark ordinal ?", "SELECT name FROM probe WHERE id = ?", []any{2}},
		{"numbered $1", "SELECT name FROM probe WHERE id = $1", []any{2}},
		{"numbered ?NNN (?1)", "SELECT name FROM probe WHERE id = ?1", []any{2}},
		{"two params $1,$2", "SELECT name FROM probe WHERE id IN ($1, $2) ORDER BY id DESC LIMIT 1", []any{2, 1}},
		{"out-of-order $2,$1", "SELECT name FROM probe WHERE id = $2", []any{99, 2}},
	}

	for _, c := range cases {
		got, err := tryQuery(db, c.query, c.args...)
		status := "OK"
		want := "beta"
		if err != nil {
			status = fmt.Sprintf("FAIL (%v)", err)
		} else if got != want {
			status = fmt.Sprintf("WRONG (got %q want %q)", got, want)
		}
		t.Logf("%-28s | %-46s | -> %s", c.name, c.query, status)
	}
}

// Named-arg path (different code path in database/sql — requires NamedArg, not
// plain []any). Included for completeness; the repo does not use NamedArgs.
func TestNamedStyles(t *testing.T) {
	db := setup(t)
	defer db.Close()

	for _, style := range []string{":id", "@id", "$id"} {
		q := fmt.Sprintf("SELECT name FROM probe WHERE id = %s", style)
		row := db.QueryRow(q, sql.Named("id", 2))
		var got string
		err := row.Scan(&got)
		status := "OK"
		if err != nil {
			status = fmt.Sprintf("FAIL (%v)", err)
		} else if got != "beta" {
			status = fmt.Sprintf("WRONG (got %q)", got)
		}
		t.Logf("named %-4s | %s | -> %s", style, q, status)
	}
}
