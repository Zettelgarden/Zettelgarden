package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"go-backend/models"
)

// MigrationResult represents the result of a spreadsheet migration operation
type MigrationResult struct {
	Migrated     int                    `json:"migrated"`      // Number of spreadsheets migrated
	Errors      []MigrationError       `json:"errors"`        // List of errors that occurred
	CardUpdates int                    `json:"card_updates"`   // Number of cards updated
}

// MigrationError represents an error that occurred during migration
type MigrationError struct {
	CardID   int    `json:"card_id"`
	CardName string `json:"card_name"`
	Error    string `json:"error"`
}

// SpreadsheetBlock represents a parsed spreadsheet block from card body
type SpreadsheetBlock struct {
	Name      string              `json:"name"`
	CardID    int                 `json:"card_id"`
	StartPos  int                 `json:"start_pos"`
	EndPos    int                 `json:"end_pos"`
	Data      models.SpreadsheetData `json:"data"`
}

// MigrateCardSpreadsheetsResponse is the response returned by the migration endpoint
type MigrateCardSpreadsheetsResponse struct {
	Message string         `json:"message"`
	Result  MigrationResult `json:"result"`
}

// migrateSpreadsheetRegex matches the old spreadsheet block format:
// ```spreadsheet:name\n{...json...}\n```
var migrateSpreadsheetRegex = regexp.MustCompile(
	"```spreadsheet:([a-zA-Z0-9_-]+)\n([\\s\\S]*?)```",
)

// MigrateCardSpreadsheets migrates all embedded spreadsheet blocks from card bodies
// to dedicated database records.
//
// This is an admin-only endpoint that performs a one-time migration of legacy
// spreadsheet data stored in markdown code blocks to the new spreadsheets table.
//
// Process:
// 1. Finds all cards for the current user (admin users migrate all users' cards)
// 2. Uses regex to find spreadsheet blocks in card bodies
// 3. Parses JSON data from blocks
// 4. Creates database records for each spreadsheet
// 5. Replaces blocks in card body with {{spreadsheet:ID}} references
// 6. Returns migration results (count migrated, errors list)
func (s *Handler) MigrateCardSpreadsheets(w http.ResponseWriter, r *http.Request) {
	// Get the current user from context (must be admin due to AdminMiddleware)
	userID, ok := r.Context().Value("current_user").(int)
	if !ok {
		log.Printf("MigrateCardSpreadsheets: current_user not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	log.Printf("Starting spreadsheet migration for user %d", userID)

	result := MigrationResult{
		Migrated: 0,
		Errors:    []MigrationError{},
		CardUpdates: 0,
	}

	// Get the admin user to check if they're admin
	adminUser, err := s.QueryUser(userID)
	if err != nil {
		log.Printf("MigrateCardSpreadsheets: error querying admin user %d: %v", userID, err)
		http.Error(w, "Error verifying permissions", http.StatusInternalServerError)
		return
	}

	// Admin users migrate all cards, regular users migrate only their own
	var cards []models.Card
	if adminUser.IsAdmin {
		// Get all cards in the system for admin
		cards, err = s.getAllCards()
		if err != nil {
			log.Printf("MigrateCardSpreadsheets: error getting all cards: %v", err)
			http.Error(w, "Error retrieving cards", http.StatusInternalServerError)
			return
		}
		log.Printf("Migrating spreadsheets from all %d cards", len(cards))
	} else {
		// Regular users can only migrate their own cards (though this endpoint is admin-only)
		http.Error(w, "Access denied. Admin privileges required.", http.StatusForbidden)
		return
	}

	// Process each card
	for _, card := range cards {
		cardMigrated, cardErrors := s.migrateCardSpreadsheets(card)
		result.Migrated += cardMigrated
		result.Errors = append(result.Errors, cardErrors...)
		if cardMigrated > 0 {
			result.CardUpdates++
		}
	}

	// Log the admin action
	s.LogAdminAction(r, "spreadsheet.migrate", "spreadsheet", 0, map[string]interface{}{
		"migrated":     result.Migrated,
		"card_updates": result.CardUpdates,
		"errors":       len(result.Errors),
	})

	log.Printf("Spreadsheet migration complete: %d spreadsheets migrated, %d cards updated, %d errors",
		result.Migrated, result.CardUpdates, len(result.Errors))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MigrateCardSpreadsheetsResponse{
		Message: fmt.Sprintf("Migration complete: %d spreadsheets migrated, %d cards updated, %d errors",
			result.Migrated, result.CardUpdates, len(result.Errors)),
		Result: result,
	})
}

// getAllCards retrieves all cards in the system (for admin migration)
func (s *Handler) getAllCards() ([]models.Card, error) {
	query := `
		SELECT id, card_id, user_id, title, body, link, parent_id, created_at, updated_at, is_deleted
		FROM cards
		ORDER BY user_id, id
	`

	rows, err := s.GetDB().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards: %w", err)
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		var card models.Card
		var parentID sql.NullInt64
		var body sql.NullString
		var link sql.NullString

		err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&body,
			&link,
			&parentID,
			&card.CreatedAt,
			&card.UpdatedAt,
			&card.IsDeleted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan card: %w", err)
		}

		if body.Valid {
			card.Body = body.String
		}
		if link.Valid {
			card.Link = link.String
		}
		if parentID.Valid {
			card.ParentID = new(int)
			*card.ParentID = int(parentID.Int64)
		}

		cards = append(cards, card)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating card rows: %w", err)
	}

	return cards, nil
}

