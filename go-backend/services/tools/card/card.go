// Package card provides card-related tools for the Zettelgarden tool registry.
//
// This package is part of Phase 3: Split Tool Handlers by Domain with Feature Flags.
// The card domain contains tools for managing user cards, including search,
// retrieval, creation, updates, and analysis.
//
// PHASE 3 DESIGN NOTES:
// ---------------------
// The card domain package follows the pattern established by memory_tools for
// splitting tools into separate domain packages. The registration is handled
// in services/card_tools.go to avoid circular import dependencies.
//
// The domain package contains:
// 1. Data access functions for card operations
// 2. Tool handler logic
// 3. Domain-specific business logic for card management
//
// This is a critical path domain with 6 tools:
// - SearchCards: Text and semantic search across cards
// - GetCardByID: Retrieve a specific card by ID
// - BrowseCardHierarchy: Navigate parent-child relationships
// - CreateCard: Create new cards
// - UpdateCard: Modify existing cards
// - GetCardAnalysis: Retrieve card analysis/summaries
package card

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"go-backend/models"
)

// GetCardByID retrieves a full card by its primary key ID.
// This is the domain data access function for single card retrieval.
func GetCardByID(db models.Database, userID int, cardPK int) (models.Card, error) {
	var card models.Card

	err := db.QueryRow(`
	SELECT
	id, card_id, user_id, title, body, link, parent_id,
        created_at, updated_at, card_schema_id, structured_data
	FROM
	cards
	WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`, cardPK, userID).Scan(
		&card.ID,
		&card.CardID,
		&card.UserID,
		&card.Title,
		&card.Body,
		&card.Link,
		&card.ParentID,
		&card.CreatedAt,
		&card.UpdatedAt,
		&card.SchemaID,
		&card.StructuredData,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Card{}, fmt.Errorf("card not found")
		}
		return models.Card{}, fmt.Errorf("unable to access card: %w", err)
	}

	return card, nil
}

// GetChildCardsWithDepth recursively retrieves child cards up to the specified depth.
// depth of 1 returns only immediate children.
// depth of -1 returns all descendants (unlimited depth).
func GetChildCardsWithDepth(db models.Database, userID int, cardID int, depth int) ([]models.PartialCard, error) {
	var allCards []models.PartialCard

	if depth == 0 {
		return allCards, nil
	}

	// Get immediate children
	children, err := getChildCards(db, userID, cardID)
	if err != nil {
		return nil, err
	}

	allCards = append(allCards, children...)

	// If depth is 1, we're done - only return immediate children
	if depth == 1 {
		return allCards, nil
	}

	// Recursively get children of each child
	nextDepth := depth
	if depth > 1 {
		nextDepth = depth - 1
	}
	// If depth is -1, we keep it at -1 for unlimited traversal

	for _, child := range children {
		grandchildren, err := GetChildCardsWithDepth(db, userID, child.ID, nextDepth)
		if err != nil {
			// Log error but continue with other children
			fmt.Printf("Failed to get children of card %d: %v\n", child.ID, err)
			continue
		}
		allCards = append(allCards, grandchildren...)
	}

	return allCards, nil
}

// GetParentCardsWithDepth recursively retrieves parent cards up to the specified depth.
// depth of 1 returns only the immediate parent.
// depth of -1 returns all ancestors (unlimited depth).
func GetParentCardsWithDepth(db models.Database, userID int, cardPK int, depth int) ([]models.PartialCard, error) {
	var allCards []models.PartialCard

	if depth == 0 {
		return allCards, nil
	}

	// Get the current card
	card, err := getPartialCard(db, userID, cardPK)
	if err != nil {
		return allCards, err
	}

	// Get immediate parent
	parent, err := getPartialCard(db, userID, card.ParentID)
	if err != nil {
		// No parent found, return empty slice
		return allCards, nil
	}

	allCards = append(allCards, parent)

	// If depth is 1, we're done - only return immediate parent
	if depth == 1 {
		return allCards, nil
	}

	// Recursively get parents of the parent
	nextDepth := depth
	if depth > 1 {
		nextDepth = depth - 1
	}
	// If depth is -1, we keep it at -1 for unlimited traversal

	grandparents, err := GetParentCardsWithDepth(db, userID, parent.ID, nextDepth)
	if err != nil {
		fmt.Printf("Failed to get parents of card %d: %v\n", parent.ID, err)
		return allCards, nil
	}
	allCards = append(allCards, grandparents...)

	return allCards, nil
}

