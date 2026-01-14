import { Task, TaskAuditEvent } from "../models/Task";
import { format } from "date-fns";
import { compareDates, compareDatesInTimezone, getToday, getTomorrow, isTodayOrPast } from "./dates";

export interface TaskFilterParams {
  searchTerms: string[];
  dateView: "all" | "today" | "tomorrow";
  showCompleted: boolean;
  specificDate: Date | null;
}

export function parseTaskQuery(query: string): TaskFilterParams {
  // Default values
  const params: TaskFilterParams = {
    searchTerms: [],
    dateView: "all",
    showCompleted: false,
    specificDate: null,
  };

  // Split query into tokens
  const tokens = query.split(' ').map(t => t.trim()).filter(t => t !== '');

  // Extract date-related tokens
  const dateTokenIndex = tokens.findIndex(t => t.startsWith('date:'));
  if (dateTokenIndex !== -1) {
    const dateToken = tokens[dateTokenIndex];
    const dateString = dateToken.substring('date:'.length);

    // Check for date:today or date:tomorrow
    if (dateString === 'today') {
      params.dateView = 'today';
      tokens.splice(dateTokenIndex, 1);
    } else if (dateString === 'tomorrow') {
      params.dateView = 'tomorrow';
      tokens.splice(dateTokenIndex, 1);
    } else {
      // Try to parse as ISO date format (YYYY-MM-DD)
      try {
        const parsedDate = new Date(dateString);
        if (!isNaN(parsedDate.getTime())) {
          params.specificDate = parsedDate;
        }
      } catch (error) {
        console.error('Invalid date format:', dateString);
      }
      // Remove from tokens
      tokens.splice(dateTokenIndex, 1);
    }
  }

  // Check for show:completed flag
  const completedIndex = tokens.findIndex(t => t === 'show:completed');
  if (completedIndex !== -1) {
    params.showCompleted = true;
    tokens.splice(completedIndex, 1);
  }

  // Remaining tokens are search terms
  params.searchTerms = tokens;

  return params;
}

/**
 * Removes specific keywords from a query string
 */
export function removeKeywordsFromQuery(query: string, keywords: string[]): string {
  const tokens = query.split(' ').map(t => t.trim()).filter(t => t !== '');
  const filtered = tokens.filter(token => {
    // Check if token matches any keyword exactly or starts with keyword:
    return !keywords.some(keyword =>
      token === keyword || token.startsWith(keyword + ':')
    );
  });
  return filtered.join(' ');
}

/**
 * Adds a keyword to a query string (removes it first if it exists)
 */
export function addKeywordToQuery(query: string, keyword: string): string {
  // First remove the keyword if it exists
  const cleaned = removeKeywordsFromQuery(query, [keyword.split(':')[0]]);
  // Add the new keyword
  return cleaned ? `${cleaned} ${keyword}`.trim() : keyword;
}

/**
 * Updates the query string to reflect date view selection
 */
export function updateQueryDateView(query: string, dateView: string): string {
  // Remove existing date view keywords and date: prefix
  let updated = removeKeywordsFromQuery(query, ['date']);

  // Add new date view if not "all"
  if (dateView !== 'all') {
    updated = updated ? `${updated} date:${dateView}`.trim() : `date:${dateView}`;
  }

  return updated;
}

/**
 * Updates the query string to reflect completed status
 */
export function updateQueryShowCompleted(query: string, showCompleted: boolean): string {
  let updated = removeKeywordsFromQuery(query, ['show']);

  if (showCompleted) {
    updated = updated ? `${updated} show:completed`.trim() : 'show:completed';
  }

  return updated;
}

/**
 * Removes special filter keywords from a query, keeping only tags and plain text
 * Used when initializing a new task with the current filter string
 */
