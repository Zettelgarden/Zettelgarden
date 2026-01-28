/**
 * PROOF OF CONCEPT: React Query for Server State
 *
 * This file demonstrates how TaskContext, TagContext, and StatusContext
 * can be replaced with React Query hooks.
 *
 * Benefits:
 * - No provider needed
 * - Automatic caching and refetching
 * - Built-in loading and error states
 * - Less boilerplate code
 */

import { useQuery, useMutation, useQueryClient, UseQueryOptions } from '@tanstack/react-query';
import { fetchTasks, Task } from '../tasks';
import { fetchUserTags, Tag } from '../tags';
import { fetchTaskStatuses, TaskStatus } from '../taskStatuses';

// =============================================================================
// TASK QUERIES
// =============================================================================

interface UseTasksOptions {
  showCompleted?: boolean;
  enabled?: boolean;
}

/**
 * Replaces: TaskContext (106 lines → ~15 lines)
 *
 * Before:
 *   const { tasks, setRefreshTasks } = useTaskContext();
 *
 * After:
 *   const { data: tasks, isLoading, error } = useTasks({ showCompleted });
 */
export function useTasks(
  options: UseTasksOptions = {}
): UseQueryOptions<Task[]> & { data: Task[] | undefined; isLoading: boolean; error: unknown } {
  const { showCompleted = false, enabled = true } = options;

  const result = useQuery({
    queryKey: ['tasks', { showCompleted }],
    queryFn: () => fetchTasks({ showCompleted }),
    enabled,
    // 60-second polling (replaces the interval in TaskContext)
    refetchInterval: 60000,
    // Stale time of 30 seconds - will refetch if data is older than 30s
    staleTime: 30000,
  });

  return {
    ...result,
    // Extract commonly used properties
    data: result.data,
    isLoading: result.isLoading,
    error: result.error,
  };
}

/**
 * Optimistic task update hook
 *
 * Replaces the updateTask function from TaskContext
 */
