/**
 * Hook for task saved searches (user-saved task "views"), synced to the backend.
 *
 * Uses plain state + the existing API client rather than react-query, matching
 * the established starred-searches pattern (the app does not yet mount a global
 * QueryClientProvider). Exposes the list plus async create/update/remove that
 * re-fetch on success so the menu stays in sync.
 */
import { useCallback, useEffect, useState } from "react";
import {
  fetchTaskSavedSearches,
  createTaskSavedSearch,
  updateTaskSavedSearch,
  deleteTaskSavedSearch,
} from "../api/taskSavedSearches";
import {
  CreateTaskSavedSearchParams,
  UpdateTaskSavedSearchParams,
  TaskSavedSearch,
} from "../models/TaskSavedSearch";

export function useTaskSavedSearches() {
  const [searches, setSearches] = useState<TaskSavedSearch[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isError, setIsError] = useState(false);

  const load = useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await fetchTaskSavedSearches();
      setSearches(data);
      setIsError(false);
    } catch {
      setIsError(true);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const create = useCallback(
    async (params: CreateTaskSavedSearchParams) => {
      await createTaskSavedSearch(params);
      await load();
    },
    [load],
  );

  const update = useCallback(
    async (id: number, params: UpdateTaskSavedSearchParams) => {
      await updateTaskSavedSearch(id, params);
      await load();
    },
    [load],
  );

  const remove = useCallback(
    async (id: number) => {
      await deleteTaskSavedSearch(id);
      await load();
    },
    [load],
  );

  return { searches, isLoading, isError, create, update, remove, refetch: load };
}
