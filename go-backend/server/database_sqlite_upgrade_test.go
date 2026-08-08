package server

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// createUsersWithoutOIDC simulates a pre-cutover SQLite database: a users
// table that predates the oidc_provider/oidc_sub columns (the exact state that
// produced the 2026-08-04 "no such column: oidc_provider" prod outage).
// openMemSQLite is provided by sqlite_test.go (shared in-memory helper).
func createUsersWithoutOIDC(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		email TEXT,
		auth_provider TEXT DEFAULT 'local'
	)`); err != nil {
		t.Fatalf("create users: %v", err)
	}
}

func TestSQLiteColumnExists(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)

	if got, err := sqliteColumnExists(db, "users", "email"); err != nil || !got {
		t.Fatalf("expected email column to exist (got=%v err=%v)", got, err)
	}
	if got, err := sqliteColumnExists(db, "users", "oidc_provider"); err != nil || got {
		t.Fatalf("expected oidc_provider to be ABSENT before upgrade (got=%v err=%v)", got, err)
	}
}

func TestEnsureSQLiteSchemaUpgrades_AddsMissingColumnsAndIndex(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)

	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("first upgrade: %v", err)
	}

	for _, col := range []string{"oidc_provider", "oidc_sub"} {
		got, err := sqliteColumnExists(db, "users", col)
		if err != nil {
			t.Fatalf("check %s: %v", col, err)
		}
		if !got {
			t.Fatalf("expected %s to exist after upgrade", col)
		}
	}

	// The partial unique index must be present.
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_users_oidc_sub'`,
	).Scan(&n); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected idx_users_oidc_sub to exist, got count=%d", n)
	}

	// The unique email index (one email = one account) must also be present.
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_users_email'`,
	).Scan(&n); err != nil {
		t.Fatalf("query email index: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected idx_users_email to exist, got count=%d", n)
	}
}

func TestEnsureSQLiteSchemaUpgrades_Idempotent(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)

	// First run adds the columns; the second must be a clean no-op (this runs
	// on every app start, so idempotency is required).
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("first upgrade: %v", err)
	}
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("second upgrade (idempotency): %v", err)
	}
}

func TestEnsureSQLiteSchemaUpgrades_NoopWhenAlreadyPresent(t *testing.T) {
	db := openMemSQLite(t)
	// Simulate a fresh build from the consolidated schema: the columns already
	// exist. The upgrade must not error and must not duplicate anything.
	if _, err := db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		email TEXT,
		oidc_provider TEXT,
		oidc_sub TEXT
	)`); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade on fresh schema: %v", err)
	}
	got, err := sqliteColumnExists(db, "users", "oidc_provider")
	if err != nil || !got {
		t.Fatalf("oidc_provider must still exist after noop upgrade (got=%v err=%v)", got, err)
	}
}

// TestEnsureSQLiteSchemaUpgrades_IndexIsEnforced guards the index definition:
// two distinct users with the same (provider, sub) must collide, while NULL
// oidc_sub rows (password/GitHub/API-key users) are excluded by the partial
// index.
func TestEnsureSQLiteSchemaUpgrades_NoopWhenTableAbsent(t *testing.T) {
	// A DB with no users table at all (e.g. a partial/test schema, or one
	// where users will be created fresh later). The self-heal must skip
	// cleanly rather than fatal on ALTER of a missing table — a fresh create
	// carries the columns, so there is nothing to back-fill.
	db := openMemSQLite(t)
	if _, err := db.Exec(`CREATE TABLE unrelated (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create unrelated: %v", err)
	}
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade with no users table: %v", err)
	}
	hasUsers, err := sqliteTableExists(db, "users")
	if err != nil {
		t.Fatalf("check users: %v", err)
	}
	if hasUsers {
		t.Fatal("upgrade must not create the users table")
	}
}

