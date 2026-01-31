# External Events Integration - Implementation Plan

**Created:** 2026-01-31
**Design:** docs/plans/2026-01-31-external-events-integration.md
**Worktree:** feature/external-events-integration

## Overview

This plan breaks down the implementation of external calendar events integration into executable steps. Each step includes verification criteria.

---

## Phase 1: Backend - Database & Models

### Step 1.1: Create external_events table migration

**File:** `go-backend/schema/0102-add-external-events.sql`

```sql
CREATE TABLE external_events (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_calendar_id INTEGER REFERENCES external_calendars(id) ON DELETE SET NULL,

    title TEXT NOT NULL,
    description TEXT,

    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    all_day BOOLEAN DEFAULT FALSE,

    location TEXT,
    external_uid TEXT,
    external_url TEXT,
    recurrence_rule TEXT,

    color TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_synced_at TIMESTAMP WITH TIME ZONE,

    UNIQUE(user_id, external_uid)
);

CREATE INDEX idx_external_events_user_time ON external_events(user_id, start_time, end_time);
CREATE INDEX idx_external_events_calendar ON external_events(external_calendar_id);
COMMENT ON TABLE external_events IS 'Imported calendar events from external iCal feeds';
```

**Verify:** Run migration, check table exists with correct schema.

### Step 1.2: Create ExternalEvent model

**File:** `go-backend/models/external_events.go`

```go
package models

import "time"

type ExternalEvent struct {
    ID                 int        `json:"id"`
    UserID             int        `json:"user_id"`
    ExternalCalendarID *int       `json:"external_calendar_id,omitempty"`
    Title              string     `json:"title"`
    Description        *string    `json:"description,omitempty"`
    StartTime          time.Time  `json:"start_time"`
    EndTime            time.Time  `json:"end_time"`
    AllDay             bool       `json:"all_day"`
    Location           *string    `json:"location,omitempty"`
    ExternalUID        *string    `json:"external_uid,omitempty"`
    ExternalURL        *string    `json:"external_url,omitempty"`
    RecurrenceRule     *string    `json:"recurrence_rule,omitempty"`
    Color              *string    `json:"color,omitempty"`
    CreatedAt          time.Time  `json:"created_at"`
    UpdatedAt          time.Time  `json:"updated_at"`
    LastSyncedAt       *time.Time `json:"last_synced_at,omitempty"`
}

type CreateExternalEventRequest struct {
    Title          string `json:"title"`
    Description    string `json:"description,omitempty"`
    StartTime      string `json:"start_time"`
    EndTime        string `json:"end_time"`
    AllDay         bool   `json:"all_day"`
    Location       string `json:"location,omitempty"`
    Color          string `json:"color,omitempty"`
}
```

**Verify:** Model compiles, JSON tags are correct.

---

## Phase 2: Backend - iCal Service

### Step 2.1: Create iCal parser with safety limits

**File:** `go-backend/services/ical.go`

- Implement `ParseICalendar` with `MAX_EVENTS_PER_FEED = 2000`
- Implement `parseICalDateTime` with TZID support and UTC conversion
- Implement `unescapeICalText` for iCal escaping
- Implement `FetchICalURL` with 30s timeout

**Verify:**
- Unit test: Parse valid iCal feed
- Unit test: Hit max event limit
- Unit test: Parse events with TZID
- Unit test: Handle malformed iCal gracefully

### Step 2.2: Create ExternalEventService

**File:** `go-backend/services/external_events.go`

- Implement `SyncExternalCalendar`
- Implement `importEvent` with upsert logic
- Implement `UpdateLastSynced`
- Implement `UpdateLastSyncError`
- Implement `GetEventsInRange`

**Verify:**
- Unit test: Import new event
- Unit test: Update existing event (same UID)
- Unit test: Query events in date range

---

## Phase 3: Backend - API Handlers

### Step 3.1: Create calendar events handlers

**File:** `go-backend/handlers/external_events.go`

- `ListExternalCalendarsRoute` - GET /api/user/external-calendars
- `CreateExternalCalendarRoute` - POST /api/user/external-calendars
- `UpdateExternalCalendarRoute` - PUT /api/user/external-calendars/{id}
- `DeleteExternalCalendarRoute` - DELETE /api/user/external-calendars/{id}
- `SyncExternalCalendarRoute` - POST /api/user/external-calendars/{id}/sync
- `GetExternalEventsRoute` - GET /api/user/external-events

**Verify:**
- Manual test: Subscribe to Google Calendar public iCal URL
- Manual test: Manual sync works
- Manual test: Events returned with correct date range

### Step 3.2: Register routes

**File:** `go-backend/routes/external_events.go`

- Register all routes with mux
- Add to main routes.go

**Verify:**
- Routes appear in route log
- All endpoints respond (401 if not auth, 200/400/500 otherwise)

---

## Phase 4: Frontend - Models & Types

### Step 4.1: Update CalendarEvent model

