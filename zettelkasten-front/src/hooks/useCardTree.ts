import { useState, useCallback } from "react";
import { ProcessedCardWithDescendants } from "../models/Card";
import { getCardWithDescendants } from "../api/cards";

export interface UseCardTreeResult {
  // Data state
  tree: ProcessedCardWithDescendants | null;
  isLoading: boolean;
  error: string | null;

  // Actions
  fetchTree: (cardId: string | number) => Promise<void>;
  setTree: (tree: ProcessedCardWithDescendants | null) => void;
  clearTree: () => void;
}

export function useCardTree(): UseCardTreeResult {
  const [tree, setTree] = useState<ProcessedCardWithDescendants | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTree = useCallback(async (cardId: string | number) => {
    setIsLoading(true);
    setError(null);

    try {
      const treeData = await getCardWithDescendants(cardId);
      setTree(treeData);
    } catch (err: any) {
      console.error("Failed to fetch card tree:", err);
      setError(err.message || "Failed to load card tree");
      setTree(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const clearTree = useCallback(() => {
    setTree(null);
    setError(null);
    setIsLoading(false);
  }, []);

  return {
    tree,
    isLoading,
    error,
    fetchTree,
    setTree,
    clearTree,
  };
}