package services

import (
	"context"
	"database/sql"
	"fmt"
	"go-backend/models"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

// Helper function to check if a match is part of a markdown link
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
func ExtractBacklinks(text string) []string {
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

func GetChildCards(db *sql.DB, userID int, cardID int) ([]models.PartialCard, error) {
	// Get the parent card's card_id first
	var parent_id int
	err := db.QueryRow("SELECT parent_id FROM cards WHERE id = $1 AND user_id = $2", cardID, userID).Scan(&parent_id)
	if err != nil {
		return nil, err
	}

	// Find child cards based on card_id hierarchy
	query := `
		SELECT id, card_id, user_id, title, parent_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND parent_id = $2 and id != $3
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

func GetParentCard(db *sql.DB, userID int, cardPK int) ([]models.PartialCard, error) {
	var results []models.PartialCard

	card, err := GetPartialCard(db, userID, cardPK)
	if err != nil {
		return results, err
	}
	parent, err := GetPartialCard(db, userID, card.ParentID)
	if err != nil {
		return results, err
	}
	results = append(results, parent)
	return results, nil
}

func ExecuteTextSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
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
func executeTextSearchFallback(db *sql.DB, userID int, query string, limit int) ([]map[string]interface{}, error) {
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

func ExecuteSemanticSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
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

func GetFullCard(db *sql.DB, userID int, cardPK int) (models.Card, error) {
	var card models.Card

	err := db.QueryRow(`
	SELECT 
	id, card_id, user_id, title, body, link, parent_id,
        created_at, updated_at
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
func GetPartialCardByCardID(db *sql.DB, userID int, cardID string) (models.PartialCard, error) {
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

func GetPartialCard(db *sql.DB, userID, id int) (models.PartialCard, error) {
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
func GetBacklinks(db *sql.DB, userID int, cardID string) ([]models.PartialCard, error) {

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

func DeleteCard(db *sql.DB, userID int, id int) error {
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

	// Start transaction to ensure all cleanup happens atomically
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// First, get all facts that originated from this card
	var factIDs []int
	factRows, err := tx.Query(`
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
		_, err = tx.Exec(`
			DELETE FROM fact_card_junction WHERE fact_id = $1
		`, factID)
		if err != nil {
			log.Printf("Error deleting fact-card junction for fact %d: %v", factID, err)
			return err
		}
	}

	// Delete the facts that originated from this card
	_, err = tx.Exec(`
		DELETE FROM facts WHERE card_pk = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		log.Printf("Error deleting facts originated from card: %v", err)
		return err
	}

	// Clean up entity-card relationships (entity_card_junction doesn't have CASCADE)
	_, err = tx.Exec(`
		DELETE FROM entity_card_junction
		WHERE card_pk = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		log.Printf("Error deleting entity-card relationships: %v", err)
		return err
	}

	// Clean up any remaining fact-card relationships for this specific card
	// (facts from other cards that reference this card)
	_, err = tx.Exec(`
		DELETE FROM fact_card_junction
		WHERE card_pk = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		log.Printf("Error deleting remaining fact-card relationships: %v", err)
		return err
	}

	// Delete the card (soft delete)
	_, err = tx.Exec(`
		UPDATE cards SET is_deleted = TRUE, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	deleteCardTypesense(card.ID)

	// Create audit event for deletion
	err = CreateAuditEvent(db, userID, id, "card", "delete", card, nil)
	if err != nil {
		log.Printf("Error creating audit event: %v", err)
		// Don't return here as deletion was successful
	}

	return nil
}

func UpdateBacklinks(db *sql.DB, cardPK int, backlinks []string) error {
	tx, _ := db.Begin()
	_, err := tx.Exec("DELETE FROM backlinks WHERE source_id_int = $1", cardPK)
	if err != nil {
		log.Fatal(err.Error())
		tx.Rollback()
		return err
	}
	for _, targetID := range backlinks {
		_, err = tx.Exec(`
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
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil

}
func UpdateCard(db *sql.DB, userID int, cardPK int, params models.EditCardParams) (models.Card, error) {
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
	parent, _ := GetPartialCardByCardID(db, userID, DiscoverParentId(params.CardID))

	// set parent id to id if there's no parent
	if parent.ID == 0 || params.CardID == "" {
		parent_id = cardPK
	} else {
		parent_id = parent.ID
	}

	query := `
	UPDATE cards SET title = $1, body = $2, link = $3, parent_id = $4, updated_at = NOW(), card_id = $5
	WHERE
	id = $6
	`
	_, err = db.Exec(query, params.Title, params.Body, params.Link, parent_id, params.CardID, cardPK)
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
	UpdateBacklinks(db, newCard.ID, backlinks)

	AddTagsFromCard(db, userID, cardPK)
	UpsertCardToTypesense(db, newCard)

	return GetFullCard(db, userID, cardPK)
}

func checkIsCardIDUnique(db *sql.DB, userID int, cardID string) bool {
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

func CreateCard(db *sql.DB, userID int, params models.EditCardParams) (models.Card, error) {
	// Strip all whitespace from card_id before proceeding
	params.CardID = strings.ReplaceAll(params.CardID, " ", "")
	params.CardID = regexp.MustCompile(`\s+`).ReplaceAllString(params.CardID, "")

	// Check if card_id is unique
	if !checkIsCardIDUnique(db, userID, params.CardID) {
		return models.Card{}, fmt.Errorf("card_id already exists")
	}

	parent, err := GetPartialCardByCardID(db, userID, DiscoverParentId(params.CardID))
	query := `
	INSERT INTO cards 
	(title, body, link, user_id, card_id, parent_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	RETURNING id;
	`
	var id int
	err = db.QueryRow(query, params.Title, params.Body, params.Link, userID, params.CardID, parent.ID).Scan(&id)
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
	UpsertCardToTypesense(db, newCard)
	// Create audit event for creation
	CreateAuditEvent(db, userID, id, "card", "create", nil, newCard)

	// set parent id to id if there's no parent
	if parent.ID == 0 || params.CardID == "" {
		_, err = db.Exec("UPDATE cards SET parent_id = $1 WHERE id = $1", id)
		if err != nil {
			return models.Card{}, err
		}
	}

	backlinks := ExtractBacklinks(newCard.Body)
	UpdateBacklinks(db, newCard.ID, backlinks)

	return GetFullCard(db, userID, id)
}

// GetCardsByEntity retrieves all cards linked to a specific entity
func GetCardsByEntity(db *sql.DB, userID int, entityID int) ([]models.PartialCard, error) {
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
		if err := rows.Scan(&c.ID, &c.CardID, &c.UserID, &c.Title, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}
