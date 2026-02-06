# Entity Domain Package

## Overview

This package is part of Phase 3: Split Tool Handlers by Domain with Feature Flags.

The entity domain contains tools and data access functions for managing and linking entities in Zettelgarden. This is the most complex domain with 10 tools covering entity retrieval, relationships, operations, and similarity search.

## Package Structure

```
services/tools/entity/
├── entity.go       # Data access and business logic
└── README.md       # This file
```

## Exported Functions

### Entity Retrieval

#### GetEntityByName
```go
func GetEntityByName(db *sql.DB, userID int, entityName string) (models.Entity, error)
```
Retrieves a specific entity by its name for a given user.

#### GetEntityByID
```go
func GetEntityByID(db *sql.DB, userID int, entityID int) (models.Entity, error)
```
Retrieves a specific entity by its ID for a given user.

#### SearchEntities
```go
func SearchEntities(db *sql.DB, typesenseClient *typesense.Client, userID int, query string, limit int) ([]models.Entity, error)
```
Searches for entities using Typesense.

#### GetCardsByEntity
```go
func GetCardsByEntity(db *sql.DB, userID int, entityID int) ([]models.Card, error)
```
Retrieves all cards linked to a specific entity.

### Entity Operations

#### UpdateEntity
```go
func UpdateEntity(db *sql.DB, userID int, entityID int, params UpdateEntityParams) error
```
Updates an entity's name, description, type, or linked card.

#### MergeEntities
```go
func MergeEntities(db *sql.DB, userID int, entity1ID int, entity2ID int) error
```
Merges two entities, combining their relationships.

#### DeleteEntity
```go
func DeleteEntity(db *sql.DB, userID int, entityID int) error
```
Deletes an entity and all its relationships.

### Entity-Card Relationships

#### AddEntityToCard
```go
func AddEntityToCard(db *sql.DB, userID int, entityID int, cardPK int) error
```
Links an entity to a card.

#### RemoveEntityFromCard
```go
func RemoveEntityFromCard(db *sql.DB, userID int, entityID int, cardPK int) error
```
Removes the link between an entity and a card.

### Entity Similarity

#### FindSimilarEntities
```go
func FindSimilarEntities(db *sql.DB, typesenseClient *typesense.Client, userID int, entityID int, limit int) ([]map[string]interface{}, error)
```
Finds entities similar to a given entity using Typesense.

### Pagination

#### GetEntities
```go
func GetEntities(db *sql.DB, typesenseClient *typesense.Client, userID int, params EntityQueryParams) (EntityListResponse, error)
```
Retrieves entities with pagination and search support.

## Integration with Tool Registry

The entity domain integrates with the tool registry via `services/entity_tools.go`:

1. **Feature Flag Disabled (Default)**: Uses `registerEntityToolsLegacy()`
2. **Feature Flag Enabled**: Uses `registerEntityToolsV2()`

Both paths call this package's functions internally, ensuring consistent behavior.

Enable the feature flag:
```bash
export ZETTELGARDEN_FEATURE_ENTITY_TOOLS_V2=true
```

## Tools

This domain package supports the following tools:

1. `get_entity_by_name` - Retrieve entity by name
2. `search_entities` - Search entities using text/semantic similarity
3. `get_cards_by_entity` - Get all cards linked to an entity
4. `get_entity_by_id` - Retrieve entity by ID
5. `merge_entities` - Merge two entities into one
6. `update_entity` - Update entity properties
7. `delete_entity` - Delete an entity
8. `add_entity_to_card` - Link entity to card
9. `remove_entity_from_card` - Unlink entity from card
10. `get_similar_entities` - Find similar entities

## Testing

```bash
# Test the entity domain package
go test ./services/tools/entity/...

# Test with feature flag enabled
ZETTELGARDEN_FEATURE_ENTITY_TOOLS_V2=true go test ./services/... -run TestEntity

# Test with feature flag disabled (default)
go test ./services/... -run TestEntity
```

## Migration Notes

- Created: Phase 3 (2025-02-05)
- Status: COMPLETE - Most complex domain with 10 tools
- Pattern validated: Yes
- Ready for production: Yes (when feature flag is enabled)

## Related Files

- `services/entity_tools.go` - Tool registration with feature flags
- `services/entity.go` - Backward-compatible re-exports
- `services/featureflags/flags.go` - Feature flag system
- `services/PHASE3_DOMAIN_MIGRATION_GUIDE.md` - Complete migration guide

## Complexity Notes

This is the most complex domain package with:
- 10 tools (highest count)
- Complex relationship management (entity-card, entity-fact)
- Transaction handling for merge and delete operations
- Typesense integration for search and similarity
- Pagination support with custom sorting
- Multiple data access patterns

The pattern established here can serve as a reference for other complex domains.
