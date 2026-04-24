import React, { useState, useRef, useEffect } from "react";
import { Link } from "react-router-dom";
import logo from "../../assets/logo.png";
import { useAuth } from "../../contexts/AuthContext";
import { Plus, Rss, Inbox } from "lucide-react";
import { InboxIcon } from "../../assets/icons/InboxIcon";

interface SidebarHeaderProps {
  onNewStandardCard: () => void;
  onNewArticle: () => void;
  onNewTask: () => void;
  onAddFeed: () => void;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
  unreadInboxCount: number;
}

export function SidebarHeader({
  onNewStandardCard,
  onNewArticle,
  onNewTask,
  onAddFeed,
  isCollapsed,
  onToggleCollapse,
  unreadInboxCount,
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
      <div className={`flex items-center ${isCollapsed ? "justify-center gap-2" : ""} flex-shrink-0`}>
        <Link
          to="/app/inbox"
          className={`relative flex items-center justify-center rounded-full transition-colors ${isCollapsed ? "" : "mr-2"} ${
            isCollapsed
              ? "w-10 h-10 text-gray-700 hover:bg-gray-100"
              : "w-11 h-11 min-h-[44px] bg-gray-100 text-gray-700 hover:bg-gray-200"
          }`}
          aria-label="Inbox"
        >
          <Inbox size={isCollapsed ? 18 : 20} strokeWidth={2.5} />
          {unreadInboxCount > 0 && (
            <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs font-bold rounded-full min-w-[18px] h-[18px] flex items-center justify-center px-1">
              {unreadInboxCount > 9 ? "9+" : unreadInboxCount}
            </span>
          )}
        </Link>
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
              <button
                onClick={onAddFeed}
                className="w-full text-left px-4 py-3 min-h-[44px] hover:bg-gray-100 flex items-center gap-2"
                role="menuitem"
              >
                <Rss size={16} />
                Add RSS Feed
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
