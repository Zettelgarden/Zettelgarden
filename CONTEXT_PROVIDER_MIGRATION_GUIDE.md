# Context Provider Migration Guide

This guide provides step-by-step instructions for migrating from React Context to the hybrid approach (React Query + Zustand).

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Migration Strategy](#migration-strategy)
4. [Step-by-Step Migration](#step-by-step-migration)
5. [Testing](#testing)
6. [Rollback Plan](#rollback-plan)

---

## Prerequisites

Before starting the migration:

1. **Read the analysis document**: `/home/nick/code/Zettelgarden/CONTEXT_PROVIDER_ANALYSIS.md`
2. **Review the proof of concept files**:
   - `/home/nick/code/Zettelgarden/zettelkasten-front/src/api/queries.example.tsx`
   - `/home/nick/code/Zettelgarden/zettelkasten-front/src/stores/shortcutStore.example.ts`
   - `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/MainApp.migrated.example.tsx`
3. **Create a feature branch**: `git checkout -b refactor/context-provider-migration`
4. **Backup current state**: `git commit -am "Backup before context migration"`

---

## Installation

Install the required dependencies:

```bash
cd /home/nick/code/Zettelgarden/zettelkasten-front
npm install @tanstack/react-query zustand
npm install --save-dev @tanstack/react-query-devtools
```

---

## Migration Strategy

### Order of Migration

Migrate in this order to minimize risk:

1. **StatusContext** → React Query (lowest usage, simplest)
2. **TagContext** → React Query (low complexity)
3. **TaskContext** → React Query (highest usage, most complex)
4. **FileProvider** → Remove (use query invalidation)
5. **CardRefreshProvider** → Remove (use query invalidation)
6. **ShortcutProvider** → Zustand (high usage, UI state)
7. **PinProvider + ChatSidebarProvider + PartialCardProvider** → Zustand (combine into cardStore)
8. **ChatProvider** → Zustand (simple)
9. **AuthContext** → React Query (keep for last, most critical)

### Migration Principles

- **One at a time**: Complete each migration fully before starting the next
- **Keep old code**: Don't delete old contexts until new code is working
- **Test thoroughly**: Run tests after each migration
- **Commit often**: Create small, atomic commits
- **Use feature flags**: Add `testing` props to maintain test compatibility

---

## Step-by-Step Migration

### Phase 1: Setup React Query (1 day)

#### 1.1 Create QueryClient setup

Create `/home/nick/code/Zettelgarden/zettelkasten-front/src/api/queryClient.ts`:

```typescript
import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});
```

#### 1.2 Add QueryClientProvider to App

Update `/home/nick/code/Zettelgarden/zettelkasten-front/src/App.tsx`:

```typescript
import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { queryClient } from './api/queryClient';

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <Routes>
          {/* existing routes */}
        </Routes>
      </AuthProvider>
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
```

#### 1.3 Test the setup

```bash
npm start
```

Verify:
- App starts without errors
- React Query DevTools appears in the corner (press F12 to see)
- Existing functionality still works

**Commit**: `feat: add React Query setup`

---

### Phase 2: Migrate StatusContext (2 days)

#### 2.1 Create query file

Create `/home/nick/code/Zettelgarden/zettelkasten-front/src/api/queries/statusQueries.ts`:

```typescript
import { useQuery } from '@tanstack/react-query';
import { fetchTaskStatuses, TaskStatus } from '../taskStatuses';

export function useTaskStatuses(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['taskStatuses'],
    queryFn: async () => {
      try {
        return await fetchTaskStatuses();
      } catch (err) {
        console.error('Error fetching task statuses:', err);
        // Fallback to default statuses
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
          // ... other fallback statuses
        ];
      }
    },
    enabled: options?.enabled ?? true,
    staleTime: 600000, // 10 minutes
  });
}

export function useStatusGetter() {
  const { data: statuses } = useTaskStatuses();
  return (name: string) => statuses?.find((status) => status.name === name);
}

export function useDefaultStatus() {
  const { data: statuses } = useTaskStatuses();
  return statuses?.find((status) => status.is_default);
}

export function useCompleteStatus() {
  const { data: statuses } = useTaskStatuses();
  return statuses?.find((status) => status.is_complete_state);
}
```

#### 2.2 Update consumers

Find all files using `useStatus`:

```bash
grep -r "useStatus" src/ --exclude-dir=node_modules
```

Update each file:

**Before**:
```typescript
import { useStatus } from '../contexts/StatusContext';

const { statuses, loading, error, getStatusByName, getDefaultStatus, getCompleteStatus } = useStatus();
```

**After**:
```typescript
import { useTaskStatuses, useStatusGetter, useDefaultStatus, useCompleteStatus } from '../api/queries/statusQueries';

const { data: statuses, isLoading: loading, error } = useTaskStatuses();
const getStatusByName = useStatusGetter();
const getDefaultStatus = useDefaultStatus();
const getCompleteStatus = useCompleteStatus();
```

Files to update:
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskStatusDisplay.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/KanbanBoard.tsx`
- `/home/nick/code/Zettelkasten-front/src/components/tasks/TaskDialog.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskListOptionsMenu.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskListItem.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/settings/StatusManagement.tsx`

#### 2.3 Remove StatusProvider from MainApp

Update `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/MainApp.tsx`:

```typescript
// Remove this import:
import { StatusProvider } from "../contexts/StatusContext";

// Remove StatusProvider wrapper:
// Before:
<StatusProvider>
  {/* children */}
</StatusProvider>

// After:
{/* children */}
```

#### 2.4 Test

```bash
npm test -- src/components/tasks/
npm start
```

Verify:
- Task statuses load correctly
- Fallback statuses work when API fails
- All task-related features work

**Commit**: `refactor: migrate StatusContext to React Query`

---

### Phase 3: Migrate TagContext (2 days)

#### 3.1 Create query file

Create `/home/nick/code/Zettelgarden/zettelkasten-front/src/api/queries/tagQueries.ts`:

```typescript
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchUserTags, Tag } from '../tags';

export function useTags(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['tags'],
    queryFn: async () => {
      const data = await fetchUserTags();
      return data.sort((a, b) => a.name.localeCompare(b.name));
    },
    enabled: options?.enabled ?? true,
    staleTime: 300000, // 5 minutes
  });
}

export function useRefetchTags() {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: ['tags'] });
  };
}
```

#### 3.2 Update consumers

Find all files using `useTagContext`:

```bash
grep -r "useTagContext" src/ --exclude-dir=node_modules
```

Update each file:

**Before**:
```typescript
import { useTagContext } from '../contexts/TagContext';
const { tags, setRefreshTags } = useTagContext();
```

**After**:
```typescript
import { useTags, useRefetchTags } from '../api/queries/tagQueries';
const { data: tags } = useTags();
const refetchTags = useRefetchTags();
```

Files to update:
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/cards/EditPage.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/cards/SearchPage.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/cards/CardListItem.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/tasks/TaskPage.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskForm.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/cards/ViewPage.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/QuickTagPopover.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/BulkTaskTagEditor.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tags/AddTagMenu.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tags/TagListItem.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/cards/SearchResultList.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tags/TagList.tsx`