**File:** `zettelkasten-front/src/models/CalendarEvent.ts`

Add:
- `source: "task" | "external"`
- `externalEventId?: number`
- `description?: string`
- `location?: string`
- `externalUrl?: string`

### Step 4.2: Create ExternalEvent model

**File:** `zettelkasten-front/src/models/ExternalEvent.ts`

```typescript
export interface ExternalEvent {
  id: number;
  user_id: number;
  external_calendar_id?: number;
  title: string;
  description?: string;
  start_time: string;
  end_time: string;
  all_day: boolean;
  location?: string;
  external_uid?: string;
  external_url?: string;
  recurrence_rule?: string;
  color?: string;
  created_at: string;
  updated_at: string;
  last_synced_at?: string;
}
```

**Verify:** TypeScript compiles without errors.

---

## Phase 5: Frontend - API Client

### Step 5.1: Create external events API client

**File:** `zettelkasten-front/src/api/externalEvents.ts`

```typescript
import { apiClient, getData } from './client';
import { ExternalEvent } from '../models/ExternalEvent';

export async function getExternalEvents(start: Date, end: Date): Promise<ExternalEvent[]> {
  return getData(apiClient.get<ExternalEvent[]>('/user/external-events', {
    params: { start: start.toISOString(), end: end.toISOString() },
  }));
}
```

**Verify:** Network requests succeed to backend.

---

## Phase 6: Frontend - Calendar Utilities

### Step 6.1: Add merge and display utilities

**File:** `zettelkasten-front/src/utils/calendar.ts`

Add functions:
- `mergeCalendarEvents(taskEvents, externalEvents)`
- `getEventIcon(event)` - returns calendar icon for external
- `isEventDraggable(event)` - returns false for external

**Verify:**
- Unit test: Merge combines both event types
- Unit test: External events get correct icon
- Unit test: External events return false for draggable

---

## Phase 7: Frontend - Calendar Component

### Step 7.1: Update CalendarView props

**File:** `zettelkasten-front/src/components/calendar/CalendarView.tsx`

Add prop: `externalEvents?: ExternalEvent[]`

### Step 7.2: Merge events in CalendarView

In component body:
```typescript
const taskEvents = tasksToCalendarEvents(tasks, timezone);
const allEvents = externalEvents
  ? mergeCalendarEvents(taskEvents, externalEvents)
  : taskEvents;
const days = populateDayEvents(grid, allEvents);
```

### Step 7.3: Update CalendarDayCell for external events

- Add `isDragDisabled={event.source === "external"}`
- Use `getEventIcon(event)` for icon display
- Keep existing color logic with `getEventColor`

### Step 7.4: Add ExternalEventDialog component

Create new dialog component for read-only external event display.

**Verify:**
- Visual: External events appear with calendar icon
- Visual: External events have indigo color
- Interaction: External events are not draggable
- Interaction: Clicking external event shows dialog
- Interaction: Tasks still draggable in mixed view

---

## Phase 8: Frontend - Settings UI

### Step 8.1: Create CalendarSubscriptions component

**File:** `zettelkasten-front/src/components/settings/CalendarSubscriptions.tsx`

- List subscribed calendars
- Add new subscription form
- Sync button per calendar
- Error display

**Verify:**
- Can add calendar subscription
- Sync button works
- Errors display correctly

### Step 8.2: Add to settings page

**File:** `zettelkasten-front/src/pages/Settings.tsx`

Add CalendarSubscriptions section.

**Verify:** Settings page renders new section.

---

## Phase 9: Integration & Testing

### Step 9.1: Wire up external events loading

In TaskPage or wherever CalendarView is used:
- Load external events when calendar view changes
- Pass to CalendarView component

### Step 9.2: End-to-end testing

- [ ] Subscribe to public Google Calendar
- [ ] Events appear in month view
- [ ] Events appear in week view
- [ ] Click event shows dialog
- [ ] Tasks still draggable
- [ ] External events not draggable
- [ ] Sync updates events
- [ ] Unsubscribe removes events

---

## Step Order Summary

1. Backend: Migration (0102)
2. Backend: ExternalEvent model
3. Backend: iCal service (parser + fetch)
4. Backend: ExternalEventService
5. Backend: API handlers + routes
6. Frontend: CalendarEvent model updates
7. Frontend: ExternalEvent model
8. Frontend: API client
9. Frontend: Calendar utilities (merge, icon)
10. Frontend: CalendarView component updates
11. Frontend: ExternalEventDialog component
12. Frontend: CalendarSubscriptions settings
13. Frontend: TaskPage integration
14. E2E testing

---

## Dependencies

- Requires `external_calendars` table from original calendar-events-design.md
- Requires existing CalendarView component
- Requires existing Task model and API

## Blocking Issues

- None identified

## Notes

- Original plan used `calendar_events` table name - changed to `external_events` for clarity
- Max 2000 events per feed prevents abuse
- All times stored in UTC
- External events are read-only in UI (can't edit imported events)
