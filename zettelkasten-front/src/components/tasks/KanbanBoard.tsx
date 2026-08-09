import React, { useState, useCallback, useEffect, useRef } from 'react';
import { Modal } from '../ui/Modal';
import {
  DragDropContext,
  Droppable,
  Draggable,
  DropResult,
} from '@hello-pangea/dnd';
import { Task } from '../../models/Task';
import { TaskListItem } from './TaskListItem';
import { saveExistingTask } from '../../api/tasks';
import { useTaskContext } from '../../contexts/TaskContext';
import { useStatus } from '../../contexts/StatusContext';
import { StatusManagement } from '../settings/StatusManagement';
import { useDialogState } from '../../contexts/DialogStateContext';
import { TaskHoverCard } from './TaskHoverCard';
import { KanbanQuickActions } from './KanbanQuickActions';
import { SubtaskDisplayMode } from '../../hooks/useSubtaskDisplayMode';

interface FocusedCard {
  columnIndex: number;
  cardIndex: number;
}

/**
 * Hook for keyboard navigation within a kanban board
 */
function useKanbanKeyboardNavigation(
  columnCount: number,
  getCardCount: (columnIndex: number) => number,
  getTaskId: (columnIndex: number, cardIndex: number) => number | null,
  enabled: boolean = true,
) {
  const [focusedCard, setFocusedCard] = useState<FocusedCard | null>(null);
  const { setSelectedTaskId, setShowTaskDialog } = useDialogState();
  const boardRef = useRef<HTMLDivElement>(null);

  // Get the current card count for a column, memoized
  const getCardCountMemo = useCallback(getCardCount, [
    columnCount,
    getCardCount,
  ]);

  // Focus the board container when a card is focused
  useEffect(() => {
    if (focusedCard && boardRef.current) {
      boardRef.current.focus();
    }
  }, [focusedCard]);

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (!enabled) return;

      // Ignore if we're inside an input field
      const target = event.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable
      ) {
        return;
      }

      switch (event.key) {
        case 'ArrowUp':
          event.preventDefault();
          setFocusedCard((prev) => {
            if (!prev) {
              // Start at first card of first column
              const firstCardCount = getCardCountMemo(0);
              if (firstCardCount > 0) {
                return { columnIndex: 0, cardIndex: 0 };
              }
              return null;
            }
            // Move up within column, wrap to bottom
            const upCardCount = getCardCountMemo(prev.columnIndex);
            if (upCardCount === 0) return prev;
            return {
              ...prev,
              cardIndex:
                prev.cardIndex > 0 ? prev.cardIndex - 1 : upCardCount - 1,
            };
          });
          break;

        case 'ArrowDown':
          event.preventDefault();
          setFocusedCard((prev) => {
            if (!prev) {
              // Start at first card of first column
              const firstCardCount = getCardCountMemo(0);
              if (firstCardCount > 0) {
                return { columnIndex: 0, cardIndex: 0 };
              }
              return null;
            }
            // Move down within column, wrap to top
            const downCardCount = getCardCountMemo(prev.columnIndex);
            if (downCardCount === 0) return prev;
            return {
              ...prev,
              cardIndex:
                prev.cardIndex < downCardCount - 1 ? prev.cardIndex + 1 : 0,
            };
          });
          break;

        case 'ArrowLeft':
          event.preventDefault();
          setFocusedCard((prev) => {
            if (!prev) {
              // Start at first card of first column
              const firstCardCount = getCardCountMemo(0);
              if (firstCardCount > 0) {
                return { columnIndex: 0, cardIndex: 0 };
              }
              return null;
            }
            // Move to previous column, keep same index if possible
            const newLeftCol =
              prev.columnIndex > 0 ? prev.columnIndex - 1 : columnCount - 1;
            const leftCardCount = getCardCountMemo(newLeftCol);
            if (leftCardCount === 0) {
              // Skip empty columns
              for (let i = 1; i < columnCount; i++) {
                const checkCol = (newLeftCol - i + columnCount) % columnCount;
                const checkCount = getCardCountMemo(checkCol);
                if (checkCount > 0) {
                  return {
                    columnIndex: checkCol,
                    cardIndex: Math.min(prev.cardIndex, checkCount - 1),
                  };
                }
              }
              return prev;
            }
            return {
              columnIndex: newLeftCol,
              cardIndex: Math.min(prev.cardIndex, leftCardCount - 1),
            };
          });
          break;

        case 'ArrowRight':
          event.preventDefault();
          setFocusedCard((prev) => {
            if (!prev) {
              // Start at first card of first column
              const firstCardCount = getCardCountMemo(0);
              if (firstCardCount > 0) {
                return { columnIndex: 0, cardIndex: 0 };
              }
              return null;
            }
            // Move to next column, keep same index if possible
            const newRightCol =
              prev.columnIndex < columnCount - 1 ? prev.columnIndex + 1 : 0;
            const rightCardCount = getCardCountMemo(newRightCol);
            if (rightCardCount === 0) {
              // Skip empty columns
              for (let i = 1; i < columnCount; i++) {
                const checkCol = (newRightCol + i) % columnCount;
                const checkCount = getCardCountMemo(checkCol);
                if (checkCount > 0) {
                  return {
                    columnIndex: checkCol,
                    cardIndex: Math.min(prev.cardIndex, checkCount - 1),
                  };
                }
              }
              return prev;
            }
            return {
              columnIndex: newRightCol,
              cardIndex: Math.min(prev.cardIndex, rightCardCount - 1),
            };
          });
          break;

        case 'Enter':
          event.preventDefault();
          if (focusedCard) {
            const taskId = getTaskId(
              focusedCard.columnIndex,
              focusedCard.cardIndex,
            );
            if (taskId !== null) {
              setSelectedTaskId(taskId);
              setShowTaskDialog(true);
            }
          }
          break;

        case 'Escape':
          event.preventDefault();
          setFocusedCard(null);
          break;

        case 'j':
        case 'J':
          // Vim-style down
          event.preventDefault();
          setFocusedCard((prev) => {
            if (!prev) {
              const firstCardCount = getCardCountMemo(0);
              if (firstCardCount > 0) {
                return { columnIndex: 0, cardIndex: 0 };
              }
              return null;
            }
            const jCardCount = getCardCountMemo(prev.columnIndex);
            if (jCardCount === 0) return prev;
            return {
              ...prev,
              cardIndex:
                prev.cardIndex < jCardCount - 1 ? prev.cardIndex + 1 : 0,
            };
          });
          break;

        case 'k':
        case 'K':
          // Vim-style up
          event.preventDefault();
          setFocusedCard((prev) => {
            if (!prev) {
              const firstCardCount = getCardCountMemo(0);
              if (firstCardCount > 0) {
                return { columnIndex: 0, cardIndex: 0 };
              }
              return null;
            }
            const kCardCount = getCardCountMemo(prev.columnIndex);
            if (kCardCount === 0) return prev;
            return {
              ...prev,
              cardIndex:
                prev.cardIndex > 0 ? prev.cardIndex - 1 : kCardCount - 1,
            };
          });
          break;

        case 'h':
        case 'H':
          // Vim-style left (when not in input)
          event.preventDefault();
          setFocusedCard((prev) => {
            if (!prev) return null;
            const newCol =
              prev.columnIndex > 0 ? prev.columnIndex - 1 : columnCount - 1;
            const hCardCount = getCardCountMemo(newCol);
            if (hCardCount === 0) return prev;
            return {
              columnIndex: newCol,
              cardIndex: Math.min(prev.cardIndex, hCardCount - 1),
            };
          });
          break;

        case 'l':
        case 'L':
          // Vim-style right (when not in input)
          event.preventDefault();
          setFocusedCard((prev) => {
            if (!prev) return null;
            const newCol =
              prev.columnIndex < columnCount - 1 ? prev.columnIndex + 1 : 0;
            const lCardCount = getCardCountMemo(newCol);
            if (lCardCount === 0) return prev;
            return {
              columnIndex: newCol,
              cardIndex: Math.min(prev.cardIndex, lCardCount - 1),
            };
          });
          break;
      }
    },
    [
      enabled,
      columnCount,
      getCardCountMemo,
      getTaskId,
      focusedCard,
      setSelectedTaskId,
      setShowTaskDialog,
    ],
  );

  const clearFocus = useCallback(() => setFocusedCard(null), []);

  return {
    focusedCard,
    handleKeyDown,
    clearFocus,
    boardRef,
  };
}