export function useUpdateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (updatedTask: Task) => {
      const response = await fetch(`/api/tasks/${updatedTask.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updatedTask),
      });
      return response.json();
    },
    // Optimistic update
    onMutate: async (updatedTask) => {
      // Cancel ongoing refetches
      await queryClient.cancelQueries({ queryKey: ['tasks'] });

      // Snapshot previous value
      const previousTasks = queryClient.getQueryData(['tasks', { showCompleted: false }]);

      // Optimistically update
      queryClient.setQueryData(['tasks', { showCompleted: false }], (old: Task[] = []) =>
        old.map((task) => (task.id === updatedTask.id ? updatedTask : task))
      );

      // Return context with previous value
      return { previousTasks };
    },
    // Rollback on error
    onError: (err, variables, context) => {
      if (context?.previousTasks) {
        queryClient.setQueryData(['tasks', { showCompleted: false }], context.previousTasks);
      }
    },
    // Refetch on success
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
  });
}

/**
 * Task refresh trigger
 *
 * Replaces: const { setRefreshTasks } = useTaskContext();
 * Usage: setRefreshTasks(true);
 *
 * After: refetchTasks();
 */
export function useRefetchTasks() {
  const queryClient = useQueryClient();

  return () => {
    queryClient.invalidateQueries({ queryKey: ['tasks'] });
  };
}

// =============================================================================
// TAG QUERIES
// =============================================================================

/**
 * Replaces: TagContext (62 lines → ~15 lines)
 *
 * Before:
 *   const { tags } = useTagContext();
 *   const { setRefreshTags } = useTagContext(); // to refresh
 *
 * After:
 *   const { data: tags } = useTags();
 *   const refetchTags = useRefetchTags();
 */
export function useTags(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['tags'],
    queryFn: async () => {
      const data = await fetchUserTags();
      // Sort alphabetically (replaces logic in TagContext)
      return data.sort((a, b) => a.name.localeCompare(b.name));
    },
    enabled: options?.enabled ?? true,
    staleTime: 300000, // 5 minutes - tags change infrequently
  });
}

/**
 * Tag refresh trigger
 *
 * Replaces: const { setRefreshTags } = useTagContext();
 * Usage: setRefreshTags(true);
 *
 * After: refetchTags();
 */
export function useRefetchTags() {
  const queryClient = useQueryClient();

  return () => {
    queryClient.invalidateQueries({ queryKey: ['tags'] });
  };
}

// =============================================================================
// STATUS QUERIES
// =============================================================================

/**
 * Replaces: StatusContext (127 lines → ~40 lines)
 *
 * Before:
 *   const { statuses, loading, error, getStatusByName } = useStatus();
 *
 * After:
 *   const { data: statuses, isLoading, error } = useTaskStatuses();
 *   const getStatusByName = useStatusGetter();
 */
export function useTaskStatuses(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['taskStatuses'],
    queryFn: async () => {
      try {
        return await fetchTaskStatuses();
      } catch (err) {
        // Fallback to default statuses (replaces logic in StatusContext)
        return [
          {
            id: 0,
            user_id: 0,
            name: 'todo',
            display_name: 'To Do',
            color: '#6B7280',
            icon: '⭕',
            position: 0,
            is_default: true,
            is_complete_state: false,
            created_at: new Date(),
            updated_at: new Date(),
          },
          {
            id: 1,
            user_id: 0,
            name: 'in_progress',
            display_name: 'In Progress',
            color: '#3B82F6',
            icon: '🔄',
            position: 1,
            is_default: false,
            is_complete_state: false,
            created_at: new Date(),
            updated_at: new Date(),
          },
          {
            id: 2,
            user_id: 0,
            name: 'blocked',
            display_name: 'Blocked',
            color: '#EF4444',
            icon: '🚫',
            position: 2,
            is_default: false,
            is_complete_state: false,
            created_at: new Date(),
            updated_at: new Date(),
          },
          {
            id: 3,
            user_id: 0,
            name: 'done',
            display_name: 'Done',
            color: '#10B981',
            icon: '✅',
            position: 3,
            is_default: false,
            is_complete_state: true,
            created_at: new Date(),
            updated_at: new Date(),
          },
        ];
      }
    },
    enabled: options?.enabled ?? true,
    staleTime: 600000, // 10 minutes - statuses change infrequently
  });
}

/**
 * Helper hook to get status by name
 *
 * Replaces: const { getStatusByName } = useStatus();
 */
export function useStatusGetter() {
  const { data: statuses } = useTaskStatuses();

  return (name: string) => statuses?.find((status) => status.name === name);
}

/**
 * Helper hook to get default status
 *
 * Replaces: const { getDefaultStatus } = useStatus();
 */
export function useDefaultStatus() {
  const { data: statuses } = useTaskStatuses();

  return statuses?.find((status) => status.is_default);
}

/**
 * Helper hook to get complete status
 *
 * Replaces: const { getCompleteStatus } = useStatus();
 */
export function useCompleteStatus() {
  const { data: statuses } = useTaskStatuses();

  return statuses?.find((status) => status.is_complete_state);
}

// =============================================================================
// EXISTING TAGS EXTRACTION (from tasks)
// =============================================================================

/**
 * Extract unique tags from task titles
 *
 * Replaces the extractTags function in TaskContext
 */
function extractTagsFromTasks(tasks: Task[]): string[] {
  const tagSet = new Set<string>();

  tasks.forEach((task) => {
    const tagsInTitle = task.title.match(/(^|\s)#\w+(\s|$)/g);
    if (tagsInTitle) {
      tagsInTitle.forEach((tag) => tagSet.add(tag));
    }
  });

  return Array.from(tagSet).sort();
}

/**
 * Computed hook for existing tags from tasks
 *
 * Usage: const existingTags = useExistingTaskTags();
 */
export function useExistingTaskTags(showCompleted = false) {
  const { data: tasks } = useTasks({ showCompleted });

  // Compute derived state
  return tasks ? extractTagsFromTasks(tasks) : [];
}

// =============================================================================
// USAGE EXAMPLES
// =============================================================================

/**
 * Example 1: Simple task fetching
 *
 * Before (with Context):
 *   const { tasks, refreshTasks, setRefreshTasks } = useTaskContext();
 *   useEffect(() => setRefreshTasks(true), [someDependency]);
 *
 * After (with React Query):
 *   const { data: tasks, isLoading, error } = useTasks({ showCompleted: false });
 *   const refetchTasks = useRefetchTasks();
 *   // React Query handles refetching automatically
 */

/**
 * Example 2: Task list with loading state
 *
 * Before:
 *   const { tasks } = useTaskContext();
 *   // No built-in loading state
 *
 * After:
 *   const { data: tasks, isLoading } = useTasks();
 *   if (isLoading) return <div>Loading...</div>;
 */

/**
 * Example 3: Optimistic task update
 *
 * Before:
 *   const { updateTask } = useTaskContext();
 *   updateTask(updatedTask); // No optimistic update
 *
 * After:
 *   const updateTask = useUpdateTask();
 *   updateTask.mutate(updatedTask); // Optimistic update built-in
 */

/**
 * Example 4: Tags with refresh
 *
 * Before:
 *   const { tags, setRefreshTags } = useTagContext();
 *   setRefreshTags(true); // Trigger refresh
 *
 * After:
 *   const { data: tags } = useTags();
 *   const refetchTags = useRefetchTags();
 *   refetchTags(); // Trigger refresh
 */

/**
 * Example 5: Status helpers
 *
 * Before:
 *   const { getStatusByName, getDefaultStatus } = useStatus();
 *   const todoStatus = getStatusByName('todo');
 *   const defaultStatus = getDefaultStatus();
 *
 * After:
 *   const getStatusByName = useStatusGetter();
 *   const defaultStatus = useDefaultStatus();
 *   const todoStatus = getStatusByName('todo');
 */
