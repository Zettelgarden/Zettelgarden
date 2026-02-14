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
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
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
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
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
