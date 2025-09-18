package llms

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

// Tool represents a tool that can be called by the LLM
type Tool struct {
	Definition openai.Tool
	Handler    func(args map[string]interface{}, userID int, db *sql.DB, typesenseClient *typesense.Client, conversationID *string, messageID *string) (map[string]interface{}, error)
}

// ToolRegistry holds all available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry with all available tools
func NewToolRegistry() *ToolRegistry {
	registry := &ToolRegistry{
		tools: make(map[string]Tool),
	}

	// Register all tools
	registry.registerSearchCards()
	registry.registerGetCardByID()
	registry.registerBrowseCardHierarchy()
	registry.registerFilterCardsByMetadata()
	registry.registerTask()

	return registry
}

// GetToolDefinitions returns all tool definitions for OpenAI API
func (tr *ToolRegistry) GetToolDefinitions() []openai.Tool {
	var tools []openai.Tool
	for _, tool := range tr.tools {
		tools = append(tools, tool.Definition)
	}
	return tools
}

// ExecuteTool executes a tool by name with given arguments
func (tr *ToolRegistry) ExecuteTool(name string, args map[string]interface{}, userID int, db *sql.DB, typesenseClient *typesense.Client, conversationID, messageID *string) (map[string]interface{}, error) {
	tool, exists := tr.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	start := time.Now()
	result, err := tool.Handler(args, userID, db, typesenseClient, conversationID, messageID)
	executionTime := int(time.Since(start).Milliseconds())

	// Log tool execution for analytics
	go func() {
		logErr := logToolExecution(db, userID, name, args, result, executionTime, err, conversationID, messageID)
		if logErr != nil {
			log.Printf("Error logging tool execution: %v", logErr)
		}
	}()

	return result, err
}

// Tool implementations

func (tr *ToolRegistry) registerSearchCards() {
	tr.tools["search_cards"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "search_cards",
				Description: "Search for cards in the user's knowledge base using text or semantic similarity. Returns relevant cards based on the search query.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query to find relevant cards",
						},
						"search_type": map[string]interface{}{
							"type":        "string",
							"description": "Type of search: 'text' for exact text matching, 'semantic' for meaning-based search",
							"enum":        []string{"text", "semantic"},
							"default":     "semantic",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of cards to return (default: 10, max: 50)",
							"default":     10,
							"minimum":     1,
							"maximum":     50,
						},
					},
					"required": []string{"query"},
				},
			},
		},
		Handler: handleSearchCards,
	}
}

func (tr *ToolRegistry) registerGetCardByID() {
	tr.tools["get_card_by_id"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_card_by_id",
				Description: "Retrieve a specific card by its ID. Returns the full card content including title, body, tags, and metadata.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"card_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the card to retrieve",
						},
					},
					"required": []string{"card_id"},
				},
			},
		},
		Handler: handleGetCardByID,
	}
}

func (tr *ToolRegistry) registerBrowseCardHierarchy() {
	tr.tools["browse_card_hierarchy"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "browse_card_hierarchy",
				Description: "Browse the hierarchical structure of cards. Get parent or child cards of a specific card.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"card_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the card to browse from",
						},
						"direction": map[string]interface{}{
							"type":        "string",
							"description": "Direction to browse: 'children' for child cards, 'parent' for parent card",
							"enum":        []string{"children", "parent"},
						},
					},
					"required": []string{"card_id", "direction"},
				},
			},
		},
		Handler: handleBrowseCardHierarchy,
	}
}

func (tr *ToolRegistry) registerFilterCardsByMetadata() {
	tr.tools["filter_cards_by_metadata"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "filter_cards_by_metadata",
				Description: "Filter cards by metadata like creation date, starred status, or tags.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"starred_only": map[string]interface{}{
							"type":        "boolean",
							"description": "If true, only return starred cards",
						},
						"tags": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "Filter by specific tags",
						},
						"created_after": map[string]interface{}{
							"type":        "string",
							"description": "Filter cards created after this date (YYYY-MM-DD format)",
						},
						"created_before": map[string]interface{}{
							"type":        "string",
							"description": "Filter cards created before this date (YYYY-MM-DD format)",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of cards to return (default: 20, max: 100)",
							"default":     20,
							"minimum":     1,
							"maximum":     100,
						},
					},
				},
			},
		},
		Handler: handleFilterCardsByMetadata,
	}
}

