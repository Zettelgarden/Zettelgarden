# Calendar Events Feature - Implementation Plan

## Overview
This document describes the implementation of importing external calendar events (VEVENT) into Zettelgarden and displaying them in a day/time view alongside tasks.

**Epic:** Zettelgarden-fbqy
**Status:** Planning Phase

## Architecture Overview

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  External Cal   │────▶│  iCal Import      │────▶│ Calendar Events │
│  (Google, etc)  │     │  Service          │     │   Table         │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                                         │
                                                         ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Day View UI    │◀────│  GET /api/       │◀────│  CalendarEvent  │
│  (Time Slots)   │     │  calendar-events │     │   Model         │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

## Phase 1: Database Schema

### 1.1 Calendar Events Table

```sql
-- Migration: 0102-add-calendar-events.sql
CREATE TABLE calendar_events (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_calendar_id INTEGER REFERENCES external_calendars(id) ON DELETE SET NULL,

    -- Event details
    title TEXT NOT NULL,
    description TEXT,

    -- Timing
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    all_day BOOLEAN DEFAULT FALSE,

    -- Location
    location TEXT,

    -- External sync tracking
    external_uid TEXT,  -- UID from iCal feed for deduplication
    external_url TEXT,  -- URL of the specific event

    -- Recurrence (stored for reference, expansion not in initial scope)
    recurrence_rule TEXT,

    -- Metadata
    color TEXT,         -- Hex color for display
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_synced_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    UNIQUE(user_id, external_uid)  -- Prevent duplicate imports
);

CREATE INDEX idx_calendar_events_user_time ON calendar_events(user_id, start_time, end_time);
CREATE INDEX idx_calendar_events_external_calendar ON calendar_events(external_calendar_id);
COMMENT ON TABLE calendar_events IS 'Imported calendar events from external iCal feeds';
```

### 1.2 External Calendars Table

```sql
-- Migration: 0103-add-external-calendars.sql
CREATE TABLE external_calendars (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Subscription details
    name TEXT NOT NULL,
    url TEXT NOT NULL,

    -- Sync settings
    sync_enabled BOOLEAN DEFAULT TRUE,
    sync_interval_hours INTEGER DEFAULT 1,  -- For background polling

    -- Display
    color TEXT DEFAULT '#3b82f6',  -- Default blue

    -- Metadata
    last_synced_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    UNIQUE(user_id, url)
);

CREATE INDEX idx_external_calendars_user ON external_calendars(user_id);
COMMENT ON TABLE external_calendars IS 'External calendar subscriptions for importing events';
```

### 1.3 Backend Models

**go-backend/models/calendar_events.go:**
```go
package models

import "time"

type CalendarEvent struct {
    ID                  int        `json:"id"`
    UserID              int        `json:"user_id"`
    ExternalCalendarID  *int       `json:"external_calendar_id,omitempty"`
    Title               string     `json:"title"`
    Description         *string    `json:"description,omitempty"`
    StartTime           time.Time  `json:"start_time"`
    EndTime             time.Time  `json:"end_time"`
    AllDay              bool       `json:"all_day"`
    Location            *string    `json:"location,omitempty"`
    ExternalUID         *string    `json:"external_uid,omitempty"`
    ExternalURL         *string    `json:"external_url,omitempty"`
    RecurrenceRule      *string    `json:"recurrence_rule,omitempty"`
    Color               *string    `json:"color,omitempty"`
    CreatedAt           time.Time  `json:"created_at"`
    UpdatedAt           time.Time  `json:"updated_at"`
    LastSyncedAt        *time.Time `json:"last_synced_at,omitempty"`
}

type ExternalCalendar struct {
    ID                 int        `json:"id"`
    UserID             int        `json:"user_id"`
    Name               string     `json:"name"`
    URL                string     `json:"url"`
    SyncEnabled        bool       `json:"sync_enabled"`
    SyncIntervalHours  int        `json:"sync_interval_hours"`
    Color              string     `json:"color"`
    LastSyncedAt       *time.Time `json:"last_synced_at,omitempty"`
    LastError          *string    `json:"last_error,omitempty"`
    CreatedAt          time.Time  `json:"created_at"`
    UpdatedAt          time.Time  `json:"updated_at"`
}

type CreateExternalCalendarRequest struct {
    Name              string `json:"name"`
    URL               string `json:"url"`
    Color             string `json:"color"`
}

type UpdateExternalCalendarRequest struct {
    Name              *string `json:"name,omitempty"`
    URL               *string `json:"url,omitempty"`
    Color             *string `json:"color,omitempty"`
    SyncEnabled       *bool   `json:"sync_enabled,omitempty"`
    SyncIntervalHours *int    `json:"sync_interval_hours,omitempty"`
}
```

