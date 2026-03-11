# Subtask Display Mode Toggle Design

**Date:** 2026-03-10
**Issue:** Zettelgarden-4g1
**Status:** Approved

## Overview

Allow users to control how subtasks are displayed across all task views with a single global setting.

## Background

Currently subtasks are always shown nested under parent tasks. Users may want different views:
- **Nested:** Subtasks indented under parent (current default)
- **Flat:** Subtasks as independent items with parent indicator
- **Hidden:** Only show parent tasks, hide subtasks

## Design Decisions

### 1. Global Setting

- **Decision:** Single global setting for all views (not per-view)
- **Rationale:** Simpler UX, consistent experience across views
- **Storage:** localStorage key `subtaskDisplayMode`
- **Default:** `'nested'`

### 2. UI Control Location

- **Decision:** Add to existing display menu in TaskDesktopLayout
- **Rationale:** Consistent with other display settings, keeps toolbar clean
- **Placement:** After "Show Completed Tasks" checkbox, before "Sort By" section

### 3. Kanban Flat Mode

- **Decision:** Subtasks use their own status field for column placement
- **Rationale:** Subtasks inherit parent properties on creation but can be modified independently
- **Behavior:** Subtasks appear in their own status column, not parent's column

### 4. Parent Indicator

- **Decision:** Clickable badge showing parent task name
- **Rationale:** Clear visual indicator with useful interaction
- **Behavior:** Badge opens parent task in dialog when clicked

## Implementation Details

### 1. Hook: `useSubtaskDisplayMode`

**File:** `zettelkasten-front/src/hooks/useSubtaskDisplayMode.ts` (new)

```typescript
type SubtaskDisplayMode = 'nested' | 'flat' | 'hidden';

function useSubtaskDisplayMode(): {
  subtaskMode: SubtaskDisplayMode;
  setSubtaskMode: (mode: SubtaskDisplayMode) => void;
}
```

- Single global setting stored in localStorage
- Defaults to `'nested'`
- Persists across page reloads and view switches

### 2. UI Toggle in Display Menu

**File:** `zettelkasten-front/src/components/tasks/TaskDesktopLayout.tsx`

**UI:**
- Radio button group with 3 options:
  - ○ Nested (subtasks indented under parent)
  - ○ Flat (subtasks as independent items)
  - ○ Hidden (only show parent tasks)
- Active mode shown with filled radio button
- Compact layout using existing menu styling

### 3. TaskList Updates

**File:** `zettelkasten-front/src/components/tasks/TaskList.tsx`

**Behavior by Mode:**
- **Nested:** Current behavior with TaskNestedGroup
- **Flat:** Render all tasks as individual TaskListItem components with parent badge for subtasks
- **Hidden:** Filter out tasks with `parent_task_id`, only render roots

**Props Enhancement:**
- Pass `parentTask` prop to TaskListItem for subtasks in flat mode

### 4. KanbanBoard Updates

**File:** `zettelkasten-front/src/components/tasks/KanbanBoard.tsx`

**Behavior by Mode:**
- **Nested:** Current behavior - parent tasks only with subtask count indicator
- **Flat:** Include subtasks in status grouping, render as separate cards with parent badge
- **Hidden:** Only show root tasks in columns

### 5. EisenhowerMatrix Updates

**File:** `zettelkasten-front/src/components/tasks/EisenhowerMatrix.tsx`

**Behavior by Mode:**
- **Nested:** Current behavior via TaskList
- **Flat:** Pass all tasks (roots + subtasks), TaskList handles flat rendering
- **Hidden:** Filter out subtasks before passing to quadrants

**Note:** Most logic handled by TaskList since EisenhowerMatrix uses it internally.

### 6. TaskListItem Parent Badge

**File:** `zettelkasten-front/src/components/tasks/TaskListItem.tsx`

**Changes:**
- Add optional `parentTask?: Task` prop
- Display clickable badge when prop is present:
  - Format: "↳ Parent Task Name"
  - Style: Small, muted, rounded chip
  - Placement: Above task title
  - Action: Opens parent in TaskDialog on click

## Files to Modify

1. `zettelkasten-front/src/hooks/useSubtaskDisplayMode.ts` (new)
2. `zettelkasten-front/src/components/tasks/TaskList.tsx`
3. `zettelkasten-front/src/components/tasks/KanbanBoard.tsx`
4. `zettelkasten-front/src/components/tasks/EisenhowerMatrix.tsx`
5. `zettelkasten-front/src/components/tasks/TaskDesktopLayout.tsx`
6. `zettelkasten-front/src/components/tasks/TaskListItem.tsx`

## Acceptance Criteria

- ✅ Toggle visible and functional in display menu
- ✅ Mode persists across page reloads
- ✅ Default is 'nested' for all views
- ✅ Subtasks display correctly in each mode across all three views
- ✅ Parent badge is clickable and opens parent task dialog
- ✅ No performance regression with large task lists

## Future Enhancements (Out of Scope)

- Per-view display mode settings
- Bulk operations on subtasks in flat mode
- Visual distinction for completed subtasks in flat mode
