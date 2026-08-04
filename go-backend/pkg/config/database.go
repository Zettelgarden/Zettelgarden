package config

// DatabaseConfig holds the SQLite database configuration.
//
// The Postgres backend was retired after the cutover (epic Zettelgarden-c7j,
// Phase 7b), so the only configurable field is the database file path. Driver
// is retained as a constant tag ("sqlite") for the Server/Driver wiring and
// the test harness, which still branch on it; the DB_DRIVER env var and the
// Postgres connection fields (Host/Port/User/Password/DatabaseName) are gone.
type DatabaseConfig struct {
	Driver     string // always "sqlite" (kept for Server/Driver wiring)
	SQLitePath string // Path to the SQLite database file
}

// loadDatabaseConfig loads the database configuration from environment
// variables. SQLITE_PATH defaults to ./data/zettelgarden.db.
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Driver:     "sqlite",
		SQLitePath: requireStringWithDefault("SQLITE_PATH", "./data/zettelgarden.db"),
	}
}
