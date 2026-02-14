# Tool Result Format Standardization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Standardize all LLM tool result formats across the codebase to use consistent `{success, data/error, metadata}` wrapper structure.

**Architecture:** Create central wrapper functions in `services/tools/response.go` that all tool handlers use. Migrate tools incrementally by category (cards → tasks → entities → facts → others). Frontend detects and handles both formats during transition.

**Tech Stack:** Go 1.x, Go testing package, React/TypeScript frontend

---

## Task 1: Create Response Wrapper Infrastructure

**Files:**
- Create: `go-backend/services/tools/response.go`
- Create: `go-backend/services/tools/response_test.go`
- Modify: `go-backend/services/tools/types.go` (check if imports needed)

**Step 1: Create response.go file**

```go
package tools

import (
	"go-backend/models"
)

// WrapToolSuccess creates a standardized success response
func WrapToolSuccess(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
	}
}

// WrapToolSuccessWithMetadata creates a success response with metadata
func WrapToolSuccessWithMetadata(data interface{}, metadata map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"success": true,
		"data":    data,
	}
	if metadata != nil {
		result["metadata"] = metadata
	}
	return result
}

// WrapToolSuccessWithList creates a success response for list results
// Automatically includes total count in metadata
func WrapToolSuccessWithList(items []map[string]interface{}) map[string]interface{} {
	return WrapToolSuccessWithMetadata(
		map[string]interface{}{"items": items},
		map[string]interface{}{"total": len(items)},
	)
}

// WrapToolError creates a standardized error response
func WrapToolError(toolErr *models.ToolError) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error":   toolErr.ToMap()["error"],
	}
}

// Metadata helper functions

// NewMetadata creates a metadata map for common fields
func NewMetadata(pairs ...func(map[string]interface{}) map[string]interface{}) map[string]interface{} {
	metadata := make(map[string]interface{})
	for _, fn := range pairs {
		metadata = fn(metadata)
	}
	return metadata
}

// WithTotal adds/sets the total count
func WithTotal(count int) func(map[string]interface{}) map[string]interface{} {
	return func(m map[string]interface{}) map[string]interface{} {
		m["total"] = count
		return m
	}
}

// WithOperation adds/sets the operation type
func WithOperation(operation string) func(map[string]interface{}) map[string]interface{} {
	return func(m map[string]interface{}) map[string]interface{} {
		m["operation"] = operation
		return m
	}
}

// WithTool adds/sets the tool name
func WithTool(toolName string) func(map[string]interface{}) map[string]interface{} {
	return func(m map[string]interface{}) map[string]interface{} {
		m["tool"] = toolName
		return m
	}
}
```

**Step 2: Create response_test.go with unit tests**

