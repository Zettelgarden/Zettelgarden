import {
  TaskStatus,
  CreateTaskStatusParams,
  UpdateTaskStatusParams,
  ReorderTaskStatusesParams,
} from '../models/TaskStatus';

const base_url = import.meta.env.VITE_URL;

export async function fetchTaskStatuses(): Promise<TaskStatus[]> {
  const response = await fetch(`${base_url}/task-statuses`, {
    headers: {
      Authorization: `Bearer ${localStorage.getItem('token')}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch task statuses');
  }

  return await response.json();
}

export async function fetchTaskStatus(id: number): Promise<TaskStatus> {
  const response = await fetch(`${base_url}/task-statuses/${id}`, {
    headers: {
      Authorization: `Bearer ${localStorage.getItem('token')}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch task status');
  }

  return await response.json();
}

export async function createTaskStatus(
  params: CreateTaskStatusParams,
): Promise<{ id: number }> {
  const response = await fetch(`${base_url}/task-statuses`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${localStorage.getItem('token')}`,
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    throw new Error('Failed to create task status');
  }

  return await response.json();
}

export async function updateTaskStatus(
  id: number,
  params: UpdateTaskStatusParams,
): Promise<{ message: string; error: boolean }> {
  const response = await fetch(`${base_url}/task-statuses/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${localStorage.getItem('token')}`,
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    throw new Error('Failed to update task status');
  }

  return await response.json();
}

export async function deleteTaskStatus(id: number): Promise<void> {
  const response = await fetch(`${base_url}/task-statuses/${id}`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${localStorage.getItem('token')}`,
    },
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(errorText || 'Failed to delete task status');
  }
}

export async function reorderTaskStatuses(
  params: ReorderTaskStatusesParams,
): Promise<{ message: string; error: boolean }> {
  const response = await fetch(`${base_url}/task-statuses/reorder`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${localStorage.getItem('token')}`,
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    throw new Error('Failed to reorder task statuses');
  }

  return await response.json();
}
