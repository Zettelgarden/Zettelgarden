import React, { useState } from "react";
import { Task, TaskStatus } from "../../models/Task";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";

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
  const [showStatusMenu, setShowStatusMenu] = useState<boolean>(false);

  // Get display text and color based on status
  const getStatusDisplay = (status: TaskStatus) => {
    switch (status) {
      case "todo":
        return { text: "To Do", color: "#6B7280", icon: "⭕" }; // Gray
      case "in_progress":
        return { text: "In Progress", color: "#3B82F6", icon: "🔄" }; // Blue
      case "blocked":
        return { text: "Blocked", color: "#EF4444", icon: "🚫" }; // Red
      case "done":
        return { text: "Done", color: "#10B981", icon: "✅" }; // Green
      default:
        return { text: "To Do", color: "#6B7280", icon: "⭕" };
    }
  };

  const statusDisplay = getStatusDisplay(task.status);

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

  async function setStatus(status: TaskStatus) {
    let editedTask = { ...task, status };
    // Sync is_complete with status
    if (status === "done") {
      editedTask.is_complete = true;
    } else {
      editedTask.is_complete = false;
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
    <div className="relative inline-block">
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
          {(["todo", "in_progress", "blocked", "done"] as TaskStatus[]).map((status) => {
            const display = getStatusDisplay(status);
            return (
              <div
                key={status}
                className="px-3 py-2 hover:bg-gray-100 cursor-pointer flex items-center gap-2 text-sm"
                onClick={() => setStatus(status)}
                style={{ color: display.color }}
              >
                <span>{display.icon}</span>
                <span>{display.text}</span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
