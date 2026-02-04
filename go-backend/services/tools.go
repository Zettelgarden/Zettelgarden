package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/typesense/typesense-go/typesense"
)

// Tool name constants
const (
	ToolSearchCards        = "search_cards"
	ToolGetCardByID        = "get_card_by_id"
	ToolBrowseCardHierarchy = "browse_card_hierarchy"
	ToolCreateCard         = "create_card"
	ToolUpdateCard         = "update_card"
	ToolGetCardAnalysis    = "get_card_analysis"
	ToolSearchFacts        = "search_facts"
	ToolGetCardFacts       = "get_card_facts"
	ToolGetEntityFacts     = "get_entity_facts"
	ToolGetFactCards       = "get_fact_cards"
	ToolGetTasks           = "get_tasks"
	ToolCreateTask         = "create_task"
	ToolUpdateTask         = "update_task"
	ToolGetTaskByID              = "get_task_by_id"
	ToolCompleteTask             = "complete_task"
	ToolDeleteTask               = "delete_task"
	ToolCompleteAndScheduleTask  = "complete_and_schedule_task"
	ToolGetEntityByName          = "get_entity_by_name"
	ToolSearchEntities           = "search_entities"
	ToolGetCardsByEntity         = "get_cards_by_entity"
	ToolGetEntityByID            = "get_entity_by_id"
	ToolMergeEntities            = "merge_entities"
	ToolUpdateEntity             = "update_entity"
	ToolDeleteEntity             = "delete_entity"
	ToolAddEntityToCard          = "add_entity_to_card"
	ToolRemoveEntityFromCard     = "remove_entity_from_card"
	ToolGetSimilarEntities       = "get_similar_entities"
	ToolGetUserMemory    = "get_user_memory"
	ToolGetTemplate      = "get_template"
	ToolListTemplates    = "list_templates"
	ToolGetNextChildID   = "get_next_child_id"
)

// ToolContext contains all the context needed for tool execution
type ToolContext struct {
	UserID          int
	DB              *sql.DB
	TypesenseClient *typesense.Client
	ConversationID  *string
	MessageID       *string
	Model           string
}

// Tool represents a tool that can be called by the LLM
type Tool struct {
	Definition openai.Tool
	Handler    func(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error)
}

// ToolRegistry holds all available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// Parameter extraction helpers

// getIntParam extracts an integer parameter from args
func getIntParam(args map[string]interface{}, key string) (int, error) {
	val, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("%s parameter is required", key)
	}
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("invalid %s format: %v", key, err)
		}
		return int(i), nil
	default:
		return 0, fmt.Errorf("%s must be an integer", key)
	}
}

// getOptionalIntParam extracts an optional integer parameter from args
func getOptionalIntParam(args map[string]interface{}, key string) (int, bool, error) {
	val, ok := args[key]
	if !ok {
		return 0, false, nil
	}
	switch v := val.(type) {
	case float64:
		return int(v), true, nil
	case int:
		return v, true, nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false, fmt.Errorf("invalid %s format: %v", key, err)
		}
		return int(i), true, nil
	default:
		return 0, false, fmt.Errorf("%s must be an integer", key)
	}
}

// getStringParam extracts a string parameter from args
func getStringParam(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%s parameter is required", key)
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("%s must be a string", key)
}

// getOptionalStringParam extracts an optional string parameter from args
func getOptionalStringParam(args map[string]interface{}, key string) (string, bool) {
	val, ok := args[key]
	if !ok {
		return "", false
	}
	if str, ok := val.(string); ok {
		return str, true
	}
	return "", false
}

// getBoolParam extracts a boolean parameter from args
func getBoolParam(args map[string]interface{}, key string) (bool, error) {
	val, ok := args[key]
	if !ok {
		return false, fmt.Errorf("%s parameter is required", key)
	}
	if b, ok := val.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("%s must be a boolean", key)
}

// getOptionalBoolParam extracts an optional boolean parameter from args
func getOptionalBoolParam(args map[string]interface{}, key string) bool {
	val, ok := args[key]
	if !ok {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
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
	registry.registerCreateCard()
	registry.registerUpdateCard()
	registry.registerGetCardAnalysis()
	registry.registerSearchFacts()
	registry.registerGetCardFacts()
	registry.registerGetEntityFacts()
	registry.registerGetFactCards()
	registry.registerGetTasks()
	registry.registerCreateTask()
	registry.registerUpdateTask()
	registry.registerGetTaskByID()
	registry.registerCompleteTask()
	registry.registerDeleteTask()
	registry.registerCompleteAndScheduleTask()
	registry.registerGetEntityByName()
	registry.registerSearchEntities()
	registry.registerGetCardsByEntity()
	registry.registerGetEntityByID()
	registry.registerMergeEntities()
	registry.registerUpdateEntity()
	registry.registerDeleteEntity()
	registry.registerAddEntityToCard()
	registry.registerRemoveEntityFromCard()
	registry.registerGetSimilarEntities()
	registry.registerGetUserMemory()
	registry.registerGetTemplate()
	registry.registerListTemplates()
	registry.registerGetNextChildID()

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

// registerTool is a helper for registering tools with less boilerplate
func (tr *ToolRegistry) registerTool(name, description string, params map[string]interface{}, handler func(map[string]interface{}, *ToolContext) (map[string]interface{}, error)) {
	tr.tools[name] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        name,
				Description: description,
				Parameters:  params,
			},
		},
		Handler: handler,
	}
}

