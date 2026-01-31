import { Task } from "./Task";

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
  const today = new Date();
  const d = new Date(date);
  const t = new Date(today);

  // Reset times to midnight for comparison
  d.setHours(0, 0, 0, 0);
  t.setHours(0, 0, 0, 0);

  return d.getTime() === t.getTime();
}

/**
 * Check if a date is in the past (before today) in the given timezone
 */
export function isPast(date: Date, timezone: string = "UTC"): boolean {
  const today = new Date();
  const d = new Date(date);

  // Reset times to midnight for comparison
  d.setHours(0, 0, 0, 0);
  today.setHours(0, 0, 0, 0);

  return d.getTime() < today.getTime();
}
