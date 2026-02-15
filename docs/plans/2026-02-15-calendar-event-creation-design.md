# Calendar Event Creation via CalDAV - Design Document

**Date:** 2026-02-15
**Status:** Design Approved
**Related Issues:** Calendar event creation and editing

## Overview

Enable users to create and edit calendar events directly from the Zettelgarden calendar UI, with events written back to user's external CalDAV calendars (Google Calendar, Fastmail, etc.) and synced back via existing sync infrastructure.

**Key Decision:** External calendars remain the single source of truth. No local event storage - events are created via CalDAV and then pulled back as `ExternalEvent`.

---

## Requirements

1. **Event Purpose:** Calendar events are separate from tasks - used for appointments, meetings, scheduled events
2. **Calendar Association:** Events MUST be associated with an external calendar (write-back via CalDAV)
3. **Sync:** Full bidirectional sync - create/edit/delete operations push to CalDAV, external changes sync back
4. **Event Fields (MVP):**
   - Title (required)
   - Description (optional)
   - Start time / End time
   - All-day toggle
   - Location (optional)
   - Calendar selection (user's writable calendars)
5. **User Interaction:**
   - "+" button appears on day hover
   - Double-click on a day opens event creation dialog
   - Replaces existing task creation on calendar

---

## Architecture

### Flow Diagram

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Double-click   │────▶│  EventDialog    │────▶│  POST /api/     │
│  day on calendar│     │  (create event) │     │  external-      │
└─────────────────┘     └─────────────────┘     │  calendars/{id}  │
                                                │  /events         │
                                                └────────┬─────────┘
                                                         │
                                                         ▼
                                                ┌─────────────────┐
                                                │  CalDAV PUT     │────▶ External Calendar
                                                │  (create .ics)  │     (Google, Fastmail, etc.)
                                                └────────┬─────────┘
                                                         │
                                                         ▼
                                                ┌─────────────────┐
                                                │  Trigger Sync   │
                                                │  (pull back)    │
                                                └────────┬─────────┘
                                                         │
                                                         ▼
                                                ┌─────────────────┐
                                                │  Event appears  │
                                                │  as ExternalEvent│
                                                └─────────────────┘
```

### No Database Changes Needed

Events are stored externally and pulled back as `ExternalEvent` records through the existing sync infrastructure.

---

## Backend Implementation

### New API Endpoint

**File:** `go-backend/handlers/external_events.go`

```go
// CreateEventOnCalendarRoute handles POST /api/user/external-calendars/{id}/events
func (s *Handler) CreateEventOnCalendarRoute(w http.ResponseWriter, r *http.Request) {
    userID, err := getUserID(r)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
        return
    }

    vars := mux.Vars(r)
    calendarID, err := strconv.Atoi(vars["id"])
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "INVALID_ID", "Invalid calendar ID")
        return
    }

    var req models.CreateEventRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondWithError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
        return
    }

    // Validate
    if req.Title == "" {
        respondWithError(w, http.StatusBadRequest, "MISSING_TITLE", "Title is required")
        return
    }

    svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)

    // Create event on CalDAV server
    eventUID, err := svc.CreateEventOnCalendar(calendarID, userID, req)
    if err != nil {
        log.Printf("Error creating event on calendar: %v", err)
        respondWithError(w, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
        return
    }

    // Trigger immediate sync to pull the event back
    go svc.SyncExternalCalendar(calendarID, userID)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "uid": eventUID,
        "message": "Event created successfully",
    })
}
```

### Request Models

**File:** `go-backend/models/external_events.go`

```go
// CreateEventRequest represents a request to create a new event
type CreateEventRequest struct {
    Title       string    `json:"title"`
    Description string    `json:"description"`
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    AllDay      bool      `json:"all_day"`
    Location    string    `json:"location"`
}
```

### CalDAV Service

**File:** `go-backend/services/caldav.go` (new file)

```go
package services

import (
    "fmt"
    "io"
    "time"

    "github.com/emersion/go-webdav/caldav"
)

type CalDAVService struct {
    client *caldav.Client
}

func NewCalDAVService(endpoint, username, password string) (*CalDAVService, error) {
    client, err := caldav.NewClient(
        http.DefaultClient,
        endpoint,
        username,
        password,
    )
    if err != nil {
        return nil, err
    }
    return &CalDAVService{client: client}, nil
}