func TestEnsureSQLiteSchemaUpgrades_IndexIsEnforced(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// First OIDC-linked row is fine.
	if _, err := db.Exec(
		`INSERT INTO users (email, oidc_provider, oidc_sub) VALUES ('a@x.com', 'pocket-id', 'sub-1')`,
	); err != nil {
		t.Fatalf("insert first oidc user: %v", err)
	}
	// Duplicate (provider, sub) must be rejected by the unique partial index.
	_, err := db.Exec(
		`INSERT INTO users (email, oidc_provider, oidc_sub) VALUES ('b@x.com', 'pocket-id', 'sub-1')`,
	)
	if err == nil {
		t.Fatal("expected unique violation on duplicate (oidc_provider, oidc_sub)")
	}
	// NULL oidc_sub rows must NOT collide (partial index excludes them).
	for _, email := range []string{"c@x.com", "d@x.com"} {
		if _, err := db.Exec(
			`INSERT INTO users (email, oidc_provider, oidc_sub) VALUES ($1, 'local', NULL)`, email,
		); err != nil {
			t.Fatalf("insert non-oidc user %s: %v", email, err)
		}
	}
}

// TestEnsureSQLiteSchemaUpgrades_EmailIndexIsEnforced guards the users.email
// unique index (Zettelgarden-rbr): duplicate emails collide, NULL emails do
// not (accounts without an email remain valid).
func TestEnsureSQLiteSchemaUpgrades_EmailIndexIsEnforced(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO users (email) VALUES ('a@x.com')`); err != nil {
		t.Fatalf("insert first user: %v", err)
	}
	_, err := db.Exec(`INSERT INTO users (email) VALUES ('a@x.com')`)
	if err == nil {
		t.Fatal("expected unique violation on duplicate email")
	}

	// NULL emails are distinct under the SQLite unique index.
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(`INSERT INTO users (email) VALUES (NULL)`); err != nil {
			t.Fatalf("insert null-email user %d: %v", i, err)
		}
	}
}

// TestEnsureSQLiteSchemaUpgrades_EmailIndexDeferredOnDupes guards the boot
// path against pre-existing duplicate emails (the old check-then-insert race):
// the upgrade must NOT crash; enforcement is deferred with a warning until the
// operator dedupes, and the next boot builds the index.
func TestEnsureSQLiteSchemaUpgrades_EmailIndexDeferredOnDupes(t *testing.T) {
	db := openMemSQLite(t)
	createUsersWithoutOIDC(t, db)
	// Seed the legacy duplicates.
	for _, email := range []string{"dup@x.com", "dup@x.com"} {
		if _, err := db.Exec(`INSERT INTO users (email) VALUES ($1)`, email); err != nil {
			t.Fatalf("seed duplicate: %v", err)
		}
	}
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade with duplicate emails must not fail: %v", err)
	}

	// Enforcement is deferred: the index does not exist yet.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_users_email'`).Scan(&n); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected idx_users_email deferred, got count=%d", n)
	}
}

// usersBaselineColumns is the FROZEN set of users-table columns as of the
// moment the SQLite self-heal mechanism was introduced (pre-OIDC, 2026-08-04).
// Existing production SQLite DBs that predate the self-heal have exactly these
// columns and no others. NEVER add new columns here — new columns go into
// sqliteSelfHealUpgrades, and TestSelfHealListMatchesSchemaDelta enforces the
// invariant (consolidated_schema - baseline) == self_heal_managed_set.
var usersBaselineColumns = []string{
	"id", "username", "email", "password", "created_at", "updated_at",
	"is_admin", "email_validated", "stripe_customer_id", "stripe_subscription_id",
	"stripe_subscription_status", "stripe_subscription_frequency", "stripe_current_plan",
	"last_login", "can_upload_files", "max_file_storage", "dashboard_card_pk",
	"last_seen", "memory_has_changed", "auth_provider", "github_id",
	"has_seen_getting_started", "stripe_cancel_at_period_end", "timezone",
	"last_memory_job_id", "caldav_url", "caldav_token", "is_agent",
	"owner_user_id", "api_key_hash",
}

