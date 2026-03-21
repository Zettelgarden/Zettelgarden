# Subtasks Feature Specification

## Overview

Enable hierarchical task relationships where a parent task can have child tasks (subtasks). Subtasks are full-featured tasks with their own dates, priorities, tags, and dependencies - they simply have a parent reference.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Nesting depth | Single level only | Keeps UI manageable, prevents infinite nesting |
| Parent completion | Blocked until children done | Enforces completion discipline |
| Property inheritance | Inherit on creation, then independent | Convenience without coupling |
| Default view | Nested under parent | Clear hierarchy visualization |

---

## User Stories

### US-1: Create Subtask
**As a** user
**I want to** add subtasks to a task
**So that** I can break down complex work into smaller pieces

**Acceptance Criteria:**
```
Given I am viewing a task
When I click "Add Subtask"
Then a new subtask is created with:
  - Title (empty, focused for input)
  - Inherited tags from parent
  - Inherited priority from parent (if set)
  - No dates inherited
  - parent_task_id set to current task
And the subtask appears nested under the parent in list view
And the parent shows a progress indicator (0/1)
```

### US-2: Complete Subtask
**As a** user
**I want to** mark individual subtasks as complete
**So that** I can track progress on the parent task

**Acceptance Criteria:**
```
Given a task with 3 subtasks (0 complete)
When I mark the first subtask complete
Then the parent shows progress (1/3)
And the subtask is crossed out
And the subtask's status changes to the "complete" status
```

### US-3: Block Parent Completion
**As a** user
**I want to** be prevented from completing a task with incomplete subtasks
**So that** I don't accidentally close work that isn't done

**Acceptance Criteria:**
```
Given a task with 2 incomplete subtasks
When I try to mark the parent task complete
Then I see a warning: "Complete all subtasks first (0/2 done)"
And the parent task remains incomplete
When I complete both subtasks
Then I can mark the parent complete
```

### US-4: View Subtasks Nested
**As a** user
**I want to** see subtasks indented under their parent
**So that** I understand the task hierarchy

**Acceptance Criteria:**
```
Given a task "Build feature" with subtasks "Design API" and "Write tests"
When I view the task list
Then I see:
  [ ] Build feature (0/2)
    [ ] Design API
    [ ] Write tests
And subtasks are indented with a visual connector
And clicking the parent collapses/expands children
```

### US-5: Subtask Independence
**As a** user
**I want to** give a subtask its own due date and priority
**So that** I can schedule pieces independently

**Acceptance Criteria:**
```
Given a parent task due Friday with priority B
When I create a subtask
Then the subtask inherits priority B
And the subtask has no due date
When I change the subtask due date to Wednesday
Then the subtask appears in Wednesday's view
And the parent still shows due Friday
When I change the parent priority to A
Then the subtask remains priority B (independent after creation)
```

### US-6: Delete Parent
**As a** user
**I want to** delete a parent task and all its subtasks
**So that** I can remove entire task groups at once

**Acceptance Criteria:**
```
Given a task with 3 subtasks
When I delete the parent task
Then all 3 subtasks are also deleted (CASCADE)
And I see a confirmation: "Delete task and 3 subtasks?"
```

### US-7: Convert Task to Subtask
**As a** user
**I want to** make an existing task a subtask of another task
**So that** I can reorganize my task hierarchy

**Acceptance Criteria:**
```
Given two independent tasks A and B
When I edit task B and set "Parent task" to A
Then task B becomes a subtask of A
And task B appears nested under A in list view
And task A shows progress indicator
```

### US-8: Promote Subtask to Task
**As a** user
**I want to** remove a subtask from its parent
**So that** it becomes an independent task

**Acceptance Criteria:**
```
Given task A with subtask B
When I edit subtask B and clear the parent
Then task B becomes independent
And task B no longer appears nested under A
And task A's progress indicator updates
```

---

## Data Model

### Schema Migration

```sql
-- Add parent_task_id column to tasks table
ALTER TABLE tasks ADD COLUMN parent_task_id INT REFERENCES tasks(id) ON DELETE CASCADE;

-- Index for fetching children of a parent
CREATE INDEX idx_tasks_parent_id ON tasks(parent_task_id);

-- Index for fetching root tasks (no parent) efficiently
CREATE INDEX idx_tasks_is_root ON tasks((parent_task_id IS NULL)) WHERE is_deleted = FALSE;
```

### Model Changes