```go
package tools

import (
	"testing"
)

func TestWrapToolSuccess(t *testing.T) {
	data := map[string]interface{}{"id": 1, "name": "test"}
	result := WrapToolSuccess(data)

	if result["success"] != true {
		t.Errorf("expected success=true, got %v", result["success"])
	}
	if result["data"] == nil {
		t.Error("expected data to be set")
	}
	if result["metadata"] != nil {
		t.Error("expected no metadata in basic success")
	}
}

func TestWrapToolSuccessWithMetadata(t *testing.T) {
	data := map[string]interface{}{"id": 1}
	metadata := NewMetadata(WithTotal(5), WithOperation("created"))
	result := WrapToolSuccessWithMetadata(data, metadata)

	if result["success"] != true {
		t.Error("expected success=true")
	}
	if result["metadata"] == nil {
		t.Error("expected metadata to be set")
	}
	meta := result["metadata"].(map[string]interface{})
	if meta["total"] != 5 {
		t.Errorf("expected total=5, got %v", meta["total"])
	}
	if meta["operation"] != "created" {
		t.Errorf("expected operation=created, got %v", meta["operation"])
	}
}

func TestWrapToolSuccessWithList(t *testing.T) {
	items := []map[string]interface{}{
		{"id": 1},
		{"id": 2},
		{"id": 3},
	}
	result := WrapToolSuccessWithList(items)

	if result["success"] != true {
		t.Error("expected success=true")
	}
	meta := result["metadata"].(map[string]interface{})
	if meta["total"] != 3 {
		t.Errorf("expected total=3, got %v", meta["total"])
	}
	data := result["data"].(map[string]interface{})
	if data["items"] == nil {
		t.Error("expected items in data")
	}
}

func TestWrapToolError(t *testing.T) {
	toolErr := &models.ToolError{
		Type:      models.ToolErrorTypeNotFound,
		Message:   "test not found",
		Retryable: false,
		ToolName:  "test_tool",
	}
	result := WrapToolError(toolErr)

	if result["success"] != false {
		t.Error("expected success=false")
	}
	if result["error"] == nil {
		t.Error("expected error to be set")
	}
}

func TestMetadataHelpers(t *testing.T) {
	metadata := NewMetadata(
		WithTotal(10),
		WithOperation("searched"),
		WithTool("search_cards"),
	)

	if metadata["total"] != 10 {
		t.Errorf("expected total=10, got %v", metadata["total"])
	}
	if metadata["operation"] != "searched" {
		t.Errorf("expected operation=searched, got %v", metadata["operation"])
	}
	if metadata["tool"] != "search_cards" {
		t.Errorf("expected tool=search_cards, got %v", metadata["tool"])
	}
}

func TestWrapToolSuccessWithEmptyData(t *testing.T) {
	result := WrapToolSuccess(nil)

	if result["success"] != true {
		t.Error("expected success=true even with nil data")
	}
}

func TestWrapToolSuccessWithNilMetadata(t *testing.T) {
	data := map[string]interface{}{"id": 1}
	result := WrapToolSuccessWithMetadata(data, nil)

	if result["metadata"] != nil {
		t.Error("expected no metadata when nil passed")
	}
}
```

**Step 3: Run tests to verify they pass**

Run: `cd go-backend && go test ./services/tools/... -v`
Expected: PASS for all tests

**Step 4: Commit**

