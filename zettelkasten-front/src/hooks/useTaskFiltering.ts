import { useMemo } from "react";
import { Task } from "../models/Task";
import { filterTasks, filterTasksByDateView } from "../utils/tasks";

type SortField = "updated_at" | "title" | "priority" | "status" | "id";
type SortDirection = "asc" | "desc";

interface UseTaskFilteringParams {
  tasks: Task[];
  dateView: string;
  showCompleted: boolean;
  filterString: string;
  sortField: SortField;
  sortDirection: SortDirection;
  viewMode: "list" | "matrix" | "kanban";
  currentPage: number;
  itemsPerPage: number;
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
}: UseTaskFilteringParams): UseTaskFilteringReturn {
  // Filter, sort, and prepare tasks to display
  const tasksToDisplay = useMemo(() => {
    // First, filter by date view
    let filtered = tasks.filter(task =>
      filterTasksByDateView(task, dateView, showCompleted)
    );

    // Then, filter by search string
    let searched = filterTasks(filtered, filterString);

    // Matrix and Kanban views don't need sorting, just return filtered tasks
    if (viewMode === "matrix" || viewMode === "kanban") {
      return searched;
    }

    // Sort tasks for list view
    searched.sort((a, b) => {
      let comparison = 0;
      switch (sortField) {
        case "updated_at":
          comparison =
            new Date(a.updated_at).getTime() -
            new Date(b.updated_at).getTime();
          break;
        case "title":
          const titleA = a.title || "";
          const titleB = b.title || "";
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
        case "id":
          comparison = a.id - b.id;
          break;
        default:
          comparison = a.id - b.id; // Fallback to id
      }
      return sortDirection === "asc" ? comparison : -comparison;
    });

    return searched;
  }, [tasks, dateView, showCompleted, filterString, sortField, sortDirection, viewMode]);

  // Calculate total tasks for the current date view (before search filtering)
  const totalTasksForDateView = useMemo(() => {
    return tasks.filter(task =>
      filterTasksByDateView(task, dateView, showCompleted)
    ).length;
  }, [tasks, dateView, showCompleted]);

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
