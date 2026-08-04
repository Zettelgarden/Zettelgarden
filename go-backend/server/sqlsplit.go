package server

import "strings"

// SplitSQL splits a SQL script into individual statements for drivers (such as
// modernc.org/sqlite) that execute only one statement per Exec call. It is
// quote/comment aware so that a semicolon inside a string literal, a quoted
// identifier, a line comment (--) or a block comment (/* */) does not terminate
// a statement.
//
// Comments are stripped from the output and whitespace-only statements are
// dropped. Each returned statement is trimmed of surrounding whitespace but
// retains its interior string literals and identifiers verbatim.
//
// Not handled: Postgres dollar-quoted function bodies ($$ ... $$). The
// consolidated SQLite schema contains none (trigger logic is ported to Go in
// Phase 5); the Postgres driver would parse multi-statement strings itself,
// but Postgres was retired after the cutover and the runner now only loads the
// SQLite schema. If the SQLite schema ever needs $$ support, add a dollar-tag
// state here.
func SplitSQL(script string) []string {
	var (
		stmts []string
		buf   strings.Builder
		runes = []rune(script)
		n     = len(runes)
		i     = 0
	)

	flush := func() {
		if s := strings.TrimSpace(buf.String()); s != "" {
			stmts = append(stmts, s)
		}
		buf.Reset()
	}

	skipSpace := func() {
		// Preserve token separation when a comment sits between two tokens:
		// e.g. "a-- c\nb" should not collapse to "ab". Only insert a separator
		// when the buffer is non-empty and does not already end in whitespace.
		if buf.Len() == 0 {
			return
		}
		s := buf.String()
		last := s[len(s)-1]
		if last == ' ' || last == '\t' || last == '\n' || last == '\r' {
			return
		}
		buf.WriteByte(' ')
	}

	for i < n {
		r := runes[i]

		// Line comment: -- ... through the terminating newline (inclusive).
		if r == '-' && i+1 < n && runes[i+1] == '-' {
			for i < n && runes[i] != '\n' {
				i++
			}
			if i < n {
				i++ // consume the newline
			}
			skipSpace()
			continue
		}

		// Block comment: /* ... */ (non-nested).
		if r == '/' && i+1 < n && runes[i+1] == '*' {
			i += 2
			for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			i += 2 // consume closing */ (or run past EOF harmlessly)
			if i > n {
				i = n
			}
			skipSpace()
			continue
		}

		// Statement terminator.
		if r == ';' {
			flush()
			i++
			continue
		}

		// Single-quoted string literal: '...' with '' as the only escape.
		if r == '\'' {
			buf.WriteByte('\'')
			i++
			for i < n {
				c := runes[i]
				if c == '\'' {
					if i+1 < n && runes[i+1] == '\'' {
						buf.WriteString("''") // escaped quote, stays inside the string
						i += 2
						continue
					}
					buf.WriteByte('\'') // closing quote
					i++
					break
				}
				buf.WriteRune(c)
				i++
			}
			continue
		}

		// Double-quoted identifier: "..." with "" as the only escape.
		if r == '"' {
			buf.WriteByte('"')
			i++
			for i < n {
				c := runes[i]
				if c == '"' {
					if i+1 < n && runes[i+1] == '"' {
						buf.WriteString(`""`)
						i += 2
						continue
					}
					buf.WriteByte('"')
					i++
					break
				}
				buf.WriteRune(c)
				i++
			}
			continue
		}

		buf.WriteRune(r)
		i++
	}

	flush() // final statement without a trailing semicolon
	return stmts
}
