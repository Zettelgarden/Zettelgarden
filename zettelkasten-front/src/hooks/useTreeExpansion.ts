import { useState, useCallback, SetStateAction } from "react";
import { ProcessedCardWithDescendants } from "../models/Card";

export interface UseTreeExpansionResult {
  // State
  expandedCards: Set<string>;
  loadingCards: Set<string>;

  // Actions
  toggleExpansion: (cardId: string) => void;
  loadChildren: (cardId: string, children: ProcessedCardWithDescendants[]) => void;
  setExpandedCards: React.Dispatch<SetStateAction<Set<string>>>;
  setLoadingCards: React.Dispatch<SetStateAction<Set<string>>>;

  // Helpers
  isExpanded: (cardId: string) => boolean;
  isLoading: (cardId: string) => boolean;
  reset: () => void;
}

export function useTreeExpansion(): UseTreeExpansionResult {
  const [expandedCards, setExpandedCards] = useState<Set<string>>(new Set());
  const [loadingCards, setLoadingCards] = useState<Set<string>>(new Set());

  const isExpanded = useCallback((cardId: string) => expandedCards.has(cardId), [expandedCards]);
  const isLoading = useCallback((cardId: string) => loadingCards.has(cardId), [loadingCards]);

  const toggleExpansion = useCallback((cardId: string) => {
    const isCurrentlyExpanded = expandedCards.has(cardId);

    if (isCurrentlyExpanded) {
      // Collapse: remove from expanded set
      setExpandedCards(prev => {
        const next = new Set(prev);
        next.delete(cardId);
        return next;
      });
    } else {
      // Expand: add to expanded set
      // Note: In the current implementation, children should already be loaded
      // since getCardWithDescendants retrieves the full tree structure
      setExpandedCards(prev => new Set([...prev, cardId]));

      // If lazy loading were implemented in the future, this is where
      // we'd check if children need to be loaded and mark as loading
    }
  }, [expandedCards]);

  const loadChildren = useCallback((cardId: string, children: ProcessedCardWithDescendants[]) => {
    // This function would typically be called from the CardTreeNode
    // when lazy loading is triggered, but in our current implementation
    // we preload everything, so it's a no-op for now
    // Future: integrate with lazy loading by fetching children on demand
  }, []);

  const reset = useCallback(() => {
    setExpandedCards(new Set());
    setLoadingCards(new Set());
  }, []);

  return {
    expandedCards,
    loadingCards,
    toggleExpansion,
    loadChildren,
    setExpandedCards,
    setLoadingCards,
    isExpanded,
    isLoading,
    reset,
  };
}