func (tr *ToolRegistry) registerTask() {
	tr.tools["Task"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "Task",
				Description: "Launch a specialized subagent to handle complex, multi-step tasks autonomously. Use this for complex research, searches, or when you need to perform multiple knowledge base operations while preserving the main conversation context.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"description": map[string]interface{}{
							"type":        "string",
							"description": "A short (3-5 word) description of the task",
						},
						"prompt": map[string]interface{}{
							"type":        "string",
							"description": "The detailed task for the agent to perform autonomously",
						},
						"subagent_type": map[string]interface{}{
							"type":        "string",
							"description": "The type of specialized agent to use",
							"enum":        []string{"general-purpose"},
							"default":     "general-purpose",
						},
					},
					"required": []string{"description", "prompt", "subagent_type"},
				},
			},
		},
		Handler: handleTask,
	}
}

// Tool handlers

func handleSearchCards(args map[string]interface{}, userID int, db *sql.DB, typesenseClient *typesense.Client, conversationID *string, messageID *string) (map[string]interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	searchType := "semantic"
	if st, ok := args["search_type"].(string); ok {
		searchType = st
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Execute search based on type
	var cards []map[string]interface{}
	var err error

	if searchType == "text" {
		cards, err = executeTextSearch(db, userID, query, limit, typesenseClient)
	} else {
		cards, err = executeSemanticSearch(db, userID, query, limit, typesenseClient)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	return map[string]interface{}{
		"cards":       cards,
		"query":       query,
		"search_type": searchType,
		"total":       len(cards),
	}, nil
}

func handleGetCardByID(args map[string]interface{}, userID int, db *sql.DB, typesenseClient *typesense.Client, conversationID *string, messageID *string) (map[string]interface{}, error) {
	cardIDFloat, ok := args["card_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("card_id parameter is required")
	}
	cardID := int(cardIDFloat)

	card, err := getFullCard(db, userID, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	return card, nil
}

func handleBrowseCardHierarchy(args map[string]interface{}, userID int, db *sql.DB, typesenseClient *typesense.Client, conversationID *string, messageID *string) (map[string]interface{}, error) {
	cardIDFloat, ok := args["card_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("card_id parameter is required")
	}
	cardID := int(cardIDFloat)

	direction, ok := args["direction"].(string)
	if !ok {
		return nil, fmt.Errorf("direction parameter is required")
	}

	var cards []map[string]interface{}
	var err error

	if direction == "children" {
		cards, err = getChildCards(db, userID, cardID)
	} else if direction == "parent" {
		cards, err = getParentCard(db, userID, cardID)
	} else {
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to browse hierarchy: %v", err)
	}

	return map[string]interface{}{
		"cards":     cards,
		"direction": direction,
		"total":     len(cards),
	}, nil
}

func handleFilterCardsByMetadata(args map[string]interface{}, userID int, db *sql.DB, typesenseClient *typesense.Client, conversationID *string, messageID *string) (map[string]interface{}, error) {
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	cards, err := filterCards(db, userID, args, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to filter cards: %v", err)
	}

	return map[string]interface{}{
		"cards": cards,
		"total": len(cards),
	}, nil
}

func handleTask(args map[string]interface{}, userID int, db *sql.DB, typesenseClient *typesense.Client, conversationID *string, messageID *string) (map[string]interface{}, error) {
	description, ok := args["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description parameter is required")
	}

	prompt, ok := args["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt parameter is required")
	}

	subagentType, ok := args["subagent_type"].(string)
	if !ok {
		subagentType = "general-purpose"
	}

	log.Printf("Launching subagent - Description: %s, Type: %s", description, subagentType)

	// Execute the subagent task
	result, err := executeSubagentTask(prompt, subagentType, userID, db, typesenseClient, conversationID, messageID)
	if err != nil {
		return nil, fmt.Errorf("subagent execution failed: %v", err)
	}

	return map[string]interface{}{
		"status":        "completed",
		"description":   description,
		"subagent_type": subagentType,
		"result":        result,
	}, nil
}

// executeSubagentTask runs a subagent with access to knowledge base tools
func executeSubagentTask(prompt, subagentType string, userID int, db *sql.DB, typesenseClient *typesense.Client, conversationID *string, messageID *string) (string, error) {
	// Create LLM client for the subagent
	client := NewDefaultClient(db, userID)

	// Create a tool registry for the subagent (excluding the Task tool to prevent recursion)
	subagentRegistry := &ToolRegistry{
		tools: make(map[string]Tool),
	}
	subagentRegistry.registerSearchCards()
	subagentRegistry.registerGetCardByID()
	subagentRegistry.registerBrowseCardHierarchy()
	subagentRegistry.registerFilterCardsByMetadata()

	tools := subagentRegistry.GetToolDefinitions()

	// System prompt for the subagent
	systemPrompt := `You are a specialized research assistant with access to a user's knowledge base. Your task is to help answer questions and gather information using the available tools.

Available tools:
- search_cards: Search for cards using text or semantic similarity
- get_card_by_id: Retrieve a specific card by its ID
- browse_card_hierarchy: Browse parent/child relationships between cards
- filter_cards_by_metadata: Filter cards by dates, tags, or starred status

Be thorough in your research and provide comprehensive, well-organized results. Use multiple tools if needed to gather complete information.`

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	// Execute the subagent conversation with potential tool calls
	maxIterations := 5 // Prevent infinite loops
	for i := 0; i < maxIterations; i++ {
		resp, err := client.Client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    client.Model,
				Messages: messages,
				Tools:    tools,
			},
		)
		if err != nil {
			return "", fmt.Errorf("LLM request failed: %v", err)
		}

		assistantMessage := resp.Choices[0].Message
		messages = append(messages, assistantMessage)

		// If no tool calls, we're done
		if len(assistantMessage.ToolCalls) == 0 {
			return assistantMessage.Content, nil
		}

		// Execute tool calls
		for _, tc := range assistantMessage.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				log.Printf("Error parsing tool arguments: %v", err)
				continue
			}

			result, err := subagentRegistry.ExecuteTool(tc.Function.Name, args, userID, db, typesenseClient, conversationID, messageID)
			if err != nil {
				log.Printf("Error executing tool %s: %v", tc.Function.Name, err)
				result = map[string]interface{}{
					"error": err.Error(),
				}
			}

			// Convert result to JSON string for tool response
			resultJSON, _ := json.Marshal(result)

			// Add tool response message
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	return "Subagent completed after maximum iterations", nil
}

