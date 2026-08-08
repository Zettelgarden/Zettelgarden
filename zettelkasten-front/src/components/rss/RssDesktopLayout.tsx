import React from 'react';
import {
  RSSFeed,
  RSSFolder,
  RSSArticle,
  RSSArticleWithScore,
  UnreadCounts,
} from '../../api/rss';
import { RssFeedsPanel } from './RssFeedsPanel';
import { RssArticlesPanel } from './RssArticlesPanel';
import { RssReaderPanel } from './RssReaderPanel';
import { RssErrorBoundary } from './RssErrorBoundary';

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
  loadingMoreArticles: boolean;
  totalArticles: number;
  hasMore: boolean;
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
  onLoadMore: () => void;
  onConvertClick: () => void;
  onMarkAsUnread: () => void;
  isStarredFeedActive: boolean;
  starredCount?: number;
  onSelectStarredFeed: () => void;
  onToggleStar: (articleId: number, isStarred: boolean) => void;
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
  loadingMoreArticles,
  totalArticles,
  hasMore,
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
  onLoadMore,
  onConvertClick,
  onMarkAsUnread,
  isStarredFeedActive,
  starredCount,
  onSelectStarredFeed,
  onToggleStar,
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
          isStarredFeedActive={isStarredFeedActive}
          starredCount={starredCount}
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
          onSelectStarredFeed={onSelectStarredFeed}
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
          loadingMore={loadingMoreArticles}
          totalArticles={totalArticles}
          currentUnreadCount={currentUnreadCount}
          showUnreadOnly={showUnreadOnly}
          isSmartFeedActive={isSmartFeedActive}
          hasMore={hasMore}
          onArticleClick={onArticleClick}
          onToggleShowUnreadOnly={onToggleShowUnreadOnly}
          onLoadMore={onLoadMore}
          onToggleStar={onToggleStar}
        />
      </RssErrorBoundary>
      <RssErrorBoundary>
        <RssReaderPanel
          selectedArticle={selectedArticle}
          feeds={feeds}
          onConvertClick={onConvertClick}
          onMarkAsUnread={onMarkAsUnread}
          onFeedClick={onSelectFeed}
          onToggleStar={onToggleStar}
        />
      </RssErrorBoundary>
    </div>
  );
}
