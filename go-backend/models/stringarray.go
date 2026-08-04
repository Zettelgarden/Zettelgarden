package models

import (
	"database/sql/driver"
	"errors"
	"strings"
	"unicode/utf8"
)

// StringArray is a []string that scans from / values as a PostgreSQL
// one-dimensional text array literal, i.e. the "{a,b,c}" on-disk format.
//
// It is a drop-in replacement for github.com/lib/pq.StringArray, which was the
// last runtime dependency on lib/pq after the Postgres->SQLite cutover. The
// notifications.filter_tags column stores its values in exactly this "{a,b}"
// format (written historically by pq.StringArray and by the Phase-6b ETL, which
// passes the PG-array text through verbatim), so this type must read and write
// that format byte-for-byte the same way pq does — otherwise existing rows
// would become unreadable. The encode/decode rules below are the standard
// PostgreSQL array-literal format that lib/pq implements (see
// https://www.postgresql.org/docs/current/arrays.html#ARRAYS-IO).
//
// Quoting rule (matches Postgres/pq): an element is double-quoted if it is
// empty, equals "NULL", or contains any of , { } " \ or whitespace; inside
// quotes, \ and " are escaped with a leading backslash.
type StringArray []string

// Value implements driver.Valuer. A nil slice is written as SQL NULL; a
// non-nil slice (including the empty slice) is written as an array literal.
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, s := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		writeArrayElement(&b, s)
	}
	b.WriteByte('}')
	return b.String(), nil
}

// Scan implements sql.Scanner. It accepts nil, []byte, or string in the
// "{a,b,c}" array-literal format produced by Postgres / pq.StringArray / Value
// above. A SQL NULL becomes a nil slice.
func (a *StringArray) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return errors.New("StringArray.Scan: unsupported source type")
	}
	elems, err := parseArray(s)
	if err != nil {
		return err
	}
	*a = elems
	return nil
}

// writeArrayElement appends one element to the array literal, quoting it when
// the PostgreSQL formatting rules require it.
func writeArrayElement(b *strings.Builder, s string) {
	if needsArrayQuoting(s) {
		b.WriteByte('"')
		for _, r := range s {
			if r == '"' || r == '\\' {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
		b.WriteByte('"')
	} else {
		b.WriteString(s)
	}
}

func needsArrayQuoting(s string) bool {
	if s == "" || strings.EqualFold(s, "NULL") {
		return true
	}
	for _, r := range s {
		switch r {
		case ',', '{', '}', '"', '\\':
			return true
		}
		if r <= ' ' {
			// Any whitespace / control rune (space, tab, newline, etc.) requires quoting.
			return true
		}
	}
	return false
}

// parseArray parses a PostgreSQL array literal ("{a,b,c}") into its elements,
// honoring double-quoted elements and backslash escapes. This mirrors the
// format produced by Value above and by lib/pq.StringArray.
func parseArray(s string) ([]string, error) {
	// Postgres strips leading/trailing whitespace around the literal; the value
	// itself is "{...}". Be lenient about surrounding spaces.
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("StringArray: empty array literal")
	}
	if s == "{}" {
		return []string{}, nil
	}
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return nil, errors.New("StringArray: malformed array literal (missing braces)")
	}
	body := s[1 : len(s)-1]

	var (
		elems []string
		buf   strings.Builder
		i     = 0
	)
	for i < len(body) {
		// Skip leading spaces before an element (Postgres formatting tolerance).
		for i < len(body) && body[i] == ' ' {
			i++
		}
		if i >= len(body) {
			break
		}
		if body[i] == '"' {
			// Quoted element: read until the closing unescaped quote.
			i++ // consume opening quote
			buf.Reset()
			closed := false
			for i < len(body) {
				c := body[i]
				if c == '\\' && i+1 < len(body) {
					// Backslash escape: take the next rune literally (Postgres
					// accepts \ before any character; only \" and \\ are produced
					// here, but \" \{ \} \, \\ all decode to the literal char).
					r, size := utf8.DecodeRuneInString(body[i+1:])
					buf.WriteRune(r)
					i += 1 + size
					continue
				}
				if c == '"' {
					closed = true
					i++ // consume closing quote
					break
				}
				buf.WriteByte(c)
				i++
			}
			if !closed {
				return nil, errors.New("StringArray: unterminated quoted element")
			}
			elems = append(elems, buf.String())
		} else {
			// Unquoted element: read until the next comma or end.
			buf.Reset()
			for i < len(body) && body[i] != ',' {
				buf.WriteByte(body[i])
				i++
			}
			val := strings.TrimSpace(buf.String())
			// Postgres treats an unquoted NULL token as SQL NULL. Our values are
			// never intentionally NULL (this is a []string), so normalize to the
			// empty string — matching lib/pq.StringArray, which cannot represent
			// NULL elements and decodes them as "".
			if strings.EqualFold(strings.TrimSpace(val), "NULL") {
				val = ""
			}
			elems = append(elems, val)
		}
		// Skip trailing spaces then expect a comma or end.
		for i < len(body) && body[i] == ' ' {
			i++
		}
		if i < len(body) {
			if body[i] != ',' {
				return nil, errors.New("StringArray: expected comma between elements")
			}
			i++ // consume comma
		}
	}
	if elems == nil {
		elems = []string{}
	}
	return elems, nil
}
