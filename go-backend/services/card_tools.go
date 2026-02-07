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
//
// PHASE 3: Domain Package Migration with Feature Flags
// ---------------------------------------------------
// This file now supports both legacy and new domain package registration
// controlled by the FeatureFlagCardTools feature flag.
//
// - Feature flag DISABLED (default): Uses legacy registration in this file
// - Feature flag ENABLED: Uses services/tools/card package
//
// To enable the new domain package:
//   export ZETTELGARDEN_FEATURE_CARD_TOOLS_V2=true
package services

import (
	"fmt"
	"go-backend/models"
	"reflect"

	"go-backend/services/featureflags"
	"go-backend/services/tools/card"
)

// RegisterCardTools registers all card-related tools.
//
// This method supports two registration paths controlled by feature flag:
// 1. Legacy path (default): Registers tools directly from this file
// 2. New path (feature flag): Delegates to services/tools/card package
func (tr *ToolRegistry) RegisterCardTools() {
	if featureflags.IsEnabled(featureflags.FeatureFlagCardTools) {
		// NEW: Use the domain package
		tr.registerCardToolsV2()
	} else {
		// LEGACY: Use the original registration in this file
		tr.registerCardToolsLegacy()
	}
}

// registerCardToolsV2 uses the new card domain package
func (tr *ToolRegistry) registerCardToolsV2() {
	// Search cards - text or semantic search
	RegisterTool(tr,
		"search_cards",
		"Search for cards in the user's knowledge base using text or semantic similarity. Returns relevant cards based on the search query.",
		handleSearchCardsV2,
		ToolParam{Name: "query", Type: "string", Required: true, Desc: "The search query to find relevant cards"},
		ToolParam{Name: "search_type", Type: "string", Required: false, Default: "semantic", Enum: []interface{}{"text", "semantic"}, Desc: "Type of search: 'text' for exact text matching, 'semantic' for meaning-based search"},
		ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Minimum: intPtr(1), Maximum: intPtr(50), Desc: "Maximum number of cards to return (default: 10, max: 50)"},
	)

	// Get card by ID
	RegisterTool(tr,
		"get_card_by_id",
		"Retrieve a specific card by its ID. Returns the full card content including title, body, tags, and metadata.",
		handleGetCardByIDV2,
		ToolParam{Name: "card_id", Type: "integer", Required: true, Desc: "The ID of the card to retrieve"},
	)

	// Browse card hierarchy
	RegisterTool(tr,
		"browse_card_hierarchy",
		"Browse the hierarchical structure of cards. Get parent or child cards of a specific card, optionally traversing multiple levels deep.",
		handleBrowseCardHierarchyV2,
		ToolParam{Name: "card_id", Type: "integer", Required: true, Desc: "The ID of the card to browse from"},
		ToolParam{Name: "direction", Type: "string", Required: true, Enum: []interface{}{"children", "parent"}, Desc: "Direction to browse: 'children' for child cards, 'parent' for parent cards"},
		ToolParam{Name: "depth", Type: "integer", Required: false, Default: 1, Minimum: intPtr(-1), Desc: "How many levels to traverse (default: 1 for immediate children/parent only). Use -1 for unlimited depth to get all descendants or ancestors."},
	)

	// Create card
	RegisterTool(tr,
		"create_card",
		"Create a new card with title, body, and optional link. The card_id will be set to empty string for user categorization.",
		handleCreateCardV2,
		ToolParam{Name: "title", Type: "string", Required: true, Desc: "Title for the new card (required)"},
		ToolParam{Name: "body", Type: "string", Required: true, Desc: "Body content for the new card (required)"},
		ToolParam{Name: "link", Type: "string", Required: false, Desc: "Optional link for the card (can be empty string)"},
	)

	// Update card
	RegisterTool(tr,
		"update_card",
		"Update an existing card's title, body, or link. All fields except id and existing_card_id are optional - only provided fields will be updated.",
		handleUpdateCardV2,
		ToolParam{Name: "id", Type: "integer", Required: true, Desc: "The primary key ID of the card to update (required)"},
		ToolParam{Name: "existing_card_id", Type: "string", Required: true, Desc: "The current card_id (user-readable identifier) for verification (required)"},
		ToolParam{Name: "title", Type: "string", Required: false, Desc: "New title for the card (optional)"},
		ToolParam{Name: "body", Type: "string", Required: false, Desc: "New body content for the card (optional)"},
		ToolParam{Name: "link", Type: "string", Required: false, Desc: "New link for the card (optional)"},
	)

	// Get card analysis
	RegisterTool(tr,
		"get_card_analysis",
		"Retrieve the analysis/summary for a specific card by its primary key ID. Returns structured analysis with sections, theses, and arguments.",
		handleGetCardAnalysisV2,
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the card to get analysis for. This is different from the card_id, which is meant to be human readable"},
		ToolParam{Name: "card_id", Type: "integer", Required: false, Desc: "The human readable identifier of the card to get analysis for. This is different from the card_pk, which is just an int"},
	)
}

