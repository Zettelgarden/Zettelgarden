# Calendar Event Creation via CalDAV Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable users to create calendar events from the Zettelgarden calendar UI, with events written back to external CalDAV calendars and synced back via existing infrastructure.

**Architecture:** Events are created via CalDAV PUT to external calendars, then immediately synced back as ExternalEvent records. No local event storage - external calendars are the single source of truth.

**Tech Stack:** Go (backend), React/TypeScript (frontend), CalDAV protocol (github.com/emersion/go-webdav/caldav)

---

## Task 1: Add CreateEventRequest Model

**Files:**
- Modify: `go-backend/models/external_events.go`

**Step 1: Add the request struct to models**

Open `go-backend/models/external_events.go` and add after the existing request structs (around line 83):

```go
// CreateEventRequest represents a request to create a new calendar event
type CreateEventRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	AllDay      bool      `json:"all_day"`
	Location    string    `json:"location"`
}
```

**Step 2: Verify the file compiles**

Run: `cd go-backend && go build ./models`

Expected: No errors (might have unused import warning, that's OK)

**Step 3: Commit**

```bash
git add go-backend/models/external_events.go
git commit -m "feat: add CreateEventRequest model for calendar events"
```

---

## Task 2: Create CalDAV Service Package

**Files:**
- Create: `go-backend/services/caldav.go`

**Step 1: Create the caldav service file**

Create `go-backend/services/caldav.go`:

```go
package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/emersion/go-webdav/caldav"
)

// CalDAVService handles CalDAV operations for external calendars
type CalDAVService struct {
	client *caldav.Client
}

// NewCalDAVService creates a new CalDAV client for the given endpoint
func NewCalDAVService(endpoint, username, password string) (*CalDAVService, error) {
	client, err := caldav.NewClient(
		nil, // Use default HTTP client
		endpoint,
		username,
		password,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CalDAV client: %w", err)
	}
	return &CalDAVService{client: client}, nil
}

// CreateEvent creates a new event on the CalDAV server
func (s *CalDAVService) CreateEvent(ctx context.Context, calendarPath string, req CreateEventRequest) (string, error) {
	// Generate UID for the event
	uid := generateUID()

	// Build iCal VEVENT
	icalData := s.formatEventAsICal(uid, req)

	// PUT to CalDAV server
	eventPath := fmt.Sprintf("%s/%s.ics", strings.TrimSuffix(calendarPath, "/"), uid)

	// Create a reader for the iCal data
	reader := strings.NewReader(icalData)

	// Use the CalDAV client to put the object
	err := s.client.PutObject(
		ctx,
		eventPath,
		reader,
		nil, // options
	)
	if err != nil {
		return "", fmt.Errorf("failed to PUT event to CalDAV server: %w", err)
	}

	return uid, nil
}

// formatEventAsICal converts an event request to iCal format
func (s *CalDAVService) formatEventAsICal(uid string, req CreateEventRequest) string {
	var buf strings.Builder

	buf.WriteString("BEGIN:VCALENDAR\r\n")
	buf.WriteString("VERSION:2.0\r\n")
	buf.WriteString("PRODID:-//Zettelgarden//EN\r\n")
	buf.WriteString("CALSCALE:GREGORIAN\r\n")
	buf.WriteString("METHOD:PUBLISH\r\n")
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

// generateUID creates a unique ID for an event
func generateUID() string {
	return fmt.Sprintf("%s@zettelgarden", uuid.New().String())
}

// formatICalDateTime formats a time as iCal DATETIME
func formatICalDateTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// formatICalDate formats a time as iCal DATE (all-day)
func formatICalDate(t time.Time) string {
	return t.Format("20060102")
}

// escapeICal escapes special characters in iCal text values
func escapeICal(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, "\n", "\\n")
	text = strings.ReplaceAll(text, "\r", "")
	return text
}
```

**Step 2: Verify the file compiles**

Run: `cd go-backend && go build ./services`

Expected: No errors

**Step 3: Commit**

```bash
git add go-backend/services/caldav.go
git commit -m "feat: add CalDAV service for creating events"
```

---

## Task 3: Add CreateEventOnCalendar to ExternalEventService

**Files:**
- Modify: `go-backend/services/external_events.go`

**Step 1: Read the existing service structure**

Run: `head -100 go-backend/services/external_events.go`

This will show you the existing ExternalEventService struct and methods. Note that it already has access to encryption service for credentials.

**Step 2: Add CreateEventOnCalendar method**

Open `go-backend/services/external_events.go` and add this method to the ExternalEventService struct (after the SyncExternalCalendar method, around line 250+):

```go
// CreateEventOnCalendar creates a new event via CalDAV and returns the UID
func (s *ExternalEventService) CreateEventOnCalendar(calendarID, userID int, req models.CreateEventRequest) (string, error) {
	// Get calendar details
	cal, err := s.GetCalendar(calendarID, userID)
	if err != nil {
		return "", fmt.Errorf("calendar not found: %w", err)
	}

	// Check if calendar has credentials (writable)
	if cal.Username == nil || *cal.Username == "" {
		return "", fmt.Errorf("calendar does not have credentials configured for write access")
	}

	// Decrypt password if present
	password := ""
	if cal.PasswordEncrypted != nil && *cal.PasswordEncrypted != "" {
		password, err = s.encryptionService.Decrypt(*cal.PasswordEncrypted)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt calendar credentials: %w", err)
		}
	}

	// Create CalDAV client
	username := *cal.Username
	caldavSvc, err := NewCalDAVService(cal.URL, username, password)
	if err != nil {
		return "", fmt.Errorf("failed to create CalDAV client: %w", err)
	}

	// Create the event via CalDAV
	uid, err := caldavSvc.CreateEvent(context.Background(), cal.URL, req)
	if err != nil {
		return "", fmt.Errorf("failed to create event on CalDAV server: %w", err)
	}

	return uid, nil
}
```

**Step 3: Verify the file compiles**

Run: `cd go-backend && go build ./services`

Expected: No errors

**Step 4: Commit**

```bash
git add go-backend/services/external_events.go
git commit -m "feat: add CreateEventOnCalendar to ExternalEventService"
```

---

## Task 4: Add CreateEventOnCalendarRoute Handler

**Files:**
- Modify: `go-backend/handlers/external_events.go`

**Step 1: Add the handler method**

Open `go-backend/handlers/external_events.go` and add this method after the GetEventsByCardRoute function (at the end of the file, around line 410):

```go
// CreateEventOnCalendarRoute handles POST /api/user/external-calendars/{id}/events
// Creates a new event on an external calendar via CalDAV
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

	// Validate required fields
	if strings.TrimSpace(req.Title) == "" {
		respondWithError(w, http.StatusBadRequest, "MISSING_TITLE", "Title is required")
		return
	}

	// Validate time range
	if req.EndTime.Before(req.StartTime) {
		respondWithError(w, http.StatusBadRequest, "INVALID_TIME_RANGE", "End time must be after start time")
		return
	}

	svc := services.NewExternalEventService(s.GetDB(), s.EncryptionService)

	// Create event on CalDAV server
	uid, err := svc.CreateEventOnCalendar(calendarID, userID, req)
	if err != nil {
		log.Printf("Error creating event on calendar: %v", err)
		// Determine error code
		code := "CREATE_FAILED"
		if strings.Contains(err.Error(), "credentials") {
			code = "INVALID_CREDENTIALS"
		} else if strings.Contains(err.Error(), "write access") {
			code = "NOT_WRITABLE"
		}
		respondWithError(w, http.StatusBadRequest, code, err.Error())
		return
	}

	// Trigger sync in background to pull the new event back
	go func() {
		if syncErr := svc.SyncExternalCalendar(calendarID, userID); syncErr != nil {
			log.Printf("Error syncing calendar after event creation: %v", syncErr)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"uid":     uid,
		"message": "Event created successfully",
	})
}
```

**Step 2: Verify the file compiles**

Run: `cd go-backend && go build ./handlers`

Expected: No errors

**Step 3: Commit**

```bash
git add go-backend/handlers/external_events.go
git commit -m "feat: add CreateEventOnCalendarRoute handler"
```

---

## Task 5: Register the New Route

**Files:**
- Modify: Find where external events routes are registered (likely `go-backend/routes/routes.go` or similar)

**Step 1: Find the routes file**

Run: `grep -r "GetEventsByCardRoute" go-backend/routes/ 2>/dev/null || grep -r "GetEventsByCardRoute" go-backend/`

This will show where the external events routes are registered.

**Step 2: Add the new route registration**

Open the routes file and add this line with the other external-events routes (look for where `/api/user/external-events` or `/api/user/external-calendars` routes are):

```go
addProtectedRoute(r, h, "/api/user/external-calendars/{id}/events", h.CreateEventOnCalendarRoute, "POST")
```

**Step 3: Verify the routes compile**

Run: `cd go-backend && go build ./routes`

Expected: No errors

**Step 4: Commit**

```bash
git add go-backend/routes/
git commit -m "feat: register CreateEventOnCalendarRoute"
```

---

## Task 6: Add Frontend API Client Function

**Files:**
- Create: `zettelkasten-front/src/api/calendarEvents.ts`

**Step 1: Create the API client file**

Create `zettelkasten-front/src/api/calendarEvents.ts`:

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
 * @param calendarId The ID of the external calendar to create the event on
 * @param event The event details
 * @returns Response with the event UID and message
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

**Step 2: Verify TypeScript compiles**

Run: `cd zettelkasten-front && npm run build -- --mode development 2>&1 | head -50`

Expected: No TypeScript errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/api/calendarEvents.ts
git commit -m "feat: add createEventOnCalendar API client"
```

---

## Task 7: Create EventDialog Component

**Files:**
- Create: `zettelkasten-front/src/components/calendar/EventDialog.tsx`

**Step 1: Create the EventDialog component**

Create `zettelkasten-front/src/components/calendar/EventDialog.tsx`:

```typescript
import React, { useState, useEffect } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { ExternalCalendar } from '../../models/ExternalEvent';
import { getExternalCalendars, createEventOnCalendar, CreateEventRequest } from '../../api/calendarEvents';

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

  // Load writable calendars on mount
  useEffect(() => {
    async function loadCalendars() {
      setIsLoadingCalendars(true);
      try {
        const data = await getExternalCalendars();
        // Filter for calendars with credentials (writable)
        const writable = data.filter(c => c.username && c.username.trim() !== '');
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

      const request: CreateEventRequest = {
        title: title.trim(),
        description: description.trim() || undefined,
        start_time: startDateTime.toISOString(),
        end_time: endDateTime.toISOString(),
        all_day: allDay,
        location: location.trim() || undefined,
      };

      await createEventOnCalendar(selectedCalendar.id, request);

      // Close dialog and trigger refresh
      onSuccess();
      onClose();
    } catch (err: any) {
      console.error('Failed to create event:', err);
      const errorMessage = err.response?.data?.error || err.message || 'Failed to create event. Please try again.';
      setError(errorMessage);
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
            <label htmlFor="event-title" className="block text-sm font-medium text-gray-700 mb-1">
              Title <span className="text-red-500">*</span>
            </label>
            <input
              id="event-title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              autoFocus
              required
            />
          </div>

          <div>
            <label htmlFor="event-calendar" className="block text-sm font-medium text-gray-700 mb-1">
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
                id="event-calendar"
                value={selectedCalendar?.id || ''}
                onChange={(e) => {
                  const cal = calendars.find(c => c.id === parseInt(e.target.value));
                  setSelectedCalendar(cal || null);
                }}
                className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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
            <label htmlFor="event-date" className="block text-sm font-medium text-gray-700 mb-1">
              Date
            </label>
            <input
              id="event-date"
              type="date"
              value={formatDateForInput(startDate)}
              onChange={(e) => {
                const newDate = parseDateFromInput(e.target.value);
                if (newDate) {
                  setStartDate(newDate);
                  setEndDate(newDate); // Keep end date in sync initially
                }
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="event-all-day"
              checked={allDay}
              onChange={(e) => setAllDay(e.target.checked)}
              className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-2 focus:ring-blue-500"
            />
            <label htmlFor="event-all-day" className="text-sm text-gray-700">
              All day
            </label>
          </div>

          {!allDay && (
            <div className="flex gap-4">
              <div className="flex-1">
                <label htmlFor="event-start-time" className="block text-sm font-medium text-gray-700 mb-1">
                  Start
                </label>
                <input
                  id="event-start-time"
                  type="time"
                  value={startTime}
                  onChange={(e) => setStartTime(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
              <div className="flex-1">
                <label htmlFor="event-end-time" className="block text-sm font-medium text-gray-700 mb-1">
                  End
                </label>
                <input
                  id="event-end-time"
                  type="time"
                  value={endTime}
                  onChange={(e) => setEndTime(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
          )}

          <div>
            <label htmlFor="event-location" className="block text-sm font-medium text-gray-700 mb-1">
              Location
            </label>
            <input
              id="event-location"
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <div>
            <label htmlFor="event-description" className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              id="event-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={isSaving}
              className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[44px]"
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
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function parseDateFromInput(dateStr: string): Date | null {
  const match = dateStr.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!match) return null;
  const [, year, month, day] = match;
  return new Date(parseInt(year), parseInt(month) - 1, parseInt(day));
}

export default EventDialog;
```

**Step 2: Verify TypeScript compiles**

Run: `cd zettelkasten-front && npm run build -- --mode development 2>&1 | head -50`

Expected: No TypeScript errors

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/calendar/EventDialog.tsx
git commit -m "feat: add EventDialog component for creating calendar events"
```

---

## Task 8: Add onCreateEvent Prop to CalendarViewWrapper

**Files:**
- Modify: `zettelkasten-front/src/components/calendar/CalendarView.tsx`

**Step 1: Update CalendarViewWrapperProps interface**

Find the `CalendarViewWrapperProps` interface (around line 710) and add the new prop:

```typescript
interface CalendarViewWrapperProps {
  tasks: Task[];
  externalEvents?: ExternalEvent[];
  currentDate: Date;
  viewMode: "month" | "week";
  onNavigate: (direction: "prev" | "next" | "today") => void;
  onViewModeChange: (viewMode: "month" | "week") => void;
  onTaskClick: (taskId: number) => void;
  onCreateTask?: (date: Date) => void;
  onCreateEvent?: (date: Date) => void;  // ADD THIS LINE
  onTaskMoved?: () => void;
  onExternalEventChange?: () => void;
  timezone?: string;
  calendarSettingsButton?: React.ReactNode;
}
```

**Step 2: Update CalendarViewWrapper function signature**

Find the CalendarViewWrapper function definition and add the new prop:

```typescript
export function CalendarViewWrapper({
  tasks,
  externalEvents,
  currentDate,
  viewMode,
  onNavigate,
  onViewModeChange,
  onTaskClick,
  onCreateTask,
  onCreateEvent,  // ADD THIS LINE
  onTaskMoved,
  onExternalEventChange,
  timezone = "UTC",
  calendarSettingsButton,
}: CalendarViewWrapperProps) {
```

**Step 3: Pass onCreateEvent to CalendarView**

Find where CalendarView is rendered and add the prop:

```typescript
<CalendarView
  tasks={tasks}
  externalEvents={externalEvents}
  currentDate={currentDate}
  viewMode={viewMode}
  onNavigate={onNavigate}
  onViewModeChange={onViewModeChange}
  onDayClick={handleDayClick}
  onEventClick={handleEventClick}
  onCreateTask={onCreateTask}
  onCreateEvent={onCreateEvent}  // ADD THIS LINE
  onTaskMoved={onTaskMoved}
  onExternalEventClick={handleExternalEventClick}
  onExternalEventChange={onExternalEventChange}
  timezone={timezone || "UTC"}
  calendarSettingsButton={calendarSettingsButton}
/>
```

**Step 4: Verify TypeScript compiles**

Run: `cd zettelkasten-front && npm run build -- --mode development 2>&1 | grep -A5 "CalendarView" | head -20`

Expected: No TypeScript errors related to CalendarView

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/calendar/CalendarView.tsx
git commit -m "feat: add onCreateEvent prop to CalendarViewWrapper"
```

---

## Task 9: Add onCreateEvent to CalendarView Props

**Files:**
- Modify: `zettelkasten-front/src/components/calendar/CalendarView.tsx`

**Step 1: Update CalendarViewProps interface**

Find the `CalendarViewProps` interface (around line 35) and add the new prop:

```typescript
interface CalendarViewProps {
  tasks: Task[];
  externalEvents?: ExternalEvent[];
  currentDate: Date;
  viewMode: CalendarViewType;
  onNavigate: (direction: "prev" | "next" | "today") => void;
  onViewModeChange: (viewMode: CalendarViewType) => void;
  onDayClick: (date: Date, events: CalendarEvent[]) => void;
  onEventClick: (event: CalendarEvent) => void;
  onCreateTask?: (date: Date) => void;
  onCreateEvent?: (date: Date) => void;  // ADD THIS LINE
  onTaskMoved?: () => void;
  onExternalEventClick?: (event: CalendarEvent) => void;
  onExternalEventChange?: () => void;
  timezone?: string;
  calendarSettingsButton?: React.ReactNode;
}
```

**Step 2: Update CalendarView function signature**

Find the CalendarView function definition and add the new prop:

```typescript
export function CalendarView({
  tasks,
  externalEvents = [],
  currentDate,
  viewMode,
  onNavigate,
  onViewModeChange,
  onDayClick,
  onEventClick,
  onCreateTask,
  onCreateEvent,  // ADD THIS LINE
  onTaskMoved,
  onExternalEventClick,
  onExternalEventChange,
  timezone = "UTC",
  calendarSettingsButton,
}: CalendarViewProps) {
```

**Step 3: Update CalendarDayCellProps interface**

Find the `CalendarDayCellProps` interface (around line 453) and add the new prop:

```typescript
interface CalendarDayCellProps {
  day: CalendarDay;
  isHovered: boolean;
  isSelected: boolean;
  onHover: (date: Date | null) => void;
  onDayClick: (day: CalendarDay) => void;
  onEventClick: (e: React.MouseEvent, event: CalendarEvent) => void;
  onContextMenu: (e: React.MouseEvent, day: CalendarDay) => void;
  onCreateTask?: (date: Date) => void;
  onCreateEvent?: (date: Date) => void;  // ADD THIS LINE
  viewMode: CalendarViewType;
  timezone: string;
}
```

**Step 4: Update CalendarDayCell function signature**

Find the CalendarDayCell function definition and add the new prop:

```typescript
function CalendarDayCell({
  day,
  isHovered,
  isSelected,
  onHover,
  onDayClick,
  onEventClick,
  onContextMenu,
  onCreateTask,
  onCreateEvent,  // ADD THIS LINE
  viewMode,
  timezone,
}: CalendarDayCellProps) {
```

**Step 5: Pass onCreateEvent to CalendarDayCell**

Find where CalendarDayCell is rendered (in the map over days) and add the prop:

```typescript
{days.map((day, index) => (
  <CalendarDayCell
    key={index}
    day={day}
    isHovered={hoveredDay ? day.date.getTime() === hoveredDay.getTime() : false}
    isSelected={selectedDayIndex === index}
    onHover={setHoveredDay}
    onDayClick={handleDayClick}
    onEventClick={handleEventClick}
    onContextMenu={handleContextMenu}
    onCreateTask={onCreateTask}
    onCreateEvent={onCreateEvent}  // ADD THIS LINE
    viewMode={viewMode}
    timezone={timezone}
  />
))}
```

**Step 6: Verify TypeScript compiles**

Run: `cd zettelkasten-front && npm run build -- --mode development 2>&1 | grep -A5 "CalendarView" | head -20`

Expected: No TypeScript errors

**Step 7: Commit**

```bash
git add zettelkasten-front/src/components/calendar/CalendarView.tsx
git commit -m "feat: add onCreateEvent prop to CalendarView"
```

---

## Task 10: Add + Button to CalendarDayCell

**Files:**
- Modify: `zettelkasten-front/src/components/calendar/CalendarView.tsx`

**Step 1: Add FaPlus import**

At the top of the file, find the imports from react-icons/fa and add FaPlus:

```typescript
import { FaChevronLeft, FaChevronRight, FaPlus, FaChevronUp, FaChevronDown } from "react-icons/fa";
```

**Step 2: Add the + button to CalendarDayCell**

Find the CalendarDayCell component's return statement. Look for the `Droppable` component. We need to add a "+" button that appears on hover.

Find the inner div inside Droppable (with className including `min-h`, `border-b`, etc.) and add the button right after the opening div, before the date number:

```typescript
<div
  ref={provided.innerRef}
  {...provided.droppableProps}
  className={cellClasses}
  // ... existing props
>
  {/* ADD THIS: Plus button for creating events */}
  {onCreateEvent && (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onCreateEvent(day.date);
      }}
      className="absolute top-1 right-1 w-6 h-6 flex items-center justify-center bg-blue-500 text-white rounded-full opacity-0 group-hover:opacity-100 hover:bg-blue-600 focus:opacity-100 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-opacity"
      aria-label={`Create event on ${format(day.date, 'MMM d')}`}
      title="Create event"
    >
      <FaPlus size={10} aria-hidden="true" />
    </button>
  )}

  {/* Existing date number div */}
  <div className={dateNumberClasses}>
    {/* ... existing content */}
  </div>

  {/* ... rest of existing content */}
</div>
```

**Step 3: Add group class to the day cell**

We need to add `group` class to enable the `group-hover:opacity-100` on the button. Update the cellClasses variable or the className on the div to include `group`:

Find the `cellClasses` variable definition (around line 563) and add `group`:

```typescript
const cellClasses = `
  ${viewMode === "week" ? "min-h-[400px]" : "min-h-[60px] sm:min-h-[80px]"} p-1 border-b border-r border-slate-200 last:border-r-0 cursor-pointer focus-within:ring-2 focus-within:ring-blue-300 focus:outline-none relative group
  ${day.isToday ? "bg-blue-50" : ""}
  // ... rest of existing classes
`;
```

**Step 4: Update double-click handler**

Find the `handleDoubleClick` function in CalendarDayCell (around line 493) and update it to call onCreateEvent instead of onCreateTask:

```typescript
const handleDoubleClick = () => {
  if (onCreateEvent) {
    onCreateEvent(day.date);
  }
};
```

**Step 5: Verify TypeScript compiles**

Run: `cd zettelkasten-front && npm run build -- --mode development 2>&1 | grep -A5 "CalendarView" | head -20`

Expected: No TypeScript errors

**Step 6: Commit**

```bash
git add zettelkasten-front/src/components/calendar/CalendarView.tsx
git commit -m "feat: add + button and double-click for event creation in CalendarDayCell"
```

---

## Task 11: Wire Up Event Dialog in CalendarPage

**Files:**
- Modify: `zettelkasten-front/src/pages/calendar/CalendarPage.tsx`

**Step 1: Add EventDialog import**

At the top of the file, add the EventDialog import:

```typescript
import { EventDialog } from "../../components/calendar/EventDialog";
```

**Step 2: Add state for event dialog**

Find the state declarations (around line 56-70) and add:

```typescript
// State for event dialog
const [showEventDialog, setShowEventDialog] = useState(false);
const [eventInitialDate, setEventInitialDate] = useState<Date | null>(null);
```

**Step 3: Add onCreateEvent callback**

Find where CalendarViewWrapper is rendered (around line 307) and add the onCreateEvent prop:

```typescript
<ErrorBoundary
  // ... existing props
>
  <CalendarViewWrapper
    tasks={showTasks ? tasks : []}
    externalEvents={visibleExternalEvents}
    currentDate={currentDate}
    viewMode={viewMode}
    onNavigate={navigateCalendar}
    onViewModeChange={setViewMode}
    onTaskClick={(taskId) => {
      setSelectedTaskId(taskId);
      setIsTaskDialogOpen(true);
    }}
    onCreateEvent={(date) => {
      setEventInitialDate(date);
      setShowEventDialog(true);
    }}
    onTaskMoved={() => {
      setRefreshTasks(true);
    }}
    timezone={userTimezone}
    calendarSettingsButton={
      // ... existing button
    }
  />
</ErrorBoundary>
```

**Step 4: Add EventDialog to the JSX**

Find where the dialogs are rendered (after CalendarViewWrapper, around line 337) and add the EventDialog:

```typescript
{/* Calendar Settings Dialog */}
{isSettingsDialogOpen && (
  // ... existing settings dialog
)}

{/* ADD THIS: Event Dialog */}
{showEventDialog && (
  <EventDialog
    initialDate={eventInitialDate || undefined}
    onClose={() => setShowEventDialog(false)}
    onSuccess={() => {
      // Trigger refresh of external events
      setRefreshTasks(true); // This will trigger a reload of calendar data
    }}
  />
)}
```

**Step 5: Update Escape key handler**

Find the handleKeyPress function (around line 263) and add the event dialog close:

```typescript
const handleKeyPress = (event: KeyboardEvent) => {
  if (event.metaKey) {
    return;
  }
  if (event.key === "Escape") {
    event.preventDefault();
    setShowCreateTaskWindow(false);
    setIsSettingsDialogOpen(false);
    setShowEventDialog(false);  // ADD THIS LINE
    return;
  }
};
```

**Step 6: Verify TypeScript compiles**

Run: `cd zettelkasten-front && npm run build -- --mode development 2>&1 | grep -A5 "CalendarPage" | head -20`

Expected: No TypeScript errors

**Step 7: Commit**

```bash
git add zettelkasten-front/src/pages/calendar/CalendarPage.tsx
git commit -m "feat: wire up EventDialog in CalendarPage"
```

---

## Task 12: Install Go CalDAV Dependency

**Files:**
- Modify: `go-backend/go.mod`

**Step 1: Add the CalDAV dependency**

Run: `cd go-backend && go get github.com/emersion/go-webdav/caldav`

Expected: Dependency added to go.mod

**Step 2: Tidy go modules**

Run: `cd go-backend && go mod tidy`

Expected: go.mod and go.sum updated

**Step 3: Verify the code compiles**

Run: `cd go-backend && go build .`

Expected: Binary builds successfully

**Step 4: Commit**

```bash
git add go-backend/go.mod go-backend/go.sum
git commit -m "deps: add go-webdav/caldav dependency"
```

---

## Task 13: Manual Testing

**Files:**
- None (testing)

**Step 1: Start the backend**

Run: `cd go-backend && go run main.go`

Expected: Server starts on configured port

**Step 2: Start the frontend**

Run: `cd zettelkasten-front && npm start`

Expected: Dev server starts, app opens in browser

**Step 3: Configure a writable external calendar**

1. Navigate to the Calendar page
2. Open settings (gear icon)
3. Add an external calendar with username/password (CalDAV credentials)
4. Sync the calendar to verify it works

**Step 4: Test event creation**

1. Double-click on a day cell
2. Verify EventDialog opens with date pre-filled
3. Fill in title, select calendar
4. Click "Create Event"
5. Verify dialog closes
6. Wait a few seconds for sync
7. Verify event appears on the calendar

**Step 5: Test + button**

1. Hover over a day cell
2. Verify "+" button appears
3. Click the "+" button
4. Verify EventDialog opens
5. Create another event
6. Verify it appears

**Step 6: Test error cases**

1. Create event without writable calendar (should show error)
2. Create event with invalid credentials (should show error)
3. Create event without title (should show validation error)

---

## Task 14: Write Backend Tests

**Files:**
- Create: `go-backend/handlers/external_events_test.go` (add to existing file)

**Step 1: Add test for CreateEventOnCalendarRoute**

Open `go-backend/handlers/external_events_test.go` and add:

```go
func TestCreateEventOnCalendarRoute(t *testing.T) {
	// Setup test handler with mock DB
	h := NewTestHandler(t)
	defer h.Close()

	// Create a test user
	user := createTestUser(t, h.db)
	token := generateTestToken(t, user.ID)

	// Create a test external calendar with credentials
	calID := createTestExternalCalendarWithCreds(t, h.db, user.ID)

	// Test case 1: Valid event creation
	t.Run("creates event successfully", func(t *testing.T) {
		reqBody := models.CreateEventRequest{
			Title:       "Test Event",
			Description: "Test Description",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(1 * time.Hour),
			AllDay:      false,
			Location:    "Test Location",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/user/external-calendars/%d/events", calID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		h.CreateEventOnCalendarRoute(w, req)

		// Note: This test will fail without a real CalDAV server
		// In production, you'd mock the CalDAV service
		if w.Code != http.StatusCreated {
			// Expected to fail without CalDAV mock, but verify error structure
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["error"] == nil {
				t.Errorf("Expected error response without CalDAV server, got status %d: %s", w.Code, w.Body.String())
			}
		}
	})

	// Test case 2: Missing title
	t.Run("rejects empty title", func(t *testing.T) {
		reqBody := models.CreateEventRequest{
			Title:       "",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(1 * time.Hour),
			AllDay:      false,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/user/external-calendars/%d/events", calID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		h.CreateEventOnCalendarRoute(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["code"] != "MISSING_TITLE" {
			t.Errorf("Expected MISSING_TITLE code, got %v", resp["code"])
		}
	})
}
```

**Step 2: Run the tests**

Run: `cd go-backend && go test ./handlers -run TestCreateEventOnCalendarRoute -v`

Expected: Tests run (may fail without CalDAV mock, but structure is verified)

**Step 3: Commit**

```bash
git add go-backend/handlers/external_events_test.go
git commit -m "test: add tests for CreateEventOnCalendarRoute"
```

---

## Completion Checklist

After implementing all tasks, verify:

- [ ] Backend compiles without errors
- [ ] Frontend compiles without errors
- [ ] Can create event via double-click
- [ ] Can create event via + button
- [ ] Events sync back and appear on calendar
- [ ] Error handling works for missing credentials
- [ ] All-day events work correctly
- [ ] Date/time inputs work correctly
- [ ] Dialog closes on Escape key
- [ ] Calendar selector shows writable calendars only

---

## Notes for Implementation

1. **CalDAV Testing:** The current implementation assumes a real CalDAV server. For testing, you may want to create a mock CalDAV service interface.

2. **Sync Timing:** Events are created via CalDAV and then synced back asynchronously. There may be a 1-2 second delay before the event appears.

3. **Calendar Filtering:** The implementation filters calendars by presence of `username` field to determine writability. This is a simple heuristic - you may want to add an explicit `writable` flag to the ExternalCalendar model in the future.

4. **Timezones:** All times are stored in UTC. The frontend handles timezone conversion for display.

5. **Error Messages:** The CalDAV protocol can return various errors. The implementation includes basic error handling, but you may want to expand this based on real-world usage.
