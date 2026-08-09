import React, { useState, useRef, useEffect, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { Task } from '../../models/Task';

import { saveExistingTask, deleteTask } from '../../api/tasks';

import { useTaskContext } from '../../contexts/TaskContext';
import { useStatus } from '../../contexts/StatusContext';
import { useAuth } from '../../contexts/AuthContext';
import { Menu, MenuItem, MenuRawItem } from '../ui/Menu';
import {
  getToday,
  getTomorrow,
  getNextWeek,
  getNextMonday,
  isFriday,
} from '../../utils/dates';

interface TaskListOptionsMenuProps {
  task: Task;
  showCardLink: boolean;
  setShowCardLink: (show: boolean) => void;
  onDelete: () => void;
  onToggleComplete: () => void;
  onRefresh: () => void;
  onClose: () => void;
  showHistory?: boolean;
  onToggleHistory?: () => void;
}

export function TaskListOptionsMenu({
  task,
  showCardLink,
  setShowCardLink,
  onDelete,
  onToggleComplete,
  onRefresh,
  onClose,
  showHistory = false,
  onToggleHistory,
}: TaskListOptionsMenuProps) {
  const { setRefreshTasks } = useTaskContext();
  const { getDefaultStatus, getCompleteStatus } = useStatus();
  const { user } = useAuth();
  const userTimezone = user?.timezone || 'UTC';

  const [showSchedulePicker, setShowSchedulePicker] = useState(false);
  const [selectedScheduleDate, setSelectedScheduleDate] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [dropdownPosition, setDropdownPosition] = useState<{
    top: number;
    left: number;
    width: number;
  } | null>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = () => setShowSchedulePicker(false);
    if (showSchedulePicker) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  }, [showSchedulePicker]);

  // Set default schedule date to tomorrow
  useEffect(() => {
    const tomorrow = getTomorrow(userTimezone);
    setSelectedScheduleDate(tomorrow.toISOString().split('T')[0]);
  }, [userTimezone]);

  function toggleCardLink() {
    setShowCardLink(!showCardLink);
  }

  async function handleCardUnlink() {
    let editedTask = { ...task, card_pk: 0 };
    let response = await saveExistingTask(editedTask);
    if (!('error' in response)) {
      setRefreshTasks(true);
    }
  }

  const calculateDaysFromNow = (targetDate: Date): number => {
    const today = getToday(userTimezone);
    const todayMidnight = new Date(
      today.getFullYear(),
      today.getMonth(),
      today.getDate(),
    );
    const targetMidnight = new Date(
      targetDate.getFullYear(),
      targetDate.getMonth(),
      targetDate.getDate(),
    );
    const diffTime = targetMidnight.getTime() - todayMidnight.getTime();
    return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  };

  const scheduleForDate = useCallback(
    async (date: Date) => {
      const { completeAndScheduleTask } = await import('../../api/tasks');
      const days = calculateDaysFromNow(date);
      if (days <= 0) {
        alert('Please select a future date.');
        return;
      }
      try {
        await completeAndScheduleTask(task.id, days);
        onRefresh();
        setShowSchedulePicker(false);
        setDropdownPosition(null);
        onClose();
      } catch (error) {
        console.error('Error completing and scheduling task:', error);
        alert('Failed to complete and schedule task. Please try again.');
      }
    },
    [task.id, onRefresh, onClose, userTimezone],
  );

  const handleScheduleDateChange = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const inputValue = e.target.value;
      if (!inputValue) return;
      const [year, month, day] = inputValue.split('-').map(Number);
      const newDate = new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0));
      await scheduleForDate(newDate);
    },
    [scheduleForDate],
  );

  const scheduleTomorrow = useCallback(async () => {
    await scheduleForDate(getTomorrow(userTimezone));
  }, [scheduleForDate, userTimezone]);

  const scheduleNextWeek = useCallback(async () => {
    await scheduleForDate(getNextWeek(userTimezone));
  }, [scheduleForDate, userTimezone]);

  const scheduleNextMonday = useCallback(async () => {
    await scheduleForDate(getNextMonday(userTimezone));
  }, [scheduleForDate, userTimezone]);

  const handleCompleteAndSchedule = useCallback(async () => {
    if (!showSchedulePicker && buttonRef.current) {
      const rect = buttonRef.current.getBoundingClientRect();
      setDropdownPosition({
        top: rect.top,
        left: rect.left,
        width: rect.width,
      });
    } else {
      setDropdownPosition(null);
    }
    setShowSchedulePicker(!showSchedulePicker);
  }, [showSchedulePicker]);

  // Keyboard navigation for dropdown menu
  const handleDropdownKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!showSchedulePicker) return;
      const items = dropdownRef.current?.querySelectorAll('button, input');
      if (!items || items.length === 0) return;
      const currentIndex = Array.from(items).indexOf(
        document.activeElement as HTMLElement,
      );
      switch (e.key) {
        case 'Escape':
          e.preventDefault();
          setShowSchedulePicker(false);
          break;
        case 'ArrowDown':
          e.preventDefault();
          const nextIndex =
            currentIndex < items.length - 1 ? currentIndex + 1 : 0;
          (items[nextIndex] as HTMLElement).focus();
          break;
        case 'ArrowUp':
          e.preventDefault();
          const prevIndex =
            currentIndex > 0 ? currentIndex - 1 : items.length - 1;
          (items[prevIndex] as HTMLElement).focus();
          break;
        case 'Home':
          e.preventDefault();
          (items[0] as HTMLElement).focus();
          break;
        case 'End':
          e.preventDefault();
          (items[items.length - 1] as HTMLElement).focus();
          break;
      }
    },
    [showSchedulePicker],
  );

  return (
    <>
      <Menu
        align="right"
        panelClassName="w-40 z-10"
        button={
          <svg
            className="w-4 h-4 text-gray-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
            />
          </svg>
        }
      >
        <MenuItem
          onClick={task.card_pk === 0 ? toggleCardLink : handleCardUnlink}
        >
          {task.card_pk === 0 ? 'Link Card' : 'Unlink Card'}
        </MenuItem>
        {!task.is_complete && (
          <MenuRawItem>
            {({ active }) => (
              <button
                ref={buttonRef}
                type="button"
                onClick={handleCompleteAndSchedule}
                className={`${
                  active ? 'bg-gray-100' : ''
                } flex w-full items-center px-4 py-3 min-h-[44px] text-sm text-gray-700 hover:bg-gray-100`}
              >
                Complete & Schedule
              </button>
            )}
          </MenuRawItem>
        )}
        {onToggleHistory && (
          <MenuItem onClick={onToggleHistory}>
            {showHistory ? 'Hide' : 'Show'} History
          </MenuItem>
        )}
        <MenuItem onClick={onDelete} className="!text-red-600">
          Delete Task
        </MenuItem>
      </Menu>
      {showSchedulePicker &&
        dropdownPosition &&
        createPortal(
          <div
            ref={dropdownRef}
            role="menu"
            className="fixed z-[1001] bg-white rounded-md shadow-lg py-1 border border-gray-200"
            style={{
              top: dropdownPosition.top + 40,
              left: dropdownPosition.left,
              minWidth: Math.max(160, dropdownPosition.width),
            }}
            onClick={(e) => e.stopPropagation()}
            onKeyDown={handleDropdownKeyDown}
          >
            <div className="flex flex-col">
              <button
                role="menuitem"
                tabIndex={-1}
                onClick={() => scheduleTomorrow()}
                className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
              >
                Tomorrow
              </button>
              <button
                role="menuitem"
                tabIndex={-1}
                onClick={() => scheduleNextMonday()}
                className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
              >
                Next Monday
              </button>
              <button
                role="menuitem"
                tabIndex={-1}
                onClick={() => scheduleNextWeek()}
                className="w-full text-left px-2 py-1 min-h-[26px] hover:bg-gray-100 text-xs whitespace-nowrap"
              >
                Next Week
              </button>
              <div className="border-t mt-1 pt-1 px-2">
                <input
                  aria-label="Schedule Date"
                  type="date"
                  className="w-full px-2 py-1 min-h-[26px] border border-gray-300 rounded text-xs focus:outline-none focus:ring-2 focus:ring-blue-500"
                  value={selectedScheduleDate}
                  min={getToday(userTimezone).toISOString().split('T')[0]}
                  onChange={(e) => {
                    e.stopPropagation();
                    setSelectedScheduleDate(e.target.value);
                    handleScheduleDateChange(e);
                  }}
                  onClick={(e) => e.stopPropagation()}
                />
              </div>
            </div>
          </div>,
          document.body,
        )}
    </>
  );
}
