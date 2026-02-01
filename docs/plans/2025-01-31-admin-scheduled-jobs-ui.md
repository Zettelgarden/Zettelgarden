# Admin UI for Scheduled Job Runner Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an admin interface for monitoring and managing cron-based scheduled jobs with expandable execution history.

**Architecture:** Frontend React/TypeScript page with table + expandable rows; Go backend API handlers for job list, summary, and history; follows existing admin UI patterns in AdminJobQueuePage.

**Tech Stack:** React, TypeScript, Go, Gorilla Mux, PostgreSQL, cron library (robfig/cron/v3)

---

## Task 1: Backend - Enhance ScheduledJob Interface

**Files:**
- Modify: `go-backend/services/scheduler.go`

**Step 1: Add NextRun method to ScheduledJob interface**

Add this method to the `ScheduledJob` interface in `go-backend/services/scheduler.go`:

```go
// ScheduledJob represents a job that can be scheduled with cron
type ScheduledJob interface {
	Name() string
	Schedule() string
	Handler(ctx context.Context) error
	MaxRetries() int
	NextRun(time.Time) time.Time
}
```

**Step 2: Update existing job implementations**

Modify `go-backend/services/jobs/cleanup_job.go` to add the `NextRun` method:

```go
import (
	"time"
	"github.com/robfig/cron/v3"
)

// NextRun returns the next scheduled run time for this job
func (j *CleanupJob) NextRun(from time.Time) time.Time {
	schedule, err := cron.ParseStandard(j.schedule)
	if err != nil {
		return time.Time{}
	}
	return schedule.Next(from)
}
```

**Step 3: Run tests to verify no regression**

Run: `cd go-backend && go test ./services/jobs/... -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add go-backend/services/scheduler.go go-backend/services/jobs/cleanup_job.go
git commit -m "feat(scheduler): add NextRun method to ScheduledJob interface"
```

---

## Task 2: Backend - Add ScheduledJobInfo Response Type

**Files:**
- Modify: `go-backend/handlers/scheduler.go`

**Step 1: Add new response types**

Add to `go-backend/handlers/scheduler.go` after the existing types:

```go
// ScheduledJobInfo represents a scheduled job with its configuration
type ScheduledJobInfo struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	NextRun  string `json:"next_run"`
}

// ScheduledJobsResponse is the response for listing scheduled jobs
type ScheduledJobsResponse struct {
	Jobs []ScheduledJobInfo `json:"jobs"`
}
```

**Step 2: Run tests to verify compilation**

Run: `cd go-backend && go build ./handlers/...`
Expected: Successful build, no errors

**Step 3: Commit**

```bash
git add go-backend/handlers/scheduler.go
git commit -m "feat(scheduler): add ScheduledJobInfo response type"
```

---

## Task 3: Backend - Modify ListScheduledJobs Handler

**Files:**
- Modify: `go-backend/handlers/scheduler.go`

**Step 1: Update ListScheduledJobs to include schedule and next run**

Replace the existing `ListScheduledJobs` function with:

```go
// ListScheduledJobs returns a handler that lists all registered scheduled jobs with their schedules
func ListScheduledJobs(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobNames := scheduler.ListJobs()

		now := time.Now()
		jobs := make([]ScheduledJobInfo, 0, len(jobNames))

		for _, name := range jobNames {
			// Get job info - we need access to the actual scheduler to get schedule/next run
			// This will be passed via a new interface method or we'll enhance SchedulerAPI
			jobInfo := ScheduledJobInfo{
				Name:     name,
				Schedule: "",  // Will be filled by enhanced scheduler
				NextRun:  "",  // Will be filled by enhanced scheduler
			}
			jobs = append(jobs, jobInfo)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := ScheduledJobsResponse{Jobs: jobs}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
```

**Step 2: Enhance SchedulerAPI interface**

Update the `SchedulerAPI` interface in `go-backend/handlers/scheduler.go`:

```go
// SchedulerAPI interface for testability
type SchedulerAPI interface {
	ListJobs() []string
	GetJobHistory(ctx context.Context, jobName string, limit int) ([]services.JobRun, error)
	GetJobInfo(name string) (schedule string, nextRun time.Time, err error)
}
```

**Step 3: Implement GetJobInfo in Scheduler**

Add to `go-backend/services/scheduler.go`:

```go
import "time"

// GetJobInfo returns the schedule and next run time for a job
func (s *Scheduler) GetJobInfo(name string) (schedule string, nextRun time.Time, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[name]
	if !exists {
		return "", time.Time{}, fmt.Errorf("job '%s' not found", name)
	}

	return job.Schedule(), job.NextRun(time.Now()), nil
}
```

**Step 4: Update handler to use GetJobInfo**

Fix the ListScheduledJobs implementation:

