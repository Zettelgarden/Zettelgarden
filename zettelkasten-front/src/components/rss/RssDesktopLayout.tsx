import React from "react";
import { RSSFeed, RSSFolder, RSSArticle, RSSArticleWithScore, UnreadCounts } from "../../api/rss";
import { RssFeedsPanel } from "./RssFeedsPanel";
import { RssArticlesPanel } from "./RssArticlesPanel";
import { RssReaderPanel } from "./RssReaderPanel";
import { RssErrorBoundary } from "./RssErrorBoundary";

interface RssDesktopLayoutProps {
  feeds: RSSFeed[];
  folders: RSSFolder[];
  unreadCounts: UnreadCounts;
  articles: (RSSArticle | RSSArticleWithScore)[];
  selectedFolder: string | null;
  selectedFeedId: number | null;
  selectedArticle: RSSArticle | null;
  showUnreadOnly: boolean;
  isSmartFeedActive: boolean;
  expandedFolders: Set<string>;
  currentUnreadCount: number;
  refreshMessage: string;
  errorMessage: string;
  refreshing: boolean;
  showSettingsMenu: boolean;
  showFeedMenuId: number | null;
  loadingArticles: boolean;
  totalArticles: number;
  currentPage: number;
  onSelectAllFeeds: () => void;
  onSelectFolder: (folderName: string) => void;
  onSelectFeed: (feedId: number) => void;
  onSelectSmartFeed: () => void;
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
  onArticleClick: (article: RSSArticle) => void;
  onPageChange: (page: number) => void;
  onConvertClick: () => void;
  onMarkAsUnread: () => void;
}

/**
 * Desktop three-panel layout for RSS reader
 */
export function RssDesktopLayout({
  feeds,
  folders,
  unreadCounts,
  articles,
  selectedFolder,
  selectedFeedId,
  selectedArticle,
  showUnreadOnly,
  isSmartFeedActive,
  expandedFolders,
  currentUnreadCount,
  refreshMessage,
  errorMessage,
  refreshing,
  showSettingsMenu,
  showFeedMenuId,
  loadingArticles,
  totalArticles,
  currentPage,
  onSelectAllFeeds,
  onSelectFolder,
  onSelectFeed,
  onSelectSmartFeed,
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
  onArticleClick,
  onPageChange,
  onConvertClick,
  onMarkAsUnread,
}: RssDesktopLayoutProps) {
  return (
    <div className="hidden md:flex flex-row h-screen overflow-hidden">
      <RssErrorBoundary>
        <RssFeedsPanel
          feeds={feeds}
          folders={folders}
          unreadCounts={unreadCounts}
          selectedFolder={selectedFolder}
          selectedFeedId={selectedFeedId}
          showUnreadOnly={showUnreadOnly}
          isSmartFeedActive={isSmartFeedActive}
          expandedFolders={expandedFolders}
          refreshMessage={refreshMessage}
          errorMessage={errorMessage}
          refreshing={refreshing}
          showSettingsMenu={showSettingsMenu}
          showFeedMenuId={showFeedMenuId}
          onSelectAllFeeds={onSelectAllFeeds}
          onSelectFolder={onSelectFolder}
          onSelectFeed={onSelectFeed}
          onSelectSmartFeed={onSelectSmartFeed}
          onToggleFolder={onToggleFolder}
          onToggleShowUnreadOnly={onToggleShowUnreadOnly}
          onAddFeed={onAddFeed}
          onCreateFolder={onCreateFolder}
          onEditFeed={onEditFeed}
          onDeleteFeed={onDeleteFeed}
          onMarkFeedAsRead={onMarkFeedAsRead}
          onEditFolder={onEditFolder}
          onDeleteFolder={onDeleteFolder}
          onMarkFolderAsRead={onMarkFolderAsRead}
          onRefresh={onRefresh}
          onExportOPML={onExportOPML}
          onImportOPML={onImportOPML}
          onToggleSettingsMenu={onToggleSettingsMenu}
          onToggleFeedMenu={onToggleFeedMenu}
        />
      </RssErrorBoundary>
      <RssErrorBoundary>
        <RssArticlesPanel
          articles={articles}
          feeds={feeds}
          selectedArticle={selectedArticle}
          loading={loadingArticles}
          totalArticles={totalArticles}
          currentPage={currentPage}
          currentUnreadCount={currentUnreadCount}
          showUnreadOnly={showUnreadOnly}
          isSmartFeedActive={isSmartFeedActive}
          onArticleClick={onArticleClick}
          onToggleShowUnreadOnly={onToggleShowUnreadOnly}
          onPageChange={onPageChange}
        />
      </RssErrorBoundary>
      <RssErrorBoundary>
        <RssReaderPanel
          selectedArticle={selectedArticle}
          feeds={feeds}
          onConvertClick={onConvertClick}
          onMarkAsUnread={onMarkAsUnread}
        />
      </RssErrorBoundary>
    </div>
  );
}
