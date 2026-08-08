import React, { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { RSSFeed, RSSFolder, UnreadCounts } from '../../api/rss';
import { RssFeedItem } from './RssFeedItem';
import {
  getFeedsByFolder,
  getUnreadCountForFeed,
  getUnreadCountForFolder,
  getTotalUnreadCount,
  renderUnreadBadge,
} from '../../utils/rssHelpers';

interface RssFeedsPanelProps {
  feeds: RSSFeed[];
  folders: RSSFolder[];
  unreadCounts: UnreadCounts;
  selectedFolder: string | null;
  selectedFeedId: number | null;
  showUnreadOnly: boolean;
  isSmartFeedActive: boolean;
  isStarredFeedActive: boolean;
  starredCount?: number;
  expandedFolders: Set<string>;
  refreshMessage: string;
  errorMessage: string;
  refreshing: boolean;
  showSettingsMenu: boolean;
  showFeedMenuId: number | null;
  onSelectAllFeeds: () => void;
  onSelectFolder: (folderName: string) => void;
  onSelectFeed: (feedId: number) => void;
  onSelectSmartFeed: () => void;
  onSelectStarredFeed: () => void;
  onToggleFolder: (folderName: string) => void;
  onToggleShowUnreadOnly: () => void;
  onAddFeed: () => void;
  onCreateFolder: () => void;
  onEditFeed: (feed: RSSFeed) => void;
  onDeleteFeed: (feed: RSSFeed) => void;
  onMarkFeedAsRead: (feed: RSSFeed) => void;
  onEditFolder: (folder: RSSFolder) => void;
  onDeleteFolder: (folder: RSSFolder) => void;
  onMarkFolderAsRead: (folder: RSSFolder) => void;
  onRefresh: () => void;
  onExportOPML: () => void;
  onImportOPML: () => void;
  onToggleSettingsMenu: () => void;
  onToggleFeedMenu: (feedId: number | null) => void;
}

/**
 * Left sidebar panel showing feeds, folders, and settings
 */
export function RssFeedsPanel({
  feeds,
  folders,
  unreadCounts,
  selectedFolder,
  selectedFeedId,
  showUnreadOnly,
  isSmartFeedActive,
  isStarredFeedActive,
  starredCount,
  expandedFolders,
  refreshMessage,
  errorMessage,
  refreshing,
  showSettingsMenu,
  showFeedMenuId,
  onSelectAllFeeds,
  onSelectFolder,
  onSelectFeed,
  onSelectSmartFeed,
  onSelectStarredFeed,
  onToggleFolder,
  onToggleShowUnreadOnly,
  onAddFeed,
  onCreateFolder,
  onEditFeed,
  onDeleteFeed,
  onMarkFeedAsRead,
  onEditFolder,
  onDeleteFolder,
  onMarkFolderAsRead,
  onRefresh,
  onExportOPML,
  onImportOPML,
  onToggleSettingsMenu,
  onToggleFeedMenu,
}: RssFeedsPanelProps) {
  const navigate = useNavigate();
  const settingsMenuRef = useRef<HTMLDivElement>(null);
  const feedMenuRefs = useRef<Map<number, HTMLDivElement>>(new Map());
  const [internalFeedMenuId, setInternalFeedMenuId] = useState<number | null>(
    null,
  );

  // Store the latest callbacks in refs to avoid recreating event listeners
  const onToggleSettingsMenuRef = useRef(onToggleSettingsMenu);
  const onToggleFeedMenuRef = useRef(onToggleFeedMenu);

  useEffect(() => {
    onToggleSettingsMenuRef.current = onToggleSettingsMenu;
    onToggleFeedMenuRef.current = onToggleFeedMenu;
  }, [onToggleSettingsMenu, onToggleFeedMenu]);

  // Sync external showFeedMenuId to internal state when it changes from parent
  useEffect(() => {
    if (showFeedMenuId !== internalFeedMenuId) {
      setInternalFeedMenuId(showFeedMenuId);
    }
  }, [showFeedMenuId]);

  // Close settings menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        settingsMenuRef.current &&
        !settingsMenuRef.current.contains(event.target as Node)
      ) {
        onToggleSettingsMenuRef.current();
      }
    };

    if (showSettingsMenu) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [showSettingsMenu]);

  // Close feed menus when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (internalFeedMenuId !== null) {
        const target = event.target as Node;
        const menuElement = feedMenuRefs.current.get(internalFeedMenuId);
        if (menuElement && !menuElement.contains(target)) {
          setInternalFeedMenuId(null);
          onToggleFeedMenuRef.current(null);
        }
      }
    };

    if (internalFeedMenuId !== null) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [internalFeedMenuId]);

  const totalUnreadCount = getTotalUnreadCount(unreadCounts);

  const handleToggleFeedMenu = (feedId: number | null) => {
    setInternalFeedMenuId(feedId);
    // Notify parent of the change
    if (feedId !== internalFeedMenuId) {
      onToggleFeedMenu(feedId);
    }
  };

  return (
    <div className="hidden md:flex w-64 border-r border-gray-200 p-4 overflow-y-auto bg-gray-50 flex-shrink-0 flex-col">
      <div className="mb-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold">RSS Feeds</h2>
          <div className="relative" ref={settingsMenuRef}>
            <button
              onClick={onToggleSettingsMenu}
              className="p-1.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-md transition-colors"
              aria-label="Settings"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                />
              </svg>
            </button>

            {showSettingsMenu && (
              <div className="absolute right-0 mt-1 w-48 bg-white rounded-md shadow-lg border border-gray-200 py-1 z-50">
                <button
                  onClick={() => {
                    onToggleSettingsMenu();
                    onRefresh();
                  }}
                  disabled={refreshing}
                  className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                >
                  <svg
                    className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`}
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                    />
                  </svg>
                  {refreshing ? 'Refreshing...' : 'Refresh All'}
                </button>
                <button
                  onClick={() => {
                    onToggleSettingsMenu();
                    onExportOPML();
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                    />
                  </svg>
                  Export OPML
                </button>
                <button
                  onClick={() => {
                    onToggleSettingsMenu();
                    onImportOPML();
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
                    />
                  </svg>
                  Import OPML
                </button>
                <button
                  onClick={() => {
                    onToggleSettingsMenu();
                    navigate('/app/rss/manage');
                  }}
                  className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                    />
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                    />
                  </svg>
                  Manage Feeds
                </button>
              </div>
            )}
          </div>
        </div>
        <div className="space-y-2">
          {refreshMessage && (
            <div
              className={`text-sm text-center px-2 py-1 rounded ${
                refreshMessage.includes('Failed')
                  ? 'text-red-600'
                  : 'text-green-600'
              }`}
            >
              {refreshMessage}
            </div>
          )}
          {errorMessage && (
            <div className="text-sm text-center px-2 py-1 rounded bg-red-50 text-red-600">
              {errorMessage}
            </div>
          )}
          <button
            onClick={onAddFeed}
            className="w-full bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700 transition-colors flex items-center justify-center gap-2"
          >
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                clipRule="evenodd"
              />
            </svg>
            Add Feed
          </button>
        </div>
      </div>

      {/* Smart Feed */}
      <div className="mb-3">
        <button
          onClick={onSelectSmartFeed}
          className={`w-full text-left px-3 py-2 rounded-md transition-colors font-medium flex items-center gap-2 ${
            isSmartFeedActive
              ? 'bg-blue-100 text-blue-900'
              : 'hover:bg-gray-100'
          }`}
        >
          <svg
            className="w-4 h-4 text-amber-500"
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
          </svg>
          Smart Feed
        </button>
      </div>

      {/* Starred feed */}
      <button
        onClick={onSelectStarredFeed}
        className={`w-full flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors ${
          isStarredFeedActive
            ? 'bg-amber-100 text-amber-900 font-medium'
            : 'text-gray-700 hover:bg-gray-100'
        }`}
      >
        <svg
          className={`w-5 h-5 ${
            isStarredFeedActive
              ? 'fill-amber-500 text-amber-500'
              : 'text-gray-500'
          }`}
          fill={isStarredFeedActive ? 'currentColor' : 'none'}
          stroke="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={isStarredFeedActive ? 0 : 2}
            d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.364 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.364-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
          />
        </svg>
        <span>Starred</span>
        {starredCount !== undefined && starredCount > 0 && (
          <span className="ml-auto text-xs bg-gray-200 px-2 py-0.5 rounded-full">
            {starredCount}
          </span>
        )}
      </button>

      {/* All Feeds */}
      <div className="mb-3">
        <button
          onClick={onSelectAllFeeds}
          className={`w-full text-left px-3 py-2 rounded-md transition-colors font-medium ${
            !isSmartFeedActive &&
            selectedFolder === null &&
            selectedFeedId === null
              ? 'bg-blue-100 text-blue-900'
              : 'hover:bg-gray-100'
          }`}
        >
          All Feeds ({feeds.length})
        </button>
      </div>

      {/* Folders with Feeds */}
      {folders.length > 0 && (
        <div className="mb-3 space-y-2">
          {folders.map((folder) => {
            const folderFeeds = getFeedsByFolder(feeds, folder.name);
            const isExpanded = expandedFolders.has(folder.name);
            const isSelected =
              selectedFolder === folder.name && selectedFeedId === null;
            const unreadCount = getUnreadCountForFolder(
              unreadCounts,
              folder.name,
            );

            return (
              <div key={folder.id} className="bg-gray-100/50 rounded-lg p-2">
                {/* Folder header */}
                <div
                  className={`group flex items-center rounded-md transition-colors text-sm ${
                    isSelected ? 'bg-amber-100' : 'hover:bg-amber-50'
                  }`}
                >
                  <button
                    onClick={() => {
                      onSelectFolder(folder.name);
                    }}
                    className="flex-1 text-left px-3 py-2 flex items-center gap-1"
                  >
                    <svg
                      className={`w-3 h-3 text-gray-400 transition-transform ${
                        isExpanded ? 'rotate-90' : ''
                      }`}
                      fill="currentColor"
                      viewBox="0 0 20 20"
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleFolder(folder.name);
                      }}
                    >
                      <path
                        fillRule="evenodd"
                        d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                    <span className="font-medium">{folder.name}</span>
                    <span className="text-gray-400 text-xs">
                      ({folderFeeds.length})
                    </span>
                    {renderUnreadBadge(unreadCount)}
                  </button>
                  <div className="flex items-center pr-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onMarkFolderAsRead(folder);
                      }}
                      className="p-1 text-gray-400 hover:text-green-600 transition-colors"
                      aria-label={`Mark folder ${folder.name} as read`}
                      title="Mark all as read"
                    >
                      <svg
                        className="w-3 h-3"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M5 13l4 4L19 7"
                        />
                      </svg>
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onEditFolder(folder);
                      }}
                      className="p-1 text-gray-400 hover:text-blue-600 transition-colors"
                      aria-label={`Rename folder ${folder.name}`}
                      title="Rename folder"
                    >
                      <svg
                        className="w-3 h-3"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                        />
                      </svg>
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onDeleteFolder(folder);
                      }}
                      className="p-1 text-gray-400 hover:text-red-600 transition-colors"
                      aria-label={`Delete folder ${folder.name}`}
                      title="Delete folder"
                    >
                      <svg
                        className="w-3 h-3"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    </button>
                  </div>
                </div>

                {/* Feeds in folder (expanded) */}
                {isExpanded && (
                  <div className="ml-2 space-y-1 mt-1">
                    {folderFeeds.map((feed) => (
                      <RssFeedItem
                        key={feed.id}
                        feed={feed}
                        isSelected={selectedFeedId === feed.id}
                        unreadCount={getUnreadCountForFeed(
                          unreadCounts,
                          feed.id,
                        )}
                        showMenu={internalFeedMenuId === feed.id}
                        onToggleMenu={handleToggleFeedMenu}
                        onSelect={onSelectFeed}
                        onMarkAsRead={onMarkFeedAsRead}
                        onEdit={onEditFeed}
                        onDelete={onDeleteFeed}
                        menuRef={(el) => {
                          if (el) feedMenuRefs.current.set(feed.id, el);
                          else feedMenuRefs.current.delete(feed.id);
                        }}
                      />
                    ))}
                    {folderFeeds.length === 0 && (
                      <div className="px-3 py-1.5 text-xs text-gray-400 italic">
                        No feeds
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Uncategorized Feeds */}
      {(() => {
        const uncategorizedFeeds = getFeedsByFolder(feeds, null);
        if (uncategorizedFeeds.length === 0) return null;

        const isExpanded = expandedFolders.has('__uncategorized__');
        return (
          <div className="mb-3 bg-gray-100/50 rounded-lg p-2">
            <div
              className={`group flex items-center rounded-md transition-colors text-sm ${
                selectedFolder === null && selectedFeedId === null
                  ? 'bg-amber-100'
                  : 'hover:bg-amber-50'
              }`}
            >
              <button
                onClick={() => onToggleFolder('__uncategorized__')}
                className="flex-1 text-left px-1 py-1 flex items-center gap-1"
              >
                <svg
                  className={`w-3 h-3 text-gray-400 transition-transform ${
                    isExpanded ? 'rotate-90' : ''
                  }`}
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
                    clipRule="evenodd"
                  />
                </svg>
                <span className="font-medium text-gray-600">Uncategorized</span>
                <span className="text-gray-400 text-xs">
                  ({uncategorizedFeeds.length})
                </span>
              </button>
            </div>
            {isExpanded && (
              <div className="ml-2 space-y-1 mt-1">
                {uncategorizedFeeds.map((feed) => (
                  <RssFeedItem
                    key={feed.id}
                    feed={feed}
                    isSelected={selectedFeedId === feed.id}
                    unreadCount={getUnreadCountForFeed(unreadCounts, feed.id)}
                    showMenu={internalFeedMenuId === feed.id}
                    onToggleMenu={handleToggleFeedMenu}
                    onSelect={onSelectFeed}
                    onMarkAsRead={onMarkFeedAsRead}
                    onEdit={onEditFeed}
                    onDelete={onDeleteFeed}
                    menuRef={(el) => {
                      if (el) feedMenuRefs.current.set(feed.id, el);
                      else feedMenuRefs.current.delete(feed.id);
                    }}
                  />
                ))}
              </div>
            )}
          </div>
        );
      })()}

      {/* Create Folder Link */}
      <div className="mb-4">
        <button
          onClick={onCreateFolder}
          className="text-xs text-blue-600 hover:text-blue-800 transition-colors flex items-center gap-1"
        >
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
              clipRule="evenodd"
            />
          </svg>
          Create new folder
        </button>
      </div>

      {/* Unread Filter */}
      <div className="mb-4">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={showUnreadOnly}
            onChange={() => onToggleShowUnreadOnly()}
            className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          <span className="text-sm">Unread only</span>
        </label>
      </div>
    </div>
  );
}
