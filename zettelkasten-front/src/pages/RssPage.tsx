import React, { useState, useCallback, useEffect, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { setDocumentTitle } from "../utils/title";
import {
  markAsRead,
  markFeedAsRead,
  markFolderAsRead,
  deleteFeed,
  deleteFolder,
  exportOPML,
  importOPML,
  OPMLImportResult,
  RSSFeed,
  RSSFolder,
  RSSArticle,
  RSSArticleWithScore,
} from "../api/rss";
import { RssAddFeedDialog } from "../components/rss/RssAddFeedDialog";
import { RssEditFeedDialog } from "../components/rss/RssEditFeedDialog";
import { RssEditFolderDialog } from "../components/rss/RssEditFolderDialog";
import { RssCreateFolderDialog } from "../components/rss/RssCreateFolderDialog";
import { RssConfirmDialog } from "../components/rss/RssConfirmDialog";
import { RssConvertDialog } from "../components/rss/RssConvertDialog";
import { RssImportDialog } from "../components/rss/RssImportDialog";
import { useRSS } from "../contexts/RSSContext";
import { useUIState } from "../contexts/UIStateContext";
import { useRssData } from "../hooks/useRssData";
import { useRssArticles } from "../hooks/useRssArticles";
import { useSmartRssArticles } from "../hooks/useSmartRssArticles";
import { RssDesktopLayout } from "../components/rss/RssDesktopLayout";
import { RssMobileLayout } from "../components/rss/RssMobileLayout";
import { DialogState, DialogStates, initialDialogState } from "../types/rssDialogs";
import { RSS_CONFIG } from "../constants/rss";

// Stable empty object to avoid infinite re-renders
const EMPTY_FILTERS = Object.freeze({});

export function RssPage() {
  const navigate = useNavigate();
  const { setUnreadCount } = useRSS();
  const { toggleMobileSidebar } = useUIState();

  // Custom hooks for data fetching
  const {
    feeds,
    setFeeds,
    folders,
    setFolders,
    unreadCounts,
    loading: loadingData,
    refreshing,
    refreshAllFeeds,
    refreshUnreadCounts,
  } = useRssData();

  // Article management
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [selectedFeedId, setSelectedFeedId] = useState<number | null>(null);
  const [selectedArticle, setSelectedArticle] = useState<RSSArticle | null>(null);
  const [showUnreadOnly, setShowUnreadOnly] = useState(false);
  const [isSmartFeedActive, setIsSmartFeedActive] = useState(false);
  const [markingAsRead, setMarkingAsRead] = useState<Set<number>>(new Set());
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());
  const [showSettingsMenu, setShowSettingsMenu] = useState(false);
  const [showFeedMenuId, setShowFeedMenuId] = useState<number | null>(null);

  // Articles hook - use smart feed hook when smart feed is active, regular hook otherwise
  // Smart feed uses no filters, regular feed uses folder/feed/unread filters
  const articleFilters = useMemo(() => ({
    folder: selectedFolder ?? undefined,
    feed_id: selectedFeedId ?? undefined,
    unread: showUnreadOnly || undefined,
  }), [selectedFolder, selectedFeedId, showUnreadOnly]);

  // Only one hook should fetch at a time - skip the inactive one
  const regularArticles = useRssArticles({
    filters: isSmartFeedActive ? EMPTY_FILTERS : articleFilters,
    skip: isSmartFeedActive
  });
  // Smart feed respects the unread filter when enabled
  const smartFeedFilters = useMemo(() => ({
    unread: showUnreadOnly || undefined,
  }), [showUnreadOnly]);
  const smartArticles = useSmartRssArticles({
    filters: isSmartFeedActive ? smartFeedFilters : EMPTY_FILTERS,
    skip: !isSmartFeedActive
  });

  // Use either regular or smart articles based on mode
  const articles = isSmartFeedActive ? smartArticles.articles : regularArticles.articles;
  const totalArticles = isSmartFeedActive ? smartArticles.totalArticles : regularArticles.totalArticles;
  const loadingArticles = isSmartFeedActive ? smartArticles.loading : regularArticles.loading;
  const loadingMoreArticles = isSmartFeedActive ? false : regularArticles.loadingMore;
  const articlesError = isSmartFeedActive ? smartArticles.error : regularArticles.error;
  const hasMoreArticles = isSmartFeedActive ? smartArticles.hasMore : regularArticles.hasMore;
  const updateArticle = isSmartFeedActive ? smartArticles.updateArticle : regularArticles.updateArticle;
  const updateArticles = isSmartFeedActive ? smartArticles.updateArticles : regularArticles.updateArticles;
  const resetToFirstPage = isSmartFeedActive ? smartArticles.resetToFirstPage : regularArticles.resetToFirstPage;

  const handleLoadMore = useCallback(() => {
    if (isSmartFeedActive) {
      smartArticles.loadMore();
    } else {
      regularArticles.loadMore();
    }
  }, [isSmartFeedActive, smartArticles, regularArticles]);

  // Dialog state
  const [dialogState, setDialogState] = useState<DialogState>(initialDialogState);
  const [refreshMessage, setRefreshMessage] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [importResult, setImportResult] = useState<OPMLImportResult | null>(null);
  const [importing, setImporting] = useState(false);

  // Mobile navigation state
  const [mobileView, setMobileView] = useState<'list' | 'reader' | 'feeds'>('list');

  // Mobile breakpoint detection
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window !== 'undefined') {
      return window.innerWidth < RSS_CONFIG.MOBILE_BREAKPOINT;
    }
    return false;
  });

  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth < RSS_CONFIG.MOBILE_BREAKPOINT);
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  // Sort folders alphabetically
  const sortedFolders = useMemo(() => {
    return [...folders].sort((a, b) => a.name.localeCompare(b.name));
  }, [folders]);

  // Calculate total unread count (all feeds)
  const totalUnreadCount = useMemo(() => {
    return Object.values(unreadCounts.feeds).reduce((sum, count) => sum + count, 0);
  }, [unreadCounts]);

  // Calculate unread count for current view
  const currentUnreadCount = useMemo(() => {
    if (selectedFeedId) {
      return unreadCounts.feeds[selectedFeedId] || 0;
    } else if (selectedFolder) {
      return unreadCounts.folders[selectedFolder] || 0;
    }
    return totalUnreadCount;
  }, [unreadCounts, selectedFeedId, selectedFolder, totalUnreadCount]);

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

  // Reset to page 1 when filters change
  useEffect(() => {
    resetToFirstPage();
  }, [selectedFolder, selectedFeedId, showUnreadOnly, isSmartFeedActive, resetToFirstPage]);

  const handleRefresh = useCallback(async () => {
    setRefreshMessage("");
    try {
      const result = await refreshAllFeeds();
      setRefreshMessage(`Refreshed ${result.fetched} feeds`);
      setTimeout(() => setRefreshMessage(""), 3000);
    } catch (error) {
      console.error("Failed to refresh feeds:", error);
      setRefreshMessage("Failed to refresh feeds");
      setTimeout(() => setRefreshMessage(""), 3000);
    }
  }, [refreshAllFeeds]);

  const handleSelectSmartFeed = useCallback(() => {
    setIsSmartFeedActive(true);
    setSelectedFolder(null);
    setSelectedFeedId(null);
  }, []);

  const handleArticleClick = useCallback(async (article: RSSArticle) => {
    // Prevent duplicate requests for the same article
    if (markingAsRead.has(article.id)) {
      return;
    }

    setSelectedArticle(article);

    // Mobile: show reader view
    if (isMobile) {
      setMobileView('reader');
    }

    if (!article.read) {
      setMarkingAsRead((prev) => new Set(prev).add(article.id));
      try {
        await markAsRead(article.id, true);
        updateArticle(article.id, { read: true });
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
  }, [markingAsRead, refreshUnreadCounts, isMobile, updateArticle]);

  const handleMarkAsUnread = useCallback(async () => {
    if (!selectedArticle) return;

    try {
      await markAsRead(selectedArticle.id, false);
      updateArticle(selectedArticle.id, { read: false });
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
  }, [selectedArticle, updateArticle, refreshUnreadCounts]);

  const handleConvertClick = useCallback(() => {
    if (!selectedArticle) {
      return;
    }
    setDialogState(DialogStates.convert(selectedArticle));
  }, [selectedArticle]);

  const handleConverted = useCallback((cardId: number) => {
    setDialogState(initialDialogState);
    // Navigate to the new card
    navigate(`/app/card/${cardId}`);
  }, [navigate]);

  const handleFeedAdded = useCallback((feed: RSSFeed) => {
    setFeeds((prev) => [...prev, feed]);
    setDialogState(initialDialogState);
  }, [setFeeds]);

  const handleFeedUpdated = useCallback((updatedFeed: RSSFeed) => {
    setFeeds((prev) =>
      prev.map((f) => (f.id === updatedFeed.id ? updatedFeed : f))
    );
    setDialogState(initialDialogState);
  }, [setFeeds]);

  const handleFolderUpdated = useCallback((updatedFolder: RSSFolder) => {
    setFolders((prev) =>
      prev.map((f) => (f.id === updatedFolder.id ? updatedFolder : f))
    );
    // Update feeds that reference this folder
    const currentState = dialogState;
    if (currentState.type === 'editFolder') {
      setFeeds((prev) =>
        prev.map((f) => (f.folder === currentState.folder.name ? { ...f, folder: updatedFolder.name } : f))
      );
      // Update selected folder if it's the one being edited
      if (selectedFolder === currentState.folder.name) {
        setSelectedFolder(updatedFolder.name);
      }
    }
    setDialogState(initialDialogState);
  }, [setFolders, setFeeds, dialogState, selectedFolder]);

  const handleFolderCreated = useCallback((newFolder: RSSFolder) => {
    setFolders((prev) => [...prev, newFolder]);
    setDialogState(initialDialogState);
  }, [setFolders]);

  const handleConfirmDelete = useCallback(async () => {
    if (dialogState.type !== 'deleteConfirm') return;

    try {
      if (dialogState.itemType === "feed") {
        await deleteFeed((dialogState.item as RSSFeed).id);
        setFeeds((prev) => prev.filter((f) => f.id !== (dialogState.item as RSSFeed).id));
        // Clear selected feed if it was deleted
        if (selectedFeedId === (dialogState.item as RSSFeed).id) {
          setSelectedFeedId(null);
        }
      } else if (dialogState.itemType === "folder") {
        const folder = dialogState.item as RSSFolder;
        // Get all feeds in this folder
        const folderFeeds = feeds.filter((f) => f.folder === folder.name);
        // Delete all feeds in the folder first
        await Promise.all(folderFeeds.map((feed) => deleteFeed(feed.id)));
        // Then delete the folder
        await deleteFolder(folder.id);
        setFolders((prev) => prev.filter((f) => f.id !== folder.id));
        // Remove feeds that were in the deleted folder
        setFeeds((prev) => prev.filter((f) => f.folder !== folder.name));
        // Clear selected folder if it was deleted
        if (selectedFolder === folder.name) {
          setSelectedFolder(null);
        }
      }
      setDialogState(initialDialogState);
    } catch (error) {
      console.error("Failed to delete:", error);
      setErrorMessage("Failed to delete. Please try again.");
      setTimeout(() => setErrorMessage(""), 5000);
    }
  }, [dialogState, setFeeds, setFolders, selectedFeedId, selectedFolder]);

  const handleMarkFeedAsRead = useCallback(async (feed: RSSFeed) => {
    try {
      await markFeedAsRead(feed.id);
      // Update articles to mark as read
      updateArticles((prev) =>
        prev.map((a) => (a.feed_id === feed.id ? { ...a, read: true } : a))
      );
      // Refresh unread counts
      await refreshUnreadCounts();
    } catch (error) {
      console.error("Failed to mark feed as read:", error);
      setErrorMessage("Failed to mark feed as read. Please try again.");
      setTimeout(() => setErrorMessage(""), 3000);
    }
  }, [updateArticles, refreshUnreadCounts]);

  const handleMarkFolderAsRead = useCallback(async (folder: RSSFolder) => {
    try {
      await markFolderAsRead(folder.id);
      // Get feeds in this folder
      const folderFeeds = feeds.filter((f) => f.folder === folder.name);
      const feedIds = new Set(folderFeeds.map((f) => f.id));
      // Update articles to mark as read
      updateArticles((prev) =>
        prev.map((a) => (feedIds.has(a.feed_id) ? { ...a, read: true } : a))
      );
      // Refresh unread counts
      await refreshUnreadCounts();
    } catch (error) {
      console.error("Failed to mark folder as read:", error);
      setErrorMessage("Failed to mark folder as read. Please try again.");
      setTimeout(() => setErrorMessage(""), 3000);
    }
  }, [feeds, updateArticles, refreshUnreadCounts]);

  const handleExportOPML = useCallback(async () => {
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
  }, []);

  const handleImportOPML = useCallback(async (file: File) => {
    setImporting(true);
    setImportResult(null);
    try {
      const result = await importOPML(file);
      setImportResult(result);
      // Reload data to show new feeds
      await refreshAllFeeds();
    } catch (error) {
      console.error("Failed to import OPML:", error);
      setErrorMessage("Failed to import feeds. Please check the file format and try again.");
      setTimeout(() => setErrorMessage(""), 5000);
    } finally {
      setImporting(false);
    }
  }, [refreshAllFeeds]);

  const toggleFolderExpanded = useCallback((folderName: string) => {
    setExpandedFolders((prev) => {
      const next = new Set(prev);
      if (next.has(folderName)) {
        next.delete(folderName);
      } else {
        next.add(folderName);
      }
      return next;
    });
  }, []);

  // Mobile handlers
  const handleFeedSelectMobile = useCallback((feedId: number) => {
    setIsSmartFeedActive(false);
    setSelectedFeedId(feedId);
    setSelectedFolder(null);
    setMobileView('list');
  }, []);

  const handleFolderSelectMobile = useCallback((folderName: string) => {
    setIsSmartFeedActive(false);
    setSelectedFolder(folderName);
    setSelectedFeedId(null);
    setMobileView('list');
  }, []);

  const handleAllFeedsSelectMobile = useCallback(() => {
    setIsSmartFeedActive(false);
    setSelectedFolder(null);
    setSelectedFeedId(null);
    setMobileView('feeds');
  }, []);

  const handleMobileBack = useCallback(() => {
    setMobileView('list');
  }, []);

  // Loading state
  if (loadingData) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-gray-500">Loading RSS feeds...</div>
      </div>
    );
  }

  // Prepare dialog state for components
  const showAddFeedDialog = dialogState.type === 'addFeed';
  const showEditFeedDialog = dialogState.type === 'editFeed';
  const showEditFolderDialog = dialogState.type === 'editFolder';
  const showCreateFolderDialog = dialogState.type === 'createFolder';
  const showDeleteConfirm = dialogState.type === 'deleteConfirm';
  const showConvertDialog = dialogState.type === 'convert';
  const showImportDialogState = dialogState.type === 'import';

  const editingFeed = dialogState.type === 'editFeed' ? dialogState.feed : null;
  const editingFolder = dialogState.type === 'editFolder' ? dialogState.folder : null;
  const convertingArticle = dialogState.type === 'convert' ? dialogState.article : null;
  const deletingType = dialogState.type === 'deleteConfirm' ? dialogState.itemType : null;
  const deletingItem = dialogState.type === 'deleteConfirm' ? dialogState.item : null;

  return (
    <div className="flex flex-col md:flex-row h-screen overflow-hidden">
      {/* Desktop Layout */}
      {!isMobile && (
        <RssDesktopLayout
          feeds={feeds}
          folders={sortedFolders}
          unreadCounts={unreadCounts}
          articles={articles}
          selectedFolder={selectedFolder}
          selectedFeedId={selectedFeedId}
          selectedArticle={selectedArticle}
          showUnreadOnly={showUnreadOnly}
          isSmartFeedActive={isSmartFeedActive}
          expandedFolders={expandedFolders}
          currentUnreadCount={currentUnreadCount}
          refreshMessage={refreshMessage}
          errorMessage={errorMessage || articlesError || ""}
          refreshing={refreshing}
          showSettingsMenu={showSettingsMenu}
          showFeedMenuId={showFeedMenuId}
          loadingArticles={loadingArticles}
          loadingMoreArticles={loadingMoreArticles}
          totalArticles={totalArticles}
          hasMore={hasMoreArticles}
          onSelectAllFeeds={() => {
            setIsSmartFeedActive(false);
            setSelectedFolder(null);
            setSelectedFeedId(null);
          }}
          onSelectFolder={(folderName) => {
            setIsSmartFeedActive(false);
            setSelectedFolder(folderName);
            setSelectedFeedId(null);
          }}
          onSelectFeed={(feedId) => {
            setIsSmartFeedActive(false);
            setSelectedFeedId(feedId);
            setSelectedFolder(null);
          }}
          onToggleFolder={toggleFolderExpanded}
          onToggleShowUnreadOnly={() => setShowUnreadOnly((prev) => !prev)}
          onSelectSmartFeed={handleSelectSmartFeed}
          onAddFeed={() => setDialogState(DialogStates.addFeed())}
          onCreateFolder={() => setDialogState(DialogStates.createFolder())}
          onEditFeed={(feed) => setDialogState(DialogStates.editFeed(feed))}
          onDeleteFeed={(feed) => setDialogState(DialogStates.deleteConfirm('feed', feed))}
          onMarkFeedAsRead={handleMarkFeedAsRead}
          onEditFolder={(folder) => setDialogState(DialogStates.editFolder(folder))}
          onDeleteFolder={(folder) => setDialogState(DialogStates.deleteConfirm('folder', folder))}
          onMarkFolderAsRead={handleMarkFolderAsRead}
          onRefresh={handleRefresh}
          onExportOPML={handleExportOPML}
          onImportOPML={() => setDialogState(DialogStates.import())}
          onToggleSettingsMenu={() => setShowSettingsMenu((prev) => !prev)}
          onToggleFeedMenu={setShowFeedMenuId}
          onArticleClick={handleArticleClick}
          onLoadMore={handleLoadMore}
          onConvertClick={handleConvertClick}
          onMarkAsUnread={handleMarkAsUnread}
        />
      )}

      {/* Mobile Layout */}
      {isMobile && (
        <RssMobileLayout
          feeds={feeds}
          folders={sortedFolders}
          unreadCounts={unreadCounts}
          articles={articles}
          selectedFolder={selectedFolder}
          selectedFeedId={selectedFeedId}
          selectedArticle={selectedArticle}
          showUnreadOnly={showUnreadOnly}
          isSmartFeedActive={isSmartFeedActive}
          expandedFolders={expandedFolders}
          totalUnreadCount={totalUnreadCount}
          currentUnreadCount={currentUnreadCount}
          showFeedMenuId={showFeedMenuId}
          mobileView={mobileView}
          setMobileView={setMobileView}
          loadingArticles={loadingArticles}
          loadingMoreArticles={loadingMoreArticles}
          totalArticles={totalArticles}
          hasMore={hasMoreArticles}
          onMenuClick={toggleMobileSidebar}
          onFeedSelectMobile={handleFeedSelectMobile}
          onFolderSelectMobile={handleFolderSelectMobile}
          onAllFeedsSelectMobile={handleAllFeedsSelectMobile}
          onSmartFeedSelectMobile={handleSelectSmartFeed}
          onToggleFolder={toggleFolderExpanded}
          onAddFeed={() => setDialogState(DialogStates.addFeed())}
          onCreateFolder={() => setDialogState(DialogStates.createFolder())}
          onEditFeed={(feed) => setDialogState(DialogStates.editFeed(feed))}
          onDeleteFeed={(feed) => setDialogState(DialogStates.deleteConfirm('feed', feed))}
          onMarkFeedAsRead={handleMarkFeedAsRead}
          onEditFolder={(folder) => setDialogState(DialogStates.editFolder(folder))}
          onDeleteFolder={(folder) => setDialogState(DialogStates.deleteConfirm('folder', folder))}
          onMarkFolderAsRead={handleMarkFolderAsRead}
          onShowFeedMenu={setShowFeedMenuId}
          onArticleClick={handleArticleClick}
          onLoadMore={handleLoadMore}
          onToggleShowUnreadOnly={() => setShowUnreadOnly((prev) => !prev)}
          onSelectSmartFeed={handleSelectSmartFeed}
          onConvertClick={handleConvertClick}
          onMarkAsUnread={handleMarkAsUnread}
          onMobileBack={handleMobileBack}
          onRefresh={handleRefresh}
          onExportOPML={handleExportOPML}
          onImportOPML={() => setDialogState(DialogStates.import())}
        />
      )}

      {/* Dialogs */}
      <RssAddFeedDialog
        isOpen={showAddFeedDialog}
        onClose={() => setDialogState(initialDialogState)}
        folders={sortedFolders}
        onFeedAdded={handleFeedAdded}
      />
      <RssEditFeedDialog
        isOpen={showEditFeedDialog}
        onClose={() => setDialogState(initialDialogState)}
        feed={editingFeed}
        folders={sortedFolders}
        onFeedUpdated={handleFeedUpdated}
      />
      <RssEditFolderDialog
        isOpen={showEditFolderDialog}
        onClose={() => setDialogState(initialDialogState)}
        folder={editingFolder}
        onFolderUpdated={handleFolderUpdated}
      />
      <RssCreateFolderDialog
        isOpen={showCreateFolderDialog}
        onClose={() => setDialogState(initialDialogState)}
        onFolderCreated={handleFolderCreated}
      />
      <RssConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={() => setDialogState(initialDialogState)}
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
        onClose={() => setDialogState(initialDialogState)}
        article={convertingArticle}
        onConverted={handleConverted}
      />
      <RssImportDialog
        isOpen={showImportDialogState}
        onClose={() => {
          setDialogState(initialDialogState);
          setImportResult(null);
        }}
        onImport={handleImportOPML}
        importing={importing}
        importResult={importResult}
      />
    </div>
  );
}
