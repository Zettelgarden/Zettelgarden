package services

import (
	"encoding/json"
	"go-backend/models"
	"log"
)

// LogAgentActivity logs an agent action asynchronously to the database.
// The function runs in a goroutine and will not block the caller.
//
// Parameters:
//   - db: Database connection for inserting the log entry
//   - agentID: ID of the agent performing the action
//   - action: Type of action (e.g., "create_card", "update_card")
//   - targetType: Entity type being acted upon (e.g., "card", "task")
//   - targetID: Optional ID of the target entity (nil if not applicable)
//   - details: Optional additional information about the action
//
// The function includes panic recovery and error logging, making it safe
// for production use. Errors are logged but do not propagate to the caller.
func LogAgentActivity(db models.Database, agentID int, action, targetType string, targetID *int, details map[string]interface{}) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in LogAgentActivity: %v", r)
			}
		}()

		var detailsJSON interface{}
		if details != nil {
			bytes, err := json.Marshal(details)
			if err != nil {
				log.Printf("Error marshaling activity details: %v", err)
				return
			}
			detailsJSON = bytes
		}

		_, err := db.Exec(`
			INSERT INTO agent_activity_log (agent_id, action, target_type, target_id, details)
			VALUES ($1, $2, $3, $4, $5)
		`, agentID, action, targetType, targetID, detailsJSON)

		if err != nil {
			log.Printf("Error logging agent activity: %v", err)
		}
	}()
}

// GetAgentActivity retrieves paginated activity logs for a specific agent.
// Results are ordered by created_at DESC (newest first).
//
// Parameters:
//   - db: Database connection for querying logs
//   - agentID: ID of the agent to retrieve logs for
//   - page: Page number (1-indexed, values < 1 are treated as 1)
//   - perPage: Number of results per page (values < 1 are treated as 10, max 100)
//
// Returns:
//   - []models.AgentActivityLog: Slice of activity logs (empty slice if no results, never nil)
//   - int: Total count of logs for the agent
//   - error: Database error if query fails
//
// Example:
//
//	logs, total, err := GetAgentActivity(db, 42, 1, 10)
//	if err != nil {
//	    // handle error
//	}
func GetAgentActivity(db models.Database, agentID, page, perPage int) ([]models.AgentActivityLog, int, error) {
	// Validate and sanitize pagination parameters
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100 // Prevent unbounded queries
	}

	offset := (page - 1) * perPage

	// Get total count
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM agent_activity_log WHERE agent_id = $1`, agentID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get logs
	rows, err := db.Query(`
		SELECT id, agent_id, action, target_type, target_id, details, created_at
		FROM agent_activity_log
		WHERE agent_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, agentID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Initialize empty slice to avoid nil return
	logs := []models.AgentActivityLog{}
	for rows.Next() {
		var log models.AgentActivityLog
		var detailsJSON []byte
		err := rows.Scan(
			&log.ID,
			&log.AgentID,
			&log.Action,
			&log.TargetType,
			&log.TargetID,
			&detailsJSON,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &log.Details)
		}

		logs = append(logs, log)
	}

	return logs, total, nil
}
