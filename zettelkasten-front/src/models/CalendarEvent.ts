import { Task } from "./Task";

/**
 * Represents a calendar event derived from a task
 */
export interface CalendarEvent {
  id: number;
  taskId: number;
  title: string;
  date: Date;
  allDay: boolean;
  priority: string | null;
  status: string;
  isComplete: boolean;
  task: Task;
  eventType: "scheduled" | "due" | "completed";
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
 * Get calendar event color based on priority and status
 */
export function getEventColor(event: CalendarEvent): string {
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
