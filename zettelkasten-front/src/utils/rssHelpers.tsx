import { RSSFeed, RSSFolder, UnreadCounts } from "../api/rss";
import { RSS_CONFIG } from "../constants/rss";
import React from "react";

/**
 * Get feeds filtered by folder name
 * @param feeds - All feeds to filter
 * @param folderName - Folder name to filter by (null for uncategorized)
 * @returns Filtered feeds
 */
export function getFeedsByFolder(feeds: RSSFeed[], folderName: string | null): RSSFeed[] {
  return feeds.filter(
    (f) => f.folder === folderName || (folderName === null && !f.folder)
  );
}

/**
 * Get unread count for a specific feed
 * @param unreadCounts - Unread counts object
 * @param feedId - Feed ID to get count for
 * @returns Unread count
 */
export function getUnreadCountForFeed(
  unreadCounts: UnreadCounts,
  feedId: number
): number {
  return unreadCounts.feeds[feedId] || 0;
}

/**
 * Get unread count for a specific folder
 * @param unreadCounts - Unread counts object
 * @param folderName - Folder name to get count for
 * @returns Unread count
 */
export function getUnreadCountForFolder(
  unreadCounts: UnreadCounts,
  folderName: string
): number {
  return unreadCounts.folders[folderName] || 0;
}

/**
 * Get total unread count across all feeds
 * @param unreadCounts - Unread counts object
 * @returns Total unread count
 */
export function getTotalUnreadCount(unreadCounts: UnreadCounts): number {
  return Object.values(unreadCounts.feeds).reduce((sum, count) => sum + count, 0);
}

/**
 * Render an unread badge component
 * @param count - Unread count to display
 * @returns JSX element or null if count is 0
 */
export function renderUnreadBadge(count: number): React.ReactNode {
  if (count === 0) return null;
  return (
    <span className="ml-1.5 bg-red-500 text-white text-xs font-bold px-1.5 py-0.5 rounded-full min-w-[1.25rem] text-center">
      {count > RSS_CONFIG.UNREAD_BADGE_MAX ? "99+" : count}
    </span>
  );
}

/**
 * Format unread count as a string (for display in non-JSX contexts)
 * @param count - Unread count to format
 * @returns Formatted string
 */
export function formatUnreadCount(count: number): string {
  if (count === 0) return "";
  return count > RSS_CONFIG.UNREAD_BADGE_MAX ? "99+" : count.toString();
}
