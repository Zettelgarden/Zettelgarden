# RSS Feed Management Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a dedicated RSS feed and folder management page at `/app/rss/manage` with split-panel layout, sortable table, and bulk operations.

**Architecture:** Separate page component with left folder panel and right feeds table. Reuse existing RSS API and dialog components. New bulk operation dialogs for multi-feed actions.

**Tech Stack:** React, TypeScript, existing RSS API (`src/api/rss.ts`), Tailwind CSS

---

## Task 1: Add Route for Manage Page

**Files:**
- Modify: `zettelkasten-front/src/App.tsx`

**Step 1: Add route import and path**

Add the new route to App.tsx alongside existing RSS routes:

```tsx
// In App.tsx, add import
import { RssManagePage } from "./pages/RssManagePage";

// In routes section, add this line after the RSS route
<Route path="/app/rss/manage" element={<RssManagePage />} />
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/App.tsx
git commit -m "feat: add route for RSS management page"
```

---

## Task 2: Create Main Page Component Structure

**Files:**
- Create: `zettelkasten-front/src/pages/RssManagePage.tsx`

**Step 1: Create basic page shell**

```tsx
import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { setDocumentTitle } from "../utils/title";
import {
  RSSFeed,
  RSSFolder,
  UnreadCounts,
  listFeeds,
  listFolders,
  getUnreadCounts,
  updateFeed,
  deleteFeed,
  updateFolder,
  deleteFolder,
  createFolder,
  refreshFeeds,
  markFeedAsRead,
} from "../api/rss";
import { RssManageFolderPanel } from "../components/rss/RssManageFolderPanel";
import { RssManageFeedsTable } from "../components/rss/RssManageFeedsTable";

export function RssManagePage() {
  const navigate = useNavigate();
  const [feeds, setFeeds] = useState<RSSFeed[]>([]);
  const [folders, setFolders] = useState<RSSFolder[]>([]);
  const [unreadCounts, setUnreadCounts] = useState<UnreadCounts>({ folders: {}, feeds: {} });
  const [loading, setLoading] = useState(true);
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null); // null = all feeds
  const [refreshing, setRefreshing] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [successMessage, setSuccessMessage] = useState("");

  // Load initial data
  useEffect(() => {
    const loadData = async () => {
      try {
        const [feedsData, foldersData, countsData] = await Promise.all([
          listFeeds(),
          listFolders(),
          getUnreadCounts(),
        ]);
        setFeeds(feedsData);
        setFolders(foldersData);
        setUnreadCounts(countsData);
      } catch (error) {
        console.error("Failed to load RSS data:", error);
        setErrorMessage("Failed to load feeds and folders");
      } finally {
        setLoading(false);
      }
    };
    loadData();
  }, []);

  // Set page title
  useEffect(() => {
    setDocumentTitle("Manage RSS Feeds");
  }, []);

  const handleRefresh = async () => {
    setRefreshing(true);
    setErrorMessage("");
    try {
      const result = await refreshFeeds();
      setSuccessMessage(`Refreshed ${result.fetched} feeds`);
      // Reload data after refresh
      const [feedsData, countsData] = await Promise.all([
        listFeeds(),
        getUnreadCounts(),
      ]);
      setFeeds(feedsData);
      setUnreadCounts(countsData);
      setTimeout(() => setSuccessMessage(""), 3000);
    } catch (error) {
      console.error("Failed to refresh feeds:", error);
      setErrorMessage("Failed to refresh feeds");
      setTimeout(() => setErrorMessage(""), 3000);
    } finally {
      setRefreshing(false);
    }
  };

  const handleBackToReader = () => {
    navigate("/app/rss");
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-gray-500">Loading...</div>
      </div>
    );
  }

  // Filter feeds by selected folder
  const filteredFeeds = selectedFolder === null
    ? feeds
    : feeds.filter(f => f.folder === selectedFolder);

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={handleBackToReader}
            className="text-gray-500 hover:text-gray-700"
            aria-label="Back to RSS reader"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <div>
            <h1 className="text-xl font-semibold">Manage RSS Feeds</h1>
            <p className="text-sm text-gray-500">
              <button onClick={handleBackToReader} className="text-blue-600 hover:underline">RSS</button>
              {" → Manage Feeds"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {errorMessage && (
            <div className="text-sm text-red-600">{errorMessage}</div>
          )}
          {successMessage && (
            <div className="text-sm text-green-600">{successMessage}</div>
          )}
          <button
            onClick={handleRefresh}
            disabled={refreshing}
            className="px-4 py-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 disabled:opacity-50 flex items-center gap-2"
          >
            <svg className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {refreshing ? "Refreshing..." : "Refresh"}
          </button>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left Panel - Folders */}
        <RssManageFolderPanel
          folders={folders}
          feeds={feeds}
          unreadCounts={unreadCounts}
          selectedFolder={selectedFolder}
          onSelectFolder={setSelectedFolder}
          onCreateFolder={async (name) => {
            const newFolder = await createFolder({ name });
            setFolders(prev => [...prev, newFolder]);
            return newFolder;
          }}
          onRenameFolder={async (id, name) => {
            const updated = await updateFolder(id, { name });
            setFolders(prev => prev.map(f => f.id === id ? updated : f));
            // Update feeds that reference this folder
            const oldFolder = folders.find(f => f.id === id);
            if (oldFolder) {
              setFeeds(prev => prev.map(f =>
                f.folder === oldFolder.name ? { ...f, folder: name } : f
              ));
            }
            if (selectedFolder === oldFolder?.name) {
              setSelectedFolder(name);
            }
          }}
          onDeleteFolder={async (id) => {
            await deleteFolder(id);
            const folder = folders.find(f => f.id === id);
            // Remove feeds in this folder
            if (folder) {
              setFeeds(prev => prev.filter(f => f.folder !== folder.name));
            }
            setFolders(prev => prev.filter(f => f.id !== id));
            if (selectedFolder === folder?.name) {
              setSelectedFolder(null);
            }
          }}
        />

        {/* Right Panel - Feeds Table */}
        <RssManageFeedsTable
          feeds={filteredFeeds}
          folders={folders}
          unreadCounts={unreadCounts}
          onUpdateFeed={async (id, params) => {
            const updated = await updateFeed(id, params);
            setFeeds(prev => prev.map(f => f.id === id ? updated : f));
            return updated;
          }}
          onDeleteFeed={async (id) => {
            await deleteFeed(id);
            setFeeds(prev => prev.filter(f => f.id !== id));
          }}
          onBulkUpdate={async (feedIds, params) => {
            const updates = feedIds.map(id => updateFeed(id, params));
            await Promise.all(updates);
            setFeeds(prev => prev.map(f =>
              feedIds.includes(f.id) ? { ...f, ...params } : f
            ));
          }}
          onBulkDelete={async (feedIds) => {
            await Promise.all(feedIds.map(id => deleteFeed(id)));
            setFeeds(prev => prev.filter(f => !feedIds.includes(f.id)));
          }}
          refreshing={refreshing}
        />
      </div>
    </div>
  );
}
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/pages/RssManagePage.tsx
git commit -m "feat: create RSS management page structure"
```

