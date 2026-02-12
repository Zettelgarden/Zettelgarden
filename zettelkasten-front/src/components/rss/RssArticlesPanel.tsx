import React from "react";
import { RSSArticle, RSSArticleWithScore, RSSFeed } from "../../api/rss";
import { RSS_CONFIG } from "../../constants/rss";

interface RssArticlesPanelProps {
  articles: (RSSArticle | RSSArticleWithScore)[];
  feeds: RSSFeed[];
  selectedArticle: RSSArticle | null;
  loading: boolean;
  totalArticles: number;
  currentPage: number;
  currentUnreadCount: number;
  showUnreadOnly: boolean;
  isSmartFeedActive: boolean;
  onArticleClick: (article: RSSArticle) => void;
  onToggleShowUnreadOnly: () => void;
  onPageChange: (page: number) => void;
}

/**
 * Middle panel showing articles list with pagination
 */
export function RssArticlesPanel({
  articles,
  feeds,
  selectedArticle,
  loading,
  totalArticles,
  currentPage,
  currentUnreadCount,
  showUnreadOnly,
  isSmartFeedActive,
  onArticleClick,
  onToggleShowUnreadOnly,
  onPageChange,
}: RssArticlesPanelProps) {
  const getFeedName = (feedId: number): string => {
    const feed = feeds.find((f) => f.id === feedId);
    return feed?.name || "Unknown Feed";
  };

  const totalPages = Math.ceil(totalArticles / RSS_CONFIG.ARTICLES_PER_PAGE);

  return (
    <div className="hidden md:flex w-80 border-r border-gray-200 bg-white flex-shrink-0 flex-col">
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold">
            {isSmartFeedActive ? (
              <span className="flex items-center gap-2">
                <svg className="w-5 h-5 text-amber-500" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                </svg>
                Smart Feed
              </span>
            ) : "Articles"}
          </h2>
          <span className="text-xs text-gray-500">
            {totalArticles > 0 && `${totalArticles} total`}
          </span>
        </div>
        {/* Filter tabs - shown for all views including Smart Feed */}
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
      {loading ? (
        <div className="flex-1 flex items-center justify-center">
          <p className="text-gray-500">Loading...</p>
        </div>
      ) : !articles || articles.length === 0 ? (
        <div className="flex-1 flex items-center justify-center">
          <p className="text-gray-500">No articles found</p>
        </div>
      ) : (
        <>
          <div className="flex-1 overflow-y-auto p-4 space-y-2">
            {articles.map((article) => {
              const articleWithScore = article as RSSArticleWithScore;
              const hasSmartScore = 'smart_score' in article && articleWithScore.smart_score;
              return (
                <div
                  key={article.id}
                  onClick={() => onArticleClick(article)}
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
                    <div className="flex items-center gap-1 flex-shrink-0">
                      {hasSmartScore && articleWithScore.smart_score && (
                        <div className="flex items-center gap-1" title={articleWithScore.smart_score.reason}>
                          <span className="text-xs font-medium text-amber-600">
                            {articleWithScore.smart_score.score.toFixed(1)}
                          </span>
                          <svg className="w-3.5 h-3.5 text-amber-500" fill="currentColor" viewBox="0 0 20 20">
                            <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                          </svg>
                        </div>
                      )}
                      {article.card_id && (
                        <svg className="w-4 h-4 text-green-600" fill="currentColor" viewBox="0 0 20 20">
                          <title>Converted to card</title>
                          <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                        </svg>
                      )}
                    </div>
                  </div>
                  <p className="text-xs text-gray-500">
                    {getFeedName(article.feed_id)} • {new Date(article.fetched_at).toLocaleDateString()}
                  </p>
                </div>
              );
            })}
          </div>

          {/* Pagination */}
          {totalArticles > RSS_CONFIG.ARTICLES_PER_PAGE && (
            <div className="p-3 border-t border-gray-200 bg-gray-50">
              <div className="flex items-center justify-between text-sm">
                <button
                  onClick={() => onPageChange(currentPage - 1)}
                  disabled={currentPage === 1}
                  className="px-3 py-1 rounded border border-gray-300 hover:bg-gray-100 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                <span className="text-gray-600">
                  Page {currentPage} of {totalPages}
                </span>
                <button
                  onClick={() => onPageChange(currentPage + 1)}
                  disabled={currentPage >= totalPages}
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
  );
}
