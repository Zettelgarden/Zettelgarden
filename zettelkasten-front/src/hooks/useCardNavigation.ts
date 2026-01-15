import { useState, useEffect } from "react";
import { Card, PartialCard } from "../models/Card";
import { compareCardIds } from "../utils/cards";

interface UseCardNavigationResult {
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
}

/**
 * Custom hook for managing card sibling navigation logic.
 * Takes a parent card and viewing card to calculate the previous and next siblings
 * from the parent's sorted children array.
 */
export function useCardNavigation(
  parentCard: Card | null,
  viewingCard: Card | null
): UseCardNavigationResult {
  const [prevSibling, setPrevSibling] = useState<PartialCard | null>(null);
  const [nextSibling, setNextSibling] = useState<PartialCard | null>(null);

  useEffect(() => {
    if (parentCard && viewingCard) {
      const siblings = parentCard.children.sort((a, b) =>
        compareCardIds(a.card_id, b.card_id)
      );
      const currentIndex = siblings.findIndex(s => s.id === viewingCard.id);

      if (currentIndex !== -1) {
        setPrevSibling(currentIndex > 0 ? siblings[currentIndex - 1] : null);
        setNextSibling(currentIndex < siblings.length - 1 ? siblings[currentIndex + 1] : null);
      } else {
        setPrevSibling(null);
        setNextSibling(null);
      }
    } else {
      setPrevSibling(null);
      setNextSibling(null);
    }
  }, [parentCard, viewingCard]);

  return {
    prevSibling,
    nextSibling,
  };
}