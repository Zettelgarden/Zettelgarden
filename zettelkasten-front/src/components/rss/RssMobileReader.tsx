import React, { useEffect, useRef } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { safeHtmlToMarkdown } from "../../utils/markdown";
import { RSSArticle } from "../../api/rss";

interface RssMobileReaderProps {
  article: RSSArticle;
  articles: RSSArticle[];
  onBack: () => void;
  onConvert: () => void;
  onMarkAsUnread: () => void;
  getFeedName: (feedId: number) => string;
  onViewCard: (cardId: number) => void;
  onFeedClick?: (feedId: number) => void;
  onArticleClick: (article: RSSArticle) => void;
  onToggleStar?: (articleId: number, isStarred: boolean) => void;
}

export function RssMobileReader({
  article,
  articles,
  onBack,
  onConvert,
  onMarkAsUnread,
  getFeedName,
  onViewCard,
  onFeedClick,
  onArticleClick,
  onToggleStar,
}: RssMobileReaderProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  // Find current article index
  const currentIndex = articles.findIndex(a => a.id === article.id);
  const hasNextArticle = currentIndex < articles.length - 1;
  const hasPreviousArticle = currentIndex > 0;

  const handleNext = () => {
    if (hasNextArticle) {
      onArticleClick(articles[currentIndex + 1]);
    }
  };

  const handlePrevious = () => {
    if (hasPreviousArticle) {
      onArticleClick(articles[currentIndex - 1]);
    }
  };

  // Scroll to top when article changes
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [article.id]);

  return (
    <div ref={scrollRef} className="fixed inset-0 bg-white z-50 overflow-y-auto flex flex-col md:hidden animate-slide-up">
      {/* Top bar */}
      <div className="sticky top-0 z-10 bg-white border-b border-gray-200 px-2 py-3 flex items-center justify-between shadow-sm">
        <div className="flex items-center">
          <button
            onClick={onBack}
            className="p-2 -ml-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
            aria-label="Back to articles"
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
        </div>

        <h2 className="text-base font-semibold text-gray-900 truncate flex-1 mx-2 text-center">
          Article {currentIndex + 1} of {articles.length}
        </h2>

        <div className="flex items-center gap-1">
          <button
            onClick={handlePrevious}
            disabled={!hasPreviousArticle}
            className="p-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Previous article"
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 15l7-7 7 7" />
            </svg>
          </button>
          <button
            onClick={handleNext}
            disabled={!hasNextArticle}
            className="p-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg disabled:opacity-30 disabled:cursor-not-allowed"
            aria-label="Next article"
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1">
        <div className="max-w-2xl mx-auto px-4 py-6">
          {/* Title */}
          <h1 className="text-xl font-bold mb-4 text-gray-900 leading-tight">
            {article.title}
          </h1>

          {/* Meta info */}
          <div className="flex flex-wrap items-center gap-3 text-sm text-gray-600 mb-6 pb-4 border-b border-gray-200">
            {article.author && (
              <span className="flex items-center gap-1">
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clipRule="evenodd" />
                </svg>
                {article.author}
              </span>
            )}
            <span className="flex items-center gap-1">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M6 2a1 1 0 00-1 1v1H4a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-1V3a1 1 0 10-2 0v1H7V3a1 1 0 00-1-1zm0 5a1 1 0 000 2h8a1 1 0 100-2H6z" clipRule="evenodd" />
              </svg>
              {article.published_at
                ? new Date(article.published_at).toLocaleDateString()
                : new Date(article.fetched_at).toLocaleDateString()}
            </span>
            <span
              onClick={() => onFeedClick?.(article.feed_id)}
              className="flex items-center gap-1 text-blue-600 hover:text-blue-800 hover:underline cursor-pointer"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z" />
                <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" />
              </svg>
              {getFeedName(article.feed_id)}
            </span>
          </div>

          {/* Content */}
          {article.content && (
            <div className="prose prose-base max-w-none mb-8">
              <Markdown
                remarkPlugins={[remarkGfm]}
                components={{
                  a: ({ href, children, ...props }) => (
                    <a href={href} target="_blank" rel="noopener noreferrer" {...props}>
                      {children}
                    </a>
                  ),
                  img: ({ src, alt, ...props }) => (
                    <img src={src} alt={alt} className="rounded-lg my-4" {...props} />
                  ),
                }}
              >
                {safeHtmlToMarkdown(article.content)}
              </Markdown>
            </div>
          )}
        </div>
      </div>

      {/* Bottom action bar */}
      <div className="sticky bottom-0 bg-white border-t border-gray-200 px-4 py-3 safe-area-inset-bottom">
        <div className="flex gap-2">
          {onToggleStar && (
            <button
              onClick={() => onToggleStar(article.id, article.is_starred)}
              className={`flex-1 px-4 py-3 rounded-lg transition-colors flex items-center justify-center gap-2 font-medium ${
                article.is_starred
                  ? "bg-amber-100 text-amber-700 hover:bg-amber-200"
                  : "bg-gray-100 text-gray-700 hover:bg-gray-200"
              }`}
              aria-label={article.is_starred ? "Unstar article" : "Star article"}
            >
              <svg
                className="w-5 h-5"
                fill={article.is_starred ? "currentColor" : "none"}
                viewBox="0 0 20 20"
                stroke={article.is_starred ? "none" : "currentColor"}
              >
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
              </svg>
              {article.is_starred ? "Starred" : "Star"}
            </button>
          )}

          {article.read && (
            <button
              onClick={onMarkAsUnread}
              className="flex-1 bg-gray-100 text-gray-700 px-4 py-3 rounded-lg hover:bg-gray-200 transition-colors flex items-center justify-center gap-2 font-medium"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              Mark Unread
            </button>
          )}

          <a
            href={article.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex-1 bg-gray-100 text-gray-700 px-4 py-3 rounded-lg hover:bg-gray-200 transition-colors flex items-center justify-center gap-2 font-medium text-center"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            View Original
          </a>

          {!article.card_id && (
            <button
              onClick={onConvert}
              className="flex-1 bg-blue-600 text-white px-4 py-3 rounded-lg hover:bg-blue-700 transition-colors flex items-center justify-center gap-2 font-medium"
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
              </svg>
              Convert
            </button>
          )}

          {article.card_id && (
            <button
              onClick={() => onViewCard(article.card_id!)}
              className="flex-1 bg-green-600 text-white px-4 py-3 rounded-lg hover:bg-green-700 transition-colors flex items-center justify-center gap-2 font-medium"
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
              </svg>
              View Card
            </button>
          )}
        </div>
      </div>

      <style>{`
        @keyframes slide-up {
          from {
            transform: translateY(100%);
          }
          to {
            transform: translateY(0);
          }
        }
        .animate-slide-up {
          animation: slide-up 0.3s ease-out;
        }
        .safe-area-inset-bottom {
          padding-bottom: env(safe-area-inset-bottom, 16px);
        }
      `}</style>
    </div>
  );
}
