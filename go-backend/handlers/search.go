package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

type SearchParams struct {
	Tags           []string
	Terms          []string
	NegateTags     []string
	NegateTerms    []string
	Entities       []string
	NegateEntities []string
}

func (sp *SearchParams) HasAdvancedFilters() bool {
	return len(sp.Entities) > 0 || len(sp.NegateEntities) > 0
}

func (s *Handler) InitSearchCollection() {
	start := time.Now()
	var cardCount, factCount, entityCount int

	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	log.Printf("lets go")

	rows, err := s.DB.Query(`
		SELECT
	    c.id,
	    c.card_id,
	    c.user_id,
	    c.title,
		c.body,
	    c.created_at,
	    c.updated_at,
		c.parent_id,
		STRING_AGG(CASE WHEN t.name IS NOT NULL AND t.is_deleted = FALSE THEN t.name END, ',' ORDER BY t.name) as tag_names
	FROM cards c
	LEFT JOIN card_tags ct ON c.id = ct.card_pk
	LEFT JOIN tags t ON ct.tag_id = t.id
	WHERE c.is_deleted = FALSE
	GROUP BY c.id, c.card_id, c.user_id, c.title, c.body, c.created_at, c.updated_at, c.parent_id
		`)
	if err != nil {
		log.Printf("error querying cards: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var createdAtTime, updatedAtTime time.Time
			var cardPK int
			var cardID string
			var userID int
			var parentID int
			var title, body string
			var tagNames sql.NullString
			err := rows.Scan(
				&cardPK,
				&cardID,
				&userID,
				&title,
				&body,
				&createdAtTime,
				&updatedAtTime,
				&parentID,
				&tagNames,
			)
			if err != nil {
				log.Printf("error scanning fact: %v", err)
				continue
			}
			// Parse tags from comma-separated string
			var tags []string
			if tagNames.Valid && tagNames.String != "" {
				tags = strings.Split(tagNames.String, ",")
				// Trim whitespace from each tag
				for i, tag := range tags {
					tags[i] = strings.TrimSpace(tag)
				}
			} else {
				tags = []string{}
			}

			doc := map[string]interface{}{
				"id":                    "card-" + strconv.Itoa(cardPK),
				"fact_pk":               -1,
				"card_id":               cardID,
				"card_pk":               cardPK,
				"entity_pk":             -1,
				"user_id":               userID,
				"type":                  "card",
				"title":                 title,
				"preview":               body,
				"parent_id":             parentID,
				"created_at":            createdAtTime.Unix(),
				"updated_at":            updatedAtTime.Unix(),
				"linked_card_id":        "",
				"linked_card_pk":        -1,
				"linked_card_title":     "",
				"linked_card_parent_id": -1,
				"tags":                  tags,
			}

			// Upsert (insert or overwrite if exists)
			_, err = s.Server.TypesenseClient.Collection(collectionName).
				Documents().
				Upsert(context.Background(), doc)

			if err != nil {
				log.Printf("failed to upsert card ID %d: %v", cardPK, err)
			}
			cardCount++
		}
	}
	//Index all facts
	rows, err = s.DB.Query(`
		SELECT f.id, f.fact, f.created_at, f.updated_at, f.user_id,
		       c.id, c.card_id, c.user_id, c.title, c.parent_id,
		       c.created_at, c.updated_at
		FROM facts f
		JOIN cards c ON f.card_pk = c.id
		WHERE c.is_deleted = FALSE
	`)
	if err != nil {
		log.Printf("error querying facts: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var factID int
			var factText string
			var createdAtTime, updatedAtTime time.Time
			var cardPK int
			var cardCardID string
			var userID, parentID int
			var cardTitle string
			var cardCreatedAt, cardUpdatedAt time.Time
			err := rows.Scan(
				&factID, &factText, &createdAtTime, &updatedAtTime, &userID,
				&cardPK, &cardCardID, &userID, &cardTitle, &parentID, &cardCreatedAt, &cardUpdatedAt,
			)
			if err != nil {
				log.Printf("error scanning fact: %v", err)
				continue
			}
			doc := map[string]interface{}{
				"id":                    "fact-" + strconv.Itoa(factID),
				"fact_pk":               factID,
				"card_id":               "",
				"card_pk":               -1,
				"entity_pk":             -1,
				"user_id":               userID,
				"type":                  "fact",
				"title":                 factText,
				"preview":               "",
				"score":                 0.0,
				"parent_id":             -1,
				"created_at":            createdAtTime.Unix(),
				"updated_at":            updatedAtTime.Unix(),
				"linked_card_id":        cardCardID,
				"linked_card_pk":        cardPK,
				"linked_card_title":     cardTitle,
				"linked_card_parent_id": parentID,
				"tags":                  []string{},
			}
			_, err = s.Server.TypesenseClient.Collection(collectionName).Documents().Upsert(context.Background(), doc)
			if err != nil {
				log.Printf("failed to upsert fact ID %d: %v", factID, err)
			}
			factCount++
		}
	}

	// // Index all entities
	rows2, err := s.DB.Query(`
		SELECT e.id, e.name, e.description, e.type, e.created_at, e.updated_at, e.user_id,
		c.id, c.card_id, c.title, c.parent_id
		FROM entities e
		LEFT JOIN cards c ON e.card_pk = c.id
	`) // assuming user_id=1 here

	if err != nil {
		log.Printf("error querying entities: %v", err)
	} else {
		defer rows2.Close()
		for rows2.Next() {
			var entityID int
			var name, description, etype string
			var createdAtTime, updatedAtTime time.Time
			var userID int
			var parentID int
			var cardPK sql.NullInt64
			var cardCardID, cardTitle sql.NullString
			var cardParentID sql.NullInt64

			err := rows2.Scan(
				&entityID, &name, &description, &etype, &createdAtTime,
				&updatedAtTime, &userID, &cardPK, &cardCardID, &cardTitle, &cardParentID,
			)
			if err != nil {
				log.Printf("error scanning entity: %v", err)
				continue
			}

			var linkedCardID string
			if cardCardID.Valid {
				linkedCardID = cardCardID.String
			} else {
				linkedCardID = ""
			}

			var linkedCardTitle string
			if cardTitle.Valid {
				linkedCardTitle = cardTitle.String
			} else {
				linkedCardTitle = ""
			}

			var linkedCardPK int32
			if cardPK.Valid {
				linkedCardPK = int32(cardPK.Int64) // Downcast to int32 since Typesense schema requires int32
			} else {
				linkedCardPK = -1 // or 0, depending on how you want to indicate "no value"
			}
			doc := map[string]interface{}{
				"id":                    "entity-" + strconv.Itoa(entityID),
				"entity_pk":             entityID,
				"card_id":               "",
				"card_pk":               -1,
				"fact_pk":               -1,
				"type":                  "entity",
				"user_id":               userID,
				"title":                 name,
				"parent_id":             -1,
				"preview":               description,
				"score":                 0.0,
				"created_at":            createdAtTime.Unix(),
				"updated_at":            updatedAtTime.Unix(),
				"linked_card_id":        linkedCardID,
				"linked_card_pk":        linkedCardPK,
				"linked_card_title":     linkedCardTitle,
				"linked_card_parent_id": parentID,
				"tags":                  []string{},
			}
			_, err = s.Server.TypesenseClient.Collection(collectionName).Documents().Upsert(context.Background(), doc)
			if err != nil {
				log.Printf("failed to upsert entity ID %d: %v", entityID, err)
			}
			entityCount++
		}
	}

	log.Printf(
		"Indexed %d cards, %d facts, %d entities. Total: %d items. Took %s.",
		cardCount, factCount, entityCount,
		cardCount+factCount+entityCount,
		time.Since(start),
	)
}

