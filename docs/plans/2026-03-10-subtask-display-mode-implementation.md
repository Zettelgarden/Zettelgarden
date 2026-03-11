# Subtask Display Mode Toggle Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add global subtask display mode toggle (nested/flat/hidden) with parent badge for flat mode

**Architecture:** Create a global hook for display mode state, integrate into TaskDesktopLayout's display menu, update TaskList/KanbanBoard/EisenhowerMatrix to respect the mode, add parent badge to TaskListItem

**Tech Stack:** React hooks, localStorage, TypeScript, existing Task components

---

## Task 1: Create useSubtaskDisplayMode Hook

**Files:**
- Create: `zettelkasten-front/src/hooks/useSubtaskDisplayMode.ts`
- Test: `zettelkasten-front/src/hooks/useSubtaskDisplayMode.test.ts`

**Step 1: Write the failing test**

```typescript
import { renderHook, act } from '@testing-library/react';
import { useSubtaskDisplayMode } from './useSubtaskDisplayMode';

describe('useSubtaskDisplayMode', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('should default to nested mode', () => {
    const { result } = renderHook(() => useSubtaskDisplayMode());
    expect(result.current.subtaskMode).toBe('nested');
  });

  it('should persist mode to localStorage', () => {
    const { result } = renderHook(() => useSubtaskDisplayMode());

    act(() => {
      result.current.setSubtaskMode('flat');
    });

    expect(result.current.subtaskMode).toBe('flat');
    expect(localStorage.getItem('subtaskDisplayMode')).toBe('"flat"');
  });

  it('should load saved mode from localStorage', () => {
    localStorage.setItem('subtaskDisplayMode', '"hidden"');

    const { result } = renderHook(() => useSubtaskDisplayMode());
    expect(result.current.subtaskMode).toBe('hidden');
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- useSubtaskDisplayMode.test.ts`
Expected: FAIL with "Cannot find module './useSubtaskDisplayMode'"

**Step 3: Write minimal implementation**

```typescript
import { useState, useEffect } from 'react';

export type SubtaskDisplayMode = 'nested' | 'flat' | 'hidden';

const STORAGE_KEY = 'subtaskDisplayMode';

export function useSubtaskDisplayMode() {
  const [subtaskMode, setSubtaskModeState] = useState<SubtaskDisplayMode>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        return JSON.parse(saved) as SubtaskDisplayMode;
      }
    } catch (e) {
      console.error('Failed to load subtask display mode:', e);
    }
    return 'nested';
  });

  const setSubtaskMode = (mode: SubtaskDisplayMode) => {
    setSubtaskModeState(mode);
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(mode));
    } catch (e) {
      console.error('Failed to save subtask display mode:', e);
    }
  };

  return { subtaskMode, setSubtaskMode };
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- useSubtaskDisplayMode.test.ts`
Expected: PASS (3 tests)

**Step 5: Commit**

```bash
git add zettelkasten-front/src/hooks/useSubtaskDisplayMode.ts
git add zettelkasten-front/src/hooks/useSubtaskDisplayMode.test.ts
git commit -m "feat: add useSubtaskDisplayMode hook with localStorage persistence"
```

---

## Task 2: Add Display Mode Toggle to TaskDesktopLayout

**Files:**
- Modify: `zettelkasten-front/src/components/tasks/TaskDesktopLayout.tsx`

**Step 1: Import the hook and add state**

Add to imports section (around line 8):
```typescript
import { useSubtaskDisplayMode } from "../../hooks/useSubtaskDisplayMode";
```

Add inside TaskDesktopLayout function (around line 170, after other hooks):
```typescript
  const { subtaskMode, setSubtaskMode } = useSubtaskDisplayMode();
```

**Step 2: Add toggle UI to display menu**

Locate the display menu dropdown (around line 380-440). Find the "Sort By" section and add this before it (after the "Select Mode" checkbox):

