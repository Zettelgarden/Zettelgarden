import React, { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import { Task } from "../../models/Task";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { useStatus } from "../../contexts/StatusContext";
import { useAuth } from "../../contexts/AuthContext";
import { format } from "date-fns-tz";

interface KanbanQuickActionsProps {
  task: Task;
}

const PRIORITY_OPTIONS = [
  { value: "A", label: "High", icon: "🔴" },
  { value: "B", label: "Medium", icon: "🟠" },
  { value: "C", label: "Low", icon: "🔵" },
  { value: null, label: "None", icon: "○" },
];

export function KanbanQuickActions({ task }: KanbanQuickActionsProps) {
  const [showPriorityMenu, setShowPriorityMenu] = useState(false);
  const [showStatusMenu, setShowStatusMenu] = useState(false);
  const [showDatePicker, setShowDatePicker] = useState(false);
  const priorityRef = useRef<HTMLDivElement>(null);
  const statusRef = useRef<HTMLDivElement>(null);
  const dateRef = useRef<HTMLDivElement>(null);
  const { statuses } = useStatus();
  const { updateTask } = useTaskContext();
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";

  // Close menus when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (priorityRef.current && !priorityRef.current.contains(event.target as Node)) {
        setShowPriorityMenu(false);
      }
      if (statusRef.current && !statusRef.current.contains(event.target as Node)) {
        setShowStatusMenu(false);
      }
      if (dateRef.current && !dateRef.current.contains(event.target as Node)) {
        setShowDatePicker(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handlePriorityChange = async (priority: string | null) => {
    const updatedTask = { ...task, priority };
    updateTask(updatedTask);
    setShowPriorityMenu(false);

    try {
      await saveExistingTask(updatedTask);
    } catch (error) {
      console.error("Failed to update priority:", error);
      updateTask(task); // Rollback
    }
  };

  const handleStatusChange = async (status: string) => {
    const statusConfig = statuses.find((s) => s.name === status);
    const updatedTask = {
      ...task,
      status,
      is_complete: statusConfig?.is_complete_state || false,
    };
    updateTask(updatedTask);
    setShowStatusMenu(false);

    try {
      await saveExistingTask(updatedTask);
    } catch (error) {
      console.error("Failed to update status:", error);
      updateTask(task); // Rollback
    }
  };

  const handleDateChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const inputValue = e.target.value;
    if (!inputValue) {
      const updatedTask = { ...task, due_date: null };
      updateTask(updatedTask);
      setShowDatePicker(false);

      try {
        await saveExistingTask(updatedTask);
      } catch (error) {
        console.error("Failed to clear due date:", error);
        updateTask(task);
      }
      return;
    }

    const [year, month, day] = inputValue.split("-").map(Number);
    const newDate = new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0));
    const updatedTask = { ...task, due_date: newDate };
    updateTask(updatedTask);
    setShowDatePicker(false);

    try {
      await saveExistingTask(updatedTask);
    } catch (error) {
      console.error("Failed to update due date:", error);
      updateTask(task);
    }
  };

  const clearDueDate = async () => {
    const updatedTask = { ...task, due_date: null };
    updateTask(updatedTask);
    setShowDatePicker(false);

    try {
      await saveExistingTask(updatedTask);
    } catch (error) {
      console.error("Failed to clear due date:", error);
      updateTask(task);
    }
  };

  const handleToggleComplete = async () => {
    const updatedTask = {
      ...task,
      is_complete: !task.is_complete,
    };
    updateTask(updatedTask);

    try {
      await saveExistingTask(updatedTask);
    } catch (error) {
      console.error("Failed to toggle complete:", error);
      updateTask(task);
    }
  };

  const currentDate = task.due_date
    ? format(new Date(task.due_date), "yyyy-MM-dd", { timeZone: userTimezone })
    : "";

  return (
    <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
      {/* Complete Toggle */}
      <button
        onClick={handleToggleComplete}
        className={`p-1.5 rounded text-xs hover:bg-gray-100 transition-colors ${
          task.is_complete ? "text-green-600" : "text-gray-400"
        }`}
        title={task.is_complete ? "Mark incomplete" : "Mark complete"}
      >
        {task.is_complete ? "✓" : "○"}
      </button>

      {/* Status Menu */}
      <div className="relative" ref={statusRef}>
        <button
          onClick={() => setShowStatusMenu(!showStatusMenu)}
          className="p-1.5 rounded text-xs text-gray-400 hover:bg-gray-100 transition-colors"
          title="Change status"
        >
          📊
        </button>
        {showStatusMenu && createPortal(
          <div
            className="fixed bg-white border border-gray-200 rounded-lg shadow-lg z-50 w-40 max-h-60 overflow-y-auto"
            style={{
              top: statusRef.current?.getBoundingClientRect().bottom || 0,
              left: statusRef.current?.getBoundingClientRect().left || 0,
            }}
          >
            {statuses.map((status) => (
              <button
                key={status.name}
                onClick={() => handleStatusChange(status.name)}
                className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 flex items-center gap-2 ${
                  task.status === status.name ? "bg-blue-50 text-blue-700" : ""
                }`}
              >
                <span>{status.icon}</span>
                <span>{status.display_name}</span>
              </button>
            ))}
          </div>,
          document.body
        )}
      </div>

      {/* Priority Menu */}
      <div className="relative" ref={priorityRef}>
        <button
          onClick={() => setShowPriorityMenu(!showPriorityMenu)}
          className="p-1.5 rounded text-xs text-gray-400 hover:bg-gray-100 transition-colors"
          title="Change priority"
        >
          {task.priority === "A" ? "🔴" : task.priority === "B" ? "🟠" : task.priority === "C" ? "🔵" : "○"}
        </button>
        {showPriorityMenu && createPortal(
          <div
            className="fixed bg-white border border-gray-200 rounded-lg shadow-lg z-50 w-32"
            style={{
              top: priorityRef.current?.getBoundingClientRect().bottom || 0,
              left: priorityRef.current?.getBoundingClientRect().left || 0,
            }}
          >
            {PRIORITY_OPTIONS.map((option) => (
              <button
                key={option.value || "none"}
                onClick={() => handlePriorityChange(option.value)}
                className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-50 flex items-center gap-2 ${
                  task.priority === option.value ? "bg-blue-50 text-blue-700" : ""
                }`}
              >
                <span>{option.icon}</span>
                <span>{option.label}</span>
              </button>
            ))}
          </div>,
          document.body
        )}
      </div>

      {/* Date Picker */}
      <div className="relative" ref={dateRef}>
        <button
          onClick={() => setShowDatePicker(!showDatePicker)}
          className={`p-1.5 rounded text-xs hover:bg-gray-100 transition-colors ${
            task.due_date ? "text-blue-500" : "text-gray-400"
          }`}
          title="Set due date"
        >
          📅
        </button>
        {showDatePicker && createPortal(
          <div
            className="fixed bg-white border border-gray-200 rounded-lg shadow-lg z-50 p-2"
            style={{
              top: dateRef.current?.getBoundingClientRect().bottom || 0,
              left: dateRef.current?.getBoundingClientRect().left || 0,
            }}
          >
            <input
              type="date"
              value={currentDate}
              onChange={handleDateChange}
              className="px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              autoFocus
            />
            {task.due_date && (
              <button
                onClick={clearDueDate}
                className="w-full mt-1 text-xs text-gray-500 hover:text-gray-700"
              >
                Clear date
              </button>
            )}
          </div>,
          document.body
        )}
      </div>
    </div>
  );
}
