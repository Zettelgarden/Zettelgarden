import React, { useEffect, useState } from "react";
import { Task } from "../../models/Task";
import { format, toZonedTime, fromZonedTime } from "date-fns-tz";
import { getToday, getTomorrow, getNowInTimezone } from "../../utils/dates";
import { useTaskDropdown } from "../../hooks/useTaskDropdown";
import { useOptimisticTaskUpdate } from "../../hooks/useOptimisticTaskUpdate";
import { useAuth } from "../../contexts/AuthContext";

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
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";
  const dropdown = useTaskDropdown();
  const { updateTask } = useOptimisticTaskUpdate({
    task,
    setTask,
    saveOnChange,
    errorMessagePrefix: "Failed to update task reminder",
  });

  const [displayText, setDisplayText] = useState<string>("");
  const [customDateTime, setCustomDateTime] = useState<string>("");

  function setNoReminder() {
    updateTask({ ...task, reminder_time: null, reminder_sent: false });
    dropdown.close();
  }

  function setReminderIn(minutes: number) {
    // Use actual UTC time for reminders - JSON serialization will handle the rest
    const reminderTime = new Date(Date.now() + minutes * 60 * 1000);
    updateTask({ ...task, reminder_time: reminderTime, reminder_sent: false });
    dropdown.close();
  }

  function setReminderTomorrowMorning() {
    // Get tomorrow at 9am in user's timezone as actual UTC time
    const now = new Date();
    const tomorrowInTz = toZonedTime(now, userTimezone);
    tomorrowInTz.setDate(tomorrowInTz.getDate() + 1);
    tomorrowInTz.setHours(9, 0, 0, 0);
    // Convert to actual UTC time
    const reminderTime = fromZonedTime(tomorrowInTz, userTimezone);
    updateTask({ ...task, reminder_time: reminderTime, reminder_sent: false });
    dropdown.close();
  }

  function handleCustomDateTimeChange(e: React.ChangeEvent<HTMLInputElement>) {
    setCustomDateTime(e.target.value);
  }

  function applyCustomDateTime() {
    if (!customDateTime) return;

    // The datetime-local input gives us a local date/time (no timezone info)
    // We interpret this as being in the user's timezone and convert to UTC
    const browserLocalDate = new Date(customDateTime);
    const reminderTime = fromZonedTime(browserLocalDate, userTimezone);

    updateTask({ ...task, reminder_time: reminderTime, reminder_sent: false });
    dropdown.close();
    setCustomDateTime("");
  }

  function updateDisplayText() {
    if (task.reminder_time === null) {
      setDisplayText("No Reminder");
      return;
    }

    const now = getNowInTimezone(userTimezone);
    const reminderTime = new Date(task.reminder_time);
    const diffMs = reminderTime.getTime() - now.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (task.reminder_sent) {
      setDisplayText(`Sent ${format(reminderTime, 'MMM d, h:mm a', { timeZone: userTimezone })}`);
      return;
    }

    if (diffMins < 0) {
      setDisplayText(`Past ${format(reminderTime, 'MMM d, h:mm a', { timeZone: userTimezone })}`);
    } else if (diffMins < 60) {
      setDisplayText(`In ${diffMins}m`);
    } else if (diffHours < 24) {
      setDisplayText(`In ${diffHours}h`);
    } else if (diffDays === 1) {
      setDisplayText(`Tomorrow ${format(reminderTime, 'h:mm a', { timeZone: userTimezone })}`);
    } else {
      setDisplayText(format(reminderTime, 'MMM d, h:mm a', { timeZone: userTimezone }));
    }
  }

  useEffect(() => {
    updateDisplayText();
    const interval = setInterval(updateDisplayText, 60000);
    return () => clearInterval(interval);
  }, [task.reminder_time, task.reminder_sent]);

  const getDisplayColor = () => {
    if (task.reminder_sent) {
      return "gray";
    }
    if (task.reminder_time && new Date(task.reminder_time) < getNowInTimezone(userTimezone)) {
      return "red";
    }
    return "blue";
  };

  return (
    <div className="relative">
      <div
        onClick={dropdown.toggle}
        className="cursor-pointer px-2 py-1 rounded hover:bg-gray-100 text-sm font-medium"
        style={{ color: getDisplayColor() }}
        title={task.reminder_sent ? "Reminder already sent" : "Click to set reminder"}
      >
        🔔 {displayText}
      </div>
      {dropdown.isOpen && (
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
