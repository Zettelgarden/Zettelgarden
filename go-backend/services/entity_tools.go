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

// RegisterEntityTools registers all entity-related tools
func (tr *ToolRegistry) RegisterEntityTools() {
	// Get entity by name
	RegisterTool(tr,
		"get_entity_by_name",
		"Retrieve a specific entity by its name. Returns the full entity information including linked card if available.",
		handleGetEntityByName,
		ToolParam{Name: "entity_name", Type: "string", Required: true, Desc: "The name of the entity to retrieve"},
	)

	// Search entities
	RegisterTool(tr,
		"search_entities",
		"Search for entities in the user's knowledge base using text or semantic similarity. Returns relevant entities based on the search query.",
		handleSearchEntities,
		ToolParam{Name: "query", Type: "string", Required: true, Desc: "The search query to find relevant entities"},
		ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Minimum: intPtr(1), Maximum: intPtr(50), Desc: "Maximum number of entities to return (default: 10, max: 50)"},
	)

	// Get cards by entity
	RegisterTool(tr,
		"get_cards_by_entity",
		"Retrieve all cards that are linked to a specific entity. This is the primary search method for finding content related to entities - use this before other search methods.",
		handleGetCardsByEntity,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to get linked cards for"},
	)

	// Get entity by ID
	RegisterTool(tr,
		"get_entity_by_id",
		"Retrieve a specific entity by its ID. Returns the full entity information including linked card if available.",
		handleGetEntityByID,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to retrieve"},
	)

	// Merge entities
	RegisterTool(tr,
		"merge_entities",
		"Merge two entities into one. The first entity will absorb all relationships and data from the second entity, which will be deleted. Use this when you find duplicate entities that should be combined.",
		handleMergeEntities,
		ToolParam{Name: "entity1_id", Type: "integer", Required: true, Desc: "The ID of the entity that will survive (all data from entity2 will be merged into this one)"},
		ToolParam{Name: "entity2_id", Type: "integer", Required: true, Desc: "The ID of the entity that will be deleted after merging its data into entity1"},
	)

	// Update entity
	RegisterTool(tr,
		"update_entity",
		"Update an existing entity's name, description, type, or linked card. Only provided fields will be updated.",
		handleUpdateEntity,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to update (required)"},
		ToolParam{Name: "name", Type: "string", Required: false, Desc: "New name for the entity (optional)"},
		ToolParam{Name: "description", Type: "string", Required: false, Desc: "New description for the entity (optional)"},
		ToolParam{Name: "type", Type: "string", Required: false, Desc: "New type for the entity (optional, e.g., 'person', 'organization', 'concept')"},
		ToolParam{Name: "card_pk", Type: "integer", Required: false, Desc: "Primary key of the card to link to this entity (optional, set to null to remove link)"},
	)

	// Delete entity
	RegisterTool(tr,
		"delete_entity",
		"Delete an entity by its ID. This will also remove all card and fact relationships for this entity. This action cannot be undone.",
		handleDeleteEntity,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to delete"},
	)

	// Add entity to card
	RegisterTool(tr,
		"add_entity_to_card",
		"Link an entity to a card. This creates a relationship between the entity and the card, making the card appear in entity searches.",
		handleAddEntityToCard,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to link"},
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the card to link to"},
	)

	// Remove entity from card
	RegisterTool(tr,
		"remove_entity_from_card",
		"Remove the link between an entity and a card. This will not delete the entity or the card, only their relationship.",
		handleRemoveEntityFromCard,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity"},
		ToolParam{Name: "card_pk", Type: "integer", Required: true, Desc: "The primary key ID of the card"},
	)

	// Get similar entities
	RegisterTool(tr,
		"get_similar_entities",
		"Find entities that are similar to a given entity based on semantic similarity of their names and descriptions. Useful for discovering potentially duplicate entities.",
		handleGetSimilarEntities,
		ToolParam{Name: "entity_id", Type: "integer", Required: true, Desc: "The ID of the entity to find similar entities for"},
		ToolParam{Name: "limit", Type: "integer", Required: false, Default: 10, Desc: "Maximum number of similar entities to return (default: 10)"},
	)
}

// Entity tool handlers

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
