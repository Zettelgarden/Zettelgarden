# Mobile Task List Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign task list items for mobile touch with reduced inline actions, proper touch targets, and long-press interaction pattern.

**Architecture:** Responsive CSS approach using Tailwind's `md:` prefix to provide different layouts for mobile (<768px) vs desktop (≥768px). New components for long-press action sheet, tag dots, and tappable info icons. Desktop behavior remains unchanged.

**Tech Stack:** React, TypeScript, Tailwind CSS, Vitest (testing), existing TaskContext/DialogStateContext patterns

---

## Task 1: Create LongPressActionSheet component

**Files:**
- Create: `zettelkasten-front/src/components/tasks/LongPressActionSheet.tsx`
- Create: `zettelkasten-front/src/components/tasks/LongPressActionSheet.test.tsx`

**Step 1: Write the failing test**

Create `zettelkasten-front/src/components/tasks/LongPressActionSheet.test.tsx`:

```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import { LongPressActionSheet } from './LongPressActionSheet';

describe('LongPressActionSheet', () => {
  const mockOnAction = jest.fn();
  const mockOnClose = jest.fn();

  it('renders action sheet when visible', () => {
    render(
      <LongPressActionSheet
        visible={true}
        onAction={mockOnAction}
        onClose={mockOnClose}
        task={{
          id: 1,
          title: 'Test Task',
          is_complete: false,
          card: null
        }}
      />
    );
    expect(screen.getByText('Toggle Complete')).toBeInTheDocument();
  });

  it('does not render when not visible', () => {
    render(
      <LongPressActionSheet
        visible={false}
        onAction={mockOnAction}
        onClose={mockOnClose}
        task={{
          id: 1,
          title: 'Test Task',
          is_complete: false,
          card: null
        }}
      />
    );
    expect(screen.queryByText('Toggle Complete')).not.toBeInTheDocument();
  });

  it('calls onAction with toggle action when Toggle Complete is clicked', () => {
    render(
      <LongPressActionSheet
        visible={true}
        onAction={mockOnAction}
        onClose={mockOnClose}
        task={{
          id: 1,
          title: 'Test Task',
          is_complete: false,
          card: null
        }}
      />
    );
    fireEvent.click(screen.getByText('Toggle Complete'));
    expect(mockOnAction).toHaveBeenCalledWith('toggle');
  });

  it('calls onClose when backdrop is clicked', () => {
    render(
      <LongPressActionSheet
        visible={true}
        onAction={mockOnAction}
        onClose={mockOnClose}
        task={{
          id: 1,
          title: 'Test Task',
          is_complete: false,
          card: null
        }}
      />
    );
    const backdrop = screen.getByTestId('action-sheet-backdrop');
    fireEvent.click(backdrop);
    expect(mockOnClose).toHaveBeenCalled();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- LongPressActionSheet.test.tsx`
Expected: FAIL with "Cannot find module './LongPressActionSheet'"

**Step 3: Write minimal implementation**

Create `zettelkasten-front/src/components/tasks/LongPressActionSheet.tsx`:

