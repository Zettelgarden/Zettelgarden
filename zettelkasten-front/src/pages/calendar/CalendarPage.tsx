import React, { useEffect, useState, useRef } from "react";
import { useTaskContext } from "../../contexts/TaskContext";
import { useAuth } from "../../contexts/AuthContext";
import { setDocumentTitle } from "../../utils/title";
import { ErrorBoundary } from "../../components/ErrorBoundary";
import { CalendarViewWrapper } from "../../components/calendar/CalendarView";
import { TaskDialog } from "../../components/tasks/TaskDialog";
import { CreateTaskWindow } from "../../components/tasks/CreateTaskWindow";
import { getExternalEvents } from "../../api/externalEvents";
import { ExternalEvent } from "../../models/ExternalEvent";
import {
  getStartOfMonthInTimezone,
  getEndOfMonthInTimezone,
  getStartOfWeekInTimezone,
  getEndOfWeekInTimezone,
} from "../../utils/dates";

const CALENDAR_SETTINGS_KEY = "calendarPageSettings";

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
          externalEvents={externalEvents}
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
    </div>
  );
}
