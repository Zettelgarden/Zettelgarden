package config

// Config holds all application configuration loaded from environment variables
type Config struct {
	Database DatabaseConfig // Database connection settings
	Server   ServerConfig   // Server and application settings
	Services ServiceConfig  // External service configurations
}

// LoadConfig loads and validates all application configuration from environment variables.
// This function reads all required environment variables and performs validation.
// If validation fails in production mode (DevMode false), it panics with a detailed
// error message. In dev mode (ZETTEL_DEV=true) validation errors are tolerated so
// local development and the test suite can boot with partial configuration.
func LoadConfig() Config {
	// Clear any previous validation errors
	validationErrors = nil

	config := Config{
		Database: loadDatabaseConfig(),
		Server:   loadServerConfig(),
		Services: loadServiceConfig(),
	}

	// Check for validation errors and panic if any found.
	// Production instances fail fast with a readable list of missing/invalid
	// variables; dev mode stays lenient so local development and the test suite
	// can boot with partial configuration.
	if !config.Server.DevMode {
		panicOnValidationErrors()
	}

	// Set global instance for service access
	globalConfig = &config

	return config
}

// GetConfig returns the global config instance (nil if not loaded)
func GetConfig() *Config {
	return globalConfig
}

// LoadTestConfig loads configuration suitable for testing.
// It's more permissive than LoadConfig and allows for missing optional values.
func LoadTestConfig() Config {
	// In test mode, we might not have all environment variables set
	config := LoadConfig()
	return config
}

var globalConfig *Config