// ExecuteTool executes a tool by name with given arguments
func (tr *ToolRegistry) ExecuteTool(name string, args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	tool, exists := tr.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	start := time.Now()
	result, err := tool.Handler(args, ctx)
	executionTime := int(time.Since(start).Milliseconds())

	// Log tool execution for analytics (with timeout to prevent goroutine leaks)
	go func() {
		// Create a context with timeout for logging
		logCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logErr := logToolExecution(ctx.DB, ctx.UserID, name, args, result, executionTime, err, ctx.ConversationID, ctx.MessageID)
		if logErr != nil {
			if logCtx.Err() == context.DeadlineExceeded {
				log.Printf("Timeout logging tool execution for %s", name)
			} else {
				log.Printf("Error logging tool execution: %v", logErr)
			}
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
				Description: "Browse the hierarchical structure of cards. Get parent or child cards of a specific card, optionally traversing multiple levels deep.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"card_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the card to browse from",
						},
						"direction": map[string]interface{}{
							"type":        "string",
							"description": "Direction to browse: 'children' for child cards, 'parent' for parent cards",
							"enum":        []string{"children", "parent"},
						},
						"depth": map[string]interface{}{
							"type":        "integer",
							"description": "How many levels to traverse (default: 1 for immediate children/parent only). Use -1 for unlimited depth to get all descendants or ancestors.",
							"default":     1,
							"minimum":     -1,
						},
					},
					"required": []string{"card_id", "direction"},
				},
			},
		},
		Handler: handleBrowseCardHierarchy,
	}
}

func (tr *ToolRegistry) registerCreateCard() {
	tr.tools["create_card"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "create_card",
				Description: "Create a new card with title, body, and optional link. The card_id will be set to empty string for user categorization.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Title for the new card (required)",
						},
						"body": map[string]interface{}{
							"type":        "string",
							"description": "Body content for the new card (required)",
						},
						"link": map[string]interface{}{
							"type":        "string",
							"description": "Optional link for the card (can be empty string)",
						},
					},
					"required": []string{"title", "body"},
				},
			},
		},
		Handler: handleCreateCard,
	}
}

func (tr *ToolRegistry) registerGetCardAnalysis() {
	tr.tools["get_card_analysis"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_card_analysis",
				Description: "Retrieve the analysis/summary for a specific card by its primary key ID. Returns structured analysis with sections, theses, and arguments.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "The primary key ID of the card to get analysis for. This is different from the card_id, which is meant to be human readable",
						},
						"card_id": map[string]interface{}{
							"type":        "integer",
							"description": "The human readable identifier of the card to get analysis for. This is different from the card_pk, which is just an int",
						},
					},
					"required": []string{"card_pk"},
				},
			},
		},
		Handler: handleGetCardAnalysis,
	}
}

func (tr *ToolRegistry) registerUpdateCard() {
	tr.tools["update_card"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "update_card",
				Description: "Update an existing card's title, body, or link. All fields except id and existing_card_id are optional - only provided fields will be updated.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "integer",
							"description": "The primary key ID of the card to update (required)",
						},
						"existing_card_id": map[string]interface{}{
							"type":        "string",
							"description": "The current card_id (user-readable identifier) for verification (required)",
						},
						"title": map[string]interface{}{
							"type":        "string",
							"description": "New title for the card (optional)",
						},
						"body": map[string]interface{}{
							"type":        "string",
							"description": "New body content for the card (optional)",
						},
						"link": map[string]interface{}{
							"type":        "string",
							"description": "New link for the card (optional)",
						},
					},
					"required": []string{"id", "existing_card_id"},
				},
			},
		},
		Handler: handleUpdateCard,
	}
}

func (tr *ToolRegistry) registerSearchFacts() {
	tr.tools["search_facts"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "search_facts",
				Description: "Search for facts in the user's knowledge base using text or semantic similarity. Returns relevant facts based on the search query.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query to find relevant facts",
						},
						"search_type": map[string]interface{}{
							"type":        "string",
							"description": "Type of search: 'text' for exact text matching, 'semantic' for meaning-based search",
							"enum":        []string{"text", "semantic"},
							"default":     "semantic",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of facts to return (default: 10, max: 50)",
							"default":     10,
							"minimum":     1,
							"maximum":     50,
						},
					},
					"required": []string{"query"},
				},
			},
		},
		Handler: handleSearchFacts,
	}
}

// Tool handlers

