# Admin UI for Scheduled Job Runner - Design Document

**Date:** 2025-01-31
**Status:** Approved
**Related Issue:** Zettelgarden-yboc.12

## Overview

Build an admin interface for the Scheduled Job Runner feature to manage and monitor scheduled jobs. The UI will display all registered scheduled jobs with their execution status, history, and statistics.

## Design Approach

- **Pattern:** Expandable row table (not separate pages or modals)
- **Refresh:** Manual only (no auto-refresh)
- **Pagination:** "Load more" button for history within expandable rows
- **Status Display:** Last run status + next scheduled run + summary statistics

## Backend API Changes

### 1. Enhanced List Jobs Endpoint

**Endpoint:** `GET /api/admin/scheduler/jobs`

**Response:**
```json
{
  "jobs": [
    {
      "name": "daily-cleanup",
      "schedule": "0 2 * * *",
      "next_run": "2025-01-31T02:00:00Z"
    }
  ]
}
```

### 2. New Job Summary Endpoint

**Endpoint:** `GET /api/admin/scheduler/jobs/{jobName}/summary`

**Response:**
```json
{
  "job_name": "daily-cleanup",
  "last_run_status": "completed",
  "last_run_at": "2025-01-30T02:00:00Z",
  "recent_stats": {
    "total_runs": 7,
    "success_count": 6,
    "failure_count": 1,
    "success_rate": 85.7
  }
}
```

Status values: `completed`, `failed`, `running`, `never`

### 3. Enhanced History Endpoint

**Endpoint:** `GET /api/admin/scheduler/jobs/{jobName}/history?limit=50&offset=0`

**Response:**
```json
{
  "runs": [...],
  "total": 150,
  "offset": 0,
  "limit": 50,
  "has_more": true
}
```

## Frontend Types

### ScheduledJobInfo
```typescript
interface ScheduledJobInfo {
  name: string;
  schedule: string;
  next_run: string;
  last_run_status?: string;
  last_run_at?: string;
}
```

### JobSummary
```typescript
interface JobSummary {
  job_name: string;
  last_run_status: string;
  last_run_at?: string;
  recent_stats: {
    total_runs: number;
    success_count: number;
    failure_count: number;
    success_rate: number;
  };
}
```

### JobRun
```typescript
interface JobRun {
  id: number;
  job_name: string;
  started_at: string;
  completed_at?: string;
  status: string;
  error_message?: string;
  retry_count: number;
}
```

## Component Structure

```
AdminSchedulerPage (new)
├── SchedulerHeader
├── SchedulerTable
│   └── SchedulerTableRow
│       ├── JobStatusBadge
│       ├── ScheduleDisplay
│       ├── RecentStatsSummary
│       └── ExpandableHistory
│           ├── ExecutionHistoryTable
│           └── LoadMoreButton
```

## New Files

### Frontend
- `zettelkasten-front/src/pages/admin/AdminSchedulerPage.tsx`
- `zettelkasten-front/src/components/scheduler/JobStatusBadge.tsx`
- `zettelkasten-front/src/components/scheduler/ScheduleDisplay.tsx`
- `zettelkasten-front/src/components/scheduler/RecentStatsSummary.tsx`
- `zettelkasten-front/src/components/scheduler/ExecutionHistoryTable.tsx`
- `zettelkasten-front/src/components/scheduler/ExpandableHistory.tsx`

### Backend
- `go-backend/handlers/scheduler.go` (modify existing)

## Status Badge Colors

- **Green:** Completed (success)
- **Red:** Failed
- **Yellow:** Running (with pulsing dot animation)
- **Gray:** Never run / Unknown

## Summary Stats Color Coding

- **Green:** Success rate > 95%
- **Yellow:** Success rate 70-95%
- **Red:** Success rate < 70%

## Utility Functions

### formatCronSchedule(cron: string): string
Converts cron expressions to human-readable format.
Examples:
- `"0 2 * * *"` → `"Daily at 2:00 AM"`
- `"0 * * * *"` → `"Hourly"`

### formatRelativeTime(dateStr: string): string
Returns relative time string.
Examples:
- `"2m ago"`, `"3h ago"`, `"5d ago"`

## Error Handling

- **Initial load failure:** Show full-page error with retry
- **Per-job summary failure:** Non-blocking, show "Stats unavailable"
- **History fetch failure:** Error within expandable row with retry
- **Health endpoint failure:** Show "Status unknown" badge

## Empty States

- **No jobs:** "No scheduled jobs configured"
- **No history:** "This job has never run"
- **History fully loaded:** Disable "Load more", show "All history loaded"

## Routing

Add to `Admin.tsx` sidebar:
```tsx
<Link to="/admin/scheduler">⏰ Scheduled Jobs</Link>
```

Route: `/admin/scheduler`

## Implementation Order

1. Backend API changes (handlers, tests)
2. Frontend API layer (types, functions)
3. Reusable components (StatusBadge, ScheduleDisplay, etc.)
4. Main page component (AdminSchedulerPage)
5. Routing integration
6. Tests and documentation