func contains[T comparable](collection []T, target T) bool {
	for _, v := range collection {
		if v == target {
			return true
		}
	}
	return false
}
func sanitizeSearchInput(input string) string {
	// Limit input length to prevent DoS
	if len(input) > 1000 {
		input = input[:1000]
	}

	// Remove or escape potentially dangerous characters
	// Keep alphanumeric, spaces, and valid search syntax characters
	var result strings.Builder
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			result.WriteRune(r)
		case r == ' ', r == '@', r == '[', r == ']', r == '!', r == '#', r == '-', r == '_', r == '.', r == '\'':
			result.WriteRune(r)
		case r == '\\':
			// Allow escaped characters but don't double-escape
			result.WriteRune(r)
		default:
			// Skip potentially dangerous characters
			continue
		}
	}

	return result.String()
}

func ParseSearchText(input string) SearchParams {
	// Sanitize input first
	input = sanitizeSearchInput(input)

	var searchParams SearchParams
	var currentEntity strings.Builder
	inEntity := false

	// Split the input string by spaces, but preserve spaces within @[...]
	parts := strings.Fields(input)

	for i := 0; i < len(parts); i++ {
		part := parts[i]

		// Handle entity start
		if strings.HasPrefix(part, "@[") {
			inEntity = true
			// Remove @[ prefix
			currentEntity.WriteString(strings.TrimPrefix(part, "@["))

			// If the entity name ends in this part
			if strings.HasSuffix(part, "]") {
				entityName := currentEntity.String()
				entityName = strings.TrimSuffix(entityName, "]")
				searchParams.Entities = append(searchParams.Entities, entityName)
				currentEntity.Reset()
				inEntity = false
				continue
			}
			continue
		}

		// Handle entity start with negation
		if strings.HasPrefix(part, "!@[") {
			inEntity = true
			// Remove !@[ prefix
			currentEntity.WriteString(strings.TrimPrefix(part, "!@["))

			// If the entity name ends in this part
			if strings.HasSuffix(part, "]") {
				entityName := currentEntity.String()
				entityName = strings.TrimSuffix(entityName, "]")
				searchParams.NegateEntities = append(searchParams.NegateEntities, entityName)
				currentEntity.Reset()
				inEntity = false
				continue
			}
			continue
		}

		// Handle middle or end of entity name
		if inEntity {
			if strings.HasSuffix(part, "]") {
				currentEntity.WriteString(" ")
				currentEntity.WriteString(strings.TrimSuffix(part, "]"))
				entityName := currentEntity.String()
				if strings.HasPrefix(parts[i-1], "!") {
					searchParams.NegateEntities = append(searchParams.NegateEntities, entityName)
				} else {
					searchParams.Entities = append(searchParams.Entities, entityName)
				}
				currentEntity.Reset()
				inEntity = false
				continue
			}
			currentEntity.WriteString(" ")
			currentEntity.WriteString(part)
			continue
		}

		// Handle existing conditions
		if strings.HasPrefix(part, "#") {
			searchParams.Tags = append(searchParams.Tags, strings.TrimPrefix(part, "#"))
		} else if strings.HasPrefix(part, "!#") {
			searchParams.NegateTags = append(searchParams.NegateTags, strings.TrimPrefix(part, "!#"))
		} else if strings.HasPrefix(part, "!") {
			searchParams.NegateTerms = append(searchParams.NegateTerms, strings.TrimPrefix(part, "!"))
		} else {
			searchParams.Terms = append(searchParams.Terms, part)
		}
	}

	return searchParams
}
func BuildPartialCardSqlSearchTermString(searchString string, fullText bool, schemaID *int) string {
	searchParams := ParseSearchText(searchString)

	var result string
	var termConditions []string
	var tagConditions []string
	var negateTagsConditions []string
	var excludeTerms []string
	var entityConditions []string
	var negateEntityConditions []string

	// Add conditions for terms that search both card_id and title
	for _, term := range searchParams.Terms {
		// Escape single quotes to prevent SQL injection
		escapedTerm := strings.ReplaceAll(term, "'", "''")
		// Use ILIKE for case-insensitive pattern matching
		var termCondition string
		if fullText {
			termCondition = fmt.Sprintf("(card_id ILIKE '%%%s%%' OR title ILIKE '%%%s%%' OR body ILIKE '%%%s%%')", escapedTerm, escapedTerm, escapedTerm)

		} else {
			termCondition = fmt.Sprintf("(card_id ILIKE '%%%s%%' OR title ILIKE '%%%s%%')", escapedTerm, escapedTerm)

		}
		termConditions = append(termConditions, termCondition)
	}

	for _, term := range searchParams.NegateTerms {
		// Escape single quotes to prevent SQL injection
		escapedTerm := strings.ReplaceAll(term, "'", "''")
		var excludeCondition string
		if fullText {
			excludeCondition = fmt.Sprintf("NOT (card_id ILIKE '%%%s%%' OR title ILIKE '%%%s%%' OR body ILIKE '%%%s%%')", escapedTerm, escapedTerm, escapedTerm)
		} else {
			excludeCondition = fmt.Sprintf("NOT (card_id ILIKE '%%%s%%' OR title ILIKE '%%%s%%')", escapedTerm, escapedTerm)
		}
		excludeTerms = append(excludeTerms, excludeCondition)
	}

	// Add conditions for tags
	for _, tag := range searchParams.Tags {
		// Escape single quotes to prevent SQL injection
		escapedTag := strings.ReplaceAll(tag, "'", "''")
		tagCondition := fmt.Sprintf(`EXISTS (
            SELECT 1 FROM card_tags
            JOIN tags ON card_tags.tag_id = tags.id
            WHERE card_tags.card_pk = c.id AND tags.name = '%s' AND tags.is_deleted = FALSE
        )`, escapedTag)
		tagConditions = append(tagConditions, tagCondition)
	}
	// Build SQL for tags that should NOT exist
	for _, tag := range searchParams.NegateTags {
		// Escape single quotes to prevent SQL injection
		escapedTag := strings.ReplaceAll(tag, "'", "''")
		tagCondition := fmt.Sprintf(`NOT EXISTS (
            SELECT 1 FROM card_tags
            JOIN tags ON card_tags.tag_id = tags.id
            WHERE card_tags.card_pk = c.id AND tags.name = '%s' AND tags.is_deleted = FALSE
        )`, escapedTag)
		negateTagsConditions = append(negateTagsConditions, tagCondition)
	}

	// Add conditions for entities
	for _, entity := range searchParams.Entities {
		// Escape single quotes to prevent SQL injection
		escapedEntity := strings.ReplaceAll(entity, "'", "''")
		entityCondition := fmt.Sprintf(`EXISTS (
            SELECT 1 FROM entity_card_junction ecj
            JOIN entities e ON ecj.entity_id = e.id
            WHERE ecj.card_pk = c.id AND e.name = '%s'
        )`, escapedEntity)
		entityConditions = append(entityConditions, entityCondition)
	}

	// Add conditions for negated entities
	for _, entity := range searchParams.NegateEntities {
		// Escape single quotes to prevent SQL injection
		escapedEntity := strings.ReplaceAll(entity, "'", "''")
		entityCondition := fmt.Sprintf(`NOT EXISTS (
            SELECT 1 FROM entity_card_junction ecj
            JOIN entities e ON ecj.entity_id = e.id
            WHERE ecj.card_pk = c.id AND e.name = '%s'
        )`, escapedEntity)
		negateEntityConditions = append(negateEntityConditions, entityCondition)
	}

	if len(tagConditions) > 0 {
		result = " AND (" + strings.Join(tagConditions, " AND ") + ")"
	}

	if len(termConditions) > 0 {
		result += " AND (" + strings.Join(termConditions, " AND ") + ")"
	}
	if len(excludeTerms) > 0 {
		excludeClause := strings.Join(excludeTerms, " AND ")
		result += " AND (" + excludeClause + ")"
	}
	if len(negateTagsConditions) > 0 {
		negateTagClause := strings.Join(negateTagsConditions, " AND ")
		result += " AND (" + negateTagClause + ")"
	}
	if len(entityConditions) > 0 {
		result += " AND (" + strings.Join(entityConditions, " AND ") + ")"
	}
	if len(negateEntityConditions) > 0 {
		result += " AND (" + strings.Join(negateEntityConditions, " AND ") + ")"
	}
	if schemaID != nil {
		result += " AND c.card_schema_id = " + strconv.Itoa(*schemaID)
	}
	return result
}

