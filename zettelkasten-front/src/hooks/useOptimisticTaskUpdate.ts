import { useTaskContext } from "../contexts/TaskContext";
import { saveExistingTask } from "../api/tasks";
import { Task } from "../models/Task";

interface UseOptimisticTaskUpdateOptions {
  task: Task;
  setTask?: (task: Task) => void;
  saveOnChange: boolean;
  errorMessagePrefix?: string;
}

/**
 * Hook to handle optimistic task updates with automatic rollback on error.
 * Used consistently across Task*Display components.
 */
export function useOptimisticTaskUpdate({
  task,
  setTask,
  saveOnChange,
  errorMessagePrefix = "Failed to update task",
}: UseOptimisticTaskUpdateOptions) {
  const { updateTask: updateTaskInContext } = useTaskContext();

  async function updateTask(editedTask: Task) {
    // Always update local state first if provided
    if (setTask) {
      setTask(editedTask);
    }

    if (saveOnChange) {
      // Optimistic update: update context immediately
      updateTaskInContext(editedTask);

      // Send update to server in background
      try {
        const response = await saveExistingTask(editedTask);
        if ("error" in response) {
          // Rollback on error
          if (setTask) {
            setTask(task);
          }
          updateTaskInContext(task);
          console.error(`${errorMessagePrefix}:`, response.error);
        }
      } catch (error) {
        // Rollback on network error
        if (setTask) {
          setTask(task);
        }
        updateTaskInContext(task);
        console.error(`${errorMessagePrefix}:`, error);
      }
    }
  }

  return { updateTask };
}
