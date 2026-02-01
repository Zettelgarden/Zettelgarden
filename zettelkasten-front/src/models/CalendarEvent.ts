import { Task } from "./Task";
import { getToday } from "../utils/dates";
import { toZonedTime } from "date-fns-tz";

/**
 * Source of a calendar event - either from a task or external feed
 */
export type EventSource = "task" | "external";

/**
 * Represents a calendar event derived from a task or imported from external calendar
 */
export interface CalendarEvent {
  id: number;
  taskId?: number;              // Present for task events
  externalEventId?: number;     // Present for external events
  source: EventSource;          // Distinguish the source

  title: string;
  date: Date;
  allDay: boolean;

  // Task-specific fields
  priority?: string | null;
  status?: string;
  isComplete?: boolean;
  task?: Task;
  eventType?: "scheduled" | "due" | "completed";

  // External event-specific fields
  description?: string;
  location?: string;
  externalUrl?: string;
  color?: string;               // From calendar subscription
  cardPK?: number;              // ID of linked card (for external events)
  cardId?: string;              // Card ID string for display
}

/**
 * Represents a day in the calendar with associated events
 */
export interface CalendarDay {
  date: Date;
  isToday: boolean;
  isCurrentMonth: boolean;
  events: CalendarEvent[];
  taskCount: number;
  overdueCount: number;
  completedCount: number;
}

/**
 * Calendar view types
 */
export type CalendarViewType = "month" | "week" | "day";

/**
 * Calendar navigation state
 */
export interface CalendarState {
  currentDate: Date;
  viewMode: CalendarViewType;
  selectedDate: Date | null;
}

/**
 * Convert a task to calendar events
 * A task may generate multiple events (scheduled, due, completed)
 */
export function taskToCalendarEvents(task: Task, timezone: string = "UTC"): CalendarEvent[] {
  const events: CalendarEvent[] = [];

  // Add scheduled date event
  if (task.scheduled_date) {
    events.push({
      id: task.id * 1000,
      taskId: task.id,
      title: task.title,
      date: new Date(task.scheduled_date),
      allDay: true,
      priority: task.priority,
      status: task.status,
      isComplete: task.is_complete,
      task,
      eventType: "scheduled",
      source: "task",
    });
  }

  // Add due date event if different from scheduled
  if (task.due_date && task.scheduled_date) {
    const dueDate = new Date(task.due_date);
    const scheduledDate = new Date(task.scheduled_date);
    if (dueDate.getTime() !== scheduledDate.getTime()) {
      events.push({
        id: task.id * 1000 + 1,
        taskId: task.id,
        title: task.title,
        date: dueDate,
        allDay: true,
        priority: task.priority,
        status: task.status,
        isComplete: task.is_complete,
        task,
        eventType: "due",
        source: "task",
      });
    }
  } else if (task.due_date && !task.scheduled_date) {
    events.push({
      id: task.id * 1000 + 1,
      taskId: task.id,
      title: task.title,
      date: new Date(task.due_date),
      allDay: true,
      priority: task.priority,
      status: task.status,
      isComplete: task.is_complete,
      task,
      eventType: "due",
      source: "task",
    });
  }

  // Add completed date event
  if (task.completed_at) {
    events.push({
      id: task.id * 1000 + 2,
      taskId: task.id,
      title: task.title,
      date: new Date(task.completed_at),
      allDay: true,
      priority: task.priority,
      status: task.status,
      isComplete: true,
      task,
      eventType: "completed",
      source: "task",
    });
  }

  return events;
}

/**
 * Convert multiple tasks to calendar events
 */
export function tasksToCalendarEvents(tasks: Task[], timezone: string = "UTC"): CalendarEvent[] {
  return tasks.flatMap(task => taskToCalendarEvents(task, timezone));
}

/**
 * Get calendar event color based on priority, status, or source
 */
export function getEventColor(event: CalendarEvent): string {
  // External events use their custom color
  if (event.source === "external") {
    const color = event.color || "#6366f1"; // Default indigo
    // Generate Tailwind-style classes with the custom color
    return `border-l-4`;
  }

  // Task events use existing color logic
  if (event.isComplete) {
    return "bg-green-100 text-green-800 border-green-300";
  }

  if (event.eventType === "due") {
    return "bg-red-100 text-red-800 border-red-300";
  }

  switch (event.priority) {
    case "A":
      return "bg-orange-100 text-orange-800 border-orange-300";
    case "B":
      return "bg-yellow-100 text-yellow-800 border-yellow-300";
    case "C":
      return "bg-blue-100 text-blue-800 border-blue-300";
    case "D":
      return "bg-slate-100 text-slate-800 border-slate-300";
    default:
      return "bg-purple-100 text-purple-800 border-purple-300";
  }
}

/**
 * Get icon for an event (null for tasks, calendar icon for external)
 */
export function getEventIcon(event: CalendarEvent): string | null {
  if (event.source === "external") {
    return "📅";
  }
  return null;
}

/**
 * Check if an event is draggable (only task events are draggable)
 */
export function isEventDraggable(event: CalendarEvent): boolean {
  return event.source === "task";
}

/**
 * Check if a date is today in the given timezone
 */
export function isToday(date: Date, timezone: string = "UTC"): boolean {
  const today = getToday(timezone);

  // Convert the input date to the user's timezone to get its calendar date
  const dateInTz = toZonedTime(date, timezone);

  // getToday returns midnight UTC for today's date in the user's timezone
  // We need to check if the input date falls on the same calendar day
  // by comparing their date components in the user's timezone
  return (
    dateInTz.getFullYear() === today.getUTCFullYear() &&
    dateInTz.getMonth() === today.getUTCMonth() &&
    dateInTz.getDate() === today.getUTCDate()
  );
}

/**
 * Check if a date is in the past (before today) in the given timezone
 */
export function isPast(date: Date, timezone: string = "UTC"): boolean {
  const today = getToday(timezone);

  // Convert the input date to the user's timezone to get its calendar date
  const dateInTz = toZonedTime(date, timezone);

  // Compare the calendar dates in the user's timezone
  // GetToday returns midnight UTC for today, so we compare the date components
  const dateYear = dateInTz.getFullYear();
  const dateMonth = dateInTz.getMonth();
  const dateDay = dateInTz.getDate();

  const todayYear = today.getUTCFullYear();
  const todayMonth = today.getUTCMonth();
  const todayDay = today.getUTCDate();

  // Check if the date is before today
  if (dateYear < todayYear) return true;
  if (dateYear > todayYear) return false;
  if (dateMonth < todayMonth) return true;
  if (dateMonth > todayMonth) return false;
  return dateDay < todayDay;
}