// CreateCard creates a new card in the database.
// This is the domain data access function for card creation.
func CreateCard(db models.Database, userID int, params models.EditCardParams) (models.Card, error) {
	// Check if card_id is unique
	if !checkIsCardIDUnique(db, userID, params.CardID) {
		return models.Card{}, fmt.Errorf("card_id already exists")
	}

	discoveredParentCardID := discoverParentId(params.CardID)
	var parentID int
	var isRootCard bool

	// Check if this is a root card (no parent separator in card_id)
	if discoveredParentCardID == params.CardID {
		// Root card - will set parent_id to itself after insert
		isRootCard = true
		parentID = 0
	} else {
		// Child card - try to find parent
		parent, lookupErr := getPartialCardByCardID(db, userID, discoveredParentCardID)
		if lookupErr != nil {
			fmt.Printf("Parent card lookup failed for card %q: %v. Will set as root/self-parent.\n", params.CardID, lookupErr)
			isRootCard = true
			parentID = 0
		} else {
			parentID = parent.ID
		}
	}

	query := `
	INSERT INTO cards
	(title, body, link, user_id, card_id, parent_id, card_schema_id, structured_data, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	RETURNING id;
	`
	var id int
	err := db.QueryRow(query, params.Title, params.Body, params.Link, userID, params.CardID, parentID, params.SchemaID, params.StructuredData).Scan(&id)
	if err != nil {
		return models.Card{}, fmt.Errorf("failed to create card: %w", err)
	}

	// Get the created card
	newCard, err := GetCardByID(db, userID, id)
	if err != nil {
		return models.Card{}, err
	}

	// set parent id to id if it's a root card or if card_id is empty
	if isRootCard || params.CardID == "" {
		_, err = db.Exec("UPDATE cards SET parent_id = $1 WHERE id = $1", id)
		if err != nil {
			return models.Card{}, err
		}
	}

	return newCard, nil
}

// UpdateCard updates an existing card in the database.
// This is the domain data access function for card updates.
func UpdateCard(db models.Database, userID int, cardPK int, params models.EditCardParams) (models.Card, error) {
	// Get the old state first
	oldCard, err := GetCardByID(db, userID, cardPK)
	if err != nil {
		return models.Card{}, err
	}

	// Check if card_id is unique (excluding the current card)
	if params.CardID != "" && params.CardID != oldCard.CardID {
		var count int
		err := db.QueryRow(`SELECT count(*) FROM cards
			WHERE user_id = $1 AND card_id = $2 AND id != $3 AND is_deleted = FALSE`,
			userID, params.CardID, cardPK).Scan(&count)
		if err != nil {
			return models.Card{}, fmt.Errorf("error checking card_id uniqueness: %w", err)
		}
		if count > 0 {
			return models.Card{}, fmt.Errorf("card_id already exists")
		}
	}

	var parentID int
	parent, err := getPartialCardByCardID(db, userID, discoverParentId(params.CardID))
	if err != nil {
		fmt.Printf("Parent card lookup failed for card %q: %v. Will set as root/self-parent.\n", params.CardID, err)
		parentID = cardPK
	} else if parent.ID == 0 || params.CardID == "" {
		// set parent id to id if there's no parent
		parentID = cardPK
	} else {
		parentID = parent.ID
	}

	// Determine schema_id and structured_data values for update
	var schemaID *int
	var structuredData *json.RawMessage
	if params.SchemaID == nil {
		schemaID = oldCard.SchemaID
	} else {
		schemaID = params.SchemaID
	}
	if params.StructuredData == nil {
		structuredData = oldCard.StructuredData
	} else {
		structuredData = params.StructuredData
	}

	query := `
	UPDATE cards SET title = $1, body = $2, link = $3, parent_id = $4, updated_at = NOW(), card_id = $5, card_schema_id = $6, structured_data = $7
	WHERE
	id = $8
	`
	_, err = db.Exec(query, params.Title, params.Body, params.Link, parentID, params.CardID, schemaID, structuredData, cardPK)
	if err != nil {
		return models.Card{}, fmt.Errorf("failed to update card: %w", err)
	}

	return GetCardByID(db, userID, cardPK)
}

