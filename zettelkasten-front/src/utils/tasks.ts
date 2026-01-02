import { Task } from "../models/Task";
import { compareDates, getToday, getTomorrow, isTodayOrPast } from "./dates";

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
  const searchTerms = filterString.split(' ').map(term => term.trim()).filter(term => term !== '');

  return input.filter(task => {
    return searchTerms.every(term => {
      const isNegation = term.startsWith('!');
      const termWithoutNegation = isNegation ? term.substring(1) : term;
      const lowerTerm = termWithoutNegation.toLowerCase();

      // Has filtering (e.g., has:reminder)
      if (lowerTerm.startsWith('has:')) {
        const hasValue = lowerTerm.substring('has:'.length);
        if (hasValue === 'reminder') {
          const hasReminder = task.reminder_time !== null;
          return isNegation ? !hasReminder : hasReminder;
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
  showCompleted: boolean
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
    if (!task.is_complete && isTodayOrPast(task.scheduled_date)) {
      return true;
    } else if (
      showCompleted &&
      task.completed_at && // Check if task has a completion date
      compareDates(task.completed_at, getToday())
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
      compareDates(task.scheduled_date, getTomorrow())
    ) {
      return true;
    } else if (
      showCompleted &&
      task.completed_at && // Check if task has a completion date
      compareDates(task.completed_at, getTomorrow())
    ) {
      return true;
    } else {
      return false;
    }
  }

  // Fallback for other dateView values
  return !task.is_complete;
}
