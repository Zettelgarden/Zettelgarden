import React, { useRef } from "react";
import { Task } from "../../models/Task";
import { useTaskDropdown } from "../../hooks/useTaskDropdown";
import { useOptimisticTaskUpdate } from "../../hooks/useOptimisticTaskUpdate";
import { TaskDropdown } from "./TaskDropdown";

interface TaskPriorityDisplayProps {
  task: Task;
  setTask: (task: Task) => void;
  saveOnChange: boolean;
}

const PRIORITIES = [
  { value: "A" as const, text: "Priority A", color: "#EF4444", icon: "🔴" },
  { value: "B" as const, text: "Priority B", color: "#F59E0B", icon: "🟠" },
  { value: "C" as const, text: "Priority C", color: "#3B82F6", icon: "🔵" },
  { value: null, text: "No Priority", color: "#6B7280", icon: "○" },
];

export function TaskPriorityDisplay({
  task,
  setTask,
  saveOnChange,
}: TaskPriorityDisplayProps) {
  const dropdown = useTaskDropdown();
  const { updateTask } = useOptimisticTaskUpdate({
    task,
    setTask,
    saveOnChange,
    errorMessagePrefix: "Failed to update task priority",
  });
  const triggerRef = useRef<HTMLSpanElement>(null);

  // Get display text and color based on priority
  const priorityDisplay = React.useMemo(() => {
    const found = PRIORITIES.find((p) => p.value === task.priority);
    return (
      found || {
        text: task.priority || "No Priority",
        color: "#6B7280",
        icon: "○",
      }
    );
  }, [task.priority]);

  async function setPriority(priority: string | null) {
    const editedTask = { ...task, priority };
    await updateTask(editedTask);
    dropdown.close();
  }

  return (
    <TaskDropdown
      isOpen={dropdown.isOpen}
      onToggle={dropdown.toggle}
      onClose={dropdown.close}
      display={priorityDisplay}
      triggerRef={triggerRef}
      usePortal={true}
    >
      {PRIORITIES.map((priority) => (
        <div
          key={priority.value ?? "none"}
          className="px-4 py-3 min-h-[44px] hover:bg-gray-100 cursor-pointer flex items-center gap-2 text-sm"
          onClick={() => setPriority(priority.value)}
          style={{ color: priority.color }}
        >
          <span>{priority.icon}</span>
          <span>{priority.text}</span>
        </div>
      ))}
    </TaskDropdown>
  );
}
