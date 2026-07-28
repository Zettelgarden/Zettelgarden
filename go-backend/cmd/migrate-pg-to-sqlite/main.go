// Command migrate-pg-to-sqlite is the one-time ETL that imports the live
// Postgres database into a SQLite file built from the consolidated schema
// (Phase 6b of the Postgres → SQLite migration).
//
// It reads from Postgres via lib/pq and writes to SQLite via modernc.org/sqlite,
// copying every table that exists on BOTH sides (the dynamic PG∩SQLite
// intersection, which is lossless and adapts to schema drift). Primary keys are
// preserved verbatim so the entity/backlink/fact/task graph stays intact. The
// load runs with FK enforcement temporarily OFF on a dedicated connection and
// finishes with PRAGMA foreign_key_check; the run is idempotent (per-table
// wipe+reload) so it can be re-run dozens of times against a COPY of Postgres
// during development.
//
// Usage:
//
//	migrate-pg-to-sqlite [--sqlite-path PATH] [--pg-dsn DSN]
//	                    [--no-schema] [--verify-only] [--table T]...
//
// Postgres connection defaults to the DB_HOST/DB_PORT/DB_USER/DB_PASS/DB_NAME
// environment variables (same shape as the server), or an explicit --pg-dsn.
// The SQLite target defaults to SQLITE_PATH env or ./data/zettelgarden.db.
//
// Develop against a COPY of Postgres (pg_dump/restore to a throwaway DB), never
// the live DB. See the Cutover Runbook in
// docs/plans/2026-07-17-postgres-to-sqlite-migration-design.md.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"go-backend/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	sqlitePath := flag.String("sqlite-path", envOr("SQLITE_PATH", "./data/zettelgarden.db"),
		"path to the target SQLite database file (created if missing)")
	pgDSN := flag.String("pg-dsn", "",
		"full lib/pq Postgres DSN; overrides DB_* env (e.g. 'host=... port=5432 user=... password=... dbname=... sslmode=disable')")
	noSchema := flag.Bool("no-schema", false, "do not load the consolidated SQLite schema first (assume the target DB already has it)")
	verifyOnly := flag.Bool("verify-only", false, "skip the copy; only compare PG vs SQLite row counts and run foreign_key_check")
	tableFilter := flag.String("tables", "", "optional comma-separated allow-list of tables (for debugging); default = all shared tables")
	flag.Parse()

	ctx := context.Background()

	pg, err := openPostgres(*pgDSN)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer pg.Close()

	sqlite, err := server.OpenSQLite(*sqlitePath)
	if err != nil {
		log.Fatalf("open sqlite %q: %v", *sqlitePath, err)
	}
	defer sqlite.Close()

	if !*verifyOnly && !*noSchema {
		log.Printf("loading consolidated SQLite schema into %s", *sqlitePath)
		// RunMigrations bootstraps its own `migrations` bookkeeping table and
		// applies schema.sqlite.sql via the Phase 1 statement splitter. It is
		// idempotent — safe to run against an already-populated file.
		server.RunMigrations(&server.Server{
			DB:        sqlite,
			SchemaDir: "./schema/sqlite",
			Driver:    "sqlite",
		})
	}

	// Discover the migration set = tables present on BOTH sides. The PG-only
	// `migrations` table (runner-owned on the SQLite side) naturally drops out.
	srcTables, err := pgTables(ctx, pg)
	if err != nil {
		log.Fatalf("list postgres tables: %v", err)
	}
	dstTables, err := sqliteTables(ctx, sqlite)
	if err != nil {
		log.Fatalf("list sqlite tables: %v", err)
	}
	tables := intersectSorted(srcTables, dstTables)

	if f := splitCSV(*tableFilter); len(f) > 0 {
		want := map[string]bool{}
		for _, t := range f {
			want[t] = true
		}
		var pruned []string
		for _, t := range tables {
			if want[t] {
				pruned = append(pruned, t)
			}
		}
		log.Printf("table allow-list active: %d/%d shared tables selected", len(pruned), len(tables))
		tables = pruned
	}

	log.Printf("migration set: %d shared tables (%d PG-only skipped, %d SQLite-only skipped)",
		len(tables), len(srcTables)-len(tables), len(dstTables)-len(tables))
	if len(srcTables)-len(tables) > 0 || len(dstTables)-len(tables) > 0 {
		log.Printf("  PG-only: %v", diff(srcTables, tables))
		log.Printf("  SQLite-only: %v", diff(dstTables, tables))
	}

	if !*verifyOnly {
		if err := runCopy(ctx, sqlite, pg, tables); err != nil {
			log.Fatalf("copy failed: %v", err)
		}
	}

	// Verify (always, including --verify-only).
	results, mismatches, err := verifyCounts(ctx, pg, sqlite, tables)
	if err != nil {
		log.Fatalf("verify: %v", err)
	}
	var grandPG, grandCount int64
	for _, r := range results {
		grandPG += r.PGCount
		grandCount += r.Count
	}
	log.Printf("row totals: pg=%d sqlite=%d", grandPG, grandCount)

	fkViol, err := foreignKeyCheck(ctx, sqlite)
	if err != nil {
		log.Fatalf("foreign_key_check: %v", err)
	}

	if len(mismatches) == 0 && len(fkViol) == 0 {
		log.Printf("VERIFY OK: %d tables match, 0 FK violations", len(results))
		return
	}
	for _, m := range mismatches {
		log.Printf("  MISMATCH %-28s pg_count=%-8d sqlite_count=%-8d pg_id=[%v..%v] sqlite_id=[%v..%v]",
			m.Table, m.PGCount, m.Count, m.PGMin, m.PGMax, m.Min, m.Max)
	}
	for _, v := range fkViol {
		log.Printf("  FK VIOLATION %s", v)
	}
	log.Printf("VERIFY FAILED: %d table mismatches, %d FK violations", len(mismatches), len(fkViol))
	os.Exit(1)
}

