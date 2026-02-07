import React, { useState, useRef, useEffect } from "react";
import { Link } from "react-router-dom";
import logo from "../../assets/logo.png";
import { useAuth } from "../../contexts/AuthContext";
import { Plus } from "lucide-react";

interface SidebarHeaderProps {
  onNewStandardCard: () => void;
  onNewArticle: () => void;
  onNewTask: () => void;
  onNewChat: () => void;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}

export function SidebarHeader({
  onNewStandardCard,
  onNewArticle,
  onNewTask,
  onNewChat,
  isCollapsed,
  onToggleCollapse,
}: SidebarHeaderProps) {
  const username = localStorage.getItem("username");
  const [isNewDropdownOpen, setIsNewDropdownOpen] = useState(false);
  const { hasSubscription } = useAuth();
  const dropdownRef = useRef<HTMLDivElement>(null);

  const toggleNewDropdown = () => {
    setIsNewDropdownOpen(!isNewDropdownOpen);
  };

  const closeDropdown = () => {
    setIsNewDropdownOpen(false);
  };

  // Handle Escape key and click outside to close dropdown
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isNewDropdownOpen) {
        closeDropdown();
      }
    };

    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        closeDropdown();
      }
    };

    if (isNewDropdownOpen) {
      document.addEventListener("keydown", handleKeyDown);
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isNewDropdownOpen]);

  return (
    <div className={`flex items-center border-b ${isCollapsed ? "flex-col justify-center py-3 gap-3" : "p-4"}`}>
      <Link to="/app" className="flex-shrink-0">
        <img
          src={logo}
          alt="Company Logo"
          className={`rounded-md ${isCollapsed ? "h-10 w-10" : "h-8 w-auto"}`}
        />
      </Link>
      {!isCollapsed && (
        <div className="flex-grow mx-2 min-w-0">
          <Link to="/app/settings">
            <span className="text-sm font-medium hover:text-gray-700 truncate block">
              {username}
            </span>
          </Link>
        </div>
      )}
      <div className={`flex items-center ${isCollapsed ? "" : ""} flex-shrink-0`}>
        <div className="relative" ref={dropdownRef}>
          <button
            onClick={toggleNewDropdown}
            className={`flex items-center justify-center rounded-full transition-colors ${
              isCollapsed
                ? "w-10 h-10 text-gray-700 hover:bg-gray-100"
                : "bg-blue-500 text-white hover:bg-blue-600 min-w-[44px] min-h-[44px]"
            }`}
            aria-haspopup="true"
            aria-expanded={isNewDropdownOpen}
            aria-label="Create new item"
          >
            <Plus size={isCollapsed ? 20 : 24} strokeWidth={2.5} />
          </button>
          {isNewDropdownOpen && (
            <div
              className={`absolute ${isCollapsed ? "left-full ml-2 top-0" : "right-0"} mt-2 w-48 bg-white rounded-md shadow-lg py-1 z-[70] border`}
              role="menu"
            >
              <button
                onClick={onNewStandardCard}
                className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100"
                role="menuitem"
              >
                Create Card
              </button>
              <button
                onClick={onNewArticle}
                className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100"
                role="menuitem"
              >
                Add Article (Card)
              </button>
              <button
                onClick={onNewTask}
                className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100"
                role="menuitem"
              >
                Create Task
              </button>
              {hasSubscription && (
                <button
                  onClick={onNewChat}
                  className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100"
                  role="menuitem"
                >
                  New Chat
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
