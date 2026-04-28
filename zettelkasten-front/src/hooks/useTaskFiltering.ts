import { useMemo } from "react";
import { Task } from "../models/Task";
import { filterTasks, filterTasksByDateView, removeTagsFromTitle } from "../utils/tasks";
import { toMidnightInTimezone } from "../utils/dates";

type SortField = "updated_at" | "title" | "priority" | "status" | "id" | "scheduled_date" | "manual";
type SortDirection = "asc" | "desc";
type ViewMode = "list" | "matrix" | "kanban";

interface UseTaskFilteringParams {
  tasks: Task[];
  dateView: string;
  showCompleted: boolean;
  filterString: string;
  sortField: SortField;
  sortDirection: SortDirection;
  viewMode: ViewMode;
  currentPage: number;
  itemsPerPage: number;
  timezone?: string;
}

interface UseTaskFilteringReturn {
  tasksToDisplay: Task[];
  paginatedTasks: Task[];
  totalPages: number;
  totalTasksForDateView: number;
}

export function useTaskFiltering({
  tasks,
  dateView,
  showCompleted,
  filterString,
  sortField,
  sortDirection,
  viewMode,
  currentPage,
  itemsPerPage,
  timezone = "UTC",
}: UseTaskFilteringParams): UseTaskFilteringReturn {
  // Filter, sort, and prepare tasks to display
  const tasksToDisplay = useMemo(() => {
    // First, filter by date view
    let filtered = tasks.filter(task =>
      filterTasksByDateView(task, dateView, showCompleted, timezone)
    );

    // Then, filter by search string
    let searched = filterTasks(filtered, filterString);

    // Sort tasks for all views
    searched.sort((a, b) => {
      let comparison = 0;
      switch (sortField) {
        case "updated_at":
          comparison =
            new Date(a.updated_at).getTime() -
            new Date(b.updated_at).getTime();
          break;
        case "title":
          // Strip tags before comparing to match display order
          const titleA = removeTagsFromTitle(a.title || "");
          const titleB = removeTagsFromTitle(b.title || "");
          comparison = titleA.toLowerCase().localeCompare(titleB.toLowerCase());
          break;
        case "priority":
          const prioA = a.priority;
          const prioB = b.priority;
          if (prioA === null && prioB === null) comparison = 0;
          else if (prioA === null) comparison = 1; // nulls last
          else if (prioB === null) comparison = -1; // nulls last
          else comparison = prioA.localeCompare(prioB);
          break;
        case "status":
          const statusOrder = { 'todo': 0, 'in_progress': 1, 'blocked': 2, 'done': 3 };
          comparison = statusOrder[a.status] - statusOrder[b.status];
          break;
        case "scheduled_date":
          const scheduledA = a.scheduled_date;
          const scheduledB = b.scheduled_date;
          if (scheduledA === null && scheduledB === null) {
            comparison = 0;
          } else if (scheduledA === null) {
            return 1; // Always put nulls last, regardless of sort direction
          } else if (scheduledB === null) {
            return -1; // Always put nulls last, regardless of sort direction
          } else {
            comparison = toMidnightInTimezone(scheduledA, timezone).getTime() - toMidnightInTimezone(scheduledB, timezone).getTime();
          }
          break;
        case "id":
          comparison = a.id - b.id;
          break;
        case "manual":
          // Sort by sort_order, tasks without sort_order go to the bottom
          const orderA = a.sort_order ?? Number.MAX_SAFE_INTEGER;
          const orderB = b.sort_order ?? Number.MAX_SAFE_INTEGER;
          if (orderA !== orderB) {
            comparison = orderA - orderB;
          } else {
            // Tiebreaker: fall back to id
            comparison = a.id - b.id;
          }
          break;
        default:
          comparison = a.id - b.id; // Fallback to id
      }
      // Manual sort is always ascending (smaller sort_order = higher position)
      if (sortField === "manual") return comparison;
      return sortDirection === "asc" ? comparison : -comparison;
    });

    return searched;
  }, [tasks, dateView, showCompleted, filterString, sortField, sortDirection, viewMode, timezone]);

  // Calculate total tasks for the current date view (before search filtering)
  const totalTasksForDateView = useMemo(() => {
    return tasks.filter(task =>
      filterTasksByDateView(task, dateView, showCompleted, timezone)
    ).length;
  }, [tasks, dateView, showCompleted, timezone]);

  // Paginate tasks (only for list view)
  const paginatedTasks = useMemo(() => {
    if (viewMode === "matrix" || viewMode === "kanban") {
      return tasksToDisplay; // Don't paginate matrix or kanban views
    }
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;
    return tasksToDisplay.slice(startIndex, endIndex);
  }, [tasksToDisplay, currentPage, itemsPerPage, viewMode]);

  // Calculate total pages
  const totalPages = useMemo(() => {
    if (viewMode === "matrix" || viewMode === "kanban") return 1;
    return Math.ceil(tasksToDisplay.length / itemsPerPage);
  }, [tasksToDisplay.length, itemsPerPage, viewMode]);

  return {
    tasksToDisplay,
    paginatedTasks,
    totalPages,
    totalTasksForDateView,
  };
}
