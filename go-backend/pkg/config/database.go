package config

// DatabaseConfig holds all database connection configuration
type DatabaseConfig struct {
	Host         string // Database server hostname or IP
	Port         string // Database server port
	User         string // Database username
	Password     string // Database password (sensitive)
	DatabaseName string // Database name to connect to
}

// LoadDatabaseConfig loads and validates database configuration from environment variables
func loadDatabaseConfig() DatabaseConfig {
	config := DatabaseConfig{
		Host:         requireString("DB_HOST"),
		Port:         requireString("DB_PORT"),
		User:         requireString("DB_USER"),
		Password:     requireString("DB_PASS"),
		DatabaseName: requireString("DB_NAME"),
	}

	// Additional validation - check for localhost/empty values that would cause connection issues
	if config.Port == "" || config.Port == "0" {
		validationErrors = append(validationErrors,
			"DB_PORT cannot be empty or '0' - must be a valid database port number")
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