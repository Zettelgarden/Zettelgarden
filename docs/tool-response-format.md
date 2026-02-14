# Tool Response Format

All LLM tools in the Zettelgarden backend return responses in a standardized format. This ensures consistency across all tools and makes frontend handling more predictable.

## Overview

The tool response standardization provides a uniform structure for all tool results, including both success and error cases. This format is used by all tool handlers in the backend services.

## Standard Response Format

### Success Response

```json
{
  "success": true,
  "data": {
    // Tool-specific response data
    // This varies by tool type (see examples below)
  },
  "metadata": {
    "total": 5,           // Required for list/search results
    "operation": "created", // Optional: operation performed
    "tool": "create_card" // Optional: tool name for debugging
    // ... other optional metadata fields
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
    "suggestion": "Verify the card ID or check if the card still exists"
  }
}
```

## Helper Functions

Backend tools use these helper functions from `go-backend/services/tools/response.go`:

### Success Wrappers

- `WrapToolSuccess(data)` - Simple success response with just data
- `WrapToolSuccessWithMetadata(data, metadata)` - Success with custom metadata
- `WrapToolSuccessWithList(items)` - Success with list items and automatic total count

### Error Wrapper

- `WrapToolError(toolErr)` - Standardized error response using ToolError struct

### Metadata Builders

- `NewMetadata(...)` - Creates metadata map with functional options
- `WithTotal(n)` - Adds total count
- `WithOperation(s)` - Adds operation name
- `WithTool(s)` - Adds tool name

## Examples by Tool Category

### Single Item Retrieval

**Tools:** `get_card_by_id`, `get_entity_by_id`, `get_task_by_id`

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 123,
    "title": "My Card",
    "body": "# Content here",
    "card_id": "abc123",
    // ... other card fields
  }
}
```

**Code:**
```go
return WrapToolSuccess(card.StructToMap(card)), nil
```

### List/Search Results

**Tools:** `search_cards`, `get_tasks`, `search_entities`, `get_cards_by_entity`

**Response:**
```json
{
  "success": true,
  "data": {
    "cards": [...],     // or "tasks", "entities", etc.
    "query": "test",     // optional: search query
    "search_type": "semantic"  // optional: search parameters
  },
  "metadata": {
    "total": 5
  }
}
```

**Code:**
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

### Create Operations

**Tools:** `create_card`, `create_task`, `create_entity`

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 124,
    "title": "New Card",
    // ... all created item fields
  },
  "metadata": {
    "operation": "card_created",
    "tool": "create_card"
  }
}
```

**Code:**
```go
return WrapToolSuccessWithMetadata(
    card.StructToMap(newCard),
    NewMetadata(
        WithOperation("card_created"),
        WithTool("create_card"),
    ),
), nil
```

### Update Operations

**Tools:** `update_card`, `update_task`, `update_entity`

**Response:**
```json
{
  "success": true,
  "data": {
    // Updated item fields
  },
  "metadata": {
    "operation": "card_updated"
  }
}
```

**Code:**
```go
return WrapToolSuccessWithMetadata(
    card.StructToMap(updatedCard),
    NewMetadata(WithOperation("card_updated")),
), nil
```

### Delete Operations

**Tools:** `delete_task`, `delete_entity`, `delete_fact`

**Response:**
```json
{
  "success": true,
  "data": {
    "deleted_id": 42
  },
  "metadata": {
    "operation": "deleted"
  }
}
```

**Code:**
```go
return WrapToolSuccessWithMetadata(
    map[string]interface{}{"deleted_id": taskID},
    NewMetadata(WithOperation("deleted")),
), nil
```

### Special Operations

**Tools:** `merge_entities`, `complete_and_schedule_task`, `get_next_child_id`

These use custom data structures but follow the same wrapper pattern.

**Example - merge_entities:**
```json
{
  "success": true,
  "data": {
    "entity1_id": 1,
    "entity2_id": 2,
    "surviving_id": 1
  },
  "metadata": {
    "operation": "merged"
  }
}
```

**Example - get_next_child_id:**
```json
{
  "success": true,
  "data": {
    "new_id": "1.2.3"
  }
}
```

## Error Response Examples

### Not Found Error
```json
{
  "success": false,
  "error": {
    "type": "not_found",
    "message": "Card with ID 999 not found",
    "retryable": false,
    "tool_name": "get_card_by_id",
    "arguments": {"card_id": 999},
    "suggestion": "Check the card ID or verify the card hasn't been deleted"
  }
}
```

### Validation Error
```json
{
  "success": false,
  "error": {
    "type": "validation",
    "message": "Required parameter 'card_id' is missing",
    "retryable": false,
    "tool_name": "update_card",
    "arguments": {"title": "New Title"},
    "suggestion": "Provide the card_id parameter"
  }
}
```

### Database Error
```json
{
  "success": false,
  "error": {
    "type": "database",
    "message": "Failed to connect to database",
    "retryable": true,
    "tool_name": "search_cards",
    "arguments": {"query": "test"},
    "suggestion": "This is a temporary error. Please try again."
  }
}
```

## Frontend Integration

The frontend should check for the `success` field to determine outcome:

```typescript
interface ToolResultSuccess {
  success: true;
  data: any;
  metadata?: {
    total?: number;
    operation?: string;
    tool?: string;
    [key: string]: any;
  };
}

interface ToolResultError {
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

type StandardToolResult = ToolResultSuccess | ToolResultError;
```

## Implementation Status

All tool categories have been migrated to the standardized format:

- [x] Card tools (`go-backend/services/tools/card/`)
- [x] Task tools (`go-backend/services/tools/task/`)
- [x] Entity tools (`go-backend/services/tools/entity/`)
- [x] Fact tools (`go-backend/services/tools/fact/`)
- [x] Template tools (`go-backend/services/tools/template/`)
- [x] Memory tools (`go-backend/services/tools/memory/`)
- [x] Calendar tools (`go-backend/services/tools/calendar/`)
- [x] Article tools (`go-backend/services/tools/article/`)

## Related Files

- Backend implementation: `go-backend/services/tools/response.go`
- Backend tests: `go-backend/services/tools/response_test.go`
- Frontend types: `zettelkasten-front/src/api/chat.ts`
- Frontend display: `zettelkasten-front/src/components/chat/ToolResultCard.tsx`
