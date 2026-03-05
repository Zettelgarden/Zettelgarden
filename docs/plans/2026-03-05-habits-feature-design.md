# Habits Feature Design

**Date:** 2026-03-05
**Status:** Approved
**Author:** Claude (user collaboration)

## Overview

Add a habits tracking system to Zettelgarden for daily consistency tracking (medication, exercise, etc.). Habits are task-adjacent—separate data and UI, but can optionally link to tasks.

## Goals

- Quick daily check-ins via sidebar widget
- Rich analytics: streaks, calendar heatmap, completion rates
- Support flexible frequency (daily, specific weekdays)
- Optional notes per check-in
- Personal use only (no sharing/templates)

## Approach

Balanced approach: all core requirements met without over-engineering. Separate habits system with optional task linking, dual UI entry points (sidebar + dedicated page), and comprehensive analytics from day one.

---

## Data Model

### habits table

```sql
CREATE TABLE habits (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    frequency VARCHAR(20) NOT NULL, -- 'daily', 'weekly', 'custom_days'
    custom_days JSONB, -- [1,3,5] for Mon/Wed/Fri (ISO weekday numbers)
    icon VARCHAR(50),
    color VARCHAR(7), -- hex color
    position INTEGER NOT NULL DEFAULT 0,
    linked_task_id INTEGER REFERENCES tasks(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_habits_user_id ON habits(user_id);
CREATE INDEX idx_habits_position ON habits(user_id, position);
```

### habit_logs table

```sql
CREATE TABLE habit_logs (
    id SERIAL PRIMARY KEY,
    habit_id INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_habit_logs_habit_completed ON habit_logs(habit_id, completed_at DESC);
CREATE INDEX idx_habit_logs_user_completed ON habit_logs(user_id, completed_at DESC);
```

---

## API Endpoints

### CRUD

- `GET /api/habits` - List user's habits (ordered by position)
- `POST /api/habits` - Create habit
- `GET /api/habits/:id` - Get habit details
- `PUT /api/habits/:id` - Update habit
- `DELETE /api/habits/:id` - Delete habit

### Check-ins

- `POST /api/habits/:id/checkin` - Log completion with optional notes
- `DELETE /api/habits/:id/checkin/:logId` - Remove a check-in
- `GET /api/habits/:id/logs` - Get check-in history (paginated, date range filterable)

### Analytics

- `GET /api/habits/:id/stats` - Streaks, completion rates, totals
- `GET /api/habits/today` - All habits due today with check-in status (for sidebar)

### Task Linking

- `PUT /api/habits/:id/link` - Link/unlink to a task (body: `{"task_id": 123}` or `null`)

### Response Shapes

**GET /habits** returns array with:
```json
{
  "id": 1,
  "title": "Take vitamins",
  "frequency": "daily",
  "icon": "💊",
  "color": "#10b981",
  "today_checked_in": true,
  "current_streak": 14
}
```

**GET /habits/:id/stats** returns:
```json
{
  "current_streak": 14,
  "longest_streak": 42,
  "total_completions": 156,
  "completion_rate_7d": 0.857,
  "completion_rate_30d": 0.923,
  "last_completed_at": "2026-03-05T08:30:00Z"
}
```

**GET /habits/today** returns:
```json
[
  {
    "id": 1,
    "title": "Take vitamins",
    "is_due_today": true,
    "checked_in_today": true,
    "today_log_id": 1234
  }
]
```

---

## Frontend Components

### Pages

**/habits (main habits page)**
- Left: `HabitList` - draggable list of habits with quick check-in buttons
- Right: `HabitDetail` - selected habit shows stats, calendar heatmap, logs

### Sidebar

**Today's Habits widget**
- Compact list of habits due today
- Check-in button (turns green after)
- Shows notes indicator
- Click → opens habit detail

### Shared Components

- `HabitCheckinButton` - Reusable check-in with optional notes dialog
- `HabitCalendar` - GitHub-style heatmap (green squares), clickable for day's logs
- `HabitStatsCard` - Streak display "🔥 14 days", completion rate percentage
- `FrequencySelector` - Daily picker or weekday multi-select

### Dialogs

- `CreateHabitDialog` - Title, description, frequency, icon, color
- `CheckinNotesDialog` - Optional notes after check-in

### State Management

Add `HabitContext` alongside existing contexts (`CardContext`, `TaskContext`).

---

## Data Flow & Key Behaviors

### Check-in Flow

1. User clicks check-in → optimistic UI update
2. API call to `POST /habits/:id/checkin`
3. Optional notes dialog → POST with notes
4. Success: refetch today's widget, update detail stats
5. Error: revert optimistic update, show toast

### "Due Today" Logic (Backend)

```
IF frequency = 'daily' → always due
IF frequency = 'weekly' → due if custom_days includes today's weekday
```

### Streak Calculation (Backend)

- Query logs ordered by `completed_at DESC`
- Walk backward counting consecutive days within habit's frequency
- Break streak when gap found
- Cache in Redis with 1-hour TTL (optional optimization)

### Habit → Task Linking

- Habit creation: optional "Link to task" dropdown
- Task card shows "Related habit" badge
- Task completion does NOT auto-complete habit

### Midnight Reset (Client)

- Sidebar widget refetches `/habits/today` every minute
- Date change → clear local state and refetch

---

## Error Handling

### API Errors

- 409 Conflict: Already checked in today
- 404: Habit not found
- 400: Invalid frequency data

### Edge Cases

- **Timezones**: Store UTC, convert to user's timezone (reuse `GetUserTimezone`)
- **Duplicate check-ins**: Backend checks for existing log within calendar day
- **Deleting habits**: Cascade delete logs
- **Linked task deleted**: Set `linked_task_id` to NULL
- **Missed days**: No penalty, just breaks streak
- **Retroactive check-ins**: Not supported in MVP

### Optimistic UI Fallback

- API fails → revert button state, show error toast
- Stats fail → show "Stats unavailable" with retry

---

## Testing Strategy

### Backend (Go)

- Unit tests for streak calculation (timezone boundaries, missed days)
- API handler tests for all endpoints
- Integration test: check-in → stats update flow

### Frontend (Vitest + RTL)

- Component tests: `HabitCheckinButton`, `HabitCalendar`, `FrequencySelector`
- Integration test: check-in → optimistic update → API → success
- Context tests: `HabitContext` provider and hooks

### Manual Testing

- Create habit → appears in list and sidebar
- Check-in → button updates, streak increments, calendar updates
- Miss a day → streak resets
- Timezone change → "today" correctly shifts
- Link/unlink task → relationship updates

---

## Migration

Add migration to `go-backend/schema/`:
- Create `habits` table
- Create `habit_logs` table
- Create indexes

---

## Out of Scope (Future Enhancements)

- Habit templates/library
- Retroactive check-ins (backfilling history)
- Soft-delete for habits
- Advanced scheduling (X times per week, rolling windows)
- Reminders for habits
- Habit insights/trends
- Export habit data