**zettelkasten-front/src/models/CalendarEvent.ts:**
```typescript
export interface CalendarEvent {
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

export interface ExternalCalendar {
  id: number;
  user_id: number;
  name: string;
  url: string;
  sync_enabled: boolean;
  sync_interval_hours: number;
  color: string;
  last_synced_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateExternalCalendarRequest {
  name: string;
  url: string;
  color?: string;
}

export interface UpdateExternalCalendarRequest {
  name?: string;
  url?: string;
  color?: string;
  sync_enabled?: boolean;
  sync_interval_hours?: number;
}
```

## Phase 2: iCal Import Service

### 2.1 iCal Parser

The iCal parser will:
1. Fetch the URL (with timeout)
2. Parse iCal format (BEGIN:VCALENDAR ... END:VCALENDAR)
3. Extract VEVENT components
4. Convert to CalendarEvent models

**go-backend/services/ical.go:**
```go
package services

import (
    "bufio"
    "fmt"
    "io"
    "net/http"
    "regexp"
    "strings"
    "time"
)

// ICalEvent represents a parsed VEVENT from iCal format
type ICalEvent struct {
    UID         string
    DTStart     time.Time
    DTEnd       time.Time
    Summary     string
    Description string
    Location    string
    AllDay      bool
    RecurrenceRule string
    URL         string
}

// ParseICalendar parses an iCal feed and returns VEVENT components
func ParseICalendar(r io.Reader) ([]ICalEvent, error) {
    scanner := bufio.NewScanner(r)
    var events []ICalEvent
    var currentEvent *ICalEvent
    var inVEVENT bool

    for scanner.Scan() {
        line := scanner.Text()
        line = strings.TrimSpace(line)

        switch {
        case line == "BEGIN:VEVENT":
            inVEVENT = true
            currentEvent = &ICalEvent{}

        case line == "END:VEVENT":
            if currentEvent != nil && currentEvent.UID != "" {
                events = append(events, *currentEvent)
            }
            inVEVENT = false
            currentEvent = nil

        case inVEVENT && currentEvent != nil:
            parseEventProperty(currentEvent, line)
        }
    }

    return events, scanner.Err()
}

func parseEventProperty(event *ICalEvent, line string) {
    parts := strings.SplitN(line, ":", 2)
    if len(parts) < 2 {
        return
    }

    key := parts[0]
    value := parts[1]

    // Handle parameters (e.g., DTSTART;VALUE=DATE:20230101)
    keyParts := strings.Split(key, ";")
    key = keyParts[0]

    switch key {
    case "UID":
        event.UID = value
    case "DTSTART":
        event.DTStart = parseICalDateTime(value, keyParts)
    case "DTEND":
        event.DTEnd = parseICalDateTime(value, keyParts)
    case "SUMMARY":
        event.Summary = unescapeICalText(value)
    case "DESCRIPTION":
        event.Description = unescapeICalText(value)
    case "LOCATION":
        event.Location = unescapeICalText(value)
    case "RRULE":
        event.RecurrenceRule = value
    case "URL":
        event.URL = value
    }
}

func parseICalDateTime(value string, params []string) time.Time {
    // Check if it's a DATE (all-day) or DATETIME
    isDate := false
    for _, p := range params {
        if strings.Contains(p, "VALUE=DATE") {
            isDate = true
            break
        }
    }

    // Parse based on format
    if isDate || len(value) == 8 {
        // DATE format: 20230101
        t, _ := time.Parse("20060102", value)
        return t
    } else {
        // DATETIME format: 20230101T120000Z
        layouts := []string{"20060102T150405Z", "20060102T150405"}
        for _, layout := range layouts {
            if t, err := time.Parse(layout, value); err == nil {
                return t
            }
        }
    }

    return time.Time{}
}

func unescapeICalText(text string) string {
    text = strings.ReplaceAll(text, "\\n", "\n")
    text = strings.ReplaceAll(text, "\\,", ",")
    text = strings.ReplaceAll(text, "\\;", ";")
    text = strings.ReplaceAll(text, "\\\\", "\\")
    return text
}

// FetchICalURL fetches an iCal feed from a URL
func FetchICalURL(url string) ([]ICalEvent, error) {
    client := &http.Client{
        Timeout: 30 * time.Second,
    }

    resp, err := client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch iCal feed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("iCal feed returned status %d", resp.StatusCode)
    }

    return ParseICalendar(resp.Body)
}
```

