import React, { useState } from "react";
import { FaChevronLeft, FaChevronRight } from "react-icons/fa";
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

interface CalendarViewProps {
  tasks: Task[];
  currentDate: Date;
  viewMode: CalendarViewType;
  onNavigate: (direction: "prev" | "next" | "today") => void;
  onViewModeChange: (viewMode: CalendarViewType) => void;
  onDayClick: (date: Date, events: CalendarEvent[]) => void;
  onEventClick: (event: CalendarEvent) => void;
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
  timezone = "UTC",
}: CalendarViewProps) {
  const [hoveredDay, setHoveredDay] = useState<Date | null>(null);

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

  return (
    <div className="bg-white border border-slate-300 rounded-lg overflow-hidden">
      {/* Calendar Header */}
      <div className="bg-slate-100 px-4 py-3 border-b border-slate-300">
        <div className="flex items-center justify-between">
          {/* View Mode Toggle */}
          <div className="flex items-center gap-2">
            <button
              onClick={() => onViewModeChange("month")}
              className={`px-3 py-1 text-sm rounded-md ${
                viewMode === "month"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-slate-700 border border-slate-300 hover:bg-slate-50"
              }`}
            >
              Month
            </button>
            <button
              onClick={() => onViewModeChange("week")}
              className={`px-3 py-1 text-sm rounded-md ${
                viewMode === "week"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-slate-700 border border-slate-300 hover:bg-slate-50"
              }`}
            >
              Week
            </button>
          </div>

          {/* Navigation Controls */}
          <div className="flex items-center gap-2">
            <button
              onClick={() => onNavigate("prev")}
              className="p-1 hover:bg-slate-200 rounded"
              title="Previous"
            >
              <FaChevronLeft size={20} />
            </button>
            <button
              onClick={() => onNavigate("today")}
              className="px-3 py-1 text-sm bg-white border border-slate-300 rounded-md hover:bg-slate-50"
            >
              Today
            </button>
            <button
              onClick={() => onNavigate("next")}
              className="p-1 hover:bg-slate-200 rounded"
              title="Next"
            >
              <FaChevronRight size={20} />
            </button>
          </div>

          {/* Current Date Display */}
          <h2 className="text-lg font-semibold text-slate-800">
            {viewMode === "month" ? formatMonthHeader(currentDate) : formatWeekHeader(currentDate)}
          </h2>
        </div>
      </div>

      {/* Calendar Grid */}
      <div className={`grid ${viewMode === "month" ? "grid-cols-7" : "grid-cols-7"} bg-slate-50`}>
        {/* Week Day Headers */}
        {weekDayNames.map((dayName, index) => (
          <div
            key={index}
            className="py-2 px-1 text-center text-xs font-semibold text-slate-600 border-b border-r border-slate-200 last:border-r-0"
          >
            {dayName}
          </div>
        ))}

        {/* Calendar Days */}
        {days.map((day, index) => (
          <CalendarDayCell
            key={index}
            day={day}
            isHovered={hoveredDay ? day.date.getTime() === hoveredDay.getTime() : false}
            onHover={setHoveredDay}
            onDayClick={handleDayClick}
            onEventClick={handleEventClick}
          />
        ))}
      </div>
    </div>
  );
}

interface CalendarDayCellProps {
  day: CalendarDay;
  isHovered: boolean;
  onHover: (date: Date | null) => void;
  onDayClick: (day: CalendarDay) => void;
  onEventClick: (e: React.MouseEvent, event: CalendarEvent) => void;
}

function CalendarDayCell({ day, isHovered, onHover, onDayClick, onEventClick }: CalendarDayCellProps) {
  const visibleEvents = getVisibleEvents(day);
  const hiddenCount = getHiddenEventCount(day);

  const cellClasses = `
    min-h-[80px] p-1 border-b border-r border-slate-200 last:border-r-0 cursor-pointer
    ${day.isToday ? "bg-blue-50" : ""}
    ${!day.isCurrentMonth ? "bg-slate-100 text-slate-400" : ""}
    ${isHovered ? "bg-slate-100" : "bg-white"}
    ${day.isToday && isHovered ? "bg-blue-100" : ""}
  `;

  const dateNumberClasses = `
    text-sm font-medium mb-1
    ${day.isToday ? "text-blue-600" : ""}
  `;

  return (
    <div
      className={cellClasses}
      onMouseEnter={() => onHover(day.date)}
      onMouseLeave={() => onHover(null)}
      onClick={() => onDayClick(day)}
    >
      <div className={dateNumberClasses}>
        {format(day.date, "d")}
      </div>

      {/* Task Count Indicator */}
      {day.taskCount > 0 && (
        <div className="space-y-0.5">
          {visibleEvents.map((event) => (
            <div
              key={event.id}
              onClick={(e) => onEventClick(e, event)}
              className={`
                px-1 py-0.5 text-xs rounded border truncate
                ${getEventColor(event)}
              `}
              title={event.title}
            >
              {event.title}
            </div>
          ))}

          {/* Hidden Events Indicator */}
          {hiddenCount > 0 && (
            <div className="px-1 py-0.5 text-xs text-slate-500 italic">
              +{hiddenCount} more
            </div>
          )}
        </div>
      )}

      {/* Overdue Indicator */}
      {day.overdueCount > 0 && (
        <div className="mt-1">
          <span className="inline-block w-2 h-2 bg-red-500 rounded-full" title={`${day.overdueCount} overdue`} />
        </div>
      )}

      {/* Completed Indicator */}
      {day.completedCount > 0 && day.overdueCount === 0 && (
        <div className="mt-1">
          <span className="inline-block w-2 h-2 bg-green-500 rounded-full" title={`${day.completedCount} completed`} />
        </div>
      )}
    </div>
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
  timezone?: string;
}

export function CalendarViewWrapper({
  tasks,
  currentDate,
  viewMode,
  onNavigate,
  onViewModeChange,
  onTaskClick,
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
}

function DayPopover({ date, events, onClose, onTaskClick }: DayPopoverProps) {
  // Group events by task to avoid duplicates
  const uniqueEvents = Array.from(groupEventsByTask(events).values());

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-white rounded-lg shadow-xl max-w-md w-full max-h-[80vh] overflow-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-slate-200 flex items-center justify-between">
          <h3 className="text-lg font-semibold">{format(date, "MMMM d, yyyy")}</h3>
          <button
            onClick={onClose}
            className="p-1 hover:bg-slate-100 rounded"
          >
            ✕
          </button>
        </div>
        <div className="p-4">
          {uniqueEvents.length === 0 ? (
            <p className="text-slate-500 text-center py-4">No tasks scheduled</p>
          ) : (
            <div className="space-y-2">
              {uniqueEvents.map((event) => (
                <div
                  key={event.id}
                  onClick={() => onTaskClick(event.taskId)}
                  className={`
                    p-2 rounded border cursor-pointer hover:opacity-80
                    ${getEventColor(event)}
                  `}
                >
                  <div className="font-medium">{event.title}</div>
                  <div className="text-xs opacity-75">
                    {event.eventType === "scheduled" && "Scheduled"}
                    {event.eventType === "due" && "Due"}
                    {event.eventType === "completed" && "Completed"}
                    {event.priority && ` • Priority ${event.priority}`}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