**Backend (Go) - models/tasks.go:**
```go
type Task struct {
    // ... existing fields ...
    ParentTaskID *int `json:"parent_task_id,omitempty"`
    
    // Populated on fetch (not stored in DB)
    Subtasks    []Task `json:"subtasks,omitempty"`
    ParentTitle string `json:"parent_title,omitempty"` // For display purposes
}
```

**Frontend (TypeScript) - models/Task.ts:**
```typescript
export interface Task {
  // ... existing fields ...
  parent_task_id: number | null;
  subtasks?: Task[];           // Populated on fetch
  parent_title?: string;       // For display in child rows
}
```

### Constraints

1. **Single level nesting** - Enforced in service layer:
   ```go
   // When setting parent_task_id, verify the parent doesn't have a parent
   if newParent.HasParent() {
       return errors.New("cannot nest more than one level deep")
   }
   ```

2. **No self-reference** - Database constraint or service validation:
   ```go
   if task.ID == task.ParentTaskID {
       return errors.New("task cannot be its own parent")
   }
   ```

---

## API Design

### Endpoints

#### Create Subtask
```
POST /api/tasks/:id/subtasks
```
Creates a new task as a child of the specified parent.

**Request Body:**
```json
{
  "title": "Design API schema",
  "description": "",
  "priority": "B",
  "tags": ["#backend", "#feature-x"]
}
```

**Behavior:**
- `parent_task_id` is set to the URL param `:id`
- Inherits parent's tags and priority (unless overridden in request)
- Returns the created subtask

**Response:** `201 Created` with full `Task` object

---

#### Set Parent (Convert to Subtask)
```
PATCH /api/tasks/:id/parent
```
Sets or clears the parent of an existing task.

**Request Body:**
```json
{
  "parent_task_id": 123
}
```
or to clear:
```json
{
  "parent_task_id": null
}
```