interface KanbanBoardProps {
  tasks: Task[];
  onTagClick: (tag: string) => void;
  onAddTaskWithStatus: (status: string) => void;
  selectMode?: boolean;
  selectedTaskIds?: Set<number>;
  onTaskSelect?: (taskId: number) => void;
  subtaskMode?: SubtaskDisplayMode;
}

type KanbanSortField = 'priority' | 'scheduled_date' | 'title' | 'created_at';
type KanbanSortDirection = 'asc' | 'desc';

interface KanbanSortSettings {
  field: KanbanSortField;
  direction: KanbanSortDirection;
}

interface WipLimits {
  [statusName: string]: number; // 0 means no limit
}

const KANBAN_SORT_KEY = 'kanbanSortSettings';
const KANBAN_WIP_LIMITS_KEY = 'kanbanWipLimits';

const DEFAULT_SORT: KanbanSortSettings = {
  field: 'priority',
  direction: 'desc', // High priority first
};

const PRIORITY_ORDER = { A: 3, B: 2, C: 1 };

function sortTasks(tasks: Task[], settings: KanbanSortSettings): Task[] {
  return [...tasks].sort((a, b) => {
    let comparison = 0;

    switch (settings.field) {
      case 'priority':
        const aPriority = a.priority
          ? PRIORITY_ORDER[a.priority as keyof typeof PRIORITY_ORDER] || 0
          : 0;
        const bPriority = b.priority
          ? PRIORITY_ORDER[b.priority as keyof typeof PRIORITY_ORDER] || 0
          : 0;
        comparison = aPriority - bPriority;
        break;

      case 'scheduled_date':
        const aScheduled = a.scheduled_date
          ? new Date(a.scheduled_date).getTime()
          : Infinity;
        const bScheduled = b.scheduled_date
          ? new Date(b.scheduled_date).getTime()
          : Infinity;
        comparison = aScheduled - bScheduled;
        break;

      case 'title':
        comparison = a.title.localeCompare(b.title);
        break;

      case 'created_at':
        comparison =
          new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
        break;
    }

    return settings.direction === 'asc' ? comparison : -comparison;
  });
}

