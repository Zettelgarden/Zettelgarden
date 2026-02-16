import React, { useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { RSSArticle, RSSFeed } from "../../api/rss";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { safeHtmlToMarkdown } from "../../utils/markdown";

interface RssReaderPanelProps {
  selectedArticle: RSSArticle | null;
  feeds: RSSFeed[];
  onConvertClick: () => void;
  onMarkAsUnread: () => void;
  onToggleStar: (articleId: number, isStarred: boolean) => void;
  onFeedClick?: (feedId: number) => void;
}

/**
 * Right panel showing article content reader
 */
export function RssReaderPanel({
  selectedArticle,
  feeds,
  onConvertClick,
  onMarkAsUnread,
  onToggleStar,
  onFeedClick,
}: RssReaderPanelProps) {
  const navigate = useNavigate();
  const scrollRef = useRef<HTMLDivElement>(null);

  // Scroll to top when article changes
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [selectedArticle?.id]);

  const getFeedName = (feedId: number): string => {
    const feed = feeds.find((f) => f.id === feedId);
    return feed?.name || "Unknown Feed";
  };

  if (!selectedArticle) {
    return (
      <div className="hidden md:flex flex-1 p-6 overflow-y-auto bg-white min-w-0 flex-col">
        <div className="flex flex-col items-center justify-center h-full text-gray-400">
          <svg className="w-16 h-16 mb-4" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M2 5a2 2 0 012-2h8a2 2 0 012 2v10a2 2 0 002 2H4a2 2 0 01-2-2V5zm3 1h6v4H5V6zm6 6H5v2h6v-2z" clipRule="evenodd" />
            <path d="M15 7h1a2 2 0 012 2v5.5a1.5 1.5 0 01-1.5 1.5h-1v-1h1a.5.5 0 00.5-.5V9a1 1 0 00-1-1h-1V7z" />
          </svg>
          <p className="text-lg">Select an article to read</p>
        </div>
      </div>
    );
  }

  return (
    <div ref={scrollRef} className="hidden md:flex flex-1 p-6 overflow-y-auto bg-white min-w-0 flex-col">
      <div className="max-w-3xl mx-auto">
        <div className="mb-6">
          <h1 className="text-2xl font-bold mb-3 text-gray-900">
            {selectedArticle.title}
          </h1>
          <div className="flex flex-wrap items-center gap-4 text-sm text-gray-600">
            <span
              onClick={() => onFeedClick?.(selectedArticle.feed_id)}
              className="flex items-center gap-1 text-blue-600 hover:text-blue-800 hover:underline cursor-pointer"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z" />
                <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" />
              </svg>
              {getFeedName(selectedArticle.feed_id)}
            </span>
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
            {selectedArticle.card_id ? (
              <button
                onClick={() => navigate(`/app/card/${selectedArticle.card_id}`)}
                className="flex items-center gap-1 text-green-600 hover:text-green-800 hover:underline"
              >
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                </svg>
                View card
              </button>
            ) : (
              <button
                onClick={onConvertClick}
                className="flex items-center gap-1 text-blue-600 hover:text-blue-800 hover:underline font-medium"
              >
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                </svg>
                Convert to Card
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

        <div className="flex flex-wrap gap-3 pt-6 border-t border-gray-200">
          <button
            onClick={() => selectedArticle && onToggleStar(selectedArticle.id, selectedArticle.is_starred || false)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg border ${
              selectedArticle?.is_starred
                ? "bg-amber-50 border-amber-300 text-amber-700"
                : "bg-white border-gray-300 text-gray-700 hover:bg-gray-50"
            }`}
            title={selectedArticle?.is_starred ? "Unstar article" : "Star article"}
          >
            <svg
              className={`w-5 h-5 ${selectedArticle?.is_starred ? "fill-amber-500 text-amber-500" : "text-gray-500"}`}
              fill={selectedArticle?.is_starred ? "currentColor" : "none"}
              stroke="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={selectedArticle?.is_starred ? 0 : 2}
                d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.364 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.364-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
              />
            </svg>
            <span>{selectedArticle?.is_starred ? "Starred" : "Star"}</span>
          </button>
          {!selectedArticle.card_id && (
            <button
              onClick={onConvertClick}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors"
            >
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
              </svg>
              <span>Convert to Card</span>
            </button>
          )}
          {selectedArticle.read && (
            <button
              onClick={onMarkAsUnread}
              className="bg-gray-600 text-white px-6 py-2 rounded-md hover:bg-gray-700 transition-colors flex items-center justify-center gap-2"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              Mark as Unread
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