```typescript
import React from 'react';
import { Task } from '../../models/Task';
import { TaskClosedIcon } from '../../assets/icons/TaskClosedIcon';
import { TaskOpenIcon } from '../../assets/icons/TaskOpenIcon';

interface LongPressActionSheetProps {
  visible: boolean;
  onAction: (action: 'toggle' | 'delete' | 'edit' | 'linkCard' | 'copyTitle' | 'viewDetails') => void;
  onClose: () => void;
  task: Task;
}

export function LongPressActionSheet({ visible, onAction, onClose, task }: LongPressActionSheetProps) {
  if (!visible) return null;

  const handleAction = (action: LongPressActionSheetProps['onAction'] extends (infer A) ? A : never) => {
    onAction(action);
    onClose();
  };

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div
      data-testid="action-sheet-backdrop"
      className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-end md:items-center justify-center"
      onClick={handleBackdropClick}
    >
      <div className="bg-white rounded-t-2xl md:rounded-2xl w-full max-w-md md:max-w-sm p-4">
        <h3 className="text-lg font-semibold mb-4 px-2">{task.title}</h3>

        <div className="space-y-2">
          <button
            onClick={() => handleAction('toggle')}
            className="w-full flex items-center gap-4 px-4 py-3 rounded-lg hover:bg-gray-100 active:bg-gray-200 transition-colors text-left min-h-[48px]"
          >
            <span className="text-2xl w-8 flex justify-center">
              {task.is_complete ? <TaskOpenIcon /> : <TaskClosedIcon />}
            </span>
            <span className="text-base font-medium">{task.is_complete ? 'Mark Incomplete' : 'Mark Complete'}</span>
          </button>

          <button
            onClick={() => handleAction('edit')}
            className="w-full flex items-center gap-4 px-4 py-3 rounded-lg hover:bg-gray-100 active:bg-gray-200 transition-colors text-left min-h-[48px]"
          >
            <span className="text-2xl w-8 flex justify-center">✏️</span>
            <span className="text-base font-medium">Edit Task</span>
          </button>

          <button
            onClick={() => handleAction('delete')}
            className="w-full flex items-center gap-4 px-4 py-3 rounded-lg hover:bg-red-50 active:bg-red-100 transition-colors text-left min-h-[48px] text-red-600"
          >
            <span className="text-2xl w-8 flex justify-center">✕</span>
            <span className="text-base font-medium">Delete Task</span>
          </button>

          <div className="border-t border-gray-200 my-2"></div>

          <button
            onClick={() => handleAction('copyTitle')}
            className="w-full flex items-center gap-4 px-4 py-3 rounded-lg hover:bg-gray-100 active:bg-gray-200 transition-colors text-left min-h-[48px]"
          >
            <span className="text-2xl w-8 flex justify-center">📋</span>
            <span className="text-base font-medium">Copy Title</span>
          </button>

          <button
            onClick={() => handleAction('viewDetails')}
            className="w-full flex items-center gap-4 px-4 py-3 rounded-lg hover:bg-gray-100 active:bg-gray-200 transition-colors text-left min-h-[48px]"
          >
            <span className="text-2xl w-8 flex justify-center">ℹ️</span>
            <span className="text-base font-medium">View Details</span>
          </button>
        </div>
      </div>
    </div>
  );
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- LongPressActionSheet.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tasks/LongPressActionSheet.tsx zettelkasten-front/src/components/tasks/LongPressActionSheet.test.tsx
git commit -m "feat: add LongPressActionSheet component for mobile task actions"
```

---

## Task 2: Create TappableInfoIcon component

**Files:**
- Create: `zettelkasten-front/src/components/tasks/TappableInfoIcon.tsx`
- Create: `zettelkasten-front/src/components/tasks/TappableInfoIcon.test.tsx`

**Step 1: Write the failing test**

Create `zettelkasten-front/src/components/tasks/TappableInfoIcon.test.tsx`:

```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TappableInfoIcon } from './TappableInfoIcon';

// Mock the toast
jest.mock('react-toastify', () => ({
  toast: {
    info: jest.fn(),
  },
}));

import { toast } from 'react-toastify';

describe('TappableInfoIcon', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders the icon with provided symbol', () => {
    render(
      <TappableInfoIcon
        symbol="🔔"
        tooltipText="Reminder set"
        className="text-blue-500"
      />
    );
    expect(screen.getByText('🔔')).toBeInTheDocument();
  });

  it('shows toast when tapped', async () => {
    const user = userEvent.setup();
    render(
      <TappableInfoIcon
        symbol="🚧"
        tooltipText="Blocked by other task"
        className="text-orange-500"
      />
    );

    await user.click(screen.getByText('🚧'));
    expect(toast.info).toHaveBeenCalledWith('Blocked by other task');
  });

  it('applies custom className', () => {
    const { container } = render(
      <TappableInfoIcon
        symbol="🔔"
        tooltipText="Reminder"
        className="text-blue-500"
      />
    );
    const icon = container.firstChild as HTMLElement;
    expect(icon).toHaveClass('text-blue-500');
  });

  it('has minimum touch target size of 36x36px', () => {
    const { container } = render(
      <TappableInfoIcon
        symbol="🔔"
        tooltipText="Reminder"
        className=""
      />
    );
    const icon = container.firstChild as HTMLElement;
    const styles = window.getComputedStyle(icon);
    expect(parseInt(styles.minWidth)).toBeGreaterThanOrEqual(36);
    expect(parseInt(styles.minHeight)).toBeGreaterThanOrEqual(36);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- TappableInfoIcon.test.tsx`
