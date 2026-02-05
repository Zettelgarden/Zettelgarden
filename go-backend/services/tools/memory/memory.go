// Package memory provides memory-related tools for the Zettelgarden tool registry.
//
// This package is part of Phase 3: Split Tool Handlers by Domain with Feature Flags.
// The memory domain contains tools for managing user memory and observations.
//
// PHASE 3 DESIGN NOTES:
// ---------------------
// For this proof of concept, the memory domain package demonstrates the pattern
// for splitting tools into separate domain packages. The registration is still
// handled in services/memory_tools.go to avoid circular import dependencies.
//
// The domain package contains:
// 1. Data access functions (GetUserMemory, UpdateUserMemory)
// 2. Tool handler logic
// 3. Domain-specific business logic
//
// Future domains can follow this pattern:
// - Create services/tools/{domain}/ package
// - Move data access and handler logic to the domain package
// - Keep registration in services/{domain}_tools.go with feature flag
// - Use feature flag to toggle between old and new paths
package memory

import (
	"database/sql"
)

// GetUserMemory retrieves the memory for a user from the database.
// This is the domain data access function for memory operations.
func GetUserMemory(db *sql.DB, userID int) (string, error) {
	var memory string
	err := db.QueryRow("SELECT memory FROM user_memories WHERE user_id = $1", userID).Scan(&memory)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return memory, nil
}

// UpdateUserMemory updates or creates a user's memory in the database.
// This is the domain data access function for memory operations.
func UpdateUserMemory(db *sql.DB, userID int, memory string) error {
	_, err := db.Exec(`
		INSERT INTO user_memories (user_id, memory, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET memory = $2, updated_at = NOW()
	`, userID, memory)
	return err
}
