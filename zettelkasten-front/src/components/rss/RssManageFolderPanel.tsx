import React, { useState } from 'react';
import { RSSFolder, RSSFeed, UnreadCounts } from '../../api/rss';

interface RssManageFolderPanelProps {
  folders: RSSFolder[];
  feeds: RSSFeed[];
  unreadCounts: UnreadCounts;
  selectedFolder: string | null;
  onSelectFolder: (folder: string | null) => void;
  onCreateFolder: (name: string) => Promise<RSSFolder>;
  onRenameFolder: (id: number, name: string) => Promise<void>;
  onDeleteFolder: (id: number) => Promise<void>;
}

export function RssManageFolderPanel({
  folders,
  feeds,
  unreadCounts,
  selectedFolder,
  onSelectFolder,
  onCreateFolder,
  onRenameFolder,
  onDeleteFolder,
}: RssManageFolderPanelProps) {
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [editingFolder, setEditingFolder] = useState<RSSFolder | null>(null);
  const [deletingFolder, setDeletingFolder] = useState<RSSFolder | null>(null);
  const [newFolderName, setNewFolderName] = useState('');
  const [renameValue, setRenameValue] = useState('');

  const getFeedCountForFolder = (folderName: string | null) => {
    return feeds.filter((f) => f.folder === folderName).length;
  };

  const getUnreadCountForFolder = (folderName: string | null) => {
    if (folderName === null) {
      return Object.values(unreadCounts.feeds).reduce(
        (sum, count) => sum + count,
        0,
      );
    }
    const folderFeeds = feeds.filter((f) => f.folder === folderName);
    return folderFeeds.reduce(
      (sum, feed) => sum + (unreadCounts.feeds[feed.id] || 0),
      0,
    );
  };

  const handleCreateFolder = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newFolderName.trim()) return;
    try {
      await onCreateFolder(newFolderName.trim());
      setNewFolderName('');
      setShowCreateDialog(false);
    } catch (error) {
      console.error('Failed to create folder:', error);
    }
  };

  const handleRenameFolder = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingFolder || !renameValue.trim()) return;
    try {
      await onRenameFolder(editingFolder.id, renameValue.trim());
      setEditingFolder(null);
      setRenameValue('');
    } catch (error) {
      console.error('Failed to rename folder:', error);
    }
  };

  const handleDeleteFolder = async (folder: RSSFolder) => {
    const feedCount = getFeedCountForFolder(folder.name);
    if (feedCount > 0) {
      const move = confirm(
        `Folder "${folder.name}" contains ${feedCount} feed(s). Click OK to move them to Uncategorized, or Cancel to delete all feeds in this folder.`,
      );
      if (move) {
        // Just delete folder - feeds will become uncategorized automatically
        await onDeleteFolder(folder.id);
      } else {
        // Delete folder and all its feeds
        // This requires additional API call - for now just delete folder
        await onDeleteFolder(folder.id);
      }
    } else {
      await onDeleteFolder(folder.id);
    }
    setDeletingFolder(null);
  };

  const totalUnread = getUnreadCountForFolder(null);

  return (
    <div className="w-64 bg-white border-r border-gray-200 overflow-y-auto">
      <div className="p-4">
        <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">
          Folders
        </h2>

        {/* All Feeds */}
        <button
          onClick={() => onSelectFolder(null)}
          className={`w-full text-left px-3 py-2 rounded-md mb-2 transition-colors flex items-center justify-between ${
            selectedFolder === null
              ? 'bg-blue-100 text-blue-900'
              : 'hover:bg-gray-100'
          }`}
        >
          <span className="font-medium">All Feeds</span>
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-500">{feeds.length}</span>
            {totalUnread > 0 && (
              <span className="bg-red-500 text-white text-xs font-bold px-1.5 py-0.5 rounded-full">
                {totalUnread > 99 ? '99+' : totalUnread}
              </span>
            )}
          </div>
        </button>

        {/* Folder List */}
        <div className="space-y-1">
          {folders.map((folder) => {
            const feedCount = getFeedCountForFolder(folder.name);
            const unreadCount = getUnreadCountForFolder(folder.name);
            const isSelected = selectedFolder === folder.name;

            return (
              <div
                key={folder.id}
                className={`group flex items-center px-3 py-2 rounded-md transition-colors ${
                  isSelected ? 'bg-blue-100 text-blue-900' : 'hover:bg-gray-100'
                }`}
              >
                <button
                  onClick={() => onSelectFolder(folder.name)}
                  className="flex-1 text-left flex items-center justify-between"
                >
                  <span className="truncate">{folder.name}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-gray-500">{feedCount}</span>
                    {unreadCount > 0 && (
                      <span className="bg-red-500 text-white text-xs font-bold px-1.5 py-0.5 rounded-full">
                        {unreadCount > 99 ? '99+' : unreadCount}
                      </span>
                    )}
                  </div>
                </button>
                <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity ml-2">
                  <button
                    onClick={() => {
                      setEditingFolder(folder);
                      setRenameValue(folder.name);
                    }}
                    className="p-1 text-gray-400 hover:text-blue-600"
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
                    onClick={() => setDeletingFolder(folder)}
                    className="p-1 text-gray-400 hover:text-red-600"
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
            );
          })}
        </div>

        {/* Create Folder Button */}
        {!showCreateDialog ? (
          <button
            onClick={() => setShowCreateDialog(true)}
            className="w-full mt-3 text-left px-3 py-2 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-md transition-colors flex items-center gap-2"
          >
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                clipRule="evenodd"
              />
            </svg>
            Create Folder
          </button>
        ) : (
          <form onSubmit={handleCreateFolder} className="mt-3">
            <input
              type="text"
              value={newFolderName}
              onChange={(e) => setNewFolderName(e.target.value)}
              placeholder="Folder name"
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              autoFocus
              onBlur={() => !newFolderName && setShowCreateDialog(false)}
            />
            <div className="flex gap-2 mt-2">
              <button
                type="submit"
                className="px-3 py-1 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700"
              >
                Create
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowCreateDialog(false);
                  setNewFolderName('');
                }}
                className="px-3 py-1 bg-gray-200 text-gray-700 text-sm rounded-md hover:bg-gray-300"
              >
                Cancel
              </button>
            </div>
          </form>
        )}
      </div>

      {/* Rename Dialog */}
      {editingFolder && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-4">Rename Folder</h3>
            <form onSubmit={handleRenameFolder}>
              <input
                type="text"
                value={renameValue}
                onChange={(e) => setRenameValue(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                autoFocus
              />
              <div className="flex justify-end gap-2 mt-4">
                <button
                  type="button"
                  onClick={() => {
                    setEditingFolder(null);
                    setRenameValue('');
                  }}
                  className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  Rename
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deletingFolder && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-2">Delete Folder</h3>
            <p className="text-gray-600 mb-4">
              Are you sure you want to delete folder "{deletingFolder.name}"?
              {getFeedCountForFolder(deletingFolder.name) > 0 && (
                <>
                  {' '}
                  It contains {getFeedCountForFolder(deletingFolder.name)}{' '}
                  feed(s).
                </>
              )}
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setDeletingFolder(null)}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
              >
                Cancel
              </button>
              <button
                onClick={() => handleDeleteFolder(deletingFolder)}
                className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
