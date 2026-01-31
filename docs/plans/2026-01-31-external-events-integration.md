# External Events Integration Design

## Overview

This document describes the integration of external calendar events (from iCal feeds) with the existing task calendar view in Zettelgarden.

**Date:** 2026-01-31
**Related Bead:** Zettelgarden-fbqy
**Status:** Design Approved

## Background

The existing `CalendarView` component displays tasks converted to calendar events. We are adding support for importing external calendar events from iCal feeds (Google Calendar, etc.). These need to display alongside tasks in a unified view.

## Key Decisions

### 1. Naming Convention

To avoid collision with the existing `CalendarEvent` type (which represents task-derived calendar events), external events use distinct naming:

| Old Name (from original plan) | New Name |
|-------------------------------|----------|
| `CalendarEvent` (external) | `ExternalEvent` |
| `calendar_events` table | `external_events` |
| `CalendarEventService` | `ExternalEventService` |
| `getCalendarEvents()` API | `getExternalEvents()` API |

**Rationale:** The existing `CalendarEvent` type (zettelkasten-front/src/models/CalendarEvent.ts) represents the conversion layer from tasks to calendar display. Reusing this name for a different concept (imported VEVENTs) would cause confusion.

### 2. Display Strategy

**Decision:** Mixed display - tasks and external events appear together in month/week views.

- External events are read-only (clicking shows details dialog)
- Tasks remain draggable and fully interactive
- Events stack vertically when they overlap
- Visual distinction through color and iconography

### 3. Visual Differentiation

**Task events** (existing behavior):
- Colored by priority (A=orange, B=yellow, C=blue, D=gray)
- Completed = green
- Due/overdue = red
- Colored left border indicates priority/status

**External events** (new):
- Indigo/purple by default (customizable per calendar subscription)
- No left border
- Calendar icon (📅) prefix
- Read-only, not draggable

## Architecture

### Data Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  External Cal   │────▶│  iCal Import      │────▶│ ExternalEvent   │
│  (Google, etc)  │     │  Service          │     │   Model         │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                                         │
                                                         ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  CalendarView   │◀────│  mergeCalendar    │◀────│  CalendarEvent  │
│  (Month/Week)   │     │  Events()         │     │   (unified)     │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                              ▲
                              │
                         ┌─────┴─────┐
                         │   Tasks   │
                         └───────────┘
```

### Frontend Models

**Extended CalendarEvent interface:**

```typescript
export type EventSource = "task" | "external";

export interface CalendarEvent {
  id: number;
  taskId?: number;              // Present for task events
  externalEventId?: number;     // Present for external events
  source: EventSource;          // NEW: Distinguish source

  title: string;
  date: Date;
  allDay: boolean;

  // Task-specific fields
  priority: string | null;
  status: string;
  isComplete: boolean;
  task?: Task;
  eventType: "scheduled" | "due" | "completed";

