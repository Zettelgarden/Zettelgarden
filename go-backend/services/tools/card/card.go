// Package card provides card-related data access and business logic
// for the Zettelgarden tool registry.
//
// This package contains functions for managing cards, including CRUD operations,
// search functionality (text and semantic), hierarchical relationships, and
// card analysis retrieval.
package card

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

// ExecuteTextSearch performs text-based search using Typesense with SQL fallback
func ExecuteTextSearch(db models.Database, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
	// Use Typesense for text search
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		collectionName = "cards"
	}

	filter := "user_id:=" + strconv.Itoa(userID) + " && type:=card"
	sortBy := "_text_match:desc"

	typesenseParams := &api.SearchCollectionParams{
		Q:             query,
		QueryBy:       "card_id, title, preview",
		FilterBy:      &filter,
		SortBy:        &sortBy,
		PerPage:       &limit,
		ExcludeFields: pointer.String("embedding"),
	}

	typesenseResults, err := typesenseClient.Collection(collectionName).Documents().Search(context.Background(), typesenseParams)
	if err != nil {
		log.Printf("Typesense search error: %v", err)
		// Fallback to SQL search if Typesense fails
		return executeTextSearchFallback(db, userID, query, limit)
	}

	var cards []map[string]interface{}
	for _, hit := range *typesenseResults.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			if doc["type"].(string) == "card" {
				card := map[string]interface{}{
					"id":         int(doc["card_pk"].(float64)),
					"title":      doc["title"].(string),
					"body":       doc["preview"].(string),
					"card_id":    doc["card_id"].(string),
					"created_at": time.Unix(int64(doc["created_at"].(float64)), 0),
					"updated_at": time.Unix(int64(doc["updated_at"].(float64)), 0),
				}
				cards = append(cards, card)
			}
		}
	}

	return cards, nil
}

// executeTextSearchFallback provides SQL-based fallback when Typesense is unavailable
func executeTextSearchFallback(db models.Database, userID int, query string, limit int) ([]map[string]interface{}, error) {
	searchQuery := `
		SELECT id, title, body, created_at, updated_at, card_id
		FROM cards
		WHERE user_id = $1 AND (
			title ILIKE $2 OR
			body ILIKE $2 OR
			card_id ILIKE $2
		)
		ORDER BY
			CASE WHEN title ILIKE $2 THEN 1 ELSE 2 END,
			updated_at DESC
		LIMIT $3
	`

	searchPattern := "%" + query + "%"
	rows, err := db.Query(searchQuery, userID, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []map[string]interface{}
	for rows.Next() {
		var card map[string]interface{} = make(map[string]interface{})
		var title, body, cardID sql.NullString
		var id int
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &body, &createdAt, &updatedAt, &cardID)
		if err != nil {
			continue
		}

		card["id"] = id
		card["title"] = title.String
		card["body"] = body.String
		card["card_id"] = cardID.String
		card["created_at"] = createdAt
		card["updated_at"] = updatedAt

		cards = append(cards, card)
	}

	return cards, nil
}

// ExecuteSemanticSearch performs semantic search using Typesense with embeddings
func ExecuteSemanticSearch(db models.Database, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
	// Use Typesense with embedding search for semantic search
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	if collectionName == "" {
		collectionName = "cards"
	}

	filter := "user_id:=" + strconv.Itoa(userID) + " && type:=card"
	sortBy := "_text_match:desc"

	typesenseParams := &api.SearchCollectionParams{
		Q:             query,
		QueryBy:       "card_id, title, embedding", // Include embedding for semantic search
		FilterBy:      &filter,
		SortBy:        &sortBy,
		PerPage:       &limit,
		ExcludeFields: pointer.String("embedding"),
	}

	typesenseResults, err := typesenseClient.Collection(collectionName).Documents().Search(context.Background(), typesenseParams)
	if err != nil {
		log.Printf("Typesense semantic search error: %v", err)
		// Fallback to text search if semantic search fails
		return ExecuteTextSearch(db, userID, query, limit, typesenseClient)
	}

	var cards []map[string]interface{}
	for _, hit := range *typesenseResults.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			if doc["type"].(string) == "card" {
				card := map[string]interface{}{
					"id":         int(doc["card_pk"].(float64)),
					"title":      doc["title"].(string),
					"body":       doc["preview"].(string),
					"card_id":    doc["card_id"].(string),
					"created_at": time.Unix(int64(doc["created_at"].(float64)), 0),
					"updated_at": time.Unix(int64(doc["updated_at"].(float64)), 0),
				}
				cards = append(cards, card)
			}
		}
	}

	return cards, nil
}

