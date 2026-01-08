import React, { useState } from "react";
import { Task } from "../../models/Task";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { useStatus } from "../../contexts/StatusContext";

interface TaskStatusDisplayProps {
  task: Task;
  setTask: (task: Task) => void;
  saveOnChange: boolean;
}

export function TaskStatusDisplay({
  task,
  setTask,
  saveOnChange,
}: TaskStatusDisplayProps) {
  const { updateTask: updateTaskInContext } = useTaskContext();
  const { statuses, getStatusByName } = useStatus();
  const [showStatusMenu, setShowStatusMenu] = useState<boolean>(false);

  // Get status config from dynamic statuses
  const currentStatus = getStatusByName(task.status);
  const statusDisplay = currentStatus
    ? {
        text: currentStatus.display_name,
        color: currentStatus.color,
        icon: currentStatus.icon,
      }
    : { text: "Unknown", color: "#6B7280", icon: "⭕" };

  async function updateTask(editedTask: Task) {
    if (saveOnChange) {
      // Optimistic update: update UI immediately
      updateTaskInContext(editedTask);

      // Send update to server in background
      try {
        const response = await saveExistingTask(editedTask);
        if ("error" in response) {
          // Rollback on error
          updateTaskInContext(task);
          console.error("Failed to update task status:", response.error);
        }
      } catch (error) {
        // Rollback on network error
        updateTaskInContext(task);
        console.error("Failed to update task status:", error);
      }
    } else {
      setTask(editedTask);
    }
  }

  async function setStatus(statusName: string) {
    const statusConfig = getStatusByName(statusName);
    let editedTask = { ...task, status: statusName };

    // Sync is_complete with status based on is_complete_state
    if (statusConfig) {
      editedTask.is_complete = statusConfig.is_complete_state;
    }

    updateTask(editedTask);
    setShowStatusMenu(false);
  }

  // Close menu when clicking outside
  React.useEffect(() => {
    const handleClickOutside = () => setShowStatusMenu(false);
    if (showStatusMenu) {
      document.addEventListener("click", handleClickOutside);
      return () => document.removeEventListener("click", handleClickOutside);
    }
  }, [showStatusMenu]);

  return (
    <div className="relative inline-block pr-2">
      <span
        onClick={(e) => {
          e.stopPropagation();
          setShowStatusMenu(!showStatusMenu);
        }}
        className="cursor-pointer inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium transition-colors hover:opacity-80"
        style={{
          backgroundColor: statusDisplay.color + "20",
          color: statusDisplay.color,
          border: `1px solid ${statusDisplay.color}40`,
        }}
      >
        <span>{statusDisplay.icon}</span>
        <span>{statusDisplay.text}</span>
      </span>

      {showStatusMenu && (
        <div
          className="absolute z-20 mt-1 bg-white rounded-md shadow-lg py-1 min-w-[140px] border border-gray-200"
          onClick={(e) => e.stopPropagation()}
        >
          {statuses.map((status) => (
            <div
              key={status.id}
              className="px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2 text-sm"
              onClick={() => setStatus(status.name)}
              style={{ color: status.color }}
            >
              <span>{status.icon}</span>
              <span>{status.display_name}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
