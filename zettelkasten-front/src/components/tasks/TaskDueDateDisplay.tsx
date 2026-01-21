import React, { useEffect, useState } from "react";
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
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
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
  const { setRefreshTasks, updateTask: updateTaskInContext } = useTaskContext();
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";
  const [displayText, setDisplayText] = useState<string>("");
  const [selectedDate, setSelectedDate] = useState<string>(
    task.due_date ? task.due_date.toISOString().substr(0, 10) : "",
  );
  const [displayDatePicker, setDisplayDatePicker] = useState<boolean>(false);

  async function updateTask(editedTask: Task) {
    // Always update local state for immediate UI feedback
    setTask(editedTask);

    if (saveOnChange) {
      // Also update context for other components
      updateTaskInContext(editedTask);

      // Send update to server in background
      try {
        const response = await saveExistingTask(editedTask);
        if ("error" in response) {
          // Rollback on error
          setTask(task);
          updateTaskInContext(task);
          console.error("Failed to update task due date:", response.error);
        }
      } catch (error) {
        // Rollback on network error
        setTask(task);
        updateTaskInContext(task);
        console.error("Failed to update task due date:", error);
      }
    }
  }

  async function handleDueDateChange(
    e: React.ChangeEvent<HTMLInputElement>,
  ) {
    const inputValue = e.target.value;
    if (!inputValue) return;

    // Parse the date input value (YYYY-MM-DD) in the user's timezone
    // HTML date input returns YYYY-MM-DD, which we want to interpret as user timezone
    const [year, month, day] = inputValue.split('-').map(Number);

    // Create a date object at midnight UTC that represents the same calendar date
    // in the user's timezone. This ensures the stored date matches user intent.
    const newDate = new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0));

    setSelectedDate(inputValue);
    let editedTask = { ...task, due_date: newDate };

    updateTask(editedTask);

    setDisplayDatePicker(false);
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

  // Close menu when clicking outside
  React.useEffect(() => {
    const handleClickOutside = () => setDisplayDatePicker(false);
    if (displayDatePicker) {
      document.addEventListener("click", handleClickOutside);
      return () => document.removeEventListener("click", handleClickOutside);
    }
  }, [displayDatePicker]);

  async function setNoDate() {
    let editedTask = { ...task, due_date: null };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setToday() {
    let editedTask = { ...task, due_date: getToday(userTimezone) };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setTomorrow() {
    let editedTask = { ...task, due_date: getTomorrow(userTimezone) };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setNextWeek() {
    let editedTask = { ...task, due_date: getNextWeek(userTimezone) };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }

  async function setNextMonday() {
    let editedTask = { ...task, due_date: getNextMonday(userTimezone) };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }

  function handleTextClick() {
    setDisplayDatePicker(!displayDatePicker);
  }

  function updateDisplayText() {
    if (task.due_date === null) {
      setDisplayText("No Deadline");
      return;
    }
    let isToday = compareDatesInTimezone(task.due_date, getToday(userTimezone), userTimezone);
    let isTomorrow = compareDatesInTimezone(task.due_date, getTomorrow(userTimezone), userTimezone);
    let isYesterday = compareDatesInTimezone(task.due_date, getYesterday(userTimezone), userTimezone);
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

  const color = getDisplayColor();
  const icon = getDisplayIcon();

  return (
    <div className="relative inline-block">
      <span
        onClick={(e) => {
          e.stopPropagation();
          handleTextClick();
        }}
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

      {displayDatePicker && (
        <div
          className="absolute z-20 mt-1 bg-white rounded-md shadow-lg py-1 min-w-[160px] border border-gray-200"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex flex-col">
            <button
              onClick={setNoDate}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
            >
              No Deadline
            </button>
            <button
              onClick={setToday}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
            >
              Today
            </button>
            <button
              onClick={setTomorrow}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
            >
              Tomorrow
            </button>
            {isFriday(userTimezone) && (
              <button
                onClick={setNextMonday}
                className="w-full text-left px-3 py-2 hover:bg-gray-100 text-sm"
              >
                Next Monday
              </button>
            )}
            <button
              onClick={setNextWeek}
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
        </div>
      )}
    </div>
  );
}
