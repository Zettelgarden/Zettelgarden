import { Task, TaskAuditEvent, TasksResponse } from 'src/models/Task';
import { apiClient, getData } from './client';
import { processTaskFromAPI } from '../utils/taskDataProcessing';
import { getDataProvider } from '../data/provider';
import type { FetchTasksParams } from '../data/provider';

/**
 * Fetch tasks with optional filters. Desktop: filtered from the local
 * mirror (instant, offline). Web: paginated GET /tasks.
 */
export async function fetchTasks(
  params: FetchTasksParams = {},
): Promise<Task[]> {
  return getDataProvider().fetchTasks(params);
}

/**
 * Fetch a single task by ID. Desktop: local mirror.
 */
export async function fetchTask(id: string): Promise<Task> {
  return getDataProvider().fetchTask(id);
}

/**
 * Save a new task. Desktop: local mirror + outbox (offline-safe).
 */
export async function saveNewTask(task: Task): Promise<Task> {
  return getDataProvider().saveNewTask(task);
}

/**
 * Save an existing task. Desktop: local mirror + outbox (offline-safe).
 */
export async function saveExistingTask(task: Task): Promise<Task> {
  return getDataProvider().saveExistingTask(task);
}

/**
 * Delete a task. Desktop: queues a local delete, reconciles on reconnect.
 */
export async function deleteTask(id: number): Promise<Task | null> {
  return getDataProvider().deleteTask(id);
}

/**
 * Fetch audit events for a task
 */
export async function fetchTaskAuditEvents(
  taskId: number,
): Promise<TaskAuditEvent[]> {
  const { data: events } = await apiClient.get<TaskAuditEvent[]>(
    `/tasks/${taskId}/audit`,
  );
  return events.map((event) => ({
    ...event,
    created_at: new Date(event.created_at),
  }));
}

/**
 * Add a task dependency
 */
export async function addTaskDependency(
  taskId: number,
  blockingTaskId: number,
): Promise<void> {
  await apiClient.post(`/tasks/${taskId}/dependencies`, {
    blocking_task_id: blockingTaskId,
  });
}

/**
 * Remove a task dependency
 */
export async function removeTaskDependency(
  taskId: number,
  blockingTaskId: number,
): Promise<void> {
  await apiClient.delete(`/tasks/${taskId}/dependencies/${blockingTaskId}`);
}

/**
 * Complete a task and schedule the next occurrence
 */
export async function completeAndScheduleTask(
  taskId: number,
  days: number,
): Promise<void> {
  await apiClient.post(`/tasks/${taskId}/complete-and-schedule`, { days });
}

// ===== Subtask API Functions =====

/**
 * Create a subtask under a parent task
 */
export async function createSubtask(
  parentId: number,
  task: Partial<Task>,
): Promise<Task> {
  const { data } = await apiClient.post<Task>(
    `/tasks/${parentId}/subtasks`,
    task,
  );
  return processTaskFromAPI(data);
}

/**
 * Set or clear the parent of a task
 * @param taskId The task to update
 * @param parentId The new parent ID, or null to clear
 * @remarks This API function is available for future drag-and-drop reordering of subtasks.
 *          Currently unused in the UI but the backend endpoint is fully implemented.
 */
export async function setTaskParent(
  taskId: number,
  parentId: number | null,
): Promise<Task> {
  const { data } = await apiClient.patch<Task>(`/tasks/${taskId}/parent`, {
    parent_task_id: parentId,
  });
  return processTaskFromAPI(data);
}

/** * Get all subtasks for a parent task
 */
export async function getSubtasks(parentId: number): Promise<{
  subtasks: Task[];
  total: number;
  complete_count: number;
}> {
  const { data } = await apiClient.get<{
    subtasks: Task[];
    total: number;
    complete_count: number;
  }>(`/tasks/${parentId}/subtasks`);

  return {
    subtasks: data.subtasks.map((task) => processTaskFromAPI(task)),
    total: data.total,
    complete_count: data.complete_count,
  };
}

/**
 * Batch update sort_order for multiple tasks
 */
export async function reorderTasks(
  orders: { id: number; sort_order: number }[],
): Promise<void> {
  await apiClient.put('/tasks/reorder', { orders });
}