// GetFullCard retrieves a complete card by ID for a specific user
func GetFullCard(db models.Database, userID int, cardPK int) (models.Card, error) {
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
		log.Printf("query error: %v", err)
		return models.Card{}, fmt.Errorf("unable to access card")
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
			log.Printf("Failed to get children of card %d: %v", child.ID, err)
			continue
		}
		allCards = append(allCards, grandchildren...)
	}

	return allCards, nil
}

// getChildCards retrieves immediate child cards for a given card
func getChildCards(db models.Database, userID int, cardID int) ([]models.PartialCard, error) {
	query := `
		SELECT id, card_id, user_id, title, parent_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND parent_id = $2 and id != $3 AND is_deleted = FALSE
		ORDER BY card_id
	`

	rows, err := db.Query(query, userID, cardID, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards, err := models.ScanPartialCards(rows)
	if err != nil {
		return nil, err
	}

	// Fetch tags for each card
	for i := range cards {
		tags, err := queryTagsForCard(db, userID, cards[i].ID)
		if err != nil {
			log.Printf("Failed to fetch tags for card ID %d: %v", cards[i].ID, err)
			// Continue without tags rather than failing entirely
			cards[i].Tags = []models.Tag{}
		} else {
			cards[i].Tags = tags
		}
	}

	return cards, nil
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

	// Check if card has a parent
	if card.ParentID == nil {
		// No parent, return empty slice
		return allCards, nil
	}

	// Get immediate parent
	parent, err := getPartialCard(db, userID, *card.ParentID)
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
		log.Printf("Failed to get parents of card %d: %v", parent.ID, err)
		return allCards, nil
	}
	allCards = append(allCards, grandparents...)

	return allCards, nil
}

// getPartialCard retrieves a partial card by ID
func getPartialCard(db models.Database, userID, id int) (models.PartialCard, error) {
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
		log.Printf("query partial by id err %v", err)
		return models.PartialCard{}, fmt.Errorf("something went wrong")
	}
	return card, nil
}

// CreateCard creates a new card with proper parent ID resolution and backlink handling
func CreateCard(db models.Database, userID int, params models.EditCardParams) (models.Card, error) {
	// Strip all whitespace from card_id before proceeding
	params.CardID = strings.ReplaceAll(params.CardID, " ", "")
	params.CardID = regexp.MustCompile(`\s+`).ReplaceAllString(params.CardID, "")

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
			log.Printf("Parent card lookup failed for card %q: %v. Will set as root/self-parent.", params.CardID, lookupErr)
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
		log.Printf("createcard err %v", err)
		return models.Card{}, err
	}

	// Get the created card
	newCard, err := GetFullCard(db, userID, id)
	if err != nil {
		return models.Card{}, err
	}

	// Note: AddTagsFromCard and UpsertCardToTypesense would be called by the handler
	// Create audit event for creation
	createAuditEvent(db, userID, id, "card", "create", nil, newCard)

	// set parent id to id if it's a root card or if card_id is empty
	if isRootCard || params.CardID == "" {
		_, err = db.Exec("UPDATE cards SET parent_id = $1 WHERE id = $1", id)
		if err != nil {
			return models.Card{}, err
		}
	}

	backlinks := extractBacklinks(newCard.Body)
	structuredDataBacklinks := extractBacklinksFromStructuredData(db, userID, newCard.StructuredData)
	allBacklinks := append(backlinks, structuredDataBacklinks...)
	updateBacklinks(db, newCard.ID, allBacklinks)

	return GetFullCard(db, userID, id)
}

