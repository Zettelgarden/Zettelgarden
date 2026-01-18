package config

import (
	"strconv"
	"strings"
)

// ServerConfig holds all server/application configuration
type ServerConfig struct {
	DevMode      bool   // Development mode flag
	Port         string // Server port (defaults to 8080)
	URL          string // Base URL for the application
	LogLocation  string // Path to log file (empty for dev console logging)
	AdminEmail   string // Admin email for notifications
	JwtSecretKey string // JWT signing secret (sensitive)
}

// LoadServerConfig loads and validates server configuration from environment variables
func loadServerConfig() ServerConfig {
	config := ServerConfig{
		DevMode:      requireBool("ZETTEL_DEV"),
		Port:         requireStringWithDefault("ZETTEL_PORT", "8080"),
		URL:          requireString("ZETTEL_URL"),
		LogLocation:  optionalString("ZETTEL_BACKEND_LOG_LOCATION"),
		AdminEmail:   requireString("ZETTEL_ADMIN_EMAIL"),
		JwtSecretKey: requireString("SECRET_KEY"),
	}

	// Validate port format (should be numeric)
	if config.Port != "" {
		if _, err := strconv.ParseInt(config.Port, 10, 32); err != nil {
			validationErrors = append(validationErrors,
				"invalid port format for ZETTEL_PORT: "+config.Port+" (must be numeric, e.g. '8080')")
		} else if port := requireInt("ZETTEL_PORT"); port <= 0 || port > 65535 {
			validationErrors = append(validationErrors,
				"ZETTEL_PORT must be between 1 and 65535, got: "+config.Port)
		}
	}

	// Validate URL format
	validateURL("ZETTEL_URL", config.URL)

	// Validate JWT secret key length
	//if len(config.JwtSecretKey) < 32 {
	//	validationErrors = append(validationErrors,
	//	"SECRET_KEY is too short (must be at least 32 characters for security)")
	//}

	// Validate admin email basic format
	if config.AdminEmail != "" && !strings.Contains(config.AdminEmail, "@") {
		validationErrors = append(validationErrors,
			"ZETTEL_ADMIN_EMAIL must be a valid email address: "+config.AdminEmail)
	}

	return config
}
