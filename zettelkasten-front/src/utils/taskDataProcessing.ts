import { Task } from "../models/Task";

/**
 * Converts API response task to internal Task format
 * Handles date conversion and property normalization
 * This is the main entry point for processing raw task data from the API
 */
export function processTaskFromAPI(rawTask: any): Task {
  return {
    ...rawTask,
    scheduled_date: convertDateField(rawTask.scheduled_date),
    due_date: convertDateField(rawTask.due_date),
    created_at: new Date(rawTask.created_at), // Always exists
    updated_at: new Date(rawTask.updated_at), // Always exists
    completed_at: convertDateField(rawTask.completed_at),
    reminder_time: convertDateField(rawTask.reminder_time),
    description: normalizeDescription(rawTask.description),
    tags: normalizeTags(rawTask.tags),
    // Ensure consistent default values for optional fields
    card: rawTask.card === undefined ? null : rawTask.card,
    blocked_by: rawTask.blocked_by || [],
    blocks: rawTask.blocks || []
  };
}

/**
 * Converts date fields in a task that might be null/undefined
 * Useful for processing partial task updates or forms
 */
export function convertTaskDates(rawTask: any): Partial<Task> {
  return {
    ...rawTask,
    scheduled_date: convertDateField(rawTask.scheduled_date),
    due_date: convertDateField(rawTask.due_date),
    created_at: rawTask.created_at ? new Date(rawTask.created_at) : undefined,
    updated_at: rawTask.updated_at ? new Date(rawTask.updated_at) : undefined,
    completed_at: convertDateField(rawTask.completed_at),
    reminder_time: convertDateField(rawTask.reminder_time)
  };
}

/**
 * Ensures consistent property structure and default values
 * Useful for normalizing task properties across different sources
 */
export function normalizeTaskProperties(rawTask: any): Partial<Task> {
  return {
    ...rawTask,
    description: normalizeDescription(rawTask.description),
    tags: normalizeTags(rawTask.tags)
  };
}

/**
 * Converts date fields that might be null, undefined, or string formats
 * Returns null for falsy values, Date object for valid date strings
 */
function convertDateField(dateValue: string | Date | null | undefined): Date | null {
  if (!dateValue) return null;
  return new Date(dateValue);
}

/**
 * Normalizes description field to ensure consistent null handling
 */
function normalizeDescription(desc: string | null | undefined): string | null {
  return desc || null;
}

/**
 * Normalizes tags array to ensure empty array for falsy values
 */
function normalizeTags(tags: any[] | null | undefined): any[] {
  return tags || [];
}