// UpdateCard updates an existing card with proper backlink and schema handling
func UpdateCard(db models.Database, userID int, cardPK int, params models.EditCardParams) (models.Card, error) {
	// Get the old state first
	oldCard, err := GetFullCard(db, userID, cardPK)
	if err != nil {
		return models.Card{}, err
	}

	// Strip all whitespace from card_id before proceeding
	params.CardID = strings.ReplaceAll(params.CardID, " ", "")
	params.CardID = regexp.MustCompile(`\s+`).ReplaceAllString(params.CardID, "")

	// Check if card_id is unique (excluding the current card)
	if params.CardID != "" && params.CardID != oldCard.CardID {
		var count int
		err := db.QueryRow(`SELECT count(*) FROM cards
			WHERE user_id = $1 AND card_id = $2 AND id != $3 AND is_deleted = FALSE`,
			userID, params.CardID, cardPK).Scan(&count)
		if err != nil {
			log.Printf("err checking card_id uniqueness: %v", err)
			return models.Card{}, err
		}
		if count > 0 {
			return models.Card{}, fmt.Errorf("card_id already exists")
		}
	}

	var parent_id int
	parent, err := getPartialCardByCardID(db, userID, discoverParentId(params.CardID))
	if err != nil {
		log.Printf("Parent card lookup failed for card %q: %v. Will set as root/self-parent.", params.CardID, err)
		parent_id = cardPK
	} else if parent.ID == 0 || params.CardID == "" {
		// set parent id to id if there's no parent
		parent_id = cardPK
	} else {
		parent_id = parent.ID
	}

	// Determine schema_id and structured_data values for update
	// If both are nil (not provided in request), preserve existing values
	// This prevents accidentally clearing schema when updating other fields
	// If ClearSchema flag is true, explicitly clear the schema fields
	var schemaID *int
	var structuredData *json.RawMessage
	if params.ClearSchema {
		// Explicit request to clear schema - set both to nil
		schemaID = nil
		structuredData = nil
	} else if params.SchemaID == nil && params.StructuredData == nil {
		// Neither provided - preserve existing values (prevent accidental clearing)
		schemaID = oldCard.SchemaID
		structuredData = oldCard.StructuredData
	} else {
		// At least one is provided - use params values for non-nil fields, preserve existing for nil fields
		// This handles partial updates where only one schema field is being updated
		if params.SchemaID != nil {
			schemaID = params.SchemaID
		} else {
			schemaID = oldCard.SchemaID
		}
		if params.StructuredData != nil {
			structuredData = params.StructuredData
		} else {
			structuredData = oldCard.StructuredData
		}
	}

	query := `
	UPDATE cards SET title = $1, body = $2, link = $3, parent_id = $4, updated_at = NOW(), card_id = $5, card_schema_id = $6, structured_data = $7
	WHERE
	id = $8
	`
	_, err = db.Exec(query, params.Title, params.Body, params.Link, parent_id, params.CardID, schemaID, structuredData, cardPK)
	if err != nil {
		log.Printf("updatecard err %v", err)
		return models.Card{}, err
	}

	// Get the new state
	newCard, err := GetFullCard(db, userID, cardPK)
	if err != nil {
		return models.Card{}, err
	}

	// Create audit event
	createAuditEvent(db, userID, cardPK, "card", "update", oldCard, newCard)

	backlinks := extractBacklinks(newCard.Body)
	structuredDataBacklinks := extractBacklinksFromStructuredData(db, userID, newCard.StructuredData)
	allBacklinks := append(backlinks, structuredDataBacklinks...)
	updateBacklinks(db, newCard.ID, allBacklinks)

	// Note: AddTagsFromCard and UpsertCardToTypesense would be called by the handler

	return GetFullCard(db, userID, cardPK)
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
		log.Printf("err %v", err)
		return false
	}
	if count > 0 {
		return false
	} else {
		return true
	}
}

// getPartialCardByCardID retrieves a partial card by card_id string
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
		log.Printf("query partial err %v", err)
		return models.PartialCard{}, fmt.Errorf("something went wrong")
	}
	return card, nil
}

// discoverParentId supports both old alternating format and new flexible format
func discoverParentId(cardID string) string {
	// Try new format first - find last separator and remove everything after it
	parentFromNew := getParentIdNewFormat(cardID)
	if parentFromNew != cardID {
		// New format worked (found a separator and removed something)
		return parentFromNew
	}

	// If no separators found, it's a root card
	if !hasAnySeparators(cardID) {
		return cardID
	}

	// Fall back to old alternating format
	return getParentIdAlternating(cardID)
}

