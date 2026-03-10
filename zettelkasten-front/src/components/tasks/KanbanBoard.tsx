import React, { useState, useCallback, useEffect, useRef } from "react";
import { DragDropContext, Droppable, Draggable, DropResult } from "@hello-pangea/dnd";
import { Task } from "../../models/Task";
import { TaskListItem } from "./TaskListItem";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { useStatus } from "../../contexts/StatusContext";
import { StatusManagement } from "../settings/StatusManagement";
import { useDialogState } from "../../contexts/DialogStateContext";
import { TaskHoverCard } from "./TaskHoverCard";

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
  enabled: boolean = true
) {
  const [focusedCard, setFocusedCard] = useState<FocusedCard | null>(null);
  const { setSelectedTaskId, setShowTaskDialog } = useDialogState();
  const boardRef = useRef<HTMLDivElement>(null);

  // Get the current card count for a column, memoized
  const getCardCountMemo = useCallback(getCardCount, [columnCount, getCardCount]);

  // Focus the board container when a card is focused
  useEffect(() => {
    if (focusedCard && boardRef.current) {
      boardRef.current.focus();
    }
  }, [focusedCard]);

  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    if (!enabled) return;

    // Ignore if we're inside an input field
    const target = event.target as HTMLElement;
    if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable) {
      return;
    }

    switch (event.key) {
      case "ArrowUp":
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
            cardIndex: prev.cardIndex > 0 ? prev.cardIndex - 1 : upCardCount - 1,
          };
        });
        break;

      case "ArrowDown":
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
            cardIndex: prev.cardIndex < downCardCount - 1 ? prev.cardIndex + 1 : 0,
          };
        });
        break;

      case "ArrowLeft":
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
          const newLeftCol = prev.columnIndex > 0 ? prev.columnIndex - 1 : columnCount - 1;
          const leftCardCount = getCardCountMemo(newLeftCol);
          if (leftCardCount === 0) {
            // Skip empty columns
            for (let i = 1; i < columnCount; i++) {
              const checkCol = (newLeftCol - i + columnCount) % columnCount;
              const checkCount = getCardCountMemo(checkCol);
              if (checkCount > 0) {
                return { columnIndex: checkCol, cardIndex: Math.min(prev.cardIndex, checkCount - 1) };
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

      case "ArrowRight":
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
          const newRightCol = prev.columnIndex < columnCount - 1 ? prev.columnIndex + 1 : 0;
          const rightCardCount = getCardCountMemo(newRightCol);
          if (rightCardCount === 0) {
            // Skip empty columns
            for (let i = 1; i < columnCount; i++) {
              const checkCol = (newRightCol + i) % columnCount;
              const checkCount = getCardCountMemo(checkCol);
              if (checkCount > 0) {
                return { columnIndex: checkCol, cardIndex: Math.min(prev.cardIndex, checkCount - 1) };
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

      case "Enter":
        event.preventDefault();
        if (focusedCard) {
          const taskId = getTaskId(focusedCard.columnIndex, focusedCard.cardIndex);
          if (taskId !== null) {
            setSelectedTaskId(taskId);
            setShowTaskDialog(true);
          }
        }
        break;

      case "Escape":
        event.preventDefault();
        setFocusedCard(null);
        break;

      case "j":
      case "J":
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
            cardIndex: prev.cardIndex < jCardCount - 1 ? prev.cardIndex + 1 : 0,
          };
        });
        break;

      case "k":
      case "K":
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
            cardIndex: prev.cardIndex > 0 ? prev.cardIndex - 1 : kCardCount - 1,
          };
        });
        break;

      case "h":
      case "H":
        // Vim-style left (when not in input)
        event.preventDefault();
        setFocusedCard((prev) => {
          if (!prev) return null;
          const newCol = prev.columnIndex > 0 ? prev.columnIndex - 1 : columnCount - 1;
          const hCardCount = getCardCountMemo(newCol);
          if (hCardCount === 0) return prev;
          return { columnIndex: newCol, cardIndex: Math.min(prev.cardIndex, hCardCount - 1) };
        });
        break;

      case "l":
      case "L":
        // Vim-style right (when not in input)
        event.preventDefault();
        setFocusedCard((prev) => {
          if (!prev) return null;
          const newCol = prev.columnIndex < columnCount - 1 ? prev.columnIndex + 1 : 0;
          const lCardCount = getCardCountMemo(newCol);
          if (lCardCount === 0) return prev;
          return { columnIndex: newCol, cardIndex: Math.min(prev.cardIndex, lCardCount - 1) };
        });
        break;
    }
  }, [enabled, columnCount, getCardCountMemo, getTaskId, focusedCard, setSelectedTaskId, setShowTaskDialog]);

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
}