// migrateCardSpreadsheets migrates spreadsheet blocks from a single card
// Returns number of spreadsheets migrated and any errors
func (s *Handler) migrateCardSpreadsheets(card models.Card) (int, []MigrationError) {
	migratedCount := 0
	var cardErrors []MigrationError

	// Skip deleted cards
	if card.IsDeleted {
		return 0, nil
	}

	// Find all spreadsheet blocks in the card body
	matches := migrateSpreadsheetRegex.FindAllStringSubmatchIndex(card.Body, -1)
	if len(matches) == 0 {
		// No spreadsheet blocks found
		return 0, nil
	}

	log.Printf("Found %d spreadsheet blocks in card %d (%s)", len(matches), card.ID, card.Title)

	// Process matches in forward order, adjusting positions after each replacement
	newBody := card.Body
	currentOffset := 0

	for _, match := range matches {
		// Calculate actual positions in the current newBody
		startPos := match[0] + currentOffset
		endPos := match[1] + currentOffset

		name := card.Body[match[2]:match[3]]
		jsonData := card.Body[match[4]:match[5]]

		// Parse the spreadsheet data
		var data models.SpreadsheetData
		if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
			cardErrors = append(cardErrors, MigrationError{
				CardID:   card.ID,
				CardName: card.Title,
				Error:    fmt.Sprintf("Failed to parse spreadsheet JSON: %v", err),
			})
			log.Printf("Error parsing spreadsheet JSON in card %d: %v", card.ID, err)
			continue
		}

		// Create the spreadsheet in the database
		spreadsheet := &models.Spreadsheet{
			UserID: card.UserID,
			CardID: card.ID,
			Name:   name,
			Rows:   data.Rows,
			Cols:   data.Cols,
			Data:    data,
		}

		if err := models.CreateSpreadsheet(s.GetDB(), spreadsheet); err != nil {
			cardErrors = append(cardErrors, MigrationError{
				CardID:   card.ID,
				CardName: card.Title,
				Error:    fmt.Sprintf("Failed to create spreadsheet: %v", err),
			})
			log.Printf("Error creating spreadsheet for card %d: %v", card.ID, err)
			continue
		}

		log.Printf("Created spreadsheet %d for card %d with name '%s'", spreadsheet.ID, card.ID, name)

		// Create the replacement string: {{spreadsheet:ID}}
		replacement := fmt.Sprintf("{{spreadsheet:%d}}", spreadsheet.ID)

		// Replace the block with the reference
		newBody = newBody[:startPos] + replacement + newBody[endPos:]

		// Update offset for subsequent matches
		currentOffset += len(replacement) - (match[1] - match[0])

		migratedCount++
	}

	// Update the card body if any spreadsheets were migrated
	if migratedCount > 0 && newBody != card.Body {
		if err := s.updateCardBody(card.ID, card.UserID, newBody); err != nil {
			cardErrors = append(cardErrors, MigrationError{
				CardID:   card.ID,
				CardName: card.Title,
				Error:    fmt.Sprintf("Failed to update card body: %v", err),
			})
			log.Printf("Error updating card body for card %d: %v", card.ID, err)
			// We still count the spreadsheets as migrated since they were created
			// The card body update failure is tracked as an error
		} else {
			log.Printf("Updated card body for card %d", card.ID)
		}
	}

	return migratedCount, cardErrors
}

// updateCardBody updates the body of a card
func (s *Handler) updateCardBody(cardID int, userID int, newBody string) error {
	query := `
		UPDATE cards
		SET body = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
	`

	_, err := s.GetDB().Exec(query, newBody, time.Now(), cardID, userID)
	if err != nil {
		return fmt.Errorf("failed to update card body: %w", err)
	}

	return nil
}
