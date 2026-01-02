import React, { useEffect, useState } from "react";
import { Task } from "../../models/Task";
import { saveExistingTask } from "../../api/tasks";
import { useTaskContext } from "../../contexts/TaskContext";
import { format } from "date-fns";

interface TaskReminderDisplayProps {
  task: Task;
  setTask: (task: Task) => void;
  saveOnChange: boolean;
}

export function TaskReminderDisplay({
  task,
  setTask,
  saveOnChange,
}: TaskReminderDisplayProps) {
  const { updateTask: updateTaskInContext } = useTaskContext();
  const [displayText, setDisplayText] = useState<string>("");
  const [displayPicker, setDisplayPicker] = useState<boolean>(false);
  const [customDateTime, setCustomDateTime] = useState<string>("");

  async function updateTask(editedTask: Task) {
    // Always update local state first
    setTask(editedTask);

    if (saveOnChange) {
      // Optimistic update: update context immediately
      updateTaskInContext(editedTask);

      // Send update to server in background
      try {
        const response = await saveExistingTask(editedTask);
        if ("error" in response) {
          // Rollback on error
          setTask(task);
          updateTaskInContext(task);
          console.error("Failed to update task reminder:", response.error);
        }
      } catch (error) {
        // Rollback on network error
        setTask(task);
        updateTaskInContext(task);
        console.error("Failed to update task reminder:", error);
      }
    }
  }

  function setNoReminder() {
    const editedTask = { ...task, reminder_time: null, reminder_sent: false };
    updateTask(editedTask);
    setDisplayPicker(false);
  }

  function setReminderIn(minutes: number) {
    const reminderTime = new Date();
    reminderTime.setMinutes(reminderTime.getMinutes() + minutes);
    const editedTask = { ...task, reminder_time: reminderTime, reminder_sent: false };
    updateTask(editedTask);
    setDisplayPicker(false);
  }

  function setReminderTomorrowMorning() {
    const reminderTime = new Date();
    reminderTime.setDate(reminderTime.getDate() + 1);
    reminderTime.setHours(9, 0, 0, 0);
    const editedTask = { ...task, reminder_time: reminderTime, reminder_sent: false };
    updateTask(editedTask);
    setDisplayPicker(false);
  }

  function handleCustomDateTimeChange(e: React.ChangeEvent<HTMLInputElement>) {
    setCustomDateTime(e.target.value);
  }

  function applyCustomDateTime() {
    if (!customDateTime) return;
    const newDateTime = new Date(customDateTime);
    const editedTask = { ...task, reminder_time: newDateTime, reminder_sent: false };
    updateTask(editedTask);
    setDisplayPicker(false);
    setCustomDateTime("");
  }

  function handleTextClick() {
    setDisplayPicker(!displayPicker);
  }

  function updateDisplayText() {
    if (task.reminder_time === null) {
      setDisplayText("No Reminder");
      return;
    }

    const now = new Date();
    const reminderTime = new Date(task.reminder_time);
    const diffMs = reminderTime.getTime() - now.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    // Show if reminder has been sent
    if (task.reminder_sent) {
      setDisplayText(`Sent ${format(reminderTime, 'MMM d, h:mm a')}`);
      return;
    }

    // Show relative time for upcoming reminders
    if (diffMins < 0) {
      setDisplayText(`Past ${format(reminderTime, 'MMM d, h:mm a')}`);
    } else if (diffMins < 60) {
      setDisplayText(`In ${diffMins}m`);
    } else if (diffHours < 24) {
      setDisplayText(`In ${diffHours}h`);
    } else if (diffDays === 1) {
      setDisplayText(`Tomorrow ${format(reminderTime, 'h:mm a')}`);
    } else {
      setDisplayText(format(reminderTime, 'MMM d, h:mm a'));
    }
  }

  useEffect(() => {
    updateDisplayText();
    // Update display text every minute for relative times
    const interval = setInterval(updateDisplayText, 60000);
    return () => clearInterval(interval);
  }, [task.reminder_time, task.reminder_sent]);

  const getDisplayColor = () => {
    if (task.reminder_sent) {
      return "gray";
    }
    if (task.reminder_time && new Date(task.reminder_time) < new Date()) {
      return "red";
    }
    return "blue";
  };

  return (
    <div className="relative">
      <div
        onClick={handleTextClick}
        className="cursor-pointer px-2 py-1 rounded hover:bg-gray-100 text-sm font-medium"
        style={{ color: getDisplayColor() }}
        title={task.reminder_sent ? "Reminder already sent" : "Click to set reminder"}
      >
        🔔 {displayText}
      </div>
      {displayPicker && (
        <div className="absolute top-full left-0 mt-1 z-50 bg-white border border-gray-300 rounded-lg shadow-lg p-2 min-w-[200px]">
          <div className="flex flex-col space-y-2">
            <button
              onClick={setNoReminder}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 rounded"
            >
              No Reminder
            </button>
            <button
              onClick={() => setReminderIn(15)}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 rounded"
            >
              In 15 minutes
            </button>
            <button
              onClick={() => setReminderIn(60)}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 rounded"
            >
              In 1 hour
            </button>
            <button
              onClick={() => setReminderIn(180)}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 rounded"
            >
              In 3 hours
            </button>
            <button
              onClick={setReminderTomorrowMorning}
              className="w-full text-left px-3 py-2 hover:bg-gray-100 rounded"
            >
              Tomorrow 9:00 AM
            </button>
            <div className="border-t pt-2">
              <div className="flex gap-2">
                <input
                  type="datetime-local"
                  className="flex-grow px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                  value={customDateTime}
                  onChange={handleCustomDateTimeChange}
                />
                <button
                  onClick={applyCustomDateTime}
                  disabled={!customDateTime}
                  className="px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed"
                >
                  Set
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
