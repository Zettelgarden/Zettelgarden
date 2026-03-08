# Design: GET /api/habits/{id}/logs Endpoint

**Date:** 2026-03-08
**Status:** Approved
**Issue:** Zettelgarden-orz

## Overview

Add a REST endpoint to retrieve paginated habit check-in history. The service layer already has `GetHabitLogs` function implemented, so this primarily involves exposing it via HTTP.

## API Specification

**Endpoint:** `GET /api/habits/{id}/logs`

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | int | Habit ID |

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 50 | Number of logs to return (max 100) |
| `offset` | int | 0 | Number of logs to skip for pagination |

**Response:**
```json
{
  "logs": [
    {
      "id": 1,
      "habit_id": 5,
      "user_id": 2,
      "completed_at": "2026-03-08T14:30:00Z",
      "notes": "Felt great today!",
      "created_at": "2026-03-08T14:30:00Z"
    }
  ],
  "total": 42
}
```

## Implementation

### Files to Modify

1. `go-backend/handlers/habits.go` - Add `GetHabitLogsRoute` handler
2. `go-backend/routes/habits.go` - Register the new route

### Handler Logic

1. Extract habit ID from URL path
2. Extract optional `limit` and `offset` query params with defaults
3. Validate limit (max 100)
4. Call `services.GetHabitLogs(db, userID, habitID, limit, offset)`
5. Return JSON response with `logs` array and `total` count

### Error Handling

| Status | Condition |
|--------|-----------|
| 400 | Invalid habit ID format, invalid limit/offset |
| 404 | Habit not found |
| 500 | Internal server error |

## Security

- Protected route (requires valid JWT token)
- User can only access logs for their own habits (enforced in service layer)
