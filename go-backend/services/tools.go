package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/models"
	"go-backend/prompts"
	"log"
	"reflect"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/typesense/typesense-go/typesense"
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

// NewToolRegistry creates a new tool registry with all available tools
func NewToolRegistry() *ToolRegistry {
	registry := &ToolRegistry{
		tools: make(map[string]Tool),
	}

	// Register all tools
	//	registry.registerSearchCards()
	registry.registerGetCardByID()
	//registry.registerBrowseCardHierarchy()
	//registry.registerFilterCardsByMetadata()
	registry.registerCreateCard()
	registry.registerUpdateCard()
	registry.registerTask()
	registry.registerGetTasks()
	registry.registerCreateTask()
	registry.registerUpdateTask()
	registry.registerGetTaskByID()
	registry.registerGetEntityByName()
	registry.registerSearchEntities()

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
func (tr *ToolRegistry) ExecuteTool(name string, args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	tool, exists := tr.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	start := time.Now()
	result, err := tool.Handler(args, ctx)
	executionTime := int(time.Since(start).Milliseconds())

	// Log tool execution for analytics
	go func() {
		logErr := logToolExecution(ctx.DB, ctx.UserID, name, args, result, executionTime, err, ctx.ConversationID, ctx.MessageID)
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

func handleSearchCards(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	searchType := "semantic"
	if st, ok := args["search_type"].(string); ok {
		searchType = st
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Execute search based on type
	var cards []map[string]interface{}
	var err error

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
	cardIDFloat, ok := args["card_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("card_id parameter is required")
	}
	cardID := int(cardIDFloat)

	card, err := GetFullCard(ctx.DB, ctx.UserID, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	return StructToMap(card), nil
}

func handleCreateCard(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	title, ok := args["title"].(string)
	if !ok {
		return nil, fmt.Errorf("title parameter is required")
	}

	body, ok := args["body"].(string)
	if !ok {
		return nil, fmt.Errorf("body parameter is required")
	}

	// Link is optional, default to empty string
	link := ""
	if l, ok := args["link"].(string); ok {
		link = l
	}

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

	return StructToMap(newCard), nil
}

func handleGetCardAnalysis(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardIDFloat, ok := args["card_pk"].(float64)
	if !ok {
		return nil, fmt.Errorf("card_pk parameter is required")
	}
	cardPK := int(cardIDFloat)

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
	cardIDFloat, ok := args["id"].(float64)
	if !ok {
		return nil, fmt.Errorf("id parameter is required")
	}
	cardPK := int(cardIDFloat)

	existingCardID, ok := args["existing_card_id"].(string)
	if !ok {
		return nil, fmt.Errorf("existing_card_id parameter is required")
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
	if title, ok := args["title"].(string); ok {
		params.Title = title
	}
	if body, ok := args["body"].(string); ok {
		params.Body = body
	}
	if link, ok := args["link"].(string); ok {
		params.Link = link
	}

	// Update the card
	updatedCard, err := UpdateCard(ctx.DB, ctx.UserID, cardPK, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update card: %v", err)
	}

	return StructToMap(updatedCard), nil
}

func handleBrowseCardHierarchy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardIDFloat, ok := args["card_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("card_id parameter is required")
	}
	cardPK := int(cardIDFloat)

	direction, ok := args["direction"].(string)
	if !ok {
		return nil, fmt.Errorf("direction parameter is required")
	}

	var cards []models.PartialCard
	var err error

	if direction == "children" {
		cards, err = GetChildCards(ctx.DB, ctx.UserID, cardPK)
	} else if direction == "parent" {
		cards, err = GetParentCard(ctx.DB, ctx.UserID, cardPK)
	} else {
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}
	var results []map[string]interface{}
	for _, card := range cards {
		results = append(results, StructToMap(card))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to browse hierarchy: %v", err)
	}

	return map[string]interface{}{
		"cards":     results,
		"direction": direction,
		"total":     len(cards),
	}, nil
}

func handleTask(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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
	result, err := executeSubagentTask(prompt, subagentType, ctx)
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
func executeSubagentTask(prompt, subagentType string, ctx *ToolContext) (string, error) {
	// Create LLM client for the subagent
	client := NewDefaultClient(ctx.DB, ctx.UserID)
	client.RequestType = "tools"
	client.Model = ctx.Model

	// Create a tool registry for the subagent (excluding the Task tool to prevent recursion)
	subagentRegistry := &ToolRegistry{
		tools: make(map[string]Tool),
	}
	subagentRegistry.registerSearchCards()
	subagentRegistry.registerGetCardByID()
	subagentRegistry.registerBrowseCardHierarchy()
	subagentRegistry.registerGetCardAnalysis()
	subagentRegistry.registerSearchFacts()
	subagentRegistry.registerGetCardFacts()
	subagentRegistry.registerGetEntityFacts()
	subagentRegistry.registerGetFactCards()
	subagentRegistry.registerGetTasks()
	subagentRegistry.registerGetTaskByID()
	subagentRegistry.registerGetEntityByName()
	subagentRegistry.registerSearchEntities()

	tools := subagentRegistry.GetToolDefinitions()

	// Load system prompt for the subagent
	systemPrompt, err := prompts.GetSubagentResearcherPrompt()
	if err != nil {
		log.Printf("Error loading subagent system prompt: %v, using fallback", err)
		// Fallback to a basic prompt if file loading fails
		systemPrompt = "You are a specialized research assistant with access to a user's knowledge base. Use the available tools to help answer questions and gather information."
	}

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
		resp, err := ExecuteLLMToolRequest(client, messages, tools)
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

			start := time.Now()
			log.Printf("subagent tool - %v", tc.Function.Name)
			result, err := subagentRegistry.ExecuteTool(tc.Function.Name, args, ctx)
			executionTime := int(time.Since(start).Milliseconds())

			if err != nil {
				log.Printf("Error executing tool %s: %v", tc.Function.Name, err)
				result = map[string]interface{}{
					"error": err.Error(),
				}
			}

			// Log subagent tool execution
			go func(toolName string, toolArgs, toolResult map[string]interface{}, execTime int, execErr error) {
				logErr := logToolExecution(ctx.DB, ctx.UserID, toolName, toolArgs, toolResult, execTime, execErr, ctx.ConversationID, ctx.MessageID)
				if logErr != nil {
					log.Printf("Error logging subagent tool execution: %v", logErr)
				}
			}(tc.Function.Name, args, result, executionTime, err)

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

	var facts []map[string]interface{}
	var err error

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
	cardPKFloat, ok := args["card_pk"].(float64)
	if !ok {
		return nil, fmt.Errorf("card_pk parameter is required")
	}
	cardPK := int(cardPKFloat)

	facts, err := GetCardFacts(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get card facts: %v", err)
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
	entityIDFloat, ok := args["entity_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("entity_id parameter is required")
	}
	entityID := int(entityIDFloat)

	facts, err := GetEntityFacts(ctx.DB, ctx.UserID, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity facts: %v", err)
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
	factIDFloat, ok := args["fact_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("fact_id parameter is required")
	}
	factID := int(factIDFloat)

	cards, err := GetFactCards(ctx.DB, ctx.UserID, factID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact cards: %v", err)
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
		tasks, err = GetTasks(ctx.DB, ctx.UserID, includeCompleted)
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
	title, ok := args["title"].(string)
	if !ok {
		return nil, fmt.Errorf("title parameter is required")
	}

	task := models.Task{
		UserID:     ctx.UserID,
		Title:      title,
		IsComplete: false,
	}

	if scheduledDateStr, ok := args["scheduled_date"].(string); ok {
		scheduledDate, err := time.Parse(time.RFC3339, scheduledDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %v", err)
		}
		task.ScheduledDate = &scheduledDate
	}

	if dueDateStr, ok := args["due_date"].(string); ok {
		dueDate, err := time.Parse(time.RFC3339, dueDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format: %v", err)
		}
		task.DueDate = &dueDate
	} else {
		now := time.Now()
		task.DueDate = &now
	}

	if priority, ok := args["priority"].(string); ok {
		task.Priority = &priority
	}

	if cardPKFloat, ok := args["card_pk"].(float64); ok {
		task.CardPK = int(cardPKFloat)
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
	taskIDFloat, ok := args["task_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("task_id parameter is required")
	}
	taskID := int(taskIDFloat)

	currentTask, err := GetTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %v", err)
	}

	if title, ok := args["title"].(string); ok {
		currentTask.Title = title
	}

	if scheduledDateStr, ok := args["scheduled_date"].(string); ok {
		scheduledDate, err := time.Parse(time.RFC3339, scheduledDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled_date format: %v", err)
		}
		currentTask.ScheduledDate = &scheduledDate
	}

	if dueDateStr, ok := args["due_date"].(string); ok {
		dueDate, err := time.Parse(time.RFC3339, dueDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format: %v", err)
		}
		currentTask.DueDate = &dueDate
	}

	if priority, ok := args["priority"].(string); ok {
		currentTask.Priority = &priority
	}

	if isComplete, ok := args["is_complete"].(bool); ok {
		currentTask.IsComplete = isComplete
	}

	if cardPKFloat, ok := args["card_pk"].(float64); ok {
		currentTask.CardPK = int(cardPKFloat)
	}

	err = UpdateTask(ctx.DB, ctx.UserID, taskID, currentTask)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %v", err)
	}

	updatedTask, err := GetTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated task: %v", err)
	}

	return StructToMap(updatedTask), nil
}

func handleGetTaskByID(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	taskIDFloat, ok := args["task_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("task_id parameter is required")
	}
	taskID := int(taskIDFloat)

	task, err := GetTask(ctx.DB, ctx.UserID, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %v", err)
	}

	return StructToMap(task), nil
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
	entityName, ok := args["entity_name"].(string)
	if !ok {
		return nil, fmt.Errorf("entity_name parameter is required")
	}

	entity, err := GetEntityByName(ctx.DB, ctx.UserID, entityName)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %v", err)
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
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	entities, err := SearchEntities(ctx.DB, ctx.TypesenseClient, ctx.UserID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	return map[string]interface{}{
		"entities": entities,
		"query":    query,
		"total":    len(entities),
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
