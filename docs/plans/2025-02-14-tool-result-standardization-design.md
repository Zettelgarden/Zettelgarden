# Tool Result Format Standardization Design

## Problem Statement

Tool results across all LLM tools have inconsistent formats, making frontend handling difficult. Examples of current inconsistencies:

1. **Search tools** return nested data: `{"cards": [...], "query": "...", "total": 5}`
2. **Get-by-ID tools** return flat maps: `{id: 1, title: "..."}`
3. **Delete tools** return status messages: `{"status": "deleted", "task_id": 1}`
4. **Error responses** use a different structure: `{"error": {...}}`
5. **get_next_child_id** uses boolean error field: `{"error": false, "new_id": "..."}`

## Solution: Central Wrapper Function Approach

Create helper functions that all tools use to format responses consistently.

## Standard Response Format

### Success Response

```json
{
  "success": true,
  "data": {
    // Tool-specific response data
  },
  "metadata": {
    "total": 5,           // Required for list/search results
    "operation": "created", // Optional: operation performed
    "tool": "search_cards", // Optional: tool name for debugging
    // ... other optional metadata
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "type": "not_found",
    "message": "Card with ID 123 not found",
    "retryable": false,
    "tool_name": "get_card_by_id",
    "arguments": {"card_id": 123},
    "suggestion": "..."
  }
}
```

## File Structure

```
go-backend/services/tools/
├── response.go           [NEW] Success/Error wrapper functions
├── response_test.go      [NEW] Tests for wrapper functions
└── [existing files...]
```

## API Design

### Success Wrapper

```go
// WrapToolSuccess creates a standardized success response
func WrapToolSuccess(data interface{}) map[string]interface{}

// WrapToolSuccessWithMetadata creates a success response with metadata
func WrapToolSuccessWithMetadata(data interface{}, metadata map[string]interface{}) map[string]interface{}

// WrapToolSuccessWithList creates a success response for list results (includes total)
func WrapToolSuccessWithList(items []map[string]interface{}) map[string]interface{}
```

### Error Wrapper

```go
// WrapToolError creates a standardized error response
// Takes existing *ToolError and formats it with success: false wrapper
func WrapToolError(toolErr *models.ToolError) map[string]interface{}
```

### Metadata Helpers

```go
// NewMetadata creates a metadata map for common fields
func NewMetadata() map[string]interface{}

// WithTotal adds/sets the total count
func WithTotal(metadata map[string]interface{}, count int) map[string]interface{}

// WithOperation adds/sets the operation type
func WithOperation(metadata map[string]interface{}, operation string) map[string]interface{}

// WithTool adds/sets the tool name
func WithTool(metadata map[string]interface{}, toolName string) map[string]interface{}
```

## Tool Category Response Patterns

### 1. Single Item Retrieval (get_card_by_id, get_entity_by_id, get_task_by_id)

**Before:**
```go
return card.StructToMap(card), nil
```

**After:**
```go
return WrapToolSuccess(card.StructToMap(card)), nil
```

### 2. List/Search Results (search_cards, get_tasks, search_entities)

**Before:**
```go
return map[string]interface{}{
    "cards": results,
    "query": query,
    "total": len(results),
}, nil
```

**After:**
```go
return WrapToolSuccessWithMetadata(
    map[string]interface{}{
        "cards": results,
        "query": query,
    },
    NewMetadata(WithTotal(len(results)),
), nil
```

Or simpler:
```go
return WrapToolSuccessWithList(results), nil
// But this loses the "query" field - need to decide if that's OK
```

**Decision:** Search results should keep their query/filter params in `data`, so use `WithTotal`:

```go
return WrapToolSuccessWithMetadata(
    map[string]interface{}{
        "cards": results,
        "query": query,
        "search_type": searchType,
    },
    NewMetadata(WithTotal(len(results))),
), nil
```

### 3. Create Operations (create_card, create_task)

**Before:**
```go
result := card.StructToMap(newCard)
result["operation"] = "card_created"
result["card_pk"] = newCard.ID
result["card_id"] = newCard.CardID
return result, nil
```

**After:**
```go
cardMap := card.StructToMap(newCard)
return WrapToolSuccessWithMetadata(
    cardMap,
    NewMetadata(
        WithOperation("card_created"),
        WithTool("create_card"),
    ),
), nil
```

### 4. Delete Operations (delete_task, delete_entity)

**Before:**
```go
return map[string]interface{}{
    "status": "deleted",
    "task_id": taskID,
}, nil
```

**After:**
```go
return WrapToolSuccessWithMetadata(
    map[string]interface{}{
        "deleted_id": taskID,
    },
    NewMetadata(WithOperation("deleted")),
), nil
```

### 5. Update Operations (update_card, update_task)