// getParentIdNewFormat handles the new format: remove last segment after any separator
func getParentIdNewFormat(cardID string) string {
	// Find the last occurrence of any separator
	lastSlash := strings.LastIndex(cardID, "/")
	lastDot := strings.LastIndex(cardID, ".")
	lastDash := strings.LastIndex(cardID, "-")

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

// hasAnySeparators checks if the cardID contains any of the supported separators
func hasAnySeparators(cardID string) bool {
	return strings.ContainsAny(cardID, "/.-")
}

// getParentIdAlternating handles the old alternating separator format
func getParentIdAlternating(cardID string) string {
	parts := []string{}
	currentPart := ""

	for _, char := range cardID {
		if char == '/' || char == '.' {
			parts = append(parts, currentPart)
			currentPart = ""
		} else {
			currentPart += string(char)
		}
	}

	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	if len(parts) == 1 {
		return cardID
	}

	parentID := ""
	for i := 0; i < len(parts)-1; i++ {
		parentID += parts[i]
		if i < len(parts)-2 {
			if i%2 == 0 {
				parentID += "/"
			} else {
				parentID += "."
			}
		}
	}

	return parentID
}

// extractBacklinks extracts card references from markdown text
func extractBacklinks(text string) []string {
	// Match all text within square brackets
	re := regexp.MustCompile(`\[([^\]]+)\]`)

	// Find all matches
	matches := re.FindAllStringSubmatch(text, -1)

	// Extract the first capturing group from each match
	var backlinks []string
	for _, match := range matches {
		if len(match) > 1 {
			// Check if the match is not followed by a parenthesis
			if !isMarkdownLink(text, match[0]) {
				backlinks = append(backlinks, match[1])
			}
		}
	}

	return backlinks
}

// isMarkdownLink checks if a match is part of a markdown link
func isMarkdownLink(text, match string) bool {
	// Find the position of the match in the text
	pos := strings.Index(text, match)
	if pos == -1 {
		return false
	}
	// Check if the match is followed by an opening parenthesis
	if pos+len(match) < len(text) && text[pos+len(match)] == '(' {
		return true
	}
	return false
}

// extractBacklinksFromStructuredData extracts card IDs (as human-readable card_id strings) from structured_data JSONB
// It finds all link_to_card field values and converts them from internal IDs to card_id strings
func extractBacklinksFromStructuredData(db models.Database, userID int, structuredData *json.RawMessage) []string {
	if structuredData == nil || len(*structuredData) == 0 {
		return []string{}
	}

	// Parse the structured data
	var data map[string]interface{}
	if err := json.Unmarshal(*structuredData, &data); err != nil {
		log.Printf("Error unmarshaling structured data for backlink extraction: %v", err)
		return []string{}
	}

	var backlinks []string

	// Check each field value - if it's a number, it might be a link_to_card reference
	for _, value := range data {
		if value == nil {
			continue
		}

		var internalID int
		switch v := value.(type) {
		case float64:
			internalID = int(v)
		case int:
			internalID = v
		case int64:
			internalID = int(v)
		case string:
			// Try to parse as int
			if parsedID, err := strconv.Atoi(v); err == nil {
				internalID = parsedID
			} else {
				continue
			}
		default:
			// Not a number type, skip
			continue
		}

		// Look up the human-readable card_id for this internal ID
		var cardID string
		err := db.QueryRow(`
			SELECT card_id FROM cards
			WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
		`, internalID, userID).Scan(&cardID)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Printf("Error looking up card_id for internal ID %d: %v", internalID, err)
			}
			// Card doesn't exist or was deleted, skip it
			continue
		}

		backlinks = append(backlinks, cardID)
	}

	return backlinks
}