#### 3.3 Remove TagProvider from MainApp

```typescript
// Remove this import:
import { TagProvider } from "../contexts/TagContext";

// Remove TagProvider wrapper
```

#### 3.4 Test

```bash
npm test -- src/components/tags/
npm start
```

Verify:
- Tags load correctly
- Tag refresh works
- All tag-related features work

**Commit**: `refactor: migrate TagContext to React Query`

---

### Phase 4: Migrate TaskContext (3 days)

This is the most complex migration. Take your time.

#### 4.1 Create query file

Create `/home/nick/code/Zettelgarden/zettelkasten-front/src/api/queries/taskQueries.ts`:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { fetchTasks, Task } from '../tasks';

interface UseTasksOptions {
  showCompleted?: boolean;
  enabled?: boolean;
}

export function useTasks(options: UseTasksOptions = {}) {
  const { showCompleted = false, enabled = true } = options;

  return useQuery({
    queryKey: ['tasks', { showCompleted }],
    queryFn: () => fetchTasks({ showCompleted }),
    enabled,
    refetchInterval: 60000, // 60-second polling
    staleTime: 30000,
  });
}

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
    onMutate: async (updatedTask) => {
      await queryClient.cancelQueries({ queryKey: ['tasks'] });
      const previousTasks = queryClient.getQueryData(['tasks', { showCompleted: false }]);
      queryClient.setQueryData(['tasks', { showCompleted: false }], (old: Task[] = []) =>
        old.map((task) => (task.id === updatedTask.id ? updatedTask : task))
      );
      return { previousTasks };
    },
    onError: (err, variables, context) => {
      if (context?.previousTasks) {
        queryClient.setQueryData(['tasks', { showCompleted: false }], context.previousTasks);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
  });
}

