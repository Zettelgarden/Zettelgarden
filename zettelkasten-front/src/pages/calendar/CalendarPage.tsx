import React, { useEffect, useState, useRef } from "react";
import { FaCog } from "react-icons/fa";
import { useTaskContext } from "../../contexts/TaskContext";
import { useAuth } from "../../contexts/AuthContext";
import { setDocumentTitle } from "../../utils/title";
import { ErrorBoundary } from "../../components/ErrorBoundary";
import { CalendarViewWrapper } from "../../components/calendar/CalendarView";
import { TaskDialog } from "../../components/tasks/TaskDialog";
import { CreateTaskWindow } from "../../components/tasks/CreateTaskWindow";
import { getExternalEvents, getExternalCalendars } from "../../api/externalEvents";
import { ExternalEvent, ExternalCalendar } from "../../models/ExternalEvent";
import {
  getStartOfMonthInTimezone,
  getEndOfMonthInTimezone,
  getStartOfWeekInTimezone,
  getEndOfWeekInTimezone,
} from "../../utils/dates";

const CALENDAR_SETTINGS_KEY = "calendarPageSettings";
const CALENDAR_VISIBILITY_KEY = "calendarVisibility";

interface CalendarSettings {
  viewMode: "month" | "week";
  currentDate: Date;
}

function getInitialSettings(): CalendarSettings {
  try {
    const saved = localStorage.getItem(CALENDAR_SETTINGS_KEY);
    if (saved) {
      const parsed = JSON.parse(saved);
      return {
        viewMode: parsed.viewMode || "month",
        currentDate: parsed.currentDate ? new Date(parsed.currentDate) : new Date(),
      };
    }
  } catch {
    // Fall through to defaults
  }
  return {
    viewMode: "month",
    currentDate: new Date(),
  };
}

