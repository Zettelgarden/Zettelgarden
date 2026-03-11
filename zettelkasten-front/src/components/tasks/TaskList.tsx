import React, { useMemo } from "react";
import { Task } from "../../models/Task";
import { TaskNestedGroup } from "./TaskNestedGroup";
import { TaskListItem } from "./TaskListItem";
import { SubtaskDisplayMode } from "../../hooks/useSubtaskDisplayMode";

interface TaskListProps {
  tasks: Task[];
  onTagClick: (tag: string) => void;
  hideMatrixTags?: boolean;
  selectMode?: boolean;
  selectedTaskIds?: Set<number>;
  onTaskSelect?: (taskId: number) => void;
  onTaskClick?: (taskId: number) => void;
  subtaskMode?: SubtaskDisplayMode;
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
}: TaskListProps) {
  // Separate root tasks from subtasks and build parent lookup
  const { rootTasks, subtasksByParent, taskById } = useMemo(() => {
    const rootTasks: Task[] = [];
    const subtasksByParent: Record<number, Task[]> = {};
    const taskById: Record<number, Task> = {};

    tasks.forEach((task) => {
      taskById[task.id] = task;

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

    return { rootTasks, subtasksByParent, taskById };
  }, [tasks]);

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

  // Flat mode: show all tasks with parent badges for subtasks
  if (subtaskMode === 'flat') {
    return (
      <ul className="divide-y divide-slate-200">
        {tasks.map((task) => {
          const parentTask = task.parent_task_id ? taskById[task.parent_task_id] : undefined;

          return (
            <li key={task.id} className="py-1">
              <TaskListItem
                task={task}
                onTagClick={onTagClick}
                hideMatrixTags={hideMatrixTags}
                selectMode={selectMode}
                isSelected={selectedTaskIds.has(task.id)}
                onSelect={() => onTaskSelect?.(task.id)}
                parentTask={parentTask}
              />
            </li>
          );
        })}
      </ul>
    );
  }

  // Nested mode (default): current behavior with TaskNestedGroup
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
                onTaskClick={onTaskClick || (() => {})}
                selectMode={selectMode}
                selectedTaskIds={selectedTaskIds}
                onTaskSelect={onTaskSelect}
                hideMatrixTags={hideMatrixTags}
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
}
