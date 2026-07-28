package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"go-backend/models"
)

// verifyResult holds the per-table comparison for one table.
type verifyResult struct {
	Table   string
	PGCount int64
	Count   int64
	// PK min/max are only populated for tables with a single integer PK column
	// named "id" (the overwhelmingly common case). They catch off-by-one,
	// truncation, or duplicate-row errors that a count alone would miss.
	PGMin, PGMax any
	Min, Max     any
}

// Pass reports whether this table verifies cleanly.
func (r verifyResult) Pass() bool {
	if r.PGCount != r.Count {
		return false
	}
	return eq(r.PGMin, r.Min) && eq(r.PGMax, r.Max)
}

func eq(a, b any) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

// verifyCounts compares row counts (and, where there is an `id` column, the
// id min/max) between the source PG DB and the destination SQLite DB for every
// migrated table. Returns one verifyResult per table and the subset that did
// not match.
func verifyCounts(ctx context.Context, pg models.Database, sqlite models.Database, tables []string) ([]verifyResult, []verifyResult, error) {
	sort.Strings(tables)
	var results, mismatches []verifyResult
	for _, t := range tables {
		qid := quoteIdent(t)
		r := verifyResult{Table: t}

		if err := pg.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+qid).Scan(&r.PGCount); err != nil {
			return nil, nil, fmt.Errorf("pg count %s: %w", t, err)
		}
		if err := sqlite.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+qid).Scan(&r.Count); err != nil {
			return nil, nil, fmt.Errorf("sqlite count %s: %w", t, err)
		}

		// PK stats only when the table has an `id` column on the SQLite side
		// (cheap to detect and covers the common integer-PK case).
		if hasIDColumn(ctx, sqlite, t) {
			_ = pg.QueryRowContext(ctx, minMaxSQL(qid)).Scan(&r.PGMin, &r.PGMax)
			_ = sqlite.QueryRowContext(ctx, minMaxSQL(qid)).Scan(&r.Min, &r.Max)
		}

		results = append(results, r)
		if !r.Pass() {
			mismatches = append(mismatches, r)
		}
	}
	return results, mismatches, nil
}

// minMaxSQL returns a SELECT for the min and max of the `id` column, returning
// NULL when the table is empty (so it scans cleanly into *interface{}).
func minMaxSQL(qid string) string {
	return "SELECT MIN(id), MAX(id) FROM " + qid
}

// hasIDColumn reports whether the SQLite table has a column literally named
// "id". PRAGMA table_info is SQLite-specific; that's fine because we only use
// it against the SQLite side.
func hasIDColumn(ctx context.Context, sqlite models.Database, table string) bool {
	rows, err := sqlite.QueryContext(ctx, "PRAGMA table_info("+quoteIdent(table)+")")
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if strings.EqualFold(name, "id") {
			return true
		}
	}
	return false
}

// foreignKeyCheck runs SQLite's PRAGMA foreign_key_check (DB-wide, independent
// of which connection or whether FK enforcement is on) and returns any
// violations as human-readable strings. A clean migration has zero violations.
func foreignKeyCheck(ctx context.Context, sqlite models.Database) ([]string, error) {
	rows, err := sqlite.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return nil, fmt.Errorf("foreign_key_check: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var (
			table  string
			rowid  sql.NullInt64
			parent sql.NullString
			fkID   sql.NullInt64
		)
		if err := rows.Scan(&table, &rowid, &parent, &fkID); err != nil {
			return out, err
		}
		out = append(out, fmt.Sprintf("table=%s rowid=%v parent=%v fkid=%v", table, rowid, parent, fkID))
	}
	return out, rows.Err()
}
