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
// - article_tools.go: Article parsing and creation
// - memory_tools.go: User memory operations
//
// PHASE 3: Domain Package Migration with Feature Flags
// ---------------------------------------------------
// This file now supports both legacy and new domain package registration
// controlled by the FeatureFlagFactTools feature flag.
//
// - Feature flag DISABLED (default): Uses legacy registration in this file
// - Feature flag ENABLED: Uses services/tools/fact package
//
// To enable the new domain package:
//   export ZETTELGARDEN_FEATURE_FACT_TOOLS_V2=true
package services

import (
	"fmt"

	"go-backend/services/featureflags"
	"go-backend/services/tools"
	"go-backend/services/tools/fact"
)

// RegisterFactTools registers all fact-related tools.
//
// This method supports two registration paths controlled by feature flag:
// 1. Legacy path (default): Registers tools directly from this file
// 2. New path (feature flag): Delegates to services/tools/fact package
func (tr *ToolRegistry) RegisterFactTools() {
	if featureflags.IsEnabled(featureflags.FeatureFlagFactTools) {
		// NEW: Use the domain package
		tr.registerFactToolsV2()
	} else {
		// LEGACY: Use the original registration in this file
		tr.registerFactToolsLegacy()
	}
}

// registerFactToolsV2 uses the new fact domain package
func (tr *ToolRegistry) registerFactToolsV2() {
	RegisterTool(tr, "search_facts",
		"Search for facts in the user's knowledge base using text or semantic similarity. Returns relevant facts based on the search query.",
		handleSearchFactsV2,
		ToolParam{Name: "query", Type: "string", Required: true, Desc: "The search query to find relevant facts"},
		ToolParam{Name: "search_type", Type: "string", Required: false, Default: "semantic", Desc: "Type of search: 'text' for exact text matching, 'semantic' for meaning-based search", Enum: []interface{}{"text", "semantic"}},
		ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Desc: "Maximum number of facts to return (default: 10, max: 50)", Minimum: intPtr(1), Maximum: intPtr(50)},
	)

	RegisterTool(tr, "get_card_facts",
		"Retrieve all facts associated with a specific card. Facts are auto-generated from card content.",
		handleGetCardFactsV2,
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the card to get facts for"},
	)

	RegisterTool(tr, "get_entity_facts",
		"Retrieve all facts linked to a specific entity. Useful for understanding what information exists about a particular entity.",
		handleGetEntityFactsV2,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to get facts for"},
	)

	RegisterTool(tr, "get_fact_cards",
		"Retrieve all cards that are linked to a specific fact. Shows where a fact appears across the knowledge base.",
		handleGetFactCardsV2,
		ToolParam{Name: "fact_id", Type: "integer", Required: true, Desc: "The ID of the fact to get linked cards for"},
	)
}

// registerFactToolsLegacy is the original fact tools registration.
// Kept for backward compatibility and as a fallback.
func (tr *ToolRegistry) registerFactToolsLegacy() {
	RegisterTool(tr, "search_facts",
		"Search for facts in the user's knowledge base using text or semantic similarity. Returns relevant facts based on the search query.",
		handleSearchFactsLegacy,
		ToolParam{Name: "query", Type: "string", Required: true, Desc: "The search query to find relevant facts"},
		ToolParam{Name: "search_type", Type: "string", Required: false, Default: "semantic", Desc: "Type of search: 'text' for exact text matching, 'semantic' for meaning-based search", Enum: []interface{}{"text", "semantic"}},
		ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Desc: "Maximum number of facts to return (default: 10, max: 50)", Minimum: intPtr(1), Maximum: intPtr(50)},
	)

	RegisterTool(tr, "get_card_facts",
		"Retrieve all facts associated with a specific card. Facts are auto-generated from card content.",
		handleGetCardFactsLegacy,
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the card to get facts for"},
	)

	RegisterTool(tr, "get_entity_facts",
		"Retrieve all facts linked to a specific entity. Useful for understanding what information exists about a particular entity.",
		handleGetEntityFactsLegacy,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to get facts for"},
	)

	RegisterTool(tr, "get_fact_cards",
		"Retrieve all cards that are linked to a specific fact. Shows where a fact appears across the knowledge base.",
		handleGetFactCardsLegacy,
		ToolParam{Name: "fact_id", Type: "integer", Required: true, Desc: "The ID of the fact to get linked cards for"},
	)
}

// V2 fact tool handlers (use domain package logic)

func handleSearchFactsV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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
		facts, err = fact.ExecuteFactTextSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	} else {
		facts, err = fact.ExecuteFactSemanticSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	data := map[string]interface{}{
		"facts":       facts,
		"query":       query,
		"search_type": searchType,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(facts)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

func handleGetCardFactsV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	facts, lerr := fact.GetCardFacts(ctx.DB, ctx.UserID, cardPK)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get card facts: %v", lerr)
	}

	var results []map[string]interface{}
	for _, fact := range facts {
		results = append(results, StructToMap(fact))
	}

	data := map[string]interface{}{
		"facts": results,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(facts)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

func handleGetEntityFactsV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	facts, lerr := fact.GetEntityFacts(ctx.DB, ctx.UserID, entityID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get entity facts: %v", lerr)
	}

	var results []map[string]interface{}
	for _, fact := range facts {
		results = append(results, StructToMap(fact))
	}

	data := map[string]interface{}{
		"facts": results,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(facts)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

func handleGetFactCardsV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	factID, err := getIntParam(args, "fact_id")
	if err != nil {
		return nil, err
	}

	cards, lerr := fact.GetFactCards(ctx.DB, ctx.UserID, factID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get fact cards: %v", lerr)
	}

	var results []map[string]interface{}
	for _, card := range cards {
		results = append(results, StructToMap(card))
	}

	data := map[string]interface{}{
		"cards": results,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(cards)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

// Legacy fact tool handlers (kept for backward compatibility)

func handleSearchFactsLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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
		facts, err = fact.ExecuteFactTextSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	} else {
		facts, err = fact.ExecuteFactSemanticSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	data := map[string]interface{}{
		"facts":       facts,
		"query":       query,
		"search_type": searchType,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(facts)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

func handleGetCardFactsLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	facts, lerr := fact.GetCardFacts(ctx.DB, ctx.UserID, cardPK)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get card facts: %v", lerr)
	}

	var results []map[string]interface{}
	for _, fact := range facts {
		results = append(results, StructToMap(fact))
	}

	data := map[string]interface{}{
		"facts": results,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(facts)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

func handleGetEntityFactsLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	entityID, err := getIntParam(args, "entity_id")
	if err != nil {
		return nil, err
	}

	facts, lerr := fact.GetEntityFacts(ctx.DB, ctx.UserID, entityID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get entity facts: %v", lerr)
	}

	var results []map[string]interface{}
	for _, fact := range facts {
		results = append(results, StructToMap(fact))
	}

	data := map[string]interface{}{
		"facts": results,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(facts)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}

func handleGetFactCardsLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	factID, err := getIntParam(args, "fact_id")
	if err != nil {
		return nil, err
	}

	cards, lerr := fact.GetFactCards(ctx.DB, ctx.UserID, factID)
	if lerr != nil {
		return nil, fmt.Errorf("failed to get fact cards: %v", lerr)
	}

	var results []map[string]interface{}
	for _, card := range cards {
		results = append(results, StructToMap(card))
	}

	data := map[string]interface{}{
		"cards": results,
	}
	metadata := tools.NewMetadata(tools.WithTotal(len(cards)))
	return tools.WrapToolSuccessWithMetadata(data, metadata), nil
}