// TestSelfHealListMatchesSchemaDelta is the SQLite schema-sync guard.
//
// It enforces the single-source-of-truth invariant for the users table: every
// column the consolidated schema (schema.sqlite.sql) carries BEYOND the frozen
// pre-self-heal baseline MUST have a matching entry in sqliteSelfHealUpgrades
// — and vice versa. This is the exact gap that caused the 2026-08-04 OIDC prod
// outage: oidc_provider/oidc_sub were added to the consolidated schema for
// fresh builds but not to the self-heal, so existing DBs never received them
// ("no such column: oidc_provider").
//
// If this test fails, either:
//   - you added a column to schema.sqlite.sql's users table without adding it
//     to sqliteSelfHealUpgrades (existing DBs will break on the next deploy) —
//     add the self-heal entry; or
//   - you added a self-heal entry without the matching consolidated-schema
//     column — add it to schema.sqlite.sql.
func TestSelfHealListMatchesSchemaDelta(t *testing.T) {
	consolidated := toSet(tableColumnNames(t, freshConsolidatedDB(t), "users"))
	baseline := toSet(usersBaselineColumns)

	var managed []string
	for _, u := range sqliteSelfHealUpgrades {
		if u.table == "users" {
			managed = append(managed, u.column)
		}
	}
	selfHealed := toSet(managed)

	// delta = columns a fresh build carries that a pre-self-heal existing DB lacks.
	delta := subtract(consolidated, baseline)

	if !equal(delta, selfHealed) {
		t.Fatalf("SQLite schema/self-heal DRIFT for users table.\n"+
			"  In consolidated schema but missing from self-heal "+
			"(add an entry to sqliteSelfHealUpgrades or existing DBs will break): %v\n"+
			"  In self-heal but missing from consolidated schema "+
			"(add the column to schema.sqlite.sql): %v",
			toSortedSlice(subtract(delta, selfHealed)),
			toSortedSlice(subtract(selfHealed, delta)))
	}
}

// freshConsolidatedDB builds an in-memory SQLite DB from the real production
// consolidated schema (schema/sqlite/schema.sqlite.sql) via RunMigrations, so
// the drift test reasons about the actual fresh-build schema rather than a
// hand-maintained copy. (RunMigrations also invokes the self-heal, but it is
// a no-op on a fresh build because the columns already exist.)
func freshConsolidatedDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openMemSQLite(t)
	S := &Server{DB: db, Driver: "sqlite", SchemaDir: findSchemaSqliteDir(t)}
	RunMigrations(S)
	return db
}

// tableColumnNames returns the column names of `table` via PRAGMA table_info.
func tableColumnNames(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		t.Fatalf("pragma table_info(%s): %v", table, err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info(%s): %v", table, err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err table_info(%s): %v", table, err)
	}
	return cols
}

func toSet(xs []string) map[string]struct{} {
	m := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		m[x] = struct{}{}
	}
	return m
}

func subtract(a, b map[string]struct{}) map[string]struct{} {
	r := make(map[string]struct{}, len(a))
	for k := range a {
		if _, ok := b[k]; !ok {
			r[k] = struct{}{}
		}
	}
	return r
}

