/**
 * Tests for React Query hooks (Proof of Concept)
 *
 * This file demonstrates testing patterns for React Query hooks,
 * which is significantly easier than testing React Context.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useTasks, useUpdateTask, useCreateTask } from './useTasks';
import { fetchTasks, saveNewTask, saveExistingTask } from '../../api/tasks';
import { Task } from '../../models/Task';

// Mock the API functions
vi.mock('../../api/tasks', () => ({
  fetchTasks: vi.fn(),
  saveNewTask: vi.fn(),
  saveExistingTask: vi.fn(),
}));

// Mock localStorage for token
const localStorageMock = {
  getItem: vi.fn(() => 'fake-token'),
  setItem: vi.fn(),
  removeItem: vi.fn(),
};
global.localStorage = localStorageMock as any;

/**
 * Helper to create a test QueryClient
 */
function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false, // Disable retries for faster tests
        staleTime: 0, // Immediate staleness for testing
      },
      mutations: {
        retry: false,
      },
    },
    logger: {
      log: vi.fn(),
      warn: vi.fn(),
      error: vi.fn(),
    },
  });
}

/**
 * Helper to wrap hook with QueryClientProvider
 */
function wrapper({ children }: { children: React.ReactNode }) {
  const queryClient = createTestQueryClient();
  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

describe('useTasks', () => {
  const mockTasks: Task[] = [
    {
      id: 1,
      card_pk: 1,
      user_id: 1,
      title: 'Test Task 1',
      description: null,
      priority: 'high',
      status: 'todo',
      is_complete: false,
      is_deleted: false,
      scheduled_date: new Date('2024-01-01'),
      due_date: null,
      created_at: new Date('2024-01-01'),
      updated_at: new Date('2024-01-01'),
      completed_at: null,
      reminder_time: null,
      reminder_sent: false,
      card: null,
      tags: [],
      blocked_by: [],
      blocks: [],
    },
    {
      id: 2,
      card_pk: 1,
      user_id: 1,
      title: 'Test Task 2 #work',
      description: null,
      priority: null,
      status: 'todo',
      is_complete: false,
      is_deleted: false,
      scheduled_date: new Date('2024-01-02'),
      due_date: null,
      created_at: new Date('2024-01-01'),
      updated_at: new Date('2024-01-01'),
      completed_at: null,
      reminder_time: null,
      reminder_sent: false,
      card: null,
      tags: [],
      blocked_by: [],
      blocks: [],
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches tasks successfully', async () => {
    vi.mocked(fetchTasks).mockResolvedValue(mockTasks);

    const { result } = renderHook(() => useTasks({ showCompleted: false }), {
      wrapper,
    });

    // Initially loading
    expect(result.current.isLoading).toBe(true);

    // Wait for data to be fetched
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    // Verify data
    expect(result.current.data).toEqual(mockTasks);
    expect(fetchTasks).toHaveBeenCalledWith({ showCompleted: false });
  });

  it('extracts tags from tasks', async () => {
    vi.mocked(fetchTasks).mockResolvedValue(mockTasks);

    const { result } = renderHook(() => useTasks(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    // Verify tags are extracted
    expect(result.current.tags).toEqual(['#work']);
  });

  it('handles fetch errors', async () => {
    const error = new Error('Failed to fetch');
    vi.mocked(fetchTasks).mockRejectedValue(error);

    const { result } = renderHook(() => useTasks(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.error).toEqual(error);
    expect(result.current.data).toBeUndefined();
  });

  it('caches results between renders', async () => {
    vi.mocked(fetchTasks).mockResolvedValue(mockTasks);

    const { result, rerender } = renderHook(() => useTasks(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    // Fetch should be called once
    expect(fetchTasks).toHaveBeenCalledTimes(1);

    // Rerender should use cache
    rerender();

    // Still called only once (cached)
    expect(fetchTasks).toHaveBeenCalledTimes(1);
  });

  it('refetches when filters change', async () => {
    vi.mocked(fetchTasks).mockResolvedValue(mockTasks);

    const { result, rerender } = renderHook(
      ({ showCompleted }) => useTasks({ showCompleted }),
      {
        wrapper,
        initialProps: { showCompleted: false },
      }
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(fetchTasks).toHaveBeenCalledTimes(1);
    expect(fetchTasks).toHaveBeenLastCalledWith({ showCompleted: false });

    // Change filter
    rerender({ showCompleted: true });

    await waitFor(() => {
      expect(fetchTasks).toHaveBeenCalledTimes(2);
    });

    expect(fetchTasks).toHaveBeenLastCalledWith({ showCompleted: true });
  });
});

describe('useUpdateTask', () => {
  const mockTask: Task = {
    id: 1,
    card_pk: 1,
    user_id: 1,
    title: 'Test Task',
    description: null,
    priority: 'high',
    status: 'todo',
    is_complete: false,
    is_deleted: false,
    scheduled_date: new Date('2024-01-01'),
    due_date: null,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    completed_at: null,
    reminder_time: null,
    reminder_sent: false,
    card: null,
    tags: [],
    blocked_by: [],
    blocks: [],
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('updates a task successfully', async () => {
    const updatedTask = { ...mockTask, is_complete: true };
    vi.mocked(saveExistingTask).mockResolvedValue(updatedTask);

    const queryClient = createTestQueryClient();
    // Pre-populate cache
    queryClient.setQueryData(['tasks', 'detail', 1], mockTask);

    const { result } = renderHook(() => useUpdateTask(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    result.current.mutate(updatedTask);

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(saveExistingTask).toHaveBeenCalledWith(updatedTask);

    // Verify cache was updated (via invalidateQueries in onSettled)
    // In a real test, we'd check the cache state here
  });

  it('handles update errors', async () => {
    const error = new Error('Update failed');
    vi.mocked(saveExistingTask).mockRejectedValue(error);

    const { result } = renderHook(() => useUpdateTask(), { wrapper });

    result.current.mutate(mockTask);

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    expect(result.current.error).toEqual(error);
  });

  it('provides optimistic update', async () => {
    const updatedTask = { ...mockTask, title: 'Updated Title' };
    let resolveUpdate: (value: Task) => void;
    const updatePromise = new Promise<Task>((resolve) => {
      resolveUpdate = resolve;
    });
    vi.mocked(saveExistingTask).mockReturnValue(updatePromise);

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(['tasks', 'detail', 1], mockTask);

    const { result } = renderHook(() => useUpdateTask(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    // Start mutation
    result.current.mutate(updatedTask);

    // Optimistic update should be immediate
    await waitFor(() => {
      const cachedData = queryClient.getQueryData(['tasks', 'detail', 1]);
      expect(cachedData).toEqual(updatedTask);
    });

    // Complete the mutation
    resolveUpdate!(updatedTask);

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });
  });
});

describe('useCreateTask', () => {
  const mockTask: Task = {
    id: 1,
    card_pk: 1,
    user_id: 1,
    title: 'New Task',
    description: null,
    priority: null,
    status: 'todo',
    is_complete: false,
    is_deleted: false,
    scheduled_date: new Date('2024-01-01'),
    due_date: null,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    completed_at: null,
    reminder_time: null,
    reminder_sent: false,
    card: null,
    tags: [],
    blocked_by: [],
    blocks: [],
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates a task successfully and invalidates lists', async () => {
    vi.mocked(saveNewTask).mockResolvedValue(mockTask);

    const queryClient = createTestQueryClient();

    const { result } = renderHook(() => useCreateTask(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    result.current.mutate(mockTask);

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(saveNewTask).toHaveBeenCalledWith(mockTask);

    // Verify that task list queries were invalidated
    const invalidatedQueries = queryClient.getQueryCache().getAll();
    const hasInvalidatedTasksList = invalidatedQueries.some(
      (query) => query.state.invalidated && query.queryKey[0] === 'tasks'
    );
    expect(hasInvalidatedTasksList).toBe(true);
  });

  it('handles create errors', async () => {
    const error = new Error('Create failed');
    vi.mocked(saveNewTask).mockRejectedValue(error);

    const { result } = renderHook(() => useCreateTask(), { wrapper });

    result.current.mutate(mockTask);

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    expect(result.current.error).toEqual(error);
  });
});

/**
 * COMPARISON: Testing with React Context vs React Query
 *
 * BEFORE (React Context):
 * ```tsx
 * // Test file
 * import { render, screen } from '@testing-library/react';
 * import { TaskProvider, useTaskContext } from './TaskContext';
 *
 * function TestComponent() {
 *   const { tasks } = useTaskContext();
 *   return <div>{tasks.length} tasks</div>;
 * }
 *
 * test('renders tasks', () => {
 *   // Need to mock fetchTasks
 *   // Need to wrap in provider
 *   // Need to wait for useEffect to run
 *   // More complex setup
 *   render(
 *     <TaskProvider>
 *       <TestComponent />
 *     </TaskProvider>
 *   );
 * });
 * ```
 *
 * AFTER (React Query):
 * ```tsx
 * // Test file
 * import { renderHook } from '@testing-library/react';
 * import { useTasks } from './useTasks';
 *
 * test('fetches tasks', () => {
 *   // Just test the hook directly
 *   // Simple, focused testing
 *   const { result } = renderHook(() => useTasks(), { wrapper });
 *
 *   await waitFor(() => expect(result.current.data).toEqual(mockTasks));
 * });
 * ```
 *
 * BENEFITS:
 * 1. No provider wrapping in tests
 * 2. Direct hook testing
 * 3. Built-in loading/error states
 * 4. Easier to test optimistic updates
 * 5. Better isolation
 */