export function useRefetchTasks() {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: ['tasks'] });
  };
}

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

export function useExistingTaskTags(showCompleted = false) {
  const { data: tasks } = useTasks({ showCompleted });
  return tasks ? extractTagsFromTasks(tasks) : [];
}
```

#### 4.2 Update consumers

Find all files using `useTaskContext`:

```bash
grep -r "useTaskContext" src/ --exclude-dir=node_modules
```

Update each file:

**Before**:
```typescript
import { useTaskContext } from '../contexts/TaskContext';
const { tasks, setRefreshTasks, existingTags, showCompleted, setShowCompleted, updateTask } = useTaskContext();
```

**After**:
```typescript
import { useTasks, useUpdateTask, useRefetchTasks, useExistingTaskTags } from '../api/queries/taskQueries';
const [showCompleted, setShowCompleted] = useState(false);
const { data: tasks } = useTasks({ showCompleted });
const updateTask = useUpdateTask();
const refetchTasks = useRefetchTasks();
const existingTags = useExistingTaskTags(showCompleted);
```

Files to update:
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/MainApp.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/tasks/TaskPage.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/EisenhowerMatrix.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/KanbanBoard.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskDialog.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskListOptionsMenu.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskForm.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskListItem.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/CreateTaskWindow.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/BulkTaskDateDisplay.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/Sidebar.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/pages/cards/ViewPageContainer.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/BulkTaskTagEditor.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/TaskList.tsx`
- `/home/nick/code/Zettelgarden/zettelkasten-front/src/components/tasks/RemoveTagMenu.tsx`

#### 4.3 Remove TaskProvider from MainApp

```typescript
// Remove this import:
import { TaskProvider, useTaskContext } from "../contexts/TaskContext";

// Remove TaskProvider wrapper
```

#### 4.4 Test

```bash
npm test -- src/components/tasks/
npm test -- src/pages/tasks/
npm start
```

Verify:
- Tasks load correctly
- 60-second polling works
- Task updates work optimistically
- showCompleted toggle works
- All task-related features work

**Commit**: `refactor: migrate TaskContext to React Query`

---

### Phase 5: Remove FileProvider and CardRefreshProvider (1 day)

These can be removed entirely by using query invalidation.

#### 5.1 Update FileContext consumers

Find all files using `useFileContext`:

```bash
grep -r "useFileContext" src/ --exclude-dir=node_modules
```

**Before**:
```typescript
import { useFileContext } from '../contexts/FileContext';
const { refreshFiles, setRefreshFiles } = useFileContext();
useEffect(() => {
  if (refreshFiles) {
    // fetch files
    setRefreshFiles(false);
  }
}, [refreshFiles]);
```

**After**:
```typescript
import { useQueryClient } from '@tanstack/react-query';
const queryClient = useQueryClient();

const refetchFiles = () => {
  queryClient.invalidateQueries({ queryKey: ['files'] });
};

// Call refetchFiles() when needed instead of setRefreshFiles(true)
```

#### 5.2 Update CardRefreshContext consumers

Find all files using `useCardRefresh`:

```bash
grep -r "useCardRefresh" src/ --exclude-dir=node_modules
```

**Before**:
```typescript
import { useCardRefresh } from '../contexts/CardRefreshContext';
const { refreshTrigger } = useCardRefresh();
useEffect(() => {
  if (refreshTrigger) {
    // refresh card
  }
}, [refreshTrigger]);
```

