import { useState, useEffect, useCallback } from "react";
import {
  listFeeds,
  listFolders,
  getUnreadCounts,
  refreshFeeds,
  RSSFeed,
  RSSFolder,
  UnreadCounts,
} from "../api/rss";

/**
 * Hook for managing RSS feeds, folders, and unread counts data
 * @returns Object containing feeds, folders, unread counts, loading state, and refresh functions
 */
export function useRssData() {
  const [feeds, setFeeds] = useState<RSSFeed[]>([]);
  const [folders, setFolders] = useState<RSSFolder[]>([]);
  const [unreadCounts, setUnreadCounts] = useState<UnreadCounts>({ folders: {}, feeds: {} });
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  /**
   * Load all RSS data (feeds, folders, unread counts)
   */
  const loadData = useCallback(async () => {
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
      throw error;
    } finally {
      setLoading(false);
    }
  }, []);

  /**
   * Refresh all RSS feeds and reload data
   */
  const refreshAllFeeds = useCallback(async () => {
    setRefreshing(true);
    try {
      const result = await refreshFeeds();
      await loadData();
      return result;
    } catch (error) {
      console.error("Failed to refresh feeds:", error);
      throw error;
    } finally {
      setRefreshing(false);
    }
  }, [loadData]);

  /**
   * Refresh unread counts only
   */
  const refreshUnreadCounts = useCallback(async () => {
    try {
      const counts = await getUnreadCounts();
      setUnreadCounts(counts);
    } catch (error) {
      console.error("Failed to refresh unread counts:", error);
      // Non-blocking error - counts will update on next refresh
    }
  }, []);

  // Initial data load
  useEffect(() => {
    loadData();
  }, [loadData]);

  return {
    feeds,
    setFeeds,
    folders,
    setFolders,
    unreadCounts,
    setUnreadCounts,
    loading,
    refreshing,
    loadData,
    refreshAllFeeds,
    refreshUnreadCounts,
  };
}
