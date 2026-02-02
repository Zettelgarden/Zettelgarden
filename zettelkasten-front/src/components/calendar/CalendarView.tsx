import React, { useState, useCallback, useEffect, useRef, useMemo } from "react";
import { FaChevronLeft, FaChevronRight, FaPlus, FaChevronUp, FaChevronDown } from "react-icons/fa";
import { DragDropContext, Droppable, Draggable, DropResult } from "@hello-pangea/dnd";
import { toZonedTime } from "date-fns-tz";
import { Task } from "../../models/Task";
import { ExternalEvent } from "../../models/ExternalEvent";
import {
  CalendarDay,
  CalendarEvent,
  CalendarViewType,
  tasksToCalendarEvents,
  getEventColor,
  getEventIcon,
  isEventDraggable,
} from "../../models/CalendarEvent";
import {
  generateMonthGrid,
  generateWeekGrid,
  populateDayEvents,
  formatMonthHeader,
  formatWeekHeader,
  getWeekDayNames,
  getVisibleEvents,
  getHiddenEventCount,
  groupEventsByTask,
  mergeCalendarEvents,
} from "../../utils/calendar";
import { format } from "date-fns";
import { saveExistingTask } from "../../api/tasks";
import { isSameDayInTimezone, createMidnightInTimezone } from "../../utils/dates";
import { useTaskContext } from "../../contexts/TaskContext";
import { BacklinkInputDropdownList } from "../cards/BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";

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
  onTaskMoved?: () => void;
  onExternalEventClick?: (event: CalendarEvent) => void;
  onExternalEventChange?: () => void;
  timezone?: string;
}

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
  onTaskMoved,
  onExternalEventClick,
  onExternalEventChange,
  timezone = "UTC",
}: CalendarViewProps) {
  const [hoveredDay, setHoveredDay] = useState<Date | null>(null);
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    date: Date;
  } | null>(null);
  const [selectedDayIndex, setSelectedDayIndex] = useState<number>(-1);
  const [externalEventDialog, setExternalEventDialog] = useState<CalendarEvent | null>(null);
  const [dragError, setDragError] = useState<string | null>(null);
  const calendarRef = useRef<HTMLDivElement>(null);

  // Memoize week day names
  const weekDayNames = useMemo(() => getWeekDayNames(), []);

  // Convert tasks to calendar events and merge with external events
  const events = useMemo(() => {
    const taskEvents = tasksToCalendarEvents(tasks, timezone);
    return mergeCalendarEvents(taskEvents, externalEvents);
  }, [tasks, externalEvents, timezone]);

  // Generate the calendar grid based on view mode
  const grid = useMemo(() => {
    return viewMode === "month"
      ? generateMonthGrid(currentDate, timezone)
      : generateWeekGrid(currentDate, timezone);
  }, [viewMode, currentDate, timezone]);

  // Populate events into the grid
  const days = useMemo(() => {
    return populateDayEvents(grid, events, timezone);
  }, [grid, events, timezone]);

  const handleDayClick = (day: CalendarDay) => {
    onDayClick(day.date, day.events);
  };

  const handleEventClick = (e: React.MouseEvent, event: CalendarEvent) => {
    e.stopPropagation();

    // External events show read-only dialog
    if (event.source === "external") {
      setExternalEventDialog(event);
      onExternalEventClick?.(event);
    } else {
      onEventClick(event);
    }
  };

  // Context menu handlers
  const handleContextMenu = (e: React.MouseEvent, day: CalendarDay) => {
    e.preventDefault();
    e.stopPropagation();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      date: day.date,
    });
  };

  const closeContextMenu = () => {
    setContextMenu(null);
  };

  const handleContextMenuCreateTask = () => {
    if (contextMenu && onCreateTask) {
      onCreateTask(contextMenu.date);
      closeContextMenu();
    }
  };

  const handleContextMenuViewTasks = () => {
    if (contextMenu) {
      // Use consistent timezone-aware date comparison
      const dayEvents = events.filter(e => isSameDayInTimezone(e.date, contextMenu.date, timezone));
      onDayClick(contextMenu.date, dayEvents);
      closeContextMenu();
    }
  };

  // Close context menu on click outside - only attach listener when menu is open
  useEffect(() => {
    if (!contextMenu) return;

    const handleClickOutside = () => closeContextMenu();
    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [contextMenu]);

  // Keyboard navigation - use ref pattern to avoid frequent listener re-attachment
  const handleKeyDownRef = useRef<(e: KeyboardEvent) => void>();

  handleKeyDownRef.current = useCallback((e: KeyboardEvent) => {
    // Only handle keyboard when calendar is focused or no input is focused
    const target = e.target as HTMLElement;
    if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable) {
      return;
    }

    const grid = viewMode === "month" ? generateMonthGrid(currentDate, timezone) : generateWeekGrid(currentDate, timezone);

    switch (e.key) {
      case "ArrowLeft":
        e.preventDefault();
        setSelectedDayIndex(prev => {
          const newIndex = prev > 0 ? prev - 1 : grid.length - 1;
          setHoveredDay(grid[newIndex].date);
          return newIndex;
        });
        break;
      case "ArrowRight":
        e.preventDefault();
        setSelectedDayIndex(prev => {
          const newIndex = prev < grid.length - 1 ? prev + 1 : 0;
          setHoveredDay(grid[newIndex].date);
          return newIndex;
        });
        break;
      case "ArrowUp":
        e.preventDefault();
        setSelectedDayIndex(prev => {
          const newIndex = prev >= 7 ? prev - 7 : prev;
          setHoveredDay(grid[newIndex].date);
          return newIndex;
        });
        break;
      case "ArrowDown":
        e.preventDefault();
        setSelectedDayIndex(prev => {
          const newIndex = prev < grid.length - 7 ? prev + 7 : prev;
          setHoveredDay(grid[newIndex].date);
          return newIndex;
        });
        break;
      case "Enter":
        e.preventDefault();
        if (selectedDayIndex >= 0 && hoveredDay) {
          const dayEvents = events.filter(
            ev => isSameDayInTimezone(ev.date, hoveredDay, timezone)
          );
          onDayClick(hoveredDay, dayEvents);
        }
        break;
      case "Escape":
        e.preventDefault();
        closeContextMenu();
        setSelectedDayIndex(-1);
        setHoveredDay(null);
        break;
    }
  }, [currentDate, events, hoveredDay, onDayClick, selectedDayIndex, viewMode, timezone]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => handleKeyDownRef.current?.(e);
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, []);

  // Get setRefreshTasks from TaskContext for refreshing after drag-drop
  const { setRefreshTasks } = useTaskContext();

  // Drag and drop handler
  const handleDragEnd = useCallback(async (result: DropResult) => {
    if (!result.destination) return;

    const sourceDateStr = result.source.droppableId;
    const destDateStr = result.destination.droppableId;

    // No change if dropped in the same day
    if (sourceDateStr === destDateStr) return;

    // External events (prefixed with "ext-") cannot be dragged
    if (result.draggableId.startsWith("ext-")) return;

    const taskId = parseInt(result.draggableId, 10);
    if (isNaN(taskId)) return;

    const task = tasks.find(t => t.id === taskId);
    if (!task) return;

    // Calculate new scheduled date from destination
    // destDateStr is "YYYY-MM-DD" format - create midnight in user's timezone
    const newScheduledDate = createMidnightInTimezone(destDateStr, timezone);

    // Update task with new scheduled date
    const updatedTask = {
      ...task,
      scheduled_date: newScheduledDate,
    };

    setDragError(null);
    try {
      // Persist changes
      const response = await saveExistingTask(updatedTask);
      if (!("error" in response)) {
        setRefreshTasks(true);
        onTaskMoved?.();
      } else {
        setDragError("Failed to reschedule task. Please try again.");
      }
    } catch (err) {
      console.error("Failed to reschedule task after drag-and-drop:", err);
      setDragError("Failed to reschedule task. Please try again.");
    }
  }, [tasks, onTaskMoved, setRefreshTasks, setDragError]);

  return (
    <div className="bg-white border border-slate-300 rounded-lg overflow-hidden">
      {/* Calendar Header */}
      <div className="bg-slate-100 px-2 sm:px-4 py-2 sm:py-3 border-b border-slate-300">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-2 sm:gap-3">
          {/* View Mode Toggle - Hidden on mobile, show on tablet+ */}
          <div className="flex items-center gap-1 sm:gap-2 hidden sm:flex">
            <span className="text-xs text-slate-600 font-medium">View:</span>
            <button
              onClick={() => onViewModeChange("month")}
              className={`px-2 sm:px-3 py-1.5 sm:py-1 text-xs sm:text-sm rounded-md min-h-[44px] min-w-[44px] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                viewMode === "month"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-slate-700 border border-slate-300 hover:bg-slate-50"
              }`}
              aria-label="Switch to month view"
              aria-pressed={viewMode === "month"}
            >
              <span className="hidden sm:inline">Month</span>
              <span className="sm:hidden">Mo</span>
            </button>
            <button
              onClick={() => onViewModeChange("week")}
              className={`px-2 sm:px-3 py-1.5 sm:py-1 text-xs sm:text-sm rounded-md min-h-[44px] min-w-[44px] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${
                viewMode === "week"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-slate-700 border border-slate-300 hover:bg-slate-50"
              }`}
              aria-label="Switch to week view"
              aria-pressed={viewMode === "week"}
            >
              <span className="hidden sm:inline">Week</span>
              <span className="sm:hidden">Wk</span>
            </button>
          </div>

          {/* Mobile View Mode Indicator */}
          <div className="flex sm:hidden items-center gap-1">
            <span className="text-xs font-medium text-slate-700">
              {viewMode === "month" ? "Month" : "Week"}
            </span>
            <button
              onClick={() => onViewModeChange(viewMode === "month" ? "week" : "month")}
              className="px-2 py-1.5 text-xs bg-slate-200 rounded min-h-[44px] min-w-[44px]"
              aria-label={`Switch to ${viewMode === "month" ? "week" : "month"} view`}
            >
              Toggle
            </button>
          </div>

          {/* Navigation Controls */}
          <div className="flex items-center gap-1">
            <button
              onClick={() => onNavigate("prev")}
              className="p-2 hover:bg-slate-200 rounded min-h-[44px] min-w-[44px] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              aria-label="Previous period"
              title="Previous"
            >
              <FaChevronLeft size={18} aria-hidden="true" />
              <span className="sr-only">Previous</span>
            </button>
            <button
              onClick={() => onNavigate("today")}
              className="px-2 sm:px-3 py-1.5 text-xs sm:text-sm bg-white border border-slate-300 rounded-md hover:bg-slate-50 min-h-[44px] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              aria-label="Go to today"
            >
              Today
            </button>
            <button
              onClick={() => onNavigate("next")}
              className="p-2 hover:bg-slate-200 rounded min-h-[44px] min-w-[44px] focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              aria-label="Next period"
              title="Next"
            >
              <FaChevronRight size={18} aria-hidden="true" />
              <span className="sr-only">Next</span>
            </button>
          </div>

          {/* Current Date Display */}
          <h2 className="text-base sm:text-lg font-semibold text-slate-800">
            {viewMode === "month" ? formatMonthHeader(currentDate) : formatWeekHeader(currentDate)}
          </h2>
        </div>
      </div>

      {/* Error Message */}
      {dragError && (
        <div className="mx-4 mt-2 p-2 bg-red-50 border border-red-200 rounded-md flex items-center justify-between">
          <p className="text-sm text-red-600">{dragError}</p>
          <button
            onClick={() => setDragError(null)}
            className="text-red-500 hover:text-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 rounded p-1"
            aria-label="Dismiss error"
          >
            ✕
          </button>
        </div>
      )}

      {/* Calendar Grid */}
      <DragDropContext onDragEnd={handleDragEnd}>
        <div className={`grid ${viewMode === "month" ? "grid-cols-7" : "grid-cols-7"} bg-slate-50`} role="grid" aria-label={`Calendar ${viewMode} view`}>
          {/* Week Day Headers */}
          {weekDayNames.map((dayName, index) => (
            <div
              key={index}
              className="py-2 px-0.5 sm:px-1 text-center text-xs font-semibold text-slate-600 border-b border-r border-slate-200 last:border-r-0"
              aria-label={`Column for ${dayName}`}
            >
              <span className="hidden sm:inline">{dayName}</span>
              <span className="sm:hidden">{dayName.slice(0, 2)}</span>
            </div>
          ))}

          {/* Calendar Days */}
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
            />
          ))}
        </div>
      </DragDropContext>

      {/* Context Menu */}
      {contextMenu && (
        <div
          className="fixed bg-white border border-slate-300 rounded-lg shadow-lg py-1 z-50 min-w-[200px]"
          style={{ left: contextMenu.x, top: contextMenu.y }}
          onClick={(e) => e.stopPropagation()}
          role="menu"
          aria-label="Calendar day context menu"
          autoFocus
        >
          {onCreateTask && (
            <button
              onClick={handleContextMenuCreateTask}
              className="w-full px-4 py-2 text-left hover:bg-slate-100 flex items-center gap-2 focus:outline-none focus:bg-slate-100"
              role="menuitem"
              tabIndex={0}
            >
              <FaPlus size={14} aria-hidden="true" />
              Create Task for {format(contextMenu.date, "MMM d")}
            </button>
          )}
          <button
            onClick={handleContextMenuViewTasks}
            className="w-full px-4 py-2 text-left hover:bg-slate-100 focus:outline-none focus:bg-slate-100"
            role="menuitem"
            tabIndex={0}
          >
            View Tasks for {format(contextMenu.date, "MMM d")}
          </button>
        </div>
      )}

      {/* External Event Dialog */}
      {externalEventDialog && (
        <ExternalEventDialog
          event={externalEventDialog}
          onClose={() => setExternalEventDialog(null)}
          onSuccess={onExternalEventChange}
        />
      )}
    </div>
  );
}

