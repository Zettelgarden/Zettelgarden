package server

import (
	"database/sql"
	"fmt"
	"go-backend/models"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func ConnectToDatabase(dbConfig models.DatabaseConfig) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%v port=%v user=%v "+
		"password=%v dbname=%v sslmode=disable",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DatabaseName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Configure connection pool for better performance
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, err
}

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
}

// execScript executes a multi-statement SQL script within tx. lib/pq parses
// multiple statements in a single Exec; modernc.org/sqlite does not, so for the
// sqlite driver the script is split into individual statements first (via
// SplitSQL) before execution.
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
