# React Query Migration Plan for Zettelgarden

## Executive Summary

This document provides a comprehensive evaluation and implementation plan for introducing React Query (TanStack Query) to the Zettelgarden frontend. React Query will replace the current manual state management pattern using React Context for data fetching, resulting in reduced boilerplate, better performance, and improved developer experience.

---

## 1. Current State Analysis

### 1.1 Existing Data Fetching Patterns

The Zettelgarden frontend currently uses React Context for state management with the following patterns:

#### TaskContext (`src/contexts/TaskContext.tsx`)
```typescript
// Current pattern
const TaskContext = createContext<TaskContextType | undefined>(undefined);

export const TaskProvider: React.FC<TaskProviderProps> = ({ children }) => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [refreshTasks, setRefreshTasks] = useState(false);
  const [existingTags, setExistingTags] = useState<string[]>([]);

  const getTasks = async () => {
    await fetchTasks({ showCompleted }).then((data) => {
      setTasks(data);
      extractTags(data);
      setRefreshTasks(false);
    });
  };

  useEffect(() => {
    getTasks();
    const intervalId = setInterval(() => getTasks(), 60000);
    return () => clearInterval(intervalId);
  }, [refreshTasks, showCompleted]);

  return (
    <TaskContext.Provider value={{ tasks, refreshTasks, setRefreshTasks, getTasks, existingTags, showCompleted, setShowCompleted, updateTask }}>
      {children}
    </TaskContext.Provider>
  );
};
```

**Issues identified:**
- Manual polling every 60 seconds
- Manual refresh triggers via boolean flag
- No automatic caching or deduplication
- Boilerplate for loading/error states
- Manual tag extraction from tasks
- No built-in optimistic updates

#### TagContext (`src/contexts/TagContext.tsx`)
```typescript
// Current pattern
const [tags, setTags] = useState<Tag[]>([]);
const [refreshTags, setRefreshTags] = useState(true);

const getTags = async () => {
  await fetchUserTags().then((data) => {
    const sortedTags = data.sort((a, b) => a.name.localeCompare(b.name));
    setTags(sortedTags);
  });
};

useEffect(() => {
  if (refreshTags) {
    getTags();
    setRefreshTags(false);
  }
}, [refreshTags]);
```

**Issues identified:**
- Manual refresh pattern with boolean flag
- No caching
- Sorting logic in the context

#### useCardData Hook (`src/hooks/useCardData.ts`)
```typescript
// Current pattern
export function useCardData(cardId?: string): UseCardDataResult {
  const [viewingCard, setViewCard] = useState<Card | null>(null);
  const [parentCard, setParentCard] = useState<Card | null>(null);
  const [linkedEntities, setLinkedEntities] = useState<Entity[]>([]);
  // ... more state

  async function fetchCard(id: string) {
    // Fetches card, references, children, files, tags, tasks, entities, etc.
    // Manual error handling
    // Manual state updates
  }

  useEffect(() => {
    if (cardId) {
      fetchCard(cardId);
      loadSummaries(parseInt(cardId));
      loadAnalysis(parseInt(cardId));
    }
  }, [cardId]);

  return { viewingCard, parentCard, linkedEntities, ... };
}
```

**Issues identified:**
- Multiple parallel fetches that could be combined
- No caching between page navigations
- Manual error handling
- No automatic refetching on window focus (if desired)
- Complex state management

### 1.2 API Layer Analysis

The API layer (`src/api/`) is well-structured:

**Strengths:**
- Clean separation between API calls and state management
- Consistent use of `checkStatus()` for error handling
- Type-safe function signatures
- Proper token management via localStorage

**Example from `tasks.ts`:**
```typescript
export function fetchTasks(params: FetchTasksParams = {}): Promise<Task[]> {
  let token = localStorage.getItem("token");
  // ... pagination logic
  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(checkStatus)
    .then((response) => response.json());
}
```

This API layer is **well-suited for React Query integration** as it returns promises and handles errors consistently.

### 1.3 Authentication Pattern

