package models

import (
	"strconv"
	"strings"
)

// InList returns a parenthesized, comma-separated list of n numbered SQL
// placeholders beginning at startIdx, e.g. InList(3, 2) returns "($3,$4)".
//
// It translates the historical Postgres "col = ANY($k)" array-binding queries
// to a form that runs on SQLite (which has no array type and no = ANY operator).
// modernc.org/sqlite binds $N positionally, so the expanded IN (...) query runs
// unchanged. (Driver gating is moot now that Postgres is retired.)
//
// Convention: assign fixed scalar args the LOW placeholder numbers ($1, $2, …)
// and the expanding slice the numbers AFTER them, so growing the slice never
// renumbers the fixed args. Example:
//
//	q := `SELECT ... WHERE user_id = $1 AND card_pk != $2 AND entity_id IN ` +
//	     models.InList(3, len(entityIDs)) + ` AND ...`
//	rows, err := db.Query(q, userID, sourceCardID, models.IntArgs(entityIDs)...)
//
// Callers must guard len(slice)==0 before building the query (an empty IN list
// is invalid SQL); if not guarded, InList returns "(NULL)" so the mistake is a
// loud no-match rather than a syntax error.
func InList(startIdx, n int) string {
	if n <= 0 {
		return "(NULL)"
	}
	var b strings.Builder
	b.WriteByte('(')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('$')
		b.WriteString(strconv.Itoa(startIdx + i))
	}
	b.WriteByte(')')
	return b.String()
}

// IntArgs converts an []int to []any so it can be spread as trailing variadic
// args into Query/Exec alongside fixed scalars:
//
//	db.Query(q, userID, sourceCardID, models.IntArgs(entityIDs)...)
func IntArgs(ids []int) []any {
	out := make([]any, len(ids))
	for i, v := range ids {
		out[i] = v
	}
	return out
}