func handleSearchCards(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	query, err := getStringParam(args, "query")
	if err != nil {
		return nil, err
	}

	searchType, _ := getOptionalStringParam(args, "search_type")
	if searchType == "" {
		searchType = "semantic"
	}

	limit := 20
	if l, ok, lerr := getOptionalIntParam(args, "limit"); ok && lerr == nil {
		limit = l
	}

	// Execute search based on type
	var cards []map[string]interface{}

	if searchType == "text" {
		cards, err = ExecuteTextSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	} else {
		cards, err = ExecuteSemanticSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
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

func handleGetCardByID(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardID, err := getIntParam(args, "card_id")
	if err != nil {
		return nil, err
	}

	card, err := GetFullCard(ctx.DB, ctx.UserID, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	return StructToMap(card), nil
}

func handleCreateCard(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	title, err := getStringParam(args, "title")
	if err != nil {
		return nil, err
	}

	body, err := getStringParam(args, "body")
	if err != nil {
		return nil, err
	}

	// Link is optional, default to empty string
	link, _ := getOptionalStringParam(args, "link")

	// Create card parameters with empty card_id for user categorization
	params := models.EditCardParams{
		Title:  title,
		Body:   body,
		Link:   link,
		CardID: "", // Empty string as requested
	}

	// Create the card
	newCard, err := CreateCard(ctx.DB, ctx.UserID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %v", err)
	}

	result := StructToMap(newCard)
	// Add metadata about the operation for frontend refresh detection
	result["operation"] = "card_created"
	result["card_pk"] = newCard.ID
	result["card_id"] = newCard.CardID

	return result, nil
}

func handleGetCardAnalysis(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	// Get the card analysis using the services function
	analysis, err := GetCardAnalysis(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get card analysis: %v", err)
	}

	// Convert analysis to map for tool response
	return map[string]interface{}{
		"card_pk":  cardPK,
		"analysis": analysis,
	}, nil
}

func handleUpdateCard(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "id")
	if err != nil {
		return nil, err
	}

	existingCardID, err := getStringParam(args, "existing_card_id")
	if err != nil {
		return nil, err
	}

	// Get the current card
	currentCard, err := GetFullCard(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	// Safety check: verify the existing_card_id matches what's on disk
	if currentCard.CardID != existingCardID {
		return nil, fmt.Errorf("card_id mismatch: expected %s but found %s", existingCardID, currentCard.CardID)
	}

	// Build update parameters, using current values as defaults
	params := models.EditCardParams{
		Title:  currentCard.Title,
		Body:   currentCard.Body,
		Link:   currentCard.Link,
		CardID: currentCard.CardID,
	}

	// Update only provided fields
	if title, ok := getOptionalStringParam(args, "title"); ok {
		params.Title = title
	}
	if body, ok := getOptionalStringParam(args, "body"); ok {
		params.Body = body
	}
	if link, ok := getOptionalStringParam(args, "link"); ok {
		params.Link = link
	}

	// Update the card
	updatedCard, err := UpdateCard(ctx.DB, ctx.UserID, cardPK, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update card: %v", err)
	}

	result := StructToMap(updatedCard)
	// Add metadata about the operation for frontend refresh detection
	result["operation"] = "card_updated"
	result["card_pk"] = cardPK
	result["card_id"] = updatedCard.CardID

	return result, nil
}

func handleBrowseCardHierarchy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_id")
	if err != nil {
		return nil, err
	}

	direction, err := getStringParam(args, "direction")
	if err != nil {
		return nil, err
	}

	// Get optional depth parameter, default to 1 for immediate children/parent only
	depth := 1
	if d, ok, derr := getOptionalIntParam(args, "depth"); ok && derr == nil {
		depth = d
	}

	var cards []models.PartialCard

	if direction == "children" {
		cards, err = GetChildCardsWithDepth(ctx.DB, ctx.UserID, cardPK, depth)
	} else if direction == "parent" {
		cards, err = GetParentCardsWithDepth(ctx.DB, ctx.UserID, cardPK, depth)
	} else {
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to browse hierarchy: %v", err)
	}

	var results []map[string]interface{}
	for _, card := range cards {
		results = append(results, StructToMap(card))
	}

	return map[string]interface{}{
		"cards":     results,
		"direction": direction,
		"depth":     depth,
		"total":     len(cards),
	}, nil
}

func (tr *ToolRegistry) registerGetCardFacts() {
	tr.tools["get_card_facts"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_card_facts",
				Description: "Retrieve all facts associated with a specific card. Facts are auto-generated from card content.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "The primary key ID of the card to get facts for",
						},
					},
					"required": []string{"card_pk"},
				},
			},
		},
		Handler: handleGetCardFacts,
	}
}

func (tr *ToolRegistry) registerGetEntityFacts() {
	tr.tools["get_entity_facts"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_entity_facts",
				Description: "Retrieve all facts linked to a specific entity. Useful for understanding what information exists about a particular entity.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity to get facts for",
						},
					},
					"required": []string{"entity_id"},
				},
			},
		},
		Handler: handleGetEntityFacts,
	}
}

func (tr *ToolRegistry) registerGetFactCards() {
	tr.tools["get_fact_cards"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_fact_cards",
				Description: "Retrieve all cards that are linked to a specific fact. Shows where a fact appears across the knowledge base.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"fact_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the fact to get linked cards for",
						},
					},
					"required": []string{"fact_id"},
				},
			},
		},
		Handler: handleGetFactCards,
	}
}

func handleSearchFacts(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	query, err := getStringParam(args, "query")
	if err != nil {
		return nil, err
	}

	searchType, _ := getOptionalStringParam(args, "search_type")
	if searchType == "" {
		searchType = "semantic"
	}

	limit := 10
	if l, ok, lerr := getOptionalIntParam(args, "limit"); ok && lerr == nil {
		limit = l
	}

	var facts []map[string]interface{}

	if searchType == "text" {
		facts, err = ExecuteFactTextSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	} else {
		facts, err = ExecuteFactSemanticSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	return map[string]interface{}{
		"facts":       facts,
		"query":       query,
		"search_type": searchType,
		"total":       len(facts),
	}, nil
}

func handleGetCardFacts(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	facts, lerr := GetCardFacts(ctx.DB, ctx.UserID, cardPK)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get card facts: %v", lerr)
	}

	var results []map[string]interface{}
	for _, fact := range facts {
		results = append(results, StructToMap(fact))
	}

	return map[string]interface{}{
		"facts": results,
		"total": len(facts),
	}, nil
}

