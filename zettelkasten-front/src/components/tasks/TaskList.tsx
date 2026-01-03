import React from "react";

import { TaskListItem } from "./TaskListItem";
import { Task } from "../../models/Task";
import { useTaskContext } from "../../contexts/TaskContext";

interface TaskListProps {
  tasks: Task[];
  onTagClick: (tag: string) => void;
  hideMatrixTags?: boolean;
  selectMode?: boolean;
  selectedTaskIds?: Set<number>;
  onTaskSelect?: (taskId: number) => void;
}

export function TaskList({
  tasks,
  onTagClick,
  hideMatrixTags = false,
  selectMode = false,
  selectedTaskIds = new Set(),
  onTaskSelect,
}: TaskListProps) {
  const { setRefreshTasks } = useTaskContext();
  return (
    <ul>
      {tasks.map((task, index) => (
        <li key={task.id} className="pb-0 mb border-b border-slate-200 last:border-0">
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