```bash
git add go-backend/services/tools/response.go go-backend/services/tools/response_test.go
git commit -m "feat: add standardized tool result wrapper functions

- Add WrapToolSuccess, WrapToolSuccessWithMetadata, WrapToolSuccessWithList
- Add WrapToolError for error responses
- Add metadata helper functions: NewMetadata, WithTotal, WithOperation, WithTool
- Add comprehensive unit tests

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 2: Update WrapToolError Usage in Tool Execution

**Files:**
- Modify: `go-backend/services/tool_retry.go`
- Modify: `go-backend/handlers/chat_agent/tools_execution.go`

**Step 1: Check existing WrapToolError function**

First, find where WrapToolError currently exists:

Run: `grep -r "func WrapToolError" go-backend/services/`

Note: This function likely exists in tool_error.go or similar. We need to check if it's compatible with our new wrapper.

**Step 2: Update tool_retry.go if needed**

If WrapToolError exists in tool_retry.go, ensure it returns the new format:

```go
// In services/tool_retry.go
func WrapToolError(toolName string, args map[string]interface{}, err error) map[string]interface{} {
	toolErr := ClassifyToolError(toolName, args, err)
	return tools.WrapToolError(toolErr)  // Use new wrapper
}
```

If the function signature doesn't match, you may need to create a bridge function or update call sites.

**Step 3: Verify tools_execution.go uses the wrapper correctly**

The executeToolCall function should continue using WrapToolError - verify it gets the new format.

**Step 4: Run backend tests**

Run: `cd go-backend && go test ./... -run TestChatService`
Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/tool_retry.go go-backend/handlers/chat_agent/tools_execution.go
git commit -m "refactor: update WrapToolError to use new standard format

- Ensure tool execution uses new standardized error wrapper
- Error responses now include success: false

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 3: Migrate Card Tools - Search Cards

**Files:**
- Modify: `go-backend/services/card_tools.go` (handleSearchCardsV2 and handleSearchCardsLegacy)

**Step 1: Write failing test for new format**

Add test in `go-backend/services/card_tools_test.go`:

```go
func TestHandleSearchCardsV2_ReturnsStandardFormat(t *testing.T) {
	// Setup test DB, user, etc.
	db := setupTestDB(t)
	defer db.Close()
	userID := createTestUser(db, t)
	ctx := &ToolContext{DB: db, UserID: userID}

	args := map[string]interface{}{
		"query": "test",
		"limit": 5,
	}

	result, err := handleSearchCardsV2(args, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check new format
	if result["success"] != true {
		t.Errorf("expected success=true in result")
	}
	if result["data"] == nil {
		t.Errorf("expected data field in result")
	}
	if result["metadata"] == nil {
		t.Errorf("expected metadata field in result")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services/... -run TestHandleSearchCardsV2_ReturnsStandardFormat -v`
Expected: FAIL - current format doesn't have success/data/metadata

**Step 3: Update handleSearchCardsV2 to use new format**

Find handleSearchCardsV2 in card_tools.go (around line 182). Update return statement:

```go
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

	var results []map[string]interface{}

	if searchType == "text" {
		results, err = card.ExecuteTextSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	} else {
		results, err = card.ExecuteSemanticSearch(ctx.DB, ctx.UserID, query, limit, ctx.TypesenseClient)
	}

	if err != nil {
		return nil, fmt.Errorf("search failed: %v", err)
	}

	// NEW: Use standardized wrapper
	return WrapToolSuccessWithMetadata(
		map[string]interface{}{
			"cards":       results,
			"query":       query,
			"search_type": searchType,
		},
		NewMetadata(WithTotal(len(results))),
	), nil
}
```

**Step 4: Update handleSearchCardsLegacy the same way**

Find handleSearchCardsLegacy (around line 391) and make identical change to return statement.

**Step 5: Run test to verify it passes**

Run: `cd go-backend && go test ./services/... -run TestHandleSearchCardsV2_ReturnsStandardFormat -v`
Expected: PASS

**Step 6: Run all card tools tests**

Run: `cd go-backend && go test ./services/card_tools_test.go -v`
Expected: All PASS (may need updates to other tests)

**Step 7: Commit**

```bash
git add go-backend/services/card_tools.go go-backend/services/card_tools_test.go
git commit -m "refactor: search_cards now returns standardized format

- Update handleSearchCardsV2 to use WrapToolSuccessWithMetadata
- Update handleSearchCardsLegacy to use WrapToolSuccessWithMetadata
- Result format: {success: true, data: {cards, query, search_type}, metadata: {total}}
- Add test for new format

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 4: Migrate Card Tools - Get Card By ID

**Files:**
- Modify: `go-backend/services/card_tools.go` (handleGetCardByIDV2 and handleGetCardByIDLegacy)

**Step 1: Write failing test**

```go
func TestHandleGetCardByIDV2_ReturnsStandardFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	userID := createTestUser(db, t)
	cardID := createTestCard(db, userID, t)
	ctx := &ToolContext{DB: db, UserID: userID}

	args := map[string]interface{}{"card_id": cardID}

	result, err := handleGetCardByIDV2(args, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["success"] != true {
		t.Errorf("expected success=true")
	}
	if result["data"] == nil {
		t.Errorf("expected data field")
	}
}
```

**Step 2: Run test - verify fails**

Run: `cd go-backend && go test ./services/... -run TestHandleGetCardByIDV2_ReturnsStandardFormat -v`
Expected: FAIL

**Step 3: Update handleGetCardByIDV2**

Find the function (around line 219) and update return:

```go
func handleGetCardByIDV2(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	cardID, err := getIntParam(args, "card_id")
	if err != nil {
		return nil, err
	}

	c, err := card.GetFullCard(ctx.DB, ctx.UserID, cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %v", err)
	}

	// NEW: Wrap in standard format
	return WrapToolSuccess(card.StructToMap(c)), nil
}
```

**Step 4: Update handleGetCardByIDLegacy**

Find the function (around line 428) and make same change.

**Step 5: Run tests**

Run: `cd go-backend && go test ./services/... -run TestHandleGetCardByIDV2_ReturnsStandardFormat -v`
Expected: PASS

**Step 6: Commit**

```bash
git add go-backend/services/card_tools.go go-backend/services/card_tools_test.go
git commit -m "refactor: get_card_by_id returns standardized format

- Update handleGetCardByIDV2 to use WrapToolSuccess
- Update handleGetCardByIDLegacy to use WrapToolSuccess
- Result format: {success: true, data: {card fields...}}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 5: Migrate Card Tools - Browse Card Hierarchy

**Files:**
- Modify: `go-backend/services/card_tools.go` (handleBrowseCardHierarchyV2 and Legacy)

**Step 1: Update return statement**

Find handleBrowseCardHierarchyV2 (around line 345). Update:

```go
// At the end of handleBrowseCardHierarchyV2
return WrapToolSuccessWithMetadata(
	map[string]interface{}{
		"cards":     results,
		"direction": direction,
		"depth":     depth,
	},
	NewMetadata(WithTotal(len(cards))),
), nil
```

**Step 2: Update legacy version**

Find handleBrowseCardHierarchyLegacy (around line 554) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/card_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/card_tools.go
git commit -m "refactor: browse_card_hierarchy returns standardized format

- Use WrapToolSuccessWithMetadata with total count
- Result format: {success: true, data: {cards, direction, depth}, metadata: {total}}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Migrate Card Tools - Create Card

**Files:**
- Modify: `go-backend/services/card_tools.go` (handleCreateCardV2 and Legacy)

**Step 1: Update return statement**

Find handleCreateCardV2 (around line 233). Update:

```go
// After creating the card
result := card.StructToMap(newCard)
return WrapToolSuccessWithMetadata(
	result,
	NewMetadata(
		WithOperation("card_created"),
		WithTool("create_card"),
	),
), nil
```

**Step 2: Update legacy version**

Find handleCreateCardLegacy (around line 442) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/card_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/card_tools.go
git commit -m "refactor: create_card returns standardized format

- Use WrapToolSuccessWithMetadata with operation
- Result format: {success: true, data: {card...}, metadata: {operation: card_created}}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 7: Migrate Card Tools - Update Card

**Files:**
- Modify: `go-backend/services/card_tools.go` (handleUpdateCardV2 and Legacy)

**Step 1: Update return statement**

Find handleUpdateCardV2 (around line 289). Update:

```go
result := card.StructToMap(updatedCard)
return WrapToolSuccessWithMetadata(
	result,
	NewMetadata(WithOperation("card_updated")),
), nil
```

**Step 2: Update legacy version**

Find handleUpdateCardLegacy (around line 498) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/card_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/card_tools.go
git commit -m "refactor: update_card returns standardized format

- Use WrapToolSuccessWithMetadata with operation
- Result format: {success: true, data: {card...}, metadata: {operation: card_updated}}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 8: Migrate Card Tools - Get Card Analysis

**Files:**
- Modify: `go-backend/services/card_tools.go` (handleGetCardAnalysisV2 and Legacy)

**Step 1: Update return statement**

Find handleGetCardAnalysisV2 (around line 270). Update:

```go
return WrapToolSuccess(
	map[string]interface{}{
		"card_pk":  cardPK,
		"analysis": analysis,
	},
), nil
```

**Step 2: Update legacy version**

Find handleGetCardAnalysisLegacy (around line 479) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/card_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/card_tools.go
git commit -m "refactor: get_card_analysis returns standardized format

- Use WrapToolSuccess for simple data response
- Result format: {success: true, data: {card_pk, analysis}}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 9: Frontend - Add Type Definitions for New Format

**Files:**
- Modify: `zettelkasten-front/src/api/chat.ts`

**Step 1: Add standard result types**

Add to src/api/chat.ts:

```typescript
// Standard tool result types
export interface ToolResultMetadata {
  total?: number;
  operation?: string;
  tool?: string;
  [key: string]: any;
}

export interface ToolResultSuccess {
  success: true;
  data: any;
  metadata?: ToolResultMetadata;
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

// Helper to detect standard format
export function isStandardToolResult(result: any): result is StandardToolResult {
  return typeof result === 'object' && result !== null && 'success' in result;
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/api/chat.ts
git commit -m "feat(frontend): add types for standardized tool results

- Add ToolResultSuccess, ToolResultError, StandardToolResult types
- Add isStandardToolResult helper function

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 10: Frontend - Update Tool Result Display Component

**Files:**
- Modify: `zettelkasten-front/src/components/chat/ToolResultCard.tsx`

**Step 1: Update to handle standard format**

Modify ToolResultCard to detect and handle both formats:

```typescript
// Add import
import { isStandardToolResult, StandardToolResult } from '../../api/chat';

// In component, update result parsing
const parsedResult = useMemo(() => {
  const raw = message.content;

  try {
    const parsed = JSON.parse(raw);

    // Check for standard format
    if (isStandardToolResult(parsed)) {
      return parsed as StandardToolResult;
    }

    // Legacy format - wrap for compatibility
    if (parsed.error) {
      return {
        success: false,
        error: parsed.error,
      } as StandardToolResult;
    }

    return {
      success: true,
      data: parsed,
    } as StandardToolResult;
  } catch {
    return null;
  }
}, [message.content]);

// Update rendering logic
const hasError = useMemo(() => {
  return parsedResult?.success === false;
}, [parsedResult]);

const displayData = useMemo(() => {
  if (!parsedResult || parsedResult.success === false) return null;
  return parsedResult.data;
}, [parsedResult]);

const metadata = useMemo(() => {
  if (!parsedResult || parsedResult.success === false) return null;
  return parsedResult.metadata;
}, [parsedResult]);
```

**Step 2: Update error display**

```typescript
// Update error display section
{hasError && (
  <div className="bg-red-50 border border-red-200 rounded-lg p-4">
    <h4 className="text-red-800 font-semibold mb-2">Tool Error</h4>
    <p className="text-red-700 text-sm">{parsedResult.error.message}</p>
    {parsedResult.error.suggestion && (
      <p className="text-red-600 text-xs mt-2">Suggestion: {parsedResult.error.suggestion}</p>
    )}
  </div>
)}
```

**Step 3: Update metadata display**

```typescript
// Add metadata display section
{metadata && Object.keys(metadata).length > 0 && (
  <div className="bg-gray-50 border border-gray-200 rounded-lg p-3 mt-2">
    <h5 className="text-gray-700 text-xs font-semibold mb-1">Metadata</h5>
    <dl className="grid grid-cols-2 gap-2 text-xs">
      {metadata.total !== undefined && (
        <>
          <dt className="text-gray-500">Total:</dt>
          <dd className="text-gray-700">{metadata.total}</dd>
        </>
      )}
      {metadata.operation && (
        <>
          <dt className="text-gray-500">Operation:</dt>
          <dd className="text-gray-700">{metadata.operation}</dd>
        </>
      )}
    </dl>
  </div>
)}
```

**Step 4: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`

**Step 5: Run dev server and test**

Run: `cd zettelkasten-front && npm run dev`
Test by creating cards, searching cards in chat interface

**Step 6: Commit**

```bash
git add zettelkasten-front/src/components/chat/ToolResultCard.tsx
git commit -m "refactor(frontent): update ToolResultCard for standard format

- Add isStandardToolResult detection
- Handle both legacy and standard formats
- Display metadata section with total, operation
- Update error display for new format

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 11: Migrate Task Tools - Get Tasks

**Files:**
- Modify: `go-backend/services/task_tools.go` (handleGetTasksV2 and Legacy)

**Step 1: Update return statement**

Find handleGetTasksV2 (around line 200). Update:

```go
return WrapToolSuccessWithMetadata(
	map[string]interface{}{
		"tasks": results,
	},
	NewMetadata(WithTotal(len(tasks))),
), nil
```

**Step 2: Update legacy version**

Find handleGetTasksLegacy (around line 434) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/task_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/task_tools.go
git commit -m "refactor: get_tasks returns standardized format

- Use WrapToolSuccessWithMetadata with total count
- Result format: {success: true, data: {tasks: [...]}, metadata: {total}}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 12: Migrate Task Tools - Create Task

**Files:**
- Modify: `go-backend/services/task_tools.go` (handleCreateTaskV2 and Legacy)

**Step 1: Update return statement**

Find handleCreateTaskV2 (around line 228). Update:

```go
return WrapToolSuccess(StructToMap(newTask)), nil
```

**Step 2: Update legacy version**

Find handleCreateTaskLegacy (around line 462) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/task_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/task_tools.go
git commit -m "refactor: create_task returns standardized format

- Use WrapToolSuccess for simple response
- Result format: {success: true, data: {task fields...}}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 13: Migrate Task Tools - Update Task

**Files:**
- Modify: `go-backend/services/task_tools.go` (handleUpdateTaskV2 and Legacy)

**Step 1: Update return statement**

Find handleUpdateTaskV2 (around line 280). Update:

```go
return WrapToolSuccess(StructToMap(updatedTask)), nil
```

**Step 2: Update legacy version**

Find handleUpdateTaskLegacy (around line 514) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/task_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/task_tools.go
git commit -m "refactor: update_task returns standardized format

- Use WrapToolSuccess for simple response

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 14: Migrate Task Tools - Get Task By ID

**Files:**
- Modify: `go-backend/services/task_tools.go` (handleGetTaskByIDV2 and Legacy)

**Step 1: Update return statement**

Find handleGetTaskByIDV2 (around line 336). Update:

```go
return WrapToolSuccess(StructToMap(task)), nil
```

**Step 2: Update legacy version**

Find handleGetTaskByIDLegacy (around line 570) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/task_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/task_tools.go
git commit -m "refactor: get_task_by_id returns standardized format

- Use WrapToolSuccess for simple response

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 15: Migrate Task Tools - Complete Task

**Files:**
- Modify: `go-backend/services/task_tools.go` (handleCompleteTaskV2 and Legacy)

**Step 1: Update return statement**

Find handleCompleteTaskV2 (around line 350). Update:

```go
return WrapToolSuccess(StructToMap(updatedTask)), nil
```

**Step 2: Update legacy version**

Find handleCompleteTaskLegacy (around line 584) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/task_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/task_tools.go
git commit -m "refactor: complete_task returns standardized format

- Use WrapToolSuccess for simple response

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 16: Migrate Task Tools - Delete Task

**Files:**
- Modify: `go-backend/services/task_tools.go` (handleDeleteTaskV2 and Legacy)

**Step 1: Update return statement**

Find handleDeleteTaskV2 (around line 376). Update:

```go
return WrapToolSuccessWithMetadata(
	map[string]interface{}{"deleted_id": taskID},
	NewMetadata(WithOperation("deleted")),
), nil
```

**Step 2: Update legacy version**

Find handleDeleteTaskLegacy (around line 610) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/task_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/task_tools.go
git commit -m "refactor: delete_task returns standardized format

- Use WrapToolSuccessWithMetadata with operation
- Result format: {success: true, data: {deleted_id}, metadata: {operation: deleted}}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 17: Migrate Task Tools - Complete And Schedule Task

**Files:**
- Modify: `go-backend/services/task_tools.go` (handleCompleteAndScheduleTaskV2 and Legacy)

**Step 1: Update return statement**

Find handleCompleteAndScheduleTaskV2 (around line 393). Update:

```go
return WrapToolSuccessWithMetadata(
	map[string]interface{}{
		"task_id":      taskID,
		"new_task_id":  newTaskID,
		"scheduled_in": fmt.Sprintf("%d days", days),
	},
	NewMetadata(WithOperation("completed_and_scheduled")),
), nil
```

**Step 2: Update legacy version**

Find handleCompleteAndScheduleTaskLegacy (around line 627) and make same change.

**Step 3: Run tests**

Run: `cd go-backend && go test ./services/task_tools_test.go -v`

**Step 4: Commit**

```bash
git add go-backend/services/task_tools.go
git commit -m "refactor: complete_and_schedule_task returns standardized format

- Use WrapToolSuccessWithMetadata with operation

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 18: Migrate Entity Tools

**Files:**
- Modify: `go-backend/services/entity_tools.go`

**Step 1: Update all entity tool handlers**

For each handler in entity_tools.go, update returns:

- `handleGetEntityByNameV2/Legacy`: `WrapToolSuccess(StructToMap(ent))`
- `handleSearchEntitiesV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"entities": entities, "query": query}, NewMetadata(WithTotal(len(entities))))`
- `handleGetCardsByEntityV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"cards": results, "entity_id": entityID}, NewMetadata(WithTotal(len(cards))))`
- `handleGetEntityByIDV2/Legacy`: `WrapToolSuccess(StructToMap(ent))`
- `handleMergeEntitiesV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"entity1_id": entity1ID, "entity2_id": entity2ID, "surviving_id": entity1ID}, NewMetadata(WithOperation("merged")))`
- `handleUpdateEntityV2/Legacy`: `WrapToolSuccess(StructToMap(updatedEntity))`
- `handleDeleteEntityV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"deleted_id": entityID}, NewMetadata(WithOperation("deleted")))`
- `handleAddEntityToCardV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"entity_id": entityID, "card_pk": cardPK}, NewMetadata(WithOperation("linked")))`
- `handleRemoveEntityFromCardV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"entity_id": entityID, "card_pk": cardPK}, NewMetadata(WithOperation("unlinked")))`
- `handleGetSimilarEntitiesV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"entities": results, "entity_id": entityID}, NewMetadata(WithTotal(len(results)), WithTool("limit", limit)))`

**Step 2: Run tests**

Run: `cd go-backend && go test ./services/entity_tools_test.go -v`

**Step 3: Commit**

```bash
git add go-backend/services/entity_tools.go
git commit -m "refactor: entity tools return standardized format