func handleGetEntityFacts(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	facts, lerr := GetEntityFacts(ctx.DB, ctx.UserID, entityID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get entity facts: %v", lerr)
	}

	var results []map[string]interface{}
	for _, fact := range facts {
		results = append(results, StructToMap(fact))
	}

	return map[string]interface{}{
		"facts": results,
		"total": len(facts),
	}, nil
}

func handleGetFactCards(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	factID, err := getIntParam(args, "fact_id")
	if err != nil {
		return nil, err
	}

	cards, lerr := GetFactCards(ctx.DB, ctx.UserID, factID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get fact cards: %v", lerr)
	}

	var results []map[string]interface{}
	for _, card := range cards {
		results = append(results, StructToMap(card))
	}

	return map[string]interface{}{
		"cards": results,
		"total": len(cards),
	}, nil
}

func (tr *ToolRegistry) registerGetTasks() {
	tr.tools["get_tasks"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_tasks",
				Description: "Retrieve a list of tasks for the user. Can optionally filter to include completed tasks or get tasks for a specific card.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"include_completed": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to include completed tasks in the results (default: false)",
							"default":     false,
						},
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "Optional card primary key to filter tasks by card (returns only tasks linked to this card)",
						},
					},
				},
			},
		},
		Handler: handleGetTasks,
	}
}

func (tr *ToolRegistry) registerCreateTask() {
	tr.tools["create_task"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "create_task",
				Description: "Create a new task with a title and optional scheduling, priority, and card linkage.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Title of the task (required)",
						},
						"scheduled_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional scheduled date in ISO 8601 format (e.g., 2024-01-15T10:30:00Z)",
						},
						"due_date": map[string]interface{}{
							"type":        "string",
							"description": "Optional due date in ISO 8601 format (e.g., 2024-01-15T10:30:00Z)",
						},
						"priority": map[string]interface{}{
							"type":        "string",
							"description": "Optional priority level (e.g., 'high', 'medium', 'low')",
						},
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "Optional card primary key to link the task to a specific card",
						},
					},
					"required": []string{"title"},
				},
			},
		},
		Handler: handleCreateTask,
	}
}

func (tr *ToolRegistry) registerUpdateTask() {
	tr.tools["update_task"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "update_task",
				Description: "Update an existing task's properties. Only provided fields will be updated.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"task_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the task to update (required)",
						},
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Updated title for the task (optional)",
						},
						"scheduled_date": map[string]interface{}{
							"type":        "string",
							"description": "Updated scheduled date in ISO 8601 format (optional)",
						},
						"due_date": map[string]interface{}{
							"type":        "string",
							"description": "Updated due date in ISO 8601 format (optional)",
						},
						"priority": map[string]interface{}{
							"type":        "string",
							"description": "Updated priority level (optional)",
						},
						"is_complete": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the task is complete (optional)",
						},
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "Updated card primary key to link the task to (optional)",
						},
					},
					"required": []string{"task_id"},
				},
			},
		},
		Handler: handleUpdateTask,
	}
}

func (tr *ToolRegistry) registerGetTaskByID() {
	tr.tools["get_task_by_id"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_task_by_id",
				Description: "Retrieve a specific task by its ID.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"task_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the task to retrieve",
						},
					},
					"required": []string{"task_id"},
				},
			},
		},
		Handler: handleGetTaskByID,
	}
}

func handleGetTasks(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	includeCompleted := false // we do not want completed tasks, there are too many

	var tasks []models.Task
	var err error

	if cardPKFloat, ok := args["card_pk"].(float64); ok {
		cardPK := int(cardPKFloat)
		tasks, err = GetTasksByCard(ctx.DB, ctx.UserID, cardPK)
	} else {
		tasks, err = GetTasks(ctx.DB, ctx.UserID, includeCompleted, "UTC")
	}
	log.Printf("tasks")
	for _, task := range tasks {
		log.Printf("- %v", task)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %v", err)
	}

	var results []map[string]interface{}
	for _, task := range tasks {
		results = append(results, StructToMap(task))
	}

	return map[string]interface{}{
		"tasks": results,
		"total": len(tasks),
	}, nil
}

func handleCreateTask(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	title, err := getStringParam(args, "title")
	if err != nil {
		return nil, err
	}

	task := models.Task{
		UserID:     ctx.UserID,
		Title:      title,
		IsComplete: false,
	}

	if scheduledDateStr, ok := getOptionalStringParam(args, "scheduled_date"); ok {
		scheduledDate, perr := time.Parse(time.RFC3339, scheduledDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %v", perr)
		}
		task.ScheduledDate = &scheduledDate
	}

	if dueDateStr, ok := getOptionalStringParam(args, "due_date"); ok {
		dueDate, perr := time.Parse(time.RFC3339, dueDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid due_date format: %v", perr)
		}
		task.DueDate = &dueDate
	} else {
		now := time.Now()
		task.DueDate = &now
	}

	if priority, ok := getOptionalStringParam(args, "priority"); ok {
		task.Priority = &priority
	}

	if cardPK, ok, _ := getOptionalIntParam(args, "card_pk"); ok {
		task.CardPK = cardPK
	}

	taskID, err := CreateTask(ctx.DB, task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %v", err)
	}

	newTask, err := GetTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created task: %v", err)
	}

	return StructToMap(newTask), nil
}