func equal(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func toSortedSlice(m map[string]struct{}) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

// createAgentEraTables simulates a database from the half-implemented
// AI-agent era (Zettelgarden-thw) plus the disabled user-memory feature
// (Zettelgarden-c1x): users carries the frozen pre-self-heal baseline columns
// PLUS the agent columns (is_agent, owner_user_id, api_key_hash), their CHECK
// constraints, and the memory columns (memory_has_changed, last_memory_job_id);
// cards has created_by_agent_id; and the always-empty agent_activity_log and
// user_memories tables exist. Neither feature shipped to users, so these
// columns are all NULL/false and the tables are empty.
func createAgentEraTables(t *testing.T, db *sql.DB) {
	cols := []string{"id INTEGER PRIMARY KEY AUTOINCREMENT"}
	for _, c := range usersBaselineColumns[1:] {
		cols = append(cols, c+" TEXT")
	}
	cols = append(cols,
		"CONSTRAINT check_agent_has_api_key CHECK (((NOT is_agent) OR (api_key_hash IS NOT NULL)))",
		"CONSTRAINT check_agent_not_admin CHECK ((NOT ((is_agent = 1) AND (is_admin = 1))))",
		"FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE",
	)
	if _, err := db.Exec("CREATE TABLE users (" + strings.Join(cols, ", ") + ")"); err != nil {
		t.Fatalf("create agent-era users: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE cards (
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
		created_by_agent_id INTEGER,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (created_by_agent_id) REFERENCES users(id) ON DELETE SET NULL
	)`); err != nil {
		t.Fatalf("create agent-era cards: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE agent_activity_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id INTEGER,
		action TEXT NOT NULL,
		FOREIGN KEY (agent_id) REFERENCES users(id) ON DELETE SET NULL
	)`); err != nil {
		t.Fatalf("create agent_activity_log: %v", err)
	}
	// cards references schema_definitions in the real schema; provide it so
	// FK enforcement on the rebuilt cards table resolves.
	if _, err := db.Exec(`CREATE TABLE schema_definitions (id INTEGER PRIMARY KEY AUTOINCREMENT)`); err != nil {
		t.Fatalf("create schema_definitions: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE user_memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		memory TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	)`); err != nil {
		t.Fatalf("create user_memories: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (username, email, is_admin, email_validated) VALUES ('alice', 'alice@x.com', 0, 1)`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cards (title, user_id) VALUES ('hello', 1)`); err != nil {
		t.Fatalf("seed card: %v", err)
	}
}

// TestEnsureSQLiteSchemaUpgrades_DropsAgentRemnants guards the Zettelgarden-thw
// and Zettelgarden-c1x cleanups: a pre-cleanup SQLite DB (agent + memory
// columns present) converges on the consolidated schema on boot — agent and
// memory columns/tables dropped, data preserved, FK + CHECK constraints
// rebuilt, and a second run is a clean idempotent no-op.
func TestEnsureSQLiteSchemaUpgrades_DropsAgentRemnants(t *testing.T) {
	db := openMemSQLite(t)
	createAgentEraTables(t, db)

	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	for _, col := range []string{"is_agent", "owner_user_id", "api_key_hash", "memory_has_changed", "last_memory_job_id"} {
		got, err := sqliteColumnExists(db, "users", col)
		if err != nil || got {
			t.Fatalf("users.%s must be gone after upgrade (got=%v err=%v)", col, got, err)
		}
	}
	got, err := sqliteColumnExists(db, "cards", "created_by_agent_id")
	if err != nil || got {
		t.Fatalf("cards.created_by_agent_id must be gone (got=%v err=%v)", got, err)
	}
	exists, err := sqliteTableExists(db, "agent_activity_log")
	if err != nil || exists {
		t.Fatalf("agent_activity_log must be gone (exists=%v err=%v)", exists, err)
	}
	exists, err = sqliteTableExists(db, "user_memories")
	if err != nil || exists {
		t.Fatalf("user_memories must be gone (exists=%v err=%v)", exists, err)
	}
	// Agent + memory indexes gone.
	for _, idx := range []string{"idx_users_agent", "idx_users_owner", "idx_users_last_memory_job_id", "idx_cards_created_by_agent", "idx_agent_activity_action"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=$1`, idx).Scan(&n); err != nil || n != 0 {
			t.Fatalf("index %s must be gone (n=%d err=%v)", idx, n, err)
		}
	}
	// Non-agent/non-memory indexes recreated to match the consolidated schema.
	for _, idx := range []string{"idx_users_caldav_token", "idx_cards_card_schema_id", "idx_cards_user_created", "idx_users_oidc_sub", "idx_users_email"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=$1`, idx).Scan(&n); err != nil || n != 1 {
			t.Fatalf("index %s must exist after upgrade (n=%d err=%v)", idx, n, err)
		}
	}
	// Data preserved across the rebuild.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("users rows = %d (want 1), err=%v", n, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM cards`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("cards rows = %d (want 1), err=%v", n, err)
	}
	var email string
	if err := db.QueryRow(`SELECT email FROM users WHERE username='alice'`).Scan(&email); err != nil || email != "alice@x.com" {
		t.Fatalf("user data lost across rebuild (email=%q err=%v)", email, err)
	}
	// New rows still obey the remaining FK (user_id → users) after the rebuild.
	if _, err := db.Exec(`INSERT INTO cards (title, user_id) VALUES ('second', 1)`); err != nil {
		t.Fatalf("insert card after rebuild: %v", err)
	}

	// Idempotent: a second run is a clean no-op (self-heal runs on every boot).
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("second upgrade (idempotency): %v", err)
	}
}

