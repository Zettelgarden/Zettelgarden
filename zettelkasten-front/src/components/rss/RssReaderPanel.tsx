import React from "react";
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
}

/**
 * Right panel showing article content reader
 */
export function RssReaderPanel({
  selectedArticle,
  feeds,
  onConvertClick,
  onMarkAsUnread,
}: RssReaderPanelProps) {
  const navigate = useNavigate();

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
    <div className="hidden md:flex flex-1 p-6 overflow-y-auto bg-white min-w-0 flex-col">
      <div className="max-w-3xl mx-auto">
        <div className="mb-6">
          <h1 className="text-2xl font-bold mb-3 text-gray-900">
            {selectedArticle.title}
          </h1>
          <div className="flex flex-wrap items-center gap-4 text-sm text-gray-600">
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

        <div className="flex flex-col sm:flex-row gap-3 pt-6 border-t border-gray-200">
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
