import { useState, useEffect, useCallback, useRef } from "react";
import {
  listArticles,
  RSSArticle,
  ArticleFilters,
} from "../api/rss";
import { RSS_CONFIG } from "../constants/rss";

/**
 * Options for useRssArticles hook
 */
interface UseRssArticlesOptions {
  filters?: ArticleFilters;
  skip?: boolean; // Skip fetching when true (useful for conditional hooks)
}

/**
 * Hook for fetching and managing RSS articles with load more functionality
 * @param options - Options object with filters and skip flag
 * @returns Object containing articles, loading state, and control functions
 */
export function useRssArticles(options: UseRssArticlesOptions = {}) {
  const { filters = {}, skip = false } = options;
  const [articles, setArticles] = useState<RSSArticle[]>([]);
  const [totalArticles, setTotalArticles] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const errorTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const filtersRef = useRef(filters);

  // Keep track of current filters for resetting
  useEffect(() => {
    filtersRef.current = filters;
  }, [filters]);

  /**
   * Load initial articles (first batch)
   */
  const loadArticles = useCallback(async () => {
    if (skip) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const requestFilters: ArticleFilters = {
        ...filters,
        limit: RSS_CONFIG.ARTICLES_PER_PAGE,
        offset: 0,
      };

      const response = await listArticles(requestFilters);
      setArticles(response?.articles || []);
      setTotalArticles(response?.total || 0);
    } catch (error) {
      console.error("Failed to load articles:", error);
      setError("Failed to load articles. Please try again.");
      if (errorTimeoutRef.current) {
        clearTimeout(errorTimeoutRef.current);
      }
      errorTimeoutRef.current = setTimeout(() => setError(null), 5000);
    } finally {
      setLoading(false);
    }
  }, [filters, skip]);

  /**
   * Load more articles (append to existing)
   */
  const loadMore = useCallback(async () => {
    if (loadingMore || articles.length >= totalArticles) {
      return;
    }

    setLoadingMore(true);
    setError(null);
    try {
      const requestFilters: ArticleFilters = {
        ...filtersRef.current,
        limit: RSS_CONFIG.ARTICLES_PER_PAGE,
        offset: articles.length,
      };

      const response = await listArticles(requestFilters);
      setArticles((prev) => [...prev, ...(response?.articles || [])]);
      setTotalArticles(response?.total || 0);
    } catch (error) {
      console.error("Failed to load more articles:", error);
      setError("Failed to load more articles. Please try again.");
      if (errorTimeoutRef.current) {
        clearTimeout(errorTimeoutRef.current);
      }
      errorTimeoutRef.current = setTimeout(() => setError(null), 5000);
    } finally {
      setLoadingMore(false);
    }
  }, [loadingMore, articles.length, totalArticles]);

  /**
   * Reset and reload articles (called when filters change)
   */
  const resetToFirstPage = useCallback(() => {
    loadArticles();
  }, [loadArticles]);

  /**
   * Update an article in the local state
   */
  const updateArticle = useCallback((articleId: number, updates: Partial<RSSArticle>) => {
    setArticles((prev) =>
      prev.map((a) => (a.id === articleId ? { ...a, ...updates } : a))
    );
  }, []);

  /**
   * Update multiple articles in the local state
   */
  const updateArticles = useCallback((updater: (articles: RSSArticle[]) => RSSArticle[]) => {
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

  return {
    articles,
    totalArticles,
    loading,
    loadingMore,
    error,
    hasMore: articles.length < totalArticles,
    loadMore,
    resetToFirstPage,
    loadArticles,
    updateArticle,
    updateArticles,
  };
}
