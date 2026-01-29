/**
 * React Query hooks for task data fetching and mutations
 *
 * This module provides a drop-in replacement for TaskContext with:
 * - Automatic caching and refetching
 * - Optimistic updates with rollback on error
 * - Background refetching
 * - Proper error handling
 */

import { useQuery, useMutation, useQueryClient, UseQueryResult } from '@tanstack/react-query';
import { queryKeys, mutationKeys } from '../../api/queryClient';
import {
  fetchTasks,
  fetchTask,
  saveNewTask,
  saveExistingTask,
  deleteTask,
  fetchTaskAuditEvents,
  addTaskDependency,
  removeTaskDependency,
  completeAndScheduleTask,
} from '../../api/tasks';
import { Task, TaskAuditEvent, TasksResponse } from '../../models/Task';
import { TaskListFilters } from '../../api/queryClient';

/**
 * Extract tags from task titles
 * Matches hashtags like #tagname in task titles
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
 * Hook to fetch all tasks with optional filtering
 *
 * Replaces TaskContext's getTasks() with automatic caching and refetching.
 *
 * @param filters - Optional filters for the task list
 * @returns Query result with tasks array and derived tags
 *
 * @example
 * ```tsx
 * function TaskList() {
 *   const { data: tasks, isLoading, error } = useTasks({ showCompleted: false });
 *
 *   if (isLoading) return <div>Loading...</div>;
 *   if (error) return <div>Error: {error.message}</div>;
 *
 *   return <ul>{tasks?.map(task => <TaskItem key={task.id} task={task} />)}</ul>;
 * }
 * ```
 */
export function useTasks(filters: TaskListFilters = {}): UseQueryResult<Task[], Error> & {
  tags: string[];
} {
  const query = useQuery({
    queryKey: queryKeys.tasks.list(filters),
    queryFn: () => fetchTasks(filters),
    staleTime: 2 * 60 * 1000, // 2 minutes - tasks change frequently
  });

  // Derive tags from tasks
  const tags = query.data ? extractTagsFromTasks(query.data) : [];

  return {
    ...query,
    tags,
  };
}

/**
 * Hook to fetch a single task by ID
 *
 * @param id - Task ID
 * @returns Query result with task data
 *
 * @example
 * ```tsx
 * function TaskDetail({ taskId }) {
 *   const { data: task, isLoading, error } = useTask(taskId);
 *
 *   if (isLoading) return <div>Loading...</div>;
 *   if (error) return <div>Error: {error.message}</div>;
 *
 *   return <div>{task?.title}</div>;
 * }
 * ```
 */
export function useTask(id: string | number): UseQueryResult<Task, Error> {
  return useQuery({
    queryKey: queryKeys.tasks.detail(Number(id)),
    queryFn: () => fetchTask(String(id)),
    enabled: !!id, // Only fetch if ID is provided
  });
}

/**
 * Hook to fetch audit events for a task
 *
 * @param taskId - Task ID
 * @returns Query result with audit events
 */
export function useTaskAuditEvents(taskId: number): UseQueryResult<TaskAuditEvent[], Error> {
  return useQuery({
    queryKey: queryKeys.tasks.auditEvents(taskId),
    queryFn: () => fetchTaskAuditEvents(taskId),
    enabled: !!taskId,
  });
}

/**
 * Mutation hook to create a new task
 *
 * Automatically invalidates the task list cache on success.
 *
 * @returns Mutation with status and handlers
 *
 * @example
 * ```tsx
 * function CreateTaskForm() {
 *   const createTask = useCreateTask();
 *
 *   const handleSubmit = (task: Task) => {
 *     createTask.mutate(task, {
 *       onSuccess: () => {
 *         console.log('Task created!');
 *       },
 *     });
 *   };
 *
 *   return <form onSubmit={handleSubmit}>...</form>;
 * }
 * ```
 */
export function useCreateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: mutationKeys.tasks.create(),
    mutationFn: saveNewTask,
    onSuccess: () => {
      // Invalidate all task list queries to refetch
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
    },
  });
}

/**
 * Mutation hook to update an existing task
 *
 * Supports optimistic updates - the UI updates immediately,
 * then rolls back on error.
 *
 * @returns Mutation with status and handlers
 *
 * @example
 * ```tsx
 * function TaskItem({ task }) {
 *   const updateTask = useUpdateTask();
 *
 *   const handleToggleComplete = () => {
 *     const updatedTask = { ...task, is_complete: !task.is_complete };
 *     updateTask.mutate(updatedTask);
 *   };
 *
 *   return <button onClick={handleToggleComplete}>{task.is_complete ? 'Undo' : 'Complete'}</button>;
 * }
 * ```
 */
export function useUpdateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: saveExistingTask,

    // Optimistic update
    onMutate: async (updatedTask: Task) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: queryKeys.tasks.detail(updatedTask.id) });

      // Snapshot previous value
      const previousTask = queryClient.getQueryData(queryKeys.tasks.detail(updatedTask.id));

      // Optimistically update to the new value
      queryClient.setQueryData(queryKeys.tasks.detail(updatedTask.id), updatedTask);

      // Also update the task in any list queries
      queryClient.setQueriesData(
        { queryKey: queryKeys.tasks.lists() },
        (old: Task[] | undefined) =>
          old?.map((task) => (task.id === updatedTask.id ? updatedTask : task))
      );

      // Return context with previous value for rollback
      return { previousTask };
    },

    // If mutation fails, use context returned from onMutate to roll back
    onError: (error, variables, context) => {
      if (context?.previousTask) {
        queryClient.setQueryData(
          queryKeys.tasks.detail(variables.id),
          context.previousTask
        );
      }
    },

    // Always refetch after error or success
    onSettled: (newTask) => {
      if (newTask) {
        queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(newTask.id) });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
    },
  });
}

/**
 * Mutation hook to delete a task
 *
 * @returns Mutation with status and handlers
 */
export function useDeleteTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteTask,

    onMutate: async (deletedTaskId: number) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.tasks.lists() });

      const previousTasks = queryClient.getQueryData(queryKeys.tasks.lists());

      // Optimistically remove the task from lists
      queryClient.setQueriesData(
        { queryKey: queryKeys.tasks.lists() },
        (old: Task[] | undefined) => old?.filter((task) => task.id !== deletedTaskId)
      );

      return { previousTasks };
    },

    onError: (error, variables, context) => {
      if (context?.previousTasks) {
        queryClient.setQueriesData(
          { queryKey: queryKeys.tasks.lists() },
          context.previousTasks
        );
      }
    },

    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
    },
  });
}

/**
 * Mutation hook to add a dependency between tasks
 */
export function useAddTaskDependency() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ taskId, blockingTaskId }: { taskId: number; blockingTaskId: number }) =>
      addTaskDependency(taskId, blockingTaskId),

    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(variables.taskId) });
    },
  });
}

/**
 * Mutation hook to remove a dependency between tasks
 */
export function useRemoveTaskDependency() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ taskId, blockingTaskId }: { taskId: number; blockingTaskId: number }) =>
      removeTaskDependency(taskId, blockingTaskId),

    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(variables.taskId) });
    },
  });
}

/**
 * Mutation hook to complete a task and schedule the next occurrence
 */
export function useCompleteAndScheduleTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ taskId, days }: { taskId: number; days: number }) =>
      completeAndScheduleTask(taskId, days),

    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(variables.taskId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
    },
  });
}