Expected: FAIL with "Cannot find module './TappableInfoIcon'"

**Step 3: Write minimal implementation**

Create `zettelkasten-front/src/components/tasks/TappableInfoIcon.tsx`:

```typescript
import React from 'react';
import { toast } from 'react-toastify';

interface TappableInfoIconProps {
  symbol: string;
  tooltipText: string;
  className?: string;
}

export function TappableInfoIcon({ symbol, tooltipText, className = '' }: TappableInfoIconProps) {
  const handleClick = () => {
    toast.info(tooltipText);
  };

  return (
    <span
      onClick={handleClick}
      className={`inline-flex items-center justify-center text-base cursor-pointer select-none min-w-[36px] min-h-[36px] active:scale-95 transition-transform ${className}`}
      title={tooltipText}
    >
      {symbol}
    </span>
  );
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- TappableInfoIcon.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tasks/TappableInfoIcon.tsx zettelkasten-front/src/components/tasks/TappableInfoIcon.test.tsx
git commit -m "feat: add TappableInfoIcon component for mobile-friendly info badges"
```

---

## Task 3: Create MobileTagDots component

**Files:**
- Create: `zettelkasten-front/src/components/tasks/MobileTagDots.tsx`
- Create: `zettelkasten-front/src/components/tasks/MobileTagDots.test.tsx`

**Step 1: Write the failing test**

Create `zettelkasten-front/src/components/tasks/MobileTagDots.test.tsx`:

```typescript
import { render, screen } from '@testing-library/react';
import { MobileTagDots } from './MobileTagDots';
import { Tag } from '../../models/Tags';

describe('MobileTagDots', () => {
  const mockTags: Tag[] = [
    { id: 1, name: '#urgent', color: '#ef4444' },
    { id: 2, name: '#work', color: '#3b82f6' },
    { id: 3, name: '#personal', color: '#10b981' },
  ];

  it('renders up to 3 tag dots', () => {
    render(<MobileTagDots tags={mockTags} />);
    const dots = screen.getAllByTestId(/tag-dot-/);
    expect(dots).toHaveLength(3);
  });

  it('shows more indicator when more than 3 tags', () => {
    const manyTags: Tag[] = [
      ...mockTags,
      { id: 4, name: '#extra1', color: '#f59e0b' },
      { id: 5, name: '#extra2', color: '#8b5cf6' },
    ];
    render(<MobileTagDots tags={manyTags} />);
    expect(screen.getByText('•••')).toBeInTheDocument();
  });

  it('does not show more indicator when exactly 3 tags', () => {
    render(<MobileTagDots tags={mockTags} />);
    expect(screen.queryByText('•••')).not.toBeInTheDocument();
  });

  it('renders nothing when no tags', () => {
    const { container } = render(<MobileTagDots tags={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it('applies correct colors to dots', () => {
    render(<MobileTagDots tags={mockTags} />);
    const redDot = screen.getByTestId('tag-dot-1');
    const blueDot = screen.getByTestId('tag-dot-2');
    expect(redDot).toHaveStyle({ backgroundColor: '#ef4444' });
    expect(blueDot).toHaveStyle({ backgroundColor: '#3b82f6' });
  });

  it('dots are not interactive', () => {
    render(<MobileTagDots tags={mockTags} />);
    const dots = screen.getAllByTestId(/tag-dot-/);
    dots.forEach(dot => {
      expect(dot).toHaveClass('pointer-events-none');
    });
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- MobileTagDots.test.tsx`
Expected: FAIL with "Cannot find module './MobileTagDots'"

**Step 3: Write minimal implementation**

Create `zettelkasten-front/src/components/tasks/MobileTagDots.tsx`:

```typescript
import React from 'react';
import { Tag } from '../../models/Tags';

interface MobileTagDotsProps {
  tags: Tag[];
  maxDots?: number;
}

export function MobileTagDots({ tags, maxDots = 3 }: MobileTagDotsProps) {
  if (tags.length === 0) {
    return null;
  }

  const visibleDots = tags.slice(0, maxDots);
  const hasMore = tags.length > maxDots;

  return (
    <span className="flex items-center gap-1 ml-2">
      {visibleDots.map((tag) => (
        <span
          key={tag.id}
          data-testid={`tag-dot-${tag.id}`}
          className="pointer-events-none rounded-full w-2.5 h-2.5"
          style={{ backgroundColor: tag.color || '#6b7280' }}
          title={tag.name}
        />
      ))}
      {hasMore && (
        <span className="text-gray-400 text-xs ml-1">•••</span>
      )}
    </span>
  );
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- MobileTagDots.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tasks/MobileTagDots.tsx zettelkasten-front/src/components/tasks/MobileTagDots.test.tsx
git commit -m "feat: add MobileTagDots component for compact tag display on mobile"
```

---

## Task 4: Create useLongPress hook

**Files:**
- Create: `zettelkasten-front/src/hooks/useLongPress.ts`
- Create: `zettelkasten-front/src/hooks/useLongPress.test.ts`

**Step 1: Write the failing test**

Create `zettelkasten-front/src/hooks/useLongPress.test.ts`:

```typescript
import { renderHook, act } from '@testing-library/react';
import { useLongPress } from './useLongPress';
import { vi } from 'vitest';

describe('useLongPress', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should trigger long press after 500ms', () => {
    const onLongPress = vi.fn();
    const { result } = renderHook(() => useLongPress(onLongPress));

    act(() => {
      const touchStart = new TouchEvent('touchstart', {
        bubbles: true,
        cancelable: true,
      });
      result.current.onTouchStart(touchStart);
      vi.advanceTimersByTime(500);
    });

    expect(onLongPress).toHaveBeenCalledTimes(1);
  });

  it('should not trigger long press if touch ends before 500ms', () => {
    const onLongPress = vi.fn();
    const { result } = renderHook(() => useLongPress(onLongPress));

    act(() => {
      result.current.onTouchStart(new TouchEvent('touchstart'));
      vi.advanceTimersByTime(200);
      result.current.onTouchEnd();
    });

    expect(onLongPress).not.toHaveBeenCalled();
  });

  it('should cancel long press if touch moves', () => {
    const onLongPress = vi.fn();
    const { result } = renderHook(() => useLongPress(onLongPress));

    act(() => {
      result.current.onTouchStart(new TouchEvent('touchstart'));
      vi.advanceTimersByTime(300);
      result.current.onTouchMove();
      vi.advanceTimersByTime(200);
    });

    expect(onLongPress).not.toHaveBeenCalled();
  });

  it('should reset timer after touch ends', () => {
    const onLongPress = vi.fn();
    const { result } = renderHook(() => useLongPress(onLongPress));

    act(() => {
      // First touch - short tap, no long press
      result.current.onTouchStart(new TouchEvent('touchstart'));
      vi.advanceTimersByTime(200);
      result.current.onTouchEnd();

      // Second touch - should be able to trigger long press
      result.current.onTouchStart(new TouchEvent('touchstart'));
      vi.advanceTimersByTime(500);
    });

    expect(onLongPress).toHaveBeenCalledTimes(1);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- useLongPress.test.ts`
Expected: FAIL with "Cannot find module './useLongPress'"

**Step 3: Write minimal implementation**

Create `zettelkasten-front/src/hooks/useLongPress.ts`:

```typescript
import { useRef, useCallback, useEffect } from 'react';

const LONG_PRESS_DELAY = 500;

export function useLongPress(
  onLongPress: () => void,
  delay: number = LONG_PRESS_DELAY
) {
  const timerRef = useRef<NodeJS.Timeout | null>(null);
  const isLongPressTriggeredRef = useRef(false);

  const start = useCallback(() => {
    isLongPressTriggeredRef.current = false;
    timerRef.current = setTimeout(() => {
      isLongPressTriggeredRef.current = true;
      onLongPress();
    }, delay);
  }, [delay, onLongPress]);

  const clear = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const onTouchStart = useCallback((e: React.TouchEvent) => {
    // Prevent default context menu on long press
    e.preventDefault();
    start();
  }, [start]);

  const onTouchEnd = useCallback(() => {
    clear();
  }, [clear]);

  const onTouchMove = useCallback(() => {
    clear();
  }, [clear]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      clear();
    };
  }, [clear]);

  return {
    onTouchStart,
    onTouchEnd,
    onTouchMove,
    // Also provide mouse event handlers for desktop testing
    onMouseDown: start,
    onMouseUp: clear,
    onMouseLeave: clear,
  };
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- useLongPress.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/hooks/useLongPress.ts zettelkasten-front/src/hooks/useLongPress.test.ts
git commit -m "feat: add useLongPress hook for mobile long press detection"
```

---

## Task 5: Update TaskListItem for mobile responsive layout

**Files:**
- Modify: `zettelkasten-front/src/components/tasks/TaskListItem.tsx`
- Modify: `zettelkasten-front/src/components/tasks/TaskListItem.test.tsx`

**Step 1: Write the failing tests**

Add to existing test file `zettelkasten-front/src/components/tasks/TaskListItem.test.tsx` (create if doesn't exist):

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TaskListItem } from './TaskListItem';
import { Task } from '../../models/Task';
import { TaskContextProvider } from '../../contexts/TaskContext';
import { DialogStateContextProvider } from '../../contexts/DialogStateContext';
import { StatusContextProvider } from '../../contexts/StatusContext';
import { AuthContextProvider } from '../../contexts/AuthContext';

const mockTask: Task = {
  id: 1,
  title: 'Test task with #tag1 and #tag2',
  status: 'TODO',
  is_complete: false,
  card: null,
  tags: [
    { id: 1, name: '#tag1', color: '#ef4444' },
    { id: 2, name: '#tag2', color: '#3b82f6' },
  ],
  reminder_time: new Date().toISOString(),
  blocked_by: [],
};

const renderWithProviders = (component: React.ReactElement) => {
  return render(
    <AuthContextProvider>
      <StatusContextProvider>
        <TaskContextProvider>
          <DialogStateContextProvider>
            {component}
          </DialogStateContextProvider>
        </TaskContextProvider>
      </StatusContextProvider>
    </AuthContextProvider>
  );
};