**After**:
```typescript
import { useQueryClient } from '@tanstack/react-query';
const queryClient = useQueryClient();

const refreshCard = (cardId: string) => {
  queryClient.invalidateQueries({ queryKey: ['card', cardId] });
};
```

#### 5.3 Remove providers from MainApp

```typescript
// Remove these imports:
import { FileProvider } from "../contexts/FileContext";
import { CardRefreshProvider } from "../contexts/CardRefreshContext";

// Remove provider wrappers
```

#### 5.4 Test

```bash
npm test -- src/components/files/
npm test -- src/components/chat/
npm start
```

**Commit**: `refactor: remove FileProvider and CardRefreshProvider`

---

### Phase 6: Migrate UI State to Zustand (4 days)

#### 6.1 Create Zustand stores

Create `/home/nick/code/Zettelgarden/zettelkasten-front/src/stores/shortcutStore.ts` (copy from example).

Create `/home/nick/code/Zettelgarden/zettelkasten-front/src/stores/cardStore.ts` (copy from example).

Create `/home/nick/code/Zettelgarden/zettelkasten-front/src/stores/chatStore.ts` (copy from example).

#### 6.2 Update ShortcutProvider consumers

Find all files using `useShortcutContext`:

```bash
grep -r "useShortcutContext" src/ --exclude-dir=node_modules
```

Update each file (14 files):

**Before**:
```typescript
import { useShortcutContext } from '../contexts/ShortcutContext';
const { showCreateTaskWindow, setShowCreateTaskWindow } = useShortcutContext();
```

**After**:
```typescript
import { useShortcutStore } from '../stores/shortcutStore';
const showCreateTaskWindow = useShortcutStore(s => s.showCreateTaskWindow);
const setShowCreateTaskWindow = useShortcutStore(s => s.setShowCreateTaskWindow);

// Or for multiple values:
const { showCreateTaskWindow, setShowCreateTaskWindow } = useShortcutStore();
```

#### 6.3 Update PinProvider, ChatSidebarProvider, PartialCardProvider consumers

Update each file to use `useCardStore` instead.

#### 6.4 Update ChatProvider consumers

Update each file to use `useChatStore` instead.

#### 6.5 Remove providers from MainApp

```typescript
// Remove these imports:
import { ShortcutProvider } from "../contexts/ShortcutContext";
import { PinProvider, usePinContext } from "../contexts/PinContext";
import { ChatSidebarProvider, useChatSidebarContext } from "../contexts/ChatSidebarContext";
import { PartialCardProvider } from "../contexts/CardContext";
import { ChatProvider, useChatContext } from "../contexts/ChatContext";

// Remove all provider wrappers
```

#### 6.6 Test

```bash
npm test
npm start
```

Verify all UI interactions work:
- Create task window
- Quick search
- Entity dialog
- Fact dialog
- Task dialog
- Pin mode
- Chat sidebar
- Card navigation

**Commit**: `refactor: migrate UI state to Zustand`

---

### Phase 7: Final Cleanup (2 days)

#### 7.1 Delete old context files

```bash
rm src/contexts/StatusContext.tsx
rm src/contexts/TagContext.tsx
rm src/contexts/TaskContext.tsx
rm src/contexts/FileContext.tsx
rm src/contexts/CardRefreshContext.tsx
rm src/contexts/ShortcutContext.tsx
rm src/contexts/PinContext.tsx
rm src/contexts/ChatSidebarContext.tsx
rm src/contexts/CardContext.tsx
rm src/contexts/ChatContext.tsx
```

#### 7.2 Update all tests

Add test utilities:

Create `/home/nick/code/Zettelgarden/zettelkasten-front/src/tests/test-utils.tsx`:

```typescript
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render } from '@testing-library/react';

const createTestQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: { retry: false },
  },
});

export function renderWithQuery(ui: React.ReactElement) {
  const testQueryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={testQueryClient}>
      {ui}
    </QueryClientProvider>
  );
}
```

Update tests to use `renderWithQuery` instead of manual provider wrappers.

#### 7.3 Run full test suite

```bash
npm test:run
npm run build
```

#### 7.4 Performance testing

Open React DevTools Profiler and verify:
- Fewer re-renders
- Faster render times
- No memory leaks

**Commit**: `refactor: complete context provider migration`