Current auth in `AuthContext.tsx`:
```typescript
const [isAuthenticated, setIsAuthenticated] = useState(false);
const [isLoading, setIsLoading] = useState(true);

useEffect(() => {
  const initializeAuth = async () => {
    setIsLoading(true);
    const token = localStorage.getItem("token");
    if (token) {
      const currentUser = await getCurrentUser();
      setCurrentUser(currentUser);
    }
    setIsLoading(false);
  };
  initializeAuth();
}, []);
```

This pattern is compatible with React Query's `useQuery` for user data fetching.

---

## 2. Design Strategy

### 2.1 Pilot Feature Selection

**Recommended Pilot: Tasks Feature**

Rationale:
1. **Well-defined boundaries** - Tasks are self-contained with clear CRUD operations
2. **High usage** - Tasks are used throughout the app (Sidebar, TaskPage, CardView)
3. **Moderate complexity** - Enough complexity to demonstrate benefits without being overwhelming
4. **Existing issues** - Current polling every 60 seconds is inefficient
5. **Clear migration path** - TaskContext can be replaced incrementally

Alternative pilots (for later):
- Tags feature (simpler, good second step)
- Cards feature (more complex, save for after pilot succeeds)

### 2.2 Query Key Organization

React Query requires query keys for cache management. We'll use a hierarchical factory pattern:

```typescript
// Query key structure
export const queryKeys = {
  tasks: {
    all: ['tasks'] as const,
    lists: () => ['tasks', 'list'] as const,
    list: (filters: TaskListFilters) => ['tasks', 'list', filters] as const,
    details: () => ['tasks', 'detail'] as const,
    detail: (id: number) => ['tasks', 'detail', id] as const,
  },
  cards: {
    // Similar structure
  },
};
```

**Benefits:**
- Type-safe query keys
- Easy partial invalidation (e.g., invalidate all task lists but not details)
- Hierarchical cache management
- Consistent across the application

### 2.3 Authentication Token Handling

**Current approach:** Token stored in `localStorage`, manually included in each fetch call.

**React Query approach:** Keep the existing pattern but centralize token handling:

```typescript
// Option 1: Keep existing (recommended for minimal changes)
// API functions already handle tokens via localStorage
// No changes needed to API layer

// Option 2: Custom fetch wrapper (future enhancement)
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      queryFn: async ({ queryKey, signal }) => {
        const token = localStorage.getItem("token");
        // Custom fetch with token
      }
    }
  }
});
```

**Recommendation:** Start with Option 1 - keep existing API layer unchanged. The API functions already handle tokens correctly.

### 2.4 Optimistic Updates Strategy

React Query provides built-in optimistic updates via mutation callbacks:

```typescript
export function useUpdateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: saveExistingTask,

    // 1. Cancel outgoing refetches
    onMutate: async (updatedTask) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.tasks.detail(updatedTask.id) });

      // 2. Snapshot previous value
      const previousTask = queryClient.getQueryData(queryKeys.tasks.detail(updatedTask.id));

      // 3. Optimistically update
      queryClient.setQueryData(queryKeys.tasks.detail(updatedTask.id), updatedTask);

      return { previousTask };
    },

    // 4. Rollback on error
    onError: (error, variables, context) => {
      if (context?.previousTask) {
        queryClient.setQueryData(queryKeys.tasks.detail(variables.id), context.previousTask);
      }
    },

    // 5. Always refetch after settled
    onSettled: (newTask) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(newTask.id) });
    },
  });
}
```

This replaces the manual optimistic update pattern in `useOptimisticTaskUpdate.ts`:

```typescript
// Current pattern (more manual)
const { updateTask: updateTaskInContext } = useTaskContext();

async function updateTask(editedTask: Task) {
  setTask(editedTask); // Local state
  updateTaskInContext(editedTask); // Context state

  try {
    await saveExistingTask(editedTask);
  } catch (error) {
    setTask(task); // Rollback local
    updateTaskInContext(task); // Rollback context
  }
}
```

---

## 3. Implementation Plan

### Phase 1: Setup and Installation (Week 1)

#### Step 1.1: Install Dependencies
```bash
cd zettelkasten-front
npm install @tanstack/react-query@5
npm install -D @tanstack/react-query-devtools@5
```

#### Step 1.2: Create Query Client Setup
Create `src/api/queryClient.ts`:
- Configure QueryClient with appropriate defaults
- Create query key factory for tasks
- Create mutation key factory

