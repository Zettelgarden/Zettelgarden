import React from "react";
import { useNavigate } from "react-router-dom";
import { RSSFeed, RSSFolder, RSSArticle, UnreadCounts } from "../../api/rss";
import { RssMobileTopBar } from "./RssMobileTopBar";
import { RssMobileReader } from "./RssMobileReader";
import { RssFeedsBottomSheet } from "./RssFeedsBottomSheet";
import { RssErrorBoundary } from "./RssErrorBoundary";
import { RSS_CONFIG } from "../../constants/rss";

interface RssMobileLayoutProps {
  feeds: RSSFeed[];
  folders: RSSFolder[];
  unreadCounts: UnreadCounts;
  articles: RSSArticle[];
  selectedFolder: string | null;
  selectedFeedId: number | null;
  selectedArticle: RSSArticle | null;
  showUnreadOnly: boolean;
  expandedFolders: Set<string>;
  totalUnreadCount: number;
  currentUnreadCount: number;
  showFeedMenuId: number | null;
  mobileView: 'list' | 'reader' | 'feeds';
  loadingArticles: boolean;
  totalArticles: number;
  currentPage: number;
  onMenuClick: () => void;
  onSettingsClick: () => void;
  onFeedSelectMobile: (feedId: number) => void;
  onFolderSelectMobile: (folderName: string) => void;
  onAllFeedsSelectMobile: () => void;
  onToggleFolder: (folderName: string) => void;
  onAddFeed: () => void;
  onCreateFolder: () => void;
  onEditFeed: (feed: RSSFeed) => void;
  onDeleteFeed: (feed: RSSFeed) => void;
  onMarkFeedAsRead: (feed: RSSFeed) => void;
  onEditFolder: (folder: RSSFolder) => void;
  onDeleteFolder: (folder: RSSFolder) => void;
  onMarkFolderAsRead: (folder: RSSFolder) => void;
  onShowFeedMenu: (feedId: number | null) => void;
  onArticleClick: (article: RSSArticle) => void;
  onLoadMore: () => void;
  onToggleShowUnreadOnly: () => void;
  onConvertClick: () => void;
  onMarkAsUnread: () => void;
}

/**
 * Mobile layout for RSS reader with article list and reader views
 */