interface CalendarDayCellProps {
  day: CalendarDay;
  isHovered: boolean;
  isSelected: boolean;
  onHover: (date: Date | null) => void;
  onDayClick: (day: CalendarDay) => void;
  onEventClick: (e: React.MouseEvent, event: CalendarEvent) => void;
  onContextMenu: (e: React.MouseEvent, day: CalendarDay) => void;
}

function CalendarDayCell({
  day,
  isHovered,
  isSelected,
  onHover,
  onDayClick,
  onEventClick,
  onContextMenu,
}: CalendarDayCellProps) {
  const visibleEvents = getVisibleEvents(day);
  const hiddenCount = getHiddenEventCount(day);

  const cellClasses = `
    min-h-[60px] sm:min-h-[80px] p-1 border-b border-r border-slate-200 last:border-r-0 cursor-pointer focus-within:ring-2 focus-within:ring-blue-300 focus:outline-none
    ${day.isToday ? "bg-blue-50" : ""}
    ${!day.isCurrentMonth ? "bg-slate-100 text-slate-400" : ""}
    ${isHovered ? "bg-slate-100" : "bg-white"}
    ${day.isToday && isHovered ? "bg-blue-100" : ""}
    ${isSelected ? "ring-2 ring-blue-500 ring-inset" : ""}
    ${day.isToday ? "focus-visible:ring-2 focus-visible:ring-blue-500" : "focus-visible:ring-2 focus-visible:ring-slate-300"}
  `;

  const dateNumberClasses = `
    text-sm font-medium mb-1 flex items-center justify-center
    ${day.isToday ? "text-blue-600" : ""}
  `;

  // Format date for droppableId and ARIA labels (YYYY-MM-DD format)
  const droppableId = format(day.date, "yyyy-MM-dd");
  const formattedDate = format(day.date, "MMMM d, yyyy");

  return (
    <Droppable droppableId={droppableId}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.droppableProps}
          className={cellClasses}
          onMouseEnter={() => onHover(day.date)}
          onMouseLeave={() => onHover(null)}
          onClick={() => onDayClick(day)}
          onContextMenu={(e) => onContextMenu(e, day)}
          role="gridcell"
          aria-label={`${format(day.date, "MMM d")}, ${day.taskCount} task${day.taskCount !== 1 ? "s" : ""}`}
          aria-selected={isSelected}
          tabIndex={0}
        >
          <div className={dateNumberClasses}>
            <span className="sr-only">{format(day.date, "MMMM d")}</span>
            <span aria-hidden="true">{format(day.date, "d")}</span>
          </div>

          {/* Task Count Indicator */}
          {day.taskCount > 0 && (
            <div className="space-y-0.5" role="list" aria-label={`${day.taskCount} task${day.taskCount !== 1 ? "s" : ""}`}>
              {visibleEvents.map((event, index) => (
                <Draggable
                  key={event.id}
                  draggableId={event.source === "task" && event.taskId ? event.taskId.toString() : `ext-${event.id}`}
                  index={index}
                  isDragDisabled={!isEventDraggable(event)}
                >
                  {(dragProvided, dragSnapshot) => {
                    const icon = getEventIcon(event);
                    const baseColor = getEventColor(event);
                    const isExternal = event.source === "external";
                    const customColor = event.color || "#6366f1";

                    return (
                      <div
                        ref={dragProvided.innerRef}
                        {...(isExternal ? {} : dragProvided.draggableProps)}
                        {...(isExternal ? {} : dragProvided.dragHandleProps)}
                        onClick={(e) => onEventClick(e, event)}
                        className={`
                          px-2 py-1 sm:px-3 sm:py-1.5 text-xs sm:text-sm rounded border truncate transition-shadow
                          ${baseColor}
                          ${isExternal ? "border-l-4" : ""}
                          ${isExternal ? "" : (dragSnapshot.isDragging ? "shadow-lg opacity-50" : "hover:opacity-80")}
                          focus-within:ring-2 focus-within:ring-blue-500 focus:outline-none
                          ${isExternal ? "cursor-pointer hover:opacity-80" : ""}
                        `}
                        style={{
                          ...dragProvided.draggableProps.style,
                          ...(isExternal ? { borderLeftColor: customColor } : {}),
                        }}
                        title={event.title}
                        role="listitem"
                        tabIndex={0}
                        aria-label={`${event.title}${isExternal ? " (external event)" : ""}${event.eventType ? ` - ${event.eventType}` : ""}${event.priority ? `, priority ${event.priority}` : ""}`}
                      >
                        {icon && <span className="mr-1" aria-hidden="true">{icon}</span>}
                        {event.title}
                      </div>
                    );
                  }}
                </Draggable>
              ))}

              {/* Hidden Events Indicator */}
              {hiddenCount > 0 && (
                <div
                  className="px-1 py-0.5 text-xs text-slate-500 italic"
                  aria-label={`${hiddenCount} more tasks not shown`}
                >
                  +{hiddenCount} more
                </div>
              )}
            </div>
          )}

          {/* Overdue Indicator */}
          {day.overdueCount > 0 && (
            <div className="mt-1" title={`${day.overdueCount} overdue task${day.overdueCount > 1 ? "s" : ""}`}>
              <span className="inline-block w-3 h-3 bg-red-500 rounded-full border-2 border-white" aria-hidden="true"></span>
              <span className="sr-only">{day.overdueCount} overdue</span>
            </div>
          )}

          {/* Completed Indicator */}
          {day.completedCount > 0 && day.overdueCount === 0 && (
            <div className="mt-1" title={`${day.completedCount} completed task${day.completedCount > 1 ? "s" : ""}`}>
              <span className="inline-block w-3 h-3 bg-green-500 rounded-full border-2 border-white" aria-hidden="true"></span>
              <span className="sr-only">{day.completedCount} completed</span>
            </div>
          )}

          {provided.placeholder}
        </div>
      )}
    </Droppable>
  );
}

