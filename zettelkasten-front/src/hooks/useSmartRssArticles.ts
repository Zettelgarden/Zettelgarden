import { useState, useEffect, useCallback, useRef } from "react";
import {
  getSmartRSSArticles,
  RSSArticleWithScore,
} from "../api/rss";

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
 * Fetches all articles at once to avoid duplicate issues with pagination.
 * @param options - Options object with filters and skip flag
 * @returns Object containing all articles, loading state, and control functions
 */
export function useSmartRssArticles(options: UseSmartRssArticlesOptions = {}) {
  const { filters = {}, skip = false } = options;
  const [articles, setArticles] = useState<RSSArticleWithScore[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
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

  return {
    articles,
    totalArticles: articles.length,
    loading,
    error,
    currentPage: 1, // Always page 1 since we load everything
    setCurrentPage: () => {}, // No-op
    resetToFirstPage,
    loadArticles,
    updateArticle,
    updateArticles,
    totalPages: 1,
    hasMore: false,
  };
}
