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
  isRecurringTask,
} from "../../utils/dates";
import { useTaskDropdown } from "../../hooks/useTaskDropdown";
import { useOptimisticTaskUpdate } from "../../hooks/useOptimisticTaskUpdate";
import { useAuth } from "../../contexts/AuthContext";

interface TaskDateDisplayProps {
  task: Task;
  setTask: (task: Task) => void;
  saveOnChange: boolean;
}

export function TaskDateDisplay({
  task,
  setTask,
  saveOnChange,
}: TaskDateDisplayProps) {
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";
  const dropdown = useTaskDropdown();
  const { updateTask } = useOptimisticTaskUpdate({
    task,
    setTask,
    saveOnChange,
    errorMessagePrefix: "Failed to update task date",
  });

  const triggerRef = useRef<HTMLSpanElement>(null);
  const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number } | null>(null);

  const [displayText, setDisplayText] = useState<string>("");
  const [selectedDate, setSelectedDate] = useState<string>(
    task.scheduled_date ? task.scheduled_date.toISOString().substr(0, 10) : "",
  );

  async function handleScheduledDateChange(e: React.ChangeEvent<HTMLInputElement>) {
    const inputValue = e.target.value;
    if (!inputValue) return;

    const [year, month, day] = inputValue.split("-").map(Number);
    const newDate = new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0));

    setSelectedDate(inputValue);
    await updateTask({ ...task, scheduled_date: newDate });
    dropdown.close();
    setSelectedDate("");
  }

  const getDisplayColor = () => {
    if (!task.is_complete && isPast(task.scheduled_date, userTimezone)) {
      return "#EF4444"; // red
    }
    switch (displayText) {
      case "Today":
        return "#10B981"; // green
      case "Tomorrow":
        return "#3B82F6"; // blue
      case "Yesterday":
        return "#EF4444"; // red
      default:
        return "#6B7280"; // gray
    }
  };

  const getDisplayIcon = () => {
    if (!task.is_complete && isPast(task.scheduled_date, userTimezone)) {
      return "⚠️";
    }
    switch (displayText) {
      case "Today":
        return "📅";
      case "Tomorrow":
        return "📆";
      case "No Date":
        return "○";
      default:
        return "📅";
    }
  };

  async function setDate(dateSetter: () => Date) {
    await updateTask({ ...task, scheduled_date: dateSetter() });
    dropdown.close();
    setSelectedDate("");
  }

  async function setNoDate() {
    await updateTask({ ...task, scheduled_date: null });
    dropdown.close();
    setSelectedDate("");
  }

  function updateDisplayText() {
    if (task.scheduled_date === null) {
      setDisplayText("No Date");
      return;
    }

    const compareDate = (date1: Date | null, date2: Date | null) =>
      compareDatesInTimezone(date1, date2, userTimezone);

    let isToday = compareDate(task.scheduled_date, getToday(userTimezone));
    let isTomorrow = compareDate(task.scheduled_date, getTomorrow(userTimezone));
    let isYesterday = compareDate(task.scheduled_date, getYesterday(userTimezone));

    if (isToday) {
      setDisplayText("Today");
    } else if (isTomorrow) {
      setDisplayText("Tomorrow");
    } else if (isYesterday) {
      setDisplayText("Yesterday");
    } else if (task.scheduled_date) {
      setDisplayText(
        task.scheduled_date.toLocaleDateString(undefined, { timeZone: userTimezone }),
      );
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
        className="cursor-pointer inline-flex items-center justify-center gap-1 px-1.5 py-0 min-w-[32px] min-h-[24px] rounded-md text-xs font-medium transition-colors hover:opacity-80"
        style={{
          backgroundColor: color + "20",
          color: color,
          border: `1px solid ${color}40`,
        }}
      >
        <span>{icon}</span>
        <span>{displayText}</span>
        {isRecurringTask(task) && (
          <span className="ml-1 text-[10px]">🔄</span>
        )}
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
              className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
            >
              No Date
            </button>
            <button
              onClick={() => setDate(() => getToday(userTimezone))}
              className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
            >
              Today
            </button>
            <button
              onClick={() => setDate(() => getTomorrow(userTimezone))}
              className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
            >
              Tomorrow
            </button>
            {isFriday(userTimezone) && (
              <button
                onClick={() => setDate(() => getNextMonday(userTimezone))}
                className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
              >
                Next Monday
              </button>
            )}
            <button
              onClick={() => setDate(() => getNextWeek(userTimezone))}
              className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
            >
              Next Week
            </button>
            <div className="border-t mt-1 pt-1 px-2">
              <input
                aria-label="Date"
                type="date"
                className="w-full px-2 py-1 min-h-[26px] border border-gray-300 rounded text-xs focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={selectedDate}
                onChange={handleScheduledDateChange}
              />
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  );
}