- Update all entity tool handlers to use wrapper functions
- Standardize success/error/metadata format

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 19: Migrate Fact Tools

**Files:**
- Modify: `go-backend/services/fact_tools.go`

**Step 1: Update all fact tool handlers**

- `handleSearchFactsV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"facts": facts, "query": query, "search_type": searchType}, NewMetadata(WithTotal(len(facts))))`
- `handleGetCardFactsV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"facts": results}, NewMetadata(WithTotal(len(facts))))`
- `handleGetEntityFactsV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"facts": results}, NewMetadata(WithTotal(len(facts))))`
- `handleGetFactCardsV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"cards": results}, NewMetadata(WithTotal(len(cards))))`

**Step 2: Run tests**

Run: `cd go-backend && go test ./services/fact_tools_test.go -v`

**Step 3: Commit**

```bash
git add go-backend/services/fact_tools.go
git commit -m "refactor: fact tools return standardized format

- Update all fact tool handlers to use wrapper functions

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 20: Migrate Template Tools

**Files:**
- Modify: `go-backend/services/template_tools.go`

**Step 1: Update handlers**

- `handleGetTemplateV2/Legacy`: `WrapToolSuccess(StructToMap(tmpl))`
- `handleListTemplatesV2/Legacy`: `WrapToolSuccessWithMetadata(map[string]interface{}{"templates": results}, NewMetadata(WithTotal(len(templates))))`
- `handleGetNextChildIDV2/Legacy`: For error case return error, for success: `WrapToolSuccess(map[string]interface{}{"new_id": nextID})`

**Step 2: Run tests**

Run: `cd go-backend && go test ./services/template_tools_test.go -v`

**Step 3: Commit**

```bash
git add go-backend/services/template_tools.go
git commit -m "refactor: template tools return standardized format

