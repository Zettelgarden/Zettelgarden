import { Task, TaskAuditEvent, TasksResponse } from "src/models/Task";
import { apiClient, getData } from "./client";
import { processTaskFromAPI } from "../utils/taskDataProcessing";

const base_url = import.meta.env.VITE_URL;

export interface FetchTasksParams {
  showCompleted?: boolean;
  scheduledDate?: Date | null;
  completedDate?: Date | null;
  status?: string | null;
}

/**
 * Fetch tasks with optional filters
 * Handles pagination automatically to fetch all tasks
 */
export async function fetchTasks(params: FetchTasksParams = {}): Promise<Task[]> {
  const { showCompleted = false, scheduledDate = null, completedDate = null, status = null } = params;

  const fetchAllTasks = async (offset = 0, allTasks: Task[] = []): Promise<Task[]> => {
    const requestParams: Record<string, string | number | boolean | undefined> = {
      limit: 100,
      offset,
      completed: showCompleted,
    };

    if (scheduledDate) {
      requestParams.scheduled_date = scheduledDate.toISOString().split('T')[0];
    }
    if (completedDate) {
      requestParams.completed_date = completedDate.toISOString().split('T')[0];
    }
    if (status) {
      requestParams.status = status;
    }

    const { data: tasksResponse } = await apiClient.get<TasksResponse>("/tasks", {
      params: requestParams,
    });

    if (!tasksResponse.tasks) {
      return allTasks;
    }

    const formattedTasks = tasksResponse.tasks.map(task => processTaskFromAPI(task));
    const combinedTasks = [...allTasks, ...formattedTasks];

    // If we got fewer tasks than the limit, we've reached the end
    if (tasksResponse.tasks.length < tasksResponse.limit) {
      return combinedTasks;
    }

    // If there are more tasks to fetch, make another request
    return fetchAllTasks(offset + tasksResponse.limit, combinedTasks);
  };

  return fetchAllTasks();
}

/**
 * Fetch a single task by ID
 */
export async function fetchTask(id: string): Promise<Task> {
  const encoded = encodeURIComponent(id);
  const { data: rawTask } = await apiClient.get<Task>(`/tasks/${encoded}`);
  return processTaskFromAPI(rawTask);
}

/**
 * Save a new task
 */
export async function saveNewTask(task: Task): Promise<Task> {
  return saveTask("/tasks", "POST", task);
}

/**
 * Save an existing task
 */
export async function saveExistingTask(task: Task): Promise<Task> {
  return saveTask(`/tasks/${encodeURIComponent(task.id)}`, "PUT", task);
}

/**
 * Save task (internal function)
 */
async function saveTask(url: string, method: string, task: Task): Promise<Task> {
  if (method === "POST") {
    const { data } = await apiClient.post<Task>(url, task);
    return data;
  } else {
    const { data } = await apiClient.put<Task>(url, task);
    return data;
  }
}

/**
 * Delete a task
 */
export async function deleteTask(id: number): Promise<Task | null> {
  const encodedId = encodeURIComponent(id);
  const { response, data } = await apiClient.delete<Task>(`/tasks/${encodedId}`);

  if (response.status === 204) {
    return null;
  }
  return data;
}

/**
 * Fetch audit events for a task
 */
export async function fetchTaskAuditEvents(taskId: number): Promise<TaskAuditEvent[]> {
  const { data: events } = await apiClient.get<TaskAuditEvent[]>(`/tasks/${taskId}/audit`);
  return events.map((event) => ({
    ...event,
    created_at: new Date(event.created_at)
  }));
}

/**
 * Add a task dependency
 */
export async function addTaskDependency(taskId: number, blockingTaskId: number): Promise<void> {
  await apiClient.post(`/tasks/${taskId}/dependencies`, {
    blocking_task_id: blockingTaskId,
  });
}

/**
 * Remove a task dependency
 */
export async function removeTaskDependency(taskId: number, blockingTaskId: number): Promise<void> {
  await apiClient.delete(`/tasks/${taskId}/dependencies/${blockingTaskId}`);
}

/**
 * Complete a task and schedule the next occurrence
 */
export async function completeAndScheduleTask(taskId: number, days: number): Promise<void> {
  await apiClient.post(`/tasks/${taskId}/complete-and-schedule`, { days });
}