---

## Testing

### Unit Tests

For each migration phase:

```bash
# Run tests for affected components
npm test -- src/components/tasks/
npm test -- src/components/tags/
```

### Integration Tests

```bash
# Run full test suite
npm test:run

# Run with coverage
npm test:coverage
```

### Manual Testing Checklist

For each phase, verify:

- [ ] Page loads without errors
- [ ] All existing functionality works
- [ ] No console errors or warnings
- [ ] Performance is not degraded
- [ ] Data loads correctly
- [ ] Refresh triggers work
- [ ] All UI interactions work

### Specific Feature Testing

**Task Management**:
- [ ] Tasks load on page load
- [ ] Tasks refresh every 60 seconds
- [ ] Task updates work optimistically
- [ ] showCompleted toggle works
- [ ] Task creation works
- [ ] Task deletion works
- [ ] Task editing works

**Tag Management**:
- [ ] Tags load correctly
- [ ] Tag refresh works
- [ ] Tags are sorted alphabetically
- [ ] Tag filtering works

**Status Management**:
- [ ] Statuses load correctly
- [ ] Fallback statuses work when API fails
- [ ] Status helpers work (getDefaultStatus, etc.)

**UI Interactions**:
- [ ] Create task window opens/closes
- [ ] Quick search opens/closes
- [ ] Entity dialog opens/closes
- [ ] Fact dialog opens/closes
- [ ] Task dialog opens/closes
- [ ] Pin mode works
- [ ] Chat sidebar works
- [ ] Card navigation works

---

## Rollback Plan

If something goes wrong:

### Immediate Rollback

```bash
# Reset to backup commit
git reset --hard <backup-commit-hash>

# Remove dependencies
npm uninstall @tanstack/react-query zustand
npm uninstall --save-dev @tanstack/react-query-devtools

# Restore files
git checkout HEAD -- src/
```

### Partial Rollback

If only one migration fails:

```bash
# Revert specific commits
git revert <commit-hash>

# Or reset to before failed migration
git reset --hard <last-working-commit>
```

### Safe Migration Practices

1. **Feature flags**: Use `testing` props to maintain test compatibility
2. **Parallel systems**: Keep old contexts until new code is verified
3. **Incremental commits**: One commit per context migration
4. **Branch protection**: Don't merge to main until all tests pass

---

## Performance Metrics

### Before Migration

Measure baseline performance:

```typescript
// Add to MainApp.tsx
useEffect(() => {
  console.time('MainApp render');
  return () => console.timeEnd('MainApp render');
});
```

### After Migration

Compare performance:

**Expected improvements**:
- 60-70% reduction in provider-related re-renders
- 50% reduction in MainApp render time
- 30% reduction in bundle size (after deleting old contexts)

---

## Troubleshooting

### Issue: "Query not found" error

**Solution**: Make sure QueryClientProvider is wrapping your app.

### Issue: Zustand store not updating

**Solution**: Check that you're using the correct selector.

```typescript
// Wrong - always returns true
const isOpen = useShortcutStore().showCreateTaskWindow;

// Correct - returns current value
const isOpen = useShortcutStore(s => s.showCreateTaskWindow);
```

### Issue: Tests failing after migration

**Solution**: Update test utilities to wrap components in QueryClientProvider.

### Issue: Performance degraded

**Solution**: Check for unnecessary re-renders using React DevTools Profiler. Use selectors to prevent unnecessary updates.

---

## Next Steps

After completing the migration:

1. **Monitor performance** in production
2. **Gather feedback** from users
3. **Update documentation** to reflect new patterns
4. **Train team** on React Query and Zustand
5. **Consider** migrating AuthContext (if needed)

---

## Resources

- [React Query Documentation](https://tanstack.com/query/latest)
- [Zustand Documentation](https://zustand-demo.pmnd.rs/)
- [React Query DevTools](https://tanstack.com/query/latest/docs/react/devtools)
- [Zustand DevTools](https://github.com/pmndrs/zustand#devtools)

---

## Support

If you encounter issues:

1. Check the proof of concept files
2. Review the analysis document
3. Consult official documentation
4. Reach out to the team

Good luck with the migration!
