package llms

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-backend/prompts"
	"go-backend/services"
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

	limit := 100
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Execute search based on type
	var cards []map[string]interface{}
	var err error

	if searchType == "text" {
		cards, err = services.ExecuteTextSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	} else {
		cards, err = services.ExecuteSemanticSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
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

	card, err := services.GetFullCard(ctx.DB, ctx.UserID, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	return StructToMap(card), nil
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

	var cards []map[string]interface{}
	var err error

	if direction == "children" {
		cards, err = services.GetChildCards(ctx.DB, ctx.UserID, cardPK)
	} else if direction == "parent" {
		cards, err = services.GetParentCard(ctx.DB, ctx.UserID, cardPK)
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

			result, err := subagentRegistry.ExecuteTool(tc.Function.Name, args, ctx)
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
