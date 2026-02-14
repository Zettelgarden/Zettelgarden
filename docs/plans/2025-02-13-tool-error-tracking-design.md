# Tool Error Tracking Design

**Date**: 2025-02-13
**Status**: Approved
**Issue**: Zettelgarden-d3q

## Overview

Add structured error tracking to the `chat_tool_calls` table to enable analytics on tool failures. Currently errors are embedded in the `tool_result` JSONB field, making queries difficult.

## Problem

The current system stores tool execution results in a `tool_result` JSONB column. Errors are embedded within this JSON, making it difficult to:
- Query which tools fail most often
- Analyze common error types
- Track failure trends over time

## Solution

### Schema Changes

Add three columns to `chat_tool_calls`:

```sql
ALTER TABLE chat_tool_calls
  ADD COLUMN success BOOLEAN DEFAULT true,
  ADD COLUMN error_type TEXT,
  ADD COLUMN error_message TEXT;

CREATE INDEX idx_chat_tool_calls_success ON chat_tool_calls(success);
CREATE INDEX idx_chat_tool_calls_error_type ON chat_tool_calls(error_type) WHERE success = false;
```

**Error types** (documented, not enforced):
- `timeout` - Tool execution exceeded time limit
- `not_found` - Requested resource doesn't exist
- `permission_denied` - User lacks permission
- `invalid_input` - Invalid parameters
- `rate_limit` - Rate limit exceeded
- `internal_error` - Unexpected system error
- `unknown` - Unclassified error

### Code Changes

**File**: `services/registry.go`

Update `logToolExecution()` to:
1. Check `execErr` parameter for errors
2. Check result map for `error` field
3. Classify error type using new helper function
4. Insert success/error columns

Add `classifyErrorType()` helper function to categorize errors by pattern matching.

### Analytics Queries

**Tool failure rates**:
```sql
SELECT tool_name,
       COUNT(*) as total_calls,
       SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failures,
       ROUND(100.0 * SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) / COUNT(*), 2) as failure_rate
FROM chat_tool_calls
GROUP BY tool_name
ORDER BY failure_rate DESC;
```

**Error type breakdown**:
```sql
SELECT tool_name, error_type, COUNT(*) as count
FROM chat_tool_calls
WHERE NOT success
GROUP BY tool_name, error_type
ORDER BY tool_name, count DESC;
```

**Trends over time**:
```sql
SELECT tool_name,
       DATE_TRUNC('day', created_at) as date,
       COUNT(*) as total_calls,
       SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failures
FROM chat_tool_calls
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY tool_name, DATE_TRUNC('day', created_at)
ORDER BY date DESC, tool_name;
```

## Testing

1. Unit test `classifyErrorType()` for all error patterns
2. Unit test `logToolExecution()` for success and failure cases
3. Migration test to verify columns and defaults
4. Integration test to verify database state after tool call

## Migration Strategy

**File**: `schema/0130-add-tool-error-tracking.sql`

Existing rows will get `success=true` (default) since we can't retroactively determine success/failure from JSONB. This is acceptable as the analytics will improve going forward.

## Success Criteria

- Can query tool failure rates by percentage
- Can query error types per tool
- Can track trends over time
- Existing functionality unaffected
