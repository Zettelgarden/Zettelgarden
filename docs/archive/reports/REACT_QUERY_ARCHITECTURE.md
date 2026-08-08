> **ARCHIVED** — Historical document moved to `docs/archive/` on 2026-08-08 during the documentation audit (Zettelgarden-0ui). Does not describe the current app; kept for the record.

# React Query Architecture for Zettelgarden

## Visual Overview

### Current Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Zettelgarden Frontend                      │
│                                                                     │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐ │
│  │   Components    │───▶│    Contexts     │───▶│      API        │ │
│  │                 │    │                 │    │                 │ │
│  │ • Sidebar       │    │ • TaskContext   │    │ • fetchTasks()  │ │
│  │ • TaskList      │    │ • TagContext    │    │ • saveTask()    │ │
│  │ • TaskForm      │    │ • AuthContext   │    │ • fetchTags()   │ │
│  │ • CardView      │    │ • CardContext   │    │ • getCard()     │ │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘ │
│                                  │                    │            │
│                                  │                    ▼            │
│                         ┌─────────────────────────────────────────┐│
│                         │  Manual State Management                ││
│                         │  • useState for data                    ││
│                         │  • useEffect for fetching               ││
│                         │  • setInterval for polling (60s)        ││
│                         │  • setRefresh flags                     ││
│                         │  • Manual optimistic updates            ││
│                         └─────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────┘

ISSUES WITH CURRENT ARCHITECTURE:
• Boilerplate in every context (fetch, loading, error, refresh)
• Manual polling wastes resources
• No automatic caching between navigations
• No request deduplication
• Manual optimistic update logic
• Difficult to test (requires provider wrapping)
```

### Proposed Architecture with React Query

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      Zettelgarden Frontend with React Query             │
│                                                                          │
│  ┌─────────────────┐    ┌───────────────────────────────────────────┐  │
│  │   Components    │    │          React Query Hooks                 │  │
│  │                 │    │                                           │  │
│  │ • Sidebar       │───▶│ • useTasks()                              │  │
│  │ • TaskList      │    │   - data: Task[]                          │  │
│  │ • TaskForm      │    │   - tags: string[]                        │  │
│  │ • CardView      │    │   - isLoading: boolean                    │  │
│  │                 │    │   - error: Error | null                   │  │
│  │                 │    │   - refetch(): void                       │  │
│  │                 │    │                                           │  │
│  │                 │    │ • useCards()                              │  │
│  │                 │    │ • useAuth()                               │  │
│  │                 │    │ • useTags()                               │  │
│  └─────────────────┘    └───────────────────────────────────────────┘  │
│                                    │                                   │
│                                    ▼                                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                     Query Cache Layer                            │   │
│  │                                                                  │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │   │
│  │  │   Tasks     │  │    Cards    │  │    Tags     │              │   │
│  │  │   Cache     │  │    Cache    │  │    Cache    │              │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │   │
│  │                                                                  │   │
│  │  Features:                                                       │   │
│  │  • Automatic caching (staleTime configurable)                   │   │
│  │  • Background refetching                                         │   │
│  │  • Request deduplication                                        │   │
│  │  • Optimistic updates                                            │   │
│  │  • Retry logic                                                   │   │
│  │  • Window focus refetch (optional)                               │   │
│  │                                                                  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                   │
│                                    ▼                                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                        Existing API Layer                        │   │
│  │                                                                  │   │
│  │  • fetchTasks()      • saveTask()      • deleteTask()           │   │
│  │  • getCard()         • saveCard()      • deleteCard()           │   │
│  │  • fetchTags()       • getUser()                               │   │
│  │                                                                  │   │
│  │  (No changes needed - works as-is)                               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘

BENEFITS OF NEW ARCHITECTURE:
• Automatic caching with configurable staleTime
• Background refetching (no more setInterval)
• Request deduplication (same query = one request)
• Built-in loading and error states
• Optimistic updates with rollback
• Better testability (direct hook testing)
• Devtools for debugging
• Less boilerplate code
```

## Data Flow Comparison

### Current Data Flow (TaskContext)