**Behavior:**
- Validates single-level nesting (parent can't have a parent)
- Validates no self-reference
- Updates task's `parent_task_id`

**Response:** `200 OK` with updated `Task` object

---

#### Get Subtasks
```
GET /api/tasks/:id/subtasks
```
Returns all subtasks for a parent task.

**Response:**
```json
{
  "subtasks": [
    { "id": 2, "title": "...", "is_complete": false, ... },
    { "id": 3, "title": "...", "is_complete": true, ... }
  ],
  "total": 2,
  "complete_count": 1
}
```

---

#### Complete Parent (with Validation)
```
PATCH /api/tasks/:id/complete
```
Existing endpoint, enhanced with subtask validation.

**Behavior:**
- Check if task has incomplete subtasks
- If incomplete subtasks exist, return error:
  ```json
  {
    "error": "incomplete_subtasks",
    "message": "Complete all subtasks first",
    "incomplete_count": 2,
    "total_subtasks": 5
  }
  ```
- Force override with query param `?force=true` (optional, for power users)

---

## UI/UX Design

### Task List View - Nested Display

```
┌─────────────────────────────────────────────────────────────┐
│ [ ] Build authentication feature (1/3)              #feature │
│   ├─ [x] Design OAuth flow                       #backend   │
│   ├─ [ ] Implement login UI                      #frontend  │
│   └─ [ ] Write integration tests                 #testing   │
├─────────────────────────────────────────────────────────────┤
│ [ ] Review pull requests                                  │
├─────────────────────────────────────────────────────────────┤
│ [x] Deploy to staging                                      │
└─────────────────────────────────────────────────────────────┘
```

**Interactions:**
- Click parent row → expands/collapses children
- Progress indicator `(1/3)` shows completion status
- Visual connector lines show hierarchy
- Children indented with left padding

### Task Detail Dialog - Subtasks Section

```
┌─────────────────────────────────────────────────────────────┐
│ Edit Task                                                   │
├─────────────────────────────────────────────────────────────┤
│ Title: [Build authentication feature________________]        │
│                                                             │
│ Subtasks (1/3)                              [+ Add Subtask] │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ [x] Design OAuth flow                            [×]    │ │
│ │ [ ] Implement login UI                           [×]    │ │
│ │ [ ] Write integration tests                      [×]    │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ Or, link existing task as subtask:                          │
│ [Search tasks...___________________________] [Link]         │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                          [Cancel] [Save]    │
└─────────────────────────────────────────────────────────────┘
```

### Kanban View - Subtask Cards

**Option A: Show children in same column with indentation**
```
┌──────────────────────┐
│ TO DO                │
├──────────────────────┤
│ ┌──────────────────┐ │
│ │ Build feature    │ │
│ │ (1/3) #feature   │ │
│ └──────────────────┘ │
│   ┌────────────────┐ │
│   │ ↳ Design API   │ │
│   └────────────────┘ │
│   ┌────────────────┐ │
│   │ ↳ Write tests  │ │
│   └────────────────┘ │
└──────────────────────┘
```

**Option B: Collapse children, show on hover/click**
- Parent card shows `(1/3)` badge
- Click to expand inline or open dialog

**Recommendation:** Option A for visibility, with collapse toggle

### Completion Blocked - Warning Dialog

```
┌─────────────────────────────────────────────────────────────┐
│ ⚠️ Incomplete Subtasks                                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ "Build authentication feature" has 2 incomplete subtasks:   │
│                                                             │
│   • Implement login UI                                      │
│   • Write integration tests                                 │
│                                                             │
│ Complete these first, or mark all subtasks complete.        │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│              [Cancel]  [Force Complete Anyway]              │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Breakdown

### New Components

| Component | Purpose |
|-----------|---------|
| `TaskSubtasksSection.tsx` | Subtasks list in task dialog, with add/edit/delete |
| `TaskSubtaskItem.tsx` | Single subtask row (checkbox, title, delete button) |
| `TaskNestedGroup.tsx` | Renders parent + children in list view with collapse |
| `TaskCompletionWarningDialog.tsx` | Modal when trying to complete parent with incomplete children |
| `TaskParentSelector.tsx` | Dropdown to select/clear parent task |

### Modified Components

| Component | Changes |
|-----------|---------|
| `TaskListItem.tsx` | Add progress indicator, render nested children |
| `TaskList.tsx` | Group subtasks under parents, handle expand/collapse |
| `TaskDialog.tsx` | Add `TaskSubtasksSection` |
| `TaskForm.tsx` | Add parent task selector field |
| `KanbanBoard.tsx` | Show subtasks nested under parent cards |
| `useTaskFiltering.ts` | Handle parent/child relationships in filtering |

---

## Business Logic

### Inheritance Rules (On Creation Only)

When creating a subtask via `POST /api/tasks/:id/subtasks`:

```go
func CreateSubtask(parentTask *Task, input CreateTaskInput) *Task {
    subtask := &Task{
        ParentTaskID: &parentTask.ID,
        Title:        input.Title,
        Description:  input.Description,
        Priority:     input.Priority,      // Use provided
    }
    
    // Inherit from parent if not provided
    if subtask.Priority == nil && parentTask.Priority != nil {
        subtask.Priority = parentTask.Priority
    }
    
    // Inherit tags from parent
    for _, tag := range parentTask.Tags {
        subtask.Tags = append(subtask.Tags, tag)
    }
    
    return subtask
}
```

### Completion Validation

```go
func (s *Service) CompleteTask(taskID int, force bool) error {
    task := s.GetTask(taskID)
    
    if !force {
        incompleteCount := s.CountIncompleteSubtasks(taskID)
        if incompleteCount > 0 {
            return &SubtaskIncompleteError{
                IncompleteCount: incompleteCount,
                TotalSubtasks:   len(task.Subtasks),
            }
        }
    }
    
    task.IsComplete = true
    task.CompletedAt = time.Now()
    return s.SaveTask(task)
}
```

### Single-Level Nesting Enforcement

```go
func (s *Service) SetParentTask(taskID, parentID int) error {
    if taskID == parentID {
        return errors.New("task cannot be its own parent")
    }
    
    parent := s.GetTask(parentID)
    if parent.ParentTaskID != nil {
        return errors.New("cannot nest more than one level deep")
    }
    
    // Also check if task has children (can't make it a child if it's a parent)
    children := s.GetSubtasks(taskID)
    if len(children) > 0 {
        return errors.New("cannot make a parent task into a subtask")
    }
    
    task := s.GetTask(taskID)
    task.ParentTaskID = &parentID
    return s.SaveTask(task)
}
```

---

## Edge Cases

### EC-1: Parent Deleted
- **Behavior:** All children deleted via CASCADE
- **UI:** Show confirmation dialog: "Delete 'Build feature' and its 3 subtasks?"

### EC-2: Orphaned Subtask (Data Corruption)
- **Behavior:** Task with `parent_task_id` pointing to non-existent/deleted parent
- **Detection:** Periodic cleanup job or foreign key constraint handles it
- **UI:** Treat as root task (parent_task_id effectively null)

### EC-3: Circular Reference (Attempted)
- **Behavior:** Blocked by service layer validation
- **Prevention:** Check if new parent is a descendant (impossible with single-level, but validate anyway)

### EC-4: Make Parent a Subtask
- **Scenario:** Task A has children. User tries to make Task A a child of Task B.
- **Behavior:** Blocked - "This task has subtasks and cannot be made into a subtask"
- **Workaround:** User must first promote all children, or delete them

### EC-5: Subtask in Different Status
- **Scenario:** Parent in "In Progress", subtask moved to "Done"
- **Behavior:** Allowed - subtasks are independent tasks with their own status
- **UI:** Subtask appears in Done column, parent still in In Progress with (1/2) indicator

### EC-6: Filter Hides Parent, Shows Child
- **Scenario:** Filter `status:done` hides parent, but subtask is done
- **Behavior:** Subtask appears as root-level item in filtered view
- **Note:** Filtered views flatten hierarchy for relevance

---

## Testing Strategy

### Backend Tests

```go
// services/tasks_subtasks_test.go

func TestCreateSubtask_InheritsPriority(t *testing.T) {
    parent := createTaskWithPriority("A")
    subtask := CreateSubtask(parent, CreateTaskInput{Title: "Child"})
    assert.Equal(t, "A", *subtask.Priority)
}

func TestCreateSubtask_DoesNotInheritPriorityWhenProvided(t *testing.T) {
    parent := createTaskWithPriority("A")
    subtask := CreateSubtask(parent, CreateTaskInput{Title: "Child", Priority: "C"})
    assert.Equal(t, "C", *subtask.Priority)
}

func TestCompleteParent_BlockedByIncompleteSubtasks(t *testing.T) {
    parent := createTaskWithSubtasks(2) // 2 incomplete
    err := service.CompleteTask(parent.ID, false)
    assert.Error(t, err)
    assert.IsType(t, &SubtaskIncompleteError{}, err)
}

func TestCompleteParent_AllowedWhenSubtasksComplete(t *testing.T) {
    parent := createTaskWithSubtasks(2)
    completeAllSubtasks(parent)
    err := service.CompleteTask(parent.ID, false)
    assert.NoError(t, err)
}

func TestCompleteParent_ForceBypassesValidation(t *testing.T) {
    parent := createTaskWithSubtasks(2) // 2 incomplete
    err := service.CompleteTask(parent.ID, true) // force=true
    assert.NoError(t, err)
}

func TestSetParent_SingleLevelOnly(t *testing.T) {
    grandparent := createTask()
    parent := createTaskWithParent(grandparent.ID)
    child := createTask()
    
    err := service.SetParentTask(child.ID, parent.ID)
    assert.Error(t, err) // parent already has a parent
}

func TestSetParent_CannotNestParentWithChildren(t *testing.T) {
    parent := createTaskWithSubtasks(2)
    newParent := createTask()
    
    err := service.SetParentTask(parent.ID, newParent.ID)
    assert.Error(t, err) // parent has children
}

func TestDeleteParent_DeletesChildren(t *testing.T) {
    parent := createTaskWithSubtasks(3)
    service.DeleteTask(parent.ID)
    
    children := service.GetSubtasks(parent.ID)
    assert.Empty(t, children) // CASCADE deleted
}
```

### Frontend Tests

```typescript
// components/tasks/TaskSubtasksSection.test.tsx

describe('TaskSubtasksSection', () => {
  it('renders subtasks with checkboxes', () => {
    const task = { id: 1, subtasks: [
      { id: 2, title: 'Subtask 1', is_complete: false },
      { id: 3, title: 'Subtask 2', is_complete: true },
    ]};
    render(<TaskSubtasksSection task={task} />);
    
    expect(screen.getByText('Subtask 1')).not.toBeChecked();
    expect(screen.getByText('Subtask 2')).toBeChecked();
  });

  it('shows progress indicator (1/2)', () => {
    const task = { id: 1, subtasks: [
      { id: 2, title: 'A', is_complete: true },
      { id: 3, title: 'B', is_complete: false },
    ]};
    render(<TaskSubtasksSection task={task} />);
    
    expect(screen.getByText('(1/2)')).toBeInTheDocument();
  });

  it('calls onCreateSubtask when add button clicked', async () => {
    const onCreateSubtask = jest.fn();
    render(<TaskSubtasksSection task={task} onCreateSubtask={onCreateSubtask} />);
    
    await userEvent.click(screen.getByText('+ Add Subtask'));
    expect(onCreateSubtask).toHaveBeenCalled();
  });
});

// components/tasks/TaskNestedGroup.test.tsx

describe('TaskNestedGroup', () => {
  it('renders children nested under parent', () => {
    const parent = { id: 1, title: 'Parent', subtasks: [
      { id: 2, title: 'Child 1' },
      { id: 3, title: 'Child 2' },
    ]};
    render(<TaskNestedGroup task={parent} />);
    
    expect(screen.getByText('Parent')).toBeInTheDocument();
    expect(screen.getByText('Child 1')).toBeInTheDocument();
    expect(screen.getByText('Child 2')).toBeInTheDocument();
  });

  it('collapses/expands children on click', async () => {
    const parent = { id: 1, title: 'Parent', subtasks: [
      { id: 2, title: 'Child' },
    ]};
    render(<TaskNestedGroup task={parent} />);
    
    // Initially expanded
    expect(screen.getByText('Child')).toBeVisible();
    
    // Click to collapse
    await userEvent.click(screen.getByText('Parent'));
    expect(screen.queryByText('Child')).not.toBeVisible();
  });
});
```

### Integration Tests

```typescript
// e2e/subtasks.spec.ts

test('create subtask workflow', async ({ page }) => {
  await page.goto('/app/tasks');
  
  // Create parent task
  await page.click('[data-testid="add-task"]');
  await page.fill('[data-testid="task-title"]', 'Parent task');
  await page.click('[data-testid="save-task"]');
  
  // Add subtask
  await page.click('[data-testid="task-1"]'); // Open parent
  await page.click('[data-testid="add-subtask"]');
  await page.fill('[data-testid="subtask-title"]', 'First subtask');
  await page.press('[data-testid="subtask-title"]', 'Enter');
  
  // Verify progress indicator
  await expect(page.locator('[data-testid="task-1"]')).toContainText('(0/1)');
  
  // Complete subtask
  await page.click('[data-testid="subtask-checkbox"]');
  await expect(page.locator('[data-testid="task-1"]')).toContainText('(1/1)');
});

test('blocked completion shows warning', async ({ page }) => {
  await page.goto('/app/tasks');
  
  // Setup: parent with incomplete subtask
  const parent = await createTaskWithSubtask('Parent', 'Child');
  
  // Try to complete parent
  await page.click(`[data-testid="task-${parent.id}"] [data-testid="complete-button"]`);
  
  // See warning
  await expect(page.locator('[data-testid="completion-warning"]')).toBeVisible();
  await expect(page.locator('text=Complete all subtasks first')).toBeVisible();
});
```

---

## Migration & Deployment

### Phase 1: Database Migration
```bash
# Create migration file
go-backend/migrations/XXX_add_parent_task_id.sql
```

```sql
-- Up
ALTER TABLE tasks ADD COLUMN parent_task_id INT REFERENCES tasks(id) ON DELETE CASCADE;
CREATE INDEX idx_tasks_parent_id ON tasks(parent_task_id);

-- Down
DROP INDEX idx_tasks_parent_id;
ALTER TABLE tasks DROP COLUMN parent_task_id;
```

### Phase 2: Backend API
1. Add `ParentTaskID` to Task model
2. Add `Subtasks []Task` to Task model (populated on fetch)
3. Create migration file
4. Add API endpoints:
   - `POST /api/tasks/:id/subtasks`
   - `PATCH /api/tasks/:id/parent`
   - `GET /api/tasks/:id/subtasks`
5. Update `GET /api/tasks` to include subtasks
6. Add completion validation logic

### Phase 3: Frontend Components
1. Add `parent_task_id` and `subtasks` to Task interface
2. Create `TaskSubtasksSection` component
3. Create `TaskNestedGroup` component
4. Update `TaskList` to group by parent
5. Update `TaskDialog` to include subtasks section
6. Update `KanbanBoard` to show nested children
7. Add completion warning dialog

### Phase 4: Testing & QA
1. Backend unit tests
2. Frontend component tests
3. E2E tests
4. Manual QA on all views

### Rollback Plan
1. Revert frontend deploy
2. Remove new API routes
3. Run down migration: `ALTER TABLE tasks DROP COLUMN parent_task_id;`

---

## Future Enhancements (Out of Scope)

- **Drag to reorder subtasks** - Requires `position` field
- **Batch convert tasks to subtasks** - Bulk operation
- **Subtask templates** - Pre-defined checklist templates
- **Subtask time estimates** - Roll up to parent total
- **Recurring subtasks** - Re-create when parent recurs