// cards/tasks/tags WITHOUT version/sync_uuid and no sync_log — the state of
// any DB that predates the Phase 0a migration (epic Zettelgarden-v5b).
func createSyncTablesWithoutSyncColumns(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT, auth_provider TEXT DEFAULT 'local')`,
		`CREATE TABLE cards (id INTEGER PRIMARY KEY AUTOINCREMENT, card_id TEXT, title TEXT, user_id INTEGER, created_at DATETIME, updated_at DATETIME, is_deleted BOOLEAN DEFAULT false, parent_id INTEGER)`,
		`CREATE TABLE tasks (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, title TEXT NOT NULL, created_at DATETIME, updated_at DATETIME, is_deleted BOOLEAN DEFAULT false)`,
		`CREATE TABLE tags (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, color TEXT, user_id INTEGER, is_deleted BOOLEAN DEFAULT false, created_at DATETIME, updated_at DATETIME)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create pre-sync table: %v", err)
		}
	}
}

func TestSyncSelfHealAddsColumnsCreatesLogBackfills(t *testing.T) {
	db := openMemSQLite(t)
	createSyncTablesWithoutSyncColumns(t, db)

	// Seed legacy rows with NULL sync_uuid (pre-sync writes).
	for _, ins := range []string{
		`INSERT INTO cards (title) VALUES ('legacy card')`,
		`INSERT INTO tasks (user_id, title) VALUES (1, 'legacy task')`,
		`INSERT INTO tags (name, user_id) VALUES ('legacy', 1)`,
	} {
		if _, err := db.Exec(ins); err != nil {
			t.Fatalf("seed legacy row: %v", err)
		}
	}

	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// Columns added to all three synced tables.
	for _, table := range []string{"cards", "tasks", "tags"} {
		for _, col := range []string{"version", "sync_uuid"} {
			got, err := sqliteColumnExists(db, table, col)
			if err != nil || !got {
				t.Fatalf("%s.%s missing after upgrade (got=%v err=%v)", table, col, got, err)
			}
		}
	}

	// sync_log + feed index created.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sync_log'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("sync_log missing after upgrade (n=%d err=%v)", n, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_sync_log_user_id'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("idx_sync_log_user_id missing after upgrade (n=%d err=%v)", n, err)
	}

	// Legacy rows back-filled with non-NULL sync_uuid; version defaults to 1.
	for _, table := range []string{"cards", "tasks", "tags"} {
		var nullCount int
		if err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE sync_uuid IS NULL`, table)).Scan(&nullCount); err != nil {
			t.Fatalf("check NULL sync_uuid in %s: %v", table, err)
		}
		if nullCount != 0 {
			t.Errorf("%s still has %d NULL sync_uuid rows after back-fill", table, nullCount)
		}
	}
	var version int
	if err := db.QueryRow(`SELECT version FROM cards LIMIT 1`).Scan(&version); err != nil || version != 1 {
		t.Fatalf("cards.version = %d (want 1), err=%v", version, err)
	}

	// Unique partial sync_uuid indexes present on all three tables.
	for _, table := range []string{"cards", "tasks", "tags"} {
		idx := fmt.Sprintf("idx_%s_sync_uuid", table)
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=$1`, idx).Scan(&n); err != nil || n != 1 {
			t.Fatalf("index %s missing (n=%d err=%v)", idx, n, err)
		}
	}

	// Idempotent: a second run is a clean no-op (self-heal runs on every boot).
	if err := ensureSQLiteSchemaUpgrades(db); err != nil {
		t.Fatalf("second upgrade (idempotency): %v", err)
	}
}
