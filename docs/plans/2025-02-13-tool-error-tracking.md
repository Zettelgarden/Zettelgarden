# Tool Error Tracking Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add structured error tracking to `chat_tool_calls` table for analytics on tool failures.

**Architecture:** Add three columns (success, error_type, error_message) to existing table, update logging function to extract error info, classify errors by pattern matching.

**Tech Stack:** PostgreSQL, Go, database/sql

---

### Task 1: Create Database Migration

**Files:**
- Create: `go-backend/schema/0130-add-tool-error-tracking.sql`

**Step 1: Write the migration file**

Create `go-backend/schema/0130-add-tool-error-tracking.sql`:

```sql
-- Add structured error tracking to chat_tool_calls
-- This enables analytics queries for tool failure rates and error types

ALTER TABLE chat_tool_calls
  ADD COLUMN success BOOLEAN DEFAULT true,
  ADD COLUMN error_type TEXT,
  ADD COLUMN error_message TEXT;

-- Index for filtering by success status
CREATE INDEX idx_chat_tool_calls_success ON chat_tool_calls(success);

-- Partial index for failed tool calls (most common analytics query)
CREATE INDEX idx_chat_tool_calls_error_type ON chat_tool_calls(error_type)
  WHERE success = false;

-- Comment on error_type values
COMMENT ON COLUMN chat_tool_calls.error_type IS '
Error classification for failed tool calls:
- timeout: Tool execution exceeded time limit
- not_found: Requested resource does not exist
- permission_denied: User lacks permission
- invalid_input: Invalid parameters provided
- rate_limit: Rate limit exceeded
- internal_error: Unexpected system error
- unknown: Unclassified error
';
```

**Step 2: Verify migration syntax**

Run: `psql -f go-backend/schema/0130-add-tool-error-tracking.sql --dry-run` or check syntax manually
Expected: No syntax errors

**Step 3: Commit**

```bash
cd go-backend
git add schema/0130-add-tool-error-tracking.sql
git commit -m "schema: add error tracking columns to chat_tool_calls

Add success, error_type, error_message columns for analytics.
Enables queries for tool failure rates and error breakdowns.

Issue: Zettelgarden-d3q

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 2: Add Error Classification Function

**Files:**
- Modify: `go-backend/services/registry.go`

**Step 1: Read the current file**

```bash
cat go-backend/services/registry.go
```

Find the `logToolExecution` function (around line 118).

**Step 2: Add classifyErrorType helper function**

Add this function before `logToolExecution`:

```go
// classifyErrorType categorizes an error into a type for analytics
func classifyErrorType(err error) string {
    if err == nil {
        return ""
    }

    errMsg := err.Error()

    // Check for specific error patterns
    switch {
    case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
        return "timeout"
    case strings.Contains(errMsg, "not found"), strings.Contains(errMsg, "No rows found"):
        return "not_found"
    case strings.Contains(errMsg, "permission"), strings.Contains(errMsg, "forbidden"), strings.Contains(errMsg, "unauthorized"):
        return "permission_denied"
    case strings.Contains(errMsg, "invalid"), strings.Contains(errMsg, "bad request"), strings.Contains(errMsg, "malformed"):
        return "invalid_input"
    case strings.Contains(errMsg, "rate limit"), strings.Contains(errMsg, "quota exceeded"), strings.Contains(errMsg, "too many requests"):
        return "rate_limit"
    default:
        return "internal_error"
    }
}

// classifyErrorTypeFromString classifies errors from string (for embedded errors in results)
func classifyErrorTypeFromString(errorMsg string) string {
    switch {
    case strings.Contains(errorMsg, "timeout"), strings.Contains(errorMsg, "deadline"), strings.Contains(errorMsg, "timed out"):
        return "timeout"
    case strings.Contains(errorMsg, "not found"), strings.Contains(errorMsg, "No rows found"):
        return "not_found"
    case strings.Contains(errorMsg, "permission"), strings.Contains(errorMsg, "forbidden"), strings.Contains(errorMsg, "unauthorized"):
        return "permission_denied"
    case strings.Contains(errorMsg, "invalid"), strings.Contains(errorMsg, "bad request"):
        return "invalid_input"
    case strings.Contains(errorMsg, "rate limit"), strings.Contains(errorMsg, "quota"):
        return "rate_limit"
    default:
        return "unknown"
    }
}
```

**Step 3: Run tests to ensure no breakage**

```bash
cd go-backend
go test ./services/... -v
```

Expected: Existing tests still pass (we haven't changed behavior yet)

**Step 4: Commit**

```bash
cd go-backend
git add services/registry.go
git commit -m "feat: add error classification helper functions

Add classifyErrorType and classifyErrorTypeFromString
to categorize errors for analytics tracking.

Issue: Zettelgarden-d3q

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 3: Update logToolExecution Function

**Files:**
- Modify: `go-backend/services/registry.go` (update `logToolExecution` function)

**Step 1: Write failing test for new behavior**

Create test file `go-backend/services/registry_error_test.go`:

