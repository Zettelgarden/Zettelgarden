import { useState, useEffect, useCallback, useRef } from "react";
import {
  getSmartRSSArticles,
  RSSArticleWithScore,
} from "../api/rss";
import { RSS_CONFIG } from "../constants/rss";

export interface SmartFeedFilters {
  folder?: string;
  unread?: boolean;
}

/**
 * Options for useSmartRssArticles hook
 */
interface UseSmartRssArticlesOptions {
  filters?: SmartFeedFilters;
  skip?: boolean; // Skip fetching when true (useful for conditional hooks)
}

/**
 * Hook for fetching and managing RSS articles with smart scoring.
 * Fetches all articles at once and uses client-side load more.
 * @param options - Options object with filters and skip flag
 * @returns Object containing all articles, visible count, loading state, and control functions
 */
export function useSmartRssArticles(options: UseSmartRssArticlesOptions = {}) {
  const { filters = {}, skip = false } = options;
  const [articles, setArticles] = useState<RSSArticleWithScore[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [visibleCount, setVisibleCount] = useState<number>(RSS_CONFIG.ARTICLES_PER_PAGE);
  const errorTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Load all articles (no pagination on the server side)
   */
  const loadArticles = useCallback(async () => {
    if (skip) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);
    try {
      // Fetch all articles without limit/offset to avoid duplicates
      const response = await getSmartRSSArticles(filters);
      // Ensure articles is always an array, even if API returns null/undefined
      setArticles(response?.articles || []);
      // Reset visible count when loading new articles
      setVisibleCount(RSS_CONFIG.ARTICLES_PER_PAGE);
    } catch (error) {
      console.error("Failed to load smart articles:", error);
      setError("Failed to load articles. Please try again.");
      // Clear any existing timeout before setting a new one
      if (errorTimeoutRef.current) {
        clearTimeout(errorTimeoutRef.current);
      }
      errorTimeoutRef.current = setTimeout(() => setError(null), 5000);
    } finally {
      setLoading(false);
    }
  }, [filters, skip]);

  /**
   * Load more articles (client-side - increase visible count)
   */
  const loadMore = useCallback(() => {
    setVisibleCount((prev) => prev + RSS_CONFIG.ARTICLES_PER_PAGE);
  }, []);

  /**
   * Reset and reload articles
   */
  const resetToFirstPage = useCallback(() => {
    loadArticles();
  }, [loadArticles]);

  /**
   * Update an article in the local state
   */
  const updateArticle = useCallback((articleId: number, updates: Partial<RSSArticleWithScore>) => {
    setArticles((prev) =>
      prev.map((a) => (a.id === articleId ? { ...a, ...updates } : a))
    );
  }, []);

  /**
   * Update multiple articles in the local state
   */
  const updateArticles = useCallback((updater: (articles: RSSArticleWithScore[]) => RSSArticleWithScore[]) => {
    setArticles(updater);
  }, []);

  // Load articles when filters changes
  useEffect(() => {
    loadArticles();
  }, [loadArticles]);

  // Cleanup error timeout on unmount
  useEffect(() => {
    return () => {
      if (errorTimeoutRef.current) {
        clearTimeout(errorTimeoutRef.current);
      }
    };
  }, []);

  // Client-side: filter articles to show based on visible count
  const visibleArticles = articles.slice(0, visibleCount);
  const hasMore = visibleCount < articles.length;

  return {
    articles: visibleArticles,
    allArticles: articles, // Expose all articles for filtering
    totalArticles: articles.length,
    loading,
    error,
    visibleCount,
    hasMore,
    loadMore,
    resetToFirstPage,
    loadArticles,
    updateArticle,
    updateArticles,
  };
}