#### Step 1.3: Create Provider Component
Create `src/components/ReactQueryDevtools.tsx`:
- Wrap app with QueryClientProvider
- Add devtools for development
- Document usage in comments

#### Step 1.4: Update Entry Point
Modify `src/index.tsx`:
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

### Phase 2: Pilot Implementation - Tasks (Week 2-3)

#### Step 2.1: Create Task Query Hooks
Create `src/hooks/queries/useTasks.ts`:
- `useTasks()` - Fetch task lists with filters
- `useTask(id)` - Fetch single task
- `useTaskAuditEvents(id)` - Fetch audit events
- `useCreateTask()` - Create task mutation
- `useUpdateTask()` - Update task mutation (with optimistic updates)
- `useDeleteTask()` - Delete task mutation
- `useAddTaskDependency()` - Add dependency mutation
- `useRemoveTaskDependency()` - Remove dependency mutation
- `useCompleteAndScheduleTask()` - Complete and reschedule mutation

#### Step 2.2: Create Example Component
Create `src/components/tasks/TaskListWithRQ.example.tsx`:
- Demonstrate new pattern
- Show before/after comparison
- Document benefits and usage

#### Step 2.3: Migrate Sidebar Component
Modify `src/components/Sidebar.tsx`:
- Replace `useTaskContext()` with `useTasks()`
- Add loading state handling
- Remove `setRefreshTasks` calls

#### Step 2.4: Migrate TaskPage Component
Modify `src/pages/tasks/TaskPage.tsx`:
- Replace context usage with query hooks
- Implement optimistic updates for task toggles
- Add error handling UI

#### Step 2.5: Migrate Task-Related Components
Update components using TaskContext:
- `TaskListItem.tsx`
- `TaskForm.tsx`
- `TaskDialog.tsx`
- `KanbanBoard.tsx`
- `EisenhowerMatrix.tsx`
- `TaskListOptionsMenu.tsx`

### Phase 3: Testing and Refinement (Week 3)

#### Step 3.1: Unit Tests
Create tests for new hooks:
```typescript
// src/hooks/queries/useTasks.test.ts
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useTasks } from './useTasks';

describe('useTasks', () => {
  it('fetches tasks successfully', async () => {
    // Test implementation
  });
});
```

#### Step 3.2: Integration Tests
Test full user flows:
- Loading task list
- Creating a task
- Updating a task
- Deleting a task
- Filtering tasks

#### Step 3.3: Performance Testing
- Measure cache hit rates
- Verify reduced API calls
- Check memory usage

### Phase 4: Card Feature Migration (Week 4-5)

#### Step 4.1: Create Card Query Hooks
Create `src/hooks/queries/useCards.ts`:
- `useCard(id)` - Fetch single card
- `useCardReferences(id)` - Fetch references
- `useCardChildren(id)` - Fetch children
- `useCardFiles(id)` - Fetch files
- `useCardTags(id)` - Fetch tags
- `useCardTasks(id)` - Fetch card tasks
- `useCardEntities(id)` - Fetch entities
- `useLinkedEntities(id)` - Fetch linked entities
- `useCardTree(id)` - Fetch card tree
- `useCardSearch(params)` - Search cards
- `useStarredCards()` - Fetch starred cards
- Mutations for create/update/delete/star/unstar

#### Step 4.2: Refactor useCardData Hook
Modify `src/hooks/useCardData.ts`:
- Replace manual state with query hooks
- Use parallel queries for card relationships
- Simplify error handling

#### Step 4.3: Migrate Card Components
- Update card viewing components
- Update card editing components
- Update search components

### Phase 5: Additional Features (Week 6)

#### Step 5.1: Tag Feature
Create `src/hooks/queries/useTags.ts`:
- Migrate TagContext to query hooks
- Remove TagContext provider

#### Step 5.2: Entity Feature
Create `src/hooks/queries/useEntities.ts`:
- Migrate entity fetching
- Add optimistic updates

#### Step 5.3: Fact Feature
Create `src/hooks/queries/useFacts.ts`:
- Migrate fact fetching and mutations

### Phase 6: Cleanup and Documentation (Week 7)

#### Step 6.1: Remove Old Contexts
- Delete `TaskContext.tsx` (keep as reference initially)
- Delete `TagContext.tsx`
- Update providers in `MainApp.tsx`
- Update tests that use old contexts