---

## Task 3: Create Folder Panel Component

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssManageFolderPanel.tsx`

**Step 1: Create folder panel component**

```tsx
import React, { useState } from "react";
import { RSSFolder, RSSFeed, UnreadCounts } from "../../api/rss";

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
  const [newFolderName, setNewFolderName] = useState("");
  const [renameValue, setRenameValue] = useState("");

  const getFeedCountForFolder = (folderName: string | null) => {
    return feeds.filter(f => f.folder === folderName).length;
  };

  const getUnreadCountForFolder = (folderName: string | null) => {
    if (folderName === null) {
      return Object.values(unreadCounts.feeds).reduce((sum, count) => sum + count, 0);
    }
    const folderFeeds = feeds.filter(f => f.folder === folderName);
    return folderFeeds.reduce((sum, feed) => sum + (unreadCounts.feeds[feed.id] || 0), 0);
  };

  const handleCreateFolder = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newFolderName.trim()) return;
    try {
      await onCreateFolder(newFolderName.trim());
      setNewFolderName("");
      setShowCreateDialog(false);
    } catch (error) {
      console.error("Failed to create folder:", error);
    }
  };

  const handleRenameFolder = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingFolder || !renameValue.trim()) return;
    try {
      await onRenameFolder(editingFolder.id, renameValue.trim());
      setEditingFolder(null);
      setRenameValue("");
    } catch (error) {
      console.error("Failed to rename folder:", error);
    }
  };

  const handleDeleteFolder = async (folder: RSSFolder) => {
    const feedCount = getFeedCountForFolder(folder.name);
    if (feedCount > 0) {
      const move = confirm(`Folder "${folder.name}" contains ${feedCount} feed(s). Click OK to move them to Uncategorized, or Cancel to delete all feeds in this folder.`);
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
        <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Folders</h2>

        {/* All Feeds */}
        <button
          onClick={() => onSelectFolder(null)}
          className={`w-full text-left px-3 py-2 rounded-md mb-2 transition-colors flex items-center justify-between ${
            selectedFolder === null
              ? "bg-blue-100 text-blue-900"
              : "hover:bg-gray-100"
          }`}
        >
          <span className="font-medium">All Feeds</span>
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-500">{feeds.length}</span>
            {totalUnread > 0 && (
              <span className="bg-red-500 text-white text-xs font-bold px-1.5 py-0.5 rounded-full">
                {totalUnread > 99 ? "99+" : totalUnread}
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
                  isSelected ? "bg-blue-100 text-blue-900" : "hover:bg-gray-100"
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
                        {unreadCount > 99 ? "99+" : unreadCount}
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
                    <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                    </svg>
                  </button>
                  <button
                    onClick={() => setDeletingFolder(folder)}
                    className="p-1 text-gray-400 hover:text-red-600"
                    title="Delete folder"
                  >
                    <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
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
              <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
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
                  setNewFolderName("");
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
                    setRenameValue("");
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
                <> It contains {getFeedCountForFolder(deletingFolder.name)} feed(s).</>
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
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssManageFolderPanel.tsx
git commit -m "feat: create folder management panel component"
```

---

## Task 4: Create Feeds Table Component (Basic)

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssManageFeedsTable.tsx`

**Step 1: Create feeds table component with basic structure**

```tsx
import React, { useState, useMemo } from "react";
import { RSSFeed, RSSFolder, UnreadCounts } from "../../api/rss";

interface RssManageFeedsTableProps {
  feeds: RSSFeed[];
  folders: RSSFolder[];
  unreadCounts: UnreadCounts;
  onUpdateFeed: (id: number, params: any) => Promise<RSSFeed>;
  onDeleteFeed: (id: number) => Promise<void>;
  onBulkUpdate: (feedIds: number[], params: any) => Promise<void>;
  onBulkDelete: (feedIds: number[]) => Promise<void>;
  refreshing: boolean;
}

type SortField = 'name' | 'url' | 'folder' | 'last_fetched' | 'unread';
type SortOrder = 'asc' | 'desc';

export function RssManageFeedsTable({
  feeds,
  folders,
  unreadCounts,
  onUpdateFeed,
  onDeleteFeed,
  onBulkUpdate,
  onBulkDelete,
  refreshing,
}: RssManageFeedsTableProps) {
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [sortField, setSortField] = useState<SortField>('name');
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc');
  const [searchQuery, setSearchQuery] = useState("");
  const [editingFeed, setEditingFeed] = useState<RSSFeed | null>(null);
  const [showBulkMove, setShowBulkMove] = useState(false);
  const [showBulkTags, setShowBulkTags] = useState(false);
  const [showBulkDelete, setShowBulkDelete] = useState(false);

  // Filter and sort feeds
  const filteredAndSortedFeeds = useMemo(() => {
    let result = [...feeds];

    // Filter by search
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter(f =>
        f.name.toLowerCase().includes(query) ||
        f.url.toLowerCase().includes(query)
      );
    }

    // Sort
    result.sort((a, b) => {
      let aVal: any, bVal: any;

      switch (sortField) {
        case 'name':
          aVal = a.name.toLowerCase();
          bVal = b.name.toLowerCase();
          break;
        case 'url':
          aVal = a.url.toLowerCase();
          bVal = b.url.toLowerCase();
          break;
        case 'folder':
          aVal = a.folder || '';
          bVal = b.folder || '';
          break;
        case 'last_fetched':
          aVal = a.last_fetched_at ? new Date(a.last_fetched_at).getTime() : 0;
          bVal = b.last_fetched_at ? new Date(b.last_fetched_at).getTime() : 0;
          break;
        case 'unread':
          aVal = unreadCounts.feeds[a.id] || 0;
          bVal = unreadCounts.feeds[b.id] || 0;
          break;
        default:
          return 0;
      }

      if (aVal < bVal) return sortOrder === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortOrder === 'asc' ? 1 : -1;
      return 0;
    });

    return result;
  }, [feeds, searchQuery, sortField, sortOrder, unreadCounts]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('asc');
    }
  };

  const handleSelectAll = () => {
    if (selectedIds.size === filteredAndSortedFeeds.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(filteredAndSortedFeeds.map(f => f.id)));
    }
  };

  const handleSelectOne = (id: number) => {
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setSelectedIds(next);
  };

  const handleBulkEnable = async () => {
    await onBulkUpdate(Array.from(selectedIds), { enabled: true });
    setSelectedIds(new Set());
  };

  const handleBulkDisable = async () => {
    await onBulkUpdate(Array.from(selectedIds), { enabled: false });
    setSelectedIds(new Set());
  };

  const handleBulkDeleteConfirm = async () => {
    await onBulkDelete(Array.from(selectedIds));
    setSelectedIds(new Set());
    setShowBulkDelete(false);
  };

  const formatRelativeTime = (dateStr?: string) => {
    if (!dateStr) return "Never";
    const date = new Date(dateStr);
    const now = new Date();
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

    if (seconds < 60) return "Just now";
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
    return date.toLocaleDateString();
  };

  const getUnreadCount = (feedId: number) => {
    return unreadCounts.feeds[feedId] || 0;
  };

  return (
    <div className="flex-1 overflow-hidden flex flex-col">
      {/* Bulk Actions Toolbar */}
      {selectedIds.size > 0 && (
        <div className="bg-blue-50 border-b border-blue-200 px-4 py-3 flex items-center gap-3">
          <span className="text-sm font-medium text-blue-900">
            {selectedIds.size} feed{selectedIds.size !== 1 ? 's' : ''} selected
          </span>
          <div className="h-4 w-px bg-blue-300" />
          <button
            onClick={handleBulkEnable}
            className="px-3 py-1 text-sm bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Enable
          </button>
          <button
            onClick={handleBulkDisable}
            className="px-3 py-1 text-sm bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Disable
          </button>
          <button
            onClick={() => setShowBulkMove(true)}
            className="px-3 py-1 text-sm bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Move to folder...
          </button>
          <button
            onClick={() => setShowBulkTags(true)}
            className="px-3 py-1 text-sm bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Set tags...
          </button>
          <button
            onClick={() => setShowBulkDelete(true)}
            className="px-3 py-1 text-sm bg-white border border-red-300 text-red-600 rounded-md hover:bg-red-50"
          >
            Delete
          </button>
          <button
            onClick={() => setSelectedIds(new Set())}
            className="ml-auto text-sm text-blue-600 hover:underline"
          >
            Clear selection
          </button>
        </div>
      )}

      {/* Search and Filter Bar */}
      <div className="bg-white border-b border-gray-200 px-4 py-3 flex items-center gap-4">
        <div className="relative flex-1">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search feeds by name or URL..."
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
          <svg className="w-5 h-5 absolute left-3 top-2.5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
        <div className="text-sm text-gray-500">
          {filteredAndSortedFeeds.length} feed{filteredAndSortedFeeds.length !== 1 ? 's' : ''}
        </div>
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full">
          <thead className="bg-gray-50 sticky top-0">
            <tr>
              <th className="px-4 py-3 text-left">
                <input
                  type="checkbox"
                  checked={selectedIds.size === filteredAndSortedFeeds.length && filteredAndSortedFeeds.length > 0}
                  onChange={handleSelectAll}
                  className="rounded border-gray-300"
                />
              </th>
              <th className="px-4 py-3 text-left">
                <button
                  onClick={() => handleSort('name')}
                  className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                >
                  Name
                  {sortField === 'name' && (
                    <span className="text-gray-400">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 text-left">
                <button
                  onClick={() => handleSort('url')}
                  className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                >
                  URL
                  {sortField === 'url' && (
                    <span className="text-gray-400">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 text-left">
                <button
                  onClick={() => handleSort('folder')}
                  className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                >
                  Folder
                  {sortField === 'folder' && (
                    <span className="text-gray-400">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-700">Tags</th>
              <th className="px-4 py-3 text-left font-medium text-gray-700">Status</th>
              <th className="px-4 py-3 text-left">
                <button
                  onClick={() => handleSort('last_fetched')}
                  className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                >
                  Last Fetched
                  {sortField === 'last_fetched' && (
                    <span className="text-gray-400">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-700">Error</th>
              <th className="px-4 py-3 text-left">
                <button
                  onClick={() => handleSort('unread')}
                  className="flex items-center gap-1 font-medium text-gray-700 hover:text-gray-900"
                >
                  Unread
                  {sortField === 'unread' && (
                    <span className="text-gray-400">{sortOrder === 'asc' ? '↑' : '↓'}</span>
                  )}
                </button>
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-700">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {filteredAndSortedFeeds.length === 0 ? (
              <tr>
                <td colSpan={10} className="px-4 py-12 text-center text-gray-500">
                  {searchQuery ? "No feeds match your search." : "No feeds yet."}
                </td>
              </tr>
            ) : (
              filteredAndSortedFeeds.map((feed) => (
                <tr
                  key={feed.id}
                  className={`hover:bg-gray-50 ${selectedIds.has(feed.id) ? 'bg-blue-50' : ''}`}
                >
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      checked={selectedIds.has(feed.id)}
                      onChange={() => handleSelectOne(feed.id)}
                      className="rounded border-gray-300"
                    />
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{feed.name}</span>
                      {feed.priority && (
                        <span className="text-amber-500" title="Priority feed">
                          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                            <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                          </svg>
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="max-w-xs truncate text-gray-500 text-sm" title={feed.url}>
                      {feed.url}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    {feed.folder ? (
                      <span className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded-full">
                        {feed.folder}
                      </span>
                    ) : (
                      <span className="text-gray-400 text-sm">Uncategorized</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {feed.auto_tags ? (
                      <div className="flex flex-wrap gap-1 max-w-xs">
                        {feed.auto_tags.split(',').slice(0, 3).map((tag, i) => (
                          <span key={i} className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded-full">
                            {tag.trim()}
                          </span>
                        ))}
                        {feed.auto_tags.split(',').length > 3 && (
                          <span className="text-xs text-gray-400">+{feed.auto_tags.split(',').length - 3}</span>
                        )}
                      </div>
                    ) : (
                      <span className="text-gray-400 text-sm">None</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      {feed.enabled ? (
                        <span className="text-green-600" title="Enabled">
                          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                          </svg>
                        </span>
                      ) : (
                        <span className="text-gray-400" title="Disabled">
                          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                          </svg>
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {formatRelativeTime(feed.last_fetched_at)}
                  </td>
                  <td className="px-4 py-3">
                    {feed.last_error ? (
                      <span className="text-red-500" title={feed.last_error}>
                        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                        </svg>
                      </span>
                    ) : (
                      <span className="text-gray-300">
                        <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                        </svg>
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {getUnreadCount(feed.id) > 0 && (
                      <span className="bg-red-500 text-white text-xs font-bold px-2 py-0.5 rounded-full">
                        {getUnreadCount(feed.id)}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setEditingFeed(feed)}
                        className="p-1 text-gray-400 hover:text-blue-600"
                        title="Edit feed"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                      </button>
                      <button
                        onClick={() => onDeleteFeed(feed.id)}
                        className="p-1 text-gray-400 hover:text-red-600"
                        title="Delete feed"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Edit Feed Dialog - Reuse existing dialog */}
      {editingFeed && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-[500px]">
            <h3 className="text-lg font-semibold mb-4">Edit Feed</h3>
            <form
              onSubmit={async (e) => {
                e.preventDefault();
                const formData = new FormData(e.currentTarget);
                await onUpdateFeed(editingFeed.id, {
                  name: formData.get('name') as string,
                  url: formData.get('url') as string,
                  folder: formData.get('folder') as string || undefined,
                  auto_tags: formData.get('tags') as string,
                  fetch_interval: parseInt(formData.get('interval') as string) || undefined,
                  enabled: formData.get('enabled') === 'on',
                  priority: formData.get('priority') === 'on',
                });
                setEditingFeed(null);
              }}
            >
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                  <input
                    name="name"
                    defaultValue={editingFeed.name}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">URL</label>
                  <input
                    name="url"
                    defaultValue={editingFeed.url}
                    type="url"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Folder</label>
                  <select
                    name="folder"
                    defaultValue={editingFeed.folder || ''}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  >
                    <option value="">Uncategorized</option>
                    {folders.map(f => (
                      <option key={f.id} value={f.name}>{f.name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Auto Tags (comma-separated)</label>
                  <input
                    name="tags"
                    defaultValue={editingFeed.auto_tags}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Fetch Interval (minutes)</label>
                  <input
                    name="interval"
                    defaultValue={editingFeed.fetch_interval}
                    type="number"
                    min="1"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  />
                </div>
                <div className="flex gap-4">
                  <label className="flex items-center gap-2">
                    <input name="enabled" type="checkbox" defaultChecked={editingFeed.enabled} />
                    <span className="text-sm">Enabled</span>
                  </label>
                  <label className="flex items-center gap-2">
                    <input name="priority" type="checkbox" defaultChecked={editingFeed.priority} />
                    <span className="text-sm">Priority</span>
                  </label>
                </div>
              </div>
              <div className="flex justify-end gap-2 mt-6">
                <button
                  type="button"
                  onClick={() => setEditingFeed(null)}
                  className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  Save
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Bulk Delete Confirmation */}
      {showBulkDelete && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-2">Delete Feeds</h3>
            <p className="text-gray-600 mb-4">
              Are you sure you want to delete {selectedIds.size} feed{selectedIds.size !== 1 ? 's' : ''}?
              This will also delete all articles from these feeds. This action cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowBulkDelete(false)}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
              >
                Cancel
              </button>
              <button
                onClick={handleBulkDeleteConfirm}
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
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssManageFeedsTable.tsx
git commit -m "feat: create feeds table with bulk operations"
```

---

## Task 5: Add Bulk Move Dialog

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssBulkMoveDialog.tsx`

**Step 1: Create bulk move dialog component**

```tsx
import React, { useState } from "react";
import { RSSFolder } from "../../api/rss";

interface RssBulkMoveDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (folder: string | null) => void;
  folders: RSSFolder[];
  feedCount: number;
}

export function RssBulkMoveDialog({
  isOpen,
  onClose,
  onConfirm,
  folders,
  feedCount,
}: RssBulkMoveDialogProps) {
  const [selectedFolder, setSelectedFolder] = useState<string>("");
  const [newFolderName, setNewFolderName] = useState("");
  const [showNewFolderInput, setShowNewFolderInput] = useState(false);

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (showNewFolderInput) {
      onConfirm(newFolderName.trim());
    } else {
      onConfirm(selectedFolder || null);
    }
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-96">
        <h3 className="text-lg font-semibold mb-4">Move {feedCount} Feed{feedCount !== 1 ? 's' : ''}</h3>
        <form onSubmit={handleSubmit}>
          {!showNewFolderInput ? (
            <div className="space-y-3">
              <button
                type="button"
                onClick={() => {
                  setShowNewFolderInput(true);
                  setSelectedFolder("");
                }}
                className="w-full px-3 py-2 text-left border border-gray-300 rounded-md hover:bg-gray-50 flex items-center gap-2"
              >
                <svg className="w-4 h-4 text-green-600" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
                </svg>
                Create new folder...
              </button>
              <div className="border-t border-gray-200 pt-2">
                <p className="text-sm text-gray-500 mb-2">Or select existing folder:</p>
                <div className="space-y-1 max-h-60 overflow-y-auto">
                  <button
                    type="button"
                    onClick={() => setSelectedFolder("")}
                    className={`w-full px-3 py-2 text-left rounded-md ${
                      selectedFolder === "" ? "bg-blue-100" : "hover:bg-gray-50"
                    }`}
                  >
                    Uncategorized
                  </button>
                  {folders.map((folder) => (
                    <button
                      key={folder.id}
                      type="button"
                      onClick={() => setSelectedFolder(folder.name)}
                      className={`w-full px-3 py-2 text-left rounded-md ${
                        selectedFolder === folder.name ? "bg-blue-100" : "hover:bg-gray-50"
                      }`}
                    >
                      {folder.name}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">New folder name</label>
              <input
                type="text"
                value={newFolderName}
                onChange={(e) => setNewFolderName(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                autoFocus
                required
              />
              <button
                type="button"
                onClick={() => {
                  setShowNewFolderInput(false);
                  setNewFolderName("");
                }}
                className="mt-2 text-sm text-blue-600 hover:underline"
              >
                ← Back to folder list
              </button>
            </div>
          )}
          <div className="flex justify-end gap-2 mt-6">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!showNewFolderInput && !selectedFolder && !newFolderName}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              Move
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

**Step 2: Update RssManageFeedsTable to use the dialog**

```tsx
// Add import
import { RssBulkMoveDialog } from "./RssBulkMoveDialog";

// In component, add handler:
const handleBulkMove = async (folder: string | null) => {
  await onBulkUpdate(Array.from(selectedIds), { folder: folder || undefined });
  setSelectedIds(new Set());
  setShowBulkMove(false);
};

// Replace bulk move button onClick with:
// onClick={() => setShowBulkMove(true)}

// Add dialog component at end:
<RssBulkMoveDialog
  isOpen={showBulkMove}
  onClose={() => setShowBulkMove(false)}
  onConfirm={handleBulkMove}
  folders={folders}
  feedCount={selectedIds.size}
/>
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssBulkMoveDialog.tsx zettelkasten-front/src/components/rss/RssManageFeedsTable.tsx
git commit -m "feat: add bulk move to folder dialog"
```

---

## Task 6: Add Bulk Tags Dialog

**Files:**
- Create: `zettelkasten-front/src/components/rss/RssBulkTagsDialog.tsx`

**Step 1: Create bulk tags dialog component**

```tsx
import React, { useState } from "react";

interface RssBulkTagsDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (tags: string, mode: 'replace' | 'append') => void;
  feedCount: number;
}

export function RssBulkTagsDialog({
  isOpen,
  onClose,
  onConfirm,
  feedCount,
}: RssBulkTagsDialogProps) {
  const [tags, setTags] = useState("");
  const [mode, setMode] = useState<'replace' | 'append'>('replace');

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onConfirm(tags.trim(), mode);
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-96">
        <h3 className="text-lg font-semibold mb-4">Set Tags for {feedCount} Feed{feedCount !== 1 ? 's' : ''}</h3>
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">Tags</label>
            <input
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="tech, news, ai (comma-separated)"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
              autoFocus
              required
            />
            <p className="text-xs text-gray-500 mt-1">Enter comma-separated tags</p>
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">Mode</label>
            <div className="space-y-2">
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  name="mode"
                  value="replace"
                  checked={mode === 'replace'}
                  onChange={() => setMode('replace')}
                  className="text-blue-600"
                />
                <span className="text-sm">Replace existing tags</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  name="mode"
                  value="append"
                  checked={mode === 'append'}
                  onChange={() => setMode('append')}
                  className="text-blue-600"
                />
                <span className="text-sm">Append to existing tags</span>
              </label>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
            >
              Apply
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
```

**Step 2: Update RssManageFeedsTable to use the dialog**

```tsx
// Add import
import { RssBulkTagsDialog } from "./RssBulkTagsDialog";

// In component, add handler:
const handleBulkTags = async (tags: string, mode: 'replace' | 'append') => {
  const feedUpdates = Array.from(selectedIds).map(id => {
    const feed = feeds.find(f => f.id === id);
    let finalTags = tags;
    if (mode === 'append' && feed?.auto_tags) {
      const existing = feed.auto_tags.split(',').map(t => t.trim()).filter(t => t);
      const newTags = tags.split(',').map(t => t.trim()).filter(t => t);
      const combined = [...new Set([...existing, ...newTags])];
      finalTags = combined.join(', ');
    }
    return { id, params: { auto_tags: finalTags } };
  });

  for (const { id, params } of feedUpdates) {
    await onUpdateFeed(id, params);
  }
  setSelectedIds(new Set());
  setShowBulkTags(false);
};

// Add dialog component at end:
<RssBulkTagsDialog
  isOpen={showBulkTags}
  onClose={() => setShowBulkTags(false)}
  onConfirm={handleBulkTags}
  feedCount={selectedIds.size}
/>
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssBulkTagsDialog.tsx zettelkasten-front/src/components/rss/RssManageFeedsTable.tsx
git commit -m "feat: add bulk set tags dialog"
```

---

## Task 7: Add Link from RSS Page to Manage Page

**Files:**
- Modify: `zettelkasten-front/src/components/rss/RssFeedsPanel.tsx`
- Modify: `zettelkasten-front/src/components/rss/RssMobileLayout.tsx` (if needed)

**Step 1: Add "Manage Feeds" link to settings menu**

In RssFeedsPanel.tsx, add the new menu item in the settings dropdown:

```tsx
// Inside the settings menu div, after "Import OPML" button:
<button
  onClick={() => {
    onToggleSettingsMenu();
    // Navigate to manage page
    window.location.href = '/app/rss/manage';
  }}
  className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
>
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
  </svg>
  Manage Feeds
</button>
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/components/rss/RssFeedsPanel.tsx
git commit -m "feat: add link to RSS management page"
```

---

## Task 8: Add Mobile Responsive Styles

**Files:**
- Modify: `zettelkasten-front/src/pages/RssManagePage.tsx`
- Modify: `zettelkasten-front/src/components/rss/RssManageFeedsTable.tsx`

**Step 1: Add mobile breakpoint detection to main page**

```tsx
// Add to RssManagePage.tsx state and effect:
const [isMobile, setIsMobile] = useState(() => {
  if (typeof window !== 'undefined') {
    return window.innerWidth < 768;
  }
  return false;
});

useEffect(() => {
  const handleResize = () => {
    setIsMobile(window.innerWidth < 768);
  };
  window.addEventListener('resize', handleResize);
  return () => window.removeEventListener('resize', handleResize);
}, []);

// Update the main content div to conditionally stack:
<div className={`flex flex-1 overflow-hidden ${isMobile ? 'flex-col' : ''}`}>
```

**Step 2: Add responsive table wrapper**

```tsx
// In RssManageFeedsTable.tsx, wrap table in responsive container:
<div className="flex-1 overflow-auto">
  <div className="min-w-[800px]">
    <table className="w-full">
      {/* existing table content */}
    </table>
  </div>
</div>

// Add mobile message when table is too wide:
{isMobile && (
  <div className="md:hidden fixed bottom-0 left-0 right-0 bg-yellow-100 border-t border-yellow-200 p-3 text-sm text-yellow-800">
    Table view is best viewed on larger screens.
  </div>
)}
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/pages/RssManagePage.tsx zettelkasten-front/src/components/rss/RssManageFeedsTable.tsx
git commit -m "feat: add mobile responsive styles to manage page"
```

---

## Task 9: Test the Implementation

**Step 1: Start the development server**

```bash
cd zettelkasten-front
npm start
```

**Step 2: Manual testing checklist**

1. Navigate to `/app/rss` and click "Manage Feeds" in settings menu
2. Verify page loads with feeds and folders displayed
3. Test folder filtering: click different folders, verify feed list updates
4. Test search: type in search box, verify results filter
5. Test sorting: click column headers, verify sort order changes
6. Test single feed edit: click edit on a feed, modify, save
7. Test single feed delete: click delete on a feed, confirm
8. Test bulk selection: select multiple feeds using checkboxes
9. Test bulk enable: select feeds, click Enable
10. Test bulk disable: select feeds, click Disable
11. Test bulk move: select feeds, click Move, select folder
12. Test bulk tags: select feeds, click Set tags, enter tags
13. Test bulk delete: select feeds, click Delete, confirm
14. Test folder creation: click Create Folder, enter name
15. Test folder rename: click rename on folder, change name
16. Test folder deletion: click delete on folder, confirm
17. Test refresh: click Refresh button, verify success message
18. Test back button: click back arrow, verify return to RSS reader
19. Test mobile: resize window below 768px, verify responsive behavior

**Step 3: Commit any fixes found during testing**

```bash
git add .
git commit -m "fix: address issues found during testing"
```

---

## Summary

This implementation plan creates a complete RSS feed management page with:

1. **Main page** (`RssManagePage.tsx`) with header, breadcrumbs, and two-panel layout
2. **Folder panel** (`RssManageFolderPanel.tsx`) for folder management
3. **Feeds table** (`RssManageFeedsTable.tsx`) with sortable columns, search, and bulk actions
4. **Bulk operation dialogs** for moving feeds and setting tags
5. **Responsive design** that works on mobile and desktop

All components reuse existing API functions and follow the project's established patterns.
