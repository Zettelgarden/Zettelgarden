package server

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
)

func RunMigrations(S *Server) {
	// SQLite has no pre-existing migrations table to track what's applied.
	// Bootstrap it here so the runner is self-contained on the sqlite path.
	if S.Driver == "sqlite" {
		if _, err := S.DB.Exec(`CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			migration_name TEXT NOT NULL,
			applied_at TEXT DEFAULT (datetime('now'))
		)`); err != nil {
			log.Fatal(err)
		}
	}

	// Existence check only (the applied_at value is never read). SELECT 1 scans
	// cleanly on both drivers — selecting a timestamp into time.Time would fail
	// on SQLite, which stores applied_at as an ISO-8601 TEXT.
	queryString := "SELECT 1 FROM migrations WHERE migration_name = $1"
	insertString := "INSERT INTO migrations (migration_name) VALUES ($1)"

	files, err := ioutil.ReadDir(S.SchemaDir)
	if err != nil {
		log.Fatal(err)
	}

	var fileNames []string
	for _, file := range files {
		// Skip subdirectories (e.g. schema/sqlite/ under the postgres scan root).
		// ioutil.ReadFile on a directory fails with "is a directory"; ReadDir
		// returns both files and dirs, so this guard is required. Extensionless
		// migration files (e.g. 0025-add-chunk-text) are intentionally kept.
		if file.IsDir() {
			continue
		}
		// Skip Go source files. The SQLite schema dir (schema/sqlite/) co-locates
		// the consolidated schema with its Go test files; a migration runner must
		// never interpret .go source as SQL. The postgres scan root contains only
		// .sql / extensionless migration files, so this is a no-op there.
		if strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		fileNames = append(fileNames, file.Name())
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		var applied int
		err = S.DB.QueryRow(queryString, fileName).Scan(&applied)

		if err == sql.ErrNoRows {
			content, err := ioutil.ReadFile(S.SchemaDir + "/" + fileName)
			if err != nil {
				log.Fatal(err)
			}

			tx, err := S.DB.Begin()
			if err != nil {
				log.Fatal(err)
			}

			if err = execScript(tx, S.Driver, string(content)); err != nil {
				tx.Rollback()
				log.Fatal(err)
			}

			_, err = tx.Exec(insertString, fileName)
			if err != nil {
				tx.Rollback()
				log.Fatal(err)
			}

			err = tx.Commit()
			if err != nil {
				log.Fatal(err)
			}

			//	fmt.Println("Running migration:", fileName)
		} else if err != nil {
			log.Fatal(err)
		}
	}

	// SQLite has no incremental-migration story: the runner above only applies
	// the consolidated schema/sqlite/schema.sqlite.sql (for fresh builds) and
	// never scans the numbered Postgres migrations in ./schema (which use
	// non-portable syntax anyway). Self-heal known column/index gaps here so an
	// existing pre-cutover SQLite DB converges on the consolidated schema on
	// the next start — this is what was missing when OIDC went live
	// (2026-08-04: "no such column: oidc_provider").
	if S.Driver == "sqlite" {
		if err := ensureSQLiteSchemaUpgrades(S.DB); err != nil {
			log.Fatalf("sqlite schema upgrade failed: %v", err)
		}
	}
}

// sqliteColumnExists reports whether `column` exists on `table` via
// PRAGMA table_info. SQLite has no ALTER TABLE ... ADD COLUMN IF NOT EXISTS,
// so this guard is the idempotent equivalent needed for schema self-heal.
func sqliteColumnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, rows.Close()
		}
	}
	return false, rows.Err()
}

