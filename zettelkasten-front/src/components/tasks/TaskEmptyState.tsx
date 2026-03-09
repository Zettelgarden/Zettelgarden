import React from "react";
import { Button } from "../Button";

type EmptyStateType = "no-tasks" | "no-matches" | "all-completed";

interface TaskEmptyStateProps {
  type: EmptyStateType;
  onAddTask?: () => void;
  onClearFilters?: () => void;
  onShowCompleted?: () => void;
}

export function TaskEmptyState({
  type,
  onAddTask,
  onClearFilters,
  onShowCompleted,
}: TaskEmptyStateProps) {
  if (type === "no-tasks") {
    return (
      <div className="flex flex-col items-center justify-center py-16 px-4">
        <div className="text-6xl mb-4">📝</div>
        <h3 className="text-xl font-semibold text-slate-700 mb-2">
          No tasks yet
        </h3>
        <p className="text-slate-500 mb-6 text-center max-w-sm">
          Get started by creating your first task. Break down your work into manageable pieces.
        </p>
        <Button
          onClick={onAddTask}
          className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg font-medium flex items-center gap-2"
        >
          <span className="text-lg">+</span>
          Add your first task
        </Button>
      </div>
    );
  }

  if (type === "no-matches") {
    return (
      <div className="flex flex-col items-center justify-center py-16 px-4">
        <div className="text-6xl mb-4">🔍</div>
        <h3 className="text-xl font-semibold text-slate-700 mb-2">
          No matching tasks
        </h3>
        <p className="text-slate-500 mb-6 text-center max-w-sm">
          No tasks match your current filter. Try adjusting your search terms or clearing the filter.
        </p>
        <Button
          onClick={onClearFilters}
          className="bg-slate-600 hover:bg-slate-700 text-white px-6 py-2 rounded-lg font-medium"
        >
          Clear filters
        </Button>
      </div>
    );
  }

  if (type === "all-completed") {
    return (
      <div className="flex flex-col items-center justify-center py-16 px-4">
        <div className="text-6xl mb-4">🎉</div>
        <h3 className="text-xl font-semibold text-slate-700 mb-2">
          All done for now!
        </h3>
        <p className="text-slate-500 mb-6 text-center max-w-sm">
          You've completed all your tasks. Great work! Take a break or review your completed tasks.
        </p>
        <div className="flex gap-3">
          <Button
            onClick={onAddTask}
            className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium flex items-center gap-2"
          >
            <span className="text-lg">+</span>
            Add task
          </Button>
          <Button
            onClick={onShowCompleted}
            className="bg-slate-200 hover:bg-slate-300 text-slate-700 px-4 py-2 rounded-lg font-medium"
          >
            Show completed
          </Button>
        </div>
      </div>
    );
  }

  return null;
}

/**
 * Determines the appropriate empty state type based on task data
 */
export function getEmptyStateType(params: {
  totalTasks: number;
  filteredTasks: number;
  hasActiveFilter: boolean;
  showCompleted: boolean;
}): EmptyStateType | null {
  const { totalTasks, filteredTasks, hasActiveFilter, showCompleted } = params;

  // If there are tasks to display, no empty state needed
  if (filteredTasks > 0) {
    return null;
  }

  // No tasks at all in the system
  if (totalTasks === 0) {
    return "no-tasks";
  }

  // Tasks exist but filtered out by active filter
  if (hasActiveFilter) {
    return "no-matches";
  }

  // Tasks exist but none visible (likely all completed and showCompleted is false)
  if (!showCompleted) {
    return "all-completed";
  }

  // Default to no-matches if we have tasks but none displayed
  return "no-matches";
}