func handleUpdateTask(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	currentTask, lerr := GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	if title, ok := getOptionalStringParam(args, "title"); ok {
		currentTask.Title = title
	}

	if scheduledDateStr, ok := getOptionalStringParam(args, "scheduled_date"); ok {
		scheduledDate, perr := time.Parse(time.RFC3339, scheduledDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %v", perr)
		}
		currentTask.ScheduledDate = &scheduledDate
	}

	if dueDateStr, ok := getOptionalStringParam(args, "due_date"); ok {
		dueDate, perr := time.Parse(time.RFC3339, dueDateStr)
		if perr != nil {
			return nil, fmt.Errorf("invalid due_date format: %v", perr)
		}
		currentTask.DueDate = &dueDate
	}

	if priority, ok := getOptionalStringParam(args, "priority"); ok {
		currentTask.Priority = &priority
	}

	if isComplete, ok := args["is_complete"].(bool); ok {
		currentTask.IsComplete = isComplete
	}

	if cardPK, ok, _ := getOptionalIntParam(args, "card_pk"); ok {
		currentTask.CardPK = cardPK
	}

	_, uerr := UpdateTask(ctx.DB, ctx.UserID, taskID, currentTask)
	if uerr != nil {
		return nil, fmt.Errorf("failed to update task: %v", uerr)
	}

	updatedTask, uerr := GetTask(ctx.DB, ctx.UserID, taskID)
	if uerr != nil {
		return nil, fmt.Errorf("failed to retrieve updated task: %v", uerr)
	}

	return StructToMap(updatedTask), nil
}

func handleGetTaskByID(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	task, lerr := GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	return StructToMap(task), nil
}

func (tr *ToolRegistry) registerCompleteTask() {
	tr.tools["complete_task"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "complete_task",
				Description: "Mark a task as complete. This is a convenience wrapper for updating a task's completion status.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"task_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the task to mark as complete",
						},
					},
					"required": []string{"task_id"},
				},
			},
		},
		Handler: handleCompleteTask,
	}
}

func handleCompleteTask(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	currentTask, lerr := GetTask(ctx.DB, ctx.UserID, taskID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get task: %v", lerr)
	}

	currentTask.IsComplete = true

	_, uerr := UpdateTask(ctx.DB, ctx.UserID, taskID, currentTask)
	if uerr != nil {
		return nil, fmt.Errorf("failed to complete task: %v", uerr)
	}

	updatedTask, uerr := GetTask(ctx.DB, ctx.UserID, taskID)
	if uerr != nil {
		return nil, fmt.Errorf("failed to retrieve updated task: %v", uerr)
	}

	return StructToMap(updatedTask), nil
}

func (tr *ToolRegistry) registerDeleteTask() {
	tr.tools["delete_task"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "delete_task",
				Description: "Delete a task by its ID. This action cannot be undone.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"task_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the task to delete",
						},
					},
					"required": []string{"task_id"},
				},
			},
		},
		Handler: handleDeleteTask,
	}
}

func handleDeleteTask(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	err = DeleteTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete task: %v", err)
	}

	return map[string]interface{}{
		"status":  "deleted",
		"task_id": taskID,
	}, nil
}

func (tr *ToolRegistry) registerCompleteAndScheduleTask() {
	tr.tools["complete_and_schedule_task"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "complete_and_schedule_task",
				Description: "Complete a recurring task and create a new one scheduled for a specified number of days later. Useful for managing recurring tasks.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"task_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the task to complete",
						},
						"days": map[string]interface{}{
							"type":        "integer",
							"description": "Number of days to schedule the new task in the future (must be greater than 0)",
						},
					},
					"required": []string{"task_id", "days"},
				},
			},
		},
		Handler: handleCompleteAndScheduleTask,
	}
}

func handleCompleteAndScheduleTask(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskID, err := getIntParam(args, "task_id")
	if err != nil {
		return nil, err
	}

	days, err := getIntParam(args, "days")
	if err != nil {
		return nil, err
	}

	if days <= 0 {
		return nil, fmt.Errorf("days must be greater than 0")
	}

	// Get the complete and default status names
	completeStatus, lerr := GetCompleteTaskStatus(ctx.DB, ctx.UserID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get complete status: %v", lerr)
	}

	defaultStatus, lerr := GetDefaultTaskStatus(ctx.DB, ctx.UserID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get default status: %v", lerr)
	}

	newTaskID, lerr := CompleteAndScheduleTask(ctx.DB, ctx.UserID, taskID, days, completeStatus.Name, defaultStatus.Name)
	if lerr != nil {
		return nil, fmt.Errorf("failed to complete and schedule task: %v", lerr)
	}

	return map[string]interface{}{
		"status":        "completed_and_scheduled",
		"task_id":       taskID,
		"new_task_id":   newTaskID,
		"scheduled_in":  fmt.Sprintf("%d days", days),
	}, nil
}

func (tr *ToolRegistry) registerGetEntityByName() {
	tr.tools["get_entity_by_name"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_entity_by_name",
				Description: "Retrieve a specific entity by its name. Returns the full entity information including linked card if available.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_name": map[string]interface{}{
							"type":        "string",
							"description": "The name of the entity to retrieve",
						},
					},
					"required": []string{"entity_name"},
				},
			},
		},
		Handler: handleGetEntityByName,
	}
}

func handleGetEntityByName(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityName, err := getStringParam(args, "entity_name")
	if err != nil {
		return nil, err
	}

	entity, lerr := GetEntityByName(ctx.DB, ctx.UserID, entityName)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get entity: %v", lerr)
	}

	return StructToMap(entity), nil
}

