# Daily Digest Email Feature - Design Document

**Created**: 2026-02-01
**Status**: Approved Design

## Purpose

Send users a daily digest email at 7 AM local time containing their tasks for the day plus a random card for rediscovery and spaced repetition.

## Requirements

### Functional Requirements
- Send daily email at 7 AM in each user's local timezone
- Include tasks scheduled for today (user's local date)
- Include overdue incomplete tasks (due_date < now OR scheduled_date < now with no due_date)
- Include a random card with title + ~200 character preview
- HTML email format following existing TaskRemindersJob pattern
- Testing mode: only send to user 1 initially

### Non-Functional Requirements
- No duplicate emails (prevent multiple sends on same day)
- Graceful error handling for missing data
- Respect mail service rate limits
- Track sent emails for metrics and troubleshooting

## Architecture

### Job Registration
**Location**: `go-backend/main.go`
```go
scheduler.Register(jobs.NewDailyDigestJob(s.DB, s.Mail))
```

### DailyDigestJob Structure
**Location**: `go-backend/services/jobs/daily_digest_job.go`

```go
type DailyDigestJob struct {
    db              *sql.DB
    mail            mail.MailClient
    schedule        string  // "07:00" - configurable
    enabledUserID   *int    // If set, only send to this user (testing)
}
```

### Schedule Pattern
- Runs every minute (`* * * * * *`)
- Processes users whose local time is 7:00 AM
- Similar to TaskRemindersJob approach

### Processing Flow
```
Every minute at 00 seconds
    ↓
Query users (or user 1 if testing)
    ↓
For each user:
    1. Get user's timezone
    2. Check if local time is 07:00
    3. Get today's tasks + overdue tasks
    4. Get random card (with content preview)
    5. Format HTML email
    6. Send via mail.SendHTMLEmail()
    7. Track last_sent_date in daily_digest_log table
```

## Database Schema

### New Table: daily_digest_log

```sql
CREATE TABLE daily_digest_log (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    sent_date DATE NOT NULL,
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    task_count INTEGER NOT NULL,
    card_id INTEGER REFERENCES cards(id),
    UNIQUE(user_id, sent_date)  -- Prevent duplicate sends
);
```

**Purpose**: Track when digests are sent to prevent duplicates and provide metrics.

## Email Template Structure

```html
<!-- Header -->
<h2>Good morning!</h2>
<p>Here's your daily digest for {date}</p>

<!-- Today's Tasks -->
<h3>Today's Tasks</h3>
<ul>
  <li>Task 1</li>
  <li>Task 2</li>
</ul>

<!-- Overdue Tasks -->
<h3>Overdue</h3>
<ul>
  <li>Task A</li>
  <li>Task B</li>
</ul>

<!-- Random Card -->
<h3>Random Card</h3>
<div class="card">
  <h4>{card title}</h4>
  <p>{card content preview}</p>
</div>

<!-- Footer -->
<p>View in Zettelgarden</p>
```

## New Service Methods

### services/tasks.go
- `GetDailyDigestTasks(db, userID, timezone)` - Returns today's + overdue tasks
- `GetRandomCard(db, userID)` - Returns a random card with content preview

### Task Query Details
```sql
-- Today's tasks (timezone-aware)
SELECT * FROM tasks
WHERE user_id = $1
  AND is_deleted = FALSE
  AND is_complete = FALSE
  AND DATE(scheduled_date AT TIME ZONE $2) = DATE(NOW() AT TIME ZONE $2)

-- Overdue tasks
SELECT * FROM tasks
WHERE user_id = $1
  AND is_deleted = FALSE
  AND is_complete = FALSE
  AND (
    due_date < NOW()
    OR (due_date IS NULL AND scheduled_date < NOW())
  )
```

## Error Handling

| Scenario | Handling |
|----------|----------|
| User has no timezone | Skip user, log warning |
| User has no email | Skip user, log warning |
| No tasks for today | Send digest with "No tasks scheduled for today" |
| User has no cards | Skip random card section or show "No card available" |
| Mail service unavailable | Log error, don't update log (will retry) |
| Card content empty | Show title only with "(no content)" |
| Race condition (double send) | UNIQUE constraint prevents duplicate |

## Implementation Plan

1. Create migration for `daily_digest_log` table
2. Implement `GetDailyDigestTasks()` in services/tasks.go
3. Implement `GetRandomCard()` in services/tasks.go
4. Create `DailyDigestJob` in services/jobs/daily_digest_job.go
5. Register job in main.go
6. Test with user 1 only
7. After verification, remove `enabledUserID` restriction

## Future Enhancements

- Per-user digest time preference
- User opt-out setting
- Digest frequency preference (daily/weekly)
- Multiple random cards
- Customizable card categories
- Statistics dashboard for digest engagement
