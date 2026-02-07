import React, { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { setDocumentTitle } from "../utils/title";
import { safeHtmlToMarkdown } from "../utils/markdown";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  listFeeds,
  listArticles,
  listFolders,
  markAsRead,
  convertToCard,
  refreshFeeds,
  deleteFeed,
  deleteFolder,
  getUnreadCounts,
  exportOPML,
  importOPML,
  RSSFeed,
  RSSArticle,
  RSSFolder,
  ArticleFilters,
  UnreadCounts,
  OPMLImportResult,
} from "../api/rss";
import { RssAddFeedDialog } from "../components/rss/RssAddFeedDialog";
import { RssEditFeedDialog } from "../components/rss/RssEditFeedDialog";
import { RssEditFolderDialog } from "../components/rss/RssEditFolderDialog";
import { RssCreateFolderDialog } from "../components/rss/RssCreateFolderDialog";
import { RssConfirmDialog } from "../components/rss/RssConfirmDialog";
import { RssConvertDialog } from "../components/rss/RssConvertDialog";
import { RssImportDialog } from "../components/rss/RssImportDialog";
import { useRSS } from "../contexts/RSSContext";

export function RssPage() {
  const navigate = useNavigate();
  const { setUnreadCount } = useRSS();
  const [feeds, setFeeds] = useState<RSSFeed[]>([]);
  const [articles, setArticles] = useState<RSSArticle[]>([]);
  const [folders, setFolders] = useState<RSSFolder[]>([]);
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [selectedFeedId, setSelectedFeedId] = useState<number | null>(null);
  const [selectedArticle, setSelectedArticle] = useState<RSSArticle | null>(null);
  const [showUnreadOnly, setShowUnreadOnly] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [showAddFeedDialog, setShowAddFeedDialog] = useState(false);
  const [showEditFeedDialog, setShowEditFeedDialog] = useState(false);
  const [showEditFolderDialog, setShowEditFolderDialog] = useState(false);
  const [showCreateFolderDialog, setShowCreateFolderDialog] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [editingFeed, setEditingFeed] = useState<RSSFeed | null>(null);
  const [editingFolder, setEditingFolder] = useState<RSSFolder | null>(null);
  const [deletingType, setDeletingType] = useState<"feed" | "folder" | null>(null);
  const [deletingItem, setDeletingItem] = useState<RSSFeed | RSSFolder | null>(null);
  const [showConvertDialog, setShowConvertDialog] = useState(false);
  const [refreshMessage, setRefreshMessage] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());
  const [unreadCounts, setUnreadCounts] = useState<UnreadCounts>({ folders: {}, feeds: {} });
  const [markingAsRead, setMarkingAsRead] = useState<Set<number>>(new Set());
  const [showImportDialog, setShowImportDialog] = useState(false);
  const [importResult, setImportResult] = useState<OPMLImportResult | null>(null);
  const [importing, setImporting] = useState(false);
  const [showSettingsMenu, setShowSettingsMenu] = useState(false);
  const [showFeedMenuId, setShowFeedMenuId] = useState<number | null>(null);
  const settingsMenuRef = useRef<HTMLDivElement>(null);
  const feedMenuRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const [totalArticles, setTotalArticles] = useState(0);
  const articlesPerPage = 20;

  // Calculate total unread count
  const totalUnreadCount = useMemo(() => {
    return Object.values(unreadCounts.feeds).reduce((sum, count) => sum + count, 0);
  }, [unreadCounts]);

  // Update page title with unread count
  useEffect(() => {
    if (totalUnreadCount > 0) {
      setDocumentTitle(`RSS (${totalUnreadCount})`);
    } else {
      setDocumentTitle("RSS");
    }
  }, [totalUnreadCount]);

  // Update RSS context when unread count changes
  useEffect(() => {
    setUnreadCount(totalUnreadCount);
  }, [totalUnreadCount, setUnreadCount]);

  useEffect(() => {
    loadData();
  }, []);

  // Close settings menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (settingsMenuRef.current && !settingsMenuRef.current.contains(event.target as Node)) {
        setShowSettingsMenu(false);
      }
    };

    if (showSettingsMenu) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [showSettingsMenu]);

  // Close feed menus when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (showFeedMenuId !== null) {
        const target = event.target as Node;
        feedMenuRefs.current.forEach((_, feedId) => {
          const el = feedMenuRefs.current.get(feedId);
          if (el && !el.contains(target)) {
            setShowFeedMenuId(null);
          }
        });
      }
    };

    if (showFeedMenuId !== null) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [showFeedMenuId]);

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [selectedFolder, selectedFeedId, showUnreadOnly]);

  useEffect(() => {
    loadArticles();
  }, [selectedFolder, selectedFeedId, showUnreadOnly, currentPage]);

  const loadData = async () => {
    setLoading(true);
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
    } finally {
      setLoading(false);
    }
  };

  const loadArticles = async () => {
    try {
      const filters: ArticleFilters = {
        limit: articlesPerPage,
        offset: (currentPage - 1) * articlesPerPage
      };
      if (selectedFolder) filters.folder = selectedFolder;
      if (selectedFeedId) filters.feed_id = selectedFeedId;
      if (showUnreadOnly) filters.unread = true;

      const response = await listArticles(filters);
      setArticles(response.articles);
      setTotalArticles(response.total);
      setErrorMessage("");
    } catch (error) {
      console.error("Failed to load articles:", error);
      setErrorMessage("Failed to load articles. Please try again.");
      setTimeout(() => setErrorMessage(""), 5000);
    }
  };

  const refreshUnreadCounts = useCallback(async () => {
    try {
      const counts = await getUnreadCounts();
      setUnreadCounts(counts);
    } catch (error) {
      console.error("Failed to refresh unread counts:", error);
      // Non-blocking error - counts will update on next refresh
    }
  }, []);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    setRefreshMessage("");
    try {
      const result = await refreshFeeds();
      await loadData();
      await loadArticles();
      setRefreshMessage(`Refreshed ${result.fetched} feeds`);
      setTimeout(() => setRefreshMessage(""), 3000);
    } catch (error) {
      console.error("Failed to refresh feeds:", error);
      setRefreshMessage("Failed to refresh feeds");
      setTimeout(() => setRefreshMessage(""), 3000);
    } finally {
      setRefreshing(false);
    }
  }, []);

  const handleArticleClick = useCallback(async (article: RSSArticle) => {
    // Prevent duplicate requests for the same article
    if (markingAsRead.has(article.id)) {
      return;
    }

    setSelectedArticle(article);
    if (!article.read) {
      setMarkingAsRead((prev) => new Set(prev).add(article.id));
      try {
        await markAsRead(article.id, true);
        setArticles((prev) =>
          prev.map((a) => (a.id === article.id ? { ...a, read: true } : a))
        );
        // Refresh unread counts (fire and forget, non-blocking)
        refreshUnreadCounts().catch(() => {
          // Silently fail - counts will update on next refresh
        });
      } catch (error) {
        console.error("Failed to mark as read:", error);
        setErrorMessage("Failed to mark article as read. Please try again.");
        setTimeout(() => setErrorMessage(""), 3000);
      } finally {
        setMarkingAsRead((prev) => {
          const next = new Set(prev);
          next.delete(article.id);
          return next;
        });
      }
    }
  }, [markingAsRead, refreshUnreadCounts]);

  const handleFeedAdded = (feed: RSSFeed) => {
    setFeeds((prev) => [...prev, feed]);
    setShowAddFeedDialog(false);
  };

  const handleConvertClick = () => {
    setShowConvertDialog(true);
  };

  const handleConverted = (cardId: number) => {
    setShowConvertDialog(false);
    // Navigate to the new card
    navigate(`/app/card/${cardId}`);
  };

  const getFeedName = (feedId: number): string => {
    const feed = feeds.find((f) => f.id === feedId);
    return feed?.name || "Unknown Feed";
  };

  const toggleFolderExpanded = (folderName: string) => {
    setExpandedFolders((prev) => {
      const next = new Set(prev);
      if (next.has(folderName)) {
        next.delete(folderName);
      } else {
        next.add(folderName);
      }
      return next;
    });
  };

  const getFeedsByFolder = (folderName: string | null) => {
    return feeds.filter((f) => f.folder === folderName || (folderName === null && !f.folder));
  };

  const getUnreadCountForFeed = (feedId: number): number => {
    return unreadCounts.feeds[feedId] || 0;
  };

  const getUnreadCountForFolder = (folderName: string): number => {
    return unreadCounts.folders[folderName] || 0;
  };

  const renderUnreadBadge = (count: number) => {
    if (count === 0) return null;
    return (
      <span className="ml-1.5 bg-red-500 text-white text-xs font-bold px-1.5 py-0.5 rounded-full min-w-[1.25rem] text-center">
        {count > 99 ? "99+" : count}
      </span>
    );
  };

  const handleEditFeed = (feed: RSSFeed) => {
    setEditingFeed(feed);
    setShowEditFeedDialog(true);
  };

  const handleFeedUpdated = (updatedFeed: RSSFeed) => {
    setFeeds((prev) =>
      prev.map((f) => (f.id === updatedFeed.id ? updatedFeed : f))
    );
    setShowEditFeedDialog(false);
    setEditingFeed(null);
  };

  const handleDeleteFeed = (feed: RSSFeed) => {
    setDeletingType("feed");
    setDeletingItem(feed);
    setShowDeleteConfirm(true);
  };

  const handleEditFolder = (folder: RSSFolder) => {
    setEditingFolder(folder);
    setShowEditFolderDialog(true);
  };

  const handleFolderUpdated = (updatedFolder: RSSFolder) => {
    setFolders((prev) =>
      prev.map((f) => (f.id === updatedFolder.id ? updatedFolder : f))
    );
    // Update feeds that reference this folder
    setFeeds((prev) =>
      prev.map((f) => (f.folder === editingFolder?.name ? { ...f, folder: updatedFolder.name } : f))
    );
    // Update selected folder if it's the one being edited
    if (selectedFolder === editingFolder?.name) {
      setSelectedFolder(updatedFolder.name);
    }
    setShowEditFolderDialog(false);
    setEditingFolder(null);
  };

  const handleFolderCreated = (newFolder: RSSFolder) => {
    setFolders((prev) => [...prev, newFolder]);
    setShowCreateFolderDialog(false);
  };

  const handleDeleteFolder = (folder: RSSFolder) => {
    setDeletingType("folder");
    setDeletingItem(folder);
    setShowDeleteConfirm(true);
  };

  const handleConfirmDelete = async () => {
    if (!deletingType || !deletingItem) return;

    try {
      if (deletingType === "feed") {
        await deleteFeed((deletingItem as RSSFeed).id);
        setFeeds((prev) => prev.filter((f) => f.id !== (deletingItem as RSSFeed).id));
        // Clear selected feed if it was deleted
        if (selectedFeedId === (deletingItem as RSSFeed).id) {
          setSelectedFeedId(null);
        }
      } else if (deletingType === "folder") {
        await deleteFolder((deletingItem as RSSFolder).id);
        setFolders((prev) => prev.filter((f) => f.id !== (deletingItem as RSSFolder).id));
        // Update feeds to remove folder reference
        setFeeds((prev) =>
          prev.map((f) => (f.folder === (deletingItem as RSSFolder).name ? { ...f, folder: undefined } : f))
        );
        // Clear selected folder if it was deleted
        if (selectedFolder === (deletingItem as RSSFolder).name) {
          setSelectedFolder(null);
        }
      }
      setShowDeleteConfirm(false);
      setDeletingType(null);
      setDeletingItem(null);
    } catch (error) {
      console.error("Failed to delete:", error);
      setErrorMessage("Failed to delete. Please try again.");
      setTimeout(() => setErrorMessage(""), 5000);
    }
  };

  const handleMarkAsUnread = async () => {
    if (!selectedArticle) return;

    try {
      await markAsRead(selectedArticle.id, false);
      setArticles((prev) =>
        prev.map((a) => (a.id === selectedArticle.id ? { ...a, read: false } : a))
      );
      setSelectedArticle((prev) => (prev ? { ...prev, read: false } : null));
      // Refresh unread counts (fire and forget, non-blocking)
      refreshUnreadCounts().catch(() => {
        // Silently fail - counts will update on next refresh
      });
    } catch (error) {
      console.error("Failed to mark as unread:", error);
      setErrorMessage("Failed to mark article as unread. Please try again.");
      setTimeout(() => setErrorMessage(""), 3000);
    }
  };

  const handleExportOPML = async () => {
    try {
      const blob = await exportOPML();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `zettelgarden-feeds-${new Date().toISOString().split("T")[0]}.opml`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error("Failed to export OPML:", error);
      setErrorMessage("Failed to export feeds. Please try again.");
      setTimeout(() => setErrorMessage(""), 5000);
    }
  };

  const handleImportOPML = async (file: File) => {
    setImporting(true);
    setImportResult(null);
    try {
      const result = await importOPML(file);
      setImportResult(result);
      // Reload data to show new feeds
      await loadData();
      await loadArticles();
    } catch (error) {
      console.error("Failed to import OPML:", error);
      setErrorMessage("Failed to import feeds. Please check the file format and try again.");
      setTimeout(() => setErrorMessage(""), 5000);
    } finally {
      setImporting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-gray-500">Loading RSS feeds...</div>
      </div>
    );
  }

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Left Panel: Feeds & Folders */}
      <div className="w-64 border-r border-gray-200 p-4 overflow-y-auto bg-gray-50 flex-shrink-0">
        <div className="mb-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold">RSS Feeds</h2>
            <div className="relative" ref={settingsMenuRef}>
              <button
                onClick={() => setShowSettingsMenu(!showSettingsMenu)}
                className="p-1.5 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-md transition-colors"
                aria-label="Settings"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
              </button>

              {showSettingsMenu && (
                <div className="absolute right-0 mt-1 w-48 bg-white rounded-md shadow-lg border border-gray-200 py-1 z-50">
                  <button
                    onClick={() => {
                      setShowSettingsMenu(false);
                      handleRefresh();
                    }}
                    disabled={refreshing}
                    className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                  >
                    <svg className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                    {refreshing ? "Refreshing..." : "Refresh All"}
                  </button>
                  <button
                    onClick={() => {
                      setShowSettingsMenu(false);
                      handleExportOPML();
                    }}
                    className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                    Export OPML
                  </button>
                  <button
                    onClick={() => {
                      setShowSettingsMenu(false);
                      setShowImportDialog(true);
                    }}
                    className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                    </svg>
                    Import OPML
                  </button>
                </div>
              )}
            </div>
          </div>
          <div className="space-y-2">
            {refreshMessage && (
              <div className={`text-sm text-center px-2 py-1 rounded ${
                refreshMessage.includes("Failed") ? "text-red-600" : "text-green-600"
              }`}>
                {refreshMessage}
              </div>
            )}
            {errorMessage && (
              <div className="text-sm text-center px-2 py-1 rounded bg-red-50 text-red-600">
                {errorMessage}
              </div>
            )}
            <button
              onClick={() => setShowAddFeedDialog(true)}
              className="w-full bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700 transition-colors flex items-center justify-center gap-2"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
              </svg>
              Add Feed
            </button>
          </div>
        </div>

        {/* All Feeds */}
        <div className="mb-3">
          <button
            onClick={() => {
              setSelectedFolder(null);
              setSelectedFeedId(null);
            }}
            className={`w-full text-left px-3 py-2 rounded-md transition-colors font-medium ${
              selectedFolder === null && selectedFeedId === null ? "bg-blue-100 text-blue-900" : "hover:bg-gray-100"
            }`}
          >
            All Feeds ({feeds.length})
          </button>
        </div>

        {/* Folders with Feeds */}
        {folders.length > 0 && (
          <div className="mb-3 space-y-2">
            {folders.map((folder) => {
              const folderFeeds = getFeedsByFolder(folder.name);
              const isExpanded = expandedFolders.has(folder.name);
              const isSelected = selectedFolder === folder.name && selectedFeedId === null;
              const unreadCount = getUnreadCountForFolder(folder.name);

              return (
                <div key={folder.id} className="bg-gray-100/50 rounded-lg p-2">
                  {/* Folder header */}
                  <div
                    className={`group flex items-center rounded-md transition-colors text-sm ${
                      isSelected ? "bg-amber-100" : "hover:bg-amber-50"
                    }`}
                  >
                    <button
                      onClick={() => {
                        setSelectedFolder(folder.name);
                        setSelectedFeedId(null);
                      }}
                      className="flex-1 text-left px-3 py-2 flex items-center gap-1"
                    >
                      <svg
                        className={`w-3 h-3 text-gray-400 transition-transform ${isExpanded ? "rotate-90" : ""}`}
                        fill="currentColor"
                        viewBox="0 0 20 20"
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleFolderExpanded(folder.name);
                        }}
                      >
                        <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
                      </svg>
                      <span className="font-medium">{folder.name}</span>
                      <span className="text-gray-400 text-xs">({folderFeeds.length})</span>
                      {renderUnreadBadge(unreadCount)}
                    </button>
                    <div className="flex items-center pr-2 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEditFolder(folder);
                        }}
                        className="p-1 text-gray-400 hover:text-blue-600 transition-colors"
                        aria-label={`Rename folder ${folder.name}`}
                        title="Rename folder"
                      >
                        <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteFolder(folder);
                        }}
                        className="p-1 text-gray-400 hover:text-red-600 transition-colors"
                        aria-label={`Delete folder ${folder.name}`}
                        title="Delete folder"
                      >
                        <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </div>

                  {/* Feeds in folder (expanded) */}
                  {isExpanded && (
                    <div className="ml-4 space-y-1 mt-1">
                      {folderFeeds.map((feed) => {
                        const unreadCount = getUnreadCountForFeed(feed.id);
                        const showMenu = showFeedMenuId === feed.id;
                        return (
                          <div
                            key={feed.id}
                            className={`group flex items-center gap-1 rounded-md transition-colors text-sm ${
                              selectedFeedId === feed.id ? "bg-blue-50" : "hover:bg-gray-50"
                            }`}
                            ref={(el) => {
                              if (el) feedMenuRefs.current.set(feed.id, el);
                            }}
                          >
                            {/* Settings button on left */}
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                setShowFeedMenuId(showMenu ? null : feed.id);
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
                                setSelectedFeedId(feed.id);
                                setSelectedFolder(null);
                                setShowFeedMenuId(null);
                              }}
                              className="flex-1 text-left px-3 py-1.5 truncate"
                              title={feed.url}
                            >
                              <span className="truncate">{feed.name}</span>
                            </button>

                            {/* Unread badge on right */}
                            {renderUnreadBadge(unreadCount)}

                            {/* Dropdown menu */}
                            {showMenu && (
                              <div className="absolute left-8 top-full mt-1 w-32 bg-white rounded-md shadow-lg border border-gray-200 py-1 z-50">
                                <button
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    setShowFeedMenuId(null);
                                    handleEditFeed(feed);
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
                                    setShowFeedMenuId(null);
                                    handleDeleteFeed(feed);
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
                      })}
                      {folderFeeds.length === 0 && (
                        <div className="px-3 py-1.5 text-xs text-gray-400 italic">No feeds</div>
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
          const uncategorizedFeeds = getFeedsByFolder(null);
          if (uncategorizedFeeds.length === 0) return null;

          const isExpanded = expandedFolders.has("__uncategorized__");
          return (
            <div className="mb-3 bg-gray-100/50 rounded-lg p-2">
              <div
                className={`group flex items-center rounded-md transition-colors text-sm ${
                  selectedFolder === null && selectedFeedId === null ? "bg-amber-100" : "hover:bg-amber-50"
                }`}
              >
                <button
                  onClick={() => toggleFolderExpanded("__uncategorized__")}
                  className="flex-1 text-left px-3 py-2 flex items-center gap-1"
                >
                  <svg
                    className={`w-3 h-3 text-gray-400 transition-transform ${isExpanded ? "rotate-90" : ""}`}
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
                  </svg>
                  <span className="font-medium text-gray-600">Uncategorized</span>
                  <span className="text-gray-400 text-xs">({uncategorizedFeeds.length})</span>
                </button>
              </div>
              {isExpanded && (
                <div className="ml-4 space-y-1 mt-1">
                  {uncategorizedFeeds.map((feed) => {
                    const unreadCount = getUnreadCountForFeed(feed.id);
                    const showMenu = showFeedMenuId === feed.id;
                    return (
                      <div
                        key={feed.id}
                        className={`group flex items-center gap-1 rounded-md transition-colors text-sm ${
                          selectedFeedId === feed.id ? "bg-blue-50" : "hover:bg-gray-50"
                        }`}
                        ref={(el) => {
                          if (el) feedMenuRefs.current.set(feed.id, el);
                        }}
                      >
                        {/* Settings button on left */}
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowFeedMenuId(showMenu ? null : feed.id);
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
                            setSelectedFeedId(feed.id);
                            setSelectedFolder(null);
                            setShowFeedMenuId(null);
                          }}
                          className="flex-1 text-left px-3 py-1.5 truncate"
                          title={feed.url}
                        >
                          <span className="truncate">{feed.name}</span>
                        </button>

                        {/* Unread badge on right */}
                        {renderUnreadBadge(unreadCount)}

                        {/* Dropdown menu */}
                        {showMenu && (
                          <div className="absolute left-8 top-full mt-1 w-32 bg-white rounded-md shadow-lg border border-gray-200 py-1 z-50">
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                setShowFeedMenuId(null);
                                handleEditFeed(feed);
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
                                setShowFeedMenuId(null);
                                handleDeleteFeed(feed);
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
                  })}
                </div>
              )}
            </div>
          );
        })()}

        {/* Create Folder Link */}
        <div className="mb-4">
          <button
            onClick={() => setShowCreateFolderDialog(true)}
            className="text-xs text-blue-600 hover:text-blue-800 transition-colors flex items-center gap-1"
          >
            <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
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
              onChange={(e) => setShowUnreadOnly(e.target.checked)}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm">Unread only</span>
          </label>
        </div>
      </div>

      {/* Middle Panel: Articles */}
      <div className="w-80 border-r border-gray-200 bg-white flex-shrink-0 flex flex-col">
        <div className="p-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold mb-4">Articles</h2>
        </div>
        {articles.length === 0 && !loading ? (
          <div className="flex-1 flex items-center justify-center">
            <p className="text-gray-500">No articles found</p>
          </div>
        ) : (
          <>
            <div className="flex-1 overflow-y-auto p-4 space-y-2">
              {articles.map((article) => (
                <div
                  key={article.id}
                  onClick={() => handleArticleClick(article)}
                  className={`p-3 rounded-md cursor-pointer transition-colors ${
                    selectedArticle?.id === article.id
                      ? "bg-blue-100 border-l-4 border-blue-600"
                      : article.read
                      ? "bg-gray-50 hover:bg-gray-100"
                      : "bg-white hover:bg-gray-100 border-l-4 border-blue-500 shadow-sm"
                  }`}
                >
                  <div className="flex items-start gap-2">
                    <h3 className="font-medium text-sm line-clamp-2 mb-1 flex-1">
                      {article.title}
                    </h3>
                    {article.card_id && (
                      <svg className="w-4 h-4 text-green-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                        <title>Converted to card</title>
                        <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                      </svg>
                    )}
                  </div>
                  <p className="text-xs text-gray-500">
                    {getFeedName(article.feed_id)} • {new Date(article.fetched_at).toLocaleDateString()}
                  </p>
                </div>
              ))}
            </div>

            {/* Pagination */}
            {totalArticles > articlesPerPage && (
              <div className="p-3 border-t border-gray-200 bg-gray-50">
                <div className="flex items-center justify-between text-sm">
                  <button
                    onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                    disabled={currentPage === 1}
                    className="px-3 py-1 rounded border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Previous
                  </button>
                  <span className="text-gray-600">
                    Page {currentPage} of {Math.ceil(totalArticles / articlesPerPage)}
                  </span>
                  <button
                    onClick={() => setCurrentPage(p => Math.min(Math.ceil(totalArticles / articlesPerPage), p + 1))}
                    disabled={currentPage >= Math.ceil(totalArticles / articlesPerPage)}
                    className="px-3 py-1 rounded border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Right Panel: Article Reader */}
      <div className="flex-1 p-6 overflow-y-auto bg-white min-w-0">
        {selectedArticle ? (
          <div className="max-w-3xl mx-auto">
            <div className="mb-6">
              <h1 className="text-2xl font-bold mb-3 text-gray-900">
                {selectedArticle.title}
              </h1>
              <div className="flex flex-wrap items-center gap-4 text-sm text-gray-600">
                {selectedArticle.author && (
                  <span className="flex items-center gap-1">
                    <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                    </svg>
                    {selectedArticle.author}
                  </span>
                )}
                <span className="flex items-center gap-1">
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M6 2a1 1 0 00-1 1v1H4a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-1V3a1 1 0 10-2 0v1H7V3a1 1 0 00-1-1zm0 5a1 1 0 000 2h8a1 1 0 100-2H6z" clipRule="evenodd" />
                  </svg>
                  {selectedArticle.published_at
                    ? new Date(selectedArticle.published_at).toLocaleDateString()
                    : new Date(selectedArticle.fetched_at).toLocaleDateString()}
                </span>
                <a
                  href={selectedArticle.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-blue-600 hover:text-blue-800 hover:underline"
                >
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M11 3a1 1 0 100 2h2.586l-6.293 6.293a1 1 0 101.414 1.414L15 6.414V9a1 1 0 102 0V4a1 1 0 00-1-1h-5z" />
                    <path d="M5 5a2 2 0 00-2 2v8a2 2 0 002 2h8a2 2 0 002-2v-3a1 1 0 10-2 0v3H5V7h3a1 1 0 000-2H5z" />
                  </svg>
                  View original
                </a>
                {selectedArticle.card_id && (
                  <button
                    onClick={() => navigate(`/app/card/${selectedArticle.card_id}`)}
                    className="flex items-center gap-1 text-green-600 hover:text-green-800 hover:underline"
                  >
                    <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                      <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                    </svg>
                    View card
                  </button>
                )}
              </div>
            </div>

            {selectedArticle.content && (
              <div className="prose prose-sm max-w-none mb-8">
                <Markdown
                  remarkPlugins={[remarkGfm]}
                  components={{
                    a: ({ href, children, ...props }) => (
                      <a href={href} target="_blank" rel="noopener noreferrer" {...props}>
                        {children}
                      </a>
                    ),
                  }}
                >
                  {safeHtmlToMarkdown(selectedArticle.content)}
                </Markdown>
              </div>
            )}

            <div className="flex flex-col sm:flex-row gap-3 pt-6 border-t border-gray-200">
              {selectedArticle.read && (
                <button
                  onClick={handleMarkAsUnread}
                  className="bg-gray-600 text-white px-6 py-2 rounded-md hover:bg-gray-700 transition-colors flex items-center justify-center gap-2"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                  Mark as Unread
                </button>
              )}
              {selectedArticle.card_id ? (
                <button
                  onClick={() => navigate(`/app/card/${selectedArticle.card_id}`)}
                  className="bg-green-600 text-white px-6 py-2 rounded-md hover:bg-green-700 transition-colors flex items-center justify-center gap-2"
                >
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                  </svg>
                  View card
                </button>
              ) : (
                <button
                  onClick={handleConvertClick}
                  className="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center justify-center gap-2"
                >
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                  </svg>
                  Convert to Card
                </button>
              )}
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-gray-400">
            <svg className="w-16 h-16 mb-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M2 5a2 2 0 012-2h8a2 2 0 012 2v10a2 2 0 002 2H4a2 2 0 01-2-2V5zm3 1h6v4H5V6zm6 6H5v2h6v-2z" clipRule="evenodd" />
              <path d="M15 7h1a2 2 0 012 2v5.5a1.5 1.5 0 01-1.5 1.5h-1v-1h1a.5.5 0 00.5-.5V9a1 1 0 00-1-1h-1V7z" />
            </svg>
            <p className="text-lg">Select an article to read</p>
          </div>
        )}
      </div>

      {/* Dialogs */}
      <RssAddFeedDialog
        isOpen={showAddFeedDialog}
        onClose={() => setShowAddFeedDialog(false)}
        folders={folders}
        onFeedAdded={handleFeedAdded}
      />
      <RssEditFeedDialog
        isOpen={showEditFeedDialog}
        onClose={() => setShowEditFeedDialog(false)}
        feed={editingFeed}
        folders={folders}
        onFeedUpdated={handleFeedUpdated}
      />
      <RssEditFolderDialog
        isOpen={showEditFolderDialog}
        onClose={() => setShowEditFolderDialog(false)}
        folder={editingFolder}
        onFolderUpdated={handleFolderUpdated}
      />
      <RssCreateFolderDialog
        isOpen={showCreateFolderDialog}
        onClose={() => setShowCreateFolderDialog(false)}
        onFolderCreated={handleFolderCreated}
      />
      <RssConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleConfirmDelete}
        title={deletingType === "feed" ? "Delete Feed" : "Delete Folder"}
        message={
          deletingType === "feed"
            ? `Are you sure you want to delete "${(deletingItem as RSSFeed)?.name}"? This will also delete all articles from this feed.`
            : `Are you sure you want to delete folder "${(deletingItem as RSSFolder)?.name}"? Feeds in this folder will become uncategorized.`
        }
        confirmText="Delete"
        dangerous={true}
      />
      <RssConvertDialog
        isOpen={showConvertDialog}
        onClose={() => setShowConvertDialog(false)}
        article={selectedArticle}
        onConverted={handleConverted}
      />
      <RssImportDialog
        isOpen={showImportDialog}
        onClose={() => {
          setShowImportDialog(false);
          setImportResult(null);
        }}
        onImport={handleImportOPML}
        importing={importing}
        importResult={importResult}
      />
    </div>
  );
}
