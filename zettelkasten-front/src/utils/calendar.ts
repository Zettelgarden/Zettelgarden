import { format, startOfMonth, endOfMonth, startOfWeek, endOfWeek, eachDayOfInterval, addMonths, subMonths, addWeeks, subWeeks, startOfDay, isSameDay, isSameMonth } from "date-fns";
import { CalendarDay, CalendarEvent } from "../models/CalendarEvent";
import { isToday } from "../models/CalendarEvent";
import { ExternalEvent } from "../models/ExternalEvent";

/**
 * Get the start of the week day (0 = Sunday, 1 = Monday, etc.)
 * Can be configured based on user preference
 */
export function getFirstDayOfWeek(): 0 | 1 | 6 {
  // Default to Sunday (0). Could be made configurable via user settings
  return 0;
}

/**
 * Generate a 6-week grid for a given month
 * This ensures all days of the month are visible plus some days from adjacent months
 */
export function generateMonthGrid(date: Date): CalendarDay[] {
  const firstDayOfWeek = getFirstDayOfWeek();
  const monthStart = startOfMonth(date);
  const monthEnd = endOfMonth(date);

  // Get the start of the week containing the first day of the month
  const gridStart = startOfWeek(monthStart, { weekStartsOn: firstDayOfWeek });

  // Get the end of the week containing the last day of the month
  const gridEnd = endOfWeek(monthEnd, { weekStartsOn: firstDayOfWeek });

  // Generate all days in the grid range
  const days = eachDayOfInterval({ start: gridStart, end: gridEnd });

  // Ensure we have exactly 6 weeks (42 days) for consistent display
  while (days.length < 42) {
    const nextDay = new Date(days[days.length - 1]);
    nextDay.setDate(nextDay.getDate() + 1);
    days.push(nextDay);
  }

  return days.map(day => ({
    date: day,
    isToday: isToday(day),
    isCurrentMonth: isSameMonth(day, date),
    events: [],
    taskCount: 0,
    overdueCount: 0,
    completedCount: 0,
  }));
}

/**
 * Generate a week grid for a given date
 */
export function generateWeekGrid(date: Date): CalendarDay[] {
  const firstDayOfWeek = getFirstDayOfWeek();
  const weekStart = startOfWeek(date, { weekStartsOn: firstDayOfWeek });
  const weekEnd = endOfWeek(date, { weekStartsOn: firstDayOfWeek });

  const days = eachDayOfInterval({ start: weekStart, end: weekEnd });

  return days.map(day => ({
    date: day,
    isToday: isToday(day),
    isCurrentMonth: true,
    events: [],
    taskCount: 0,
    overdueCount: 0,
    completedCount: 0,
  }));
}

/**
 * Populate calendar days with events
 */
export function populateDayEvents(days: CalendarDay[], events: CalendarEvent[]): CalendarDay[] {
  // Create a map of date strings to events for efficient lookup
  const eventsByDate = new Map<string, CalendarEvent[]>();

  events.forEach(event => {
    const dateKey = format(startOfDay(event.date), "yyyy-MM-dd");
    if (!eventsByDate.has(dateKey)) {
      eventsByDate.set(dateKey, []);
    }
    eventsByDate.get(dateKey)!.push(event);
  });

  // Assign events to days and calculate counts
  return days.map(day => {
    const dateKey = format(startOfDay(day.date), "yyyy-MM-dd");
    const dayEvents = eventsByDate.get(dateKey) || [];

    // Count by event type
    const overdueCount = dayEvents.filter(e =>
      !e.isComplete && e.date < new Date() && e.eventType === "due"
    ).length;
    const completedCount = dayEvents.filter(e => e.isComplete).length;

    // Remove duplicate events (same task on same date) - keep only one per task
    const uniqueEvents = dayEvents.filter((event, index, self) =>
      index === self.findIndex(e => e.taskId === event.taskId)
    );

    return {
      ...day,
      events: uniqueEvents,
      taskCount: uniqueEvents.length,
      overdueCount,
      completedCount,
    };
  });
}

/**
 * Navigate to the next month
 */
export function nextMonth(date: Date): Date {
  return addMonths(date, 1);
}

/**
 * Navigate to the previous month
 */
export function prevMonth(date: Date): Date {
  return subMonths(date, 1);
}

/**
 * Navigate to the next week
 */
export function nextWeek(date: Date): Date {
  return addWeeks(date, 1);
}

/**
 * Navigate to the previous week
 */
export function prevWeek(date: Date): Date {
  return subWeeks(date, 1);
}

/**
 * Go to today
 */
export function goToToday(): Date {
  return new Date();
}

/**
 * Format month header (e.g., "January 2026")
 */
export function formatMonthHeader(date: Date): string {
  return format(date, "MMMM yyyy");
}

/**
 * Format week header (e.g., "Week of Jan 5, 2026")
 */
export function formatWeekHeader(date: Date): string {
  const firstDayOfWeek = getFirstDayOfWeek();
  const weekStart = startOfWeek(date, { weekStartsOn: firstDayOfWeek });
  return `Week of ${format(weekStart, "MMM d, yyyy")}`;
}

/**
 * Get week day names (e.g., ["Sun", "Mon", "Tue", ...])
 */
export function getWeekDayNames(): string[] {
  const firstDayOfWeek = getFirstDayOfWeek();
  const names: string[] = [];

  for (let i = 0; i < 7; i++) {
    const date = new Date();
    // Set to a known Sunday (Jan 1, 2023 was a Sunday)
    date.setFullYear(2023, 0, 1 + i + (firstDayOfWeek === 0 ? 0 : firstDayOfWeek === 1 ? 0 : -6));
    names.push(format(date, "EEE"));
  }

  return names;
}

/**
 * Group events by task for display purposes
 * Returns unique tasks that have events on a given day
 */
export function groupEventsByTask(events: CalendarEvent[]): Map<number, CalendarEvent> {
  const taskMap = new Map<number, CalendarEvent>();

  events.forEach(event => {
    // Prefer scheduled events over due events over completed events
    const existing = taskMap.get(event.taskId);
    if (!existing || event.eventType === "scheduled" ||
      (existing.eventType !== "scheduled" && event.eventType === "due")) {
      taskMap.set(event.taskId, event);
    }
  });

  return taskMap;
}

/**
 * Merge task events and external events into a unified calendar event list
 * External events are converted to CalendarEvent format with source="external"
 */
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
    description: ee.description,
    location: ee.location,
    externalUrl: ee.external_url,
    color: ee.color || "#6366f1", // Default indigo
  }));

  return [...taskEvents, ...converted];
}

/**
 * Check if a date has any overdue tasks
 */
export function hasOverdueTasks(day: CalendarDay): boolean {
  return day.overdueCount > 0;
}

/**
 * Check if a date has any completed tasks
 */
export function hasCompletedTasks(day: CalendarDay): boolean {
  return day.completedCount > 0;
}

/**
 * Get max number of events to show in a day cell
 */
export const MAX_VISIBLE_EVENTS = 3;

/**
 * Truncate events to maximum visible count
 */
export function getVisibleEvents(day: CalendarDay): CalendarEvent[] {
  return day.events.slice(0, MAX_VISIBLE_EVENTS);
}

/**
 * Get count of hidden events
 */
export function getHiddenEventCount(day: CalendarDay): number {
  return Math.max(0, day.events.length - MAX_VISIBLE_EVENTS);
}
