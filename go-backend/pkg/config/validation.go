package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

var validationErrors []string

// requireString requires an environment variable to be set and non-empty
func requireString(key string) string {
	value := os.Getenv(key)
	if value == "" {
		validationErrors = append(validationErrors, fmt.Sprintf("required environment variable %s is not set or empty", key))
	}
	return value
}

// requireStringWithDefault requires an environment variable to be set and non-empty,
// returning the provided default if not set
func requireStringWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// requireBool parses an environment variable as a boolean with specific string values
func requireBool(key string) bool {
	value := os.Getenv(key)
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off", "":
		return false
	default:
		validationErrors = append(validationErrors, fmt.Sprintf("invalid boolean value for %s: '%s' (expected true/false/1/0/yes/no/on/off)", key, value))
		return false
	}
}

// optionalBool parses an environment variable as a boolean, defaulting to false if not set
func optionalBool(key string) bool {
	value := os.Getenv(key)
	if value == "" {
		return false
	}
	return requireBool(key)
}

// optionalString returns an empty string if the environment variable is not set
func optionalString(key string) string {
	return os.Getenv(key)
}

// requireInt parses an environment variable as an integer
func requireInt(key string) int {
	value := os.Getenv(key)
	if value == "" {
		validationErrors = append(validationErrors, fmt.Sprintf("required environment variable %s is not set", key))
		return 0
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("invalid integer value for %s: '%s'", key, err.Error()))
		return 0
	}
	return intValue
}

// validateURL performs basic URL validation (must contain ://)
func validateURL(key, value string) {
	if value != "" && !strings.Contains(value, "://") {
		validationErrors = append(validationErrors, fmt.Sprintf("invalid URL format for %s: '%s' (must contain ://)", key, value))
	}
}

// getValidationErrors returns all accumulated validation errors and clears the list
func getValidationErrors() []string {
	errors := make([]string, len(validationErrors))
	copy(errors, validationErrors)
	validationErrors = nil
	return errors
}

// panicOnValidationErrors panics if there are any validation errors, printing a comprehensive message
func panicOnValidationErrors() {
	errors := getValidationErrors()
	if len(errors) > 0 {
		message := "Configuration validation failed:\n"
		for _, err := range errors {
			message += "  - " + err + "\n"
		}
		message += "\nPlease check your environment variables and try again."
		panic(message)
	}
}