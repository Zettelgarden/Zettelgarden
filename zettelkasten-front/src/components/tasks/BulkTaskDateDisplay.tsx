import React, { useEffect, useState } from 'react';
import { Modal } from '../ui/Modal';
import { Task } from '../../models/Task';

import {
  getToday,
  getTomorrow,
  getNextWeek,
  getNextMonday,
  isFriday,
} from '../../utils/dates';
import { saveExistingTask } from '../../api/tasks';
import { useTaskContext } from '../../contexts/TaskContext';
import { useAuth } from '../../contexts/AuthContext';

interface BulkTaskDateDisplayProps {
  tasks: Task[];
  setShowBulkEdit: (show: boolean) => void;
}

export function BulkTaskDateDisplay({
  tasks,
  setShowBulkEdit,
}: BulkTaskDateDisplayProps) {
  const { setRefreshTasks } = useTaskContext();
  const { user } = useAuth();
  const userTimezone = user?.timezone || 'UTC';
  const [selectedDate, setSelectedDate] = useState<string>('');

  async function updateTasks(newDate: Date | null) {
    const promises = tasks.map((task) => {
      const updatedTask = { ...task, scheduled_date: newDate };
      return saveExistingTask(updatedTask);
    });

    try {
      await Promise.all(promises);
      setRefreshTasks(true);
      setShowBulkEdit(false);
    } catch (error) {
      console.error('Failed to bulk update tasks', error);
      alert('Failed to bulk update tasks. Please try again.');
    }
  }

  async function handleScheduledDateChange(
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
    updateTasks(newDate);
  }

  async function setNoDate() {
    updateTasks(null);
  }
  async function setToday() {
    updateTasks(getToday(userTimezone));
  }
  async function setTomorrow() {
    updateTasks(getTomorrow(userTimezone));
  }
  async function setNextWeek() {
    updateTasks(getNextWeek(userTimezone));
  }

  async function setNextMonday() {
    updateTasks(getNextMonday(userTimezone));
  }

  return (
    <Modal
      open
      onClose={() => setShowBulkEdit(false)}
      size="sm"
      dialogClassName="z-[100]"
    >
      <h3 className="text-lg font-semibold mb-4 text-gray-700">
        Edit Date ({tasks.length} tasks)
      </h3>
      <div className="flex flex-col space-y-2">
        <button
          onClick={setNoDate}
          className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-slate-100 rounded border border-slate-200"
        >
          No Date
        </button>
        <button
          onClick={setToday}
          className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-slate-100 rounded border border-slate-200"
        >
          Today
        </button>
        <button
          onClick={setTomorrow}
          className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-slate-100 rounded border border-slate-200"
        >
          Tomorrow
        </button>
        {isFriday(userTimezone) ? (
          <button
            onClick={setNextMonday}
            className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-slate-100 rounded border border-slate-200"
          >
            Next Monday
          </button>
        ) : null}
        <button
          onClick={setNextWeek}
          className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-slate-100 rounded border border-slate-200"
        >
          Next Week
        </button>

        <div className="pt-2 border-t mt-2">
          <label className="block text-sm font-medium text-slate-700 mb-1">
            Custom Date
          </label>
          <input
            aria-label="Date"
            type="date"
            className="p-2 min-h-[44px] w-full border border-slate-300 rounded"
            value={selectedDate}
            onChange={handleScheduledDateChange}
          />
        </div>

        <button
          onClick={() => setShowBulkEdit(false)}
          className="mt-4 px-4 py-3 min-h-[44px] border border-slate-300 rounded hover:bg-slate-50 w-full"
        >
          Cancel
        </button>
      </div>
    </Modal>
  );
}
