import React, { useEffect, useState, useRef } from "react";
import { createPortal } from "react-dom";
import { Task } from "../../models/Task";
import {
  compareDatesInTimezone,
  getToday,
  getTomorrow,
  getYesterday,
  getNextWeek,
  isPast,
  getNextMonday,
  isFriday,
} from "../../utils/dates";
import { useTaskDropdown } from "../../hooks/useTaskDropdown";
import { useOptimisticTaskUpdate } from "../../hooks/useOptimisticTaskUpdate";
import { useAuth } from "../../contexts/AuthContext";

interface TaskDueDateDisplayProps {
  task: Task;
  setTask: (task: Task) => void;
  saveOnChange: boolean;
}

export function TaskDueDateDisplay({
  task,
  setTask,
  saveOnChange,
}: TaskDueDateDisplayProps) {
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";
  const dropdown = useTaskDropdown();
  const { updateTask } = useOptimisticTaskUpdate({
    task,
    setTask,
    saveOnChange,
    errorMessagePrefix: "Failed to update task due date",
  });

  const triggerRef = useRef<HTMLSpanElement>(null);
  const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number } | null>(null);

  const [displayText, setDisplayText] = useState<string>("");
  const [selectedDate, setSelectedDate] = useState<string>(
    task.due_date ? task.due_date.toISOString().substr(0, 10) : "",
  );

  async function handleDueDateChange(e: React.ChangeEvent<HTMLInputElement>) {
    const inputValue = e.target.value;
    if (!inputValue) return;

    const [year, month, day] = inputValue.split("-").map(Number);
    const newDate = new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0));

    setSelectedDate(inputValue);
    await updateTask({ ...task, due_date: newDate });
    dropdown.close();
    setSelectedDate("");
  }

  const getDisplayColor = () => {
    if (!task.is_complete && task.due_date && isPast(task.due_date, userTimezone)) {
      return "#DC2626"; // darker red for overdue deadline
    }
    switch (displayText) {
      case "Due Today":
        return "#DC2626"; // red - urgent
      case "Due Tomorrow":
        return "#F59E0B"; // amber/orange
      case "Overdue":
        return "#DC2626"; // red
      default:
        return "#7C3AED"; // purple for deadlines
    }
  };

  const getDisplayIcon = () => {
    if (!task.is_complete && task.due_date && isPast(task.due_date, userTimezone)) {
      return "!";
    }
    switch (displayText) {
      case "Due Today":
        return "!";
      case "Due Tomorrow":
        return "!";
      case "No Deadline":
        return "";
      default:
        return "";
    }
  };

  async function setDate(dateSetter: () => Date) {
    await updateTask({ ...task, due_date: dateSetter() });
    dropdown.close();
    setSelectedDate("");
  }

  async function setNoDate() {
    await updateTask({ ...task, due_date: null });
    dropdown.close();
    setSelectedDate("");
  }

  function updateDisplayText() {
    if (task.due_date === null) {
      setDisplayText("No Deadline");
      return;
    }

    const compareDate = (date1: Date | null, date2: Date | null) =>
      compareDatesInTimezone(date1, date2, userTimezone);

    let isToday = compareDate(task.due_date, getToday(userTimezone));
    let isTomorrow = compareDate(task.due_date, getTomorrow(userTimezone));
    let isYesterday = compareDate(task.due_date, getYesterday(userTimezone));

    if (isToday) {
      setDisplayText("Due Today");
    } else if (isTomorrow) {
      setDisplayText("Due Tomorrow");
    } else if (isYesterday || isPast(task.due_date, userTimezone)) {
      setDisplayText("Overdue");
    } else if (task.due_date) {
      setDisplayText("Due " + task.due_date.toLocaleDateString(undefined, { timeZone: userTimezone }));
    }
  }

  useEffect(() => {
    updateDisplayText();
  }, [task]);

  // Close dropdown when clicking outside (portal mode)
  useEffect(() => {
    if (!dropdown.isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (triggerRef.current && !triggerRef.current.contains(e.target as Node)) {
        dropdown.close();
      }
    };

    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [dropdown.isOpen, dropdown]);

  // Calculate position when opening dropdown
  const handleToggle = (e: React.MouseEvent) => {
    if (!dropdown.isOpen && triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      setDropdownPosition({ top: rect.bottom + 4, left: rect.left });
    } else {
      setDropdownPosition(null);
    }
    dropdown.toggle(e);
  };

  // Keyboard navigation for dropdown menu
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!dropdown.isOpen) return;
    if (e.key === "Escape") {
      e.preventDefault();
      dropdown.close();
    }
  };

  const color = getDisplayColor();
  const icon = getDisplayIcon();

  return (
    <div className="relative inline-block">
      <span
        ref={triggerRef}
        onClick={handleToggle}
        className="cursor-pointer inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium transition-colors hover:opacity-80"
        style={{
          backgroundColor: color + "20",
          color: color,
          border: `1px solid ${color}40`,
        }}
      >
        {icon && <span>{icon}</span>}
        <span>{displayText}</span>
      </span>

      {dropdown.isOpen && dropdownPosition && createPortal(
        <div
          className="fixed z-[1001] bg-white rounded-md shadow-lg py-1 min-w-[160px] border border-gray-200"
          style={{
            top: `${dropdownPosition.top}px`,
            left: `${dropdownPosition.left}px`,
          }}
          onClick={(e) => e.stopPropagation()}
          onKeyDown={handleKeyDown}
        >
          <div className="flex flex-col">
            <button
              onClick={() => setNoDate()}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
            >
              No Deadline
            </button>
            <button
              onClick={() => setDate(() => getToday(userTimezone))}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
            >
              Today
            </button>
            <button
              onClick={() => setDate(() => getTomorrow(userTimezone))}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
            >
              Tomorrow
            </button>
            {isFriday(userTimezone) && (
              <button
                onClick={() => setDate(() => getNextMonday(userTimezone))}
                className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
              >
                Next Monday
              </button>
            )}
            <button
              onClick={() => setDate(() => getNextWeek(userTimezone))}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
            >
              Next Week
            </button>
            <div className="border-t mt-1 pt-1 px-2">
              <input
                aria-label="Due Date"
                type="date"
                className="w-full px-2 py-1.5 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-purple-500"
                value={selectedDate}
                onChange={handleDueDateChange}
              />
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  );
}