func BuildPartialEntitySqlSearchTermString(searchString string) string {
	searchParams := ParseSearchText(searchString)

	var result string
	var termConditions []string
	var tagConditions []string
	var negateTagsConditions []string
	var excludeTerms []string

	// Add conditions for terms that search both name and description
	for _, term := range searchParams.Terms {
		// Escape single quotes to prevent SQL injection
		escapedTerm := strings.ReplaceAll(term, "'", "''")
		termCondition := fmt.Sprintf("(name ILIKE '%%%s%%' OR description ILIKE '%%%s%%' OR type ILIKE '%%%s%%')", escapedTerm, escapedTerm, escapedTerm)
		termConditions = append(termConditions, termCondition)
	}

	// Add conditions for negated terms
	for _, term := range searchParams.NegateTerms {
		// Escape single quotes to prevent SQL injection
		escapedTerm := strings.ReplaceAll(term, "'", "''")
		excludeCondition := fmt.Sprintf("NOT (name ILIKE '%%%s%%' OR description ILIKE '%%%s%%' OR type ILIKE '%%%s%%')", escapedTerm, escapedTerm, escapedTerm)
		excludeTerms = append(excludeTerms, excludeCondition)
	}

	// Add conditions for tags
	for _, tag := range searchParams.Tags {
		// Escape single quotes to prevent SQL injection
		escapedTag := strings.ReplaceAll(tag, "'", "''")
		tagCondition := fmt.Sprintf("EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = '%s' AND tags.is_deleted = FALSE)", escapedTag)
		tagConditions = append(tagConditions, tagCondition)
	}

	// Build SQL for tags that should NOT exist
	for _, tag := range searchParams.NegateTags {
		// Escape single quotes to prevent SQL injection
		escapedTag := strings.ReplaceAll(tag, "'", "''")
		tagCondition := fmt.Sprintf("NOT EXISTS (SELECT 1 FROM card_tags JOIN tags ON card_tags.tag_id = tags.id WHERE card_tags.card_pk = ecj.card_pk AND tags.name = '%s' AND tags.is_deleted = FALSE)", escapedTag)
		negateTagsConditions = append(negateTagsConditions, tagCondition)
	}

	// Add each tag condition separately
	for _, tagCondition := range tagConditions {
		result += fmt.Sprintf(" AND (%s)", tagCondition)
	}

	if len(termConditions) > 0 {
		result += " AND (" + strings.Join(termConditions, " OR ") + ")"
	}

	if len(excludeTerms) > 0 {
		for _, excludeTerm := range excludeTerms {
			result += " AND (" + excludeTerm + ")"
		}
	}

	// Add each negate tag condition separately
	for _, negateTagCondition := range negateTagsConditions {
		result += fmt.Sprintf(" AND (%s)", negateTagCondition)
	}

	return result
}