```
User Action           Component           Context              API
─────────────────────────────────────────────────────────────────────────

Load Tasks
                           │
                           ├─────────────▶ setRefreshTasks(true)
                           │
                           │                 │
                           │                 ▼
                           │            useEffect triggered
                           │            │
                           │            ▼
                           │         fetchTasks()
                           │──────────────────────────────▶ GET /tasks
                           │            │                       │
                           │            │                       │
                           │            ◀───────────────────────┘
                           │            │
                           │            ▼
                           │        setTasks(data)
                           │        extractTags(data)
                           │            │
                           │            ▼
                           ◀───────────◀
                           (re-render with tasks)

Every 60 seconds
                           │
                           │                 │
                           │                 ▼
                           │            setInterval fires
                           │            │
                           │            ▼
                           │         fetchTasks()
                           │──────────────────────────────▶ GET /tasks
                           │            │
                           │            ◀───────────────────────┘
                           │            │
                           │            ▼
                           │        setTasks(data)
                           │            │
                           │            ▼
                           ◀───────────◀
                           (re-render)

Manual Refresh
                           │
                           ├─────────────▶ setRefreshTasks(true)
                           │            (same as initial load)
```

### New Data Flow (React Query)

```
User Action           Component           Query Cache           API
─────────────────────────────────────────────────────────────────────────

Initial Load
                           │
                           ├─────────────▶ useTasks()
                           │            │
                           │            ▼
                           │         Check Cache
                           │         (stale? empty?)
                           │            │
                           │            │ (yes, needs fetch)
                           │            ▼
                           │         fetchTasks()
                           │──────────────────────────────▶ GET /tasks
                           │            │                       │
                           │            ◀───────────────────────┘
                           │            │
                           │            ▼
                           │         Cache data
                           │         (5 min staleTime)
                           │            │
                           │            ▼
                           ◀───────────◀
                      { data, isLoading, error }

Navigate Away & Back
                           │
                           ├─────────────▶ useTasks()
                           │            │
                           │            ▼
                           │         Check Cache
                           │         (still fresh?)
                           │            │
                           │            │ (yes, 2 min old)
                           │            ▼
                           ◀───────────◀
                  (instant render from cache)

Background Refetch
                           │
                           │         (auto, 5 min)
                           │            │
                           │            ▼
                           │         fetchTasks()
                           │──────────────────────────────▶ GET /tasks
                           │            │
                           │            ◀───────────────────────┘
                           │            │
                           │            ▼
                           │         Update cache
                           │            │
                           │            ▼
                           ◀───────────◀
                  (silent update, no re-render if unchanged)

Update Task
                           │
                           ├─────────────▶ updateTask.mutate(newData)
                           │            │
                           │            ▼
                           │         Optimistic Update
                           │         (update cache immediately)
                           │            │
                           │            ▼
                           ◀───────────◀
                  (instant UI update)
                           │            │
                           │            │ (send to server)
                           │            ▼
                           │         saveTask()
                           │──────────────────────────────▶ PUT /tasks/:id
                           │            │                       │
                           │            │         Success?
                           │            │         │
                           │   ◀────────┴───────────────────┘
                           │   │
                           │   Error?  │  Success
                           │   │       │     │
                           │   ▼       │     ▼
                           │ Rollback  │  Invalidate cache
                           │ cache     │  (refetch if needed)
                           │   │       │     │
                           │   └───────┴─────┴────▶
                           │
                           ◀───────────────────────◀
                  (final UI state)
```

## File Structure

### Before (Current)

```
src/
├── api/
│   ├── tasks.ts                    # API calls (keep)
│   ├── cards.ts                    # API calls (keep)
│   ├── common.ts                   # Utilities (keep)
│   └── ...
├── contexts/
│   ├── TaskContext.tsx             # ❌ Will be replaced
│   ├── TagContext.tsx              # ❌ Will be replaced
│   ├── CardContext.tsx             # ⚠️  Partially replaced (UI-only state)
│   ├── AuthContext.tsx             # ⚠️  Partially replaced (keep auth logic)
│   └── ...
├── hooks/
│   ├── useCardData.ts              # ❌ Will be replaced
│   ├── useOptimisticTaskUpdate.ts  # ❌ Will be replaced
│   ├── useTaskFiltering.ts         # ✅ Keep (UI logic)
│   └── ...
├── components/
│   ├── Sidebar.tsx                 # ⚠️  Modify to use hooks
│   ├── tasks/
│   │   ├── TaskList.tsx            # ⚠️  Modify to use hooks
│   │   └── ...
│   └── ...
└── pages/
    ├── MainApp.tsx                 # ⚠️  Remove providers
    └── ...
```

### After (With React Query)