describe('TaskListItem mobile enhancements', () => {
  it('renders MobileTagDots on mobile (hidden on desktop)', () => {
    // This test verifies the component structure; actual visibility is handled by CSS
    renderWithProviders(
      <TaskListItem task={mockTask} onTagClick={jest.fn()} />
    );

    // The mobile tag dots should be in the DOM (controlled by CSS display)
    const dots = screen.queryAllByTestId(/tag-dot-/);
    expect(dots.length).toBeGreaterThan(0);
  });

  it('shows action sheet on long press', async () => {
    const onActionSpy = jest.fn();
    renderWithProviders(
      <TaskListItem task={mockTask} onTagClick={jest.fn()} />
    );

    const taskItem = screen.getByText(/Test task with/).closest('div');
    if (!taskItem) throw new Error('Task item not found');

    // Simulate long press (on desktop this uses mouse events)
    const user = userEvent.setup({ delay: 600 });
    await user.pointer([
      { keys: '[MouseLeft]', target: taskItem },
    ]);

    // Action sheet should appear
    await waitFor(() => {
      expect(screen.getByText('Toggle Complete')).toBeInTheDocument();
    });
  });

  it('uses TappableInfoIcon for reminder on mobile', () => {
    renderWithProviders(
      <TaskListItem task={mockTask} onTagClick={jest.fn()} />
    );

    // The reminder icon should be present with tappable behavior
    const reminderIcon = screen.getByText('🔔');
    expect(reminderIcon).toBeInTheDocument();
    expect(reminderIcon).toHaveClass('cursor-pointer');
  });

  it('hides due date badge on mobile (shown via CSS media query)', () => {
    const taskWithDueDate: Task = {
      ...mockTask,
      due_date: new Date().toISOString(),
    };

    renderWithProviders(
      <TaskListItem task={taskWithDueDate} onTagClick={jest.fn()} />
    );

    // Due date element should be in DOM, controlled by CSS for visibility
    expect(screen.getByText(/due/i)).toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- TaskListItem.test.tsx`
Expected: FAIL - tests will fail because we haven't updated the component yet

**Step 3: Update TaskListItem component**

Modify `zettelkasten-front/src/components/tasks/TaskListItem.tsx`. Key changes:

1. Import new components
2. Add state for action sheet
3. Add long press handler
4. Wrap badge row in mobile-responsive container
5. Replace emoji badges with TappableInfoIcon
6. Add MobileTagDots (hidden on desktop)

```typescript
import React, { useState, useEffect, KeyboardEvent } from "react";
import { deleteTask, saveExistingTask } from "../../api/tasks";
import { getTomorrow } from "../../utils/dates";

import { TaskDateDisplay } from "./TaskDateDisplay";
import { TaskDueDateDisplay } from "./TaskDueDateDisplay";
import { TaskPriorityDisplay } from "./TaskPriorityDisplay";
import { TaskStatusDisplay } from "./TaskStatusDisplay";
import { TaskTagDisplay } from "./TaskTagDisplay";
import { LongPressActionSheet } from "./LongPressActionSheet";
import { TappableInfoIcon } from "./TappableInfoIcon";
import { MobileTagDots } from "./MobileTagDots";
import { useLongPress } from "../../hooks/useLongPress";
import { Task } from "../../models/Task";
import { Tag } from "../../models/Tags";
import { Link } from "react-router-dom";
import { PartialCard } from "../../models/Card";
import { BacklinkInput } from "../cards/BacklinkInput";
import { linkifyWithDefaultOptions } from "../../utils/strings";
import { TaskClosedIcon } from "../../assets/icons/TaskClosedIcon";
import { TaskOpenIcon } from "../../assets/icons/TaskOpenIcon";
import { removeTagsFromTitle, parseTags } from "../../utils/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { useDialogState } from "../../contexts/DialogStateContext";
import { useStatus } from "../../contexts/StatusContext";
import { useAuth } from "../../contexts/AuthContext";
import { format } from "date-fns-tz";

// ... (existing interface and function signature)

export function TaskListItem({
  task,
  onTagClick,
  hideMatrixTags = false,
  selectMode = false,
  isSelected = false,
  onSelect,
}: TaskListItemProps) {
  const [editTitle, setEditTitle] = useState<boolean>(false);
  const [newTitle, setNewTitle] = useState<string>("");
  const [showCardLink, setShowCardLink] = useState<boolean>(false);
  const [showActionSheet, setShowActionSheet] = useState<boolean>(false);
  const [tags, setTags] = useState<Tag[]>([]);
  const { setRefreshTasks, updateTask } = useTaskContext();
  const { setShowTaskDialog, setSelectedTaskId } = useDialogState();
  const { getDefaultStatus, getCompleteStatus } = useStatus();
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";

  // ... (existing handler functions: handleTitleClick, handleBacklink, handleTitleEdit, handleToggleComplete, handleRemoveTag)

  // Add action sheet handler
  const handleActionSheetAction = (action: LongPressActionSheetProps['onAction'] extends (infer A) ? A : never) => {
    switch (action) {
      case 'toggle':
        handleToggleComplete();
        break;
      case 'delete':
        // Reuse existing delete functionality
        deleteTask(task.id);
        break;
      case 'edit':
        setSelectedTaskId(task.id);
        setShowTaskDialog(true);
        break;
      case 'copyTitle':
        navigator.clipboard.writeText(task.title);
        break;
      case 'viewDetails':
        setSelectedTaskId(task.id);
        setShowTaskDialog(true);
        break;
    }
  };

  const longPressHandlers = useLongPress(() => setShowActionSheet(true));

  useEffect(() => {
    setTags(task.tags);
  }, [task]);

  return (
    <div className="flex items-center bg-white">
      <div className="mr-2.5">
        {selectMode ? (
          <input
            type="checkbox"
            checked={isSelected}
            onChange={onSelect}
            className="w-5 h-5 cursor-pointer"
            onClick={(e) => e.stopPropagation()}
          />
        ) : (
          <span onClick={handleToggleComplete} className="cursor-pointer min-w-[44px] min-h-[44px] inline-flex items-center justify-center">
            {task.is_complete ? <TaskClosedIcon /> : <TaskOpenIcon />}
          </span>
        )}
      </div>
      <div className="flex-grow min-w-0" {...longPressHandlers}>
        <div className="whitespace-nowrap overflow-hidden text-ellipsis">
          <span
            onClick={handleTitleClick}
            className={task.is_complete ? "line-through" : "cursor-pointer"}
            dangerouslySetInnerHTML={{
              __html: linkifyWithDefaultOptions(
                removeTagsFromTitle(task.title),
              ),
            }}
          />
        </div>
        <div className="flex flex-wrap text-sm gap-x-2 items-center">
          {/* Essential badges - always visible */}
          <TaskStatusDisplay
            task={task}
            setTask={(task: Task) => { }}
            saveOnChange={true}
          />
          <TaskDateDisplay
            task={task}
            setTask={(task: Task) => { }}
            saveOnChange={true}
          />
          <TaskPriorityDisplay
            task={task}
            setTask={(task: Task) => { }}
            saveOnChange={true}
          />

          {/* Hidden on mobile, shown on desktop */}
          <div className="hidden md:inline-flex">
            {task.due_date && (
              <TaskDueDateDisplay
                task={task}
                setTask={(task: Task) => { }}
                saveOnChange={true}
              />
            )}
            <TaskTagDisplay task={task} tags={tags} onTagClick={onTagClick} onRemoveTag={handleRemoveTag} hideMatrixTags={hideMatrixTags} />
          </div>

          {/* Mobile-only items */}
          <div className="md:hidden flex items-center gap-1">
            {task.reminder_time && (
              <TappableInfoIcon
                symbol="🔔"
                tooltipText={task.reminder_sent
                  ? `Reminder sent: ${format(new Date(task.reminder_time), 'MMM d, h:mm a', { timeZone: userTimezone })}`
                  : `Reminder set for: ${format(new Date(task.reminder_time), 'MMM d, h:mm a', { timeZone: userTimezone })}`
                }
                className={task.reminder_sent ? 'text-gray-400' : 'text-blue-500'}
              />
            )}
            {task.blocked_by && task.blocked_by.filter(bt => !bt.is_complete).length > 0 && (
              <TappableInfoIcon
                symbol="🚧"
                tooltipText={`Blocked by: ${task.blocked_by.filter(bt => !bt.is_complete).map(bt => bt.title).join(', ')}`}
                className="text-orange-600"
              />
            )}
            <MobileTagDots tags={tags} />
          </div>

          {/* Desktop emoji badges (non-tappable, for reference) */}
          <div className="hidden md:inline-flex">
            {task.reminder_time && (
              <span
                className="text-xs mr-1"
                style={{
                  color: task.reminder_sent ? '#9ca3af' : '#3b82f6',
                  cursor: 'default'
                }}
                title={task.reminder_sent
                  ? `Reminder sent: ${format(new Date(task.reminder_time), 'MMM d, yyyy h:mm a', { timeZone: userTimezone })}`
                  : `Reminder set for: ${format(new Date(task.reminder_time), 'MMM d, yyyy h:mm a', { timeZone: userTimezone })}`
                }
              >
                🔔
              </span>
            )}
            {task.blocked_by && task.blocked_by.filter(bt => !bt.is_complete).length > 0 && (
              <span
                className="text-xs mr-1 px-1 rounded"
                style={{
                  backgroundColor: '#fed7aa',
                  color: '#9a3412',
                  cursor: 'default'
                }}
                title={`Blocked by: ${task.blocked_by.filter(bt => !bt.is_complete).map(bt => bt.title).join(', ')}`}
              >
                🚧
              </span>
            )}
          </div>
        </div>
      </div>
      <div className="ml-2.5">
        {task.card && task.card.id > 0 && (
          <Link
            to={`/app/card/${task.card.id}`}
            style={{ textDecoration: "none", color: "inherit" }}
          >
            <span className="card-id">[{task.card.card_id}]</span>
          </Link>
        )}
        {!task.card ||
          (task.card.id == 0 && (
            <div>
              {showCardLink && <BacklinkInput addBacklink={handleBacklink} />}
            </div>
          ))}
      </div>
      <button
        onClick={() => {
          setSelectedTaskId(task.id);
          setShowTaskDialog(true);
        }}
        className="bg-transparent border-0 cursor-pointer p-2.5 min-w-[44px] min-h-[44px] flex items-center justify-center hover:bg-gray-100 rounded transition-colors"
        aria-label="Task options"
      >
        ⋮
      </button>

      <LongPressActionSheet
        visible={showActionSheet}
        onAction={handleActionSheetAction}
        onClose={() => setShowActionSheet(false)}
        task={task}
      />
    </div>
  );
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- TaskListItem.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tasks/TaskListItem.tsx zettelkasten-front/src/components/tasks/TaskListItem.test.tsx
git commit -m "feat: add mobile responsive layout to TaskListItem with long-press actions"
```

---

## Task 6: Add CSS for proper mobile touch targets

**Files:**
- Modify: `zettelkasten-front/src/index.css` (or appropriate global CSS file)

**Step 1: Add mobile-specific CSS adjustments**

Add to the main CSS file to ensure proper spacing and touch targets:

```css
/* Mobile task list touch targets */
@media (max-width: 767px) {
  /* Ensure proper spacing between touch targets */
  .task-list-item {
    padding: 0.5rem 0;
  }

  /* Increase tap area for badges without affecting visual size */
  .task-badge {
    min-width: 44px;
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.25rem 0.5rem;
  }

  /* Prevent text selection on long press */
  .task-list-item .long-press-target {
    -webkit-touch-callout: none;
    -webkit-user-select: none;
    user-select: none;
  }

  /* Action sheet backdrop transition */
  .action-sheet-backdrop {
    transition: opacity 0.2s ease-in-out;
  }

  /* Action sheet slide animation */
  .action-sheet-content {
    transition: transform 0.3s ease-out;
  }

  /* Tag dots spacing */
  .tag-dot {
    margin-right: 0.25rem;
  }
}
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/index.css
git commit -m "style: add mobile touch target CSS with proper spacing"
```

---

## Task 7: Manual testing and verification

**Files:** None (manual testing)

**Step 1: Test on mobile viewport**

1. Open DevTools, toggle device toolbar (mobile mode)
2. Navigate to task list
3. Verify:
   - Long press (hold ~500ms) opens action sheet
   - Action sheet buttons are tappable with proper spacing
   - Tag dots appear instead of full badges
   - Reminder/blocked icons show toast on tap
   - Due date badge is hidden
   - Complete toggle has 44x44px tap area
   - No accidental taps when scrolling

**Step 2: Test on desktop viewport**

1. Open normal browser window (>768px)
2. Verify:
   - All original functionality works
   - All badges visible
   - Original emoji badges shown
   - No visual regressions

**Step 3: Edge cases to verify**

- Tasks with no tags
- Tasks with 10+ tags (dot indicator)
- Tasks with both reminder and blocked
- Tasks in select mode
- Completed tasks

**Step 4: Accessibility check**

- Touch targets minimum 44x44px
- 8px spacing between targets
- Action sheet dismissible with backdrop tap
- Focus management for keyboard users

**Step 5: Create bead for completion**

No commit needed - update tracking when complete.

---

## Summary of Beads (Tasks)

This plan creates 7 main tasks:

1. **LongPressActionSheet component** - Modal with action buttons
2. **TappableInfoIcon component** - Replace emoji badges with tappable version
3. **MobileTagDots component** - Compact tag display
4. **useLongPress hook** - Touch event handling
5. **TaskListItem updates** - Wire everything together
6. **CSS touch targets** - Spacing and sizing
7. **Manual testing** - Verification on mobile/desktop

Each task follows TDD: failing test → implementation → passing test → commit.
