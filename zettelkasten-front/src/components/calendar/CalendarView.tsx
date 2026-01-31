import React, { useState, useCallback, useEffect, useRef } from "react";
import { FaChevronLeft, FaChevronRight, FaPlus, FaChevronUp, FaChevronDown } from "react-icons/fa";
import { DragDropContext, Droppable, Draggable, DropResult } from "@hello-pangea/dnd";
import { Task } from "../../models/Task";
import {
  CalendarDay,
  CalendarEvent,
  CalendarViewType,
  tasksToCalendarEvents,
  getEventColor,
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
} from "../../utils/calendar";
import { format } from "date-fns";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";

interface CalendarViewProps {
  tasks: Task[];
  currentDate: Date;
  viewMode: CalendarViewType;
  onNavigate: (direction: "prev" | "next" | "today") => void;
  onViewModeChange: (viewMode: CalendarViewType) => void;
  onDayClick: (date: Date, events: CalendarEvent[]) => void;
  onEventClick: (event: CalendarEvent) => void;
  onCreateTask?: (date: Date) => void;
  onTaskMoved?: () => void;
  timezone?: string;
}

export function CalendarView({
  tasks,
  currentDate,
  viewMode,
  onNavigate,
  onViewModeChange,
  onDayClick,
  onEventClick,
  onCreateTask,
  onTaskMoved,
  timezone = "UTC",
}: CalendarViewProps) {
  const [hoveredDay, setHoveredDay] = useState<Date | null>(null);
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    date: Date;
  } | null>(null);
  const [selectedDayIndex, setSelectedDayIndex] = useState<number>(-1);
  const calendarRef = useRef<HTMLDivElement>(null);

  // Convert tasks to calendar events
  const events = tasksToCalendarEvents(tasks, timezone);

  // Generate the calendar grid based on view mode
  const grid = viewMode === "month"
    ? generateMonthGrid(currentDate)
    : generateWeekGrid(currentDate);

  // Populate events into the grid
  const days = populateDayEvents(grid, events);

  const weekDayNames = getWeekDayNames();

  const handleDayClick = (day: CalendarDay) => {
    onDayClick(day.date, day.events);
  };

  const handleEventClick = (e: React.MouseEvent, event: CalendarEvent) => {
    e.stopPropagation();
    onEventClick(event);
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
      const dayEvents = events.filter(
        e => format(e.date, "yyyy-MM-dd") === format(contextMenu.date, "yyyy-MM-dd")
      );
      onDayClick(contextMenu.date, dayEvents);
      closeContextMenu();
    }
  };

  // Close context menu on click outside
  useEffect(() => {
    const handleClickOutside = () => closeContextMenu();
    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, []);

  // Keyboard navigation
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    // Only handle keyboard when calendar is focused or no input is focused
    const target = e.target as HTMLElement;
    if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable) {
      return;
    }

    const grid = viewMode === "month" ? generateMonthGrid(currentDate) : generateWeekGrid(currentDate);

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
            e => format(e.date, "yyyy-MM-dd") === format(hoveredDay, "yyyy-MM-dd")
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
  }, [currentDate, events, hoveredDay, onDayClick, selectedDayIndex, viewMode]);

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  // Get setRefreshTasks from TaskContext for refreshing after drag-drop
  const { setRefreshTasks } = useTaskContext();

  // Drag and drop handler
  const handleDragEnd = useCallback(async (result: DropResult) => {
    if (!result.destination) return;

    const sourceDateStr = result.source.droppableId;
    const destDateStr = result.destination.droppableId;

    // No change if dropped in the same day
    if (sourceDateStr === destDateStr) return;

    const taskId = parseInt(result.draggableId, 10);
    const task = tasks.find(t => t.id === taskId);
    if (!task) return;

    // Calculate new scheduled date from destination
    const newScheduledDate = new Date(destDateStr);

    // Update task with new scheduled date
    const updatedTask = {
      ...task,
      scheduled_date: newScheduledDate,
    };

    try {
      // Persist changes
      const response = await saveExistingTask(updatedTask);
      if (!("error" in response)) {
        setRefreshTasks(true);
        onTaskMoved?.();
      }
    } catch (err) {
      console.error("Failed to reschedule task after drag-and-drop:", err);
    }
  }, [tasks, onTaskMoved, setRefreshTasks]);

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
                  draggableId={event.taskId.toString()}
                  index={index}
                >
                  {(dragProvided, dragSnapshot) => (
                    <div
                      ref={dragProvided.innerRef}
                      {...dragProvided.draggableProps}
                      {...dragProvided.dragHandleProps}
                      onClick={(e) => onEventClick(e, event)}
                      className={`
                        px-2 py-1 sm:px-3 sm:py-1.5 text-xs sm:text-sm rounded border truncate transition-shadow
                        ${getEventColor(event)}
                        ${dragSnapshot.isDragging ? "shadow-lg opacity-50" : "hover:opacity-80"}
                        focus-within:ring-2 focus-within:ring-blue-500 focus:outline-none
                      `}
                      style={dragProvided.draggableProps.style}
                      title={event.title}
                      role="listitem"
                      tabIndex={0}
                      aria-label={`${event.title} - ${event.eventType}${event.priority ? `, priority ${event.priority}` : ""}`}
                    >
                      {event.title}
                    </div>
                  )}
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
  currentDate: Date;
  viewMode: "month" | "week";
  onNavigate: (direction: "prev" | "next" | "today") => void;
  onViewModeChange: (viewMode: "month" | "week") => void;
  onTaskClick: (taskId: number) => void;
  onCreateTask?: (date: Date) => void;
  onTaskMoved?: () => void;
  timezone?: string;
}