```go
// ListScheduledJobs returns a handler that lists all registered scheduled jobs with their schedules
func ListScheduledJobs(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobNames := scheduler.ListJobs()

		jobs := make([]ScheduledJobInfo, 0, len(jobNames))

		for _, name := range jobNames {
			schedule, nextRun, err := scheduler.GetJobInfo(name)
			if err != nil {
				// Log but continue - skip jobs with errors
				log.Printf("Error getting info for job '%s': %v", name, err)
				continue
			}

			var nextRunStr string
			if !nextRun.IsZero() {
				nextRunStr = nextRun.Format(time.RFC3339)
			}

			jobs = append(jobs, ScheduledJobInfo{
				Name:     name,
				Schedule: schedule,
				NextRun:  nextRunStr,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := ScheduledJobsResponse{Jobs: jobs}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
```

**Step 5: Run tests**

Run: `cd go-backend && go test ./handlers/... -v -run TestListScheduledJobs`
Expected: Tests pass

**Step 6: Commit**

```bash
git add go-backend/handlers/scheduler.go go-backend/services/scheduler.go
git commit -m "feat(scheduler): enhance ListScheduledJobs to include schedule and next run"
```

---

## Task 4: Backend - Add Job Summary Handler

**Files:**
- Modify: `go-backend/handlers/scheduler.go`

**Step 1: Add JobSummary response type**

Add to `go-backend/handlers/scheduler.go`:

```go
// JobSummary represents summary statistics for a scheduled job
type JobSummary struct {
	JobName       string         `json:"job_name"`
	LastRunStatus string         `json:"last_run_status"`
	LastRunAt     *string        `json:"last_run_at,omitempty"`
	RecentStats   JobStats       `json:"recent_stats"`
}

// JobStats represents statistics for job runs
type JobStats struct {
	TotalRuns    int     `json:"total_runs"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	SuccessRate  float64 `json:"success_rate"`
}
```

**Step 2: Add GetJobSummary method to SchedulerAPI interface**

```go
type SchedulerAPI interface {
	ListJobs() []string
	GetJobHistory(ctx context.Context, jobName string, limit int) ([]services.JobRun, error)
	GetJobInfo(name string) (schedule string, nextRun time.Time, err error)
	GetJobSummary(ctx context.Context, jobName string) (JobSummary, error)
}
```

**Step 3: Implement GetJobSummary in Scheduler**

Add to `go-backend/services/scheduler.go`:

```go
import (
	"context"
	"time"
)

// JobSummary represents summary statistics for a job
type JobSummary struct {
	JobName       string
	LastRunStatus string
	LastRunAt     *time.Time
	RecentStats   JobStats
}

// JobStats represents statistics for job runs
type JobStats struct {
	TotalRuns    int
	SuccessCount int
	FailureCount int
	SuccessRate  float64
}

// GetJobSummary returns summary statistics for a job
func (s *Scheduler) GetJobSummary(ctx context.Context, jobName string) (JobSummary, error) {
	if s.tracker == nil {
		return JobSummary{}, fmt.Errorf("job tracking is not enabled")
	}

	// Get recent runs (last 7 days)
	runs, err := s.tracker.GetRecentRuns(ctx, jobName, 1000)
	if err != nil {
		return JobSummary{}, err
	}

	summary := JobSummary{
		JobName: jobName,
		RecentStats: JobStats{
			TotalRuns:    len(runs),
			SuccessCount: 0,
			FailureCount: 0,
			SuccessRate:  0,
		},
		LastRunStatus: "never",
	}

	if len(runs) == 0 {
		return summary, nil
	}

	// Calculate stats from last 7 days
	cutoff := time.Now().AddDate(0, 0, -7)
	for _, run := range runs {
		if run.StartedAt.After(cutoff) {
			summary.RecentStats.TotalRuns++
			if run.Status == "completed" {
				summary.RecentStats.SuccessCount++
			} else if run.Status == "failed" {
				summary.RecentStats.FailureCount++
			}
		}
	}

	// Calculate success rate
	if summary.RecentStats.TotalRuns > 0 {
		summary.RecentStats.SuccessRate = float64(summary.RecentStats.SuccessCount) / float64(summary.RecentStats.TotalRuns) * 100
	}

	// Get last run info
	lastRun := runs[0]
	summary.LastRunStatus = lastRun.Status
	summary.LastRunAt = &lastRun.StartedAt

	return summary, nil
}
```

**Step 4: Add HTTP handler**

Add to `go-backend/handlers/scheduler.go`:

```go
import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"go-backend/services"
)