### 2.2 Calendar Event Service

**go-backend/services/calendar_events.go:**
```go
package services

import (
    "database/sql"
    "fmt"
    "log"
    "time"
)

type CalendarEventService struct {
    db *sql.DB
}

func NewCalendarEventService(db *sql.DB) *CalendarEventService {
    return &CalendarEventService{db: db}
}

// SyncExternalCalendar fetches and imports events from an external calendar
func (s *CalendarEventService) SyncExternalCalendar(calendarID int, userID int) error {
    // Get calendar details
    var url string
    err := s.db.QueryRow("SELECT url FROM external_calendars WHERE id = $1 AND user_id = $2",
        calendarID, userID).Scan(&url)
    if err != nil {
        return fmt.Errorf("calendar not found: %w", err)
    }

    // Fetch events
    events, err := FetchICalURL(url)
    if err != nil {
        s.UpdateLastSyncError(calendarID, err.Error())
        return fmt.Errorf("failed to fetch events: %w", err)
    }

    // Import events
    for _, event := range events {
        err := s.importEvent(calendarID, userID, event)
        if err != nil {
            log.Printf("Failed to import event %s: %v", event.UID, err)
        }
    }

    // Update last synced
    s.UpdateLastSynced(calendarID)
    return nil
}

func (s *CalendarEventService) importEvent(calendarID, userID int, event ICalEvent) error {
    // Check if event already exists
    var existingID int
    err := s.db.QueryRow(
        "SELECT id FROM calendar_events WHERE user_id = $1 AND external_uid = $2",
        userID, event.UID).Scan(&existingID)

    if err == sql.ErrNoRows {
        // Insert new event
        _, err = s.db.Exec(`
            INSERT INTO calendar_events (
                user_id, external_calendar_id, title, description,
                start_time, end_time, all_day, location,
                external_uid, external_url, recurrence_rule
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        `, userID, calendarID, event.Summary, event.Description,
            event.DTStart, event.DTEnd, event.AllDay, event.Location,
            event.UID, event.URL, event.RecurrenceRule)
        return err
    } else if err == nil {
        // Update existing event
        _, err = s.db.Exec(`
            UPDATE calendar_events SET
                title = $1, description = $2, start_time = $3, end_time = $4,
                all_day = $5, location = $6, external_url = $7,
                updated_at = NOW(), last_synced_at = NOW()
            WHERE id = $8
        `, event.Summary, event.Description, event.DTStart, event.DTEnd,
            event.AllDay, event.Location, event.URL, existingID)
        return err
    }

    return err
}

func (s *CalendarEventService) UpdateLastSynced(calendarID int) {
    s.db.Exec("UPDATE external_calendars SET last_synced_at = NOW(), last_error = NULL WHERE id = $1", calendarID)
}

func (s *CalendarEventService) UpdateLastSyncError(calendarID int, errMsg string) {
    s.db.Exec("UPDATE external_calendars SET last_error = $1 WHERE id = $2", errMsg, calendarID)
}

// GetEventsInRange returns events for a user within a date range
func (s *CalendarEventService) GetEventsInRange(userID int, start, end time.Time) ([]models.CalendarEvent, error) {
    rows, err := s.db.Query(`
        SELECT id, user_id, external_calendar_id, title, description,
               start_time, end_time, all_day, location,
               external_uid, external_url, recurrence_rule, color,
               created_at, updated_at, last_synced_at
        FROM calendar_events
        WHERE user_id = $1
          AND start_time >= $2
          AND end_time <= $3
        ORDER BY start_time
    `, userID, start, end)

    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var events []models.CalendarEvent
    for rows.Next() {
        var event models.CalendarEvent
        err := rows.Scan(
            &event.ID, &event.UserID, &event.ExternalCalendarID,
            &event.Title, &event.Description,
            &event.StartTime, &event.EndTime, &event.AllDay, &event.Location,
            &event.ExternalUID, &event.ExternalURL, &event.RecurrenceRule, &event.Color,
            &event.CreatedAt, &event.UpdatedAt, &event.LastSyncedAt,
        )
        if err != nil {
            return nil, err
        }
        events = append(events, event)
    }

    return events, nil
}
```

## Phase 3: API Endpoints

### 3.1 External Calendars API

**go-backend/handlers/calendar_events.go:**
```go
package handlers

import (
    // imports...
)

// ListExternalCalendarsRoute handles GET /api/user/external-calendars
func (s *Handler) ListExternalCalendarsRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    rows, err := s.DB.Query(`
        SELECT id, user_id, name, url, sync_enabled, sync_interval_hours,
               color, last_synced_at, last_error, created_at, updated_at
        FROM external_calendars
        WHERE user_id = $1
        ORDER BY created_at
    `, userID)

    // ... return JSON array
}

// CreateExternalCalendarRoute handles POST /api/user/external-calendars
func (s *Handler) CreateExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    var req models.CreateExternalCalendarRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Validate URL is an iCal feed
    events, err := services.FetchICalURL(req.URL)
    if err != nil {
        http.Error(w, "Invalid iCal URL: "+err.Error(), http.StatusBadRequest)
        return
    }

    // Insert calendar
    var id int
    err = s.DB.QueryRow(`
        INSERT INTO external_calendars (user_id, name, url, color)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `, userID, req.Name, req.URL, req.Color).Scan(&id)

    // ... return created calendar
}

// SyncExternalCalendarRoute handles POST /api/user/external-calendars/{id}/sync
func (s *Handler) SyncExternalCalendarRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, _ := strconv.Atoi(mux.Vars(r)["id"])

    svc := services.NewCalendarEventService(s.DB)
    err := svc.SyncExternalCalendar(id, userID)

    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

// GetCalendarEventsRoute handles GET /api/user/calendar-events
func (s *Handler) GetCalendarEventsRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    // Parse date range from query params
    startStr := r.URL.Query().Get("start")
    endStr := r.URL.Query().Get("end")

    start, _ := time.Parse(time.RFC3339, startStr)
    end, _ := time.Parse(time.RFC3339, endStr)

    svc := services.NewCalendarEventService(s.DB)
    events, err := svc.GetEventsInRange(userID, start, end)

    // ... return JSON array
}
```

**go-backend/routes/calendar_events.go:**
```go
package routes

func RegisterCalendarEventRoutes(r *mux.Router, h *handlers.Handler) {
    addProtectedRoute(r, h, "/api/user/external-calendars", h.ListExternalCalendarsRoute, "GET")
    addProtectedRoute(r, h, "/api/user/external-calendars", h.CreateExternalCalendarRoute, "POST")
    addProtectedRoute(r, h, "/api/user/external-calendars/{id}", h.UpdateExternalCalendarRoute, "PUT")
    addProtectedRoute(r, h, "/api/user/external-calendars/{id}", h.DeleteExternalCalendarRoute, "DELETE")
    addProtectedRoute(r, h, "/api/user/external-calendars/{id}/sync", h.SyncExternalCalendarRoute, "POST")

    addProtectedRoute(r, h, "/api/user/calendar-events", h.GetCalendarEventsRoute, "GET")
}
```

Add to `routes/routes.go`:
```go
// Calendar event routes
RegisterCalendarEventRoutes(r, h)
```

## Phase 4: Frontend Components

### 4.1 Day View Component

**zettelkasten-front/src/components/calendar/DayView.tsx:**
```typescript
import React, { useState, useEffect } from 'react';
import { format, startOfDay, endOfDay, addHours, isSameDay, isBefore, isAfter } from 'date-fns';
import { CalendarEvent } from '../../models/CalendarEvent';

interface DayViewProps {
  date: Date;
  events: CalendarEvent[];
  onEventClick?: (event: CalendarEvent) => void;
  onNavigate: (date: Date) => void;
  userTimezone: string;
}

export function DayView({ date, events, onEventClick, onNavigate, userTimezone }: DayViewProps) {
  const hours = Array.from({ length: 24 }, (_, i) => i);
  const [currentTime, setCurrentTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => setCurrentTime(new Date()), 60000);
    return () => clearInterval(timer);
  }, []);

  const dayStart = startOfDay(date);
  const dayEnd = endOfDay(date);
  const dayEvents = events.filter(e => {
    const eventStart = new Date(e.start_time);
    return isSameDay(eventStart, date);
  });

  const allDayEvents = dayEvents.filter(e => e.all_day);
  const timedEvents = dayEvents.filter(e => !e.all_day);

  return (
    <div className="flex flex-col h-full bg-white">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <button onClick={() => onNavigate(addDays(date, -1))} className="p-2">←</button>
        <h2 className="text-lg font-semibold">{format(date, 'EEEE, MMMM d, yyyy')}</h2>
        <button onClick={() => onNavigate(addDays(date, 1))} className="p-2">→</button>
      </div>

      {/* All-day events section */}
      {allDayEvents.length > 0 && (
        <div className="p-2 border-b bg-gray-50">
          {allDayEvents.map(event => (
            <div
              key={event.id}
              onClick={() => onEventClick?.(event)}
              className="p-2 mb-1 rounded bg-blue-100 border-l-4 border-blue-500 cursor-pointer hover:bg-blue-200"
              style={{ borderLeftColor: event.color }}
            >
              <span className="font-medium">{event.title}</span>
            </div>
          ))}
        </div>
      )}

      {/* Time grid */}
      <div className="flex-1 overflow-y-auto">
        {hours.map(hour => (
          <div key={hour} className="relative border-b" style={{ minHeight: '60px' }}>
            <div className="absolute left-0 top-0 w-16 p-2 text-sm text-gray-500 text-right">
              {format(setHours(dayStart, hour), 'ha')}
            </div>
            <div className="ml-16 h-full" />
            {/* Current time indicator */}
            {isSameDay(currentTime, date) &&
             format(currentTime, 'H') === String(hour).padStart(2, '0') && (
              <div className="absolute left-16 right-0 border-t-2 border-red-500 z-10">
                <div className="w-2 h-2 bg-red-500 rounded-full -ml-1 -mt-1" />
              </div>
            )}
          </div>
        ))}

        {/* Render events positioned by time */}
        {timedEvents.map(event => renderTimeEvent(event))}
      </div>
    </div>
  );
}

function renderTimeEvent(event: CalendarEvent) {
  const start = new Date(event.start_time);
  const end = new Date(event.end_time);
  const startHour = start.getHours() + start.getMinutes() / 60;
  const durationHours = (end.getTime() - start.getTime()) / (1000 * 60 * 60);

  const style: React.CSSProperties = {
    position: 'absolute',
    left: '4rem',
    right: '0.5rem',
    top: `${startHour * 60}px`,
    height: `${Math.max(durationHours * 60, 30)}px`,
    backgroundColor: event.color || '#3b82f6',
    borderRadius: '4px',
    padding: '4px 8px',
    fontSize: '12px',
    overflow: 'hidden',
    cursor: 'pointer',
  };

  return (
    <div
      key={event.id}
      onClick={() => onEventClick?.(event)}
      style={style}
      className="hover:opacity-90"
    >
      <div className="font-medium truncate">{event.title}</div>
      {durationHours > 0.5 && (
        <div className="opacity-80 truncate">{format(start, 'h:mm')} - {format(end, 'h:mm')}</div>
      )}
    </div>
  );
}
```

### 4.2 Calendar Subscriptions Settings

**zettelkasten-front/src/components/settings/CalendarSubscriptions.tsx:**
```typescript
import React, { useState, useEffect } from 'react';
import { ExternalCalendar, getExternalCalendars, createExternalCalendar, syncExternalCalendar } from '../../api/calendarEvents';

export function CalendarSubscriptions() {
  const [calendars, setCalendars] = useState<ExternalCalendar[]>([]);
  const [showAddForm, setShowAddForm] = useState(false);
  const [syncing, setSyncing] = useState<Set<number>>(new Set());

  useEffect(() => {
    loadCalendars();
  }, []);

  async function loadCalendars() {
    const data = await getExternalCalendars();
    setCalendars(data);
  }

  async function handleAdd(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = {
      name: (form.name as any).value,
      url: (form.url as any).value,
      color: (form.color as any).value || '#3b82f6',
    };

    await createExternalCalendar(data);
    setShowAddForm(false);
    loadCalendars();
  }

  async function handleSync(id: number) {
    setSyncing(prev => new Set(prev).add(id));
    try {
      await syncExternalCalendar(id);
      await loadCalendars();
    } finally {
      setSyncing(prev => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">Calendar Subscriptions</h2>
        <button
          onClick={() => setShowAddForm(true)}
          className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
        >
          Add Calendar
        </button>
      </div>

      {showAddForm && (
        <form onSubmit={handleAdd} className="mb-6 p-4 border rounded bg-gray-50">
          <h3 className="font-medium mb-3">Subscribe to Calendar</h3>
          <div className="space-y-3">
            <input name="name" placeholder="Calendar name" required className="w-full px-3 py-2 border rounded" />
            <input name="url" placeholder="iCal URL (https://...)" type="url" required className="w-full px-3 py-2 border rounded" />
            <div className="flex gap-2 items-center">
              <input name="color" type="color" defaultValue="#3b82f6" className="w-16 h-10" />
              <span className="text-sm text-gray-600">Event color</span>
            </div>
            <div className="flex gap-2">
              <button type="submit" className="bg-blue-500 text-white px-4 py-2 rounded">Subscribe</button>
              <button type="button" onClick={() => setShowAddForm(false)} className="px-4 py-2 border rounded">Cancel</button>
            </div>
          </div>
        </form>
      )}

      <div className="space-y-3">
        {calendars.map(cal => (
          <div key={cal.id} className="flex items-center justify-between p-3 border rounded">
            <div className="flex items-center gap-3">
              <div
                className="w-4 h-4 rounded"
                style={{ backgroundColor: cal.color }}
              />
              <div>
                <div className="font-medium">{cal.name}</div>
                <div className="text-sm text-gray-500">{cal.url}</div>
                {cal.last_synced_at && (
                  <div className="text-xs text-gray-400">
                    Last synced: {new Date(cal.last_synced_at).toLocaleString()}
                  </div>
                )}
                {cal.last_error && (
                  <div className="text-xs text-red-500">{cal.last_error}</div>
                )}
              </div>
            </div>
            <button
              onClick={() => handleSync(cal.id)}
              disabled={syncing.has(cal.id)}
              className="px-3 py-1 text-sm border rounded hover:bg-gray-50 disabled:opacity-50"
            >
              {syncing.has(cal.id) ? 'Syncing...' : 'Sync'}
            </button>
          </div>
        ))}

        {calendars.length === 0 && (
          <p className="text-gray-500 text-center py-4">
            No calendars subscribed. Add one above to import events.
          </p>
        )}
      </div>
    </div>
  );
}
```

### 4.3 API Client

**zettelkasten-front/src/api/calendarEvents.ts:**
```typescript
import { apiClient, getData } from './client';
import { CalendarEvent, ExternalCalendar, CreateExternalCalendarRequest } from '../models/CalendarEvent';

export async function getExternalCalendars(): Promise<ExternalCalendar[]> {
  return getData(apiClient.get<ExternalCalendar[]>('/user/external-calendars'));
}

export async function createExternalCalendar(data: CreateExternalCalendarRequest): Promise<ExternalCalendar> {
  return getData(apiClient.post<ExternalCalendar>('/user/external-calendars', data));
}

export async function syncExternalCalendar(id: number): Promise<{ message: string }> {
  return getData(apiClient.post<{ message: string }>(`/user/external-calendars/${id}/sync`, {}));
}

export async function getCalendarEvents(start: Date, end: Date): Promise<CalendarEvent[]> {
  return getData(apiClient.get<CalendarEvent[]>('/user/calendar-events', {
    params: {
      start: start.toISOString(),
      end: end.toISOString(),
    },
  }));
}
```

## Phase 5: TaskPage Integration

Add "day" as a view mode in TaskPage.tsx:

```typescript
// Add to view mode select
<option value="day">Day View</option>

// In render:
{settings.viewMode === "day" && (
  <DayView
    date={settings.calendarCurrentDate}
    events={calendarEvents}
    onNavigate={settings.navigateCalendar}
    onEventClick={(eventId) => {/* show event dialog */}}
    userTimezone={userTimezone}
  />
)}
```

Add to `useTaskPageSettings`:
```typescript
const [calendarEvents, setCalendarEvents] = useState<CalendarEvent[]>([]);

useEffect(() => {
  async function loadEvents() {
    const dayStart = startOfDay(settings.calendarCurrentDate);
    const dayEnd = endOfDay(settings.calendarCurrentDate);
    const events = await getCalendarEvents(dayStart, dayEnd);
    setCalendarEvents(events);
  }
  if (settings.viewMode === 'day') {
    loadEvents();
  }
}, [settings.viewMode, settings.calendarCurrentDate]);
```

## Phase 6: Background Sync (P2)

**go-backend/jobs/calendar_sync.go:**
```go
package jobs

import (
    "context"
    "go-backend/handlers"
    "go-backend/models"
    "go-backend/services"
    "log"
    "time"
)

type CalendarSyncJob struct {
    db *sql.DB
}

func (j *CalendarSyncJob) Execute(ctx context.Context, job models.Job) error {
    svc := services.NewCalendarEventService(j.db)

    // Get all enabled calendars
    rows, err := j.db.Query(`
        SELECT id, user_id, sync_interval_hours
        FROM external_calendars
        WHERE sync_enabled = true
          AND (last_synced_at IS NULL
               OR last_synced_at < NOW() - (sync_interval_hours || ' hours')::interval)
    `)

    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var id, userID, intervalHours int
        rows.Scan(&id, &userID, &intervalHours)

        err := svc.SyncExternalCalendar(id, userID)
        if err != nil {
            log.Printf("Failed to sync calendar %d for user %d: %v", id, userID, err)
        }
    }

    return nil
}
```

Schedule via existing job queue infrastructure.

## Edge Cases and Considerations

1. **Timezones**: Store all times in UTC, convert to user's timezone for display
2. **Duplicates**: Use UNIQUE constraint on (user_id, external_uid) to prevent duplicates
3. **Large Calendars**: Paginate or limit imports (max 1000 events per sync)
4. **Recurrence**: Store RRULE but don't expand in initial version (complex)
5. **Deleted Events**: Check if external UID still exists; if not, soft-delete local event
6. **Rate Limiting**: Respect external server rate limits (add delay between syncs)
7. **Privacy**: Events are user-scoped, no sharing between users

## Testing Checklist

- [ ] Can subscribe to Google Calendar public iCal URL
- [ ] Events display in day view at correct times
- [ ] All-day events display in all-day section
- [ ] Manual sync updates existing events
- [ ] Sync errors are displayed to user
- [ ] Multiple calendars can be subscribed
- [ ] Calendar colors are respected
- [ ] Timezone conversion works correctly
- [ ] Current time indicator displays
- [ ] Navigate between days works

## VTODO Support (Deferred)

For VTODO import (Zettelgarden-fbqy.9), extend the iCal parser to handle VTODO components:

```go
type ICalTodo struct {
    UID         string
    Summary     string
    Description string
    Due         time.Time
    Priority    int
    Status      string  // NEEDS-ACTION, IN-PROCESS, COMPLETED
    PercentComplete int
}
```

Map to Task model:
- Summary → Title
- Description → Description
- Due → Due Date
- Priority → Priority (map 1-9 to A-D)
- Status → Status