- Update all template tool handlers to use wrapper functions

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 21: Migrate Memory Tools

**Files:**
- Modify: `go-backend/services/memory_tools.go`

**Step 1: Update handleGetUserMemory**

```go
func handleGetUserMemory(args map[string]interface{}, ctx *ToolContext) (map[string]interface{}, error) {
	userMemory, err := memory.GetUserMemory(ctx.DB, ctx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user memory: %v", err)
	}

	if userMemory == "" {
		return WrapToolSuccessWithMetadata(
			map[string]interface{}{"memory": ""},
			NewMetadata(WithOperation("no_memory")),
		), nil
	}

	return WrapToolSuccess(map[string]interface{}{"memory": userMemory}), nil
}
```

**Step 2: Run tests**

Run: `cd go-backend && go test ./services/memory_tools_test.go -v`

**Step 3: Commit**

```bash
git add go-backend/services/memory_tools.go
git commit -m "refactor: memory tools return standardized format

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 22: Migrate Calendar Tools

**Files:**
- Modify: `go-backend/services/calendar_tools.go`

**Step 1: Check and update handlers**

Review calendar_tools.go and update all handlers to use wrapper functions. Pattern will be similar to other tools.

**Step 2: Run tests**

Run: `cd go-backend && go test ./services/calendar_tools_test.go -v`

**Step 3: Commit**

```bash
git add go-backend/services/calendar_tools.go
git commit -m "refactor: calendar tools return standardized format

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 23: Migrate Article Tools

