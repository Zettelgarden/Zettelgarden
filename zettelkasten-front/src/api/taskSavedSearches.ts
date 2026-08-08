import { apiClient, getData } from './client';
import {
  TaskSavedSearch,
  CreateTaskSavedSearchParams,
  UpdateTaskSavedSearchParams,
} from '../models/TaskSavedSearch';

const BASE = '/task-saved-searches';

export function fetchTaskSavedSearches(): Promise<TaskSavedSearch[]> {
  return getData(apiClient.get<TaskSavedSearch[]>(BASE)).then(
    (searches) => searches || [],
  );
}

export function createTaskSavedSearch(
  params: CreateTaskSavedSearchParams,
): Promise<{ id: number }> {
  return getData(apiClient.post<{ id: number }>(BASE, params));
}

export function updateTaskSavedSearch(
  id: number,
  params: UpdateTaskSavedSearchParams,
): Promise<void> {
  return getData(apiClient.put<void>(`${BASE}/${id}`, params));
}

export function deleteTaskSavedSearch(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`${BASE}/${id}`));
}
