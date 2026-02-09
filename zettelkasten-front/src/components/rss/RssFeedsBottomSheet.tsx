import React, { useEffect, useRef } from "react";
import { RSSFeed, RSSFolder, UnreadCounts } from "../../api/rss";
import {
  getFeedsByFolder,
  getUnreadCountForFeed,
  getUnreadCountForFolder,
  renderUnreadBadge,
} from "../../utils/rssHelpers";

interface RssFeedsBottomSheetProps {
  isOpen: boolean;
  onClose: () => void;
  feeds: RSSFeed[];
  folders: RSSFolder[];
  unreadCounts: UnreadCounts;
  expandedFolders: Set<string>;
  onToggleFolder: (folderName: string) => void;
  onSelectFeed: (feedId: number) => void;
  onSelectFolder: (folderName: string) => void;
  onSelectAllFeeds: () => void;
  onAddFeed: () => void;
  onCreateFolder: () => void;
  onEditFeed: (feed: RSSFeed) => void;
  onDeleteFeed: (feed: RSSFeed) => void;
  onMarkFeedAsRead: (feed: RSSFeed) => void;
  onEditFolder: (folder: RSSFolder) => void;
  onDeleteFolder: (folder: RSSFolder) => void;
  onMarkFolderAsRead: (folder: RSSFolder) => void;
  selectedFeedId: number | null;
  selectedFolder: string | null;
  showFeedMenuId: number | null;
  onShowFeedMenu: (feedId: number | null) => void;
}