// Export a wrapper component for easier integration
interface CalendarViewWrapperProps {
  tasks: Task[];
  externalEvents?: ExternalEvent[];
  currentDate: Date;
  viewMode: "month" | "week";
  onNavigate: (direction: "prev" | "next" | "today") => void;
  onViewModeChange: (viewMode: "month" | "week") => void;
  onTaskClick: (taskId: number) => void;
  onCreateTask?: (date: Date) => void;
  onTaskMoved?: () => void;
  onExternalEventChange?: () => void;
  timezone?: string;
}

export function CalendarViewWrapper({
  tasks,
  externalEvents,
  currentDate,
  viewMode,
  onNavigate,
  onViewModeChange,
  onTaskClick,
  onCreateTask,
  onTaskMoved,
  onExternalEventChange,
  timezone = "UTC",
}: CalendarViewWrapperProps) {
  const [selectedDate, setSelectedDate] = useState<Date | null>(null);
  const [showDayPopover, setShowDayPopover] = useState(false);
  const [externalEventDialog, setExternalEventDialog] = useState<CalendarEvent | null>(null);

  const handleDayClick = (date: Date, events: CalendarEvent[]) => {
    setSelectedDate(date);
    if (events.length === 1) {
      const event = events[0];
      if (event.source === "task" && event.taskId) {
        onTaskClick(event.taskId);
      } else if (event.source === "external") {
        setExternalEventDialog(event);
      }
    } else if (events.length > 1) {
      setShowDayPopover(true);
    }
  };

  const handleEventClick = (event: CalendarEvent) => {
    if (event.source === "task" && event.taskId) {
      onTaskClick(event.taskId);
    } else if (event.source === "external") {
      setExternalEventDialog(event);
    }
  };

  const handleExternalEventClick = (event: CalendarEvent) => {
    setExternalEventDialog(event);
  };

  const handleCreateTask = (date: Date) => {
    if (onCreateTask) {
      setSelectedDate(date);
      onCreateTask(date);
    }
  };

  // Memoize filtered events for the selected date
  const selectedDateEvents = useMemo(() => {
    if (!selectedDate) return [];
    return mergeCalendarEvents(tasksToCalendarEvents(tasks, timezone), externalEvents || []).filter(
      e => isSameDayInTimezone(e.date, selectedDate, timezone)
    );
  }, [selectedDate, tasks, externalEvents, timezone]);

  return (
    <div>
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
        onTaskMoved={onTaskMoved}
        onExternalEventClick={handleExternalEventClick}
        onExternalEventChange={onExternalEventChange}
        timezone={timezone || "UTC"}
      />

      {/* Day Popover */}
      {showDayPopover && selectedDate && (
        <DayPopover
          date={selectedDate}
          events={selectedDateEvents}
          onClose={() => setShowDayPopover(false)}
          onTaskClick={onTaskClick}
          onExternalEventClick={handleExternalEventClick}
          onCreateTask={() => handleCreateTask(selectedDate)}
        />
      )}

      {/* External Event Dialog */}
      {externalEventDialog && (
        <ExternalEventDialog
          event={externalEventDialog}
          onClose={() => setExternalEventDialog(null)}
          onSuccess={onExternalEventChange}
        />
      )}
    </div>
  );
}

