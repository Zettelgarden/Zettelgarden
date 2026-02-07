import React, { useState, useEffect } from "react";
import { setDocumentTitle } from "../utils/title";
import {
  listFeeds,
  listArticles,
  listFolders,
  markAsRead,
  convertToCard,
  refreshFeeds,
  RSSFeed,
  RSSArticle,
  RSSFolder,
} from "../api/rss";
import { RssAddFeedDialog } from "../components/rss/RssAddFeedDialog";
import { RssConvertDialog } from "../components/rss/RssConvertDialog";

export function RssPage() {
  const [feeds, setFeeds] = useState<RSSFeed[]>([]);
  const [articles, setArticles] = useState<RSSArticle[]>([]);
  const [folders, setFolders] = useState<RSSFolder[]>([]);
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
  const [selectedArticle, setSelectedArticle] = useState<RSSArticle | null>(null);
  const [showUnreadOnly, setShowUnreadOnly] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [showAddFeedDialog, setShowAddFeedDialog] = useState(false);
  const [showConvertDialog, setShowConvertDialog] = useState(false);

  useEffect(() => {
    setDocumentTitle("RSS");
    loadData();
  }, []);

  useEffect(() => {
    loadArticles();
  }, [selectedFolder, showUnreadOnly]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [feedsData, foldersData] = await Promise.all([
        listFeeds(),
        listFolders(),
      ]);
      setFeeds(feedsData);
      setFolders(foldersData);
    } catch (error) {
      console.error("Failed to load RSS data:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadArticles = async () => {
    try {
      const filters: any = {};
      if (selectedFolder) filters.folder = selectedFolder;
      if (showUnreadOnly) filters.unread = true;
      filters.limit = 50;

      const articlesData = await listArticles(filters);
      setArticles(articlesData);
    } catch (error) {
      console.error("Failed to load articles:", error);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await refreshFeeds();
      await loadData();
      await loadArticles();
    } catch (error) {
      console.error("Failed to refresh feeds:", error);
    } finally {
      setRefreshing(false);
    }
  };

  const handleArticleClick = async (article: RSSArticle) => {
    setSelectedArticle(article);
    if (!article.read) {
      try {
        await markAsRead(article.id, true);
        setArticles((prev) =>
          prev.map((a) => (a.id === article.id ? { ...a, read: true } : a))
        );
      } catch (error) {
        console.error("Failed to mark as read:", error);
      }
    }
  };

  const handleFeedAdded = (feed: RSSFeed) => {
    setFeeds((prev) => [...prev, feed]);
    setShowAddFeedDialog(false);
  };

  const handleConvertClick = () => {
    setShowConvertDialog(true);
  };

  const handleConverted = (cardId: number) => {
    setShowConvertDialog(false);
    // Navigate to the new card
    window.location.href = `/app/card/${cardId}`;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-gray-500">Loading RSS feeds...</div>
      </div>
    );
  }

  return (
    <div className="flex h-full">
      {/* Left Panel: Folders */}
      <div className="w-64 border-r border-gray-200 p-4 overflow-y-auto bg-gray-50">
        <div className="mb-4">
          <h2 className="text-lg font-semibold mb-3">RSS Feeds</h2>
          <div className="space-y-2">
            <button
              onClick={handleRefresh}
              disabled={refreshing}
              className="w-full bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            >
              {refreshing ? "Refreshing..." : "Refresh All"}
            </button>
            <button
              onClick={() => setShowAddFeedDialog(true)}
              className="w-full bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700 transition-colors flex items-center justify-center gap-2"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
              </svg>
              Add Feed
            </button>
          </div>
        </div>

        <div className="mb-4">
          <button
            onClick={() => setSelectedFolder(null)}
            className={`w-full text-left px-3 py-2 rounded-md transition-colors ${
              selectedFolder === null ? "bg-blue-100 text-blue-900" : "hover:bg-gray-100"
            }`}
          >
            All Feeds ({feeds.length})
          </button>
        </div>

        <div className="mb-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={showUnreadOnly}
              onChange={(e) => setShowUnreadOnly(e.target.checked)}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm">Unread only</span>
          </label>
        </div>

        {folders.length > 0 && (
          <div>
            <h3 className="text-xs font-semibold text-gray-500 mb-2 uppercase tracking-wide">
              Folders
            </h3>
            <div className="space-y-1">
              {folders.map((folder) => (
                <button
                  key={folder.id}
                  onClick={() => setSelectedFolder(folder.name)}
                  className={`w-full text-left px-3 py-2 rounded-md transition-colors ${
                    selectedFolder === folder.name ? "bg-blue-100 text-blue-900" : "hover:bg-gray-100"
                  }`}
                >
                  {folder.name}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Middle Panel: Articles */}
      <div className="w-80 border-r border-gray-200 p-4 overflow-y-auto bg-white">
        <h2 className="text-lg font-semibold mb-4">Articles</h2>
        {articles.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No articles found</p>
        ) : (
          <div className="space-y-2">
            {articles.map((article) => (
              <div
                key={article.id}
                onClick={() => handleArticleClick(article)}
                className={`p-3 rounded-md cursor-pointer transition-colors ${
                  selectedArticle?.id === article.id
                    ? "bg-blue-100 border-l-4 border-blue-600"
                    : article.read
                    ? "bg-gray-50 hover:bg-gray-100"
                    : "bg-white hover:bg-gray-100 border-l-4 border-blue-500 shadow-sm"
                }`}
              >
                <h3 className="font-medium text-sm line-clamp-2 mb-1">
                  {article.title}
                </h3>
                <p className="text-xs text-gray-500">
                  {new Date(article.fetched_at).toLocaleDateString()}
                </p>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Right Panel: Article Reader */}
      <div className="flex-1 p-6 overflow-y-auto bg-white">
        {selectedArticle ? (
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
              </div>
            </div>

            {selectedArticle.content && (
              <div
                className="prose prose-sm max-w-none mb-8"
                dangerouslySetInnerHTML={{ __html: selectedArticle.content }}
              />
            )}

            <div className="flex flex-col sm:flex-row gap-3 pt-6 border-t border-gray-200">
              <button
                onClick={handleConvertClick}
                className="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700 transition-colors flex items-center justify-center gap-2"
              >
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                </svg>
                Convert to Card
              </button>
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-gray-400">
            <svg className="w-16 h-16 mb-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M2 5a2 2 0 012-2h8a2 2 0 012 2v10a2 2 0 002 2H4a2 2 0 01-2-2V5zm3 1h6v4H5V6zm6 6H5v2h6v-2z" clipRule="evenodd" />
              <path d="M15 7h1a2 2 0 012 2v5.5a1.5 1.5 0 01-1.5 1.5h-1v-1h1a.5.5 0 00.5-.5V9a1 1 0 00-1-1h-1V7z" />
            </svg>
            <p className="text-lg">Select an article to read</p>
          </div>
        )}
      </div>

      {/* Dialogs */}
      <RssAddFeedDialog
        isOpen={showAddFeedDialog}
        onClose={() => setShowAddFeedDialog(false)}
        onFeedAdded={handleFeedAdded}
      />
      <RssConvertDialog
        isOpen={showConvertDialog}
        onClose={() => setShowConvertDialog(false)}
        article={selectedArticle}
        onConverted={handleConverted}
      />
    </div>
  );
}