// updateBacklinks updates the backlinks table for a card
func updateBacklinks(db models.Database, cardPK int, backlinks []string) error {
	_, err := db.Exec("DELETE FROM backlinks WHERE source_id_int = $1", cardPK)
	if err != nil {
		log.Printf("UpdateBacklinks: failed to delete backlinks: %v", err)
		return err
	}
	for _, targetID := range backlinks {
		_, err = db.Exec(`
	WITH target_id AS (
    SELECT id
    FROM cards
    WHERE card_id = $2
)
INSERT INTO backlinks (source_id_int, target_id_int, created_at, updated_at)
SELECT $1, target_id.id, NOW(), NOW()
FROM target_id;
		`,
			cardPK, targetID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// queryTagsForCard queries tags for a specific card
func queryTagsForCard(db models.Database, userID int, cardPK int) ([]models.Tag, error) {
	tags := []models.Tag{}

	query := `
        SELECT t.id, t.name, t.user_id, t.color
        FROM tags t
        JOIN card_tags ct ON t.id = ct.tag_id
        WHERE ct.card_pk = $1 AND t.user_id = $2;
        `
	var rows *sql.Rows
	var err error

	rows, err = db.Query(query, cardPK, userID)
	if err != nil {
		log.Printf("err %v", err)
		return tags, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.UserID,
			&tag.Color,
		); err != nil {
			log.Printf("err %v", err)
			return tags, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// createAuditEvent creates an audit event for tracking changes
// This is a local copy to avoid circular imports
func createAuditEvent(db models.Database, userID int, entityID int, entityType string, action string, oldState interface{}, newState interface{}) error {
	changes := make(map[string]models.FieldChange)

	// If we have both states, compute the differences
	if oldState != nil && newState != nil {
		oldVal := reflect.ValueOf(oldState)
		newVal := reflect.ValueOf(newState)

		// Handle pointer types
		if oldVal.Kind() == reflect.Ptr {
			oldVal = oldVal.Elem()
		}
		if newVal.Kind() == reflect.Ptr {
			newVal = newVal.Elem()
		}

		// Only process if both are structs
		if oldVal.Kind() == reflect.Struct && newVal.Kind() == reflect.Struct {
			for i := 0; i < oldVal.NumField(); i++ {
				field := oldVal.Type().Field(i)
				oldField := oldVal.Field(i)
				newField := newVal.Field(i)

				// Skip certain fields
				if field.Name == "CreatedAt" || field.Name == "UpdatedAt" {
					continue
				}

				// Convert interface values to comparable types
				oldValue := oldField.Interface()
				newValue := newField.Interface()

				// Only record if values are different
				if !reflect.DeepEqual(oldValue, newValue) {
					changes[field.Name] = models.FieldChange{
						From: oldValue,
						To:   newValue,
					}
				}
			}
		}
	}

	details := models.Details{
		ChangeType: action,
		Changes:    changes,
	}

	// For create/delete actions, store the full state
	if action == "create" && newState != nil {
		details.CustomData = map[string]interface{}{
			"initial_state": newState,
		}
	} else if action == "delete" && oldState != nil {
		details.CustomData = map[string]interface{}{
			"final_state": oldState,
		}
	}

	_, err := db.Exec(`
		INSERT INTO audit_events (user_id, entity_id, entity_type, action, details)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, entityID, entityType, action, details)

	if err != nil {
		log.Printf("Error creating audit event: %v", err)
		return err
	}

	return nil
}

// GetCardAnalysis reconstructs the analysis data structure from the database for a given card.
// It fetches the most recent summarization for the card.
func GetCardAnalysis(db *sql.DB, userID int, cardPK int) ([]models.SectionAnalysis, error) {
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

	log.Printf("getting %v", summarizationID)
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
			// Explicitly close argRows after use to avoid resource leaks
			if err := argRows.Close(); err != nil {
				thesisRows.Close()
				return nil, fmt.Errorf("failed to close argument rows: %w", err)
			}
			thesis.Arguments = arguments
			theses = append(theses, thesis)
		}
		// Explicitly close thesisRows after use to avoid resource leaks
		if err := thesisRows.Close(); err != nil {
			return nil, fmt.Errorf("failed to close thesis rows: %w", err)
		}
		section.Theses = theses
		analyses = append(analyses, section)
	}

	return analyses, nil
}

// StructToMap converts a struct to a map[string]interface{}
// This is used by handlers for converting structs to maps for responses
func StructToMap(data interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	val := reflect.ValueOf(data)

	// Handle pointer types
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return result
		}
		val = val.Elem()
	}

	// Only process structs
	if val.Kind() != reflect.Struct {
		return result
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Get the JSON tag name
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		// Parse the JSON tag (e.g., `json:"card_id,omitempty"`)
		tagName := strings.Split(tag, ",")[0]

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Convert the field value to a suitable interface{} value
		if fieldValue.CanInterface() {
			result[tagName] = fieldValue.Interface()
		}
	}

	return result
}