// sqliteTableExists reports whether `table` exists. Upgrades only target
// existing tables: if a table is absent it will be created fresh (with all
// current columns) by the consolidated schema, so there is nothing to
// back-fill. This also keeps the self-heal safe on partial/test schemas that
// never create the table.
func sqliteTableExists(db *sql.DB, table string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = $1`, table).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// sqliteSelfHealUpgrades lists the columns back-filled to existing SQLite DBs
// on every boot, grouped by table. It is the SQLite analogue of the (now
// archived) numbered Postgres migrations for pre-existing databases — fresh
// builds already get these columns from the consolidated schema.sqlite.sql.
//
// It is a package-level value (not a function-local) deliberately: the
// schema-sync drift test TestSelfHealListMatchesSchemaDelta reads this SAME
// definition and asserts it stays complete against the consolidated schema.
// That test guards the exact gap that caused the 2026-08-04 OIDC prod outage
// (the oidc columns were added to the consolidated schema for fresh builds
// but never to the self-heal, so existing DBs never received them).
var sqliteSelfHealUpgrades = []struct {
	table, column, decl string
}{
	{"users", "oidc_provider", "TEXT"},
	{"users", "oidc_sub", "TEXT"},
	{"users", "show_tasks", "BOOLEAN DEFAULT true"},
	{"users", "show_rss", "BOOLEAN DEFAULT true"},
}

// ensureSQLiteSchemaUpgrades applies idempotent, SQLite-only repairs for
// schema elements that the consolidated schema carries on fresh builds but
// that an existing (pre-cutover) SQLite database lacks. Each column addition
// is guarded (SQLite ADD COLUMN has no IF NOT EXISTS); indexes use IF NOT
// EXISTS directly. Add future gaps as an entry in sqliteSelfHealUpgrades —
// this runs on every start, so it must be cheap and idempotent.
func ensureSQLiteSchemaUpgrades(db *sql.DB) error {
	for _, u := range sqliteSelfHealUpgrades {
		// Skip tables that don't exist yet (a fresh create carries the
		// columns; back-fill is only for pre-existing tables).
		exists, err := sqliteTableExists(db, u.table)
		if err != nil {
			return fmt.Errorf("check table %s: %w", u.table, err)
		}
		if !exists {
			continue
		}
		colExists, err := sqliteColumnExists(db, u.table, u.column)
		if err != nil {
			return fmt.Errorf("check %s.%s: %w", u.table, u.column, err)
		}
		if colExists {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", u.table, u.column, u.decl)); err != nil {
			return fmt.Errorf("add %s.%s: %w", u.table, u.column, err)
		}
		log.Printf("sqlite schema upgrade: added column %s.%s", u.table, u.column)
	}
	// Drop orphaned summary_* analysis tables. These were retired when the
	// structured-analysis dimension was removed (qsg + fdi); fresh builds no
	// longer create them, but pre-cutover SQLite DBs still carry them. The
	// DROP is idempotent and a no-op once they're gone.
	for _, tbl := range []string{"summary_arguments", "summary_theses", "summary_sections"} {
		if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tbl)); err != nil {
			return fmt.Errorf("drop orphan table %s: %w", tbl, err)
		}
	}
	// Partial unique index for stable (provider, sub) OIDC re-auth. Only build
	// it when the users table exists; idempotent otherwise via IF NOT EXISTS.
	if hasUsers, err := sqliteTableExists(db, "users"); err != nil {
		return fmt.Errorf("check users table: %w", err)
	} else if hasUsers {
		if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_sub
			ON users (oidc_provider, oidc_sub) WHERE oidc_sub IS NOT NULL`); err != nil {
			return fmt.Errorf("create idx_users_oidc_sub: %w", err)
		}
	}
	// One email = one account (Zettelgarden-rbr): unique index on users.email.
	// Guarded so a pre-existing DB that already carries duplicates (produced by
	// the old check-then-insert signup race) does not crash boot: warn loudly
	// and defer enforcement until the operator dedupes — the next boot builds
	// the index.
	if hasUsers, err := sqliteTableExists(db, "users"); err != nil {
		return fmt.Errorf("check users table: %w", err)
	} else if hasUsers {
		var dupes int
		if err := db.QueryRow(`SELECT COUNT(*) FROM (SELECT email FROM users WHERE email IS NOT NULL GROUP BY email HAVING COUNT(*) > 1)`).Scan(&dupes); err != nil {
			return fmt.Errorf("check duplicate user emails: %w", err)
		}
		if dupes > 0 {
			log.Printf("WARNING: %d duplicate user emails block the unique users.email index (one email = one account); dedupe users and restart — Zettelgarden-rbr", dupes)
		} else if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email)`); err != nil {
			return fmt.Errorf("create idx_users_email: %w", err)
		}
	}
	return nil
}

// execScript executes a multi-statement SQL script within tx. modernc.org/sqlite
// executes only one statement per Exec, so the script is split into individual
// statements first (via SplitSQL) before execution. (The Postgres driver used
// to parse multi-statement strings itself, but Postgres was retired after the
// cutover; the `driver` arg is now always "sqlite" and retained only so the
// test harness can call this helper directly.)
func execScript(tx *sql.Tx, driver, script string) error {
	if driver == "sqlite" {
		for _, stmt := range SplitSQL(script) {
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("migration statement failed: %w\n  statement: %s", err, stmt)
			}
		}
		return nil
	}
	_, err := tx.Exec(script)
	return err
}
