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
 * Hook for fetching and managing RSS articles with smart scoring
 * @param filters - Article filters to apply
 * @returns Object containing articles with scores, pagination state, loading state, and control functions
 */
export function useSmartRssArticles(filters: SmartFeedFilters = {}) {
  const [articles, setArticles] = useState<RSSArticleWithScore[]>([]);
  const [totalArticles, setTotalArticles] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const errorTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Load articles based on current filters and page
   */
  const loadArticles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const requestFilters = {
        ...filters,
        limit: RSS_CONFIG.ARTICLES_PER_PAGE,
        offset: (currentPage - 1) * RSS_CONFIG.ARTICLES_PER_PAGE,
      };

      const response = await getSmartRSSArticles(requestFilters);
      // Ensure articles is always an array, even if API returns null/undefined
      setArticles(response?.articles || []);
      setTotalArticles(response?.total || 0);
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
  }, [filters, currentPage]);

  /**
   * Go to next page
   */
  const nextPage = useCallback(() => {
    const maxPage = Math.ceil(totalArticles / RSS_CONFIG.ARTICLES_PER_PAGE);
    setCurrentPage((prev) => Math.min(maxPage, prev + 1));
  }, [totalArticles]);

  /**
   * Go to previous page
   */
  const prevPage = useCallback(() => {
    setCurrentPage((prev) => Math.max(1, prev - 1));
  }, []);

  /**
   * Reset to first page
   */
  const resetToFirstPage = useCallback(() => {
    setCurrentPage(1);
  }, []);

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

  // Load articles when filters or page changes
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
    error,
    currentPage,
    setCurrentPage,
    nextPage,
    prevPage,
    resetToFirstPage,
    loadArticles,
    updateArticle,
    updateArticles,
    totalPages: Math.ceil(totalArticles / RSS_CONFIG.ARTICLES_PER_PAGE),
    hasMore: currentPage * RSS_CONFIG.ARTICLES_PER_PAGE < totalArticles,
  };
}