// registerCardToolsLegacy is the original card tools registration.
// Kept for backward compatibility and as a fallback.
func (tr *ToolRegistry) registerCardToolsLegacy() {
	// Search cards - text or semantic search
	RegisterTool(tr,
		"search_cards",
		"Search for cards in the user's knowledge base using text or semantic similarity. Returns relevant cards based on the search query.",
		handleSearchCardsLegacy,
		ToolParam{Name: "query", Type: "string", Required: true, Desc: "The search query to find relevant cards"},
		ToolParam{Name: "search_type", Type: "string", Required: false, Default: "semantic", Enum: []interface{}{"text", "semantic"}, Desc: "Type of search: 'text' for exact text matching, 'semantic' for meaning-based search"},
		ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Minimum: intPtr(1), Maximum: intPtr(50), Desc: "Maximum number of cards to return (default: 10, max: 50)"},
	)

	// Get card by ID
	RegisterTool(tr,
		"get_card_by_id",
		"Retrieve a specific card by its ID. Returns the full card content including title, body, tags, and metadata.",
		handleGetCardByIDLegacy,
		ToolParam{Name: "card_id", Type: "integer", Required: true, Desc: "The ID of the card to retrieve"},
	)

	// Browse card hierarchy
	RegisterTool(tr,
		"browse_card_hierarchy",
		"Browse the hierarchical structure of cards. Get parent or child cards of a specific card, optionally traversing multiple levels deep.",
		handleBrowseCardHierarchyLegacy,
		ToolParam{Name: "card_id", Type: "integer", Required: true, Desc: "The ID of the card to browse from"},
		ToolParam{Name: "direction", Type: "string", Required: true, Enum: []interface{}{"children", "parent"}, Desc: "Direction to browse: 'children' for child cards, 'parent' for parent cards"},
		ToolParam{Name: "depth", Type: "integer", Required: false, Default: 1, Minimum: intPtr(-1), Desc: "How many levels to traverse (default: 1 for immediate children/parent only). Use -1 for unlimited depth to get all descendants or ancestors."},
	)

	// Create card
	RegisterTool(tr,
		"create_card",
		"Create a new card with title, body, and optional link. The card_id will be set to empty string for user categorization.",
		handleCreateCardLegacy,
		ToolParam{Name: "title", Type: "string", Required: true, Desc: "Title for the new card (required)"},
		ToolParam{Name: "body", Type: "string", Required: true, Desc: "Body content for the new card (required)"},
		ToolParam{Name: "link", Type: "string", Required: false, Desc: "Optional link for the card (can be empty string)"},
	)

	// Update card
	RegisterTool(tr,
		"update_card",
		"Update an existing card's title, body, or link. All fields except id and existing_card_id are optional - only provided fields will be updated.",
		handleUpdateCardLegacy,
		ToolParam{Name: "id", Type: "integer", Required: true, Desc: "The primary key ID of the card to update (required)"},
		ToolParam{Name: "existing_card_id", Type: "string", Required: true, Desc: "The current card_id (user-readable identifier) for verification (required)"},
		ToolParam{Name: "title", Type: "string", Required: false, Desc: "New title for the card (optional)"},
		ToolParam{Name: "body", Type: "string", Required: false, Desc: "New body content for the card (optional)"},
		ToolParam{Name: "link", Type: "string", Required: false, Desc: "New link for the card (optional)"},
	)

	// Get card analysis
	RegisterTool(tr,
		"get_card_analysis",
		"Retrieve the analysis/summary for a specific card by its primary key ID. Returns structured analysis with sections, theses, and arguments.",
		handleGetCardAnalysisLegacy,
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the card to get analysis for. This is different from the card_id, which is meant to be human readable"},
		ToolParam{Name: "card_id", Type: "integer", Required: false, Desc: "The human readable identifier of the card to get analysis for. This is different from the card_pk, which is just an int"},
	)
}

// V2 card tool handlers (use domain package logic)

func handleSearchCardsV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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

	// Execute search based on type
	var results []map[string]interface{}

	if searchType == "text" {
		results, err = card.ExecuteTextSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	} else {
		results, err = card.ExecuteSemanticSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	return map[string]interface{}{
		"cards":       results,
		"query":       query,
		"search_type": searchType,
		"total":       len(results),
	}, nil
}

func handleGetCardByIDV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardID, err := getIntParam(args, "card_id")
	if err != nil {
		return nil, err
	}

	c, err := card.GetFullCard(ctx.DB, ctx.UserID, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	return card.StructToMap(c), nil
}

func handleCreateCardV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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
	newCard, err := card.CreateCard(ctx.DB, ctx.UserID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %v", err)
	}

	result := card.StructToMap(newCard)
	// Add metadata about the operation for frontend refresh detection
	result["operation"] = "card_created"
	result["card_pk"] = newCard.ID
	result["card_id"] = newCard.CardID

	return result, nil
}