export function CalendarPage() {
  const { tasks, setRefreshTasks } = useTaskContext();
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";

  // Calendar-specific settings (separate from task page settings)
  const [viewMode, setViewMode] = useState<"month" | "week">(getInitialSettings().viewMode);
  const [currentDate, setCurrentDate] = useState<Date>(getInitialSettings().currentDate);

  // State for task dialog
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [isTaskDialogOpen, setIsTaskDialogOpen] = useState(false);
  const [showCreateTaskWindow, setShowCreateTaskWindow] = useState(false);
  const [calendarSelectedDate, setCalendarSelectedDate] = useState<Date | null>(null);

  // External calendar events state
  const [externalEvents, setExternalEvents] = useState<ExternalEvent[]>([]);
  const [isLoadingEvents, setIsLoadingEvents] = useState(false);

  // Calendar visibility state
  const [hiddenCalendarIds, setHiddenCalendarIds] = useState<Set<number>>(new Set());
  const [calendars, setCalendars] = useState<ExternalCalendar[]>([]);
  const [isLoadingCalendars, setIsLoadingCalendars] = useState(false);
  const [isSettingsDialogOpen, setIsSettingsDialogOpen] = useState(false);

  // Ref for abort controller to cancel pending external events requests
  const abortControllerRef = useRef<AbortController | null>(null);

  // Persist calendar settings to localStorage
  useEffect(() => {
    const settings = {
      viewMode,
      currentDate: currentDate.toISOString(),
    };
    localStorage.setItem(CALENDAR_SETTINGS_KEY, JSON.stringify(settings));
  }, [viewMode, currentDate]);

  // Load saved calendar visibility on mount
  useEffect(() => {
    const saved = localStorage.getItem(CALENDAR_VISIBILITY_KEY);
    if (saved) {
      try {
        const hiddenIds = JSON.parse(saved);
        setHiddenCalendarIds(new Set(hiddenIds));
      } catch {
        // Ignore invalid saved data
      }
    }
  }, []);

  // Load external calendars
  useEffect(() => {
    async function loadCalendars() {
      setIsLoadingCalendars(true);
      try {
        const calendarsData = await getExternalCalendars();
        setCalendars(calendarsData);

        // Clean up orphaned calendar IDs from localStorage
        const validIds = new Set(calendarsData.map(c => c.id));
        setHiddenCalendarIds(prev => {
          const cleaned = new Set<number>();
          for (const id of prev) {
            if (validIds.has(id)) cleaned.add(id);
          }
          // Update localStorage if any IDs were removed
          if (cleaned.size !== prev.size) {
            try {
              localStorage.setItem(CALENDAR_VISIBILITY_KEY, JSON.stringify([...cleaned]));
            } catch {
              // Ignore localStorage errors
            }
          }
          return cleaned;
        });
      } catch (err) {
        console.error("Failed to load external calendars:", err);
        setCalendars([]);
      } finally {
        setIsLoadingCalendars(false);
      }
    }

    loadCalendars();
  }, []);

  // Toggle calendar visibility
  const toggleCalendarVisibility = (calendarId: number) => {
    setHiddenCalendarIds(prev => {
      const next = new Set(prev);
      if (next.has(calendarId)) {
        next.delete(calendarId);
      } else {
        next.add(calendarId);
      }
      // Persist to localStorage
      localStorage.setItem(CALENDAR_VISIBILITY_KEY, JSON.stringify([...next]));
      return next;
    });
  };

  // Filter out events from hidden calendars
  const visibleExternalEvents = externalEvents.filter(event => {
    if (event.external_calendar_id == null) return true;
    return !hiddenCalendarIds.has(event.external_calendar_id);
  });

  // Load external calendar events
  useEffect(() => {
    async function loadExternalEvents() {
      // Cancel any pending request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      // Create new abort controller for this request
      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      setIsLoadingEvents(true);

      try {
        // Calculate date range based on view mode and current date
        let start: Date;
        let end: Date;

        if (viewMode === "month") {
          start = getStartOfMonthInTimezone(currentDate, userTimezone);
          end = getEndOfMonthInTimezone(currentDate, userTimezone);
        } else {
          // Week view
          start = getStartOfWeekInTimezone(currentDate, userTimezone, 0);
          end = getEndOfWeekInTimezone(currentDate, userTimezone, 0);
        }

        const events = await getExternalEvents(start, end, abortController.signal);
        // Only update state if this request wasn't aborted
        if (!abortController.signal.aborted) {
          setExternalEvents(events);
        }
      } catch (err) {
        // Only handle error if not due to abort
        if (!abortController.signal.aborted) {
          console.error("Failed to load external events:", err);
          setExternalEvents([]);
        }
      } finally {
        // Only clear loading state if this request wasn't aborted
        if (!abortController.signal.aborted) {
          setIsLoadingEvents(false);
        }
      }
    }

    loadExternalEvents();

    // Cleanup function to abort pending requests on unmount
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, [viewMode, currentDate, userTimezone]);

  // Navigation handler
  const navigateCalendar = (direction: "prev" | "next" | "today") => {
    if (direction === "today") {
      setCurrentDate(new Date());
    } else if (direction === "prev") {
      setCurrentDate(prev => {
        const newDate = new Date(prev);
        if (viewMode === "month") {
          newDate.setMonth(newDate.getMonth() - 1);
        } else {
          newDate.setDate(newDate.getDate() - 7);
        }
        return newDate;
      });
    } else if (direction === "next") {
      setCurrentDate(prev => {
        const newDate = new Date(prev);
        if (viewMode === "month") {
          newDate.setMonth(newDate.getMonth() + 1);
        } else {
          newDate.setDate(newDate.getDate() + 7);
        }
        return newDate;
      });
    }
  };

  // Set document title
  useEffect(() => {
    setDocumentTitle("Calendar");
  }, []);

  const handleKeyPress = (event: KeyboardEvent) => {
    if (event.metaKey) {
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      setShowCreateTaskWindow(false);
      setIsSettingsDialogOpen(false);
      return;
    }
  };

  useEffect(() => {
    document.addEventListener("keydown", handleKeyPress);
    return () => {
      document.removeEventListener("keydown", handleKeyPress);
    };
  }, [setShowCreateTaskWindow]);

  return (
    <div className="p-4">
      {isLoadingEvents && (
        <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-md flex items-center gap-2">
          <div className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
          <span className="text-sm text-blue-700">Loading calendar events...</span>
        </div>
      )}

      <ErrorBoundary
        fallback={
          <div className="p-4 m-4 border border-red-300 rounded bg-red-50">
            <h2 className="text-lg font-semibold text-red-800 mb-2">Calendar Error</h2>
            <p className="text-red-600 mb-3">
              We encountered an error while displaying the calendar. Please try refreshing the page.
            </p>
            <button
              className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
              onClick={() => window.location.reload()}
            >
              Refresh Page
            </button>
          </div>
        }
      >
        <CalendarViewWrapper
          tasks={tasks}
          externalEvents={visibleExternalEvents}
          currentDate={currentDate}
          viewMode={viewMode}
          onNavigate={navigateCalendar}
          onViewModeChange={setViewMode}
          onTaskClick={(taskId) => {
            setSelectedTaskId(taskId);
            setIsTaskDialogOpen(true);
          }}
          onCreateTask={(date) => {
            setCalendarSelectedDate(date);
            setShowCreateTaskWindow(true);
          }}
          onTaskMoved={() => {
            setRefreshTasks(true);
          }}
          timezone={userTimezone}
          calendarSettingsButton={
            calendars.length > 0 ? (
              <button
                onClick={() => setIsSettingsDialogOpen(true)}
                className="p-2 hover:bg-slate-200 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[44px] min-w-[44px] flex items-center justify-center"
                aria-label="Calendar settings"
              >
                <FaCog size={16} aria-hidden="true" />
              </button>
            ) : null
          }
        />
      </ErrorBoundary>

      {/* Create Task Window */}
      {showCreateTaskWindow && (
        <CreateTaskWindow
          currentCard={null}
          setShowTaskWindow={setShowCreateTaskWindow}
          currentFilter=""
          initialStatus={undefined}
          initialDate={calendarSelectedDate || undefined}
        />
      )}

      {/* Task Dialog */}
      <TaskDialog
        taskId={selectedTaskId}
        isOpen={isTaskDialogOpen}
        onClose={() => {
          setIsTaskDialogOpen(false);
          setSelectedTaskId(null);
        }}
      />

      {/* Calendar Settings Dialog */}
      {isSettingsDialogOpen && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
          onClick={() => setIsSettingsDialogOpen(false)}
        >
          <div
            className="bg-white rounded-lg shadow-xl max-w-md w-full"
            onClick={(e) => e.stopPropagation()}
            role="dialog"
            aria-modal="true"
            aria-labelledby="calendar-settings-title"
          >
            <div className="p-4 border-b flex justify-between items-center">
              <h3 id="calendar-settings-title" className="text-lg font-semibold">Calendar Settings</h3>
              <button
                onClick={() => setIsSettingsDialogOpen(false)}
                className="p-2 hover:bg-gray-100 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Close dialog"
              >
                ✕
                <span className="sr-only">Close</span>
              </button>
            </div>
            <div className="p-4 space-y-3">
              <div>
                <h4 className="text-sm font-medium text-slate-700 mb-3">Show Calendars</h4>
                {calendars.length === 0 ? (
                  <p className="text-slate-500 text-center py-4">No calendars configured</p>
                ) : (
                  <div className="space-y-2">
                    {calendars.map(calendar => {
                      const isHidden = hiddenCalendarIds.has(calendar.id);
                      return (
                        <label
                          key={calendar.id}
                          className="flex items-center gap-3 cursor-pointer select-none p-2 hover:bg-slate-50 rounded"
                        >
                          <input
                            type="checkbox"
                            checked={!isHidden}
                            onChange={() => toggleCalendarVisibility(calendar.id)}
                            className="w-5 h-5 rounded border-slate-300 text-blue-600 focus:ring-2 focus:ring-blue-500 focus:ring-offset-0"
                          />
                          <span
                            className="w-4 h-4 rounded-full border border-slate-200 flex-shrink-0"
                            style={{ backgroundColor: calendar.color }}
                            aria-hidden="true"
                          />
                          <span className={`flex-1 ${!isHidden ? "text-slate-700" : "text-slate-400 line-through"}`}>
                            {calendar.name}
                          </span>
                        </label>
                      );
                    })}
                  </div>
                )}
              </div>
              <div className="pt-3 border-t flex justify-end">
                <button
                  onClick={() => setIsSettingsDialogOpen(false)}
                  className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                >
                  Done
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