```
src/
├── api/
│   ├── tasks.ts                    # ✅ API calls (unchanged)
│   ├── cards.ts                    # ✅ API calls (unchanged)
│   ├── common.ts                   # ✅ Utilities (unchanged)
│   ├── queryClient.ts              # ✅ NEW: Query client setup
│   └── ...
├── contexts/
│   ├── TaskContext.tsx             # ❌ DELETED: Replaced by hooks
│   ├── TagContext.tsx              # ❌ DELETED: Replaced by hooks
│   ├── CardContext.tsx             # ✅ KEEP: UI-only state
│   ├── AuthContext.tsx             # ✅ KEEP: Auth logic (refined)
│   ├── ChatContext.tsx             # ✅ KEEP: Chat state
│   └── ...
├── hooks/
│   ├── queries/                    # ✅ NEW: React Query hooks
│   │   ├── useTasks.ts             # ✅ Task queries & mutations
│   │   ├── useCards.ts             # ✅ Card queries & mutations
│   │   ├── useAuth.ts              # ✅ Auth queries
│   │   ├── useTags.ts              # ✅ Tag queries
│   │   └── ...
│   ├── useCardData.ts              # ❌ DELETED: Use useCards() instead
│   ├── useOptimisticTaskUpdate.ts  # ❌ DELETED: Built into mutations
│   ├── useTaskFiltering.ts         # ✅ KEEP: UI logic
│   └── ...
├── components/
│   ├── ReactQueryDevtools.tsx      # ✅ NEW: Provider setup
│   ├── Sidebar.tsx                 # ✅ UPDATED: Use hooks directly
│   ├── tasks/
│   │   ├── TaskList.tsx            # ✅ UPDATED: Use hooks directly
│   │   └── ...
│   └── ...
└── index.tsx                       # ✅ UPDATED: Add QueryClientProvider
```

## Context vs Query Hook Mapping

| Old Context                   | New Query Hook(s)                                                                                                                                        | Notes                                            |
| ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| `TaskContext`                 | `useTasks()`, `useTask()`, `useCreateTask()`, `useUpdateTask()`, `useDeleteTask()`                                                                       | Full replacement                                 |
| `TagContext`                  | `useTags()`                                                                                                                                              | Full replacement                                 |
| `useCardData()`               | `useCard()`, `useCardReferences()`, `useCardChildren()`, `useCardFiles()`, `useCardTags()`, `useCardTasks()`, `useCardEntities()`, `useLinkedEntities()` | Split into focused hooks                         |
| `AuthContext` (data fetching) | `useCurrentUser()`, `useIsAdmin()`, `useUserSubscription()`                                                                                              | Partial replacement - keep auth logic in context |
| `CardContext`                 | (keep)                                                                                                                                                   | UI-only state, not data fetching                 |
| `ChatContext`                 | (keep)                                                                                                                                                   | UI-only state, not data fetching                 |
| `FileContext`                 | (keep)                                                                                                                                                   | UI-only state, not data fetching                 |
| `PinContext`                  | (keep)                                                                                                                                                   | UI-only state, not data fetching                 |

## Query Key Hierarchy

```
queryKeys
├── auth
│   ├── all: ['auth']
│   ├── current(): ['auth', 'current']
│   ├── admin(): ['auth', 'admin']
│   └── subscription(userId): ['auth', 'subscription', userId]
├── tasks
│   ├── all: ['tasks']
│   ├── lists(): ['tasks', 'list']
│   ├── list(filters): ['tasks', 'list', filters]
│   ├── details(): ['tasks', 'detail']
│   └── detail(id): ['tasks', 'detail', id]
├── cards
│   ├── all: ['cards']
│   ├── lists(): ['cards', 'list']
│   ├── list(filters): ['cards', 'list', filters]
│   ├── details(): ['cards', 'detail']
│   ├── detail(id): ['cards', 'detail', id]
│   ├── search(params): ['cards', 'search', params]
│   ├── starred(): ['cards', 'starred']
│   └── ...
└── tags
    └── all: ['tags']
```

## Cache Invalidation Examples

```typescript
// Invalidate all task lists
queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });

// Invalidate specific task
queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(taskId) });

// Invalidate everything (use sparingly)
queryClient.invalidateQueries();

// Set data manually (optimistic update)
queryClient.setQueryData(queryKeys.tasks.detail(taskId), newTask);

// Get cached data
const task = queryClient.getQueryData(queryKeys.tasks.detail(taskId));
```

## Testing Comparison

### Before (Context)

```typescript
// Need to wrap components in providers
test('renders tasks', () => {
  render(
    <TaskProvider>
      <TaskList />
    </TaskProvider>
  );
  // Complex setup...
});
```

### After (React Query)

```typescript
// Test hooks directly
test('fetches tasks', () => {
  const { result } = renderHook(() => useTasks(), { wrapper });
  expect(result.current.data).toEqual(mockTasks);
});
```
