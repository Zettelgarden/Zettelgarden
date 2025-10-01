package services

import (
	"database/sql"
)

// GetUserMemory retrieves the memory for a user from the database
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

// UpdateUserMemory updates or creates a user's memory in the database
func UpdateUserMemory(db *sql.DB, userID uint, memory string) error {
	_, err := db.Exec(`
		INSERT INTO user_memories (user_id, memory, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET memory = $2, updated_at = NOW()
	`, userID, memory)
	return err
}
