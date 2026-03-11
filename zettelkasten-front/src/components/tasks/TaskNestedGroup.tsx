import React, { useState } from "react";
import { Task } from "../../models/Task";
import { TaskListItem } from "./TaskListItem";

interface TaskNestedGroupProps {
  task: Task;
  onTagClick: (tag: string) => void;
  /** @deprecated TaskListItem handles clicks internally via useDialogState. Reserved for future use. */
  onTaskClick: (taskId: number) => void;
  collapsed?: boolean;
  selectMode?: boolean;
  selectedTaskIds?: Set<number>;
  onTaskSelect?: (taskId: number) => void;
  hideMatrixTags?: boolean;
}

export function TaskNestedGroup({
  task,
  onTagClick,
  onTaskClick,
  collapsed = false,
  selectMode = false,
  selectedTaskIds = new Set(),
  onTaskSelect,
  hideMatrixTags = false,
}: TaskNestedGroupProps) {
  const [isCollapsed, setIsCollapsed] = useState(collapsed);

  const subtasks = task.subtasks || [];
  const completeCount = subtasks.filter((s) => s.is_complete).length;
  const hasSubtasks = subtasks.length > 0;

  return (
    <div className="task-nested-group">
      {/* Parent Task */}
      <div className="relative">
        <TaskListItem
          task={task}
          onTagClick={onTagClick}
          hideMatrixTags={hideMatrixTags}
          selectMode={selectMode}
          isSelected={selectedTaskIds.has(task.id)}
          onSelect={() => onTaskSelect?.(task.id)}
        />

        {/* Collapse Toggle */}
        {hasSubtasks && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              setIsCollapsed(!isCollapsed);
            }}
            className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-6 w-5 h-5 flex items-center justify-center text-gray-400 hover:text-gray-600 z-10"
            title={isCollapsed ? "Expand subtasks" : "Collapse subtasks"}
            aria-expanded={!isCollapsed}
            aria-label={isCollapsed ? "Expand subtasks" : "Collapse subtasks"}
          >
            <span className="text-xs">{isCollapsed ? "▶" : "▼"}</span>
          </button>
        )}

        {/* Progress Indicator */}
        {hasSubtasks && (
          <span className="ml-2 text-xs text-gray-500">
            ({completeCount}/{subtasks.length})
          </span>
        )}
      </div>

      {/* Nested Subtasks */}
      {hasSubtasks && !isCollapsed && (
        <div className="ml-6 mt-1 border-l-2 border-gray-200 pl-4 space-y-1">
          {subtasks.map((subtask) => (
            <div key={subtask.id} className="relative">
              {/* Tree connector */}
              <div className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-4 w-3 h-0.5 bg-gray-200" />

              <TaskListItem
                task={subtask}
                onTagClick={onTagClick}
                hideMatrixTags={hideMatrixTags}
                selectMode={selectMode}
                isSelected={selectedTaskIds.has(subtask.id)}
                onSelect={() => onTaskSelect?.(subtask.id)}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
