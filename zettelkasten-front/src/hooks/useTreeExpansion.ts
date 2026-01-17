import { useState, useCallback, SetStateAction } from "react";
import { CardWithDescendants } from "../models/Card";

export interface UseTreeExpansionResult {
  // State
  expandedCards: Set<string>;
  loadingCards: Set<string>;

  // Actions
  toggleExpansion: (cardId: string) => void;
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

  const reset = useCallback(() => {
    setExpandedCards(new Set());
    setLoadingCards(new Set());
  }, []);

  return {
    expandedCards,
    loadingCards,
    toggleExpansion,
    setExpandedCards,
    setLoadingCards,
    isExpanded,
    isLoading,
    reset,
  };
}