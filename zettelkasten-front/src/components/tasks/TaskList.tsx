import React, { useMemo } from "react";
import { Task } from "../../models/Task";
import { TaskNestedGroup } from "./TaskNestedGroup";
import { TaskListItem } from "./TaskListItem";
import { useTaskContext } from "../../contexts/TaskContext";

interface TaskListProps {
  tasks: Task[];
  onTagClick: (tag: string) => void;
  hideMatrixTags?: boolean;
  selectMode?: boolean;
  selectedTaskIds?: Set<number>;
  onTaskSelect?: (taskId: number) => void;
  onTaskClick?: (taskId: number) => void;
}

export function TaskList({
  tasks,
  onTagClick,
  hideMatrixTags = false,
  selectMode = false,
  selectedTaskIds = new Set(),
  onTaskSelect,
  onTaskClick,
}: TaskListProps) {
  const { setRefreshTasks } = useTaskContext();

  // Separate root tasks from subtasks
  const { rootTasks, subtasksByParent } = useMemo(() => {
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
