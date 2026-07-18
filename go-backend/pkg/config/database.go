package config

// DatabaseConfig holds all database connection configuration
type DatabaseConfig struct {
	Driver       string // "postgres" (default) or "sqlite"
	Host         string // Database server hostname or IP (postgres only)
	Port         string // Database server port (postgres only)
	User         string // Database username (postgres only)
	Password     string // Database password (sensitive, postgres only)
	DatabaseName string // Database name to connect to (postgres only)
	SQLitePath   string // Path to the SQLite database file (sqlite only)
}

// LoadDatabaseConfig loads and validates database configuration from environment variables
func loadDatabaseConfig() DatabaseConfig {
	driver := requireStringWithDefault("DB_DRIVER", "postgres")
	config := DatabaseConfig{
		Driver:     driver,
		SQLitePath: requireStringWithDefault("SQLITE_PATH", "./data/zettelgarden.db"),
	}

	// PostgreSQL connection fields are only required when running on the
	// postgres driver. In sqlite mode the DB_* environment is not used.
	if driver != "sqlite" {
		config.Host = requireString("DB_HOST")
		config.Port = requireString("DB_PORT")
		config.User = requireString("DB_USER")
		config.Password = requireString("DB_PASS")
		config.DatabaseName = requireString("DB_NAME")

		// Additional validation - check for localhost/empty values that would cause connection issues
		if config.Port == "" || config.Port == "0" {
			validationErrors = append(validationErrors,
				"DB_PORT cannot be empty or '0' - must be a valid database port number")
		}
	}

	return config
}

// ConnectionString returns a PostgreSQL connection string for the database config
func (c DatabaseConfig) ConnectionString() string {
	return c.User + ":" + c.Password + "@" + c.Host + ":" + c.Port + "/" + c.DatabaseName + "?sslmode=disable"
}

// TestConnectionString returns a test database connection string
func (c DatabaseConfig) TestConnectionString() string {
	return c.User + ":" + c.Password + "@" + c.Host + ":" + c.Port + "/zettelkasten_testing?sslmode=disable"
}