#### Step 6.2: Update Documentation
- Document migration in CLAUDE.md
- Create migration guide for contributors
- Update component documentation

#### Step 6.3: Performance Review
- Set up React Query devtools for production monitoring
- Configure appropriate stale times
- Set up query cache monitoring

---

## 4. Migration Example: Before vs After

### Before: TaskList with TaskContext

```typescript
import { useTaskContext } from '../contexts/TaskContext';

function TaskList() {
  const {
    tasks,
    existingTags,
    showCompleted,
    setShowCompleted,
    setRefreshTasks
  } = useTaskContext();

  const handleRefresh = () => {
    setRefreshTasks(true);
  };

  if (!tasks.length) return <div>No tasks</div>;

  return (
    <div>
      {tasks.map(task => <TaskItem key={task.id} task={task} />)}
    </div>
  );
}
```

**Issues:**
- No loading state
- No error handling
- Manual refresh pattern
- Tags derived manually

### After: TaskList with React Query

```typescript
import { useTasks } from '../hooks/queries/useTasks';

function TaskList() {
  const {
    data: tasks = [],
    tags,
    isLoading,
    error,
    refetch
  } = useTasks({ showCompleted: false });

  const handleRefresh = () => refetch();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;
  if (!tasks.length) return <div>No tasks</div>;

  return (
    <div>
      {tasks.map(task => <TaskItem key={task.id} task={task} />)}
    </div>
  );
}
```

**Benefits:**
- Built-in loading state
- Built-in error handling
- One-click refresh
- Tags automatically derived
- Automatic caching
- Background refetching

---

## 5. Proof of Concept

The following files have been created as a working proof of concept:

### Created Files

1. **`src/api/queryClient.ts`**
   - QueryClient configuration
   - Query key factory for all resources
   - Mutation key factory
   - Type definitions

2. **`src/hooks/queries/useTasks.ts`**
   - Complete task query hooks
   - Mutation hooks with optimistic updates
   - Comprehensive documentation

3. **`src/hooks/queries/useCards.ts`**
   - Complete card query hooks
   - Search functionality
   - Star/unstar mutations with optimistic updates

4. **`src/hooks/queries/useAuth.ts`**
   - Authentication query hooks
   - User data fetching
   - Subscription status

5. **`src/hooks/queries/useTags.ts`**
   - Tag query hooks
   - Automatic sorting

6. **`src/components/ReactQueryDevtools.tsx`**
   - Provider component setup
   - Devtools integration

7. **`src/components/tasks/TaskListWithRQ.example.tsx`**
   - Example component showing migration
   - Before/after comparison
   - Usage documentation

8. **`src/components/SidebarWithRQ.example.tsx`**
   - Migrated Sidebar component
   - Shows how to replace TaskContext

### Usage Example

```typescript
// Install dependencies
npm install @tanstack/react-query@5
npm install -D @tanstack/react-query-devtools@5

// Update src/index.tsx
import { QueryClientProvider } from './components/ReactQueryDevtools';

// In a component
import { useTasks, useUpdateTask } from './hooks/queries/useTasks';

function MyComponent() {
  const { data: tasks, isLoading, error, tags } = useTasks({ showCompleted: false });
  const updateTask = useUpdateTask();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <div>Tags: {tags.join(', ')}</div>
      {tasks.map(task => (
        <div key={task.id}>
          {task.title}
          <button onClick={() => updateTask.mutate({ ...task, is_complete: true })}>
            Complete
          </button>
        </div>
      ))}
    </div>
  );
}
```

---

## 6. Risks and Mitigations

### Risk 1: Breaking Changes During Migration

**Mitigation:**
- Incremental migration - feature by feature
- Keep old contexts alongside new hooks during transition
- Run full test suite after each migration step
- Feature flag for gradual rollout

### Risk 2: Performance Regression

**Mitigation:**
- Benchmark before and after
- Configure appropriate stale times
- Monitor query cache performance
- Use React Query Devtools in development

### Risk 3: Learning Curve for Team

**Mitigation:**
- Comprehensive documentation
- Example components with before/after
- Team training session
- Pair programming for initial migrations

### Risk 4: Cache Invalidation Issues