export function CalendarViewWrapper({
  tasks,
  currentDate,
  viewMode,
  onNavigate,
  onViewModeChange,
  onTaskClick,
  onCreateTask,
  onTaskMoved,
  timezone,
}: CalendarViewWrapperProps) {
  const [selectedDate, setSelectedDate] = useState<Date | null>(null);
  const [showDayPopover, setShowDayPopover] = useState(false);

  const handleDayClick = (date: Date, events: CalendarEvent[]) => {
    setSelectedDate(date);
    if (events.length === 1) {
      onTaskClick(events[0].taskId);
    } else if (events.length > 1) {
      setShowDayPopover(true);
    }
  };

  const handleEventClick = (event: CalendarEvent) => {
    onTaskClick(event.taskId);
  };

  const handleCreateTask = (date: Date) => {
    if (onCreateTask) {
      setSelectedDate(date);
      onCreateTask(date);
    }
  };

  return (
    <div>
      <CalendarView
        tasks={tasks}
        currentDate={currentDate}
        viewMode={viewMode}
        onNavigate={onNavigate}
        onViewModeChange={onViewModeChange}
        onDayClick={handleDayClick}
        onEventClick={handleEventClick}
        onCreateTask={onCreateTask}
        onTaskMoved={onTaskMoved}
        timezone={timezone}
      />

      {/* Day Popover - to be implemented */}
      {showDayPopover && selectedDate && (
        <DayPopover
          date={selectedDate}
          events={tasksToCalendarEvents(tasks, timezone).filter(
            e => format(e.date, "yyyy-MM-dd") === format(selectedDate, "yyyy-MM-dd")
          )}
          onClose={() => setShowDayPopover(false)}
          onTaskClick={onTaskClick}
          onCreateTask={() => handleCreateTask(selectedDate)}
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
  onCreateTask?: () => void;
}

function DayPopover({ date, events, onClose, onTaskClick, onCreateTask }: DayPopoverProps) {
  // Group events by task to avoid duplicates
  const uniqueEvents = Array.from(groupEventsByTask(events).values());

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
          {uniqueEvents.length === 0 ? (
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
              {uniqueEvents.map((event) => (
                <div
                  key={event.id}
                  onClick={() => onTaskClick(event.taskId)}
                  className={`
                    p-3 rounded border cursor-pointer hover:opacity-80 focus:outline-none focus:ring-2 focus:ring-blue-500
                    ${getEventColor(event)}
                  `}
                  role="button"
                  tabIndex={0}
                  aria-label={`View task: ${event.title} (${event.eventType}${event.priority ? `, priority ${event.priority}` : ""})`}
                >
                  <div className="font-medium">{event.title}</div>
                  <div className="text-xs mt-1 opacity-75 flex items-center gap-2">
                    <span>
                      {event.eventType === "scheduled" && "📅 Scheduled"}
                      {event.eventType === "due" && "⏰ Due"}
                      {event.eventType === "completed" && "✅ Completed"}
                    </span>
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
              ))}
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
