import React, { useRef } from "react";
import { Task } from "../../models/Task";
import { useTaskDropdown } from "../../hooks/useTaskDropdown";
import { useOptimisticTaskUpdate } from "../../hooks/useOptimisticTaskUpdate";
import { TaskDropdown } from "./TaskDropdown";
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
  const dropdown = useTaskDropdown();
  const { updateTask } = useOptimisticTaskUpdate({
    task,
    setTask,
    saveOnChange,
    errorMessagePrefix: "Failed to update task status",
  });
  const { statuses, getStatusByName } = useStatus();
  const triggerRef = useRef<HTMLSpanElement>(null);

  // Get status config from dynamic statuses
  const statusDisplay = React.useMemo(() => {
    const currentStatus = getStatusByName(task.status);
    return currentStatus
      ? {
          text: currentStatus.display_name,
          color: currentStatus.color,
          icon: currentStatus.icon,
        }
      : { text: "Unknown", color: "#6B7280", icon: "⭕" };
  }, [task.status, getStatusByName]);

  async function setStatus(statusName: string) {
    const statusConfig = getStatusByName(statusName);
    const editedTask = { ...task, status: statusName };

    // Sync is_complete with status based on is_complete_state
    if (statusConfig) {
      editedTask.is_complete = statusConfig.is_complete_state;
    }

    await updateTask(editedTask);
    dropdown.close();
  }

  return (
    <div className="relative inline-block pr-2">
      <TaskDropdown
        isOpen={dropdown.isOpen}
        onToggle={dropdown.toggle}
        onClose={dropdown.close}
        display={statusDisplay}
        triggerRef={triggerRef}
        usePortal={true}
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
      </TaskDropdown>
    </div>
  );
}