**Mitigation:**
- Well-defined query key hierarchy
- Clear documentation of invalidation patterns
- Automated tests for cache behavior
- Use TypeScript for type safety

### Risk 5: Testing Complexity

**Mitigation:**
- Create custom test utilities
- Document testing patterns
- Use React Query's testing utilities
- Keep tests isolated per query hook

---

## 7. Success Metrics

### Performance Metrics

| Metric | Before | Target | Measurement |
|--------|--------|--------|-------------|
| API calls on page load | ~10-15 | ~3-5 | Browser DevTools |
| Time to interactive | ~2s | <1s | Lighthouse |
| Cache hit rate | 0% | >60% | React Query Devtools |
| Bundle size increase | - | <15KB | Bundle analyzer |

### Developer Experience Metrics

| Metric | Before | Target |
|--------|--------|--------|
| Lines of boilerplate per data fetch | ~30 | ~5 |
| Time to implement new feature | 4h | 2h |
| Context providers needed | 8 | 1 |
| Manual refresh patterns | Yes | No |

### Quality Metrics

| Metric | Target |
|--------|--------|
| Test coverage for query hooks | >80% |
| TypeScript errors | 0 |
| Runtime errors | 0 |
| Console warnings | 0 |

---

## 8. Rollback Plan

If issues arise during migration:

1. **Feature-level rollback:** Each feature is migrated independently, so we can revert individual features.

2. **Code rollback:** Git branches for each migration phase allow easy reversion.

3. **Runtime rollback:** Keep TaskContext and other contexts available during transition period.

4. **Data rollback:** No data migration needed - this is a frontend-only change.

---

## 9. Next Steps

1. **Review this plan** with the team
2. **Set up a branch** for the migration: `feature/react-query-migration`
3. **Complete Phase 1** (Setup) - estimated 1 day
4. **Complete Phase 2** (Pilot) - estimated 1 week
5. **Review pilot results** before proceeding to full migration
6. **Continue with remaining phases** based on pilot success

---

## 10. Additional Resources

- [React Query Documentation](https://tanstack.com/query/latest)
- [React Query Quick Start](https://tanstack.com/query/latest/docs/react/quick-start)
- [React Query Devtools](https://tanstack.com/query/latest/docs/react/devtools)
- [Query Key Factory Pattern](https://tkdodo.eu/blog/effective-react-query-keys#use-query-key-factories)
- [Optimistic Updates](https://tanstack.com/query/latest/docs/react/guides/optimistic-updates)

---

## Appendix: File Reference

### New Files Created

```
zettelkasten-front/src/
├── api/
│   └── queryClient.ts                 # Query client and key factory
├── hooks/queries/
│   ├── useTasks.ts                    # Task query hooks (proof of concept)
│   ├── useCards.ts                    # Card query hooks (proof of concept)
│   ├── useAuth.ts                     # Auth query hooks (proof of concept)
│   └── useTags.ts                     # Tag query hooks (proof of concept)
└── components/
    ├── ReactQueryDevtools.tsx         # Provider component
    ├── tasks/
    │   └── TaskListWithRQ.example.tsx # Example task component
    └── SidebarWithRQ.example.tsx      # Example sidebar component
```

### Files to Modify (in phases)

```
zettelkasten-front/src/
├── index.tsx                          # Add QueryClientProvider
├── pages/
│   ├── MainApp.tsx                    # Remove old context providers
│   ├── tasks/
│   │   └── TaskPage.tsx               # Use query hooks
│   └── cards/
│       └── ViewPageContainer.tsx      # Use query hooks
├── components/
│   ├── Sidebar.tsx                    # Use query hooks
│   └── tasks/
│       ├── TaskListItem.tsx           # Use query hooks
│       ├── TaskForm.tsx               # Use query hooks
│       └── ...
└── contexts/
    ├── TaskContext.tsx                # Eventually remove
    ├── TagContext.tsx                 # Eventually remove
    └── ...
```

### Files to Remove (after full migration)

```
zettelkasten-front/src/
├── contexts/
│   ├── TaskContext.tsx                # Replace with query hooks
│   └── TagContext.tsx                 # Replace with query hooks
└── hooks/
    └── useOptimisticTaskUpdate.ts     # Replace with mutation callbacks
```
