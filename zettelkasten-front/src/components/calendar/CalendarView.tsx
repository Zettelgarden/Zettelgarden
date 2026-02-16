import React, { useState, useCallback, useEffect, useRef, useMemo } from "react";
import { FaChevronLeft, FaChevronRight, FaPlus, FaChevronUp, FaChevronDown } from "react-icons/fa";
import { DragDropContext, Droppable, Draggable, DropResult } from "@hello-pangea/dnd";
import { toZonedTime } from "date-fns-tz";
import { format } from "date-fns";
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
  onCreateEvent?: (date: Date) => void;
  onTaskMoved?: () => void;
  onExternalEventClick?: (event: CalendarEvent) => void;
  onExternalEventChange?: () => void;
  timezone?: string;
  calendarSettingsButton?: React.ReactNode;
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
  onCreateEvent,
  onTaskMoved,
  onExternalEventClick,
  onExternalEventChange,
  timezone = "UTC",
  calendarSettingsButton,
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

  const handleContextMenuCreateEvent = () => {
    if (contextMenu && onCreateEvent) {
      onCreateEvent(contextMenu.date);
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
          {/* Left Section: View Mode Toggle + Settings Button */}
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
            {calendarSettingsButton}
          </div>

          {/* Mobile View Mode Indicator + Settings Button */}
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
            {calendarSettingsButton}
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
              onCreateTask={onCreateTask}
              onCreateEvent={onCreateEvent}
              viewMode={viewMode}
              timezone={timezone}
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
          {onCreateEvent && (
            <button
              onClick={handleContextMenuCreateEvent}
              className="w-full px-4 py-2 text-left hover:bg-slate-100 flex items-center gap-2 focus:outline-none focus:bg-slate-100"
              role="menuitem"
              tabIndex={0}
            >
              <FaPlus size={14} aria-hidden="true" />
              Create Event for {format(contextMenu.date, "MMM d")}
            </button>
          )}
          <button
            onClick={handleContextMenuViewTasks}
            className="w-full px-4 py-2 text-left hover:bg-slate-100 focus:outline-none focus:bg-slate-100"
            role="menuitem"
            tabIndex={0}
          >
            View Events for {format(contextMenu.date, "MMM d")}
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
  onCreateTask?: (date: Date) => void;
  onCreateEvent?: (date: Date) => void;
  viewMode: CalendarViewType;
  timezone: string;
}

function CalendarDayCell({
  day,
  isHovered,
  isSelected,
  onHover,
  onDayClick,
  onEventClick,
  onContextMenu,
  onCreateTask,
  onCreateEvent,
  viewMode,
  timezone,
}: CalendarDayCellProps) {
  const visibleEvents = getVisibleEvents(day, viewMode);
  const hiddenCount = getHiddenEventCount(day, viewMode);

  // Current time state - updates every minute
  const [currentTime, setCurrentTime] = useState<Date>(new Date());

  useEffect(() => {
    // Update current time every minute
    const interval = setInterval(() => {
      setCurrentTime(new Date());
    }, 60000); // 60 seconds

    return () => clearInterval(interval);
  }, []);

  const handleDoubleClick = () => {
    if (onCreateEvent) {
      onCreateEvent(day.date);
    }
  };

  // Calculate current time position for week view indicator
  const getCurrentTimePosition = (): { percentage: number; timeLabel: string } | null => {
    // Only show in week view for today
    if (viewMode !== "week" || !day.isToday) {
      return null;
    }

    // Get timed events (excluding all-day events)
    const timedEvents = visibleEvents.filter(e => !e.allDay && e.date);

    if (timedEvents.length === 0) {
      // No timed events - use default range 6AM-10PM
      const now = toZonedTime(currentTime, timezone);
      const hours = now.getHours();
      const minutes = now.getMinutes();
      const totalMinutes = hours * 60 + minutes;
      const startMinutes = 6 * 60; // 6AM
      const endMinutes = 22 * 60; // 10PM

      if (totalMinutes < startMinutes || totalMinutes > endMinutes) {
        // Current time is outside default range
        return null;
      }

      const percentage = ((totalMinutes - startMinutes) / (endMinutes - startMinutes)) * 100;
      return {
        percentage: Math.max(0, Math.min(100, percentage)),
        timeLabel: format(now, "h:mm a"),
      };
    }

    // Calculate range based on visible timed events
    const eventMinutes = timedEvents.flatMap(e => {
      const start = toZonedTime(e.date, timezone);
      const startMinutes = start.getHours() * 60 + start.getMinutes();
      const end = e.endTime ? toZonedTime(e.endTime, timezone) : start;
      const endMinutes = end.getHours() * 60 + end.getMinutes();
      return [startMinutes, endMinutes];
    });

    const minMinutes = Math.min(...eventMinutes);
    const maxMinutes = Math.max(...eventMinutes);

    // Add buffer (30 minutes before first event, 30 minutes after last event)
    const startMinutes = Math.max(0, minMinutes - 30);
    const endMinutes = Math.min(24 * 60, maxMinutes + 30);

    // Get current time in timezone
    const now = toZonedTime(currentTime, timezone);
    const totalMinutes = now.getHours() * 60 + now.getMinutes();

    // Clamp to range
    const clampedMinutes = Math.max(startMinutes, Math.min(endMinutes, totalMinutes));
    const percentage = ((clampedMinutes - startMinutes) / (endMinutes - startMinutes)) * 100;

    return {
      percentage: Math.max(0, Math.min(100, percentage)),
      timeLabel: format(now, "h:mm a"),
    };
  };

  const nowPosition = getCurrentTimePosition();

  // Week view uses taller cells to utilize available vertical space
  const cellClasses = `
    ${viewMode === "week" ? "min-h-[400px]" : "min-h-[60px] sm:min-h-[80px]"} p-1 border-b border-r border-slate-200 last:border-r-0 cursor-pointer focus-within:ring-2 focus-within:ring-blue-300 focus:outline-none relative group
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
          onDoubleClick={handleDoubleClick}
          role="gridcell"
          aria-label={`${format(day.date, "MMM d")}, ${day.taskCount} task${day.taskCount !== 1 ? "s" : ""}`}
          aria-selected={isSelected}
          tabIndex={0}
        >
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
          <div className={dateNumberClasses}>
            <span className="sr-only">{format(day.date, "MMMM d")}</span>
            <span aria-hidden="true">{format(day.date, "d")}</span>
          </div>

          {/* Current Time Indicator - only in week view for today */}
          {nowPosition && (
            <div
              className="absolute left-0 right-0 pointer-events-none z-10"
              style={{ top: `${nowPosition.percentage}%` }}
              aria-label={`Current time: ${nowPosition.timeLabel}`}
            >
              <div className="flex items-center">
                <span className="text-[10px] text-red-500 font-medium bg-white/80 px-1 rounded shadow-sm ml-1">
                  {nowPosition.timeLabel}
                </span>
                <div className="flex-1 h-[2px] bg-red-500"></div>
              </div>
              {/* Small dot at the end for visibility */}
              <div className="absolute right-1 top-1/2 -translate-y-1/2 w-2 h-2 bg-red-500 rounded-full animate-pulse"></div>
            </div>
          )}

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
                        <span className="truncate flex-1">{event.title}</span>
                        {isExternal && !event.allDay && (
                          <span className="ml-1 text-xs opacity-75 whitespace-nowrap" aria-hidden="true">
                            {format(event.date, "h:mm a")}
                          </span>
                        )}
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
  onCreateEvent?: (date: Date) => void;
  onTaskMoved?: () => void;
  onExternalEventChange?: () => void;
  timezone?: string;
  calendarSettingsButton?: React.ReactNode;
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
  onCreateEvent,
  onTaskMoved,
  onExternalEventChange,
  timezone = "UTC",
  calendarSettingsButton,
}: CalendarViewWrapperProps) {
  const [selectedDate, setSelectedDate] = useState<Date | null>(null);
  const [showDayPopover, setShowDayPopover] = useState(false);
  const [externalEventDialog, setExternalEventDialog] = useState<CalendarEvent | null>(null);

  const handleDayClick = (date: Date, events: CalendarEvent[]) => {
    setSelectedDate(date);
    setShowDayPopover(true);
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

  const handleCreateEvent = (date: Date) => {
    if (onCreateEvent) {
      setSelectedDate(date);
      onCreateEvent(date);
    }
  };

  const handleNavigateDay = (direction: number) => {
    if (!selectedDate) return;
    const newDate = new Date(selectedDate);
    newDate.setDate(newDate.getDate() + direction);
    setSelectedDate(newDate);
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
        onCreateEvent={onCreateEvent}
        onTaskMoved={onTaskMoved}
        onExternalEventClick={handleExternalEventClick}
        onExternalEventChange={onExternalEventChange}
        timezone={timezone || "UTC"}
        calendarSettingsButton={calendarSettingsButton}
      />

      {/* Day Popover */}
      {showDayPopover && selectedDate && (
        <DayPopover
          date={selectedDate}
          events={selectedDateEvents}
          onClose={() => setShowDayPopover(false)}
          onTaskClick={onTaskClick}
          onExternalEventClick={handleExternalEventClick}
          onCreateEvent={() => handleCreateEvent(selectedDate)}
          onNavigateDay={handleNavigateDay}
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
  onCreateEvent?: () => void;
  onNavigateDay?: (direction: number) => void;
}

function DayPopover({ date, events, onClose, onTaskClick, onExternalEventClick, onCreateEvent, onNavigateDay }: DayPopoverProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const [timeView, setTimeView] = useState(false);

  // Time grid configuration
  const timeGridStart = 6; // 6 AM
  const timeGridEnd = 22;  // 10 PM
  const hourHeight = 40;   // pixels per hour

  // Focus management and escape key handler
  useEffect(() => {
    // Focus the close button when dialog opens
    closeButtonRef.current?.focus();

    // Handle escape key and focus trap
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
        return;
      }

      // Simple focus trap - keep Tab/Shift+Tab within the dialog
      if (e.key === 'Tab' && dialogRef.current) {
        const focusableElements = dialogRef.current.querySelectorAll(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        const firstElement = focusableElements[0] as HTMLElement;
        const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement;

        if (e.shiftKey && document.activeElement === firstElement) {
          e.preventDefault();
          lastElement?.focus();
        } else if (!e.shiftKey && document.activeElement === lastElement) {
          e.preventDefault();
          firstElement?.focus();
        }
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

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
  // Sort events chronologically - all-day events first, then by time
  const allEvents = [...uniqueTaskEvents, ...externalEvents].sort((a, b) => {
    // If one is all-day and one isn't, all-day comes first
    if (a.allDay && !b.allDay) return -1;
    if (!a.allDay && b.allDay) return 1;
    // Both have times or both don't - sort by date
    return a.date.getTime() - b.date.getTime();
  });

  // Separate all-day and timed events for time grid view
  const allDayEvents = allEvents.filter(e => e.allDay);
  const timedEvents = allEvents.filter(e => !e.allDay);

  // Calculate position for timed events in the grid
  const getEventPosition = (event: CalendarEvent) => {
    const hour = event.date.getHours();
    const minute = event.date.getMinutes();
    const totalMinutes = hour * 60 + minute;
    const startMinutes = timeGridStart * 60;
    const top = ((totalMinutes - startMinutes) / 60) * hourHeight;

    const endHour = event.endTime?.getHours() ?? hour + 1;
    const endMinute = event.endTime?.getMinutes() ?? 0;
    const endTotalMinutes = endHour * 60 + endMinute;
    const height = Math.max(((endTotalMinutes - startMinutes) / 60) * hourHeight - top, hourHeight / 2);

    return { top, height };
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div
        ref={dialogRef}
        className="bg-white rounded-lg shadow-xl max-w-md w-full max-h-[80vh] overflow-auto"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="day-popover-title"
      >
        <div className="p-3 border-b border-slate-200">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-1">
              {onNavigateDay && (
                <button
                  onClick={() => onNavigateDay(-1)}
                  className="p-1 hover:bg-slate-100 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[32px] min-w-[32px]"
                  aria-label="Previous day"
                  title="Previous day"
                >
                  <FaChevronLeft size={14} aria-hidden="true" />
                </button>
              )}
              <h3 id="day-popover-title" className="text-base font-semibold">{format(date, "MMM d, yyyy")}</h3>
              {onNavigateDay && (
                <button
                  onClick={() => onNavigateDay(1)}
                  className="p-1 hover:bg-slate-100 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[32px] min-w-[32px]"
                  aria-label="Next day"
                  title="Next day"
                >
                  <FaChevronRight size={14} aria-hidden="true" />
                </button>
              )}
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setTimeView(!timeView)}
                className={`p-1 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[32px] min-w-[32px] text-xs font-medium ${
                  timeView ? "bg-blue-100 text-blue-700" : "hover:bg-slate-100 text-slate-600"
                }`}
                aria-label={timeView ? "Switch to list view" : "Switch to time view"}
                title={timeView ? "List view" : "Time grid view"}
              >
                {timeView ? "☰" : "⏱"}
              </button>
              <button
                ref={closeButtonRef}
                onClick={onClose}
                className="p-1 hover:bg-slate-100 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Close dialog"
              >
                ✕
                <span className="sr-only">Close</span>
              </button>
            </div>
          </div>
          {events.length > 0 && (
            <p id="day-popover-count" className="text-xs text-slate-500">
              {events.length} {events.length === 1 ? 'event' : 'events'} scheduled
            </p>
          )}
        </div>
        <div className="p-3">
          {allEvents.length === 0 ? (
            <div className="text-center py-4">
              <p className="text-slate-500 mb-3">No events scheduled</p>
              {onCreateEvent && (
                <button
                  onClick={() => {
                    onCreateEvent();
                    // Keep dialog open to allow creating multiple events
                  }}
                  className="px-6 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 min-h-[44px]"
                >
                  Create Event
                </button>
              )}
            </div>
          ) : timeView ? (
            // Time grid view
            <div className="space-y-2">
              {/* All-day events */}
              {allDayEvents.length > 0 && (
                <div className="space-y-1">
                  {allDayEvents.map((event) => {
                    const isExternal = event.source === "external";
                    const customColor = event.color || "#6366f1";
                    return (
                      <button
                        type="button"
                        key={event.id}
                        onClick={() => {
                          if (isExternal) {
                            onExternalEventClick(event);
                          } else if (event.taskId) {
                            onTaskClick(event.taskId);
                          }
                        }}
                        className={`
                          w-full text-left px-2 py-1 rounded border hover:opacity-80 focus:outline-none focus:ring-2 focus:ring-blue-500 flex items-center justify-between gap-2
                          ${getEventColor(event)}
                          ${isExternal ? "border-l-4" : ""}
                        `}
                        aria-label={isExternal
                          ? `View external event: ${event.title}`
                          : `View task: ${event.title} (${event.eventType}${event.priority ? `, priority ${event.priority}` : ""})`
                        }
                        style={isExternal ? { borderLeftColor: customColor } : undefined}
                      >
                        <span className="text-sm truncate flex-1">{event.title}</span>
                        <div className="flex items-center gap-1 shrink-0">
                          {!isExternal && (
                            <>
                              {event.eventType === "scheduled" && <span className="w-1 h-1 rounded-full bg-blue-500" aria-hidden="true"></span>}
                              {event.eventType === "due" && <span className="w-1 h-1 rounded-full bg-amber-500" aria-hidden="true"></span>}
                              {event.eventType === "completed" && <span className="w-1 h-1 rounded-full bg-green-500" aria-hidden="true"></span>}
                            </>
                          )}
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}

              {/* Time grid */}
              <div className="relative border rounded bg-slate-50" style={{ height: `${(timeGridEnd - timeGridStart) * hourHeight}px` }}>
                {/* Hour lines */}
                {Array.from({ length: timeGridEnd - timeGridStart }, (_, i) => (
                  <div
                    key={i}
                    className="absolute left-0 right-0 border-t border-slate-200 flex items-center"
                    style={{ top: `${i * hourHeight}px` }}
                  >
                    <span className="text-[10px] text-slate-400 w-10 text-center pt-0.5">
                      {format(new Date().setHours(timeGridStart + i, 0, 0, 0), "ha")}
                    </span>
                  </div>
                ))}

                {/* Timed events */}
                {timedEvents.map((event) => {
                  const isExternal = event.source === "external";
                  const customColor = event.color || "#6366f1";
                  const { top, height } = getEventPosition(event);

                  return (
                    <button
                      type="button"
                      key={event.id}
                      onClick={() => {
                        if (isExternal) {
                          onExternalEventClick(event);
                        } else if (event.taskId) {
                          onTaskClick(event.taskId);
                        }
                      }}
                      className={`
                        absolute left-12 right-1 rounded border-l-2 px-2 py-1 text-left overflow-hidden hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-blue-500
                        ${getEventColor(event)}
                      `}
                      style={{
                        top: `${top}px`,
                        height: `${height}px`,
                        minHeight: '20px',
                        borderLeftColor: isExternal ? customColor : undefined,
                      }}
                      aria-label={`${format(event.date, "h:mm a")} - ${event.endTime ? format(event.endTime, "h:mm a") : ''}: ${event.title}`}
                    >
                      <div className="text-xs font-medium truncate">{event.title}</div>
                      {height > 30 && (
                        <div className="text-[10px] opacity-75 truncate">
                          {format(event.date, "h:mm a")}{event.endTime && ` - ${format(event.endTime, "h:mm a")}`}
                        </div>
                      )}
                    </button>
                  );
                })}
              </div>

              {onCreateEvent && (
                <button
                  onClick={() => {
                    onCreateEvent();
                    // Keep dialog open to allow creating multiple events
                  }}
                  className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 flex items-center justify-center gap-2 min-h-[44px] mt-2"
                >
                  <FaPlus size={14} aria-hidden="true" />
                  Create Event
                </button>
              )}
            </div>
          ) : (
            // List view
            <div className="space-y-1">
              {allEvents.map((event) => {
                const isExternal = event.source === "external";
                const customColor = event.color || "#6366f1";

                return (
                  <button
                    type="button"
                    key={event.id}
                    onClick={() => {
                      if (isExternal) {
                        onExternalEventClick(event);
                      } else if (event.taskId) {
                        onTaskClick(event.taskId);
                      }
                    }}
                    className={`
                      w-full text-left px-2 py-1.5 rounded border hover:opacity-80 focus:outline-none focus:ring-2 focus:ring-blue-500 flex items-center justify-between gap-2
                      ${getEventColor(event)}
                      ${isExternal ? "border-l-4" : ""}
                    `}
                    aria-label={isExternal
                      ? `View external event: ${event.title}`
                      : `View task: ${event.title} (${event.eventType}${event.priority ? `, priority ${event.priority}` : ""})`
                    }
                    style={isExternal ? { borderLeftColor: customColor } : undefined}
                  >
                    <span className="font-medium text-sm truncate flex-1">{event.title}</span>
                    <div className="flex items-center gap-1.5 shrink-0">
                      {isExternal && !event.allDay && (
                        <span className="text-[10px] opacity-75 whitespace-nowrap">
                          {format(event.date, "h:mm a")}
                        </span>
                      )}
                      {!isExternal && (
                        <>
                          {event.eventType === "scheduled" && (
                            <span className="w-1.5 h-1.5 rounded-full bg-blue-500" aria-hidden="true" title="Scheduled"></span>
                          )}
                          {event.eventType === "due" && (
                            <span className="w-1.5 h-1.5 rounded-full bg-amber-500" aria-hidden="true" title="Due"></span>
                          )}
                          {event.eventType === "completed" && (
                            <span className="w-1.5 h-1.5 rounded-full bg-green-500" aria-hidden="true" title="Completed"></span>
                          )}
                          {event.priority && (
                            <span className="w-1.5 h-1.5 rounded-full" aria-hidden="true" title={`Priority ${event.priority}`} style={{
                              backgroundColor: event.priority === "A" ? "#f97316" :
                                            event.priority === "B" ? "#fbbf24" :
                                            event.priority === "C" ? "#60a5fa" : "#9ca3af"
                            }}></span>
                          )}
                        </>
                      )}
                    </div>
                  </button>
                );
              })}
              {onCreateEvent && (
                <button
                  onClick={() => {
                    onCreateEvent();
                    // Keep dialog open to allow creating multiple events
                  }}
                  className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 flex items-center justify-center gap-2 min-h-[44px] mt-2"
                >
                  <FaPlus size={14} aria-hidden="true" />
                  Create Event
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
            className="w-full min-h-[120px] max-h-[30vh] sm:max-h-none px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 resize-y"
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
