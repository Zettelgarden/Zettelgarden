/**
 * Example: Migrated TaskList component using React Query
 *
 * This file demonstrates how to migrate a component from using TaskContext
 * to using React Query hooks.
 *
 * BEFORE (using TaskContext):
 * ```tsx
 * import { useTaskContext } from '../contexts/TaskContext';
 *
 * function TaskList() {
 *   const { tasks, existingTags, showCompleted, setShowCompleted } = useTaskContext();
 *   // ... component logic
 * }
 * ```
 *
 * AFTER (using React Query):
 * ```tsx
 * import { useTasks, useUpdateTask, useDeleteTask } from '../hooks/queries/useTasks';
 *
 * function TaskList() {
 *   const { data: tasks, tags, isLoading, error } = useTasks({ showCompleted: false });
 *   const updateTask = useUpdateTask();
 *   const deleteTask = useDeleteTask();
 *   // ... component logic
 * }
 * ```

 * BENEFITS:
 * - No need for TaskContext wrapper
 * - Automatic loading states
 * - Automatic error handling
 * - Optimistic updates built-in
 * - Automatic cache invalidation
 * - Better TypeScript inference
 */

import React, { useState } from 'react';
import {
  useTasks,
  useUpdateTask,
  useDeleteTask,
} from '../../hooks/queries/useTasks';
import { Task } from '../../models/Task';

interface TaskListProps {
  /**
   * Whether to show completed tasks
   * @default false
   */
  showCompleted?: boolean;

  /**
   * Optional status filter
   */
  status?: string | null;
}

/**
 * TaskList component using React Query
 *
 * This component demonstrates:
 * 1. Data fetching with useTasks
 * 2. Loading and error states
 * 3. Optimistic updates with useUpdateTask
 * 4. Delete operations with useDeleteTask
 * 5. Derived state (tags from tasks)
 */
export function TaskList({ showCompleted = false, status }: TaskListProps) {
  // Fetch tasks with React Query
  const {
    data: tasks = [],
    tags,
    isLoading,
    error,
    refetch,
  } = useTasks({ showCompleted, status });

  // Mutations with optimistic updates
  const updateTask = useUpdateTask();
  const deleteTask = useDeleteTask();

  // Local state for UI
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);

  // Handlers
  const handleToggleComplete = (task: Task) => {
    const updatedTask = { ...task, is_complete: !task.is_complete };
    updateTask.mutate(updatedTask);
  };

  const handleDeleteTask = (taskId: number) => {
    if (confirm('Are you sure you want to delete this task?')) {
      deleteTask.mutate(taskId);
    }
  };

  const handleUpdateTitle = (task: Task, newTitle: string) => {
    const updatedTask = { ...task, title: newTitle };
    updateTask.mutate(updatedTask);
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-gray-500">Loading tasks...</div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded-md">
        <div className="text-red-700 font-medium">Failed to load tasks</div>
        <div className="text-red-600 text-sm mt-1">{error.message}</div>
        <button
          onClick={() => refetch()}
          className="mt-2 px-3 py-1 bg-red-100 hover:bg-red-200 text-red-700 rounded text-sm"
        >
          Retry
        </button>
      </div>
    );
  }

  // Empty state
  if (tasks.length === 0) {
    return (
      <div className="text-center p-8 text-gray-500">
        No tasks found. Create your first task to get started!
      </div>
    );
  }

  // Task list
  return (
    <div className="space-y-2">
      {/* Tags summary (derived from tasks) */}
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-4 pb-4 border-b">
          <span className="text-sm text-gray-500">Tags:</span>
          {tags.map((tag) => (
            <span
              key={tag}
              className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-sm"
            >
              {tag}
            </span>
          ))}
        </div>
      )}

      {/* Task items */}
      {tasks.map((task) => (
        <TaskItem
          key={task.id}
          task={task}
          isSelected={selectedTaskId === task.id}
          onSelect={() => setSelectedTaskId(task.id)}
          onToggleComplete={() => handleToggleComplete(task)}
          onDelete={() => handleDeleteTask(task.id)}
          onUpdateTitle={(newTitle) => handleUpdateTitle(task, newTitle)}
          isUpdating={updateTask.isPending}
          isDeleting={deleteTask.isPending}
        />
      ))}

      {/* Mutation status indicators */}
      {(updateTask.isPending || deleteTask.isPending) && (
        <div className="text-sm text-gray-500 text-center py-2">
          Saving changes...
        </div>
      )}
    </div>
  );
}