func (tr *ToolRegistry) registerSearchEntities() {
	tr.tools["search_entities"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "search_entities",
				Description: "Search for entities in the user's knowledge base using text or semantic similarity. Returns relevant entities based on the search query.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The search query to find relevant entities",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of entities to return (default: 10, max: 50)",
							"default":     10,
							"minimum":     1,
							"maximum":     50,
						},
					},
					"required": []string{"query"},
				},
			},
		},
		Handler: handleSearchEntities,
	}
}

func handleSearchEntities(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	query, err := getStringParam(args, "query")
	if err != nil {
		return nil, err
	}

	limit := 10
	if l, ok, lerr := getOptionalIntParam(args, "limit"); ok && lerr == nil {
		limit = l
	}

	entities, lerr := SearchEntities(ctx.DB, ctx.TypesenseClient, ctx.UserID, query, limit)
	if lerr != nil {
		return nil, fmt.Errorf("search failed: %v", lerr)
	}

	return map[string]interface{}{
		"entities": entities,
		"query":    query,
		"total":    len(entities),
	}, nil
}

func (tr *ToolRegistry) registerGetCardsByEntity() {
	tr.tools["get_cards_by_entity"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_cards_by_entity",
				Description: "Retrieve all cards that are linked to a specific entity. This is the primary search method for finding content related to entities - use this before other search methods.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity to get linked cards for",
						},
					},
					"required": []string{"entity_id"},
				},
			},
		},
		Handler: handleGetCardsByEntity,
	}
}

func handleGetCardsByEntity(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	cards, lerr := GetCardsByEntity(ctx.DB, ctx.UserID, entityID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get cards by entity: %v", lerr)
	}

	var results []map[string]interface{}
	for _, card := range cards {
		results = append(results, StructToMap(card))
	}

	return map[string]interface{}{
		"cards":     results,
		"entity_id": entityID,
		"total":     len(cards),
	}, nil
}

// Database helper functions

func logToolExecution(db *sql.DB, userID int, toolName string, args map[string]interface{}, result map[string]interface{}, executionTimeMs int, execErr error, conversationID, messageID *string) error {
	argsJSON, _ := json.Marshal(args)
	resultJSON, _ := json.Marshal(result)

	query := `
		INSERT INTO chat_tool_calls (id, user_id, conversation_id, message_id, tool_name, tool_arguments, tool_result, execution_time_ms, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW())
	`
	fmt.Printf("Tool call: %v\n", toolName)

	_, err := db.Exec(query, userID, conversationID, messageID, toolName, argsJSON, resultJSON, executionTimeMs)
	return err
}

func (tr *ToolRegistry) registerGetUserMemory() {
	tr.tools["get_user_memory"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_user_memory",
				Description: "Retrieves your memory and observations about the user. This contains important context about the user's preferences, interests, work style, and past interactions. Use this to personalize responses and maintain continuity across conversations.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		Handler: func(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
			memory, err := GetUserMemory(ctx.DB, ctx.UserID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve user memory: %w", err)
			}

			if memory == "" {
				return map[string]interface{}{
					"memory": "",
					"note":   "No memory has been recorded yet for this user.",
				}, nil
			}

			return map[string]interface{}{
				"memory": memory,
			}, nil
		},
	}
}

func (tr *ToolRegistry) registerGetTemplate() {
	tr.tools["get_template"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_template",
				Description: "Get a specific template by its numeric ID. Returns the full template details including name, title, and body templates.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"template_id": map[string]interface{}{
							"type":        "integer",
							"description": "The numeric ID of the template to retrieve",
						},
					},
					"required": []string{"template_id"},
				},
			},
		},
		Handler: handleGetTemplate,
	}
}

func handleGetTemplate(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	templateID, err := getIntParam(args, "template_id")
	if err != nil {
		return nil, err
	}

	template, err := GetTemplate(ctx.DB, ctx.UserID, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %v", err)
	}

	return StructToMap(template), nil
}

// GetTemplate retrieves a template by ID for a specific user
func GetTemplate(db *sql.DB, userID, templateID int) (models.CardTemplate, error) {
	var template models.CardTemplate

	query := `
		SELECT id, user_id, name, title, body, created_at, updated_at
		FROM card_templates
		WHERE id = $1 AND user_id = $2
	`

	err := db.QueryRow(query, templateID, userID).Scan(
		&template.ID,
		&template.UserID,
		&template.Name,
		&template.Title,
		&template.Body,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err != nil {
		return models.CardTemplate{}, fmt.Errorf("template not found")
	}

	return template, nil
}

func (tr *ToolRegistry) registerListTemplates() {
	tr.tools["list_templates"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "list_templates",
				Description: "Get all templates for the current user. Templates are reusable card structures with variable substitution.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
					"required":   []string{},
				},
			},
		},
		Handler: handleListTemplates,
	}
}

func handleListTemplates(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	templates, err := GetTemplates(ctx.DB, ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %v", err)
	}

	var results []map[string]interface{}
	for _, template := range templates {
		results = append(results, StructToMap(template))
	}

	return map[string]interface{}{
		"templates": results,
		"total":     len(templates),
	}, nil
}