export function KanbanBoard({ tasks, onTagClick, onAddTaskWithStatus, selectMode = false, selectedTaskIds = new Set(), onTaskSelect }: KanbanBoardProps) {
  const { setRefreshTasks } = useTaskContext();
  const { statuses, getStatusByName } = useStatus();
  const [showStatusManagement, setShowStatusManagement] = useState(false);

  // Group tasks by status
  const tasksByStatus = tasks.reduce((acc, task) => {
    const status = task.status || (statuses.find(s => s.is_default)?.name || "todo");
    if (!acc[status]) {
      acc[status] = [];
    }
    acc[status].push(task);
    return acc;
  }, {} as Record<string, Task[]>);

  // Create a stable array of tasks per column for keyboard navigation
  const columnTasksArrays = statuses.map((status) => tasksByStatus[status.name] || []);

  // Helper functions for keyboard navigation
  const getCardCount = useCallback((columnIndex: number) => {
    return columnTasksArrays[columnIndex]?.length || 0;
  }, [columnTasksArrays]);

  const getTaskId = useCallback((columnIndex: number, cardIndex: number) => {
    const tasks = columnTasksArrays[columnIndex];
    return tasks?.[cardIndex]?.id ?? null;
  }, [columnTasksArrays]);

  // Keyboard navigation hook
  const { focusedCard, handleKeyDown, clearFocus, boardRef } = useKanbanKeyboardNavigation(
    statuses.length,
    getCardCount,
    getTaskId,
    !selectMode // Disable keyboard nav in select mode
  );

  const onDragEnd = async (result: DropResult) => {
    if (!result.destination) return;

    const sourceStatus = result.source.droppableId;
    const destStatus = result.destination.droppableId;

    // No change if dropped in the same column
    if (sourceStatus === destStatus) return;

    const draggedId = result.draggableId;
    const task = tasks.find(t => t.id.toString() === draggedId);
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
      if (!("error" in response)) {
        setRefreshTasks(true);
      }
    } catch (err) {
      console.error("Failed to save updated task after drag-and-drop:", err);
    }
  };

  return (
    <>
      {/* Header with Manage Statuses Button */}
      <div className="mb-4 flex justify-between items-center">
        <h2 className="text-xl font-semibold text-gray-900">Task Board</h2>
        <button
          onClick={() => setShowStatusManagement(true)}
          className="flex items-center gap-2 px-4 py-2 text-sm bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          Manage Statuses
        </button>
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
          const columnTasks = tasksByStatus[column.name] || [];

          return (
            <div
              key={column.name}
              className="flex flex-col bg-gray-50 rounded-lg border border-gray-200 flex-shrink-0 min-h-[300px]"
              style={{ minWidth: "400px", maxWidth: "400px" }}
            >
              {/* Column Header */}
              <div
                className="p-3 rounded-t-lg border-b-2"
                style={{
                  backgroundColor: column.color + "15",
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
                      <span className="text-lg font-bold leading-none pb-1">+</span>
                    </button>
                    <span
                      className="text-xs font-medium px-2 py-0.5 rounded-full"
                      style={{
                        backgroundColor: column.color + "20",
                        color: column.color,
                      }}
                    >
                      {columnTasks.length}
                    </span>
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
                      snapshot.isDraggingOver
                        ? "bg-blue-50"
                        : ""
                    }`}
                  >
                    {columnTasks.length > 0 ? (
                      columnTasks.map((task, index) => {
                        const isFocused = focusedCard?.columnIndex === columnIndex && focusedCard?.cardIndex === index;
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
                                  className={`bg-white rounded border shadow-sm transition-all ${
                                    dragSnapshot.isDragging
                                      ? "border-blue-400 shadow-lg"
                                      : isFocused
                                      ? "border-blue-500 shadow-md ring-2 ring-blue-200"
                                      : "border-gray-200 hover:shadow-md"
                                  }`}
                                >
                                  <TaskListItem
                                    task={task}
                                    onTagClick={onTagClick}
                                    hideMatrixTags={false}
                                    selectMode={selectMode}
                                    isSelected={selectedTaskIds.has(task.id)}
                                    onSelect={() => onTaskSelect?.(task.id)}
                                  />
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
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg w-full max-w-5xl max-h-[90vh] overflow-y-auto relative">
            {/* Close button */}
            <button
              onClick={() => setShowStatusManagement(false)}
              className="sticky top-0 right-0 float-right m-4 p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors z-10"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>

            {/* StatusManagement component */}
            <div className="p-6">
              <StatusManagement />
            </div>
          </div>
        </div>
      )}
    </>
  );
}
