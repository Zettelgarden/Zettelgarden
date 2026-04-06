package services

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"log"
)

// LogAgentActivity logs an agent action asynchronously
func LogAgentActivity(db *sql.DB, agentID int, action, targetType string, targetID *int, details map[string]interface{}) {
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

// GetAgentActivity retrieves paginated activity logs for an agent
func GetAgentActivity(db *sql.DB, agentID, page, perPage int) ([]models.AgentActivityLog, int, error) {
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
