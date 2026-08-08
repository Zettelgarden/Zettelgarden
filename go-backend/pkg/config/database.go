package config

// DatabaseConfig holds the SQLite database configuration.
//
// The Postgres backend was retired after the cutover (epic Zettelgarden-c7j,
// Phase 7b); the backend is unconditionally SQLite, so the only configurable
// field is the database file path. The DB_DRIVER env var and the Postgres
// connection fields (Host/Port/User/Password/DatabaseName) are gone.
type DatabaseConfig struct {
	SQLitePath string // Path to the SQLite database file
}

// loadDatabaseConfig loads the database configuration from environment
// variables. SQLITE_PATH defaults to ./data/zettelgarden.db.
func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		SQLitePath: requireStringWithDefault("SQLITE_PATH", "./data/zettelgarden.db"),
	}
}