func handleGetCardAnalysisV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	// Get the card analysis using the domain package function
	analysis, err := card.GetCardAnalysis(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get card analysis: %v", err)
	}

	// Convert analysis to map for tool response
	return map[string]interface{}{
		"card_pk":  cardPK,
		"analysis": analysis,
	}, nil
}

func handleUpdateCardV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "id")
	if err != nil {
		return nil, err
	}

	existingCardID, err := getStringParam(args, "existing_card_id")
	if err != nil {
		return nil, err
	}

	// Get the current card
	currentCard, err := card.GetFullCard(ctx.DB, ctx.UserID, cardPK)
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
	updatedCard, err := card.UpdateCard(ctx.DB, ctx.UserID, cardPK, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update card: %v", err)
	}

	result := card.StructToMap(updatedCard)
	// Add metadata about the operation for frontend refresh detection
	result["operation"] = "card_updated"
	result["card_pk"] = cardPK
	result["card_id"] = updatedCard.CardID

	return result, nil
}

func handleBrowseCardHierarchyV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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
		cards, err = card.GetChildCardsWithDepth(ctx.DB, ctx.UserID, cardPK, depth)
	} else if direction == "parent" {
		cards, err = card.GetParentCardsWithDepth(ctx.DB, ctx.UserID, cardPK, depth)
	} else {
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to browse hierarchy: %v", err)
	}

	var results []map[string]interface{}
	for _, c := range cards {
		results = append(results, card.StructToMap(c))
	}

	return map[string]interface{}{
		"cards":     results,
		"direction": direction,
		"depth":     depth,
		"total":     len(cards),
	}, nil
}

// Legacy card tool handlers (kept for backward compatibility)

func handleSearchCardsLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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

	// Execute search based on type
	var results []map[string]interface{}

	if searchType == "text" {
		results, err = card.ExecuteTextSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	} else {
		results, err = card.ExecuteSemanticSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	return map[string]interface{}{
		"cards":       results,
		"query":       query,
		"search_type": searchType,
		"total":       len(results),
	}, nil
}

func handleGetCardByIDLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardID, err := getIntParam(args, "card_id")
	if err != nil {
		return nil, err
	}

	c, err := card.GetFullCard(ctx.DB, ctx.UserID, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	return card.StructToMap(c), nil
}

func handleCreateCardLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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
	newCard, err := card.CreateCard(ctx.DB, ctx.UserID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create card: %v", err)
	}

	result := card.StructToMap(newCard)
	// Add metadata about the operation for frontend refresh detection
	result["operation"] = "card_created"
	result["card_pk"] = newCard.ID
	result["card_id"] = newCard.CardID

	return result, nil
}

func handleGetCardAnalysisLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "card_pk")
	if err != nil {
		return nil, err
	}

	// Get the card analysis using the domain package function
	analysis, err := card.GetCardAnalysis(ctx.DB, ctx.UserID, cardPK)
	if err != nil {
		return nil, fmt.Errorf("failed to get card analysis: %v", err)
	}

	// Convert analysis to map for tool response
	return map[string]interface{}{
		"card_pk":  cardPK,
		"analysis": analysis,
	}, nil
}

func handleUpdateCardLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardPK, err := getIntParam(args, "id")
	if err != nil {
		return nil, err
	}

	existingCardID, err := getStringParam(args, "existing_card_id")
	if err != nil {
		return nil, err
	}

	// Get the current card
	currentCard, err := card.GetFullCard(ctx.DB, ctx.UserID, cardPK)
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
	updatedCard, err := card.UpdateCard(ctx.DB, ctx.UserID, cardPK, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update card: %v", err)
	}

	result := card.StructToMap(updatedCard)
	// Add metadata about the operation for frontend refresh detection
	result["operation"] = "card_updated"
	result["card_pk"] = cardPK
	result["card_id"] = updatedCard.CardID

	return result, nil
}

func handleBrowseCardHierarchyLegacy(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
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
		cards, err = card.GetChildCardsWithDepth(ctx.DB, ctx.UserID, cardPK, depth)
	} else if direction == "parent" {
		cards, err = card.GetParentCardsWithDepth(ctx.DB, ctx.UserID, cardPK, depth)
	} else {
		return nil, fmt.Errorf("invalid direction: %s", direction)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to browse hierarchy: %v", err)
	}

	var results []map[string]interface{}
	for _, c := range cards {
		results = append(results, card.StructToMap(c))
	}

	return map[string]interface{}{
		"cards":     results,
		"direction": direction,
		"depth":     depth,
		"total":     len(cards),
	}, nil
}

// StructToMap converts a struct to a map[string]interface{}
// This is a utility function used by multiple tools
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