interface DayPopoverProps {
  date: Date;
  events: CalendarEvent[];
  onClose: () => void;
  onTaskClick: (taskId: number) => void;
  onExternalEventClick: (event: CalendarEvent) => void;
  onCreateTask?: () => void;
}

function DayPopover({ date, events, onClose, onTaskClick, onExternalEventClick, onCreateTask }: DayPopoverProps) {
  // Separate external events from task events
  // Task events are deduplicated by taskId, external events are kept as-is
  const taskEventsById = new Map<number, CalendarEvent>();
  const externalEvents: CalendarEvent[] = [];

  events.forEach(event => {
    if (event.source === "external") {
      externalEvents.push(event);
    } else if (event.taskId) {
      // Deduplicate task events - prefer scheduled over due over completed
      const existing = taskEventsById.get(event.taskId);
      if (!existing || event.eventType === "scheduled" ||
        (existing.eventType !== "scheduled" && event.eventType === "due")) {
        taskEventsById.set(event.taskId, event);
      }
    }
  });

  const uniqueTaskEvents = Array.from(taskEventsById.values());
  const allEvents = [...uniqueTaskEvents, ...externalEvents];

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div
        className="bg-white rounded-lg shadow-xl max-w-md w-full max-h-[80vh] overflow-auto"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="day-popover-title"
      >
        <div className="p-4 border-b border-slate-200 flex items-center justify-between">
          <h3 id="day-popover-title" className="text-lg font-semibold">{format(date, "MMMM d, yyyy")}</h3>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-100 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            aria-label="Close dialog"
          >
            ✕
            <span className="sr-only">Close</span>
          </button>
        </div>
        <div className="p-4">
          {allEvents.length === 0 ? (
            <div className="text-center py-4">
              <p className="text-slate-500 mb-3">No tasks scheduled</p>
              {onCreateTask && (
                <button
                  onClick={() => {
                    onCreateTask();
                    onClose();
                  }}
                  className="px-6 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 min-h-[44px]"
                >
                  Create Task
                </button>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              {allEvents.map((event) => {
                const isExternal = event.source === "external";
                const customColor = event.color || "#6366f1";

                return (
                  <div
                    key={event.id}
                    onClick={() => {
                      if (isExternal) {
                        onExternalEventClick(event);
                      } else if (event.taskId) {
                        onTaskClick(event.taskId);
                      }
                    }}
                    className={`
                      p-3 rounded border cursor-pointer hover:opacity-80 focus:outline-none focus:ring-2 focus:ring-blue-500
                      ${getEventColor(event)}
                      ${isExternal ? "border-l-4" : ""}
                    `}
                    role="button"
                    tabIndex={0}
                    aria-label={isExternal
                      ? `View external event: ${event.title}`
                      : `View task: ${event.title} (${event.eventType}${event.priority ? `, priority ${event.priority}` : ""})`
                    }
                    style={isExternal ? { borderLeftColor: customColor } : undefined}
                  >
                    <div className="font-medium flex items-center gap-2">
                      {event.title}
                      {isExternal && <span className="text-xs">📅</span>}
                    </div>
                    <div className="text-xs mt-1 opacity-75 flex items-center gap-2">
                      {isExternal && !event.allDay && (
                        <span>
                          {format(event.date, "h:mm a")}
                          {event.endTime && ` - ${format(event.endTime, "h:mm a")}`}
                        </span>
                      )}
                      {!isExternal && (
                        <span>
                          {event.eventType === "scheduled" && "📅 Scheduled"}
                          {event.eventType === "due" && "⏰ Due"}
                          {event.eventType === "completed" && "✅ Completed"}
                        </span>
                      )}
                      {event.priority && (
                        <span className="inline-flex items-center gap-1">
                          <span className="w-2 h-2 rounded-full" aria-hidden="true" style={{
                            backgroundColor: event.priority === "A" ? "#f97316" :
                                          event.priority === "B" ? "#fbbf24" :
                                          event.priority === "C" ? "#60a5fa" : "#9ca3af"
                          }}></span>
                          <span className="sr-only">Priority {event.priority}</span>
                          <span>Priority {event.priority}</span>
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
              {onCreateTask && (
                <button
                  onClick={() => {
                    onCreateTask();
                    onClose();
                  }}
                  className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 flex items-center justify-center gap-2 min-h-[44px] mt-2"
                >
                  <FaPlus size={14} aria-hidden="true" />
                  Create Task
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

interface ExternalEventDialogProps {
  event: CalendarEvent;
  onClose: () => void;
  onSuccess?: () => void;
}

function ExternalEventDialog({ event, onClose, onSuccess }: ExternalEventDialogProps) {
  const [showLinkInput, setShowLinkInput] = useState(false);
  const [showCreateCard, setShowCreateCard] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Get timezone from user context or default to UTC
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

  const formatTime = (date: Date) => {
    return format(date, "h:mm a");
  };

  const formatDate = (date: Date) => {
    return format(date, "MMMM d, yyyy");
  };

  const handleLinkToCard = async (cardPK: number) => {
    if (!event.externalEventId) return;
    setError(null);
    setIsLoading(true);
    try {
      const { linkEventToCard } = await import("../../api/externalEvents");
      await linkEventToCard(event.externalEventId, { card_pk: cardPK });
      setIsLoading(false);
      onSuccess?.();
      onClose();
    } catch (error) {
      console.error("Failed to link event to card:", error);
      setError("Failed to link event to card. Please try again.");
      setIsLoading(false);
    }
  };

  const handleUnlinkFromCard = async () => {
    if (!event.externalEventId) return;
    setError(null);
    setIsLoading(true);
    try {
      const { unlinkEventFromCard } = await import("../../api/externalEvents");
      await unlinkEventFromCard(event.externalEventId);
      setIsLoading(false);
      onSuccess?.();
      onClose();
    } catch (error) {
      console.error("Failed to unlink event from card:", error);
      setError("Failed to unlink event from card. Please try again.");
      setIsLoading(false);
    }
  };

  const handleCreateCard = async (title: string, body: string) => {
    if (!event.externalEventId) return;
    setError(null);
    setIsLoading(true);
    try {
      const { createCard } = await import("../../api/cards");
      const newCard = await createCard({
        title: title || event.title,
        body: body || event.description || "",
      });
      // Now link the event to the new card
      const { linkEventToCard } = await import("../../api/externalEvents");
      await linkEventToCard(event.externalEventId, { card_pk: newCard.id });
      window.location.href = `/app/card/${newCard.card_id}`;
    } catch (error) {
      console.error("Failed to create card from event:", error);
      setError("Failed to create card from event. Please try again.");
      setIsLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div
        className="bg-white rounded-lg shadow-xl max-w-md w-full"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="external-event-title"
      >
        <div className="p-4 border-b flex justify-between items-center">
          <h3 id="external-event-title" className="text-lg font-semibold flex items-center gap-2">
            <span className="text-xl">📅</span>
            {event.title}
          </h3>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            aria-label="Close dialog"
            disabled={isLoading}
          >
            ✕
          </button>
        </div>
        <div className="p-4 space-y-3">
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-md">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}
          {event.description && (
            <div>
              <span className="text-sm text-gray-500">Description</span>
              <p className="mt-1 whitespace-pre-wrap">{event.description}</p>
            </div>
          )}
          {event.location && (
            <div>
              <span className="text-sm text-gray-500">Location</span>
              <p className="mt-1">{event.location}</p>
            </div>
          )}
          <div>
            <span className="text-sm text-gray-500">Time</span>
            <p className="mt-1">
              {event.allDay
                ? "All day"
                : `${formatTime(event.date)} (${formatDate(event.date)})`
              }
            </p>
          </div>
          {event.cardId && (
            <div>
              <span className="text-sm text-gray-500">Linked to Card</span>
              <p className="mt-1">
                <a
                  href={`/app/card/${event.cardPK}`}
                  className="text-blue-600 hover:text-blue-800"
                  onClick={(e) => e.stopPropagation()}
                >
                  [{event.cardId}]
                </a>
              </p>
            </div>
          )}
          <div className="flex gap-2 pt-2 flex-wrap">
            {event.externalUrl && (
              <a
                href={event.externalUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="px-4 py-2 bg-indigo-600 text-white rounded hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 flex items-center gap-2 min-h-[44px]"
              >
                Open in Calendar
              </a>
            )}
            {event.cardPK ? (
              <button
                onClick={handleUnlinkFromCard}
                disabled={isLoading}
                className="px-4 py-2 border border-red-300 text-red-600 rounded hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 min-h-[44px]"
              >
                {isLoading ? "Unlinking..." : "Unlink from Card"}
              </button>
            ) : (
              <>
                <button
                  onClick={() => setShowLinkInput(!showLinkInput)}
                  className="px-4 py-2 border rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 min-h-[44px]"
                >
                  Link to Card
                </button>
                <button
                  onClick={() => setShowCreateCard(!showCreateCard)}
                  className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 min-h-[44px]"
                >
                  Create Card
                </button>
              </>
            )}
            <button
              onClick={onClose}
              className="px-4 py-2 border rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 min-h-[44px]"
            >
              Close
            </button>
          </div>

          {/* Link to Card Input */}
          {showLinkInput && (
            <div className="mt-4 pt-4 border-t">
              <LinkToCardInput
                onLink={handleLinkToCard}
                onCancel={() => setShowLinkInput(false)}
                isLoading={isLoading}
              />
            </div>
          )}

          {/* Create Card Form */}
          {showCreateCard && (
            <div className="mt-4 pt-4 border-t">
              <CreateCardFromEventForm
                eventTitle={event.title}
                eventDescription={event.description}
                onCreate={handleCreateCard}
                onCancel={() => setShowCreateCard(false)}
                isLoading={isLoading}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Helper component for linking to card
interface LinkToCardInputProps {
  onLink: (cardPK: number) => void;
  onCancel: () => void;
  isLoading: boolean;
}

function LinkToCardInput({ onLink, onCancel, isLoading }: LinkToCardInputProps) {
  const handleSelectCard = (card: PartialCard) => {
    onLink(card.id);
  };

  return (
    <div>
      <h4 className="text-sm font-medium mb-2">Link to Card</h4>
      <BacklinkInputDropdownList
        onSelect={handleSelectCard}
        onSearch={() => {}}
        placeholder="Search cards by title or card ID..."
        autoFocus={true}
      />
      <div className="flex gap-2 mt-2">
        <button
          onClick={onCancel}
          disabled={isLoading}
          className="px-3 py-1 text-sm border rounded hover:bg-gray-50 min-h-[44px]"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

// Helper component for creating card from event
interface CreateCardFromEventFormProps {
  eventTitle: string;
  eventDescription: string | undefined;
  onCreate: (title: string, body: string) => void;
  onCancel: () => void;
  isLoading: boolean;
}

function CreateCardFromEventForm({
  eventTitle,
  eventDescription,
  onCreate,
  onCancel,
  isLoading
}: CreateCardFromEventFormProps) {
  const [title, setTitle] = useState(eventTitle);
  const [body, setBody] = useState(eventDescription || "");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onCreate(title, body);
  };

  return (
    <form onSubmit={handleSubmit}>
      <h4 className="text-sm font-medium mb-2">Create Card from Event</h4>
      <div className="space-y-2">
        <div>
          <label className="text-xs text-gray-500">Title</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            required
          />
        </div>
        <div>
          <label className="text-xs text-gray-500">Body</label>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            rows={3}
          />
        </div>
      </div>
      <div className="flex gap-2 mt-3">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[44px]"
        >
          {isLoading ? "Creating..." : "Create Card"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 border rounded hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[44px]"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