**Files:**
- Modify: `go-backend/services/article_tools.go`

**Step 1: Update handlers**

Review article_tools.go and update all handlers.

**Step 2: Run tests**

Run: `cd go-backend && go test ./services/article_tools_test.go -v`

**Step 3: Commit**

```bash
git add go-backend/services/article_tools.go
git commit -m "refactor: article tools return standardized format

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 24: Remove Legacy Format Handling from Frontend

**Files:**
- Modify: `zettelkasten-front/src/components/chat/ToolResultCard.tsx`

**Step 1: Remove legacy format handling**

Simplify ToolResultCard to only handle standard format. Remove the isStandardToolResult check and assume standard format.

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`

**Step 3: Test in dev**

Run: `cd zettelkasten-front && npm run dev`

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/chat/ToolResultCard.tsx
git commit -m "refactor(frontend): remove legacy format handling

- All tools now return standard format
- Simplify ToolResultCard component

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 25: Update Documentation

**Files:**
- Create: `docs/tool-response-format.md`

**Step 1: Create documentation**

```markdown
# Tool Response Format

All LLM tools return responses in a standardized format.

## Success Response

\`\`\`json
{
  "success": true,
  "data": {
    // Tool-specific response data
  },
  "metadata": {
    "total": 5,
    "operation": "created",
    "tool": "create_card"
  }
}
\`\`\`

## Error Response

\`\`\`json
{
  "success": false,
  "error": {
    "type": "not_found",
    "message": "Card not found",
    "retryable": false,
    "tool_name": "get_card_by_id",
    "suggestion": "..."
  }
}
\`\`\`

## Helper Functions

Backend tools use these helper functions from `services/tools/response.go`:

- `WrapToolSuccess(data)` - Simple success response
- `WrapToolSuccessWithMetadata(data, metadata)` - Success with metadata
- `WrapToolSuccessWithList(items)` - Success with list and total
- `WrapToolError(toolErr)` - Standardized error response
- `NewMetadata(...)`, `WithTotal(n)`, `WithOperation(s)`, `WithTool(s)` - Metadata builders
```

