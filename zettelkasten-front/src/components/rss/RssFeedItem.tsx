import React from "react";
import { RSSFeed } from "../../api/rss";

interface RssFeedItemProps {
  feed: RSSFeed;
  isSelected: boolean;
  unreadCount: number;
  showMenu: boolean;
  onToggleMenu: (feedId: number) => void;
  onSelect: (feedId: number) => void;
  onMarkAsRead: (feed: RSSFeed) => void;
  onEdit: (feed: RSSFeed) => void;
  onDelete: (feed: RSSFeed) => void;
  menuRef?: (el: HTMLDivElement | null) => void;
}

/**
 * Reusable feed item component for displaying RSS feeds
 * Used in both folder feeds and uncategorized feeds sections
 */
export function RssFeedItem({
  feed,
  isSelected,
  unreadCount,
  showMenu,
  onToggleMenu,
  onSelect,
  onMarkAsRead,
  onEdit,
  onDelete,
  menuRef,
}: RssFeedItemProps) {
  return (
    <div
      key={feed.id}
      className={`relative group flex items-center gap-1 rounded-md transition-colors text-sm ${
        isSelected ? "bg-blue-50" : "hover:bg-gray-50"
      }`}
      ref={menuRef}
    >
      {/* Settings button on left */}
      <button
        onClick={(e) => {
          e.stopPropagation();
          onToggleMenu(showMenu ? -1 : feed.id);
        }}
        className="p-1 text-gray-400 hover:text-gray-600 opacity-0 group-hover:opacity-100 transition-opacity"
        aria-label={`Feed options for ${feed.name}`}
      >
        <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
          <path d="M10 6a2 2 0 110-4 2 2 0 010 4z" />
          <path d="M2 10a2 2 0 114 0 2 2 0 01-4 0z" />
          <path d="M10 14a2 2 0 110-4 2 2 0 010 4z" />
        </svg>
      </button>

      {/* Feed name in middle */}
      <button
        onClick={() => {
          onSelect(feed.id);
          onToggleMenu(-1);
        }}
        className="flex-1 text-left truncate"
        title={feed.url}
      >
        <span className="truncate">{feed.name}</span>
      </button>

      {/* Unread badge on right */}
      {unreadCount > 0 && (
        <span className="ml-1.5 bg-red-500 text-white text-xs font-bold px-1.5 py-0.5 rounded-full min-w-[1.25rem] text-center">
          {unreadCount > 99 ? "99+" : unreadCount}
        </span>
      )}

      {/* Dropdown menu */}
      {showMenu && (
        <div className="absolute left-8 top-full mt-1 w-32 bg-white rounded-md shadow-lg border border-gray-200 py-1 z-50">
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggleMenu(-1);
              onMarkAsRead(feed);
            }}
            className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
          >
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            Mark as read
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggleMenu(-1);
              onEdit(feed);
            }}
            className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
          >
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
            Edit
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggleMenu(-1);
              onDelete(feed);
            }}
            className="w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-gray-100 flex items-center gap-2"
          >
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            Delete
          </button>
        </div>
      )}
    </div>
  );
}
