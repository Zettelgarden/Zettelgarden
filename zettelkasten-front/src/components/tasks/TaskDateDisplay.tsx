import React, { useEffect, useState } from 'react';
import { Task } from '../../models/Task';
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
} from '../../utils/dates';
import { useOptimisticTaskUpdate } from '../../hooks/useOptimisticTaskUpdate';
import { useAuth } from '../../contexts/AuthContext';
import { TaskDropdown } from './TaskDropdown';

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
  const userTimezone = user?.timezone || 'UTC';
  const { updateTask } = useOptimisticTaskUpdate({
    task,
    setTask,
    saveOnChange,
    errorMessagePrefix: 'Failed to update task date',
  });

  const [displayText, setDisplayText] = useState<string>('');
  const [selectedDate, setSelectedDate] = useState<string>(
    task.scheduled_date ? task.scheduled_date.toISOString().substr(0, 10) : '',
  );

  async function handleScheduledDateChange(
    e: React.ChangeEvent<HTMLInputElement>,
    close: () => void,
  ) {
    const inputValue = e.target.value;
    if (!inputValue) return;

    const [year, month, day] = inputValue.split('-').map(Number);
    const newDate = new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0));

    setSelectedDate(inputValue);
    await updateTask({ ...task, scheduled_date: newDate });
    close();
    setSelectedDate('');
  }

  const getDisplayColor = () => {
    if (!task.is_complete && isPast(task.scheduled_date, userTimezone)) {
      return '#EF4444'; // red
    }
    switch (displayText) {
      case 'Today':
        return '#10B981'; // green
      case 'Tomorrow':
        return '#3B82F6'; // blue
      case 'Yesterday':
        return '#EF4444'; // red
      default:
        return '#6B7280'; // gray
    }
  };

  const getDisplayIcon = () => {
    if (!task.is_complete && isPast(task.scheduled_date, userTimezone)) {
      return '⚠️';
    }
    switch (displayText) {
      case 'Today':
        return '📅';
      case 'Tomorrow':
        return '📆';
      case 'No Date':
        return '○';
      default:
        return '📅';
    }
  };

  async function setDate(dateSetter: () => Date, close: () => void) {
    await updateTask({ ...task, scheduled_date: dateSetter() });
    close();
    setSelectedDate('');
  }

  async function setNoDate(close: () => void) {
    await updateTask({ ...task, scheduled_date: null });
    close();
    setSelectedDate('');
  }

  function updateDisplayText() {
    if (task.scheduled_date === null) {
      setDisplayText('No Date');
      return;
    }

    const compareDate = (date1: Date | null, date2: Date | null) =>
      compareDatesInTimezone(date1, date2, userTimezone);

    let isToday = compareDate(task.scheduled_date, getToday(userTimezone));
    let isTomorrow = compareDate(
      task.scheduled_date,
      getTomorrow(userTimezone),
    );
    let isYesterday = compareDate(
      task.scheduled_date,
      getYesterday(userTimezone),
    );

    if (isToday) {
      setDisplayText('Today');
    } else if (isTomorrow) {
      setDisplayText('Tomorrow');
    } else if (isYesterday) {
      setDisplayText('Yesterday');
    } else if (task.scheduled_date) {
      setDisplayText(
        task.scheduled_date.toLocaleDateString(undefined, {
          timeZone: userTimezone,
        }),
      );
    }
  }

  useEffect(() => {
    updateDisplayText();
  }, [task]);

  const color = getDisplayColor();
  const icon = getDisplayIcon();

  return (
    <TaskDropdown
      display={{
        icon: isRecurringTask(task) ? `${icon} 🔄` : icon,
        text: displayText,
        color,
      }}
    >
      {({ close }) => (
        <div className="flex flex-col">
          <button
            onClick={() => setNoDate(close)}
            className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
          >
            No Date
          </button>
          <button
            onClick={() => setDate(() => getToday(userTimezone), close)}
            className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
          >
            Today
          </button>
          <button
            onClick={() => setDate(() => getTomorrow(userTimezone), close)}
            className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
          >
            Tomorrow
          </button>
          {isFriday(userTimezone) && (
            <button
              onClick={() => setDate(() => getNextMonday(userTimezone), close)}
              className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
            >
              Next Monday
            </button>
          )}
          <button
            onClick={() => setDate(() => getNextWeek(userTimezone), close)}
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
              onChange={(e) => handleScheduledDateChange(e, close)}
            />
          </div>
        </div>
      )}
    </TaskDropdown>
  );
}
