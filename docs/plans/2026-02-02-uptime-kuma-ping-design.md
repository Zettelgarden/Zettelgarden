# Uptime Kuma Ping Job Design

## Overview
A new scheduled job that sends heartbeat POST requests to Uptime Kuma's push monitor endpoint every minute, serving as a health check that proves the job scheduler is operational.

## Purpose
This job exists to verify that the Zettelgarden job scheduler is functioning correctly. By posting to Uptime Kuma on a regular schedule, any failure indicates the scheduler itself may be down.

## Configuration

### Environment Variable
- **`UPTIME_KUMA_PUSH_URL`**: Full URL to Uptime Kuma push endpoint
  - Example: `https://uptime.example.com/api/push/YOUR_MONITOR_ID?status=up&msg=OK`
  - Required (job will fail gracefully if not configured)

## Job Specification

| Property | Value |
|----------|-------|
| **Job Name** | `uptime_kuma_ping` |
| **Schedule** | Every minute (`0 * * * * *`) |
| **HTTP Method** | POST |
| **Max Retries** | 3 |
| **Timeout** | 30 seconds per attempt |

## Behavior

### Success
- HTTP 200 response from Uptime Kuma
- Job marked as completed in `scheduled_job_runs` table

### Failure Handling
- Retry with exponential backoff: 2s, 4s, 8s between retries
- After max retries, job marked as failed
- Error details logged to `scheduled_job_runs` table

## Implementation

### File Location
`go-backend/services/jobs/uptime_kuma_ping_job.go`

### Registration
The job will be registered in `main.go`:
```go
scheduler.Register(jobs.NewUptimeKumaPingJob())
```

### Dependencies
- None (uses only standard HTTP client)
