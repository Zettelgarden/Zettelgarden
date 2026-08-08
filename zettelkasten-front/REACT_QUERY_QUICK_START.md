# React Query Quick Start Guide

## Installation

```bash
cd zettelkasten-front
npm install @tanstack/react-query@5
npm install -D @tanstack/react-query-devtools@5
```

## Basic Setup

### 1. Update `src/index.tsx`

Add the QueryClientProvider:

```typescript
import { QueryClientProvider } from './components/ReactQueryDevtools';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <QueryClientProvider>  {/* Add this */}
          <App />
        </QueryClientProvider>
      </AuthProvider>
    </BrowserRouter>
  </React.StrictMode>
);
```

## Common Patterns

### Fetching Data (Query)

```typescript
import { useTasks } from '../hooks/queries/useTasks';

function TaskList() {
  const {
    data: tasks = [],      // Fallback to empty array
    isLoading,
    error,
    refetch,
  } = useTasks({ showCompleted: false });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <ul>
      {tasks.map(task => <li key={task.id}>{task.title}</li>)}
    </ul>
  );
}
```

### Creating Data (Mutation)

```typescript
import { useCreateTask } from '../hooks/queries/useTasks';

function CreateTaskForm() {
  const createTask = useCreateTask();

  const handleSubmit = (taskData: Task) => {
    createTask.mutate(taskData, {
      onSuccess: () => {
        console.log('Task created!');
        // Cache automatically invalidated
      },
      onError: (error) => {
        console.error('Failed to create task:', error);
      },
    });
  };

  return <form onSubmit={handleSubmit}>...</form>;
}
```

### Updating Data with Optimistic Updates

```typescript
import { useUpdateTask } from '../hooks/queries/useTasks';

function TaskItem({ task }) {
  const updateTask = useUpdateTask();

  const handleToggleComplete = () => {
    const updatedTask = { ...task, is_complete: !task.is_complete };
    updateTask.mutate(updatedTask);
    // UI updates immediately, rollback on error
  };

  return (
    <div>
      <input
        type="checkbox"
        checked={task.is_complete}
        onChange={handleToggleComplete}
      />
      {task.title}
    </div>
  );
}
```

### Dependent Queries

```typescript
import { useCurrentUser, useUserSubscription } from '../hooks/queries/useAuth';

function UserProfile() {
  const { data: user, isLoading: userLoading } = useCurrentUser();

  // This query won't run until userId is available
  const { data: subscription } = useUserSubscription(user?.id ?? 0);

  if (userLoading) return <div>Loading...</div>;

  return <div>{user?.username}</div>;
}
```

### Pagination

```typescript
import { useCardSearch } from '../hooks/queries/useCards';

function SearchResults({ searchTerm }) {
  const [page, setPage] = useState(1);

  const { data, isLoading } = useCardSearch({
    searchTerm,
    page,
    perPage: 20,
  });

  return (
    <div>
      {data?.results.map(result => <div key={result.id}>{result.title}</div>)}
      <button onClick={() => setPage(p => p - 1)} disabled={page === 1}>
        Previous
      </button>
      <button onClick={() => setPage(p => p + 1)}>
        Next
      </button>
    </div>
  );
}
```

## Migration Checklist

### From TaskContext

Before:

```typescript
const { tasks, setRefreshTasks } = useTaskContext();

const handleRefresh = () => {
  setRefreshTasks(true);
};
```

After:

```typescript
const { data: tasks, refetch } = useTasks();

const handleRefresh = () => {
  refetch();
};
```

### From TagContext

Before:

```typescript
const { tags, setRefreshTags } = useTagContext();

useEffect(() => {
  setRefreshTags(true);
}, [someDependency]);
```

After:

```typescript
const { data: tags } = useTags();
// No manual refresh needed - automatic caching and refetching
```

## Common Hooks Reference

| Hook                    | Purpose            | Returns                                     |
| ----------------------- | ------------------ | ------------------------------------------- |
| `useTasks(filters)`     | Fetch task list    | `{ data, isLoading, error, refetch, tags }` |
| `useTask(id)`           | Fetch single task  | `{ data, isLoading, error }`                |
| `useCreateTask()`       | Create task        | `{ mutate, mutateAsync, isPending, error }` |
| `useUpdateTask()`       | Update task        | `{ mutate, mutateAsync, isPending, error }` |
| `useDeleteTask()`       | Delete task        | `{ mutate, mutateAsync, isPending, error }` |
| `useCard(id)`           | Fetch card         | `{ data, isLoading, error }`                |
| `useCardSearch(params)` | Search cards       | `{ data, isLoading, error }`                |
| `useTags()`             | Fetch tags         | `{ data, isLoading, error }`                |
| `useCurrentUser()`      | Fetch current user | `{ data, isLoading, error }`                |

## Query Options

```typescript
// Custom stale time
const { data } = useTasks({
  staleTime: 10 * 60 * 1000, // 10 minutes
});

// Disable automatic refetch
const { data } = useTasks({
  refetchOnWindowFocus: false,
});

// Enable polling (if really needed)
const { data } = useTasks({
  refetchInterval: 60000, // Every 60 seconds
});
```

## Cache Management

```typescript
import { useQueryClient } from '@tanstack/react-query';

function ManualRefresh() {
  const queryClient = useQueryClient();

  const handleRefreshAll = () => {
    queryClient.invalidateQueries({ queryKey: ['tasks'] });
  };

  const handleResetCache = () => {
    queryClient.resetQueries({ queryKey: ['tasks'] });
  };

  return <button onClick={handleRefreshAll}>Refresh All</button>;
}
```

## Devtools

Press `Alt + T` (or `Option + T` on Mac) to toggle React Query Devtools in development mode.

Features:

- See all active queries
- Inspect query data
- Manually refetch/invalidate queries
- View query timing and status

## Troubleshooting

### Queries not refetching

```typescript
// Check if your query is enabled
const { data } = useTasks({
  enabled: !!userId, // Only fetch if userId exists
});
```

### Stale data showing

```typescript
// Reduce staleTime for fresh data
const { data } = useTasks({
  staleTime: 0, // Always consider data stale
});
```

### Too many re-renders

```typescript
// Use stable references for filters
const filters = useMemo(
  () => ({
    showCompleted: false,
    status: selectedStatus,
  }),
  [selectedStatus],
);

const { data } = useTasks(filters);
```

## Best Practices

1. **Destructure with defaults:**

   ```typescript
   const { data: tasks = [] } = useTasks();
   ```

2. **Handle loading and error states:**

   ```typescript
   if (isLoading) return <Spinner />;
   if (error) return <ErrorMessage error={error} />;
   ```

3. **Use query keys factory:**

   ```typescript
   import { queryKeys } from '../api/queryClient';
   // Consistent keys across the app
   ```

4. **Don't over-invalidate:**

   ```typescript
   // Good: Invalidate only what changed
   queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(id) });

   // Bad: Invalidate everything
   queryClient.invalidateQueries();
   ```

5. **Test hooks directly:**

   ```typescript
   import { renderHook } from '@testing-library/react';
   import { useTasks } from './useTasks';

   test('fetches tasks', () => {
     const { result } = renderHook(() => useTasks(), { wrapper });
     expect(result.current.data).toEqual(mockTasks);
   });
   ```

## Resources

- [React Query Docs](https://tanstack.com/query/latest)
- [Query Keys](https://tkdodo.eu/blog/effective-react-query-keys)
- [Optimistic Updates](https://tanstack.com/query/latest/docs/react/guides/optimistic-updates)
- [Testing Library](https://tanstack.com/query/latest/docs/react/testing)