```typescript
                    <div className="mb-2">
                      <label className="block text-xs font-semibold mb-1">Subtask Display</label>
                      <div className="space-y-1">
                        <label className="flex items-center gap-2 text-xs cursor-pointer">
                          <input
                            type="radio"
                            name="subtaskMode"
                            checked={subtaskMode === 'nested'}
                            onChange={() => setSubtaskMode('nested')}
                            className="rounded"
                          />
                          <span>Nested</span>
                        </label>
                        <label className="flex items-center gap-2 text-xs cursor-pointer">
                          <input
                            type="radio"
                            name="subtaskMode"
                            checked={subtaskMode === 'flat'}
                            onChange={() => setSubtaskMode('flat')}
                            className="rounded"
                          />
                          <span>Flat</span>
                        </label>
                        <label className="flex items-center gap-2 text-xs cursor-pointer">
                          <input
                            type="radio"
                            name="subtaskMode"
                            checked={subtaskMode === 'hidden'}
                            onChange={() => setSubtaskMode('hidden')}
                            className="rounded"
                          />
                          <span>Hidden</span>
                        </label>
                      </div>
                    </div>
```

**Step 3: Pass subtaskMode to child views**

Update TaskList render (around line 475):
```typescript
                <TaskList
                  onTagClick={onTagClick}
                  tasks={paginatedTasks}
                  selectMode={selectMode}
                  selectedTaskIds={selectedTaskIds}
                  onTaskSelect={toggleTaskSelection}
                  subtaskMode={subtaskMode}
                />
```

Update KanbanBoard render (around line 516):
```typescript
            <KanbanBoard
              onTagClick={onTagClick}
              tasks={tasksToDisplay}
              onAddTaskWithStatus={onAddTaskWithStatus}
              selectMode={selectMode}
              selectedTaskIds={selectedTaskIds}
              onTaskSelect={toggleTaskSelection}
              subtaskMode={subtaskMode}
            />
```

Update EisenhowerMatrix render (around line 526):
```typescript
            <EisenhowerMatrix
              onTagClick={onTagClick}
              tasks={tasksToDisplay}
              onAddTaskWithTags={(tags: string[]) => {
                setFilterString(tags.join(" "));
                setShowCreateTaskWindow(true);
              }}
              selectMode={selectMode}
              selectedTaskIds={selectedTaskIds}
              onTaskSelect={toggleTaskSelection}
              subtaskMode={subtaskMode}
            />
```

**Step 4: Test manually**