// CreateEvent creates a new event on the CalDAV server
func (s *CalDAVService) CreateEvent(calendarPath string, req CreateEventRequest) (string, error) {
    // Generate UID for the event
    uid := generateUID()

    // Build iCal VEVENT
    icalData := formatEventAsICal(uid, req)

    // PUT to CalDAV server
    eventPath := fmt.Sprintf("%s/%s.ics", calendarPath, uid)
    err := s.client.PutObject(
        context.Background(),
        eventPath,
        strings.NewReader(icalData),
        &caldav.PutOptions{
            ContentType: "text/calendar",
        },
    )

    return uid, err
}

func formatEventAsICal(uid string, req CreateEventRequest) string {
    var buf strings.Builder

    buf.WriteString("BEGIN:VCALENDAR\r\n")
    buf.WriteString("VERSION:2.0\r\n")
    buf.WriteString("PRODID:-//Zettelgarden//EN\r\n")
    buf.WriteString("BEGIN:VEVENT\r\n")
    buf.WriteString(fmt.Sprintf("UID:%s\r\n", uid))
    buf.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICal(req.Title)))

    if req.Description != "" {
        buf.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICal(req.Description)))
    }

    if req.Location != "" {
        buf.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICal(req.Location)))
    }

    if req.AllDay {
        buf.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", formatICalDate(req.StartTime)))
        buf.WriteString(fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", formatICalDate(req.EndTime)))
    } else {
        buf.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICalDateTime(req.StartTime)))
        buf.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICalDateTime(req.EndTime)))
    }

    buf.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICalDateTime(time.Now())))
    buf.WriteString("END:VEVENT\r\n")
    buf.WriteString("END:VCALENDAR\r\n")

    return buf.String()
}

func generateUID() string {
    return fmt.Sprintf("%s@zettelgarden", uuid.New().String())
}

func formatICalDateTime(t time.Time) string {
    return t.UTC().Format("20060102T150405Z")
}

func formatICalDate(t time.Time) string {
    return t.Format("20060102")
}

func escapeICal(text string) string {
    text = strings.ReplaceAll(text, "\\", "\\\\")
    text = strings.ReplaceAll(text, ";", "\\;")
    text = strings.ReplaceAll(text, ",", "\\,")
    text = strings.ReplaceAll(text, "\n", "\\n")
    return text
}
```

### Integration with ExternalEventService

Update `go-backend/services/external_events.go`:

```go
// CreateEventOnCalendar creates an event via CalDAV and returns the UID
func (s *ExternalEventService) CreateEventOnCalendar(calendarID, userID int, req models.CreateEventRequest) (string, error) {
    // Get calendar details
    cal, err := s.GetCalendar(calendarID, userID)
    if err != nil {
        return "", fmt.Errorf("calendar not found: %w", err)
    }

    // Decrypt password if needed
    password := ""
    if cal.PasswordEncrypted != nil {
        password, err = s.encryptionService.Decrypt(*cal.PasswordEncrypted)
        if err != nil {
            return "", fmt.Errorf("failed to decrypt credentials: %w", err)
        }
    }

    // Create CalDAV client
    caldavSvc, err := NewCalDAVService(cal.URL, cal.Username, password)
    if err != nil {
        return "", fmt.Errorf("failed to create CalDAV client: %w", err)
    }

    // Create the event
    uid, err := caldavSvc.CreateEvent(cal.URL, req)
    if err != nil {
        return "", fmt.Errorf("failed to create event on CalDAV server: %w", err)
    }

    return uid, nil
}
```

### Route Registration

**File:** `go-backend/routes/routes.go` (or external_events route file)

```go
addProtectedRoute(r, h, "/api/user/external-calendars/{id}/events", h.CreateEventOnCalendarRoute, "POST")
```

---

## Frontend Implementation

### New API Client

**File:** `zettelkasten-front/src/api/calendarEvents.ts`

```typescript
import { apiClient, getData } from './client';

export interface CreateEventRequest {
  title: string;
  description?: string;
  start_time: string;  // ISO 8601
  end_time: string;    // ISO 8601
  all_day: boolean;
  location?: string;
}

export interface CreateEventResponse {
  uid: string;
  message: string;
}

/**
 * Create a new event on an external calendar via CalDAV
 */
export async function createEventOnCalendar(
  calendarId: number,
  event: CreateEventRequest
): Promise<CreateEventResponse> {
  return getData(apiClient.post<CreateEventResponse>(
    `/user/external-calendars/${calendarId}/events`,
    event
  ));
}
```

### Event Dialog Component

**File:** `zettelkasten-front/src/components/calendar/EventDialog.tsx`

```typescript
import React, { useState, useEffect } from 'react';
import { FaCog } from 'react-icons/fa';
import { useAuth } from '../../contexts/AuthContext';
import { ExternalCalendar } from '../../models/ExternalEvent';
import { getExternalCalendars, createEventOnCalendar } from '../../api/calendarEvents';

