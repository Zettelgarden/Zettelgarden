package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-backend/models"
)

// quoteIdent double-quotes a SQL identifier, escaping embedded quotes. Both
// Postgres and SQLite use double-quoted identifiers, so this is portable.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// copyTable reads every row of `table` from src and writes it into the
// destination connection dst, preserving primary keys verbatim. The
// destination table is wiped (DELETE) first, so the operation is idempotent on
// re-runs. FK enforcement must be OFF on dst for the wipe to be safe across
// the FK graph (the caller acquires a dedicated connection with
// foreign_keys=OFF).
//
// srcKind is "postgres" or "sqlite": only under postgres do we translate array
// columns (PG text[] → JSON text); SQLite has no array type, so an array column
// is already stored as JSON TEXT and is copied verbatim.
//
// All writes for one table run in a single transaction on dst (one COMMIT per
// table — fast, and matches the Phase 1 bulk-import finding that modernc is
// comfortable with large transactions).
func copyTable(ctx context.Context, dst *sql.Conn, src models.Database, srcKind, table string) (int64, error) {
	qid := quoteIdent(table)

	rows, err := src.QueryContext(ctx, "SELECT * FROM "+qid)
	if err != nil {
		return 0, fmt.Errorf("select %s: %w", table, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, fmt.Errorf("columns %s: %w", table, err)
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return 0, fmt.Errorf("column types %s: %w", table, err)
	}

	// Detect PG array columns. lib/pq reports the database type for text[] as
	// "_text" (and "_int4" etc.) — the leading underscore is the PG convention
	// for array types. The consolidated schema has exactly one such column
	// (notifications.filter_tags), but detect generically.
	isArray := make([]bool, len(cols))
	if srcKind == "postgres" {
		for i, ct := range colTypes {
			if strings.HasPrefix(ct.DatabaseTypeName(), "_") {
				isArray[i] = true
			}
		}
	}

	// Build the INSERT. modernc.org/sqlite accepts $N placeholders natively
	// (Phase 1 finding), so the same placeholder numbering works regardless of
	// which driver is the source.
	placeholders := make([]string, len(cols))
	quoted := make([]string, len(cols))
	for i, c := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		quoted[i] = quoteIdent(c)
	}
	insertSQL := "INSERT INTO " + qid + " (" + strings.Join(quoted, ", ") +
		") VALUES (" + strings.Join(placeholders, ", ") + ")"

	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin %s: %w", table, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op after Commit

	// Idempotent wipe. With FKs OFF this is a plain per-table delete (no
	// cascades, no RESTRICT failures). On a freshly-schema'd DB it is a no-op.
	if _, err := tx.ExecContext(ctx, "DELETE FROM "+qid); err != nil {
		return 0, fmt.Errorf("delete %s: %w", table, err)
	}

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return 0, fmt.Errorf("prepare insert %s: %w", table, err)
	}
	defer stmt.Close()

	// Reusable scan buffers. lib/pq (and modernc) populate fresh values on
	// each Scan, so reusing the backing slice is safe.
	vals := make([]interface{}, len(cols))
	scanBuf := make([]interface{}, len(cols))
	for i := range scanBuf {
		scanBuf[i] = &vals[i]
	}

	var total int64
	for rows.Next() {
		if err := rows.Scan(scanBuf...); err != nil {
			return total, fmt.Errorf("scan %s: %w", table, err)
		}
		args := make([]interface{}, len(cols))
		for i := range vals {
			a, err := normalizeValue(vals[i], colTypes[i].DatabaseTypeName(), cols[i], isArray[i])
			if err != nil {
				return total, fmt.Errorf("normalize %s.%s: %w", table, cols[i], err)
			}
			args[i] = a
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return total, fmt.Errorf("insert %s (row %d): %w", table, total+1, err)
		}
		total++
	}
	if err := rows.Err(); err != nil {
		return total, fmt.Errorf("read %s: %w", table, err)
	}

	if err := tx.Commit(); err != nil {
		return total, fmt.Errorf("commit %s: %w", table, err)
	}
	return total, nil
}

// normalizeValue converts a value scanned from the source driver into a form
// the modernc SQLite driver will store correctly for the consolidated schema's
// column types (BOOLEAN, DATETIME, INTEGER, NUMERIC, REAL, TEXT — no BLOB
// columns exist, verified).
//
// Key conversions:
//   - bool → int64 0/1: SQLite has no native boolean; the consolidated schema
//     declares booleans as BOOLEAN (INTEGER affinity), and modernc does not
//     reliably accept a Go bool for an integer-affinity column.
//   - []byte → string: every []byte in this schema is JSON(B) text (there are
//     no bytea/BLOB columns). Storing as string preserves TEXT affinity and
//     keeps the value JSON-queryable; binding []byte would store a BLOB.
//   - time.Time → time.Time: modernc returns/stores time.Time for
//     DATETIME-declared columns (D5).
//   - PG array (text[]) → JSON text: the consolidated schema stores the one
//     array column (notifications.filter_tags) as JSON TEXT.
func normalizeValue(v interface{}, dbTypeName, colName string, isArray bool) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	if isArray {
		// lib/pq returns array text as []byte (not string) when scanning into
		// interface{}, so accept both.
		var s string
		switch x := v.(type) {
		case string:
			s = x
		case []byte:
			s = string(x)
		default:
			return nil, fmt.Errorf("array column %s: expected string or []byte from lib/pq, got %T", colName, v)
		}
		return parsePGArray(s)
	}
	switch x := v.(type) {
	case int64:
		return x, nil
	case float64:
		return x, nil
	case bool:
		if x {
			return int64(1), nil
		}
		return int64(0), nil
	case string:
		return x, nil
	case []byte:
		return string(x), nil
	case time.Time:
		return x, nil
	default:
		return nil, fmt.Errorf("unhandled Go type %T (db type %q) for column %s", v, dbTypeName, colName)
	}
}

// parsePGArray parses a PostgreSQL array in its text representation (the form
// lib/pq returns when scanning an array column into a plain interface{}) into a
// JSON-encoded string array, which is how the consolidated SQLite schema stores
// the lone array column (notifications.filter_tags).
//
// Examples:
//
//	{}            -> []
//	{work,home}   -> ["work","home"]
//	{"a,b","c"}   -> ["a,b","c"]   (commas inside quoted elements)
//	{"he said \"hi\""} -> ["he said \"hi\""]   (backslash escapes inside quotes)
//
// NULL elements (unquoted "NULL") are rare for tag arrays and are preserved as
// the literal string "NULL" — acceptable for filter_tags.
func parsePGArray(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "[]", nil
	}
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return "", fmt.Errorf("pg array literal missing braces: %q", s)
	}
	inner := s[1 : len(s)-1]
	if inner == "" {
		return "[]", nil
	}

	var elems []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		switch {
		case inQuote:
			switch {
			case c == '\\' && i+1 < len(inner):
				// Backslash escape: emit the next char literally (handles
				// \" and \\ as produced by Postgres array output).
				b.WriteByte(inner[i+1])
				i++
			case c == '"':
				inQuote = false
			default:
				b.WriteByte(c)
			}
		case c == '"':
			inQuote = true
		case c == ',':
			elems = append(elems, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	elems = append(elems, b.String())

	out, err := json.Marshal(elems)
	if err != nil {
		return "", fmt.Errorf("encode array as json: %w", err)
	}
	return string(out), nil
}