Run: `cd zettelkasten-front && npm run start`
- Open display menu
- Verify radio buttons appear for nested/flat/hidden
- Click different options and verify they persist on reload

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tasks/TaskDesktopLayout.tsx
git commit -m "feat: add subtask display mode toggle to display menu"
```

---

## Task 3: Update TaskList to Respect Display Mode

**Files:**
- Modify: `zettelkasten-front/src/components/tasks/TaskList.tsx`

**Step 1: Import hook and add prop type**

Add to imports (around line 4):
```typescript
import { SubtaskDisplayMode } from "../../hooks/useSubtaskDisplayMode";
```

Update TaskListProps interface (around line 7):
```typescript
interface TaskListProps {
  tasks: Task[];
  onTagClick: (tag: string) => void;
  hideMatrixTags?: boolean;
  selectMode?: boolean;
  selectedTaskIds?: Set<number>;
  onTaskSelect?: (taskId: number) => void;
  subtaskMode?: SubtaskDisplayMode;
}
```

Update function signature (around line 17):
```typescript
export function TaskList({
  tasks,
  onTagClick,
  hideMatrixTags = false,
  selectMode = false,
  selectedTaskIds = new Set(),
  onTaskSelect,
  subtaskMode = 'nested',
}: TaskListProps) {
```

**Step 2: Implement mode-based rendering logic**

Replace the entire useMemo and return section (lines 25-70) with:

```typescript
  // Separate root tasks from subtasks
  const { rootTasks, subtasksByParent, allTasksWithParent } = useMemo(() => {
    const rootTasks: Task[] = [];
    const subtasksByParent: Record<number, Task[]> = {};
    const allTasksWithParent: Array<{ task: Task; parent?: Task }> = [];

    tasks.forEach((task) => {
      if (task.parent_task_id) {
        // This is a subtask
        if (!subtasksByParent[task.parent_task_id]) {
          subtasksByParent[task.parent_task_id] = [];
        }
        subtasksByParent[task.parent_task_id].push(task);

        // For flat mode, find parent task
        if (subtaskMode === 'flat') {
          const parent = tasks.find(t => t.id === task.parent_task_id);
          allTasksWithParent.push({ task, parent });
        }
      } else {
        // This is a root task
        rootTasks.push(task);
        if (subtaskMode === 'flat') {
          allTasksWithParent.push({ task });
        }
      }
    });

    return { rootTasks, subtasksByParent, allTasksWithParent };
  }, [tasks, subtaskMode]);

  // Hidden mode: only show root tasks
  if (subtaskMode === 'hidden') {
    return (
      <ul className="divide-y divide-slate-200">
        {rootTasks.map((task) => (
          <li key={task.id} className="py-1">
            <TaskListItem
              task={task}
              onTagClick={onTagClick}
              hideMatrixTags={hideMatrixTags}
              selectMode={selectMode}
              isSelected={selectedTaskIds.has(task.id)}
              onSelect={() => onTaskSelect?.(task.id)}
            />
          </li>
        ))}
      </ul>
    );
  }

  // Flat mode: show all tasks with parent badges
  if (subtaskMode === 'flat') {
    return (
      <ul className="divide-y divide-slate-200">
        {allTasksWithParent.map(({ task, parent }) => (
          <li key={task.id} className="py-1">
            <TaskListItem
              task={task}
              onTagClick={onTagClick}
              hideMatrixTags={hideMatrixTags}
              selectMode={selectMode}
              isSelected={selectedTaskIds.has(task.id)}
              onSelect={() => onTaskSelect?.(task.id)}
              parentTask={parent}
            />
          </li>
        ))}
      </ul>
    );
  }

  // Nested mode (default): current behavior
  return (
    <ul className="divide-y divide-slate-200">
      {rootTasks.map((task) => {
        const subtasks = subtasksByParent[task.id] || [];

        if (subtasks.length > 0) {
          // Render as nested group with children
          return (
            <li key={task.id} className="py-1">
              <TaskNestedGroup
                task={{ ...task, subtasks }}
                onTagClick={onTagClick}
                onTaskClick={() => {}}
                selectMode={selectMode}
                selectedTaskIds={selectedTaskIds}
                onTaskSelect={onTaskSelect}
              />
            </li>
          );
        }

        // Render as single item (no children)
        return (
          <li key={task.id} className="py-1">
            <TaskListItem
              task={task}
              onTagClick={onTagClick}
              hideMatrixTags={hideMatrixTags}
              selectMode={selectMode}
              isSelected={selectedTaskIds.has(task.id)}
              onSelect={() => onTaskSelect?.(task.id)}
            />
          </li>
        );
      })}
    </ul>
  );
```

**Step 3: Test manually**

Run: `cd zettelkasten-front && npm run start`
- Create parent task with subtasks
- Switch between nested/flat/hidden modes
- Verify:
  - Nested: subtasks indented under parent
  - Flat: all tasks shown, subtasks have parent badge
  - Hidden: only parents shown

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/tasks/TaskList.tsx
git commit -m "feat: update TaskList to respect subtask display mode"
```

---

## Task 4: Update KanbanBoard to Respect Display Mode

**Files:**
- Modify: `zettelkasten-front/src/components/tasks/KanbanBoard.tsx`

**Step 1: Import hook and add prop type**

Add to imports (around line 6):
```typescript
import { SubtaskDisplayMode } from "../../hooks/useSubtaskDisplayMode";
```

Update KanbanBoardProps interface (around line 200):
```typescript
interface KanbanBoardProps {
  tasks: Task[];
  onTagClick: (tag: string) => void;
  onAddTaskWithStatus: (status: string) => void;
  selectMode?: boolean;
  selectedTaskIds?: Set<number>;
  onTaskSelect?: (taskId: number) => void;
  subtaskMode?: SubtaskDisplayMode;
}
```

Update function signature (around line 287):
```typescript
export function KanbanBoard({ tasks, onTagClick, onAddTaskWithStatus, selectMode = false, selectedTaskIds = new Set(), onTaskSelect, subtaskMode = 'nested' }: KanbanBoardProps) {
```

**Step 2: Update task grouping logic**

Locate the `tasksByStatus` grouping (around line 340). Replace with:

```typescript
  // Separate root tasks from subtasks
  const { rootTasks, subtasksByParent } = React.useMemo(() => {
    const rootTasks: Task[] = [];
    const subtasksByParent: Record<number, Task[]> = {};

    tasks.forEach((task) => {
      if (task.parent_task_id) {
        // This is a subtask
        if (!subtasksByParent[task.parent_task_id]) {
          subtasksByParent[task.parent_task_id] = [];
        }
        subtasksByParent[task.parent_task_id].push(task);
      } else {
        // This is a root task
        rootTasks.push(task);
      }
    });

    return { rootTasks, subtasksByParent };
  }, [tasks]);

  // Group tasks by status based on display mode
  const tasksByStatus = React.useMemo(() => {
    const acc: Record<string, Task[]> = {};

    if (subtaskMode === 'flat') {
      // Include all tasks (roots and subtasks) in their own status
      tasks.forEach((task) => {
        const status = task.status || (statuses.find(s => s.is_default)?.name || "todo");
        if (!acc[status]) {
          acc[status] = [];
        }
        acc[status].push(task);
      });
    } else {
      // Nested or hidden: only root tasks
      rootTasks.forEach((task) => {
        const status = task.status || (statuses.find(s => s.is_default)?.name || "todo");
        if (!acc[status]) {
          acc[status] = [];
        }
        acc[status].push(task);
      });
    }

    return acc;
  }, [tasks, rootTasks, subtaskMode, statuses]);
```

**Step 3: Update card rendering**

Locate the columnTasks.map section (around line 425). Replace the TaskListItem section with:

```typescript
                                  <TaskListItem
                                    task={task}
                                    onTagClick={onTagClick}
                                    hideMatrixTags={false}
                                    selectMode={selectMode}
                                    isSelected={selectedTaskIds.has(task.id)}
                                    onSelect={() => onTaskSelect?.(task.id)}
                                    parentTask={subtaskMode === 'flat' && task.parent_task_id ? tasks.find(t => t.id === task.parent_task_id) : undefined}
                                  />

                                  {/* Subtask count indicator - only show in nested mode */}
                                  {subtaskMode === 'nested' && subtasksByParent[task.id] && subtasksByParent[task.id].length > 0 && (
                                    <div className="px-3 pb-1 text-xs text-gray-500">
                                      {subtasksByParent[task.id].filter(s => s.is_complete).length}/{subtasksByParent[task.id].length} subtasks
                                    </div>
                                  )}
```

**Step 4: Test manually**

Run: `cd zettelkasten-front && npm run start`
- Create parent task with subtasks, set different statuses
- Switch between modes:
  - Nested: parents shown with count, no subtask cards
  - Flat: subtasks shown as separate cards in their status column with parent badge
  - Hidden: only parent tasks shown

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tasks/KanbanBoard.tsx
git commit -m "feat: update KanbanBoard to respect subtask display mode"
```

---

## Task 5: Update EisenhowerMatrix to Respect Display Mode

**Files:**
- Modify: `zettelkasten-front/src/components/tasks/EisenhowerMatrix.tsx`

**Step 1: Import hook and add prop type**

Add to imports (around line 6):
```typescript
import { SubtaskDisplayMode } from "../../hooks/useSubtaskDisplayMode";
```

Update EisenhowerMatrixProps interface (around line 11):
```typescript
interface EisenhowerMatrixProps {
    tasks: Task[];
    onTagClick: (tag: string) => void;
    onAddTaskWithTags?: (tags: string[]) => void;
    selectMode?: boolean;
    selectedTaskIds?: Set<number>;
    onTaskSelect?: (taskId: number) => void;
    subtaskMode?: SubtaskDisplayMode;
}
```

Update function signature (around line 30):
```typescript
export function EisenhowerMatrix({ tasks, onTagClick, onAddTaskWithTags, selectMode = false, selectedTaskIds = new Set(), onTaskSelect, subtaskMode = 'nested' }: EisenhowerMatrixProps) {
```

**Step 2: Filter tasks based on mode**

Update quadrant filtering logic (around line 32-35):

```typescript
    // Filter tasks based on display mode
    const displayTasks = subtaskMode === 'hidden'
      ? tasks.filter(t => !t.parent_task_id)
      : tasks;

    const q1 = displayTasks.filter(t => getQuadrant(t) === 1);
    const q2 = displayTasks.filter(t => getQuadrant(t) === 2);
    const q3 = displayTasks.filter(t => getQuadrant(t) === 3);
    const q4 = displayTasks.filter(t => getQuadrant(t) === 4);
```

**Step 3: Pass mode to TaskList**

Update TaskList render in quadrantBox (around line 87):
```typescript
                                        <TaskList
                                          onTagClick={onTagClick}
                                          tasks={[task]}
                                          hideMatrixTags={true}
                                          selectMode={selectMode}
                                          selectedTaskIds={selectedTaskIds}
                                          onTaskSelect={onTaskSelect}
                                          subtaskMode={subtaskMode}
                                        />
```

**Step 4: Test manually**

Run: `cd zettelkasten-front && npm run start`
- Create parent/subtasks with #urgent #important tags
- Switch between modes:
  - Nested: subtasks indented under parents
  - Flat: all tasks shown independently with badges
  - Hidden: only parent tasks shown

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tasks/EisenhowerMatrix.tsx
git commit -m "feat: update EisenhowerMatrix to respect subtask display mode"
```

---

## Task 6: Add Parent Badge to TaskListItem

**Files:**
- Modify: `zettelkasten-front/src/components/tasks/TaskListItem.tsx`

**Step 1: Add parentTask prop to interface**

Locate TaskListItemProps interface (around line 20). Add:
```typescript
  parentTask?: Task;
```

Update function signature (around line 50):
```typescript
export function TaskListItem({
  task,
  onTagClick,
  hideMatrixTags = false,
  selectMode = false,
  isSelected = false,
  onSelect,
  parentTask,
}: TaskListItemProps) {
```

**Step 2: Import dialog state hook**

Add to imports (around line 10):
```typescript
import { useDialogState } from "../../contexts/DialogStateContext";
```

Add inside function (around line 55):
```typescript
  const { setSelectedTaskId, setIsTaskDialogOpen } = useDialogState();
```

**Step 3: Add parent badge UI**

Locate the main container div (around line 80). Add parent badge before the main content:

```typescript
      {/* Parent task badge */}
      {parentTask && (
        <div className="mb-1">
          <button
            onClick={(e) => {
              e.stopPropagation();
              setSelectedTaskId(parentTask.id);
              setIsTaskDialogOpen(true);
            }}
            className="inline-flex items-center gap-1 px-2 py-0.5 text-xs text-gray-600 bg-gray-100 hover:bg-gray-200 rounded-full transition-colors"
            title={`Parent: ${parentTask.title}`}
          >
            <span>↳</span>
            <span className="max-w-[150px] truncate">{parentTask.title}</span>
          </button>
        </div>
      )}
```

**Step 4: Test manually**

Run: `cd zettelkasten-front && npm run start`
- Switch to flat mode
- Verify parent badge appears above subtask title
- Click badge to open parent task dialog
- Verify badge truncates long parent titles

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tasks/TaskListItem.tsx
git commit -m "feat: add clickable parent badge to TaskListItem for flat mode"
```

---

## Task 7: Integration Testing and Verification

**Step 1: Run full test suite**

Run: `cd zettelkasten-front && npm test`
Expected: All tests pass

**Step 2: Manual end-to-end testing**

Run: `cd zettelkasten-front && npm run start`

Test scenarios:
1. **List view:**
   - Nested: Subtasks indented, progress indicator shown
   - Flat: All tasks shown, parent badge on subtasks
   - Hidden: Only parent tasks visible

2. **Kanban view:**
   - Nested: Parent cards with count, no subtask cards
   - Flat: Subtask cards in their own status with badge
   - Hidden: Only parent cards

3. **Eisenhower view:**
   - Nested: Subtasks grouped under parents
   - Flat: All tasks shown with badges
   - Hidden: Only parent tasks

4. **Persistence:**
   - Set mode to "flat"
   - Reload page
   - Verify mode is still "flat"

5. **Parent badge interaction:**
   - Click parent badge in flat mode
   - Verify parent task dialog opens

**Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve integration issues with subtask display mode"
```

---

## Task 8: Update bd Issue Status

**Step 1: Mark issue as complete**

Run: `bd close Zettelgarden-4g1 --reason "Completed implementation of subtask display mode toggle"`
Expected: Issue closed successfully

**Step 2: Push all commits**

Run: `git push`
Expected: All commits pushed to remote

---

## Summary

This plan implements a global subtask display mode toggle with 8 tasks:

1. Create persistent hook with tests
2. Add UI toggle to display menu
3. Update TaskList for all three modes
4. Update KanbanBoard with flat mode card rendering
5. Update EisenhowerMatrix filtering
6. Add clickable parent badge component
7. Integration testing and verification
8. Close issue and push

**Estimated time:** 2-3 hours
**Risk areas:** None identified - straightforward feature addition with clear separation of concerns