export function KanbanBoard({
  tasks,
  onTagClick,
  onAddTaskWithStatus,
  selectMode = false,
  selectedTaskIds = new Set(),
  onTaskSelect,
  subtaskMode = 'nested',
}: KanbanBoardProps) {
  const { setRefreshTasks } = useTaskContext();
  const { statuses, getStatusByName } = useStatus();
  const [showStatusManagement, setShowStatusManagement] = useState(false);
  const [showSortMenu, setShowSortMenu] = useState(false);
  const [showWipModal, setShowWipModal] = useState(false);
  const [editingWipStatus, setEditingWipStatus] = useState<string | null>(null);
  const [tempWipValue, setTempWipValue] = useState<string>('');
  const sortMenuRef = useRef<HTMLDivElement>(null);

  // Load sort settings from localStorage
  const [sortSettings, setSortSettings] = useState<KanbanSortSettings>(() => {
    try {
      const saved = localStorage.getItem(KANBAN_SORT_KEY);
      if (saved) {
        return JSON.parse(saved) as KanbanSortSettings;
      }
    } catch (e) {
      console.error('Failed to load kanban sort settings:', e);
    }
    return DEFAULT_SORT;
  });

  // Load WIP limits from localStorage
  const [wipLimits, setWipLimits] = useState<WipLimits>(() => {
    try {
      const saved = localStorage.getItem(KANBAN_WIP_LIMITS_KEY);
      if (saved) {
        return JSON.parse(saved) as WipLimits;
      }
    } catch (e) {
      console.error('Failed to load kanban WIP limits:', e);
    }
    return {};
  });

  // Persist sort settings to localStorage
  useEffect(() => {
    try {
      localStorage.setItem(KANBAN_SORT_KEY, JSON.stringify(sortSettings));
    } catch (e) {
      console.error('Failed to save kanban sort settings:', e);
    }
  }, [sortSettings]);

  // Persist WIP limits to localStorage
  useEffect(() => {
    try {
      localStorage.setItem(KANBAN_WIP_LIMITS_KEY, JSON.stringify(wipLimits));
    } catch (e) {
      console.error('Failed to save kanban WIP limits:', e);
    }
  }, [wipLimits]);

  // Close sort menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        sortMenuRef.current &&
        !sortMenuRef.current.contains(event.target as Node)
      ) {
        setShowSortMenu(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // WIP limit handlers
  const handleSetWipLimit = (statusName: string, limit: number) => {
    setWipLimits((prev) => {
      if (limit <= 0) {
        const { [statusName]: _, ...rest } = prev;
        return rest;
      }
      return { ...prev, [statusName]: limit };
    });
    setEditingWipStatus(null);
    setShowWipModal(false);
  };

  const openWipEditor = (statusName: string) => {
    setEditingWipStatus(statusName);
    setTempWipValue(String(wipLimits[statusName] || 0));
    setShowWipModal(true);
  };

  // Separate root tasks from subtasks
  const { rootTasks, subtasksByParent, allTasksIncludingSubtasks } =
    React.useMemo(() => {
      const rootTasks: Task[] = [];
      const subtasksByParent: Record<number, Task[]> = {};
      const allTasksIncludingSubtasks: Task[] = [];

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
        allTasksIncludingSubtasks.push(task);
      });

      return { rootTasks, subtasksByParent, allTasksIncludingSubtasks };
    }, [tasks]);

  // Determine which tasks to display based on subtask mode
  // - nested/hidden: only show root tasks
  // - flat: show all tasks including subtasks (each in their own status column)
  const tasksToDisplay =
    subtaskMode === 'flat' ? allTasksIncludingSubtasks : rootTasks;

  // Create a map of parent tasks for subtasks (used in flat mode)
  const parentTaskMap = React.useMemo(() => {
    const map: Record<number, Task> = {};
    tasks.forEach((task) => {
      map[task.id] = task;
    });
    return map;
  }, [tasks]);

  // Group tasks by status based on display mode
  const tasksByStatus = tasksToDisplay.reduce(
    (acc, task) => {
      const status =
        task.status || statuses.find((s) => s.is_default)?.name || 'todo';
      if (!acc[status]) {
        acc[status] = [];
      }
      acc[status].push(task);
      return acc;
    },
    {} as Record<string, Task[]>,
  );

  // Sort tasks within each column
  const sortedTasksByStatus = React.useMemo(() => {
    const sorted: Record<string, Task[]> = {};
    for (const [status, statusTasks] of Object.entries(tasksByStatus)) {
      sorted[status] = sortTasks(statusTasks, sortSettings);
    }
    return sorted;
  }, [tasksByStatus, sortSettings]);

  // Create a stable array of tasks per column for keyboard navigation
  const columnTasksArrays = statuses.map(
    (status) => sortedTasksByStatus[status.name] || [],
  );

  // Helper functions for keyboard navigation
  const getCardCount = useCallback(
    (columnIndex: number) => {
      return columnTasksArrays[columnIndex]?.length || 0;
    },
    [columnTasksArrays],
  );

  const getTaskId = useCallback(
    (columnIndex: number, cardIndex: number) => {
      const tasks = columnTasksArrays[columnIndex];
      return tasks?.[cardIndex]?.id ?? null;
    },
    [columnTasksArrays],
  );

  // Keyboard navigation hook
  const { focusedCard, handleKeyDown, clearFocus, boardRef } =
    useKanbanKeyboardNavigation(
      statuses.length,
      getCardCount,
      getTaskId,
      !selectMode, // Disable keyboard nav in select mode
    );

  const onDragEnd = async (result: DropResult) => {
    if (!result.destination) return;

    const sourceStatus = result.source.droppableId;
    const destStatus = result.destination.droppableId;

    // No change if dropped in the same column
    if (sourceStatus === destStatus) return;

    const draggedId = result.draggableId;
    const task = tasks.find((t) => t.id.toString() === draggedId);
    if (!task) return;

    // Update task status
    const updatedTask = { ...task, status: destStatus };

    // Sync is_complete with status based on status configuration
    const statusConfig = getStatusByName(destStatus);
    if (statusConfig) {
      updatedTask.is_complete = statusConfig.is_complete_state;
    }

    try {
      // Persist changes
      const response = await saveExistingTask(updatedTask);
      if (!('error' in response)) {
        setRefreshTasks(true);
      }
    } catch (err) {
      console.error('Failed to save updated task after drag-and-drop:', err);
    }
  };

  return (
    <>
      {/* Header with Manage Statuses Button */}
      <div className="mb-4 flex justify-between items-center">
        <h2 className="text-xl font-semibold text-gray-900">Task Board</h2>
        <div className="flex items-center gap-2">
          {/* Sort Dropdown */}
          <div className="relative" ref={sortMenuRef}>
            <button
              onClick={() => setShowSortMenu(!showSortMenu)}
              className="flex items-center gap-2 px-3 py-2 text-sm bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12"
                />
              </svg>
              <span className="hidden sm:inline">Sort</span>
            </button>
            {showSortMenu && (
              <div className="absolute right-0 mt-1 w-48 bg-white border border-gray-200 rounded-lg shadow-lg z-10">
                <div className="p-2">
                  <div className="text-xs font-medium text-gray-500 uppercase mb-1">
                    Sort by
                  </div>
                  {[
                    { field: 'priority' as KanbanSortField, label: 'Priority' },
                    {
                      field: 'scheduled_date' as KanbanSortField,
                      label: 'Scheduled Date',
                    },
                    { field: 'title' as KanbanSortField, label: 'Title' },
                    {
                      field: 'created_at' as KanbanSortField,
                      label: 'Created Date',
                    },
                  ].map((option) => (
                    <button
                      key={option.field}
                      onClick={() => {
                        if (sortSettings.field === option.field) {
                          // Toggle direction if same field
                          setSortSettings({
                            ...sortSettings,
                            direction:
                              sortSettings.direction === 'asc' ? 'desc' : 'asc',
                          });
                        } else {
                          // Reset to default direction for new field
                          setSortSettings({
                            field: option.field,
                            direction:
                              option.field === 'priority' ? 'desc' : 'asc',
                          });
                        }
                        setShowSortMenu(false);
                      }}
                      className={`w-full text-left px-3 py-2 text-sm rounded flex items-center justify-between ${
                        sortSettings.field === option.field
                          ? 'bg-blue-50 text-blue-700'
                          : 'hover:bg-gray-50'
                      }`}
                    >
                      <span>{option.label}</span>
                      {sortSettings.field === option.field && (
                        <span className="text-xs">
                          {sortSettings.direction === 'asc' ? '↑' : '↓'}
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          <button
            onClick={() => setShowStatusManagement(true)}
            className="flex items-center gap-2 px-4 py-2 text-sm bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
            </svg>
            Manage Statuses
          </button>
        </div>
      </div>

      <DragDropContext onDragEnd={onDragEnd}>
        <div
          ref={boardRef}
          className="flex gap-4 overflow-x-auto pb-4 min-h-[calc(100vh-200px)] outline-none"
          tabIndex={0}
          onKeyDown={handleKeyDown}
          onClick={clearFocus}
        >
          {statuses.map((column, columnIndex) => {
            const columnTasks = sortedTasksByStatus[column.name] || [];

            return (
              <div
                key={column.name}
                className="flex flex-col bg-gray-50 rounded-lg border border-gray-200 flex-shrink-0 min-h-[300px]"
                style={{ minWidth: '400px', maxWidth: '400px' }}
              >
                {/* Column Header */}
                <div
                  className="p-3 rounded-t-lg border-b-2"
                  style={{
                    backgroundColor: column.color + '15',
                    borderBottomColor: column.color,
                  }}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-lg">{column.icon}</span>
                      <h3
                        className="font-semibold text-sm"
                        style={{ color: column.color }}
                      >
                        {column.display_name}
                      </h3>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => onAddTaskWithStatus(column.name)}
                        className="min-w-[44px] min-h-[44px] flex items-center justify-center rounded hover:bg-black/10 transition-colors"
                        title={`Add task to ${column.display_name}`}
                        style={{ color: column.color }}
                      >
                        <span className="text-lg font-bold leading-none pb-1">
                          +
                        </span>
                      </button>
                      <button
                        onClick={() => openWipEditor(column.name)}
                        className={`text-xs font-medium px-2 py-0.5 rounded-full cursor-pointer transition-colors ${
                          wipLimits[column.name] &&
                          columnTasks.length > wipLimits[column.name]
                            ? 'bg-red-100 text-red-700 ring-2 ring-red-300'
                            : wipLimits[column.name] &&
                              columnTasks.length === wipLimits[column.name]
                            ? 'bg-amber-100 text-amber-700'
                            : ''
                        }`}
                        style={
                          !wipLimits[column.name]
                            ? {
                                backgroundColor: column.color + '20',
                                color: column.color,
                              }
                            : undefined
                        }
                        title={
                          wipLimits[column.name]
                            ? `WIP limit: ${
                                wipLimits[column.name]
                              }. Click to edit.`
                            : 'Click to set WIP limit'
                        }
                      >
                        {columnTasks.length}
                        {wipLimits[column.name] && `/${wipLimits[column.name]}`}
                        {wipLimits[column.name] &&
                          columnTasks.length > wipLimits[column.name] &&
                          ' ⚠️'}
                      </button>
                    </div>
                  </div>
                </div>

                {/* Column Tasks - Droppable Zone */}
                <Droppable droppableId={column.name}>
                  {(dropProvided, snapshot) => (
                    <div
                      ref={dropProvided.innerRef}
                      {...dropProvided.droppableProps}
                      className={`flex-1 p-2 space-y-2 overflow-y-auto transition-colors ${
                        snapshot.isDraggingOver ? 'bg-blue-50' : ''
                      }`}
                    >
                      {columnTasks.length > 0 ? (
                        columnTasks.map((task, index) => {
                          const isFocused =
                            focusedCard?.columnIndex === columnIndex &&
                            focusedCard?.cardIndex === index;
                          return (
                            <Draggable
                              key={task.id.toString()}
                              draggableId={task.id.toString()}
                              index={index}
                            >
                              {(dragProvided, dragSnapshot) => (
                                <TaskHoverCard task={task}>
                                  <div
                                    ref={dragProvided.innerRef}
                                    {...dragProvided.draggableProps}
                                    {...dragProvided.dragHandleProps}
                                    className={`group bg-white rounded border shadow-sm transition-all ${
                                      dragSnapshot.isDragging
                                        ? 'border-blue-400 shadow-lg'
                                        : isFocused
                                        ? 'border-blue-500 shadow-md ring-2 ring-blue-200'
                                        : 'border-gray-200 hover:shadow-md'
                                    }`}
                                  >
                                    <TaskListItem
                                      task={task}
                                      onTagClick={onTagClick}
                                      hideMatrixTags={false}
                                      selectMode={selectMode}
                                      isSelected={selectedTaskIds.has(task.id)}
                                      onSelect={() => onTaskSelect?.(task.id)}
                                      parentTask={
                                        task.parent_task_id
                                          ? parentTaskMap[task.parent_task_id]
                                          : undefined
                                      }
                                    />

                                    {/* Nested subtasks - only shown in nested mode */}
                                    {subtaskMode === 'nested' &&
                                      subtasksByParent[task.id] &&
                                      subtasksByParent[task.id].length > 0 && (
                                        <div className="px-3 pb-2 space-y-1 border-t border-gray-100 mt-1 pt-2">
                                          <div className="text-xs text-gray-500 mb-2">
                                            {
                                              subtasksByParent[task.id].filter(
                                                (s) => s.is_complete,
                                              ).length
                                            }
                                            /{subtasksByParent[task.id].length}{' '}
                                            subtasks
                                          </div>
                                          {subtasksByParent[task.id].map(
                                            (subtask) => (
                                              <div
                                                key={subtask.id}
                                                className="ml-2 pl-2 border-l-2 border-gray-200"
                                              >
                                                <TaskListItem
                                                  task={subtask}
                                                  onTagClick={onTagClick}
                                                  hideMatrixTags={false}
                                                  selectMode={selectMode}
                                                  isSelected={selectedTaskIds.has(
                                                    subtask.id,
                                                  )}
                                                  onSelect={() =>
                                                    onTaskSelect?.(subtask.id)
                                                  }
                                                />
                                              </div>
                                            ),
                                          )}
                                        </div>
                                      )}

                                    {/* Quick Actions Toolbar */}
                                    <div className="px-2 pb-2 flex justify-end">
                                      <KanbanQuickActions task={task} />
                                    </div>
                                  </div>
                                </TaskHoverCard>
                              )}
                            </Draggable>
                          );
                        })
                      ) : (
                        <div className="text-center text-gray-400 text-sm py-8">
                          No tasks
                        </div>
                      )}
                      {dropProvided.placeholder}
                    </div>
                  )}
                </Droppable>
              </div>
            );
          })}
        </div>
      </DragDropContext>

      {/* Status Management Modal */}
      {showStatusManagement && (
        <Modal
          open
          onClose={() => setShowStatusManagement(false)}
          size="4xl"
          dialogClassName="z-50"
          className="max-h-[90vh] overflow-y-auto relative !p-0"
        >
          {/* Close button */}
          <button
            onClick={() => setShowStatusManagement(false)}
            className="sticky top-0 right-0 float-right m-4 p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors z-10"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>

          {/* StatusManagement component */}
          <div className="p-6">
            <StatusManagement />
          </div>
        </Modal>
      )}

      {/* WIP Limit Modal */}
      {showWipModal && editingWipStatus && (
        <Modal
          open
          onClose={() => {
            setEditingWipStatus(null);
            setShowWipModal(false);
          }}
          size="sm"
          dialogClassName="z-50"
        >
          <h3 className="text-lg font-semibold mb-4">Set WIP Limit</h3>
          <p className="text-sm text-gray-600 mb-4">
            Set the maximum number of tasks allowed in the{' '}
            <strong>
              {statuses.find((s) => s.name === editingWipStatus)?.display_name}
            </strong>{' '}
            column. Tasks exceeding this limit will show a warning.
          </p>
          <input
            type="number"
            min="0"
            value={tempWipValue}
            onChange={(e) => setTempWipValue(e.target.value)}
            placeholder="Enter limit (0 for no limit)"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
            autoFocus
          />
          <div className="flex justify-end gap-2">
            <button
              onClick={() => {
                setEditingWipStatus(null);
                setShowWipModal(false);
              }}
              className="px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => {
                const limit = parseInt(tempWipValue, 10);
                if (!isNaN(limit)) {
                  handleSetWipLimit(editingWipStatus, limit);
                }
              }}
              className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              Save
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}