func (s *Handler) ClassicCardSearch(userID int, params SearchRequestParams) ([]models.Card, int, error) {
	searchString := BuildPartialCardSqlSearchTermString(params.SearchTerm, params.FullText, params.SchemaID)

	// Add empty card_id filter if requested
	if params.OnlyEmptyCardId {
		searchString += " AND (c.card_id = '' OR c.card_id IS NULL)"
	}

	// Set pagination defaults
	page := params.Page
	if page < 1 {
		page = 1
	}

	perPage := params.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	// First, get the total count
	countQuery := `
	SELECT COUNT(DISTINCT c.id)
	FROM cards c
	LEFT JOIN card_tags ct ON c.id = ct.card_pk
	WHERE c.user_id = $1 AND c.is_deleted = FALSE
	` + searchString

	var total int
	err := s.DB.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Then get the paginated results
	query := `
	SELECT
    c.id,
    c.card_id,
    c.user_id,
    c.title,
    c.body,
    c.link,
    c.parent_id,
    c.created_at,
    c.updated_at,
    COUNT(ct.tag_id) AS tag_count
FROM cards c
LEFT JOIN card_tags ct ON c.id = ct.card_pk -- Use LEFT JOIN to include cards with no tags
WHERE c.user_id = $1 AND c.is_deleted = FALSE
` + searchString + `
GROUP BY
    c.id,
    c.card_id,
    c.user_id,
    c.title,
    c.body,
    c.link,
    c.parent_id,
    c.created_at,
    c.updated_at
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3
	`
	rows, err := s.DB.Query(query, userID, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	cards, err := models.ScanCards(rows)
	if err != nil {
		return nil, 0, err
	}

	return cards, total, nil
}

type SearchRequestParams struct {
	SearchTerm      string `json:"search_term"`
	FullText        bool   `json:"full_text"`
	ShowEntities    bool   `json:"show_entities"`
	ShowFacts       bool   `json:"show_facts"`
	ShowCards       bool   `json:"show_cards"`
	OnlyEmptyCardId bool   `json:"only_empty_card_id"`
	SchemaID        *int   `json:"schema_id,omitempty"`
	SortBy          string `json:"sort"`
	Rerank          bool   `json:"rerank"`
	Page            int    `json:"page"`
	PerPage         int    `json:"per_page"`
}

type PaginatedSearchResponse struct {
	Results    []models.SearchResult `json:"results"`
	Page       int                   `json:"page"`
	PerPage    int                   `json:"per_page"`
	Total      int                   `json:"total"`
	TotalPages int                   `json:"total_pages"`
}

func (s *Handler) TypesenseSearch(searchParams SearchRequestParams, userID int) (PaginatedSearchResponse, error) {

	// Set default pagination values if not provided
	page := searchParams.Page
	if page < 1 {
		page = 1
	}

	perPage := searchParams.PerPage
	if perPage <= 0 {
		perPage = 50 // Default page size
	}
	if perPage > 100 {
		perPage = 100 // Maximum page size
	}
	var sortBy string
	if searchParams.SearchTerm == "" {
		sortBy = "created_at:desc"
	} else {
		switch searchParams.SortBy {
		case "sortByRanking":
			sortBy = "_text_match:desc"
		case "sortCreatedNewOld":
			sortBy = "created_at:desc"
		case "sortCreatedOldNew":
			sortBy = "created_at:asc"
		case "sortNewOld":
			sortBy = "updated_at:desc"
		case "sortOldNew":
			sortBy = "updated_at:asc"
		case "sortBigSmall":
			sortBy = "title:asc"
		case "sortSmallBig":
			sortBy = "title:desc"
		default:
			sortBy = "_text_match:desc"
		}
	}
	filter := "user_id:=" + strconv.Itoa(userID)

	var typeFilters []string
	if !searchParams.ShowFacts {
		typeFilters = append(typeFilters, "type:!=fact")
	}
	if !searchParams.ShowEntities {
		typeFilters = append(typeFilters, "type:!=entity")
	}
	if !searchParams.ShowCards {
		typeFilters = append(typeFilters, "type:!=card")
	}
	if len(typeFilters) > 0 {
		filter += " && " + strings.Join(typeFilters, " && ")
	}

	// Parse search term to handle tag filtering
	parsedParams := ParseSearchText(searchParams.SearchTerm)

	// Add tag filters
	var tagFilters []string
	for _, tag := range parsedParams.Tags {
		tagFilters = append(tagFilters, "tags:="+tag)
	}
	for _, tag := range parsedParams.NegateTags {
		tagFilters = append(tagFilters, "tags:!="+tag)
	}
	if len(tagFilters) > 0 {
		filter += " && " + strings.Join(tagFilters, " && ")
	}

	// Remove tag and entity syntax from search term for Typesense query
	cleanSearchTerm := searchParams.SearchTerm
	for _, tag := range parsedParams.Tags {
		cleanSearchTerm = strings.ReplaceAll(cleanSearchTerm, "#"+tag, "")
	}
	for _, tag := range parsedParams.NegateTags {
		cleanSearchTerm = strings.ReplaceAll(cleanSearchTerm, "!#"+tag, "")
	}
	for _, entity := range parsedParams.Entities {
		cleanSearchTerm = strings.ReplaceAll(cleanSearchTerm, "@["+entity+"]", "")
	}
	for _, entity := range parsedParams.NegateEntities {
		cleanSearchTerm = strings.ReplaceAll(cleanSearchTerm, "!@["+entity+"]", "")
	}
	cleanSearchTerm = strings.TrimSpace(cleanSearchTerm)

	var results []models.SearchResult
	searchTerm := cleanSearchTerm
	if searchTerm == "" {
		searchTerm = "*"
	}

	typesenseParams := &api.SearchCollectionParams{
		Q:             searchTerm,
		QueryBy:       "card_id, title, tags, embedding",
		FilterBy:      &filter,
		SortBy:        &sortBy,
		PerPage:       &perPage,
		Page:          &page,
		ExcludeFields: pointer.String("embedding"),
	}
	collectionName := os.Getenv("TYPESENSE_COLLECTION")
	log.Printf("searching typesense %v", s.Server.TypesenseClient)
	typesenseResults, err := s.Server.TypesenseClient.Collection(collectionName).Documents().Search(context.Background(), typesenseParams)

	if err != nil {
		log.Printf("Search error: %v", err)
		return PaginatedSearchResponse{}, err
	}

	fmt.Printf("Found %d docs\n", *typesenseResults.Found)
	for i, hit := range *typesenseResults.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			// Extract tags from document
			var tags []models.Tag
			if tagSlice, ok := doc["tags"].([]interface{}); ok {
				for _, tagInterface := range tagSlice {
					if tagName, ok := tagInterface.(string); ok {
						tags = append(tags, models.Tag{
							Name: tagName,
						})
					}
				}
			}

			item := models.SearchResult{
				Title:   doc["title"].(string),
				Type:    doc["type"].(string),
				Preview: doc["preview"].(string),
				Score:   0.0,
				//				Score:     doc["_text_match"].(float64),
				CreatedAt: time.Unix(int64(doc["created_at"].(float64)), 0),
				UpdatedAt: time.Unix(int64(doc["updated_at"].(float64)), 0),
				Tags:      tags,
			}
			resultType := doc["type"]
			if resultType == "card" {

				item.ID = strconv.FormatInt(int64(doc["card_pk"].(float64)), 10)
				cardID := doc["card_id"].(string)
				metadata := map[string]interface{}{
					"id":        item.ID,
					"card_id":   cardID,
					"parent_id": "",
				}
				item.Metadata = metadata

			} else if resultType == "entity" {
				if !searchParams.ShowEntities {
					continue
				}
				item.ID = strconv.FormatInt(int64(doc["entity_pk"].(float64)), 10)
				metadata := map[string]interface{}{
					"id":                    item.ID,
					"linked_card_id":        doc["linked_card_id"].(string),
					"linked_card_pk":        strconv.FormatInt(int64(doc["linked_card_pk"].(float64)), 10),
					"linked_card_title":     doc["linked_card_title"].(string),
					"linked_card_parent_id": strconv.FormatInt(int64(doc["linked_card_parent_id"].(float64)), 10),
				}
				item.Metadata = metadata
				// entity_pk

			} else if resultType == "fact" {
				item.ID = strconv.FormatInt(int64(doc["fact_pk"].(float64)), 10)
				item.Preview = item.Title
				metadata := map[string]interface{}{
					"id":                    item.ID,
					"linked_card_id":        doc["linked_card_id"].(string),
					"linked_card_pk":        strconv.FormatInt(int64(doc["linked_card_pk"].(float64)), 10),
					"linked_card_title":     doc["linked_card_title"].(string),
					"linked_card_parent_id": strconv.FormatInt(int64(doc["linked_card_parent_id"].(float64)), 10),
				}
				item.Metadata = metadata
				// fact_pk

			}
			results = append(results, item)
		} else {
			log.Printf("[%d] unexpected document format: %v", i, hit.Document)
		}
	}

	// var reranked []models.SearchResult
	// if s.Server.Testing {
	// 	reranked = results
	// } else if !searchParams.Rerank {
	// 	reranked = results
	// } else {
	// 	log.Printf("reranking")
	// 	if len(results) > 0 {
	// 		client := services.NewDefaultClient(s.DB, userID)
	// 		reranked, err = llms.RerankSearchResults(client, searchParams.SearchTerm, results)
	// 		if err != nil {
	// 			return results, nil
	// 		}
	// 	}
	// }
	// Calculate pagination info
	total := int(*typesenseResults.Found)
	totalPages := (total + perPage - 1) / perPage

	return PaginatedSearchResponse{
		Results:    results,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}
