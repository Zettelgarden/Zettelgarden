package services

import (
	"database/sql"

	"go-backend/services/tools/memory"
)

// GetUserMemory retrieves the memory for a user from the database.
// This function now delegates to the memory domain package (Phase 3 migration).
func GetUserMemory(db *sql.DB, userID int) (string, error) {
	return memory.GetUserMemory(db, userID)
}

// UpdateUserMemory updates or creates a user's memory in the database.
// This function now delegates to the memory domain package (Phase 3 migration).
func UpdateUserMemory(db *sql.DB, userID int, mem string) error {
	return memory.UpdateUserMemory(db, userID, mem)
}