**Step 2: Commit**

```bash
git add docs/tool-response-format.md
git commit -m "docs: add tool response format documentation

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Task 26: Final Testing and Verification

**Step 1: Run full backend test suite**

Run: `cd go-backend && go test ./... -v`

**Step 2: Run frontend tests**

Run: `cd zettelkasten-front && npm test`

**Step 3: Manual testing**

1. Start backend: `cd go-backend && go run main.go`
2. Start frontend: `cd zettelkasten-front && npm run dev`
3. Test in chat interface:
   - Create a card
   - Search for cards
   - Create a task
   - Get tasks
   - Trigger an error (e.g., get non-existent card)
   - Verify all results display correctly

**Step 4: Update Beads issue**

Mark issue as complete in `.beads/issues.jsonl`:

```json
{"id": "Zettelgarden-npfu", "status": "completed"}
```

**Step 5: Final commit**

```bash
git add .beads/issues.jsonl
git commit -m "complete: tool result standardization

- All tools now return standardized {success, data/error, metadata} format
- Frontend updated to handle standard format
- Documentation added

Closes: Zettelgarden-npfu
Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Testing Notes

### Backend Testing

- Each tool file has a `_test.go` file - run `go test ./services/{tool}_test.go`
- For specific tests: `go test ./services/... -run TestFunctionName -v`
- Integration tests: `go test ./handlers/chat_agent/... -v`

### Frontend Testing

- Type checking: `npm run type-check`
- Unit tests: `npm test`
- Manual testing in browser: `npm run dev`

### Error Testing

To test error responses, try:
- `get_card_by_id` with non-existent ID
- `update_card` with mismatched card_id
- `delete_entity` with non-existent entity

All should return `{success: false, error: {...}}` format.