**Before:**
```go
result := card.StructToMap(updatedCard)
result["operation"] = "card_updated"
return result, nil
```

**After:**
```go
return WrapToolSuccessWithMetadata(
    card.StructToMap(updatedCard),
    NewMetadata(WithOperation("card_updated")),
), nil
```

### 6. Special/Mixed Results (get_next_child_id, merge_entities)

**get_next_child_id:**

**Before:**
```go
if nextID == "" {
    return map[string]interface{}{
        "error": true,
        "message": "Parent card not found",
        "new_id": "",
    }, nil
}
return map[string]interface{}{
    "error": false,
    "message": "",
    "new_id": nextID,
}, nil
```

**After:**
```go
if nextID == "" {
    return nil, fmt.Errorf("parent card not found") // Let WrapToolError handle it
}
return WrapToolSuccess(
    map[string]interface{}{"new_id": nextID},
), nil
```

**merge_entities:**

**Before:**
```go
return map[string]interface{}{
    "status": "merged",
    "entity1_id": entity1ID,
    "entity2_id": entity2ID,
    "surviving_id": entity1ID,
    "message": fmt.Sprintf("Successfully merged..."),
}, nil
```

**After:**
```go
return WrapToolSuccessWithMetadata(
    map[string]interface{}{
        "entity1_id": entity1ID,
        "entity2_id": entity2ID,
        "surviving_id": entity1ID,
    },
    NewMetadata(WithOperation("merged")),
), nil
```

## Migration Strategy

### Phase 1: Foundation (1 PR)
1. Create `services/tools/response.go` with wrapper functions
2. Add comprehensive tests in `response_test.go`
3. Update existing `WrapToolError` behavior to include `success: false`
4. No tool changes yet - just infrastructure

### Phase 2: Card Tools (1 PR)
1. Update all card_tools.go handlers
2. Update frontend to handle new format for card tools
3. Add feature flag to allow rollback

### Phase 3: Other Tools (Multiple PRs)
1. Task tools
2. Entity tools
3. Fact tools
4. Template/Memory/Calendar/Article tools

Each PR should:
- Update tool handlers
- Update frontend handling
- Test thoroughly

### Phase 4: Cleanup (1 PR)
1. Remove feature flags once all tools migrated
2. Update documentation
3. Remove old format handling from frontend

## Frontend Changes Required

### Type Definitions

Update `src/api/chat.ts`:

```typescript
export interface ToolResultSuccess {
  success: true;
  data: any;
  metadata?: {
    total?: number;
    operation?: string;
    tool?: string;
    [key: string]: any;
  };
}

export interface ToolResultError {
  success: false;
  error: {
    type: string;
    message: string;
    retryable: boolean;
    tool_name: string;
    arguments?: Record<string, any>;
    suggestion?: string;
  };
}

export type StandardToolResult = ToolResultSuccess | ToolResultError;
```

### Detection Logic

Add helper in frontend:

```typescript
function isStandardToolResult(result: any): result is StandardToolResult {
  return typeof result === 'object' && 'success' in result;
}
```

Handle both formats during transition:

```typescript
if (isStandardToolResult(result)) {
  if (result.success) {
    // Handle new format
    const data = result.data;
    const total = result.metadata?.total;
  } else {
    // Handle error
    showError(result.error);
  }
} else {
  // Handle legacy format
  if (result.error) {
    showError(result.error);
  } else {
    // Legacy success format
  }
}
```

## Testing Strategy

### Unit Tests

Each wrapper function should have:
- Happy path tests
- Empty/null data tests
- Metadata combination tests

### Integration Tests

For each tool category:
- Test actual tool execution
- Verify response format
- Test error paths

### Frontend Tests

- Test format detection logic
- Test rendering with both formats
- Test error display with new format

## Rollback Plan

If issues arise:
1. Feature flag allows disabling new format per tool category
2. Frontend can detect and handle both formats
3. Can revert individual tool categories independently

## Open Questions

1. **Nested list data**: Should search results keep `{"cards": [...]}` or flatten to `{"data": [...]}`?
   - **Decision**: Keep nested structure in `data` - e.g., `data.cards`
   - Reason: Maintains semantic meaning, frontend knows what type of data it is

2. **total field location**: Should `total` be in `data` or `metadata`?
   - **Decision**: Put in `metadata` as it's about the response, not the data itself

3. **Should we require metadata for all tools?**
   - **Decision**: No, only include relevant metadata. `total` required for lists, others optional

4. **Error vs success for not-found**: Is "not found" an error or success with empty data?
   - **Decision**: Error - use 404 semantics for clarity

## Success Criteria

- [ ] All tools return consistent format
- [ ] Frontend handles new format correctly
- [ ] Error handling is uniform
- [ ] Tests cover all new code
- [ ] No breaking changes to users during transition
