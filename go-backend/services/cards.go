package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/services/backlink"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

// Delegate to shared backlink package
func ExtractBacklinks(text string) []string {
	return backlink.ExtractBacklinks(text)
}

// ExtractBacklinksFromStructuredData extracts card IDs (as human-readable card_id strings) from structured_data JSONB
// It only extracts values from fields that are defined as link_to_card type in the schema.
// If schema is nil, returns empty slice since we cannot determine which fields are links.
// Delegate to shared backlink package
func ExtractBacklinksFromStructuredData(db models.Database, userID int, structuredData *json.RawMessage, schema *models.SchemaDefinition) []string {
	return backlink.ExtractBacklinksFromStructuredData(db, userID, structuredData, schema)
}

func GetChildCards(db models.Database, userID int, cardID int) ([]models.PartialCard, error) {
	// Find child cards based on card_id hierarchy
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
		tags, err := QueryTagsForCard(db, userID, cards[i].ID)
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

func GetParentCard(db models.Database, userID int, cardPK int) ([]models.PartialCard, error) {
	var results []models.PartialCard

	card, err := GetPartialCard(db, userID, cardPK)
	if err != nil {
		return results, err
	}
	if card.ParentID == nil {
		return results, nil
	}
	parent, err := GetPartialCard(db, userID, *card.ParentID)
	if err != nil {
		return results, err
	}
	results = append(results, parent)
	return results, nil
}

// GetChildCardsWithDepth recursively retrieves child cards up to the specified depth.
// depth of 1 returns only immediate children (same as GetChildCards).
// depth of -1 returns all descendants (unlimited depth).
func GetChildCardsWithDepth(db models.Database, userID int, cardID int, depth int) ([]models.PartialCard, error) {
	var allCards []models.PartialCard

	if depth == 0 {
		return allCards, nil
	}

	// Get immediate children
	children, err := GetChildCards(db, userID, cardID)
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

// GetParentCardsWithDepth recursively retrieves parent cards up to the specified depth.
// depth of 1 returns only the immediate parent (same as GetParentCard).
// depth of -1 returns all ancestors (unlimited depth).
func GetParentCardsWithDepth(db models.Database, userID int, cardPK int, depth int) ([]models.PartialCard, error) {
	var allCards []models.PartialCard

	if depth == 0 {
		return allCards, nil
	}

	// Get the current card
	card, err := GetPartialCard(db, userID, cardPK)
	if err != nil {
		return allCards, err
	}

	// Check if card has a parent
	if card.ParentID == nil {
		// No parent, return empty slice
		return allCards, nil
	}

	// Get immediate parent
	parent, err := GetPartialCard(db, userID, *card.ParentID)
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
			title LIKE $2 OR
			body LIKE $2 OR
			card_id LIKE $2
		)
		ORDER BY
			CASE WHEN title LIKE $2 THEN 1 ELSE 2 END,
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

	// Fetch source RSS article if this card was created from one
	if card.Link != "" {
		var article models.RSSArticle
		var content, author sql.NullString
		var publishedAt sql.NullTime
		var cardID sql.NullInt64

		err := db.QueryRow(`
			SELECT id, user_id, feed_id, title, content, author, url,
			       published_at, fetched_at, read, card_id
			FROM rss_articles
			WHERE user_id = $1 AND url = $2
		`, userID, card.Link).Scan(
			&article.ID, &article.UserID, &article.FeedID, &article.Title,
			&content, &author, &article.URL, &publishedAt,
			&article.FetchedAt, &article.Read, &cardID,
		)
		if err == nil {
			if content.Valid {
				article.Content = &content.String
			}
			if author.Valid {
				article.Author = &author.String
			}
			if publishedAt.Valid {
				article.PublishedAt = &publishedAt.Time
			}
			if cardID.Valid {
				cardIDInt := int(cardID.Int64)
				article.CardID = &cardIDInt
			}
			card.SourceArticle = &article
		}
		// If no article found, card.SourceArticle remains nil - that's fine
	}

	return card, nil
}
func GetPartialCardByCardID(db models.Database, userID int, cardID string) (models.PartialCard, error) {
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

func GetPartialCard(db models.Database, userID, id int) (models.PartialCard, error) {
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

func GetBacklinks(db models.Database, userID int, cardID string) ([]models.PartialCard, error) {

	query := `
	SELECT
    cards.id,
	cards.card_id,
    cards.user_id,
    cards.title,
    cards.created_at,
    cards.updated_at
FROM backlinks
JOIN cards ON backlinks.source_id_int = cards.id
JOIN cards target_card ON backlinks.target_id_int = target_card.id
WHERE target_card.card_id = $1 AND cards.user_id = $2 AND cards.is_deleted = FALSE;`

	rows, err := db.Query(query, cardID, userID)
	if err != nil {
		log.Printf("cardid %v", cardID)
		log.Printf("err %v", err)
	}
	var cards []models.PartialCard

	for rows.Next() {
		card := models.PartialCard{}
		if err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&card.CreatedAt,
			&card.UpdatedAt,
		); err != nil {
			log.Printf("err %v", err)
			return cards, err
		}

		if card.CardID != cardID {
			cards = append(cards, card)
		}
	}
	return cards, nil

}

// getParentId supports both old alternating format and new flexible format
func DiscoverParentId(cardID string) string {
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

func DeleteCard(db models.Database, userID int, id int) error {
	// Get the card before deletion for audit
	card, err := GetFullCard(db, userID, id)
	if err != nil {
		return err
	}

	backlinks, _ := GetBacklinks(db, userID, card.CardID)
	if len(backlinks) > 0 {
		return fmt.Errorf("card has backlinks, cannot be deleted")
	}
	children, err := GetChildCards(db, userID, card.ID)
	if len(children) > 0 {
		return fmt.Errorf("card has children, cannot be deleted")
	}

	// First, get all facts that originated from this card
	var factIDs []int
	factRows, err := db.Query(`
		SELECT id FROM facts WHERE card_pk = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		log.Printf("Error querying facts for card: %v", err)
		return err
	}
	defer factRows.Close()

	for factRows.Next() {
		var factID int
		if err := factRows.Scan(&factID); err != nil {
			log.Printf("Error scanning fact ID: %v", err)
			continue
		}
		factIDs = append(factIDs, factID)
	}

	// Delete all fact_card_junction entries for facts that originated from this card
	// This includes relationships to other cards, not just this card
	for _, factID := range factIDs {
		_, err = db.Exec(`
			DELETE FROM fact_card_junction WHERE fact_id = $1
		`, factID)
		if err != nil {
			log.Printf("Error deleting fact-card junction for fact %d: %v", factID, err)
			return err
		}
	}

	// Delete the facts that originated from this card
	_, err = db.Exec(`
		DELETE FROM facts WHERE card_pk = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		log.Printf("Error deleting facts originated from card: %v", err)
		return err
	}

	// Clean up entity-card relationships (entity_card_junction doesn't have CASCADE)
	_, err = db.Exec(`
		DELETE FROM entity_card_junction
		WHERE card_pk = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		log.Printf("Error deleting entity-card relationships: %v", err)
		return err
	}

	// Clean up any remaining fact-card relationships for this specific card
	// (facts from other cards that reference this card)
	_, err = db.Exec(`
		DELETE FROM fact_card_junction
		WHERE card_pk = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		log.Printf("Error deleting remaining fact-card relationships: %v", err)
		return err
	}

	// Delete the card (soft delete)
	_, err = db.Exec(`
		UPDATE cards SET is_deleted = TRUE, updated_at = $3
		WHERE id = $1 AND user_id = $2
	`, id, userID, time.Now().UTC())
	if err != nil {
		return err
	}

	deleteCardTypesense(card.ID)

	// user_stats was trigger-maintained (0093); now maintained in Go (Phase 5).
	DecrementUserCardCount(db, userID)

	// Create audit event for deletion
	err = CreateAuditEvent(db, userID, id, "card", "delete", card, nil)
	if err != nil {
		log.Printf("Error creating audit event: %v", err)
		// Don't return here as deletion was successful
	}

	return nil
}

// Delegate to shared backlink package
func UpdateBacklinks(db models.Database, cardPK int, backlinks []string) error {
	return backlink.UpdateBacklinks(db, cardPK, backlinks)
}
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
	parent, err := GetPartialCardByCardID(db, userID, DiscoverParentId(params.CardID))
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
	UPDATE cards SET title = $1, body = $2, link = $3, parent_id = $4, updated_at = $9, card_id = $5, card_schema_id = $6, structured_data = $7
	WHERE
	id = $8
	`
	_, err = db.Exec(query, params.Title, params.Body, params.Link, parent_id, params.CardID, schemaID, structuredData, cardPK, time.Now().UTC())
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
	CreateAuditEvent(db, userID, cardPK, "card", "update", oldCard, newCard)

	backlinks := ExtractBacklinks(newCard.Body)
	var schema *models.SchemaDefinition
	if newCard.SchemaID != nil {
		schema = backlink.GetSchemaByID(db, userID, *newCard.SchemaID)
	}
	structuredDataBacklinks := ExtractBacklinksFromStructuredData(db, userID, newCard.StructuredData, schema)
	allBacklinks := append(backlinks, structuredDataBacklinks...)
	UpdateBacklinks(db, newCard.ID, allBacklinks)

	AddTagsFromCard(db, userID, cardPK)
	UpsertCardToTypesense(db, newCard)

	return GetFullCard(db, userID, cardPK)
}

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

// getNextRootCardID generates the next root card ID for a user
func GetNextRootCardID(db models.Database, userID int) (string, error) {
	var result string

	// Query to get the highest numeric card_id
	query := `
        SELECT card_id
        FROM cards
        WHERE user_id = $1
        AND is_deleted = FALSE
        AND card_id ~ '^[0-9]+$'  -- Only match pure numeric card_ids
        ORDER BY CAST(card_id AS INTEGER) DESC
        LIMIT 1
    `

	err := db.QueryRow(query, userID).Scan(&result)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error finding next root card ID: %v", err)
		return "1", nil // Default to 1 if there's an error
	}

	if result == "" {
		return "1", nil // If no cards exist, start with 1
	}

	// Convert the highest card_id to int and increment
	highestNumber, err := strconv.Atoi(result)
	if err != nil {
		log.Printf("Error converting card_id to number: %v", err)
		return "1", nil
	}

	return strconv.Itoa(highestNumber + 1), nil
}

func CreateCard(db models.Database, userID int, params models.EditCardParams) (models.Card, error) {
	// Strip all whitespace from card_id before proceeding
	params.CardID = strings.ReplaceAll(params.CardID, " ", "")
	params.CardID = regexp.MustCompile(`\s+`).ReplaceAllString(params.CardID, "")

	// Check if card_id is unique
	if !checkIsCardIDUnique(db, userID, params.CardID) {
		return models.Card{}, fmt.Errorf("card_id already exists")
	}

	discoveredParentCardID := DiscoverParentId(params.CardID)
	var parentID int
	var isRootCard bool

	// Check if this is a root card (no parent separator in card_id)
	if discoveredParentCardID == params.CardID {
		// Root card - will set parent_id to itself after insert
		isRootCard = true
		parentID = 0
	} else {
		// Child card - try to find parent
		parent, lookupErr := GetPartialCardByCardID(db, userID, discoveredParentCardID)
		if lookupErr != nil {
			log.Printf("Parent card lookup failed for card %q: %v. Will set as root/self-parent.", params.CardID, lookupErr)
			isRootCard = true
			parentID = 0
		} else {
			parentID = parent.ID
		}
	}

	// App-side timestamps (not NOW()): SQLite has no NOW(), and binding
	// time.Now().UTC() explicitly works on both drivers and makes tests
	// deterministic. See migration design doc Phase 3 / D5.
	now := time.Now().UTC()
	query := `
	INSERT INTO cards
	(title, body, link, user_id, card_id, parent_id, card_schema_id, structured_data, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id;
	`
	var id int
	err := db.QueryRow(query, params.Title, params.Body, params.Link, userID, params.CardID, parentID, params.SchemaID, params.StructuredData, now, now).Scan(&id)
	if err != nil {
		log.Printf("updatecard err %v", err)
		return models.Card{}, err
	}

	// Get the created card
	newCard, err := GetFullCard(db, userID, id)
	if err != nil {
		return models.Card{}, err
	}

	AddTagsFromCard(db, userID, id)
	// user_stats was trigger-maintained (0093); now maintained in Go (Phase 5).
	IncrementUserCardCount(db, userID)
	UpsertCardToTypesense(db, newCard)
	// Create audit event for creation
	CreateAuditEvent(db, userID, id, "card", "create", nil, newCard)

	// set parent id to id if it's a root card or if card_id is empty
	if isRootCard || params.CardID == "" {
		_, err = db.Exec("UPDATE cards SET parent_id = $1 WHERE id = $1", id)
		if err != nil {
			return models.Card{}, err
		}
	}

	backlinks := ExtractBacklinks(newCard.Body)
	var schema *models.SchemaDefinition
	if newCard.SchemaID != nil {
		schema = backlink.GetSchemaByID(db, userID, *newCard.SchemaID)
	}
	structuredDataBacklinks := ExtractBacklinksFromStructuredData(db, userID, newCard.StructuredData, schema)
	allBacklinks := append(backlinks, structuredDataBacklinks...)
	UpdateBacklinks(db, newCard.ID, allBacklinks)

	return GetFullCard(db, userID, id)
}

// GetCardsByEntity retrieves all cards linked to a specific entity
func GetCardsByEntity(db models.Database, userID int, entityID int) ([]models.PartialCard, error) {
	rows, err := db.Query(`
		SELECT c.id, c.card_id, c.user_id, c.title, c.parent_id, c.created_at, c.updated_at
		FROM cards c
		JOIN entity_card_junction ecj ON c.id = ecj.card_pk
		WHERE ecj.entity_id = $1 AND ecj.user_id = $2 AND c.is_deleted = FALSE
		ORDER BY c.updated_at DESC
	`, entityID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.PartialCard
	for rows.Next() {
		var c models.PartialCard
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.CardID, &c.UserID, &c.Title, &parentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			c.ParentID = new(int)
			*c.ParentID = int(parentID.Int64)
		}
		cards = append(cards, c)
	}
	return cards, nil
}

// GetCardWithDescendants fetches a card and recursively loads all its descendants with depth information
func GetCardWithDescendants(db models.Database, userID int, cardID int) (models.CardWithDescendants, error) {
	// Fetch the root card
	card := models.CardWithDescendants{}
	err := db.QueryRow(`
		SELECT id, card_id, user_id, title, body, link, parent_id, created_at, updated_at
		FROM cards
		WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`, cardID, userID).Scan(
		&card.ID,
		&card.CardID,
		&card.UserID,
		&card.Title,
		&card.Body,
		&card.Link,
		&card.ParentID,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.CardWithDescendants{}, fmt.Errorf("card not found")
		}
		return models.CardWithDescendants{}, err
	}

	card.Depth = 0
	card.Descendants = []models.CardWithDescendants{}

	// Recursively fetch descendants
	descendants, err := getDescendantsRecursive(db, userID, cardID, 1)
	if err != nil {
		return models.CardWithDescendants{}, err
	}
	card.Descendants = descendants

	return card, nil
}

// GetCardWithDescendantsLimited fetches a card and recursively loads its descendants up to a maximum depth (performance optimization)
func GetCardWithDescendantsLimited(db models.Database, userID int, cardID int, maxDepth int) (models.CardWithDescendants, error) {
	// Fetch the root card
	card := models.CardWithDescendants{}
	err := db.QueryRow(`
		SELECT id, card_id, user_id, title, body, link, parent_id, created_at, updated_at
		FROM cards
		WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`, cardID, userID).Scan(
		&card.ID,
		&card.CardID,
		&card.UserID,
		&card.Title,
		&card.Body,
		&card.Link,
		&card.ParentID,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.CardWithDescendants{}, fmt.Errorf("card not found")
		}
		return models.CardWithDescendants{}, err
	}

	card.Depth = 0
	if maxDepth > 0 {
		// Recursively fetch descendants up to maxDepth
		descendants, err := getDescendantsRecursiveLimited(db, userID, cardID, 1, maxDepth)
		if err != nil {
			return models.CardWithDescendants{}, err
		}
		card.Descendants = descendants
	} else {
		// No depth limit - return empty descendants
		card.Descendants = []models.CardWithDescendants{}
	}

	return card, nil
}

// getDescendantsRecursiveLimited is a helper function that recursively fetches descendants up to maxDepth
func getDescendantsRecursiveLimited(db models.Database, userID int, parentCardID int, depth int, maxDepth int) ([]models.CardWithDescendants, error) {
	// If we've reached maxDepth, stop recursing
	if depth > maxDepth {
		return []models.CardWithDescendants{}, nil
	}

	// Query direct children
	rows, err := db.Query(`
		SELECT id, card_id, user_id, title, body, link, parent_id, created_at, updated_at
		FROM cards
		WHERE parent_id = $1 AND user_id = $2 AND is_deleted = FALSE AND id != $1
		ORDER BY card_id
	`, parentCardID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descendants []models.CardWithDescendants
	for rows.Next() {
		card := models.CardWithDescendants{}
		err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&card.Body,
			&card.Link,
			&card.ParentID,
			&card.CreatedAt,
			&card.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning descendant card: %v", err)
			continue
		}

		card.Depth = depth
		// Recursively fetch children if we haven't reached maxDepth
		children, err := getDescendantsRecursiveLimited(db, userID, card.ID, depth+1, maxDepth)
		if err != nil {
			log.Printf("Error fetching descendants for card %d: %v", card.ID, err)
			// Continue rather than failing entirely
		} else {
			card.Descendants = children
		}

		descendants = append(descendants, card)
	}

	return descendants, nil
}

// getDescendantsRecursive is a helper function that recursively fetches descendants at a given depth
func getDescendantsRecursive(db models.Database, userID int, parentCardID int, depth int) ([]models.CardWithDescendants, error) {
	// Query direct children
	rows, err := db.Query(`
		SELECT id, card_id, user_id, title, body, link, parent_id, created_at, updated_at
		FROM cards
		WHERE parent_id = $1 AND user_id = $2 AND is_deleted = FALSE AND id != $1
		ORDER BY card_id
	`, parentCardID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descendants []models.CardWithDescendants
	for rows.Next() {
		card := models.CardWithDescendants{}
		err := rows.Scan(
			&card.ID,
			&card.CardID,
			&card.UserID,
			&card.Title,
			&card.Body,
			&card.Link,
			&card.ParentID,
			&card.CreatedAt,
			&card.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning descendant card: %v", err)
			continue
		}

		card.Depth = depth
		card.Descendants = []models.CardWithDescendants{}

		// Recursively fetch children of this card
		children, err := getDescendantsRecursive(db, userID, card.ID, depth+1)
		if err != nil {
			log.Printf("Error fetching descendants for card %d: %v", card.ID, err)
			// Continue rather than failing entirely
		} else {
			card.Descendants = children
		}

		descendants = append(descendants, card)
	}

	return descendants, nil
}

// GetAuditEventByID fetches a single audit event by ID
func GetAuditEventByID(db models.Database, userID int, eventID int) (models.AuditEvent, error) {
	var event models.AuditEvent

	err := db.QueryRow(`
		SELECT id, user_id, entity_id, entity_type, action, details, created_at
		FROM audit_events
		WHERE id = $1 AND user_id = $2
	`, eventID, userID).Scan(
		&event.ID,
		&event.UserID,
		&event.EntityID,
		&event.EntityType,
		&event.Action,
		&event.Details,
		&event.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.AuditEvent{}, fmt.Errorf("audit event not found")
		}
		return models.AuditEvent{}, err
	}

	return event, nil
}

// reconstructCardStateFromAudit reconstructs the card state from an audit event
// For 'create' events: uses CustomData.initial_state
// For 'update' events: applies changes in reverse (uses 'from' values)
func reconstructCardStateFromAudit(event models.AuditEvent, currentCard models.Card) models.EditCardParams {
	params := models.EditCardParams{}

	// Default to current state
	params.Title = currentCard.Title
	params.Body = currentCard.Body
	params.Link = currentCard.Link
	params.CardID = currentCard.CardID

	if event.Action == "create" {
		// For create events, use the initial state if available
		if initialState, ok := event.Details.CustomData["initial_state"]; ok {
			// initialState is a map[string]interface{}
			if stateMap, ok := initialState.(map[string]interface{}); ok {
				if title, ok := stateMap["title"].(string); ok {
					params.Title = title
				}
				if body, ok := stateMap["body"].(string); ok {
					params.Body = body
				}
				if link, ok := stateMap["link"]; ok && link != nil {
					if linkStr, ok := link.(string); ok {
						params.Link = linkStr
					}
				}
				if cardID, ok := stateMap["card_id"].(string); ok {
					params.CardID = cardID
				}
			}
		}
	} else if event.Action == "update" {
		// For update events, apply the 'from' values from changes
		for fieldName, change := range event.Details.Changes {
			switch fieldName {
			case "Title":
				if fromValue, ok := change.From.(string); ok {
					params.Title = fromValue
				}
			case "Body":
				if fromValue, ok := change.From.(string); ok {
					params.Body = fromValue
				}
			case "Link":
				if fromValue, ok := change.From.(string); ok && fromValue != "" {
					params.Link = fromValue
				}
			case "CardID":
				if fromValue, ok := change.From.(string); ok {
					params.CardID = fromValue
				}
			}
		}
	} else if event.Action == "delete" {
		// For delete events, use the final state (pre-deletion state)
		if finalState, ok := event.Details.CustomData["final_state"]; ok {
			if stateMap, ok := finalState.(map[string]interface{}); ok {
				if title, ok := stateMap["title"].(string); ok {
					params.Title = title
				}
				if body, ok := stateMap["body"].(string); ok {
					params.Body = body
				}
				if link, ok := stateMap["link"]; ok && link != nil {
					if linkStr, ok := link.(string); ok {
						params.Link = linkStr
					}
				}
				if cardID, ok := stateMap["card_id"].(string); ok {
					params.CardID = cardID
				}
			}
		}
	}

	return params
}

// RestoreCardToAuditEvent restores a card to the state it was in at the time of the audit event
func RestoreCardToAuditEvent(db models.Database, userID int, cardPK int, auditEventID int) (models.Card, error) {
	// Verify the user owns the card
	currentCard, err := GetFullCard(db, userID, cardPK)
	if err != nil {
		return models.Card{}, fmt.Errorf("card not found")
	}

	// Fetch the audit event
	auditEvent, err := GetAuditEventByID(db, userID, auditEventID)
	if err != nil {
		return models.Card{}, fmt.Errorf("audit event not found")
	}

	// Verify the audit event belongs to this card
	if auditEvent.EntityID != cardPK || auditEvent.EntityType != "card" {
		return models.Card{}, fmt.Errorf("audit event does not belong to this card")
	}

	// Reconstruct the card state from the audit event
	params := reconstructCardStateFromAudit(auditEvent, currentCard)

	// Update the card with the reconstructed state
	// This will create a new audit event automatically
	restoredCard, err := UpdateCard(db, userID, cardPK, params)
	if err != nil {
		return models.Card{}, fmt.Errorf("failed to restore card: %w", err)
	}

	return restoredCard, nil
}

// GetCardsBySharedEntities finds cards that share entities with the source card
// Returns a map of cardID to score (higher = more shared entities)
func GetCardsBySharedEntities(db models.Database, userID int, sourceCardID int) (map[int]int, error) {
	// First, get all entity IDs for the source card
	entityQuery := `
		SELECT DISTINCT entity_id
		FROM entity_card_junction
		WHERE card_pk = $1 AND user_id = $2
	`

	rows, err := db.Query(entityQuery, sourceCardID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card entities: %w", err)
	}
	defer rows.Close()

	var entityIDs []int
	for rows.Next() {
		var entityID int
		if err := rows.Scan(&entityID); err != nil {
			return nil, fmt.Errorf("failed to scan entity ID: %w", err)
		}
		entityIDs = append(entityIDs, entityID)
	}

	if len(entityIDs) == 0 {
		return make(map[int]int), nil
	}

	// Now find all cards that share these entities
	query := `
		SELECT ecj.card_pk, COUNT(DISTINCT ecj.entity_id) as shared_count
		FROM entity_card_junction ecj
		JOIN cards c ON ecj.card_pk = c.id
		WHERE ecj.entity_id = ANY($1)
		  AND ecj.user_id = $2
		  AND ecj.card_pk != $3
		  AND c.is_deleted = FALSE
		GROUP BY ecj.card_pk
	`

	rows, err = db.Query(query, pq.Array(entityIDs), userID, sourceCardID)
	if err != nil {
		return nil, fmt.Errorf("failed to find shared entity cards: %w", err)
	}
	defer rows.Close()

	scores := make(map[int]int)
	for rows.Next() {
		var cardID, sharedCount int
		if err := rows.Scan(&cardID, &sharedCount); err != nil {
			return nil, fmt.Errorf("failed to scan shared entity card: %w", err)
		}
		// Each shared entity adds 3 points
		scores[cardID] = sharedCount * 3
	}

	return scores, nil
}

// GetCardsBySharedTags finds cards that share tags with the source card
// Returns a map of cardID to score (higher = more shared tags)
func GetCardsBySharedTags(db models.Database, userID int, sourceCardID int) (map[int]int, error) {
	// First, get all tag IDs for the source card
	tagQuery := `
		SELECT DISTINCT ct.tag_id
		FROM card_tags ct
		JOIN cards c ON ct.card_pk = c.id
		WHERE ct.card_pk = $1
		  AND c.user_id = $2
		  AND c.is_deleted = FALSE
	`

	rows, err := db.Query(tagQuery, sourceCardID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card tags: %w", err)
	}
	defer rows.Close()

	var tagIDs []int
	for rows.Next() {
		var tagID int
		if err := rows.Scan(&tagID); err != nil {
			return nil, fmt.Errorf("failed to scan tag ID: %w", err)
		}
		tagIDs = append(tagIDs, tagID)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
	}

	if len(tagIDs) == 0 {
		return make(map[int]int), nil
	}

	// Now find all cards that share these tags
	query := `
		SELECT ct.card_pk, COUNT(DISTINCT ct.tag_id) as shared_count
		FROM card_tags ct
		JOIN cards c ON ct.card_pk = c.id
		JOIN tags t ON ct.tag_id = t.id
		WHERE ct.tag_id = ANY($1)
		  AND c.user_id = $2
		  AND ct.card_pk != $3
		  AND c.is_deleted = FALSE
		  AND t.is_deleted = FALSE
		GROUP BY ct.card_pk
	`

	rows, err = db.Query(query, pq.Array(tagIDs), userID, sourceCardID)
	if err != nil {
		return nil, fmt.Errorf("failed to find shared tag cards: %w", err)
	}
	defer rows.Close()

	const sharedTagScoreWeight = 1
	scores := make(map[int]int)
	for rows.Next() {
		var cardID, sharedCount int
		if err := rows.Scan(&cardID, &sharedCount); err != nil {
			return nil, fmt.Errorf("failed to scan shared tag card: %w", err)
		}
		// Each shared tag adds 1 point
		scores[cardID] = sharedCount * sharedTagScoreWeight
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating shared tag rows: %w", err)
	}

	return scores, nil
}

// UpdateCardStructuredData updates only the schema_id and structured_data fields of a card
// This is a focused update that doesn't touch title, body, or other card fields
func UpdateCardStructuredData(db models.Database, userID int, cardPK int, schemaID *int, structuredData *json.RawMessage) (models.Card, error) {
	// Get the current card state for audit
	oldCard, err := GetFullCard(db, userID, cardPK)
	if err != nil {
		return models.Card{}, err
	}

	// Update only the structured data fields
	query := `
	UPDATE cards SET card_schema_id = $1, structured_data = $2, updated_at = $5
	WHERE id = $3 AND user_id = $4
	`
	result, err := db.Exec(query, schemaID, structuredData, cardPK, userID, time.Now().UTC())
	if err != nil {
		log.Printf("UpdateCardStructuredData err: %v", err)
		return models.Card{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.Card{}, err
	}
	if rowsAffected == 0 {
		return models.Card{}, fmt.Errorf("card not found or not owned by user")
	}

	// Get the new state
	newCard, err := GetFullCard(db, userID, cardPK)
	if err != nil {
		return models.Card{}, err
	}

	// Create audit event
	CreateAuditEvent(db, userID, cardPK, "card", "update_structured_data", oldCard, newCard)

	// Update backlinks from structured data
	var schema *models.SchemaDefinition
	if newCard.SchemaID != nil {
		schema = backlink.GetSchemaByID(db, userID, *newCard.SchemaID)
	}
	structuredDataBacklinks := ExtractBacklinksFromStructuredData(db, userID, newCard.StructuredData, schema)
	// Also include backlinks from body
	bodyBacklinks := ExtractBacklinks(newCard.Body)
	allBacklinks := append(bodyBacklinks, structuredDataBacklinks...)
	UpdateBacklinks(db, newCard.ID, allBacklinks)

	// Update search index
	UpsertCardToTypesense(db, newCard)

	return newCard, nil
}

// MergeStructuredData merges new data into existing structured data
// Both existing and new data must conform to the provided schema
func MergeStructuredData(db models.Database, userID int, existingData *json.RawMessage, newData json.RawMessage, schema *models.SchemaDefinition) (json.RawMessage, error) {
	// Parse existing data
	existing := make(map[string]interface{})
	if existingData != nil && len(*existingData) > 0 {
		if err := json.Unmarshal(*existingData, &existing); err != nil {
			return nil, fmt.Errorf("failed to parse existing structured data: %w", err)
		}
	}

	// Parse new data
	newVals := make(map[string]interface{})
	if len(newData) > 0 {
		if err := json.Unmarshal(newData, &newVals); err != nil {
			return nil, fmt.Errorf("failed to parse new structured data: %w", err)
		}
	}

	// Build a map of field definitions for quick lookup
	fieldMap := make(map[string]models.FieldDefinition)
	for _, field := range schema.Fields {
		fieldMap[field.Name] = field
	}

	// Merge new values into existing
	for fieldName, value := range newVals {
		// Only merge fields that are defined in the schema
		if _, exists := fieldMap[fieldName]; exists {
			existing[fieldName] = value
		}
		// Fields not in schema are silently ignored (will be cleaned by validation)
	}

	// Marshal merged data
	mergedJSON, err := json.Marshal(existing)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged data: %w", err)
	}

	return mergedJSON, nil
}