// GetTemplates retrieves all templates for a specific user
func GetTemplates(db *sql.DB, userID int) ([]models.CardTemplate, error) {
	query := `
		SELECT id, user_id, name, title, body, created_at, updated_at
		FROM card_templates
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.CardTemplate
	for rows.Next() {
		var template models.CardTemplate
		if err := rows.Scan(
			&template.ID,
			&template.UserID,
			&template.Name,
			&template.Title,
			&template.Body,
			&template.CreatedAt,
			&template.UpdatedAt,
		); err != nil {
			return nil, err
		}
		templates = append(templates, template)
	}

	return templates, nil
}

func (tr *ToolRegistry) registerGetNextChildID() {
	tr.tools["get_next_child_id"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_next_child_id",
				Description: "Get the next available child card ID for a parent card (e.g., '1a2.3'). This is useful for creating structured card hierarchies.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "The primary key ID of the parent card",
						},
					},
					"required": []string{"card_pk"},
				},
			},
		},
		Handler: handleGetNextChildID,
	}
}

func handleGetNextChildID(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	nextID, err := GetNextChildCardID(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get next child ID: %v", err)
	}

	if nextID == "" {
		return map[string]interface{}{
			"error":   true,
			"message": "Parent card not found or error occurred",
			"new_id":  "",
		}, nil
	}

	return map[string]interface{}{
		"error":   false,
		"message": "",
		"new_id":  nextID,
	}, nil
}

// GetNextChildCardID returns the next available child card ID for a parent card
func GetNextChildCardID(db *sql.DB, userID int, parentID int) (string, error) {
	// 1. Get parent card's card_id (human readable ID)
	var parentCardID string
	err := db.QueryRow("SELECT card_id FROM cards WHERE id = $1 AND user_id = $2", parentID, userID).Scan(&parentCardID)
	if err != nil {
		log.Printf("Error finding parent card ID for parentID %d: %v", parentID, err)
		return "", fmt.Errorf("parent card not found")
	}

	// 2. Get all existing children using service
	children, err := GetChildCards(db, userID, parentID)
	if err != nil {
		log.Printf("Error getting child cards for parentID %d: %v", parentID, err)
		return parentCardID + ".1", nil // Default to .1 if there's an error
	}

	// 3. Extract numeric suffixes from children's card_ids
	childNumbers := make([]int, 0)
	parentIDLength := len(parentCardID)

	for _, child := range children {
		childID := child.CardID

		// Verify this is actually a direct child by checking it starts with parent ID
		if !strings.HasPrefix(childID, parentCardID) || len(childID) <= parentIDLength {
			continue
		}

		// Get the part after the parent ID
		suffix := childID[parentIDLength:]

		// Extract the first number after any separator using regex
		re := regexp.MustCompile(`^[.\\/-]+(\d+)`)
		match := re.FindStringSubmatch(suffix)
		if len(match) == 2 {
			num, err := strconv.Atoi(match[1])
			if err == nil {
				childNumbers = append(childNumbers, num)
			}
		}
	}

	// 4. Find the highest number and increment
	if len(childNumbers) == 0 {
		return parentCardID + ".1", nil // No existing children, start with 1
	}

	maxNumber := 0
	for _, num := range childNumbers {
		if num > maxNumber {
			maxNumber = num
		}
	}

	nextNumber := maxNumber + 1
	return fmt.Sprintf("%s.%d", parentCardID, nextNumber), nil
}

// Entity tool registration and handlers

func (tr *ToolRegistry) registerGetEntityByID() {
	tr.tools["get_entity_by_id"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_entity_by_id",
				Description: "Retrieve a specific entity by its ID. Returns the full entity information including linked card if available.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity to retrieve",
						},
					},
					"required": []string{"entity_id"},
				},
			},
		},
		Handler: handleGetEntityByID,
	}
}

func handleGetEntityByID(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	entity, lerr := GetEntityByID(ctx.DB, ctx.UserID, entityID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get entity: %v", lerr)
	}

	return StructToMap(entity), nil
}

func (tr *ToolRegistry) registerMergeEntities() {
	tr.tools["merge_entities"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "merge_entities",
				Description: "Merge two entities into one. The first entity will absorb all relationships and data from the second entity, which will be deleted. Use this when you find duplicate entities that should be combined.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity1_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity that will survive (all data from entity2 will be merged into this one)",
						},
						"entity2_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity that will be deleted after merging its data into entity1",
						},
					},
					"required": []string{"entity1_id", "entity2_id"},
				},
			},
		},
		Handler: handleMergeEntities,
	}
}

func handleMergeEntities(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entity1ID, err := getIntParam(args, "entity1_id")
	if err != nil {
		return nil, err
	}

	entity2ID, err := getIntParam(args, "entity2_id")
	if err != nil {
		return nil, err
	}

	if entity1ID == entity2ID {
		return nil, fmt.Errorf("cannot merge an entity with itself")
	}

	lerr := MergeEntities(ctx.DB, ctx.UserID, entity1ID, entity2ID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to merge entities: %v", lerr)
	}

	return map[string]interface{}{
		"status":       "merged",
		"entity1_id":   entity1ID,
		"entity2_id":   entity2ID,
		"surviving_id": entity1ID,
		"message":      fmt.Sprintf("Successfully merged entity %d into entity %d", entity2ID, entity1ID),
	}, nil
}

func (tr *ToolRegistry) registerUpdateEntity() {
	tr.tools["update_entity"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "update_entity",
				Description: "Update an existing entity's name, description, type, or linked card. Only provided fields will be updated.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity to update (required)",
						},
						"name": map[string]interface{}{
							"type":        "string",
							"description": "New name for the entity (optional)",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "New description for the entity (optional)",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"description": "New type for the entity (optional, e.g., 'person', 'organization', 'concept')",
						},
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "Primary key of the card to link to this entity (optional, set to null to remove link)",
						},
					},
					"required": []string{"entity_id"},
				},
			},
		},
		Handler: handleUpdateEntity,
	}
}

func handleUpdateEntity(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	// Get current entity first to have default values
	entity, lerr := GetEntityByID(ctx.DB, ctx.UserID, entityID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get entity: %v", lerr)
	}

	// Build update params, using current values as defaults
	params := UpdateEntityParams{
		Name:        entity.Name,
		Description: entity.Description,
		Type:        entity.Type,
		CardPK:      entity.CardPK,
	}

	// Update only provided fields
	if name, ok := getOptionalStringParam(args, "name"); ok {
		params.Name = name
	}
	if description, ok := getOptionalStringParam(args, "description"); ok {
		params.Description = description
	}
	if entityType, ok := getOptionalStringParam(args, "type"); ok {
		params.Type = entityType
	}
	if cardPK, ok, _ := getOptionalIntParam(args, "card_pk"); ok {
		params.CardPK = &cardPK
	}

	// Handle explicit null for card_pk (to remove link)
	if cardPKVal, exists := args["card_pk"]; exists && cardPKVal == nil {
		params.CardPK = nil
	}

	lerr = UpdateEntity(ctx.DB, ctx.UserID, entityID, params)
	if lerr != nil {
		return nil, fmt.Errorf("failed to update entity: %v", lerr)
	}

	// Fetch updated entity
	updatedEntity, lerr := GetEntityByID(ctx.DB, ctx.UserID, entityID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get updated entity: %v", lerr)
	}

	return StructToMap(updatedEntity), nil
}

func (tr *ToolRegistry) registerDeleteEntity() {
	tr.tools["delete_entity"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "delete_entity",
				Description: "Delete an entity by its ID. This will also remove all card and fact relationships for this entity. This action cannot be undone.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity to delete",
						},
					},
					"required": []string{"entity_id"},
				},
			},
		},
		Handler: handleDeleteEntity,
	}
}

func handleDeleteEntity(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	lerr := DeleteEntity(ctx.DB, ctx.UserID, entityID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to delete entity: %v", lerr)
	}

	return map[string]interface{}{
		"status":    "deleted",
		"entity_id": entityID,
		"message":   "Entity deleted successfully",
	}, nil
}

func (tr *ToolRegistry) registerAddEntityToCard() {
	tr.tools["add_entity_to_card"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "add_entity_to_card",
				Description: "Link an entity to a card. This creates a relationship between the entity and the card, making the card appear in entity searches.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity to link",
						},
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "The primary key ID of the card to link to",
						},
					},
					"required": []string{"entity_id", "card_pk"},
				},
			},
		},
		Handler: handleAddEntityToCard,
	}
}

func handleAddEntityToCard(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	lerr := AddEntityToCard(ctx.DB, ctx.UserID, entityID, cardPK)
	if lerr != nil {
		return nil, fmt.Errorf("failed to add entity to card: %v", lerr)
	}

	return map[string]interface{}{
		"status":    "linked",
		"entity_id": entityID,
		"card_pk":   cardPK,
		"message":   "Entity successfully linked to card",
	}, nil
}

func (tr *ToolRegistry) registerRemoveEntityFromCard() {
	tr.tools["remove_entity_from_card"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "remove_entity_from_card",
				Description: "Remove the link between an entity and a card. This will not delete the entity or the card, only their relationship.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity",
						},
						"card_pk": map[string]interface{}{
							"type":        "integer",
							"description": "The primary key ID of the card",
						},
					},
					"required": []string{"entity_id", "card_pk"},
				},
			},
		},
		Handler: handleRemoveEntityFromCard,
	}
}

func handleRemoveEntityFromCard(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	lerr := RemoveEntityFromCard(ctx.DB, ctx.UserID, entityID, cardPK)
	if lerr != nil {
		return nil, fmt.Errorf("failed to remove entity from card: %v", lerr)
	}

	return map[string]interface{}{
		"status":    "unlinked",
		"entity_id": entityID,
		"card_pk":   cardPK,
		"message":   "Entity successfully unlinked from card",
	}, nil
}

func (tr *ToolRegistry) registerGetSimilarEntities() {
	tr.tools["get_similar_entities"] = Tool{
		Definition: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_similar_entities",
				Description: "Find entities that are similar to a given entity based on semantic similarity of their names and descriptions. Useful for discovering potentially duplicate entities.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_id": map[string]interface{}{
							"type":        "integer",
							"description": "The ID of the entity to find similar entities for",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of similar entities to return (default: 10)",
							"default":     10,
						},
					},
					"required": []string{"entity_id"},
				},
			},
		},
		Handler: handleGetSimilarEntities,
	}
}

func handleGetSimilarEntities(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	limit := 10
	if l, ok, lerr := getOptionalIntParam(args, "limit"); ok && lerr == nil {
		limit = l
	}

	entities, lerr := FindSimilarEntities(ctx.DB, ctx.TypesenseClient, ctx.UserID, entityID, limit)
	if lerr != nil {
		return nil, fmt.Errorf("failed to find similar entities: %v", lerr)
	}

	var results []map[string]interface{}
	for _, entity := range entities {
		result := StructToMap(entity)
		// Add similarity score if available
		if score, ok := entity["score"].(float64); ok {
			result["similarity_score"] = score
		}
		results = append(results, result)
	}

	return map[string]interface{}{
		"entities":      results,
		"entity_id":     entityID,
		"total":         len(results),
		"limit":         limit,
	}, nil
}