// runCopy performs the bulk load of every table. FK enforcement is turned OFF
// on a dedicated connection for the duration (the design doc's recommended
// alternative to INSERT OR REPLACE), so the per-table wipe+reload is safe
// across the FK graph; foreign_key_check runs afterward (DB-wide) to catch any
// genuine integrity errors.
func runCopy(ctx context.Context, sqlite *sql.DB, pg *sql.DB, tables []string) error {
	loadConn, err := sqlite.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire sqlite load conn: %w", err)
	}
	defer loadConn.Close()

	// PRAGMA foreign_keys is connection-scoped and cannot change inside a tx,
	// so set it on the fresh conn before any BEGIN. This overrides the DSN's
	// per-connection foreign_keys=ON for the load conn only.
	if _, err := loadConn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return fmt.Errorf("disable FKs: %w", err)
	}
	defer func() { _, _ = loadConn.ExecContext(ctx, "PRAGMA foreign_keys=ON") }()

	startAll := time.Now()
	for _, t := range tables {
		start := time.Now()
		n, err := copyTable(ctx, loadConn, pg, "postgres", t)
		if err != nil {
			return fmt.Errorf("table %s: %w", t, err)
		}
		log.Printf("  %-30s %8d rows  %s", t, n, time.Since(start).Round(time.Millisecond))
	}
	log.Printf("copied %d tables in %s", len(tables), time.Since(startAll).Round(time.Millisecond))
	return nil
}

// openPostgres opens a lib/pq connection, using the explicit DSN or building
// one from the DB_* env vars (the same shape the server uses).
func openPostgres(dsn string) (*sql.DB, error) {
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			envOr("DB_HOST", "localhost"),
			envOr("DB_PORT", "5432"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASS"),
			envOr("DB_NAME", os.Getenv("DB_USER")),
		)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping (dsn hidden): %w", err)
	}
	// Bulk copy is single-threaded over one table at a time; a small read pool
	// is plenty.
	db.SetMaxOpenConns(4)
	return db, nil
}

// pgTables returns the user tables in the public schema of the Postgres DB.
func pgTables(ctx context.Context, pg *sql.DB) ([]string, error) {
	rows, err := pg.QueryContext(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// sqliteTables returns the user tables in the SQLite DB (excludes internal
// sqlite_* and the runner-owned migrations table).
func sqliteTables(ctx context.Context, sqlite *sql.DB) ([]string, error) {
	rows, err := sqlite.QueryContext(ctx, `
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		if t == "migrations" {
			continue // runner-owned bookkeeping, not domain data
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func intersectSorted(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, x := range b {
		set[x] = struct{}{}
	}
	var out []string
	for _, x := range a {
		if _, ok := set[x]; ok {
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

// diff returns elements of a not in b (for logging skipped tables).
func diff(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, x := range b {
		set[x] = struct{}{}
	}
	var out []string
	for _, x := range a {
		if _, ok := set[x]; !ok {
			out = append(out, x)
		}
	}
	return out
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func envOr(key, dflt string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return dflt
}
