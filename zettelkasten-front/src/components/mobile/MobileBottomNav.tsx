import React from "react";
import { useNavigate } from "react-router-dom";
import { CardIcon } from "../../assets/icons/CardIcon";
import { TasksIcon } from "../../assets/icons/TasksIcon";
import { SearchIcon } from "../../assets/icons/SearchIcon";

interface MobileBottomNavProps {
  onCreateCard: () => void;
  onCreateTask: () => void;
}

/**
 * Mobile bottom navigation bar for primary actions.
 * Visible only on mobile devices (md:hidden) and positioned at the bottom
 * of the viewport with safe area inset support for notched devices.
 *
 * Part of Task 4: Mobile Bottom Navigation Bar
 */
export function MobileBottomNav({ onCreateCard, onCreateTask }: MobileBottomNavProps) {
  const navigate = useNavigate();

  const handleSearch = () => {
    navigate("/app/search?recent=true");
  };

  return (
    <nav
      className="md:hidden fixed bottom-0 left-0 right-0 z-[55] bg-white border-t safe-bottom-fixed"
      aria-label="Primary actions"
    >
      <div className="flex items-center justify-around py-2 pb-safe">
        {/* Create Card Button */}
        <button
          onClick={onCreateCard}
          className="flex flex-col items-center justify-center p-2 min-w-[64px] min-h-[48px] text-gray-700 hover:text-blue-500 active:text-blue-600 transition-colors"
          aria-label="Create new card"
        >
          <div className="mb-1">
            <CardIcon />
          </div>
          <span className="text-xs">Card</span>
        </button>

        {/* Create Task Button */}
        <button
          onClick={onCreateTask}
          className="flex flex-col items-center justify-center p-2 min-w-[64px] min-h-[48px] text-gray-700 hover:text-blue-500 active:text-blue-600 transition-colors"
          aria-label="Create new task"
        >
          <div className="mb-1">
            <TasksIcon />
          </div>
          <span className="text-xs">Task</span>
        </button>

        {/* Search Button */}
        <button
          onClick={handleSearch}
          className="flex flex-col items-center justify-center p-2 min-w-[64px] min-h-[48px] text-gray-700 hover:text-blue-500 active:text-blue-600 transition-colors"
          aria-label="Search"
        >
          <div className="mb-1">
            <SearchIcon />
          </div>
          <span className="text-xs">Search</span>
        </button>
      </div>

      {/* Extra padding for home indicator on devices with safe area insets */}
      <div className="safe-bottom h-0" />
    </nav>
  );
}