// GetJobSummaryHandler returns a handler that gets summary statistics for a job
func GetJobSummaryHandler(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		jobName := vars["jobName"]

		summary, err := scheduler.GetJobSummary(r.Context(), jobName)
		if err != nil {
			http.Error(w, "Failed to get job summary", http.StatusInternalServerError)
			return
		}

		// Convert to response DTO
		response := handlers.JobSummary{
			JobName:       summary.JobName,
			LastRunStatus: summary.LastRunStatus,
			RecentStats: handlers.JobStats{
				TotalRuns:    summary.RecentStats.TotalRuns,
				SuccessCount: summary.RecentStats.SuccessCount,
				FailureCount: summary.RecentStats.FailureCount,
				SuccessRate:  summary.RecentStats.SuccessRate,
			},
		}

		if summary.LastRunAt != nil {
			lastRunAt := summary.LastRunAt.Format("2006-01-02T15:04:05Z07:00")
			response.LastRunAt = &lastRunAt
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
```

**Step 5: Run tests**

Run: `cd go-backend && go test ./handlers/... -v`
Expected: Tests pass

**Step 6: Commit**

```bash
git add go-backend/handlers/scheduler.go go-backend/services/scheduler.go
git commit -m "feat(scheduler): add job summary endpoint and handler"
```

---

## Task 5: Backend - Add Offset Support to Job History

**Files:**
- Modify: `go-backend/handlers/scheduler.go`
- Modify: `go-backend/services/scheduler.go`

**Step 1: Update GetJobHistory in Scheduler to support offset**

Modify `go-backend/services/scheduler.go`:

```go
// GetJobHistory returns execution history for a specific job with pagination
func (s *Scheduler) GetJobHistory(ctx context.Context, jobName string, limit int, offset int) ([]JobRun, error) {
	if s.tracker == nil {
		return nil, fmt.Errorf("job tracking is not enabled")
	}

	return s.tracker.GetRecentRunWithOffset(ctx, jobName, limit, offset)
}
```

**Step 2: Add GetRecentRunWithOffset to tracker**

Add to `go-backend/services/scheduler.go` in the `ScheduledExecutionTracker`:

```go
// GetRecentRunWithOffset returns recent runs with offset support
func (t *ScheduledExecutionTracker) GetRecentRunWithOffset(ctx context.Context, jobName string, limit int, offset int) ([]JobRun, error) {
	query := `
		SELECT id, job_name, started_at, completed_at, status, error_message, retry_count
		FROM scheduled_job_runs
		WHERE job_name = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := t.db.QueryContext(ctx, query, jobName, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query job history: %w", err)
	}
	defer rows.Close()

	var runs []JobRun
	for rows.Next() {
		var run JobRun
		var completedAt sql.NullTime
		err := rows.Scan(
			&run.ID,
			&run.JobName,
			&run.StartedAt,
			&completedAt,
			&run.Status,
			&run.ErrorMessage,
			&run.RetryCount,
		)
		if err != nil {
			return nil, err
		}
		if completedAt.Valid {
			run.CompletedAt = completedAt.Time
		}
		runs = append(runs, run)
	}

	return runs, nil
}
```

**Step 3: Update handler to parse offset**

Modify `GetJobHistory` in `go-backend/handlers/scheduler.go`:

```go
// GetJobHistory returns a handler that gets execution history for a specific job
func GetJobHistory(scheduler SchedulerAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		jobName := vars["jobName"]

		// Parse limit from query params, default to 50
		limit := 50
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		// Parse offset from query params, default to 0
		offset := 0
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		runs, err := scheduler.GetJobHistory(r.Context(), jobName, limit, offset)
		if err != nil {
			http.Error(w, "Failed to get job history", http.StatusInternalServerError)
			return
		}

		// Get total count for pagination
		// Note: This would require an additional query or a different method
		total := len(runs) + offset
		hasMore := len(runs) == limit

		// Convert services.JobRun to JobRunResponse
		responses := convertJobRunsToResponses(runs)

		response := map[string]interface{}{
			"runs":     responses,
			"total":    total,
			"offset":   offset,
			"limit":    limit,
			"has_more": hasMore,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
```

**Step 4: Update SchedulerAPI interface**

```go
type SchedulerAPI interface {
	ListJobs() []string
	GetJobHistory(ctx context.Context, jobName string, limit int, offset int) ([]services.JobRun, error)
	GetJobInfo(name string) (schedule string, nextRun time.Time, err error)
	GetJobSummary(ctx context.Context, jobName string) (JobSummary, error)
}
```

**Step 5: Run tests**

Run: `cd go-backend && go test ./handlers/... -v -run TestGetJobHistory`
Expected: Tests pass

**Step 6: Commit**

```bash
git add go-backend/handlers/scheduler.go go-backend/services/scheduler.go
git commit -m "feat(scheduler): add offset support to job history endpoint"
```

---

## Task 6: Backend - Register New Routes

**Files:**
- Modify: `go-backend/main.go`

**Step 1: Add new scheduler admin routes**

Find the scheduler admin routes section in `go-backend/main.go` and add:

```go
// Add these routes after the existing scheduler routes
r.HandleFunc("/admin/scheduler/jobs/{jobName}/summary",
	handlers.AdminMiddleware(handlers.GetJobSummaryHandler(scheduler))).
	Methods("GET")
```

**Step 2: Verify existing routes are present**

Ensure these routes exist:
- `/api/admin/scheduler/jobs` - ListScheduledJobs
- `/api/admin/scheduler/health` - GetSchedulerHealth
- `/api/admin/scheduler/jobs/{jobName}/history` - GetJobHistory

**Step 3: Run tests**

Run: `cd go-backend && go test ./... -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add go-backend/main.go
git commit -m "feat(scheduler): register job summary endpoint"
```

---

## Task 7: Frontend - Add API Types

**Files:**
- Modify: `zettelkasten-front/src/api/admin.ts`

**Step 1: Add scheduler types**

Add to `zettelkasten-front/src/api/admin.ts` after the existing types:

```typescript
/**
 * Information about a scheduled job
 */
export interface ScheduledJobInfo {
  name: string;
  schedule: string;
  next_run: string;
}

/**
 * Summary statistics for a scheduled job
 */
export interface JobSummary {
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

/**
 * Single execution record for a scheduled job
 */
export interface JobRun {
  id: number;
  job_name: string;
  started_at: string;
  completed_at?: string;
  status: string;
  error_message?: string;
  retry_count: number;
}

/**
 * Response from job history endpoint with pagination
 */
export interface JobHistoryResponse {
  runs: JobRun[];
  total: number;
  offset: number;
  limit: number;
  has_more: boolean;
}

/**
 * Scheduler health status
 */
export interface SchedulerHealth {
  running: boolean;
  jobs: string[];
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/api/admin.ts
git commit -m "feat(scheduler): add TypeScript types for scheduler API"
```

---

## Task 8: Frontend - Add API Functions

**Files:**
- Modify: `zettelkasten-front/src/api/admin.ts`

**Step 1: Add scheduler API functions**

Add to `zettelkasten-front/src/api/admin.ts` after the existing API functions:

```typescript
/**
 * Get all scheduled jobs
 */
export async function getScheduledJobs(): Promise<{ jobs: ScheduledJobInfo[] }> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/scheduler/jobs`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get scheduled jobs: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Get job summary statistics
 */
export async function getJobSummary(jobName: string): Promise<JobSummary> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/scheduler/jobs/${encodeURIComponent(jobName)}/summary`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get job summary: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Get job execution history with pagination
 */
export async function getJobHistory(
  jobName: string,
  limit: number = 50,
  offset: number = 0
): Promise<JobHistoryResponse> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");

  const params = new URLSearchParams();
  params.set("limit", limit.toString());
  params.set("offset", offset.toString());

  const url = `${base_url}/admin/scheduler/jobs/${encodeURIComponent(jobName)}/history?${params}`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get job history: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Get scheduler health status
 */
export async function getSchedulerHealth(): Promise<SchedulerHealth> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/scheduler/health`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get scheduler health: ${response.statusText}`);
  }

  return response.json();
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/api/admin.ts
git commit -m "feat(scheduler): add scheduler API functions"
```

---

## Task 9: Frontend - Add Utility Functions

**Files:**
- Create: `zettelkasten-front/src/utils/scheduler.ts`

**Step 1: Create scheduler utility file**

Create `zettelkasten-front/src/utils/scheduler.ts`:

```typescript
/**
 * Convert cron expression to human-readable description
 */
export function formatCronSchedule(cron: string): string {
  const parts = cron.trim().split(/\s+/);

  if (parts.length < 5 || parts.length > 6) {
    return cron;
  }

  const [seconds, minute, hour, dayOfMonth, month, dayOfWeek] =
    parts.length === 6 ? parts : ["0", ...parts];

  // Common patterns
  if (minute === "0" && hour === "2" && dayOfMonth === "*" && month === "*" && dayOfWeek === "*") {
    return "Daily at 2:00 AM";
  }
  if (minute === "0" && hour === "*" && dayOfMonth === "*" && month === "*" && dayOfWeek === "*") {
    return "Hourly";
  }
  if (cron === "0 0 * * 0") {
    return "Weekly (Sunday midnight)";
  }
  if (cron === "0 0 1 * *") {
    return "Monthly (1st at midnight)";
  }

  // Generic fallback
  const timeStr = `${hour}:${minute.padStart(2, "0")}`;
  if (dayOfMonth !== "*" && month !== "*") {
    return `Day ${dayOfMonth} of every month at ${timeStr}`;
  }
  if (dayOfWeek !== "*") {
    const days = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
    const dayIndex = parseInt(dayOfWeek);
    const day = days[dayIndex] ?? dayOfWeek;
    return `Every ${day} at ${timeStr}`;
  }

  return `Cron: ${cron}`;
}

/**
 * Format relative time for timestamps
 */
export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
}

/**
 * Format duration in milliseconds to human readable string
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = Math.floor((ms % 60000) / 1000);
  return `${minutes}m ${seconds}s`;
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/utils/scheduler.ts
git commit -m "feat(scheduler): add scheduler utility functions"
```

---

## Task 10: Frontend - Create JobStatusBadge Component

**Files:**
- Create: `zettelkasten-front/src/components/scheduler/JobStatusBadge.tsx`

**Step 1: Create JobStatusBadge component**

Create `zettelkasten-front/src/components/scheduler/JobStatusBadge.tsx`:

```typescript
import React from "react";

interface JobStatusBadgeProps {
  status: "completed" | "failed" | "running" | "never";
}

export function JobStatusBadge({ status }: JobStatusBadgeProps) {
  const styles = {
    completed: "bg-green-100 text-green-800",
    failed: "bg-red-100 text-red-800",
    running: "bg-yellow-100 text-yellow-800",
    never: "bg-gray-100 text-gray-800",
  };

  const labels = {
    completed: "Completed",
    failed: "Failed",
    running: "Running",
    never: "Never run",
  };

  const showPulse = status === "running";

  return (
    <span
      className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${styles[status]}`}
    >
      {showPulse && (
        <span className="w-2 h-2 bg-yellow-500 rounded-full mr-2 animate-pulse" />
      )}
      {!showPulse && status !== "never" && (
        <span className={`w-2 h-2 rounded-full mr-2 ${
          status === "completed" ? "bg-green-500" :
          status === "failed" ? "bg-red-500" :
          "bg-gray-500"
        }`} />
      )}
      {labels[status]}
    </span>
  );
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/scheduler/JobStatusBadge.tsx
git commit -m "feat(scheduler): add JobStatusBadge component"
```

---

## Task 11: Frontend - Create ScheduleDisplay Component

**Files:**
- Create: `zettelkasten-front/src/components/scheduler/ScheduleDisplay.tsx`

**Step 1: Create ScheduleDisplay component**

Create `zettelkasten-front/src/components/scheduler/ScheduleDisplay.tsx`:

```typescript
import React from "react";
import { formatCronSchedule, formatRelativeTime } from "../../utils/scheduler";

interface ScheduleDisplayProps {
  schedule: string;
  nextRun: string;
}

export function ScheduleDisplay({ schedule, nextRun }: ScheduleDisplayProps) {
  const humanReadable = formatCronSchedule(schedule);
  const relativeTime = formatRelativeTime(nextRun);

  return (
    <div className="group relative">
      <div className="text-sm text-gray-900">{humanReadable}</div>
      <div className="text-xs text-gray-500">
        Next: {relativeTime}
      </div>
      {/* Tooltip with raw cron */}
      <div className="absolute bottom-full left-0 mb-2 hidden group-hover:block bg-gray-900 text-white text-xs px-2 py-1 rounded whitespace-nowrap z-10">
        {schedule}
      </div>
    </div>
  );
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/scheduler/ScheduleDisplay.tsx
git commit -m "feat(scheduler): add ScheduleDisplay component"
```

---

## Task 12: Frontend - Create RecentStatsSummary Component

**Files:**
- Create: `zettelkasten-front/src/components/scheduler/RecentStatsSummary.tsx`

**Step 1: Create RecentStatsSummary component**

Create `zettelkasten-front/src/components/scheduler/RecentStatsSummary.tsx`:

```typescript
import React from "react";
import { JobSummary } from "../../api/admin";

interface RecentStatsSummaryProps {
  summary?: JobSummary;
}

export function RecentStatsSummary({ summary }: RecentStatsSummaryProps) {
  if (!summary) {
    return <span className="text-sm text-gray-400 italic">Stats unavailable</span>;
  }

  const { total_runs, success_count, failure_count, success_rate } = summary.recent_stats;

  if (total_runs === 0) {
    return <span className="text-sm text-gray-400 italic">No recent runs</span>;
  }

  // Determine color based on success rate
  const getColor = () => {
    if (success_rate >= 95) return "bg-green-500";
    if (success_rate >= 70) return "bg-yellow-500";
    return "bg-red-500";
  };

  return (
    <div className="group relative flex items-center gap-2">
      {/* Success rate bar */}
      <div className="w-16 h-2 bg-gray-200 rounded-full overflow-hidden">
        <div
          className={`h-full ${getColor()} transition-all`}
          style={{ width: `${success_rate}%` }}
        />
      </div>
      <span className="text-sm text-gray-600">{success_rate.toFixed(0)}%</span>

      {/* Tooltip with detailed stats */}
      <div className="absolute bottom-full left-0 mb-2 hidden group-hover:block bg-gray-900 text-white text-xs px-3 py-2 rounded z-10">
        <div className="font-medium mb-1">Last 7 days</div>
        <div>Total runs: {total_runs}</div>
        <div className="text-green-400">Success: {success_count}</div>
        <div className="text-red-400">Failed: {failure_count}</div>
      </div>
    </div>
  );
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/scheduler/RecentStatsSummary.tsx
git commit -m "feat(scheduler): add RecentStatsSummary component"
```

---

## Task 13: Frontend - Create ExecutionHistoryTable Component

**Files:**
- Create: `zettelkasten-front/src/components/scheduler/ExecutionHistoryTable.tsx`

**Step 1: Create ExecutionHistoryTable component**

Create `zettelkasten-front/src/components/scheduler/ExecutionHistoryTable.tsx`:

```typescript
import React, { useState } from "react";
import { JobRun } from "../../api/admin";
import { formatDuration } from "../../utils/scheduler";

interface ExecutionHistoryTableProps {
  runs: JobRun[];
}

export function ExecutionHistoryTable({ runs }: ExecutionHistoryTableProps) {
  const [expandedError, setExpandedError] = useState<number | null>(null);

  if (runs.length === 0) {
    return (
      <div className="py-8 text-center text-gray-500 text-sm">
        This job has never run
      </div>
    );
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  const getStatusBadge = (status: string) => {
    const styles = {
      completed: "bg-green-100 text-green-800",
      failed: "bg-red-100 text-red-800",
      running: "bg-yellow-100 text-yellow-800",
    };
    return (
      <span
        className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${styles[status as keyof typeof styles] || styles.running}`}
      >
        {status}
      </span>
    );
  };

  return (
    <table className="w-full text-sm">
      <thead className="bg-gray-50">
        <tr>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Started</th>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Duration</th>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Retries</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-200">
        {runs.map((run) => {
          const duration =
            run.completed_at && run.started_at
              ? new Date(run.completed_at).getTime() - new Date(run.started_at).getTime()
              : null;

          return (
            <React.Fragment key={run.id}>
              <tr className="hover:bg-gray-50">
                <td className="px-3 py-2 text-gray-900">{run.id}</td>
                <td className="px-3 py-2 text-gray-600">{formatDate(run.started_at)}</td>
                <td className="px-3 py-2 text-gray-600">
                  {duration !== null ? formatDuration(duration) : "-"}
                </td>
                <td className="px-3 py-2">{getStatusBadge(run.status)}</td>
                <td className="px-3 py-2 text-gray-600">{run.retry_count}</td>
              </tr>
              {run.status === "failed" && run.error_message && (
                <tr>
                  <td colSpan={5} className="px-3 py-2 bg-red-50">
                    <button
                      onClick={() => setExpandedError(expandedError === run.id ? null : run.id)}
                      className="text-xs text-red-700 hover:text-red-900 font-medium"
                    >
                      {expandedError === run.id ? "Hide" : "Show"} error message
                    </button>
                    {expandedError === run.id && (
                      <pre className="mt-2 text-xs text-red-800 whitespace-pre-wrap font-mono bg-red-100 p-2 rounded">
                        {run.error_message}
                      </pre>
                    )}
                  </td>
                </tr>
              )}
            </React.Fragment>
          );
        })}
      </tbody>
    </table>
  );
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/scheduler/ExecutionHistoryTable.tsx
git commit -m "feat(scheduler): add ExecutionHistoryTable component"
```

---

## Task 14: Frontend - Create ExpandableHistory Component

**Files:**
- Create: `zettelkasten-front/src/components/scheduler/ExpandableHistory.tsx`

**Step 1: Create ExpandableHistory component**

Create `zettelkasten-front/src/components/scheduler/ExpandableHistory.tsx`:

```typescript
import React, { useState, useEffect } from "react";
import { getJobHistory, JobRun } from "../../api/admin";
import { ExecutionHistoryTable } from "./ExecutionHistoryTable";

interface ExpandableHistoryProps {
  jobName: string;
  isExpanded: boolean;
}

export function ExpandableHistory({ jobName, isExpanded }: ExpandableHistoryProps) {
  const [runs, setRuns] = useState<JobRun[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);

  // Load history when first expanded
  useEffect(() => {
    if (isExpanded && runs.length === 0) {
      loadHistory(0);
    }
  }, [isExpanded]);

  const loadHistory = async (newOffset: number) => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await getJobHistory(jobName, 50, newOffset);
      if (newOffset === 0) {
        setRuns(response.runs);
      } else {
        setRuns((prev) => [...prev, ...response.runs]);
      }
      setOffset(newOffset + response.runs.length);
      setHasMore(response.has_more);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load history");
    } finally {
      setIsLoading(false);
    }
  };

  const handleLoadMore = () => {
    loadHistory(offset);
  };

  if (!isExpanded) {
    return null;
  }

  return (
    <div className="mt-4 pl-4 border-l-2 border-gray-200">
      {error && (
        <div className="mb-4 p-3 bg-red-50 text-red-700 text-sm rounded">
          {error}
          <button
            onClick={() => loadHistory(0)}
            className="ml-2 underline hover:no-underline"
          >
            Retry
          </button>
        </div>
      )}

      {isLoading && runs.length === 0 ? (
        <div className="py-8 text-center text-gray-500 text-sm">Loading job history...</div>
      ) : (
        <>
          <ExecutionHistoryTable runs={runs} />

          {hasMore && !isLoading && (
            <div className="mt-4 text-center">
              <button
                onClick={handleLoadMore}
                className="px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg text-sm font-medium transition-colors"
              >
                Load more
              </button>
            </div>
          )}

          {isLoading && runs.length > 0 && (
            <div className="mt-4 text-center text-sm text-gray-500">Loading more...</div>
          )}
        </>
      )}
    </div>
  );
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/scheduler/ExpandableHistory.tsx
git commit -m "feat(scheduler): add ExpandableHistory component"
```

---

## Task 15: Frontend - Create AdminSchedulerPage Component

**Files:**
- Create: `zettelkasten-front/src/pages/admin/AdminSchedulerPage.tsx`

**Step 1: Create AdminSchedulerPage component**

Create `zettelkasten-front/src/pages/admin/AdminSchedulerPage.tsx`:

```typescript
import React, { useState, useEffect } from "react";
import {
  getScheduledJobs,
  getSchedulerHealth,
  getJobSummary,
  ScheduledJobInfo,
  SchedulerHealth,
  JobSummary,
} from "../../api/admin";
import { AdminErrorDisplay } from "../../components/admin/AdminErrorDisplay";
import { JobStatusBadge } from "../../components/scheduler/JobStatusBadge";
import { ScheduleDisplay } from "../../components/scheduler/ScheduleDisplay";
import { RecentStatsSummary } from "../../components/scheduler/RecentStatsSummary";
import { ExpandableHistory } from "../../components/scheduler/ExpandableHistory";
import { formatRelativeTime } from "../../utils/scheduler";

interface SchedulerJobData extends ScheduledJobInfo {
  summary?: JobSummary;
}

interface ErrorState {
  message: string;
  details?: string;
}

export function AdminSchedulerPage() {
  const [jobs, setJobs] = useState<SchedulerJobData[]>([]);
  const [health, setHealth] = useState<SchedulerHealth | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<ErrorState | null>(null);
  const [expandedJobs, setExpandedJobs] = useState<Set<string>>(new Set());

  const fetchAllData = async (showRefreshing = false) => {
    if (showRefreshing) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }
    setError(null);

    try {
      const [jobsData, healthData] = await Promise.all([
        getScheduledJobs(),
        getSchedulerHealth(),
      ]);

      setHealth(healthData);

      // Fetch summaries for each job
      const jobsWithSummaries = await Promise.all(
        jobsData.jobs.map(async (job) => {
          try {
            const summary = await getJobSummary(job.name);
            return { ...job, summary };
          } catch {
            return { ...job, summary: undefined };
          }
        })
      );

      setJobs(jobsWithSummaries);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load scheduler data";
      setError({ message, details: err instanceof Error ? err.stack : undefined });
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    fetchAllData();
  }, []);

  const handleRefresh = () => {
    fetchAllData(true);
  };

  const toggleExpanded = (jobName: string) => {
    setExpandedJobs((prev) => {
      const next = new Set(prev);
      if (next.has(jobName)) {
        next.delete(jobName);
      } else {
        next.add(jobName);
      }
      return next;
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-pulse text-gray-600">Loading scheduled jobs...</div>
      </div>
    );
  }

  if (error && !health) {
    return (
      <AdminErrorDisplay
        message={error.message}
        details={error.details}
        severity="error"
        onRetry={() => fetchAllData()}
        onDismiss={() => setError(null)}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Scheduled Jobs</h1>
          <p className="text-gray-600 mt-1">
            Monitor and manage cron-based scheduled jobs
          </p>
        </div>
        <div className="flex items-center gap-3">
          {health && (
            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${
              health.running ? "bg-green-100 text-green-800" : "bg-gray-100 text-gray-800"
            }`}>
              <span className={`w-2 h-2 rounded-full mr-2 ${health.running ? "bg-green-500" : "bg-gray-500"}`} />
              {health.running ? "Running" : "Stopped"}
            </span>
          )}
          <button
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg font-medium transition-colors disabled:opacity-50 flex items-center gap-2"
          >
            {isRefreshing ? (
              <>
                <span className="animate-spin">⟳</span>
                Refreshing...
              </>
            ) : (
              <>
                <span>🔄</span>
                Refresh
              </>
            )}
          </button>
        </div>
      </div>

      {error && health && (
        <AdminErrorDisplay
          message={error.message}
          details={error.details}
          severity="warning"
          onDismiss={() => setError(null)}
        />
      )}

      {/* Jobs Table */}
      {jobs.length === 0 ? (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
          <p className="text-gray-500">No scheduled jobs configured</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Job Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Schedule
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Recent Stats
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {jobs.map((job) => {
                const isExpanded = expandedJobs.has(job.name);
                const lastStatus = job.summary?.last_run_status ?? "never";

                return (
                  <React.Fragment key={job.name}>
                    <tr className="hover:bg-gray-50">
                      <td className="px-6 py-4 text-sm font-medium text-gray-900">
                        {job.name}
                      </td>
                      <td className="px-6 py-4">
                        <ScheduleDisplay schedule={job.schedule} nextRun={job.next_run} />
                      </td>
                      <td className="px-6 py-4">
                        <JobStatusBadge status={lastStatus as any} />
                      </td>
                      <td className="px-6 py-4">
                        <RecentStatsSummary summary={job.summary} />
                      </td>
                      <td className="px-6 py-4">
                        <button
                          onClick={() => toggleExpanded(job.name)}
                          className="text-sm text-blue-600 hover:text-blue-800 font-medium"
                        >
                          {isExpanded ? "Hide" : "View"} History
                        </button>
                      </td>
                    </tr>
                    {isExpanded && (
                      <tr>
                        <td colSpan={5} className="px-6 py-0">
                          <ExpandableHistory jobName={job.name} isExpanded={isExpanded} />
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
```

**Step 2: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/pages/admin/AdminSchedulerPage.tsx
git commit -m "feat(scheduler): add AdminSchedulerPage component"
```

---

## Task 16: Frontend - Add Routing and Sidebar Link

**Files:**
- Modify: `zettelkasten-front/src/pages/admin/AdminPage.tsx`

**Step 1: Import AdminSchedulerPage**

Add import to `zettelkasten-front/src/pages/admin/AdminPage.tsx`:

```typescript
import { AdminSchedulerPage } from "./AdminSchedulerPage";
```

**Step 2: Add sidebar link**

Add to the sidebar navigation list in `AdminPage.tsx` after the Job Queue link:

```tsx
<li>
  <Link
    to="/admin/scheduler"
    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
    onClick={() => setIsSidebarOpen(false)}
  >
    ⏰ Scheduled Jobs
  </Link>
</li>
```

**Step 3: Add route**

Add to the Routes section in `AdminPage.tsx`:

```tsx
<Route path="scheduler" element={<AdminSchedulerPage />} />
```

**Step 4: Run TypeScript check**

Run: `cd zettelkasten-front && npm run type-check`
Expected: No type errors

**Step 5: Commit**

```bash
git add zettelkasten-front/src/pages/admin/AdminPage.tsx
git commit -m "feat(scheduler): add scheduled jobs page to admin navigation"
```

---

## Task 17: Testing - Backend Tests

**Files:**
- Modify: `go-backend/handlers/scheduler_test.go`

**Step 1: Add test for enhanced ListScheduledJobs**

Add to `go-backend/handlers/scheduler_test.go`:

```go
func TestListScheduledJobs_WithSchedule(t *testing.T) {
	// Create mock scheduler
	mockScheduler := &MockScheduler{
		jobs: map[string]MockScheduledJob{
			"daily-cleanup": {
				name:     "daily-cleanup",
				schedule: "0 2 * * *",
			},
		},
	}

	handler := ListScheduledJobs(mockScheduler)
	req := httptest.NewRequest("GET", "/admin/scheduler/jobs", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var response ScheduledJobsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(response.Jobs))
	}

	if response.Jobs[0].Name != "daily-cleanup" {
		t.Errorf("expected job name 'daily-cleanup', got '%s'", response.Jobs[0].Name)
	}

	if response.Jobs[0].Schedule != "0 2 * * *" {
		t.Errorf("expected schedule '0 2 * * *', got '%s'", response.Jobs[0].Schedule)
	}
}
```

**Step 2: Run tests**

Run: `cd go-backend && go test ./handlers/... -v -run TestListScheduledJobs`
Expected: Tests pass

**Step 3: Commit**

```bash
git add go-backend/handlers/scheduler_test.go
git commit -m "test(scheduler): add tests for enhanced ListScheduledJobs handler"
```

---

## Task 18: Final Verification

**Files:** None (verification)

**Step 1: Run all backend tests**

Run: `cd go-backend && go test ./... -v`
Expected: All tests pass

**Step 2: Run frontend build**

Run: `cd zettelkasten-front && npm run build`
Expected: Successful build

**Step 3: Run frontend tests**

Run: `cd zettelkasten-front && npm test`
Expected: All tests pass

**Step 4: Manual smoke test**

1. Start backend: `cd go-backend && go run main.go`
2. Start frontend: `cd zettelkasten-front && npm start`
3. Login as admin user
4. Navigate to /admin/scheduler
5. Verify:
   - Jobs list displays
   - Status badges show correct colors
   - Schedule displays human-readable format
   - Expanding row shows history
   - Load more button works

**Step 5: Update CLAUDE.md**

Add to `CLAUDE.md` under "Key Features > Admin":

```markdown
### Scheduled Jobs Admin
- View all registered scheduled jobs with schedules
- Monitor job execution status and history
- Per-job statistics (success rate, recent runs)
- Expandable history with pagination
- Manual refresh for current status
```

**Step 6: Final commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with scheduled jobs admin feature"
```

---

## Implementation Complete

The admin UI for Scheduled Job Runner is now fully implemented with:
- Backend API endpoints for jobs, summaries, and history
- Frontend components for displaying job information
- Expandable rows for execution history
- Manual refresh functionality
- Error handling and empty states
- Tests and documentation