export function stripSpecialFilters(query: string): string {
  const tokens = query.split(' ').map(t => t.trim()).filter(t => t !== '');

  const filteredTokens = tokens.filter(token => {
    // Keep tags (anything starting with #)
    if (token.startsWith('#')) {
      return true;
    }

    // Remove special filters
    if (token.startsWith('priority:')) return false;
    if (token.startsWith('status:')) return false;
    if (token.startsWith('date:')) return false;
    if (token.startsWith('has:')) return false;
    if (token.startsWith('show:')) return false;
    if (token.startsWith('!')) return false; // Remove negations

    // Keep plain text
    return true;
  });

  return filteredTokens.join(' ');
}

export function removeTagsFromTitle(title: string): string {
  const tagPattern = /#[\w-]+/g;
  const cleanedTitle = title.replace(tagPattern, "");
  return cleanedTitle.trim();
}

export function parseTags(title: string): string[] {
  const tagPattern = /(?<=\s|^)(#[\w-]+(?:[.,!?])?)+(?=\s|$)/g;
  const matches = title.match(tagPattern);

  return matches ? Array.from(matches) : [];
}

export function filterTasks(input: Task[], filterString: string): Task[] {
  // Parse the query to extract special keywords and get only the search terms
  const parsed = parseTaskQuery(filterString);
  const searchTerms = parsed.searchTerms;

  return input.filter(task => {
    return searchTerms.every(term => {
      const isNegation = term.startsWith('!');
      const termWithoutNegation = isNegation ? term.substring(1) : term;
      const lowerTerm = termWithoutNegation.toLowerCase();

      // Has filtering (e.g., has:reminder, has:description)
      if (lowerTerm.startsWith('has:')) {
        const hasValue = lowerTerm.substring('has:'.length);
        if (hasValue === 'reminder') {
          const hasReminder = task.reminder_time !== null;
          return isNegation ? !hasReminder : hasReminder;
        }
        if (hasValue === 'description') {
          const hasDescription = task.description !== null && task.description.trim() !== '';
          return isNegation ? !hasDescription : hasDescription;
        }
      }

      // Priority filtering
      if (lowerTerm.startsWith('priority:')) {
        const priorityValue = lowerTerm.substring('priority:'.length);
        if (task.priority === null) { // Task has no priority
          return isNegation; // If !priority:X and task has no priority, it's a match. If priority:X, it's not.
        }
        const taskPriorityLower = task.priority.toLowerCase();
        const matchesPriority = taskPriorityLower === priorityValue;
        return isNegation ? !matchesPriority : matchesPriority;
      }

      // Status filtering (status:todo, status:in_progress, status:blocked, status:done)
      if (lowerTerm.startsWith('status:')) {
        const statusValue = lowerTerm.substring('status:'.length);
        const taskStatusLower = task.status.toLowerCase();
        const matchesStatus = taskStatusLower === statusValue;
        return isNegation ? !matchesStatus : matchesStatus;
      }

      // Tag filtering
      if (lowerTerm.startsWith('#')) {
        const tagName = lowerTerm.substring(1);
        const hasTag = task.tags.some(tag => tag.name.toLowerCase() === tagName);
        return isNegation ? !hasTag : hasTag;
      }

      // Text filtering (title)
      const hasText = task.title.toLowerCase().includes(lowerTerm);
      return isNegation ? !hasText : hasText;
    });
  });
}

export function filterTasksByDateView(
  task: Task,
  dateView: string,
  showCompleted: boolean,
  timezone: string = "UTC"
): boolean {
  // Handle "all" view
  if (dateView === "all") {
    // Only show completed tasks if the "Closed" tab is active
    if (!showCompleted && task.is_complete) {
      return false;
    }
    return true;
  }

  // Handle "today" view
  if (dateView === "today") {
    if (!task.is_complete && isTodayOrPast(task.scheduled_date, timezone)) {
      return true;
    } else if (
      showCompleted &&
      task.completed_at && // Check if task has a completion date
      compareDatesInTimezone(task.completed_at, getToday(timezone), timezone)
    ) {
      return true;
    } else {
      return false;
    }
  }

  // Handle "tomorrow" view
  if (dateView === "tomorrow") {
    if (
      !task.is_complete &&
      task.scheduled_date && // Ensure scheduled_date is not null
      compareDatesInTimezone(task.scheduled_date, getTomorrow(timezone), timezone)
    ) {
      return true;
    } else if (
      showCompleted &&
      task.completed_at && // Check if task has a completion date
      compareDatesInTimezone(task.completed_at, getTomorrow(timezone), timezone)
    ) {
      return true;
    } else {
      return false;
    }
  }

  // Fallback for other dateView values
  return !task.is_complete;
}

const AUDIT_ACTIONS = {
  CREATE: "create",
  DELETE: "delete",
  UPDATE: "update",
} as const;

export function formatAuditEvent(event: TaskAuditEvent): string {
  if (event.action === AUDIT_ACTIONS.CREATE) {
    return "Task created";
  }

  if (event.action === AUDIT_ACTIONS.DELETE) {
    return "Task deleted";
  }

  if (event.action === AUDIT_ACTIONS.UPDATE && event.details.change_type === "update") {
    const changes: string[] = [];
    const changeDetails = event.details.changes;

    if (changeDetails.Title) {
      changes.push(`Changed title from "${changeDetails.Title.from}" to "${changeDetails.Title.to}"`);
    }

    if (changeDetails.IsComplete) {
      changes.push(changeDetails.IsComplete.to ? "Marked as complete" : "Marked as incomplete");
    }

    if (changeDetails.ScheduledDate) {
      const newDate = changeDetails.ScheduledDate.to
        ? format(new Date(changeDetails.ScheduledDate.to), "MMM d, yyyy")
        : "none";
      changes.push(`Changed scheduled date to ${newDate}`);
    }

    if (changeDetails.CardPK) {
      if (changeDetails.CardPK.from === 0 && changeDetails.CardPK.to > 0) {
        changes.push(`Linked to card [${changeDetails.CardPK.to}]`);
      } else if (changeDetails.CardPK.from > 0 && changeDetails.CardPK.to === 0) {
        changes.push(`Unlinked from card [${changeDetails.CardPK.from}]`);
      } else {
        changes.push(`Changed linked card from [${changeDetails.CardPK.from}] to [${changeDetails.CardPK.to}]`);
      }
    }

    if (changeDetails.Priority) {
      const fromPriority = changeDetails.Priority.from ? `Priority ${changeDetails.Priority.from}` : "No Priority";
      const toPriority = changeDetails.Priority.to ? `Priority ${changeDetails.Priority.to}` : "No Priority";
      changes.push(`Changed priority from ${fromPriority} to ${toPriority}`);
    }

    if (changeDetails.Description) {
      if (!changeDetails.Description.from && changeDetails.Description.to) {
        changes.push(`Added description`);
      } else if (changeDetails.Description.from && !changeDetails.Description.to) {
        changes.push(`Removed description`);
      } else {
        changes.push(`Updated description`);
      }
    }

    if (changeDetails.ReminderTime) {
      if (!changeDetails.ReminderTime.from && changeDetails.ReminderTime.to) {
        const newReminder = format(new Date(changeDetails.ReminderTime.to), "MMM d, yyyy h:mm a");
        changes.push(`Set reminder for ${newReminder}`);
      } else if (changeDetails.ReminderTime.from && !changeDetails.ReminderTime.to) {
        changes.push(`Removed reminder`);
      } else if (changeDetails.ReminderTime.from && changeDetails.ReminderTime.to) {
        const newReminder = format(new Date(changeDetails.ReminderTime.to), "MMM d, yyyy h:mm a");
        changes.push(`Changed reminder to ${newReminder}`);
      }
    }

    if (changes.length === 0) {
      return "Task updated";
    }

    return changes.join("; ");
  }

  return "Unknown change";
}
