package server

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"github.com/google/uuid"
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
	// Local-first sync columns (epic Zettelgarden-v5b, Phase 0a). version is
	// the monotonic per-row concurrency counter for last-write-wins; sync_uuid
	// is the immutable sync identity (id is server-assigned, card_id is
	// user-editable, tags/tasks have int PKs — none safe for offline sync).
	{"cards", "version", "INTEGER DEFAULT 1"},
	{"cards", "sync_uuid", "TEXT"},
	{"tasks", "version", "INTEGER DEFAULT 1"},
	{"tasks", "sync_uuid", "TEXT"},
	{"tags", "version", "INTEGER DEFAULT 1"},
	{"tags", "sync_uuid", "TEXT"},
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
	// Drop the SaaS mailing-list tables (6er.11): the feature was removed from
	// the self-hosted app, so existing DBs that still carry the tables (fresh
	// builds no longer create them) drop them on boot. Idempotent no-op once
	// they're gone. Recipients references messages, so drop order matters.
	for _, tbl := range []string{"mailing_list_recipients", "mailing_list_messages", "mailing_list"} {
		if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tbl)); err != nil {
			return fmt.Errorf("drop mailing list table %s: %w", tbl, err)
		}
	}

	// Remove the half-implemented AI-agent multi-user remnants (Zettelgarden-thw).
	// The feature was never completed — no agent handlers/routes ever shipped —
	// so existing DBs carry an always-empty agent_activity_log table and
	// always-NULL columns (users.is_agent/owner_user_id/api_key_hash,
	// cards.created_by_agent_id). Fresh builds no longer create them; existing
	// DBs converge here on boot. SQLite cannot DROP COLUMN while a CHECK or FK
	// constraint references the column, so users and cards are rebuilt without
	// the agent columns (see sqliteRebuildTable). All steps are idempotent and
	// no-ops once the columns are gone.
	for _, idx := range []string{
		"idx_agent_activity_action", "idx_agent_activity_agent", "idx_agent_activity_created",
		"idx_cards_created_by_agent", "idx_users_agent", "idx_users_owner",
	} {
		if _, err := db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", idx)); err != nil {
			return fmt.Errorf("drop index %s: %w", idx, err)
		}
	}
	if _, err := db.Exec(`DROP TABLE IF EXISTS agent_activity_log`); err != nil {
		return fmt.Errorf("drop agent_activity_log: %w", err)
	}

	if hasUsers, err := sqliteTableExists(db, "users"); err != nil {
		return fmt.Errorf("check users table: %w", err)
	} else if hasUsers {
		hasAgentCol, err := sqliteColumnExists(db, "users", "is_agent")
		if err != nil {
			return fmt.Errorf("check users.is_agent: %w", err)
		}
		if hasAgentCol {
			// users rebuild: full consolidated column set minus the agent
			// columns (is_agent, owner_user_id, api_key_hash) and their CHECK
			// constraints. The email/oidc unique indexes are recreated by the
			// guarded self-heal steps below; caldav/last_memory_job_id indexes
			// are recreated here to match the consolidated schema.
			if err := sqliteRebuildTable(db,
				`CREATE TABLE users_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT,
  email TEXT,
  password TEXT,
  created_at DATETIME,
  updated_at DATETIME,
  is_admin BOOLEAN DEFAULT false NOT NULL,
  email_validated BOOLEAN DEFAULT false NOT NULL,
  stripe_customer_id TEXT,
  stripe_subscription_id TEXT,
  stripe_subscription_status TEXT,
  stripe_subscription_frequency TEXT,
  stripe_current_plan TEXT,
  last_login DATETIME,
  can_upload_files BOOLEAN DEFAULT true,
  max_file_storage INTEGER DEFAULT 100000000,
  dashboard_card_pk INTEGER DEFAULT 0,
  last_seen DATETIME,
  memory_has_changed BOOLEAN DEFAULT true,
  auth_provider TEXT DEFAULT 'local',
  github_id TEXT,
  has_seen_getting_started BOOLEAN DEFAULT false,
  stripe_cancel_at_period_end BOOLEAN DEFAULT false,
  timezone TEXT DEFAULT 'UTC',
  show_tasks BOOLEAN DEFAULT true,
  show_rss BOOLEAN DEFAULT true,
  last_memory_job_id INTEGER,
  caldav_url TEXT,
  caldav_token TEXT,
  oidc_provider TEXT,
  oidc_sub TEXT,
  FOREIGN KEY (last_memory_job_id) REFERENCES llm_jobs(id) ON DELETE SET NULL
)`, `INSERT INTO users_new (id, username, email, password, created_at, updated_at, is_admin, email_validated, stripe_customer_id, stripe_subscription_id, stripe_subscription_status, stripe_subscription_frequency, stripe_current_plan, last_login, can_upload_files, max_file_storage, dashboard_card_pk, last_seen, memory_has_changed, auth_provider, github_id, has_seen_getting_started, stripe_cancel_at_period_end, timezone, show_tasks, show_rss, last_memory_job_id, caldav_url, caldav_token, oidc_provider, oidc_sub) SELECT id, username, email, password, created_at, updated_at, is_admin, email_validated, stripe_customer_id, stripe_subscription_id, stripe_subscription_status, stripe_subscription_frequency, stripe_current_plan, last_login, can_upload_files, max_file_storage, dashboard_card_pk, last_seen, memory_has_changed, auth_provider, github_id, has_seen_getting_started, stripe_cancel_at_period_end, timezone, show_tasks, show_rss, last_memory_job_id, caldav_url, caldav_token, oidc_provider, oidc_sub FROM users`, `DROP TABLE users`, `ALTER TABLE users_new RENAME TO users`, `CREATE INDEX idx_users_caldav_token ON users (caldav_token) WHERE (caldav_token IS NOT NULL)`, `CREATE INDEX idx_users_last_memory_job_id ON users (last_memory_job_id)`); err != nil {
				return fmt.Errorf("rebuild users without agent columns: %w", err)
			}
			log.Printf("sqlite schema upgrade: dropped agent columns from users")
		}
	}

	if hasCards, err := sqliteTableExists(db, "cards"); err != nil {
		return fmt.Errorf("check cards table: %w", err)
	} else if hasCards {
		hasAgentCol, err := sqliteColumnExists(db, "cards", "created_by_agent_id")
		if err != nil {
			return fmt.Errorf("check cards.created_by_agent_id: %w", err)
		}
		if hasAgentCol {
			if err := sqliteRebuildTable(db,
				`CREATE TABLE cards_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  card_id TEXT,
  title TEXT,
  body TEXT,
  is_reference INTEGER DEFAULT 0,
  link TEXT,
  created_at DATETIME,
  updated_at DATETIME,
  user_id INTEGER,
  is_deleted BOOLEAN DEFAULT false,
  parent_id INTEGER,
  is_literature_card BOOLEAN DEFAULT false,
  is_flashcard BOOLEAN DEFAULT false,
  flashcard_state TEXT,
  flashcard_reps INTEGER DEFAULT 0,
  flashcard_lapses INTEGER DEFAULT 0,
  flashcard_last_review DATETIME,
  flashcard_due DATETIME,
  flashcard_stability REAL DEFAULT 0,
  flashcard_difficulty REAL DEFAULT 0,
  card_schema_id INTEGER,
  structured_data TEXT,
  version INTEGER DEFAULT 1,
  sync_uuid TEXT,
  FOREIGN KEY (card_schema_id) REFERENCES schema_definitions(id),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
)`, `INSERT INTO cards_new (id, card_id, title, body, is_reference, link, created_at, updated_at, user_id, is_deleted, parent_id, is_literature_card, is_flashcard, flashcard_state, flashcard_reps, flashcard_lapses, flashcard_last_review, flashcard_due, flashcard_stability, flashcard_difficulty, card_schema_id, structured_data, version, sync_uuid) SELECT id, card_id, title, body, is_reference, link, created_at, updated_at, user_id, is_deleted, parent_id, is_literature_card, is_flashcard, flashcard_state, flashcard_reps, flashcard_lapses, flashcard_last_review, flashcard_due, flashcard_stability, flashcard_difficulty, card_schema_id, structured_data, version, sync_uuid FROM cards`, `DROP TABLE cards`, `ALTER TABLE cards_new RENAME TO cards`, `CREATE INDEX idx_cards_card_schema_id ON cards (card_schema_id)`, `CREATE INDEX idx_cards_user_created ON cards (user_id, created_at) WHERE (is_deleted = 0)`); err != nil {
				return fmt.Errorf("rebuild cards without agent column: %w", err)
			}
			log.Printf("sqlite schema upgrade: dropped created_by_agent_id from cards")
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
	// sync_clients heartbeat table (v5b.5): needed by sync_log retention. Fresh
	// builds carry it in the consolidated schema; existing DBs get it here.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sync_clients (
		user_id INTEGER NOT NULL,
		device_id TEXT NOT NULL,
		cursor INTEGER NOT NULL DEFAULT 0,
		last_seen_at DATETIME NOT NULL,
		PRIMARY KEY (user_id, device_id)
	)`); err != nil {
		return fmt.Errorf("create sync_clients: %w", err)
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

	// Local-first sync (epic Zettelgarden-v5b, Phase 0a): create the sync_log
	// change feed (fresh builds get it from the consolidated schema; existing
	// DBs need it here) and back-fill sync_uuid for legacy rows. The columns
	// themselves are handled by sqliteSelfHealUpgrades above.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sync_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		collection TEXT NOT NULL,
		row_uuid TEXT NOT NULL,
		op TEXT NOT NULL,
		version INTEGER NOT NULL,
		created_at DATETIME DEFAULT (datetime('now')) NOT NULL
	)`); err != nil {
		return fmt.Errorf("create sync_log: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_sync_log_user_id ON sync_log (user_id, id)`); err != nil {
		return fmt.Errorf("create idx_sync_log_user_id: %w", err)
	}
	// sync_uuid uniqueness + legacy back-fill for the three synced tables.
	for _, table := range []string{"cards", "tasks", "tags"} {
		exists, err := sqliteTableExists(db, table)
		if err != nil {
			return fmt.Errorf("check %s table: %w", table, err)
		}
		if !exists {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_%s_sync_uuid ON %s (sync_uuid) WHERE sync_uuid IS NOT NULL`, table, table)); err != nil {
			return fmt.Errorf("create idx_%s_sync_uuid: %w", table, err)
		}
		// Back-fill needs the canonical id column; skip tables that lack it
		// (synthetic mini-schemas in tests) — real synced tables always have id.
		hasID, err := sqliteColumnExists(db, table, "id")
		if err != nil {
			return fmt.Errorf("check %s.id: %w", table, err)
		}
		if !hasID {
			continue
		}
		// One-time back-fill: legacy rows (and test fixtures) have NULL
		// sync_uuid; assign canonical UUIDs so every synced row has an
		// identity. Cheap after the first boot (no NULLs remain).
		rows, err := db.Query(fmt.Sprintf(`SELECT id FROM %s WHERE sync_uuid IS NULL`, table))
		if err != nil {
			return fmt.Errorf("select NULL-sync_uuid %s: %w", table, err)
		}
		var ids []int
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return fmt.Errorf("scan %s id: %w", table, err)
			}
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate %s ids: %w", table, err)
		}
		for _, id := range ids {
			if _, err := db.Exec(fmt.Sprintf(`UPDATE %s SET sync_uuid = $1 WHERE id = $2`, table), uuid.New().String(), id); err != nil {
				return fmt.Errorf("back-fill %s.%d sync_uuid: %w", table, id, err)
			}
		}
	}
	return nil
}

// sqliteRebuildTable runs the SQLite "recreate the table" procedure for a
// table that must lose columns SQLite cannot DROP COLUMN directly (columns
// referenced by CHECK or FK constraints). It temporarily disables foreign-key
// enforcement (required so DROP TABLE users does not cascade-delete children)
// and runs the statements atomically on a single dedicated connection:
//
//  1. CREATE TABLE <t>_new (...)         — new shape, no removed columns
//  2. INSERT INTO <t>_new SELECT ...     — copy all rows
//  3. DROP TABLE <t>
//  4. ALTER TABLE <t>_new RENAME TO <t> — SQLite rewrites FK references in
//     other tables to the renamed table
//
// plus any index recreation passed by the caller (indexes are dropped with
// the table). foreign_keys=ON is restored before returning (PRAGMA foreign_keys
// cannot change inside a transaction, so it is toggled around the BEGIN/COMMIT).
func sqliteRebuildTable(db *sql.DB, stmts ...string) error {
	conn, err := db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(context.Background(), `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	restoreFK := func() {
		_, _ = conn.ExecContext(context.Background(), `PRAGMA foreign_keys=ON`)
	}
	if _, err := conn.ExecContext(context.Background(), `BEGIN`); err != nil {
		restoreFK()
		return err
	}
	for _, s := range stmts {
		if _, err := conn.ExecContext(context.Background(), s); err != nil {
			_, _ = conn.ExecContext(context.Background(), `ROLLBACK`)
			restoreFK()
			return fmt.Errorf("rebuild statement failed: %w\n  statement: %s", err, s)
		}
	}
	if _, err := conn.ExecContext(context.Background(), `COMMIT`); err != nil {
		restoreFK()
		return err
	}
	restoreFK()
	return nil
}

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
