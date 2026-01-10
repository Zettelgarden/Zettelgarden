import React, { useEffect, useState } from "react";
import { Task } from "../../models/Task";

import {
  compareDates,
  getToday,
  getTomorrow,
  getYesterday,
  getNextWeek,
  isPast,
  isRecurringTask,
  getNextMonday,
  isFriday,
} from "../../utils/dates";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
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
  const { setRefreshTasks, updateTask: updateTaskInContext } = useTaskContext();
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";
  const [displayText, setDisplayText] = useState<string>("");
  const [selectedDate, setSelectedDate] = useState<string>(
    task.scheduled_date ? task.scheduled_date.toISOString().substr(0, 10) : "",
  );
  const [displayDatePicker, setDisplayDatePicker] = useState<boolean>(false);

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
          console.error("Failed to update task date:", response.error);
        }
      } catch (error) {
        // Rollback on network error
        updateTaskInContext(task);
        console.error("Failed to update task date:", error);
      }
    } else {
      setTask(editedTask);
    }
  }

  async function handleScheduledDateChange(
    e: React.ChangeEvent<HTMLInputElement>,
  ) {
    console.log(e);
    const newDate = new Date(e.target.value);

    setSelectedDate(newDate.toISOString().substr(0, 10)); // Update selected date in the state
    let editedTask = { ...task, scheduled_date: newDate };

    updateTask(editedTask);

    setDisplayDatePicker(false);
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
    if (!task.is_complete && isPast(task.scheduled_date)) {
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

  // Close menu when clicking outside
  React.useEffect(() => {
    const handleClickOutside = () => setDisplayDatePicker(false);
    if (displayDatePicker) {
      document.addEventListener("click", handleClickOutside);
      return () => document.removeEventListener("click", handleClickOutside);
    }
  }, [displayDatePicker]);

  async function setNoDate() {
    let editedTask = { ...task, scheduled_date: null };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setToday() {
    let editedTask = { ...task, scheduled_date: getToday(userTimezone) };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setTomorrow() {
    let editedTask = { ...task, scheduled_date: getTomorrow(userTimezone) };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setNextWeek() {
    let editedTask = { ...task, scheduled_date: getNextWeek(userTimezone) };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }

  async function setNextMonday() {
    let editedTask = { ...task, scheduled_date: getNextMonday() };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }

  function handleTextClick() {
    setDisplayDatePicker(!displayDatePicker);
  }

  function updateDisplayText() {
    if (task.scheduled_date === null) {
      setDisplayText("No Date");
      return;
    }
    let isToday = compareDates(task.scheduled_date, getToday());
    let isTomorrow = compareDates(task.scheduled_date, getTomorrow());
    let isYesterday = compareDates(task.scheduled_date, getYesterday());
    if (isToday) {
      setDisplayText("Today");
    } else if (isTomorrow) {
      setDisplayText("Tomorrow");
    } else if (isYesterday) {
      setDisplayText("Yesterday");
    } else if (task.scheduled_date) {
      setDisplayText(task.scheduled_date.toLocaleDateString());
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
        <span>{icon}</span>
        <span>{displayText}</span>
        {isRecurringTask(task) && (
          <span className="ml-1 text-[10px]">🔄</span>
        )}
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
              No Date
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
            {isFriday() && (
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
                aria-label="Date"
                type="date"
                className="w-full px-2 py-1.5 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={selectedDate}
                onChange={handleScheduledDateChange}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
