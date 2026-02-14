import React from "react";
import { useNavigate } from "react-router-dom";
import { RSSFeed, RSSFolder, RSSArticle, RSSArticleWithScore, UnreadCounts } from "../../api/rss";
import { RssMobileTopBar } from "./RssMobileTopBar";
import { RssMobileReader } from "./RssMobileReader";
import { RssFeedsBottomSheet } from "./RssFeedsBottomSheet";
import { RssErrorBoundary } from "./RssErrorBoundary";
import { RSS_CONFIG } from "../../constants/rss";

interface RssMobileLayoutProps {
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
  totalUnreadCount: number;
  currentUnreadCount: number;
  showFeedMenuId: number | null;
  mobileView: 'list' | 'reader' | 'feeds';
  setMobileView: (view: 'list' | 'reader' | 'feeds') => void;
  loadingArticles: boolean;
  totalArticles: number;
  currentPage: number;
  hasMore?: boolean; // For smart feed client-side pagination
  onMenuClick: () => void;
  onFeedSelectMobile: (feedId: number) => void;
  onFolderSelectMobile: (folderName: string) => void;
  onAllFeedsSelectMobile: () => void;
  onSmartFeedSelectMobile?: () => void;
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
  onSelectSmartFeed: () => void;
  onConvertClick: () => void;
  onMarkAsUnread: () => void;
  onRefresh?: () => void;
  onExportOPML?: () => void;
  onImportOPML?: () => void;
  onMobileBack: () => void;
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
  isSmartFeedActive,
  expandedFolders,
  totalUnreadCount,
  currentUnreadCount,
  showFeedMenuId,
  mobileView,
  setMobileView,
  loadingArticles,
  totalArticles,
  currentPage,
  hasMore,
  onMenuClick,
  onFeedSelectMobile,
  onFolderSelectMobile,
  onAllFeedsSelectMobile,
  onSmartFeedSelectMobile,
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
  onSelectSmartFeed,
  onConvertClick,
  onMarkAsUnread,
  onRefresh,
  onExportOPML,
  onImportOPML,
  onMobileBack,
}: RssMobileLayoutProps) {
  const navigate = useNavigate();

  // Calculate whether to show "Load More" button
  // Use hasMore if provided (for smart feed), otherwise use server-side pagination calculation
  const showLoadMore = hasMore ?? (totalArticles > RSS_CONFIG.ARTICLES_PER_PAGE && currentPage * RSS_CONFIG.ARTICLES_PER_PAGE < totalArticles);

  const getFeedName = (feedId: number): string => {
    const feed = feeds.find((f) => f.id === feedId);
    return feed?.name || "Unknown Feed";
  };

  const handleMobileBack = () => {
    onMobileBack();
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
            title={isSmartFeedActive ? "Smart Feed" : "RSS"}
            unreadCount={totalUnreadCount}
            onMenuClick={onMenuClick}
            rightAction={
              <button
                onClick={() => onAllFeedsSelectMobile()}
                className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
                aria-label="Manage feeds and settings"
              >
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                </svg>
              </button>
            }
          />

          {/* Articles list content */}
          <div className="flex-1 bg-white flex flex-col overflow-hidden">
            {/* Filter tabs - shown for all views including Smart Feed */}
            <div className="p-4 border-b border-gray-200">
                <div className="flex bg-gray-100 rounded-lg p-1">
                  <button
                    onClick={() => {
                      if (showUnreadOnly) onToggleShowUnreadOnly();
                    }}
                    className={`flex-1 py-1.5 px-3 rounded-md text-sm font-medium transition-colors ${
                      !showUnreadOnly
                        ? "bg-white text-gray-900 shadow-sm"
                        : "text-gray-600 hover:text-gray-900"
                    }`}
                  >
                    All
                  </button>
                  <button
                    onClick={() => {
                      if (!showUnreadOnly) onToggleShowUnreadOnly();
                    }}
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
                  {articles.map((article) => {
                    const articleWithScore = article as RSSArticleWithScore;
                    const hasSmartScore = 'smart_score' in article && articleWithScore.smart_score;
                    return (
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
                          <div className="flex items-center gap-2 flex-shrink-0">
                            {hasSmartScore && articleWithScore.smart_score && (
                              <div className="flex items-center gap-1" title={articleWithScore.smart_score.reason}>
                                <span className="text-sm font-medium text-amber-600">
                                  {articleWithScore.smart_score.score.toFixed(1)}
                                </span>
                                <svg className="w-4 h-4 text-amber-500" fill="currentColor" viewBox="0 0 20 20">
                                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                                </svg>
                              </div>
                            )}
                            {article.card_id && (
                              <svg className="w-5 h-5 text-green-600" fill="currentColor" viewBox="0 0 20 20">
                                <title>Converted to card</title>
                                <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                              </svg>
                            )}
                          </div>
                        </div>
                        <p className="text-sm text-gray-500 mt-2">
                          {getFeedName(article.feed_id)} • {new Date(article.fetched_at).toLocaleDateString()}
                        </p>
                      </div>
                    );
                  })}
                </div>

                {/* Pagination - Load More button for mobile */}
                {showLoadMore && (
                  <div className="p-4 border-t border-gray-200 bg-gray-50">
                    <button
                      onClick={onLoadMore}
                      className="w-full bg-white border border-gray-300 text-gray-700 px-4 py-3 rounded-lg hover:bg-gray-50 transition-colors font-medium"
                    >
                      Load More Articles
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
          onFeedClick={onFeedSelectMobile}
        />
      </RssErrorBoundary>
    );
  }

  // Mobile Feeds View
  if (mobileView === 'feeds') {
    return (
      <RssErrorBoundary>
        <RssFeedsBottomSheet
          isOpen={true}
          onClose={() => setMobileView('list')}
          feeds={feeds}
          folders={folders}
          unreadCounts={unreadCounts}
          expandedFolders={expandedFolders}
          onToggleFolder={onToggleFolder}
          onSelectFeed={onFeedSelectMobile}
          onSelectFolder={onFolderSelectMobile}
          onSelectAllFeeds={onAllFeedsSelectMobile}
          onSelectSmartFeed={onSmartFeedSelectMobile}
          onAddFeed={onAddFeed}
          onCreateFolder={onCreateFolder}
          onEditFeed={onEditFeed}
          onDeleteFeed={onDeleteFeed}
          onMarkFeedAsRead={onMarkFeedAsRead}
          onEditFolder={onEditFolder}
          onDeleteFolder={onDeleteFolder}
          onMarkFolderAsRead={onMarkFolderAsRead}
          selectedFeedId={selectedFeedId}
          selectedFolder={selectedFolder}
          isSmartFeedActive={isSmartFeedActive}
          showFeedMenuId={showFeedMenuId}
          onShowFeedMenu={onShowFeedMenu}
          onRefresh={onRefresh}
          onExportOPML={onExportOPML}
          onImportOPML={onImportOPML}
        />
      </RssErrorBoundary>
    );
  }

  return null;
}