interface TaskItemProps {
  task: Task;
  isSelected: boolean;
  onSelect: () => void;
  onToggleComplete: () => void;
  onDelete: () => void;
  onUpdateTitle: (title: string) => void;
  isUpdating: boolean;
  isDeleting: boolean;
}

/**
 * Individual task item component
 *
 * Demonstrates how mutation state can be used to show loading indicators
 */
function TaskItem({
  task,
  isSelected,
  onSelect,
  onToggleComplete,
  onDelete,
  onUpdateTitle,
  isUpdating,
  isDeleting,
}: TaskItemProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(task.title);

  const handleSaveEdit = () => {
    if (editValue.trim() && editValue !== task.title) {
      onUpdateTitle(editValue);
    }
    setIsEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSaveEdit();
    } else if (e.key === 'Escape') {
      setEditValue(task.title);
      setIsEditing(false);
    }
  };

  return (
    <div
      className={`
        flex items-center gap-3 p-3 rounded-lg border transition-colors
        ${
          isSelected
            ? 'bg-blue-50 border-blue-300'
            : 'bg-white border-gray-200 hover:bg-gray-50'
        }
        ${task.is_complete ? 'opacity-60' : ''}
        ${isUpdating || isDeleting ? 'animate-pulse' : ''}
      `}
    >
      {/* Checkbox */}
      <input
        type="checkbox"
        checked={task.is_complete}
        onChange={onToggleComplete}
        disabled={isUpdating || isDeleting}
        className="w-4 h-4 text-blue-600 rounded"
      />

      {/* Task title (editable) */}
      <div className="flex-1 min-w-0">
        {isEditing ? (
          <input
            type="text"
            value={editValue}
            onChange={(e) => setEditValue(e.target.value)}
            onBlur={handleSaveEdit}
            onKeyDown={handleKeyDown}
            autoFocus
            className="w-full px-2 py-1 border border-blue-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
            disabled={isUpdating}
          />
        ) : (
          <span
            onClick={() => onSelect()}
            onDoubleClick={() => setIsEditing(true)}
            className={`
              cursor-pointer block truncate
              ${
                task.is_complete
                  ? 'line-through text-gray-500'
                  : 'text-gray-900'
              }
            `}
          >
            {task.title}
          </span>
        )}

        {/* Task metadata */}
        <div className="flex items-center gap-2 mt-1 text-xs text-gray-500">
          {task.scheduled_date && (
            <span>
              Scheduled: {new Date(task.scheduled_date).toLocaleDateString()}
            </span>
          )}
          {task.priority && (
            <span className="px-1.5 py-0.5 bg-gray-100 rounded">
              {task.priority}
            </span>
          )}
          {task.status && (
            <span className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded">
              {task.status}
            </span>
          )}
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2">
        <button
          onClick={() => setIsEditing(!isEditing)}
          disabled={isUpdating || isDeleting}
          className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded"
          title="Edit"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
            />
          </svg>
        </button>
        <button
          onClick={onDelete}
          disabled={isDeleting}
          className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
          title="Delete"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
        </button>
      </div>
    </div>
  );
}

/**
 * USAGE EXAMPLE:
 *
 * ```tsx
 * import { TaskList } from './components/tasks/TaskListWithRQ';
 *
 * function TaskPage() {
 *   return (
 *     <div className="p-6">
 *       <h1 className="text-2xl font-bold mb-4">My Tasks</h1>
 *       <TaskList showCompleted={false} />
 *     </div>
 *   );
 * }
 * ```
 *
 * MIGRATION NOTES:
 *
 * 1. Remove TaskContext dependency:
 *    - No need to wrap component in TaskProvider
 *    - Direct hook usage instead of context
 *
 * 2. State management:
 *    - Remove manual state management for tasks
 *    - Remove refreshTasks / setRefreshTasks pattern
 *    - Let React Query handle caching and refetching
 *
 * 3. Optimistic updates:
 *    - Built into useUpdateTask mutation
 *    - Automatic rollback on error
 *
 * 4. Error handling:
 *    - Built-in error state from useQuery
 *    - Retry functionality included
 *
 * 5. Testing:
 *    - Easier to test with mocked query hooks
 *    - No need to wrap tests in context providers
 */
