package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrSpreadsheetNotFound is returned when a spreadsheet cannot be found
var ErrSpreadsheetNotFound = errors.New("spreadsheet not found")

// Spreadsheet represents a spreadsheet attached to a card
type Spreadsheet struct {
	ID        int               `json:"id"`
	UserID    int               `json:"user_id"`
	CardID    int               `json:"card_id"`
	Name      string            `json:"name"`
	Rows      int               `json:"rows"`
	Cols      int               `json:"cols"`
	Data      SpreadsheetData    `json:"data"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// SpreadsheetData represents the cell data of a spreadsheet
type SpreadsheetData struct {
	Rows int                       `json:"rows"`
	Cols int                       `json:"cols"`
	Data map[string]SpreadsheetCell `json:"data"`
}

// SpreadsheetCell represents a single cell in a spreadsheet
type SpreadsheetCell struct {
	Value    string   `json:"value"`
	Formula   string   `json:"formula,omitempty"`
	Computed  *float64 `json:"computed,omitempty"`
}

// GetSpreadsheetsByCardID retrieves all spreadsheets for a specific card
func GetSpreadsheetsByCardID(db Database, cardID int, userID int) ([]Spreadsheet, error) {
	query := `
		SELECT id, user_id, card_id, name, rows, cols, data, created_at, updated_at
		FROM spreadsheets
		WHERE card_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query, cardID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query spreadsheets: %w", err)
	}
	defer rows.Close()

	var spreadsheets []Spreadsheet
	for rows.Next() {
		var spreadsheet Spreadsheet
		var dataJSON []byte

		err := rows.Scan(
			&spreadsheet.ID,
			&spreadsheet.UserID,
			&spreadsheet.CardID,
			&spreadsheet.Name,
			&spreadsheet.Rows,
			&spreadsheet.Cols,
			&dataJSON,
			&spreadsheet.CreatedAt,
			&spreadsheet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan spreadsheet: %w", err)
		}

		// Unmarshal JSONB data
		if len(dataJSON) > 0 {
			err = json.Unmarshal(dataJSON, &spreadsheet.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal spreadsheet data: %w", err)
			}
		}

		spreadsheets = append(spreadsheets, spreadsheet)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating spreadsheet rows: %w", err)
	}

	return spreadsheets, nil
}

// GetSpreadsheetByID retrieves a single spreadsheet by ID
func GetSpreadsheetByID(db Database, id int, userID int) (*Spreadsheet, error) {
	query := `
		SELECT id, user_id, card_id, name, rows, cols, data, created_at, updated_at
		FROM spreadsheets
		WHERE id = $1 AND user_id = $2
	`

	var spreadsheet Spreadsheet
	var dataJSON []byte

	err := db.QueryRow(query, id, userID).Scan(
		&spreadsheet.ID,
		&spreadsheet.UserID,
		&spreadsheet.CardID,
		&spreadsheet.Name,
		&spreadsheet.Rows,
		&spreadsheet.Cols,
		&dataJSON,
		&spreadsheet.CreatedAt,
		&spreadsheet.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSpreadsheetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query spreadsheet: %w", err)
	}

	// Unmarshal JSONB data
	if len(dataJSON) > 0 {
		err = json.Unmarshal(dataJSON, &spreadsheet.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal spreadsheet data: %w", err)
		}
	}

	return &spreadsheet, nil
}

// CreateSpreadsheet creates a new spreadsheet
func CreateSpreadsheet(db Database, spreadsheet *Spreadsheet) error {
	// Marshal data to JSON for JSONB storage
	dataJSON, err := json.Marshal(spreadsheet.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal spreadsheet data: %w", err)
	}

	query := `
		INSERT INTO spreadsheets (user_id, card_id, name, rows, cols, data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err = db.QueryRow(
		query,
		spreadsheet.UserID,
		spreadsheet.CardID,
		spreadsheet.Name,
		spreadsheet.Rows,
		spreadsheet.Cols,
		dataJSON,
	).Scan(&spreadsheet.ID, &spreadsheet.CreatedAt, &spreadsheet.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create spreadsheet: %w", err)
	}

	return nil
}

// UpdateSpreadsheet updates an existing spreadsheet's data
func UpdateSpreadsheet(db Database, id int, userID int, data SpreadsheetData) error {
	// Verify spreadsheet exists and belongs to user
	_, err := GetSpreadsheetByID(db, id, userID)
	if err != nil {
		return err
	}

	// Marshal data to JSON for JSONB storage
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal spreadsheet data: %w", err)
	}

	query := `
		UPDATE spreadsheets
		SET data = $1, rows = $2, cols = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
	`

	_, err = db.Exec(query, dataJSON, data.Rows, data.Cols, id, userID)
	if err != nil {
		return fmt.Errorf("failed to update spreadsheet: %w", err)
	}

	return nil
}

// DeleteSpreadsheet deletes a spreadsheet
func DeleteSpreadsheet(db Database, id int, userID int) error {
	// Verify spreadsheet exists and belongs to user
	_, err := GetSpreadsheetByID(db, id, userID)
	if err != nil {
		return err
	}

	query := `DELETE FROM spreadsheets WHERE id = $1 AND user_id = $2`

	_, err = db.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete spreadsheet: %w", err)
	}

	return nil
}
