import React, { useState, useRef, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { Task } from "../../models/Task";

import { saveExistingTask, deleteTask } from "../../api/tasks";

import { useTaskContext } from "../../contexts/TaskContext";
import { useStatus } from "../../contexts/StatusContext";
import { useAuth } from "../../contexts/AuthContext";
import { Menu } from "@headlessui/react";
import { getToday, getTomorrow, getNextWeek, getNextMonday, isFriday } from "../../utils/dates";

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
  const userTimezone = user?.timezone || "UTC";

  const [showSchedulePicker, setShowSchedulePicker] = useState(false);
  const [selectedScheduleDate, setSelectedScheduleDate] = useState("");
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [dropdownPosition, setDropdownPosition] = useState<{ top: number; left: number; width: number } | null>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = () => setShowSchedulePicker(false);
    if (showSchedulePicker) {
      document.addEventListener("click", handleClickOutside);
      return () => document.removeEventListener("click", handleClickOutside);
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
    if (!("error" in response)) {
      setRefreshTasks(true);
    }
  }

  const calculateDaysFromNow = (targetDate: Date): number => {
    const today = getToday(userTimezone);
    const todayMidnight = new Date(today.getFullYear(), today.getMonth(), today.getDate());
    const targetMidnight = new Date(targetDate.getFullYear(), targetDate.getMonth(), targetDate.getDate());
    const diffTime = targetMidnight.getTime() - todayMidnight.getTime();
    return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  };

  const scheduleForDate = useCallback(async (date: Date) => {
    const { completeAndScheduleTask } = await import("../../api/tasks");
    const days = calculateDaysFromNow(date);
    if (days <= 0) {
      alert("Please select a future date.");
      return;
    }
    try {
      await completeAndScheduleTask(task.id, days);
      onRefresh();
      setShowSchedulePicker(false);
      setDropdownPosition(null);
      onClose();
    } catch (error) {
      console.error("Error completing and scheduling task:", error);
      alert("Failed to complete and schedule task. Please try again.");
    }
  }, [task.id, onRefresh, onClose, userTimezone]);

  const handleScheduleDateChange = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const inputValue = e.target.value;
    if (!inputValue) return;
    const [year, month, day] = inputValue.split('-').map(Number);
    const newDate = new Date(Date.UTC(year, month - 1, day, 0, 0, 0, 0));
    await scheduleForDate(newDate);
  }, [scheduleForDate]);

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
  const handleDropdownKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (!showSchedulePicker) return;
    const items = dropdownRef.current?.querySelectorAll('button, input');
    if (!items || items.length === 0) return;
    const currentIndex = Array.from(items).indexOf(document.activeElement as HTMLElement);
    switch (e.key) {
      case 'Escape':
        e.preventDefault();
        setShowSchedulePicker(false);
        break;
      case 'ArrowDown':
        e.preventDefault();
        const nextIndex = currentIndex < items.length - 1 ? currentIndex + 1 : 0;
        (items[nextIndex] as HTMLElement).focus();
        break;
      case 'ArrowUp':
        e.preventDefault();
        const prevIndex = currentIndex > 0 ? currentIndex - 1 : items.length - 1;
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
  }, [showSchedulePicker]);

  return (
    <>
      <Menu as="div" className="relative inline-block text-left">
        <div>
          <Menu.Button className="inline-flex justify-center items-center w-full rounded-md border border-gray-300 shadow-sm px-2 py-1 min-w-[44px] min-h-[44px] bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-offset-gray-100 focus:ring-indigo-500">
            ⋮
          </Menu.Button>
        </div>
        <Menu.Items className="origin-top-right absolute right-0 mt-2 w-48 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
          <div className="py-1">
            <Menu.Item>
              {({ active }) => (
                <button
                  onClick={task.card_pk === 0 ? toggleCardLink : handleCardUnlink}
                  className={`${
                    active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                  } group flex rounded-md items-center w-full px-4 py-3 min-h-[44px] text-sm`}
                >
                  {task.card_pk === 0 ? 'Link Card' : 'Unlink Card'}
                </button>
              )}
            </Menu.Item>
            {!task.is_complete && (
              <Menu.Item>
                {({ active }) => (
                  <button
                    ref={buttonRef}
                    onClick={handleCompleteAndSchedule}
                    className={`${
                      active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                    } group flex rounded-md items-center w-full px-4 py-3 min-h-[44px] text-sm`}
                  >
                    Complete & Schedule Again
                  </button>
                )}
              </Menu.Item>
            )}
            {onToggleHistory && (
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={onToggleHistory}
                    className={`${
                      active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                    } group flex rounded-md items-center w-full px-4 py-3 min-h-[44px] text-sm`}
                  >
                    {showHistory ? 'Hide' : 'Show'} History
                  </button>
                )}
              </Menu.Item>
            )}
            <Menu.Item>
              {({ active }) => (
                <button
                  onClick={onDelete}
                  className={`${
                    active ? 'bg-gray-100 text-red-600' : 'text-red-600'
                  } group flex rounded-md items-center w-full px-4 py-3 min-h-[44px] text-sm`}
                >
                  Delete Task
                </button>
              )}
            </Menu.Item>
          </div>
        </Menu.Items>
      </Menu>
      {showSchedulePicker && dropdownPosition && createPortal(
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
              className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100 text-sm"
            >
              Tomorrow
            </button>
            <button
              role="menuitem"
              tabIndex={-1}
              onClick={() => scheduleNextMonday()}
              className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100 text-sm"
            >
              Next Monday
            </button>
            <button
              role="menuitem"
              tabIndex={-1}
              onClick={() => scheduleNextWeek()}
              className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100 text-sm"
            >
              Next Week
            </button>
            <div className="border-t mt-1 pt-1 px-2">
              <input
                aria-label="Schedule Date"
                type="date"
                className="w-full px-3 py-2 min-h-[44px] border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
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
        </div>
      , document.body)}
    </>
  );
}