  // External event-specific fields
  description?: string;
  location?: string;
  externalUrl?: string;
  color?: string;               // From calendar subscription
}
```

**New ExternalEvent model:**

```typescript
export interface ExternalEvent {
  id: number;
  user_id: number;
  external_calendar_id?: number;
  title: string;
  description?: string;
  start_time: string;  // ISO 8601
  end_time: string;    // ISO 8601
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

## Implementation Changes

### Backend Changes

#### 1. Renamed Table

```sql
CREATE TABLE external_events (  -- was calendar_events
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
```

#### 2. Max Event Limit

Add safety limit to iCal parser:

```go
const MAX_EVENTS_PER_FEED = 2000

func ParseICalendar(r io.Reader) ([]ICalEvent, error) {
    // ... existing code ...
    case line == "END:VEVENT":
        if currentEvent != nil && currentEvent.UID != "" {
            if len(events) >= MAX_EVENTS_PER_FEED {
                return events, fmt.Errorf("exceeded maximum events (%d)", MAX_EVENTS_PER_FEED)
            }
            events = append(events, *currentEvent)
        }
```

#### 3. Improved Timezone Handling

Parse TZID parameter and convert to UTC:

```go
func parseICalDateTime(value string, params []string) time.Time {
    var tz string
    for _, p := range params {
        if strings.HasPrefix(p, "TZID=") {
            tz = strings.TrimPrefix(p, "TZID=")
            break
        }
    }

    // Parse and convert to UTC
    if isDate || len(value) == 8 {
        t, _ := time.Parse("20060102", value)
        return t.UTC()
    } else {
        layouts := []string{"20060102T150405Z", "20060102T150405"}
        for _, layout := range layouts {
            if t, err := time.Parse(layout, value); err == nil {
                if tz != "" {
                    loc, _ := time.LoadLocation(tz)
                    return t.In(loc).UTC()
                }
                return t.UTC()
            }
        }
    }
    return time.Time{}
}
```

### Frontend Changes

#### 1. Calendar Utilities

```typescript
// src/utils/calendar.ts

export function mergeCalendarEvents(
  taskEvents: CalendarEvent[],
  externalEvents: ExternalEvent[]
): CalendarEvent[] {
  const converted: CalendarEvent[] = externalEvents.map(ee => ({
    id: ee.id,
    externalEventId: ee.id,
    source: "external" as const,
    title: ee.title,
    date: new Date(ee.start_time),
    allDay: ee.all_day,
    priority: null,
    status: "",
    isComplete: false,
    eventType: "scheduled",
    description: ee.description,
    location: ee.location,
    externalUrl: ee.external_url,
    color: ee.color || "#6366f1",
  }));

  return [...taskEvents, ...converted];
}

export function getEventColor(event: CalendarEvent): string {
  if (event.source === "external") {
    return `bg-[${event.color}] bg-opacity-10 text-[${event.color}] border-[${event.color}]`;
  }

  // Existing task color logic unchanged
  if (event.isComplete) {
    return "bg-green-100 text-green-800 border-green-300";
  }
  if (event.eventType === "due") {
    return "bg-red-100 text-red-800 border-red-300";
  }
  switch (event.priority) {
    case "A": return "bg-orange-100 text-orange-800 border-orange-300";
    case "B": return "bg-yellow-100 text-yellow-800 border-yellow-300";
    case "C": return "bg-blue-100 text-blue-800 border-blue-300";
    case "D": return "bg-slate-100 text-slate-800 border-slate-300";
    default: return "bg-purple-100 text-purple-800 border-purple-300";
  }
}

export function getEventIcon(event: CalendarEvent): string | null {
  if (event.source === "external") {
    return "📅";
  }
  return null;
}

export function isEventDraggable(event: CalendarEvent): boolean {
  return event.source === "task";
}
```

#### 2. CalendarView Component Updates

```typescript
// src/components/calendar/CalendarView.tsx

interface CalendarViewProps {
  tasks: Task[];
  externalEvents?: ExternalEvent[];  // NEW
  currentDate: Date;
  viewMode: CalendarViewType;
  onNavigate: (direction: "prev" | "next" | "today") => void;
  // ... existing props
}

export function CalendarView({ tasks, externalEvents = [], ... }: CalendarViewProps) {
  // Convert tasks to calendar events
  const taskEvents = tasksToCalendarEvents(tasks, timezone);

  // Merge with external events
  const allEvents = mergeCalendarEvents(taskEvents, externalEvents);

  // Use allEvents for grid population
  const days = populateDayEvents(grid, allEvents);
  // ... rest of component
}
```

#### 3. CalendarDayCell Updates

External events are not draggable:

```typescript
<Draggable
  key={event.id}
  draggableId={event.source === "task" ? event.taskId.toString() : `ext-${event.id}`}
  index={index}
  isDragDisabled={event.source === "external"}
>
  {/* ... */}
</Draggable>
```

#### 4. ExternalEventDialog Component

New read-only dialog for external events:

```typescript
function ExternalEventDialog({ event, onClose }: { event: CalendarEvent; onClose: () => void }) {
  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full" onClick={e => e.stopPropagation()}>
        <div className="p-4 border-b flex justify-between items-center">
          <h3 className="text-lg font-semibold">{event.title}</h3>
          <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded">✕</button>
        </div>
        <div className="p-4 space-y-3">
          {event.description && (
            <div>
              <span className="text-sm text-gray-500">Description</span>
              <p className="mt-1">{event.description}</p>
            </div>
          )}
          {event.location && (
            <div>
              <span className="text-sm text-gray-500">Location</span>
              <p className="mt-1">{event.location}</p>
            </div>
          )}
          <div className="flex gap-2 pt-2">
            {event.externalUrl && (
              <a
                href={event.externalUrl}
                target="_blank"
                rel="noopener"
                className="px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700"
              >
                Open in Calendar
              </a>
            )}
            <button onClick={onClose} className="px-4 py-2 border rounded hover:bg-gray-50">
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
```

## API Changes

### Endpoints

```
GET    /api/user/external-calendars        # List subscriptions
POST   /api/user/external-calendars        # Subscribe to calendar
PUT    /api/user/external-calendars/:id    # Update subscription
DELETE /api/user/external-calendars/:id    # Unsubscribe
POST   /api/user/external-calendars/:id/sync  # Manual sync

GET    /api/user/external-events           # Get events (with start/end query params)
```

### Response Format

```typescript
// GET /api/user/external-events?start=2026-01-01T00:00:00Z&end=2026-01-31T23:59:59Z
{
  "events": [
    {
      "id": 1,
      "external_calendar_id": 1,
      "title": "Team Meeting",
      "description": "Weekly sync",
      "start_time": "2026-01-15T14:00:00Z",
      "end_time": "2026-01-15T15:00:00Z",
      "all_day": false,
      "location": "Conference Room A",
      "external_uid": "abc123@google.com",
      "external_url": "https://calendar.google.com/event...",
      "color": "#6366f1"
    }
  ]
}
```

## Testing Checklist

- [ ] External events display in month view alongside tasks
- [ ] External events display in week view alongside tasks
- [ ] External events have distinct visual style (color, icon)
- [ ] Clicking external event shows read-only dialog
- [ ] Tasks remain draggable in mixed view
- [ ] External events are not draggable
- [ ] External events link to source calendar
- [ ] Timezone conversion works correctly
- [ ] Max event limit prevents runaway imports
- [ ] Sync errors are displayed in settings

## Future Considerations

- **Event deletion:** Sync should soft-delete local events when removed from source
- **Authentication:** Support password-protected iCal feeds
- **Rate limiting:** Prevent abuse of external calendar servers
- **Recurrence:** Expand RRULE into individual events (deferred)
- **VTODO support:** Import tasks from iCal (separate feature)