export function RssMobileLayout({
  feeds,
  folders,
  unreadCounts,
  articles,
  selectedFolder,
  selectedFeedId,
  selectedArticle,
  showUnreadOnly,
  expandedFolders,
  totalUnreadCount,
  currentUnreadCount,
  showFeedMenuId,
  mobileView,
  loadingArticles,
  totalArticles,
  currentPage,
  onMenuClick,
  onSettingsClick,
  onFeedSelectMobile,
  onFolderSelectMobile,
  onAllFeedsSelectMobile,
  onToggleFolder,
  onAddFeed,
  onCreateFolder,
  onEditFeed,
  onDeleteFeed,
  onMarkFeedAsRead,
  onEditFolder,
  onDeleteFolder,
  onMarkFolderAsRead,
  onShowFeedMenu,
  onArticleClick,
  onLoadMore,
  onToggleShowUnreadOnly,
  onConvertClick,
  onMarkAsUnread,
}: RssMobileLayoutProps) {
  const navigate = useNavigate();

  const getFeedName = (feedId: number): string => {
    const feed = feeds.find((f) => f.id === feedId);
    return feed?.name || "Unknown Feed";
  };

  const handleMobileBack = () => {
    // This will be handled by parent component
  };

  const handleViewCard = (cardId: number) => {
    navigate(`/app/card/${cardId}`);
  };

  // Mobile Article List View
  if (mobileView === 'list') {
    return (
      <RssErrorBoundary>
        <div className="md:hidden flex flex-col flex-1 overflow-hidden">
          <RssMobileTopBar
            title="RSS"
            unreadCount={totalUnreadCount}
            onMenuClick={onMenuClick}
            onSettingsClick={onSettingsClick}
            rightAction={
              <div className="flex items-center gap-1">
                {/* Feeds button */}
                <button
                  onClick={() => onAllFeedsSelectMobile()}
                  className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
                  aria-label="Open feeds"
                >
                  <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                  </svg>
                </button>
                {/* Settings button */}
                <button
                  onClick={onSettingsClick}
                  className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
                  aria-label="Settings"
                >
                  <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  </svg>
                </button>
              </div>
            }
          />

          {/* Articles list content */}
          <div className="flex-1 bg-white flex flex-col overflow-hidden">
            {/* Filter tabs */}
            <div className="p-4 border-b border-gray-200">
              <div className="flex bg-gray-100 rounded-lg p-1">
                <button
                  onClick={() => !showUnreadOnly && onToggleShowUnreadOnly()}
                  className={`flex-1 py-1.5 px-3 rounded-md text-sm font-medium transition-colors ${
                    !showUnreadOnly
                      ? "bg-white text-gray-900 shadow-sm"
                      : "text-gray-600 hover:text-gray-900"
                  }`}
                >
                  All
                </button>
                <button
                  onClick={() => showUnreadOnly && onToggleShowUnreadOnly()}
                  className={`flex-1 py-1.5 px-3 rounded-md text-sm font-medium transition-colors relative ${
                    showUnreadOnly
                      ? "bg-white text-gray-900 shadow-sm"
                      : "text-gray-600 hover:text-gray-900"
                  }`}
                >
                  Unread
                  {currentUnreadCount > 0 && (
                    <span className="ml-1 bg-blue-500 text-white text-xs px-1.5 py-0.5 rounded-full">
                      {currentUnreadCount}
                    </span>
                  )}
                </button>
              </div>
            </div>

            {/* Articles list */}
            {articles.length === 0 && !loadingArticles ? (
              <div className="flex-1 flex items-center justify-center">
                <p className="text-gray-500">No articles found</p>
              </div>
            ) : (
              <>
                <div className="flex-1 overflow-y-auto p-4 space-y-3">
                  {articles.map((article) => (
                    <div
                      key={article.id}
                      onClick={() => onArticleClick(article)}
                      className={`p-4 rounded-lg cursor-pointer transition-colors min-h-[60px] ${
                        selectedArticle?.id === article.id
                          ? "bg-blue-100 border-l-4 border-blue-600"
                          : article.read
                            ? "bg-gray-50 hover:bg-gray-100"
                            : "bg-white hover:bg-gray-100 border-l-4 border-blue-500 shadow-sm"
                      }`}
                    >
                      <div className="flex items-start gap-3">
                        <h3 className="font-semibold text-base line-clamp-3 flex-1 text-gray-900">
                          {article.title}
                        </h3>
                        {article.card_id && (
                          <svg className="w-5 h-5 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                            <title>Converted to card</title>
                            <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                          </svg>
                        )}
                      </div>
                      <p className="text-sm text-gray-500 mt-2">
                        {getFeedName(article.feed_id)} • {new Date(article.fetched_at).toLocaleDateString()}
                      </p>
                    </div>
                  ))}
                </div>

                {/* Pagination - Load More button for mobile */}
                {totalArticles > RSS_CONFIG.ARTICLES_PER_PAGE && currentPage * RSS_CONFIG.ARTICLES_PER_PAGE < totalArticles && (
                  <div className="p-4 border-t border-gray-200 bg-gray-50">
                    <button
                      onClick={onLoadMore}
                      className="w-full bg-white border border-gray-300 text-gray-700 px-4 py-3 rounded-lg hover:bg-gray-50 transition-colors font-medium"
                    >
                      Load More Articles ({totalArticles - currentPage * RSS_CONFIG.ARTICLES_PER_PAGE} remaining)
                    </button>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      </RssErrorBoundary>
    );
  }

  // Mobile Reader View
  if (mobileView === 'reader' && selectedArticle) {
    return (
      <RssErrorBoundary>
        <RssMobileReader
          article={selectedArticle}
          onBack={handleMobileBack}
          onConvert={onConvertClick}
          onMarkAsUnread={onMarkAsUnread}
          getFeedName={getFeedName}
          onViewCard={handleViewCard}
        />
      </RssErrorBoundary>
    );
  }

  return null;
}