// Database helper functions

func executeTextSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
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
					"id":           int(doc["card_pk"].(float64)),
					"title":        doc["title"].(string),
					"body_preview": doc["preview"].(string),
					"card_id":      doc["card_id"].(string),
					"created_at":   time.Unix(int64(doc["created_at"].(float64)), 0),
					"updated_at":   time.Unix(int64(doc["updated_at"].(float64)), 0),
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
		SELECT id, title, LEFT(body, 200) as body_preview, created_at, updated_at, card_id
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
		var title, bodyPreview, cardID sql.NullString
		var id int
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &bodyPreview, &createdAt, &updatedAt, &cardID)
		if err != nil {
			continue
		}

		card["id"] = id
		card["title"] = title.String
		card["body_preview"] = bodyPreview.String + "..."
		card["card_id"] = cardID.String
		card["created_at"] = createdAt
		card["updated_at"] = updatedAt

		cards = append(cards, card)
	}

	return cards, nil
}

func executeSemanticSearch(db *sql.DB, userID int, query string, limit int, typesenseClient *typesense.Client) ([]map[string]interface{}, error) {
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
		return executeTextSearch(db, userID, query, limit, typesenseClient)
	}

	var cards []map[string]interface{}
	for _, hit := range *typesenseResults.Hits {
		if hit.Document != nil {
			doc := *hit.Document
			if doc["type"].(string) == "card" {
				card := map[string]interface{}{
					"id":           int(doc["card_pk"].(float64)),
					"title":        doc["title"].(string),
					"body_preview": doc["preview"].(string),
					"card_id":      doc["card_id"].(string),
					"created_at":   time.Unix(int64(doc["created_at"].(float64)), 0),
					"updated_at":   time.Unix(int64(doc["updated_at"].(float64)), 0),
				}
				cards = append(cards, card)
			}
		}
	}

	return cards, nil
}