func convertCardsToSearchResults(cards []models.Card, total int, page int, perPage int) PaginatedSearchResponse {
	var searchResults []models.SearchResult
	for _, card := range cards {
		searchResult := models.SearchResult{
			ID:        strconv.Itoa(card.ID),
			Title:     card.Title,
			Type:      "card",
			Preview:   card.Body, // Assuming Card has a Body field for preview
			Score:     0.0,       // Classic search doesn't provide a score
			CreatedAt: card.CreatedAt,
			UpdatedAt: card.UpdatedAt,
			Metadata: map[string]interface{}{
				"id":        strconv.Itoa(card.ID),
				"card_id":   card.CardID,
				"parent_id": card.ParentID,
			},
		}
		searchResults = append(searchResults, searchResult)
	}

	totalPages := (total + perPage - 1) / perPage

	return PaginatedSearchResponse{
		Results:    searchResults,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

func (s *Handler) SearchRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	var reqParams SearchRequestParams
	err := json.NewDecoder(r.Body).Decode(&reqParams)
	if err != nil {
		log.Printf("json decode error %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var response PaginatedSearchResponse
	parsedParams := ParseSearchText(reqParams.SearchTerm)

	// If the search contains entities, tags, or onlyEmptyCardId filter, use ClassicCardSearch, otherwise use Typesense
	// Note: Typesense cannot filter for empty card_id since it's not an optional field
	if parsedParams.HasAdvancedFilters() || reqParams.OnlyEmptyCardId || reqParams.SchemaID != nil {
		cards, total, err := s.ClassicCardSearch(userID, reqParams)
		if err != nil {
			log.Printf("ClassicCardSearch error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set pagination defaults for classic search response
		page := reqParams.Page
		if page < 1 {
			page = 1
		}
		perPage := reqParams.PerPage
		if perPage <= 0 {
			perPage = 50
		}
		if perPage > 100 {
			perPage = 100
		}

		response = convertCardsToSearchResults(cards, total, page, perPage)
	} else {
		response, err = s.TypesenseSearch(reqParams, userID)
		if err != nil {
			log.Printf("TypesenseSearch error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