interface EventDialogProps {
  initialDate?: Date;
  onClose: () => void;
  onSuccess: () => void;
}

export function EventDialog({ initialDate, onClose, onSuccess }: EventDialogProps) {
  const { user } = useAuth();
  const userTimezone = user?.timezone || 'UTC';

  // Form state
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [startDate, setStartDate] = useState(initialDate || new Date());
  const [startTime, setStartTime] = useState('09:00');
  const [endDate, setEndDate] = useState(initialDate || new Date());
  const [endTime, setEndTime] = useState('10:00');
  const [allDay, setAllDay] = useState(false);
  const [location, setLocation] = useState('');

  // Calendar selection
  const [calendars, setCalendars] = useState<ExternalCalendar[]>([]);
  const [selectedCalendar, setSelectedCalendar] = useState<ExternalCalendar | null>(null);
  const [isLoadingCalendars, setIsLoadingCalendars] = useState(false);

  // Submit state
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load writable calendars
  useEffect(() => {
    async function loadCalendars() {
      setIsLoadingCalendars(true);
      try {
        const data = await getExternalCalendars();
        // Filter for calendars with credentials (writable)
        const writable = data.filter(c => c.username);
        setCalendars(writable);
        if (writable.length > 0) {
          setSelectedCalendar(writable[0]);
        }
      } catch (err) {
        console.error('Failed to load calendars:', err);
      } finally {
        setIsLoadingCalendars(false);
      }
    }
    loadCalendars();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!selectedCalendar) {
      setError('Please select a calendar');
      return;
    }

    if (!title.trim()) {
      setError('Please enter a title');
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      // Combine date and time
      const startDateTime = combineDateTime(startDate, startTime);
      const endDateTime = combineDateTime(endDate, endTime);

      await createEventOnCalendar(selectedCalendar.id, {
        title: title.trim(),
        description: description.trim() || undefined,
        start_time: startDateTime.toISOString(),
        end_time: endDateTime.toISOString(),
        all_day: allDay,
        location: location.trim() || undefined,
      });

      // Close dialog and trigger refresh
      onSuccess();
      onClose();
    } catch (err: any) {
      console.error('Failed to create event:', err);
      setError(err.response?.data?.error || 'Failed to create event. Please try again.');
    } finally {
      setIsSaving(false);
    }
  };

  const handleKeyPress = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      onClose();
    }
  };

  useEffect(() => {
    document.addEventListener('keydown', handleKeyPress);
    return () => document.removeEventListener('keydown', handleKeyPress);
  }, []);

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-lg shadow-xl max-w-md w-full"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="event-dialog-title"
      >
        <div className="p-4 border-b flex justify-between items-center">
          <h3 id="event-dialog-title" className="text-lg font-semibold">New Event</h3>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            aria-label="Close dialog"
          >
            ✕
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-md">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Title <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              autoFocus
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Calendar
            </label>
            {isLoadingCalendars ? (
              <p className="text-sm text-gray-500">Loading calendars...</p>
            ) : calendars.length === 0 ? (
              <p className="text-sm text-amber-600">
                No writable calendars found. Configure calendar credentials first.
              </p>
            ) : (
              <select
                value={selectedCalendar?.id || ''}
                onChange={(e) => {
                  const cal = calendars.find(c => c.id === parseInt(e.target.value));
                  setSelectedCalendar(cal || null);
                }}
                className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {calendars.map(cal => (
                  <option key={cal.id} value={cal.id}>
                    {cal.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Date
            </label>
            <input
              type="date"
              value={formatDateForInput(startDate)}
              onChange={(e) => {
                const newDate = new Date(e.target.value + 'T00:00:00');
                setStartDate(newDate);
                setEndDate(newDate); // Keep end date in sync initially
              }}
              className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="allDay"
              checked={allDay}
              onChange={(e) => setAllDay(e.target.checked)}
              className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-2 focus:ring-blue-500"
            />
            <label htmlFor="allDay" className="text-sm text-gray-700">
              All day
            </label>
          </div>

          {!allDay && (
            <div className="flex gap-4">
              <div className="flex-1">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Start
                </label>
                <input
                  type="time"
                  value={startTime}
                  onChange={(e) => setStartTime(e.target.value)}
                  className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex-1">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  End
                </label>
                <input
                  type="time"
                  value={endTime}
                  onChange={(e) => setEndTime(e.target.value)}
                  className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Location
            </label>
            <input
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 resize-y"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={isSaving}
              className="px-4 py-2 border rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[44px]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSaving || !selectedCalendar}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[44px] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSaving ? 'Creating...' : 'Create Event'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// Helper functions
function combineDateTime(date: Date, time: string): Date {
  const [hours, minutes] = time.split(':').map(Number);
  const result = new Date(date);
  result.setHours(hours, minutes, 0, 0);
  return result;
}

function formatDateForInput(date: Date): string {
  return date.toISOString().split('T')[0];
}
```

### CalendarView Integration

**File:** `zettelkasten-front/src/components/calendar/CalendarView.tsx`

**Add to `CalendarDayCell` component:**

```typescript
// Add "+" button in day cell
<div className="absolute top-1 right-1 opacity-0 group-hover:opacity-100 transition-opacity">
  <button
    onClick={(e) => {
      e.stopPropagation();
      onCreateEvent?.(day.date);
    }}
    className="w-6 h-6 flex items-center justify-center bg-blue-500 text-white rounded-full hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
    aria-label={`Create event on ${format(day.date, 'MMM d')}`}
  >
    <FaPlus size={12} />
  </button>
</div>
```

**Update `CalendarViewWrapper` props:**

```typescript
interface CalendarViewWrapperProps {
  // ... existing props ...
  onCreateEvent?: (date: Date) => void;
}
```

**Update `CalendarPage.tsx`:**

```typescript
const [showEventDialog, setShowEventDialog] = useState(false);
const [eventInitialDate, setEventInitialDate] = useState<Date | null>(null);

// In CalendarViewWrapper, add:
onCreateEvent={(date) => {
  setEventInitialDate(date);
  setShowEventDialog(true);
}}

// Add dialog:
{showEventDialog && (
  <EventDialog
    initialDate={eventInitialDate || undefined}
    onClose={() => setShowEventDialog(false)}
    onSuccess={() => {
      // Trigger refresh of external events
      // This could be done by setting a refresh flag or calling the API directly
      window.location.reload(); // Simple approach for now
    }}
  />
)}
```

### Styling for Hover Button

**CSS addition** (or inline styles as shown above) to make the "+" button appear on hover:

- Wrap day cell content in a group div
- Apply `group-hover:opacity-100` to the button
- Apply `opacity-0` as default state

---

## Trade-offs and Considerations

### Simplicity vs. Features
- ✅ **Chosen:** Simple direct CalDAV write, reuses existing sync
- ❌ **Not chosen:** Local storage with sync state (more complex)
- ❌ **Not chosen:** Abstract provider layer (over-engineering for CalDAV-only)

### Limitations
1. **Network dependency:** Cannot create events offline
2. **Sync delay:** Small delay between creation and appearance (sync round-trip)
3. **Calendar required:** Must have a writable external calendar configured
4. **No recurrence:** MVP doesn't support recurring events (RRULE)
5. **No attendees:** No invite/RSCP functionality
6. **Edit/Delete:** Future work - initial version only creates

### Error Handling
- CalDAV authentication failures
- Network timeouts
- Calendar not found
- Invalid credentials

---

## Future Enhancements

1. **Edit events:** Similar dialog, pre-filled with event data
2. **Delete events:** Confirmation dialog + CalDAV DELETE
3. **Drag to resize:** In week view, drag event edges to change duration
4. **Recurring events:** RRULE support in iCal generation
5. **Attendees:** VTODO/attendee support for invitations
6. **Offline support:** Local queue + sync when connection restored

---

## Testing Checklist

- [ ] Can create event via "+" button
- [ ] Can create event via double-click
- [ ] Event appears after sync (automatic or manual)
- [ ] All-day events work correctly
- [ ] Timed events have correct duration
- [ ] Calendar selector shows writable calendars
- [ ] Error handling for no writable calendars
- [ ] Error handling for CalDAV failures
- [ ] Timezone handling works correctly
- [ ] Dialog closes on Escape key
- [ ] Validation (title required)
- [ ] Multiple calendar support

---

## Dependencies

**Go:**
- `github.com/emersion/go-webdav/caldav` - CalDAV client

**Frontend:**
- Existing: `date-fns` for date formatting
- Existing: `react-icons` for FaPlus icon

---

## Implementation Order

1. **Backend:**
   - Create `CreateEventRequest` model
   - Implement `CalDAVService` with `CreateEvent`
   - Add `CreateEventOnCalendarRoute` handler
   - Register route

2. **Frontend:**
   - Create `api/calendarEvents.ts` with `createEventOnCalendar`
   - Create `EventDialog` component
   - Add "+" button to `CalendarDayCell`
   - Wire up `onCreateEvent` callback chain
   - Update `CalendarPage` to show dialog

3. **Testing:**
   - Unit tests for CalDAV service
   - Integration tests for API endpoint
   - Manual testing with real CalDAV server
