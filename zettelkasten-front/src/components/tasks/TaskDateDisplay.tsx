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
    const localTimezoneOffset = newDate.getTimezoneOffset();
    const utcDate = new Date(newDate.getTime() + localTimezoneOffset * 60000);

    setSelectedDate(newDate.toISOString().substr(0, 10)); // Update selected date in the state
    let editedTask = { ...task, scheduled_date: utcDate };

    updateTask(editedTask);

    setDisplayDatePicker(false);
    setSelectedDate("");
  }

  const getDisplayColor = () => {
    if (!task.is_complete && isPast(task.scheduled_date)) {
      return "red";
    }
    switch (displayText) {
      case "Today":
        return "green";
      case "Tomorrow":
        return "green";
      case "Yesterday":
        return "red";
      default:
        return "black";
    }
  };

  async function setNoDate() {
    let editedTask = { ...task, scheduled_date: null };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setToday() {
    let editedTask = { ...task, scheduled_date: getToday() };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setTomorrow() {
    let editedTask = { ...task, scheduled_date: getTomorrow() };
    updateTask(editedTask);
    setDisplayDatePicker(false);
    setSelectedDate("");
  }
  async function setNextWeek() {
    let editedTask = { ...task, scheduled_date: getNextWeek() };
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

  return (
    <div>
      <div
        onClick={handleTextClick}
        className="task-date mr-1"
        style={{ color: getDisplayColor() }}
      >
        {displayText}
        {isRecurringTask(task) && (
          <>
            <span> - </span>
            <span className="task-recurring">Recurring</span>
          </>
        )}
      </div>
      {displayDatePicker && (
        <div className="dropdown">
          <div className="popup-menu-left-aligned">
            <div className="flex flex-col space-y-2">
              {" "}
              {/* This creates vertical spacing between children */}
              <button onClick={setNoDate} className="w-full">
                No Date
              </button>
              <button onClick={setToday} className="w-full">
                Today
              </button>
              <button onClick={setTomorrow} className="w-full">
                Tomorrow
              </button>
              {isFriday() ? (
                <button onClick={setNextMonday} className="w-full">
                  Next Monday
                </button>
              ) : (
                <div></div>
              )}
              <button onClick={setNextWeek} className="w-full">
                Next Week
              </button>
              <input
                aria-label="Date"
                type="date"
                className="p-2 w-full"
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