func getFullCard(db *sql.DB, userID int, cardID int) (map[string]interface{}, error) {
	query := `
		SELECT id, title, body, card_id, created_at, updated_at
		FROM cards
		WHERE id = $1 AND user_id = $2
	`

	var card map[string]interface{} = make(map[string]interface{})
	var title, body, cardIDStr sql.NullString
	var id int
	var createdAt, updatedAt time.Time

	err := db.QueryRow(query, cardID, userID).Scan(&id, &title, &body, &cardIDStr, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	card["id"] = id
	card["title"] = title.String
	card["body"] = body.String
	card["card_id"] = cardIDStr.String
	card["created_at"] = createdAt
	card["updated_at"] = updatedAt

	// Get tags
	tags, err := getCardTags(db, id)
	if err == nil {
		card["tags"] = tags
	}

	return card, nil
}

func getChildCards(db *sql.DB, userID int, cardID int) ([]map[string]interface{}, error) {
	// Get the parent card's card_id first
	var parentCardID string
	err := db.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", cardID, userID).Scan(&parentCardID)
	if err != nil {
		return nil, err
	}

	// Find child cards based on card_id hierarchy
	query := `
		SELECT id, title, LEFT(body, 200) as body_preview, card_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND card_id LIKE $2 AND card_id != $3
		ORDER BY card_id
		LIMIT 50
	`

	pattern := parentCardID + "%"
	rows, err := db.Query(query, userID, pattern, parentCardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []map[string]interface{}
	for rows.Next() {
		var card map[string]interface{} = make(map[string]interface{})
		var title, bodyPreview, cardIDStr sql.NullString
		var id int
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &bodyPreview, &cardIDStr, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		card["id"] = id
		card["title"] = title.String
		card["body_preview"] = bodyPreview.String + "..."
		card["card_id"] = cardIDStr.String
		card["created_at"] = createdAt
		card["updated_at"] = updatedAt

		cards = append(cards, card)
	}

	return cards, nil
}

func getParentCard(db *sql.DB, userID int, cardID int) ([]map[string]interface{}, error) {
	// Get the card's card_id first
	var currentCardID string
	err := db.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", cardID, userID).Scan(&currentCardID)
	if err != nil {
		return nil, err
	}

	// Find parent by removing last segment
	parts := strings.Split(currentCardID, "/")
	if len(parts) <= 1 {
		return []map[string]interface{}{}, nil // No parent (root card)
	}

	parentCardID := strings.Join(parts[:len(parts)-1], "/")

	query := `
		SELECT id, title, LEFT(body, 200) as body_preview, card_id, created_at, updated_at
		FROM cards
		WHERE user_id = $1 AND card_id = $2
	`

	var card map[string]interface{} = make(map[string]interface{})
	var title, bodyPreview, cardIDStr sql.NullString
	var id int
	var createdAt, updatedAt time.Time

	err = db.QueryRow(query, userID, parentCardID).Scan(&id, &title, &bodyPreview, &cardIDStr, &createdAt, &updatedAt)
	if err != nil {
		return []map[string]interface{}{}, nil // Parent not found
	}

	card["id"] = id
	card["title"] = title.String
	card["body_preview"] = bodyPreview.String + "..."
	card["card_id"] = cardIDStr.String
	card["created_at"] = createdAt
	card["updated_at"] = updatedAt

	return []map[string]interface{}{card}, nil
}

func filterCards(db *sql.DB, userID int, filters map[string]interface{}, limit int) ([]map[string]interface{}, error) {
	query := "SELECT id, title, LEFT(body, 200) as body_preview, card_id, created_at, updated_at FROM cards WHERE user_id = $1"
	args := []interface{}{userID}
	argIndex := 2

	// Add filters
	if starredOnly, ok := filters["starred_only"].(bool); ok && starredOnly {
		query += fmt.Sprintf(" AND id IN (SELECT card_id FROM starred_cards WHERE user_id = $%d)", argIndex)
		args = append(args, userID)
		argIndex++
	}

	if createdAfter, ok := filters["created_after"].(string); ok {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, createdAfter)
		argIndex++
	}

	if createdBefore, ok := filters["created_before"].(string); ok {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, createdBefore)
		argIndex++
	}

	query += " ORDER BY updated_at DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []map[string]interface{}
	for rows.Next() {
		var card map[string]interface{} = make(map[string]interface{})
		var title, bodyPreview, cardIDStr sql.NullString
		var id int
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &bodyPreview, &cardIDStr, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		card["id"] = id
		card["title"] = title.String
		card["body_preview"] = bodyPreview.String + "..."
		card["card_id"] = cardIDStr.String
		card["created_at"] = createdAt
		card["updated_at"] = updatedAt

		cards = append(cards, card)
	}

	return cards, nil
}

func getCardTags(db *sql.DB, cardID int) ([]string, error) {
	query := `
		SELECT t.name
		FROM tags t
		JOIN card_tags ct ON t.id = ct.tag_id
		WHERE ct.card_id = $1
		ORDER BY t.name
	`

	rows, err := db.Query(query, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			continue
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

func logToolExecution(db *sql.DB, userID int, toolName string, args map[string]interface{}, result map[string]interface{}, executionTimeMs int, execErr error, conversationID, messageID *string) error {
	argsJSON, _ := json.Marshal(args)
	resultJSON, _ := json.Marshal(result)

	query := `
		INSERT INTO chat_tool_calls (id, user_id, conversation_id, message_id, tool_name, tool_arguments, tool_result, execution_time_ms, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW())
	`
	fmt.Printf("Tool call: %v", toolName)

	_, err := db.Exec(query, userID, conversationID, messageID, toolName, argsJSON, resultJSON, executionTimeMs)
	return err
}
