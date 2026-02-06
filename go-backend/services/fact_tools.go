// Package services provides business logic and tool implementations for Zettelgarden.
//
// Tool Registry Infrastructure:
// - registry.go: Core tool registration and execution
// - context.go: Tool execution context with transaction support
// - params.go: Parameter extraction and validation helpers
// - types.go: Tool type definitions and constants
//
// Domain-specific tools:
// - card_tools.go: Card CRUD operations and search
// - task_tools.go: Task management and scheduling
// - entity_tools.go: Entity management and linking
// - fact_tools.go: Fact extraction and retrieval
// - template_tools.go: Card template management
// - calendar_tools.go: Calendar integration
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
package services

import (
	"fmt"
)

// RegisterFactTools registers all fact-related tools
func (tr *ToolRegistry) RegisterFactTools() {
	RegisterTool(tr, "search_facts",
		"Search for facts in the user's knowledge base using text or semantic similarity. Returns relevant facts based on the search query.",
		handleSearchFacts,
		ToolParam{Name: "query", Type: "string", Required: true, Desc: "The search query to find relevant facts"},
		ToolParam{Name: "search_type", Type: "string", Required: false, Default: "semantic", Desc: "Type of search: 'text' for exact text matching, 'semantic' for meaning-based search", Enum: []interface{}{"text", "semantic"}},
		ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Desc: "Maximum number of facts to return (default: 10, max: 50)", Minimum: intPtr(1), Maximum: intPtr(50)},
	)

	RegisterTool(tr, "get_card_facts",
		"Retrieve all facts associated with a specific card. Facts are auto-generated from card content.",
		handleGetCardFacts,
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the card to get facts for"},
	)

	RegisterTool(tr, "get_entity_facts",
		"Retrieve all facts linked to a specific entity. Useful for understanding what information exists about a particular entity.",
		handleGetEntityFacts,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to get facts for"},
	)

	RegisterTool(tr, "get_fact_cards",
		"Retrieve all cards that are linked to a specific fact. Shows where a fact appears across the knowledge base.",
		handleGetFactCards,
		ToolParam{Name: "fact_id", Type: "integer", Required: true, Desc: "The ID of the fact to get linked cards for"},
	)
}

// Fact tool handlers

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
