import React, { useState, useCallback } from "react";
import { DragDropContext, Droppable, Draggable, DropResult } from "@hello-pangea/dnd";
import { Task } from "../../models/Task";
import { TaskNestedGroup } from "./TaskNestedGroup";
import { TaskListItem } from "./TaskListItem";
import { SubtaskDisplayMode } from "../../hooks/useSubtaskDisplayMode";
import { reorderTasks } from "../../api/tasks";

interface TaskListProps {
  tasks: Task[];
  onTagClick: (tag: string) => void;
  hideMatrixTags?: boolean;
  selectMode?: boolean;
  selectedTaskIds?: Set<number>;
  onTaskSelect?: (taskId: number) => void;
  onTaskClick?: (taskId: number) => void;
  subtaskMode?: SubtaskDisplayMode;
  /** When true, enable drag-and-drop manual reordering */
  manualSort?: boolean;
  /** Callback after reorder is persisted to server */
  onReorder?: () => void;
}

export function TaskList({
  tasks,
  onTagClick,
  hideMatrixTags = false,
  selectMode = false,
  selectedTaskIds = new Set(),
  onTaskSelect,
  onTaskClick,
  subtaskMode = 'nested',
  manualSort = false,
  onReorder,
}: TaskListProps) {
  // Local order state for optimistic updates during drag
  const [localOrder, setLocalOrder] = useState<number[] | null>(null);

  // Compute the task display order
  const orderedTasks = React.useMemo(() => {
    if (localOrder) {
      // Use local order during/after drag
      const taskMap = new Map(tasks.map(t => [t.id, t]));
      const ordered: Task[] = [];
      for (const id of localOrder) {
        const task = taskMap.get(id);
        if (task) ordered.push(task);
      }
      // Add any tasks not in localOrder (new tasks, etc.)
      for (const task of tasks) {
        if (!localOrder.includes(task.id)) {
          ordered.push(task);
        }
      }
      return ordered;
    }
    return tasks;
  }, [tasks, localOrder]);

  // Clear local order when the underlying tasks change (e.g., filter change)
  React.useEffect(() => {
    setLocalOrder(null);
  }, [tasks.map(t => t.id).join(',')]);

  const handleDragEnd = useCallback(async (result: DropResult) => {
    if (!result.destination) return;

    const sourceIndex = result.source.index;
    const destIndex = result.destination.index;
    if (sourceIndex === destIndex) return;

    // Get the root tasks for ordering (same logic as the render below)
    const rootTasks = orderedTasks.filter(t => !t.parent_task_id);
    if (sourceIndex >= rootTasks.length || destIndex >= rootTasks.length) return;

    // Optimistic reorder: update local state immediately
    const newRootOrder = [...rootTasks];
    const [moved] = newRootOrder.splice(sourceIndex, 1);
    newRootOrder.splice(destIndex, 0, moved);

    // Build new full order (root tasks in new order + subtasks in original positions)
    const newFullOrder: number[] = [];
    const subtasksByParent: Record<number, Task[]> = {};
    orderedTasks.forEach(t => {
      if (t.parent_task_id) {
        if (!subtasksByParent[t.parent_task_id]) subtasksByParent[t.parent_task_id] = [];
        subtasksByParent[t.parent_task_id].push(t);
      }
    });

    for (const rootTask of newRootOrder) {
      newFullOrder.push(rootTask.id);
      const children = subtasksByParent[rootTask.id] || [];
      for (const child of children) {
        newFullOrder.push(child.id);
      }
    }
    // Add any remaining tasks not in rootTasks
    for (const t of orderedTasks) {
      if (!newFullOrder.includes(t.id)) {
        newFullOrder.push(t.id);
      }
    }

    setLocalOrder(newFullOrder);

    // Persist to server
    const orders = newFullOrder.map((id, index) => ({ id, sort_order: index }));
    try {
      await reorderTasks(orders);
      onReorder?.();
    } catch (err) {
      console.error("Failed to persist task reorder:", err);
      // Revert on error
      setLocalOrder(null);
    }
  }, [orderedTasks, onReorder]);

  // Separate root tasks from subtasks and build parent lookup
  const { rootTasks, subtasksByParent, taskById } = React.useMemo(() => {
    const rootTasks: Task[] = [];
    const subtasksByParent: Record<number, Task[]> = {};
    const taskById: Record<number, Task> = {};

    orderedTasks.forEach((task) => {
      taskById[task.id] = task;

      if (task.parent_task_id) {
        if (!subtasksByParent[task.parent_task_id]) {
          subtasksByParent[task.parent_task_id] = [];
        }
        subtasksByParent[task.parent_task_id].push(task);
      } else {
        rootTasks.push(task);
      }
    });

    return { rootTasks, subtasksByParent, taskById };
  }, [orderedTasks]);

  // Render a single task item (used by both draggable and non-draggable modes)
  const renderTaskItem = (task: Task, parentTask?: Task) => (
    <TaskListItem
      task={task}
      onTagClick={onTagClick}
      hideMatrixTags={hideMatrixTags}
      selectMode={selectMode}
      isSelected={selectedTaskIds.has(task.id)}
      onSelect={() => onTaskSelect?.(task.id)}
      parentTask={parentTask}
    />
  );

  // Hidden mode: only show root tasks
  if (subtaskMode === 'hidden') {
    if (manualSort) {
      return (
        <DragDropContext onDragEnd={handleDragEnd}>
          <Droppable droppableId="task-list">
            {(provided) => (
              <ul
                ref={provided.innerRef}
                {...provided.droppableProps}
                className="divide-y divide-slate-200"
              >
                {rootTasks.map((task, index) => (
                  <Draggable key={task.id} draggableId={`task-${task.id}`} index={index}>
                    {(provided, snapshot) => (
                      <li
                        ref={provided.innerRef}
                        {...provided.draggableProps}
                        className={`py-1 ${snapshot.isDragging ? 'bg-blue-50 shadow-lg rounded' : ''}`}
                      >
                        <div className="flex items-center">
                          <span
                            {...provided.dragHandleProps}
                            className="cursor-grab active:cursor-grabbing text-slate-300 hover:text-slate-500 px-1 select-none text-sm"
                            title="Drag to reorder"
                          >
                            ⠿
                          </span>
                          <div className="flex-grow">
                            {renderTaskItem(task)}
                          </div>
                        </div>
                      </li>
                    )}
                  </Draggable>
                ))}
                {provided.placeholder}
              </ul>
            )}
          </Droppable>
        </DragDropContext>
      );
    }

    return (
      <ul className="divide-y divide-slate-200">
        {rootTasks.map((task) => (
          <li key={task.id} className="py-1">
            {renderTaskItem(task)}
          </li>
        ))}
      </ul>
    );
  }

  // Flat mode: show all tasks with parent badges for subtasks
  if (subtaskMode === 'flat') {
    // Flat mode doesn't support manual sort (ordering mixes parents and children)
    return (
      <ul className="divide-y divide-slate-200">
        {orderedTasks.map((task) => {
          const parentTask = task.parent_task_id ? taskById[task.parent_task_id] : undefined;
          return (
            <li key={task.id} className="py-1">
              {renderTaskItem(task, parentTask)}
            </li>
          );
        })}
      </ul>
    );
  }

  // Nested mode (default)
  if (manualSort) {
    return (
      <DragDropContext onDragEnd={handleDragEnd}>
        <Droppable droppableId="task-list">
          {(provided) => (
            <ul
              ref={provided.innerRef}
              {...provided.droppableProps}
              className="divide-y divide-slate-200"
            >
              {rootTasks.map((task, index) => {
                const subtasks = subtasksByParent[task.id] || [];

                return (
                  <Draggable key={task.id} draggableId={`task-${task.id}`} index={index}>
                    {(provided, snapshot) => (
                      <li
                        ref={provided.innerRef}
                        {...provided.draggableProps}
                        className={`py-1 ${snapshot.isDragging ? 'bg-blue-50 shadow-lg rounded' : ''}`}
                      >
                        <div className="flex items-center">
                          <span
                            {...provided.dragHandleProps}
                            className="cursor-grab active:cursor-grabbing text-slate-300 hover:text-slate-500 px-1 select-none self-start mt-1 text-sm"
                            title="Drag to reorder"
                          >
                            ⠿
                          </span>
                          <div className="flex-grow">
                            {subtasks.length > 0 ? (
                              <TaskNestedGroup
                                task={{ ...task, subtasks }}
                                onTagClick={onTagClick}
                                onTaskClick={onTaskClick || (() => {})}
                                selectMode={selectMode}
                                selectedTaskIds={selectedTaskIds}
                                onTaskSelect={onTaskSelect}
                                hideMatrixTags={hideMatrixTags}
                              />
                            ) : (
                              renderTaskItem(task)
                            )}
                          </div>
                        </div>
                      </li>
                    )}
                  </Draggable>
                );
              })}
              {provided.placeholder}
            </ul>
          )}
        </Droppable>
      </DragDropContext>
    );
  }

  // Default nested mode (no drag)
  return (
    <ul className="divide-y divide-slate-200">
      {rootTasks.map((task) => {
        const subtasks = subtasksByParent[task.id] || [];

        if (subtasks.length > 0) {
          return (
            <li key={task.id} className="py-1">
              <TaskNestedGroup
                task={{ ...task, subtasks }}
                onTagClick={onTagClick}
                onTaskClick={onTaskClick || (() => {})}
                selectMode={selectMode}
                selectedTaskIds={selectedTaskIds}
                onTaskSelect={onTaskSelect}
                hideMatrixTags={hideMatrixTags}
              />
            </li>
          );
        }

        return (
          <li key={task.id} className="py-1">
            {renderTaskItem(task)}
          </li>
        );
      })}
    </ul>
  );
}