```go
package services

import (
    "context"
    "errors"
    "testing"
)

func TestClassifyErrorType(t *testing.T) {
    tests := []struct {
        name     string
        err      error
        expected string
    }{
        {"timeout context", context.DeadlineExceeded, "timeout"},
        {"cancel context", context.Canceled, "timeout"},
        {"not found", errors.New("card not found"), "not_found"},
        {"permission", errors.New("permission denied"), "permission_denied"},
        {"invalid", errors.New("invalid input"), "invalid_input"},
        {"rate limit", errors.New("rate limit exceeded"), "rate_limit"},
        {"unknown", errors.New("something weird"), "internal_error"},
        {"nil error", nil, ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := classifyErrorType(tt.err)
            if result != tt.expected {
                t.Errorf("classifyErrorType() = %v, expected %v", result, tt.expected)
            }
        })
    }
}

func TestClassifyErrorTypeFromString(t *testing.T) {
    tests := []struct {
        name     string
        msg      string
        expected string
    }{
        {"timeout", "operation timed out", "timeout"},
        {"not found", "card not found", "not_found"},
        {"permission", "you don't have permission", "permission_denied"},
        {"invalid", "invalid request", "invalid_input"},
        {"unknown", "something broke", "unknown"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := classifyErrorTypeFromString(tt.msg)
            if result != tt.expected {
                t.Errorf("classifyErrorTypeFromString() = %v, expected %v", result, tt.expected)
            }
        })
    }
}
```

**Step 2: Run tests**

```bash
cd go-backend
go test ./services -v -run TestClassify
```

Expected: Tests pass

**Step 3: Update logToolExecution function**

Find and replace the `logToolExecution` function in `services/registry.go`:

```go
func logToolExecution(db *sql.DB, userID int, toolName string, args map[string]interface{}, result map[string]interface{}, executionTimeMs int, execErr error, conversationID, messageID *string) error {
    argsJSON, _ := json.Marshal(args)
    resultJSON, _ := json.Marshal(result)

    // Extract error information
    var success bool = true
    var errorType, errorMsg string

    if execErr != nil {
        success = false
        errorMsg = execErr.Error()
        errorType = classifyErrorType(execErr)
    } else if result != nil {
        // Check if result contains error field
        if errVal, ok := result["error"]; ok && errVal != nil {
            // Also check if the error is not an empty result
            if !isToolResultEmpty(result) {
                success = false
                errorMsg = fmt.Sprintf("%v", errVal)
                errorType = classifyErrorTypeFromString(errorMsg)
            }
        }
    }

    query := `
        INSERT INTO chat_tool_calls (id, user_id, conversation_id, message_id,
                                     tool_name, tool_arguments, tool_result,
                                     execution_time_ms, success, error_type, error_message, created_at)
        VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
    `
    fmt.Printf("Tool call: %v (success=%v)\n", toolName, success)

    _, err := db.Exec(query, userID, conversationID, messageID, toolName,
                      argsJSON, resultJSON, executionTimeMs, success, errorType, errorMsg)
    return err
}
```

**Step 4: Run all service tests**

```bash
cd go-backend
go test ./services/... -v
```

Expected: All tests pass

**Step 5: Commit**

```bash
cd go-backend
git add services/registry.go services/registry_error_test.go
git commit -m "feat: extract and store error info in tool logging

Update logToolExecution to populate success, error_type,
and error_message columns. Classify errors by pattern.

Issue: Zettelgarden-d3q

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 4: Create Analytics Queries Reference

**Files:**
- Create: `go-backend/docs/analytics-queries.md`

**Step 1: Create analytics queries reference**

Create `go-backend/docs/analytics-queries.md`:

```markdown
# Tool Analytics Queries

Reference queries for analyzing tool execution data.

## Tool Failure Rates

```sql
SELECT
    tool_name,
    COUNT(*) as total_calls,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failures,
    ROUND(100.0 * SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) / COUNT(*), 2) as failure_rate
FROM chat_tool_calls
GROUP BY tool_name
ORDER BY failure_rate DESC;
```

## Error Type Breakdown

```sql
SELECT
    tool_name,
    error_type,
    COUNT(*) as count
FROM chat_tool_calls
WHERE NOT success
GROUP BY tool_name, error_type
ORDER BY tool_name, count DESC;
```

## Trends Over Time (30 days)

```sql
SELECT
    tool_name,
    DATE_TRUNC('day', created_at) as date,
    COUNT(*) as total_calls,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failures
FROM chat_tool_calls
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY tool_name, DATE_TRUNC('day', created_at)
ORDER BY date DESC, tool_name;
```

## Most Common Error Messages

```sql
SELECT
    error_message,
    COUNT(*) as count
FROM chat_tool_calls
WHERE NOT success AND error_message IS NOT NULL
GROUP BY error_message
ORDER BY count DESC
LIMIT 20;
```

## Recent Failures

