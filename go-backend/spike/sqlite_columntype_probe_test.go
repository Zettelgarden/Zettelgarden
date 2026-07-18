package sqliteparam

// Probe: does the SQLite *column declared type* change whether modernc returns
// time.Time (and thus whether Scan into *time.Time succeeds)? SQLite type
// affinity is loose, but modernc consults the declared type to decide the Go
// return type for "DATE"/"TIME"/"DATETIME"-ish columns.

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestColumnTypeAffectsTimeScan(t *testing.T) {
	cases := []struct {
		declared string // SQLite column declared type
	}{
		{"TEXT"},
		{"DATETIME"},
		{"TIMESTAMP"},
		{"DATE"},
	}

	rfc := time.Now().UTC().Format(time.RFC3339Nano)
	for _, c := range cases {
		t.Run(c.declared, func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec("CREATE TABLE t (c " + c.declared + ")"); err != nil {
				t.Fatal(err)
			}
			// Store an RFC3339 string value.
			if _, err := insertReturning(db, rfc); err != nil {
				t.Fatal(err)
			}
			var got time.Time
			err = db.QueryRow("SELECT c FROM t").Scan(&got)
			if err != nil {
				t.Logf("declared=%-10s scan into time.Time FAILED: %v", c.declared, err)
				return
			}
			t.Logf("declared=%-10s scan into time.Time OK -> %v", c.declared, got)
		})
	}
}

func insertReturning(db *sql.DB, v string) (int64, error) {
	var id int64
	err := db.QueryRow("INSERT INTO t (c) VALUES (?) RETURNING rowid", v).Scan(&id)
	return id, err
}