// GetCardAnalysis retrieves the analysis/summary for a specific card.
// This is the domain data access function for card analysis.
func GetCardAnalysis(db models.Database, userID int, cardPK int) ([]models.SectionAnalysis, error) {
	// Find the most recent summarization ID for the card
	var summarizationID int
	err := db.QueryRow(`
		SELECT id FROM summarizations
		WHERE user_id = $1 AND card_pk = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, cardPK).Scan(&summarizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find summarization for card: %w", err)
	}

	// Fetch sections
	sectionRows, err := db.Query(`
		SELECT id, section_title FROM summary_sections
		WHERE user_id = $1 AND summarization_id = $2
		ORDER BY COALESCE(section_order, 0), id
	`, userID, summarizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sections: %w", err)
	}
	defer sectionRows.Close()

	var analyses []models.SectionAnalysis
	for sectionRows.Next() {
		var sectionID int
		var section models.SectionAnalysis
		if err := sectionRows.Scan(&sectionID, &section.Section); err != nil {
			return nil, fmt.Errorf("failed to scan section: %w", err)
		}

		// Fetch theses for the current section
		thesisRows, err := db.Query(`
			SELECT id, thesis FROM summary_theses
			WHERE user_id = $1 AND section_id = $2
			ORDER BY id
		`, userID, sectionID)
		if err != nil {
			return nil, fmt.Errorf("failed to query theses for section %d: %w", sectionID, err)
		}

		var theses []models.ThesisEntry
		for thesisRows.Next() {
			var thesisID int
			var thesis models.ThesisEntry
			if err := thesisRows.Scan(&thesisID, &thesis.Thesis); err != nil {
				thesisRows.Close()
				return nil, fmt.Errorf("failed to scan thesis: %w", err)
			}

			// Fetch arguments for the current thesis
			argRows, err := db.Query(`
				SELECT argument, importance FROM summary_arguments
				WHERE user_id = $1 AND thesis_id = $2
				ORDER BY id
			`, userID, thesisID)
			if err != nil {
				thesisRows.Close()
				return nil, fmt.Errorf("failed to query arguments for thesis %d: %w", thesisID, err)
			}

			var arguments []models.Argument
			for argRows.Next() {
				var arg models.Argument
				if err := argRows.Scan(&arg.Argument, &arg.Importance); err != nil {
					argRows.Close()
					thesisRows.Close()
					return nil, fmt.Errorf("failed to scan argument: %w", err)
				}
				arguments = append(arguments, arg)
			}
			argRows.Close()
			thesis.Arguments = arguments
			theses = append(theses, thesis)
		}
		thesisRows.Close()
		section.Theses = theses
		analyses = append(analyses, section)
	}

	return analyses, nil
}

// StructToMap converts a struct to a map[string]interface{}
// This is a utility function used by multiple tools
func StructToMap(obj interface{}) map[string]interface{} {
	v := reflect.ValueOf(obj)
	t := reflect.TypeOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	result := make(map[string]interface{})
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Only exported fields can be accessed
		if field.PkgPath == "" {
			result[field.Name] = fieldValue.Interface()
		}
	}
	return result
}

// Helper functions (internal to the card domain)

// getChildCards retrieves immediate children of a card
func getChildCards(db models.Database, userID int, cardID int) ([]models.PartialCard, error) {
	query := `
		SELECT id, card_id, user_id, title, parent_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND parent_id = $2 AND id != $3 AND is_deleted = FALSE
		ORDER BY card_id
	`

	rows, err := db.Query(query, userID, cardID, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.PartialCard
	for rows.Next() {
		var card models.PartialCard
		if err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&card.ParentID,
			&card.CreatedAt,
			&card.UpdatedAt,
		); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}

	return cards, nil
}

// getPartialCard retrieves a partial card by primary key
func getPartialCard(db models.Database, userID int, id int) (models.PartialCard, error) {
	var card models.PartialCard

	err := db.QueryRow(`
	SELECT
	id, card_id, user_id, title, parent_id, created_at, updated_at
	FROM cards
	WHERE is_deleted = FALSE AND id = $1 AND user_id = $2
	`, id, userID).Scan(
		&card.ID,
		&card.CardID,
		&card.UserID,
		&card.Title,
		&card.ParentID,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.PartialCard{}, fmt.Errorf("card not found")
		}
		return models.PartialCard{}, fmt.Errorf("failed to get partial card: %w", err)
	}
	return card, nil
}

// getPartialCardByCardID retrieves a partial card by card_id
func getPartialCardByCardID(db models.Database, userID int, cardID string) (models.PartialCard, error) {
	var card models.PartialCard

	err := db.QueryRow(`
	SELECT
	id, card_id, user_id, title, parent_id, created_at, updated_at
	FROM cards
	WHERE is_deleted = FALSE AND card_id = $1 AND user_id = $2
	`, cardID, userID).Scan(
		&card.ID,
		&card.CardID,
		&card.UserID,
		&card.Title,
		&card.ParentID,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.PartialCard{}, fmt.Errorf("card not found")
		}
		return models.PartialCard{}, fmt.Errorf("failed to get partial card by card_id: %w", err)
	}
	return card, nil
}

// checkIsCardIDUnique checks if a card_id is unique for a user
func checkIsCardIDUnique(db models.Database, userID int, cardID string) bool {
	if cardID == "" {
		return true
	}
	var count int
	err := db.QueryRow(`SELECT count(*) FROM cards
		WHERE user_id = $1 AND card_id = $2 AND is_deleted = FALSE`, userID, cardID).Scan(&count)
	if err != nil {
		return false
	}
	return count == 0
}

// discoverParentId extracts the parent card_id from a card_id
func discoverParentId(cardID string) string {
	// Try new format first - find last separator and remove everything after it
	lastSlash := -1
	lastDot := -1
	lastDash := -1

	for i, char := range cardID {
		switch char {
		case '/':
			lastSlash = i
		case '.':
			lastDot = i
		case '-':
			lastDash = i
		}
	}

	lastSeparatorIndex := -1
	if lastSlash > lastSeparatorIndex {
		lastSeparatorIndex = lastSlash
	}
	if lastDot > lastSeparatorIndex {
		lastSeparatorIndex = lastDot
	}
	if lastDash > lastSeparatorIndex {
		lastSeparatorIndex = lastDash
	}

	if lastSeparatorIndex == -1 {
		// No separator found, return as-is (root card)
		return cardID
	}

	// Return everything before the last separator
	return cardID[:lastSeparatorIndex]
}