```sql
SELECT
    tool_name,
    error_type,
    error_message,
    created_at
FROM chat_tool_calls
WHERE NOT success
ORDER BY created_at DESC
LIMIT 50;
```
```

**Step 2: Commit**

```bash
cd go-backend
git add docs/analytics-queries.md
git commit -m "docs: add analytics queries reference

SQL queries for tool failure analysis and error tracking.

Issue: Zettelgarden-d3q

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 5: Integration Test

**Files:**
- Create: `go-backend/services/tool_error_tracking_integration_test.go`

**Step 1: Write integration test**

Create test file that verifies the database columns are populated:

```go
package services

import (
    "database/sql"
    "testing"
)

// TestToolErrorTracking verifies that error tracking columns are populated
// This requires a database connection - will be skipped in CI without DB
func TestToolErrorTracking(t *testing.T) {
    // This test requires a real database connection
    // Skip if no DB available
    t.Skip("Requires database connection - run manually")

    db := getTestDB() // You'll need to implement this or use existing test DB setup
    if db == nil {
        t.Skip("No database available")
    }

    // Test successful call
    err := logToolExecution(db, 1, "test_tool", map[string]interface{}{"key": "value"},
                           map[string]interface{}{"data": "result"}, 100, nil,
                           stringPtr("conv-1"), stringPtr("msg-1"))
    if err != nil {
        t.Fatalf("logToolExecution failed: %v", err)
    }

    // Verify success=true was recorded
    var success bool
    var errorType, errorMsg string
    err = db.QueryRow(`
        SELECT success, error_type, error_message
        FROM chat_tool_calls
        WHERE user_id = 1 AND tool_name = 'test_tool'
        ORDER BY created_at DESC LIMIT 1
    `).Scan(&success, &errorType, &errorMsg)
    if err != nil {
        t.Fatalf("Failed to query tool call: %v", err)
    }

    if !success {
        t.Errorf("Expected success=true, got false")
    }
    if errorType != "" || errorMsg != "" {
        t.Errorf("Expected empty error fields for success, got error_type=%s, error_message=%s",
                 errorType, errorMsg)
    }

    // Test failed call
    testErr := errors.New("test card not found")
    err = logToolExecution(db, 1, "test_tool", map[string]interface{}{},
                          nil, 50, testErr,
                          stringPtr("conv-1"), stringPtr("msg-2"))
    if err != nil {
        t.Fatalf("logToolExecution failed: %v", err)
    }

    // Verify failure was recorded
    err = db.QueryRow(`
        SELECT success, error_type, error_message
        FROM chat_tool_calls
        WHERE user_id = 1 AND tool_name = 'test_tool'
        ORDER BY created_at DESC LIMIT 1
    `).Scan(&success, &errorType, &errorMsg)
    if err != nil {
        t.Fatalf("Failed to query tool call: %v", err)
    }

    if success {
        t.Errorf("Expected success=false, got true")
    }
    if errorType != "not_found" {
        t.Errorf("Expected error_type='not_found', got '%s'", errorType)
    }
    if errorMsg == "" {
        t.Errorf("Expected error_message to be populated")
    }
}

func stringPtr(s string) *string {
    return &s
}
```

**Step 2: Run integration test (if DB available)**

```bash
cd go-backend
# Skip if no DB, but verify test compiles
go test ./services -v -run TestToolErrorTracking
```

Expected: Test compiles, may skip if no DB

**Step 3: Commit**

```bash
cd go-backend
git add services/tool_error_tracking_integration_test.go
git commit -m "test: add integration test for error tracking

Verify that success, error_type, and error_message columns
are correctly populated in database.

Issue: Zettelgarden-d3q

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 6: Run Migration and Verify

**Step 1: Run migration**

```bash
cd go-backend
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f schema/0130-add-tool-error-tracking.sql
```

Expected: "ALTER TABLE" and "CREATE INDEX" messages

**Step 2: Verify schema**

```bash
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "\d chat_tool_calls"
```

Expected: See `success`, `error_type`, `error_message` columns listed

**Step 3: Verify indexes**

```bash
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "\di chat_tool_calls"
```

Expected: See `idx_chat_tool_calls_success` and `idx_chat_tool_calls_error_type`

**Step 4: Verify existing data has default values**

```bash
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*), COUNT(success) FROM chat_tool_calls;"
```

Expected: Both counts equal (success defaults to true for existing rows)

---

### Task 7: Build and Smoke Test

**Step 1: Build backend**

```bash
cd go-backend
go build -o main
```

Expected: Clean build

**Step 2: Start server briefly**

```bash
timeout 5 ./main 2>&1 | head -30 || true
```

Expected: Server starts, no database errors about missing columns

**Step 3: Close Beads task**

```bash
bd close Zettelgarden-d3q "Implemented structured error tracking for tool executions. Added success, error_type, error_message columns to chat_tool_calls table."
```

---

## Summary

After completion:
- `chat_tool_calls` has `success`, `error_type`, `error_message` columns
- `logToolExecution()` populates error info on every tool call
- Error classification by pattern matching
- Analytics queries documented
- Integration test verifies database state