export function RssFeedsBottomSheet({
  isOpen,
  onClose,
  feeds,
  folders,
  unreadCounts,
  expandedFolders,
  onToggleFolder,
  onSelectFeed,
  onSelectFolder,
  onSelectAllFeeds,
  onAddFeed,
  onCreateFolder,
  onEditFeed,
  onDeleteFeed,
  onMarkFeedAsRead,
  onEditFolder,
  onDeleteFolder,
  onMarkFolderAsRead,
  selectedFeedId,
  selectedFolder,
  showFeedMenuId,
  onShowFeedMenu,
}: RssFeedsBottomSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);

  // Handle escape key and backdrop click
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      // Prevent body scroll when sheet is open
      document.body.style.overflow = "hidden";
    }

    return () => {
      document.removeEventListener("keydown", handleEscape);
      document.body.style.overflow = "";
    };
  }, [isOpen, onClose]);

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === backdropRef.current) {
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <div
      ref={backdropRef}
      onClick={handleBackdropClick}
      className="fixed inset-0 bg-black/50 z-50 md:hidden"
      style={{ animation: "fade-in 0.2s ease-out" }}
    >
      <div
        ref={sheetRef}
        className="fixed bottom-0 left-0 right-0 bg-white rounded-t-2xl shadow-2xl max-h-[80vh] flex flex-col"
        style={{ animation: "slide-up 0.3s ease-out" }}
      >
        {/* Drag Handle */}
        <div className="flex justify-center pt-3 pb-2 px-4 flex-shrink-0">
          <div className="w-12 h-1.5 bg-gray-300 rounded-full" />
        </div>

        {/* Header */}
        <div className="px-4 pb-3 border-b border-gray-200 flex-shrink-0">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Feeds</h2>
            <button
              onClick={onClose}
              className="p-2 -mr-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
              aria-label="Close"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
          {/* All Feeds Button */}
          <button
            onClick={() => {
              onSelectAllFeeds();
              onClose();
            }}
            className={`w-full text-left px-4 py-3 rounded-lg transition-colors font-medium ${
              selectedFolder === null && selectedFeedId === null
                ? "bg-blue-100 text-blue-900"
                : "hover:bg-gray-100 bg-gray-50"
            }`}
          >
            <div className="flex items-center justify-between">
              <span>All Feeds</span>
              <span className="text-sm text-gray-500">({feeds.length})</span>
            </div>
          </button>

          {/* Folders with Feeds */}
          {folders.length > 0 && (
            <div className="space-y-2">
              {folders.map((folder) => {
                const folderFeeds = getFeedsByFolder(feeds, folder.name);
                const isExpanded = expandedFolders.has(folder.name);
                const isSelected = selectedFolder === folder.name && selectedFeedId === null;
                const unreadCount = getUnreadCountForFolder(unreadCounts, folder.name);

                return (
                  <div key={folder.id} className="bg-gray-50 rounded-lg overflow-hidden">
                    {/* Folder header */}
                    <div
                      className={`px-3 py-2 transition-colors ${
                        isSelected ? "bg-amber-100" : ""
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <button
                          onClick={() => {
                            onSelectFolder(folder.name);
                            onClose();
                          }}
                          className="flex-1 text-left flex items-center gap-2"
                        >
                          <svg
                            className={`w-4 h-4 text-gray-400 transition-transform ${
                              isExpanded ? "rotate-90" : ""
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
                          <span className="text-gray-400 text-xs">({folderFeeds.length})</span>
                          {renderUnreadBadge(unreadCount)}
                        </button>
                        <div className="flex items-center gap-1">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              onMarkFolderAsRead(folder);
                            }}
                            className="p-1.5 text-gray-400 hover:text-green-600 hover:bg-gray-100 rounded-md transition-colors"
                            aria-label={`Mark folder ${folder.name} as read`}
                          >
                            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                            className="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-gray-100 rounded-md transition-colors"
                            aria-label={`Rename folder ${folder.name}`}
                          >
                            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                            className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-gray-100 rounded-md transition-colors"
                            aria-label={`Delete folder ${folder.name}`}
                          >
                            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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
                        <div className="ml-4 mt-1 space-y-1">
                          {folderFeeds.map((feed) => {
                            const unreadCount = getUnreadCountForFeed(unreadCounts, feed.id);
                            const showMenu = showFeedMenuId === feed.id;

                            return (
                              <div
                                key={feed.id}
                                className={`relative group flex items-center gap-2 rounded-md transition-colors ${
                                  selectedFeedId === feed.id ? "bg-blue-100" : "hover:bg-gray-100"
                                }`}
                              >
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    onShowFeedMenu(showMenu ? null : feed.id);
                                  }}
                                  className="p-1 text-gray-400 hover:text-gray-600"
                                  aria-label={`Feed options for ${feed.name}`}
                                >
                                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                    <path d="M10 6a2 2 0 110-4 2 2 0 010 4z" />
                                    <path d="M2 10a2 2 0 114 0 2 2 0 01-4 0z" />
                                    <path d="M10 14a2 2 0 110-4 2 2 0 010 4z" />
                                  </svg>
                                </button>

                                <button
                                  onClick={() => {
                                    onSelectFeed(feed.id);
                                    onClose();
                                  }}
                                  className="flex-1 text-left truncate py-1.5 px-2 text-sm"
                                  title={feed.url}
                                >
                                  <span className="truncate">{feed.name}</span>
                                </button>

                                {renderUnreadBadge(unreadCount)}

                                {/* Dropdown menu */}
                                {showMenu && (
                                  <div className="absolute left-8 top-full mt-1 w-32 bg-white rounded-md shadow-lg border border-gray-200 py-1 z-50">
                                    <button
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        onShowFeedMenu(null);
                                        onMarkFeedAsRead(feed);
                                      }}
                                      className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                                    >
                                      <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path
                                          strokeLinecap="round"
                                          strokeLinejoin="round"
                                          strokeWidth={2}
                                          d="M5 13l4 4L19 7"
                                        />
                                      </svg>
                                      Mark as read
                                    </button>
                                    <button
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        onShowFeedMenu(null);
                                        onEditFeed(feed);
                                      }}
                                      className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                                    >
                                      <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path
                                          strokeLinecap="round"
                                          strokeLinejoin="round"
                                          strokeWidth={2}
                                          d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                                        />
                                      </svg>
                                      Edit
                                    </button>
                                    <button
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        onShowFeedMenu(null);
                                        onDeleteFeed(feed);
                                      }}
                                      className="w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-gray-100 flex items-center gap-2"
                                    >
                                      <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path
                                          strokeLinecap="round"
                                          strokeLinejoin="round"
                                          strokeWidth={2}
                                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                                        />
                                      </svg>
                                      Delete
                                    </button>
                                  </div>
                                )}
                              </div>
                            );
                          })}
                          {folderFeeds.length === 0 && (
                            <div className="px-3 py-2 text-xs text-gray-400 italic">No feeds</div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {/* Uncategorized Feeds */}
          {(() => {
            const uncategorizedFeeds = getFeedsByFolder(feeds, null);
            if (uncategorizedFeeds.length === 0) return null;

            const isExpanded = expandedFolders.has("__uncategorized__");
            return (
              <div className="bg-gray-50 rounded-lg overflow-hidden">
                <div className="px-3 py-2">
                  <div className="flex items-center justify-between">
                    <button
                      onClick={() => onToggleFolder("__uncategorized__")}
                      className="flex-1 text-left flex items-center gap-2"
                    >
                      <svg
                        className={`w-4 h-4 text-gray-400 transition-transform ${
                          isExpanded ? "rotate-90" : ""
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
                      <span className="text-gray-400 text-xs">({uncategorizedFeeds.length})</span>
                    </button>
                  </div>
                  {isExpanded && (
                    <div className="ml-4 mt-1 space-y-1">
                      {uncategorizedFeeds.map((feed) => {
                        const unreadCount = getUnreadCountForFeed(unreadCounts, feed.id);
                        const showMenu = showFeedMenuId === feed.id;

                        return (
                          <div
                            key={feed.id}
                            className={`relative group flex items-center gap-2 rounded-md transition-colors ${
                              selectedFeedId === feed.id ? "bg-blue-100" : "hover:bg-gray-100"
                            }`}
                          >
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                onShowFeedMenu(showMenu ? null : feed.id);
                              }}
                              className="p-1 text-gray-400 hover:text-gray-600"
                              aria-label={`Feed options for ${feed.name}`}
                            >
                              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                <path d="M10 6a2 2 0 110-4 2 2 0 010 4z" />
                                <path d="M2 10a2 2 0 114 0 2 2 0 01-4 0z" />
                                <path d="M10 14a2 2 0 110-4 2 2 0 010 4z" />
                              </svg>
                            </button>

                            <button
                              onClick={() => {
                                onSelectFeed(feed.id);
                                onClose();
                              }}
                              className="flex-1 text-left truncate py-1.5 px-2 text-sm"
                              title={feed.url}
                            >
                              <span className="truncate">{feed.name}</span>
                            </button>

                            {renderUnreadBadge(unreadCount)}

                            {/* Dropdown menu */}
                            {showMenu && (
                              <div className="absolute left-8 top-full mt-1 w-32 bg-white rounded-md shadow-lg border border-gray-200 py-1 z-50">
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    onShowFeedMenu(null);
                                    onMarkFeedAsRead(feed);
                                  }}
                                  className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                                >
                                  <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path
                                      strokeLinecap="round"
                                      strokeLinejoin="round"
                                      strokeWidth={2}
                                      d="M5 13l4 4L19 7"
                                    />
                                  </svg>
                                  Mark as read
                                </button>
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    onShowFeedMenu(null);
                                    onEditFeed(feed);
                                  }}
                                  className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                                >
                                  <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path
                                      strokeLinecap="round"
                                      strokeLinejoin="round"
                                      strokeWidth={2}
                                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                                    />
                                  </svg>
                                  Edit
                                </button>
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    onShowFeedMenu(null);
                                    onDeleteFeed(feed);
                                  }}
                                  className="w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-gray-100 flex items-center gap-2"
                                >
                                  <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path
                                      strokeLinecap="round"
                                      strokeLinejoin="round"
                                      strokeWidth={2}
                                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                                    />
                                  </svg>
                                  Delete
                                </button>
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              </div>
            );
          })()}
        </div>

        {/* Bottom Actions */}
        <div className="px-4 py-3 border-t border-gray-200 space-y-2 flex-shrink-0">
          <button
            onClick={() => {
              onAddFeed();
              onClose();
            }}
            className="w-full bg-green-600 text-white px-4 py-3 rounded-lg hover:bg-green-700 transition-colors flex items-center justify-center gap-2 font-medium"
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                clipRule="evenodd"
              />
            </svg>
            Add Feed
          </button>
          <button
            onClick={() => {
              onCreateFolder();
              onClose();
            }}
            className="w-full bg-gray-100 text-gray-700 px-4 py-3 rounded-lg hover:bg-gray-200 transition-colors flex items-center justify-center gap-2 font-medium"
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                clipRule="evenodd"
              />
            </svg>
            Create Folder
          </button>
        </div>

        <style>{`
          @keyframes fade-in {
            from {
              opacity: 0;
            }
            to {
              opacity: 1;
            }
          }
          @keyframes slide-up {
            from {
              transform: translateY(100%);
            }
            to {
              transform: translateY(0);
            }
          }
        `}</style>
      </div>
    </div>
  );
}
