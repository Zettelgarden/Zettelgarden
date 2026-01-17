import React, { useState, useEffect, useCallback } from "react";
import { ProcessedCardWithDescendants } from "../../../models/Card";
import { CardTreeNode } from "./CardTreeNode";
import { useCardTree } from "../../../hooks/useCardTree";
import { useTreeExpansion } from "../../../hooks/useTreeExpansion";

interface CardTreeViewProps {
  rootCardId: string | number;
  maxDepth?: number; // Limit rendering depth for performance
  displayMode?: 'compact' | 'full' | 'minimal';
  onCardSelect?: (card: ProcessedCardWithDescendants) => void;
  onCardUpdate?: () => void;
  className?: string;
  height?: string; // For scrollable containers
}

export function CardTreeView({
  rootCardId,
  maxDepth,
  displayMode = 'compact',
  onCardSelect,
  onCardUpdate,
  className,
  height = '400px'
}: CardTreeViewProps) {
  const { tree, isLoading: treeLoading, error, fetchTree } = useCardTree();
  const { expandedCards, loadingCards, toggleExpansion, isExpanded, isLoading } = useTreeExpansion();

  const [selectedCardId, setSelectedCardId] = useState<string | null>(null);

  // Load tree data when rootCardId changes
  useEffect(() => {
    if (rootCardId) {
      fetchTree(rootCardId);
      // Reset selection when loading a new tree
      setSelectedCardId(null);
    }
  }, [rootCardId, fetchTree]);

  // Handle card selection
  const handleSelectCard = useCallback((card: ProcessedCardWithDescendants) => {
    setSelectedCardId(card.card_id);
    onCardSelect?.(card);
  }, [onCardSelect]);

  // Loading state
  if (treeLoading && !tree) {
    return (
      <div
        className={`flex items-center justify-center p-4 ${className || ''}`}
        style={{ height }}
      >
        <div className="text-gray-500">Loading tree...</div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div
        className={`flex items-center justify-center p-4 ${className || ''}`}
        style={{ height }}
      >
        <div className="text-red-500">
          Error loading tree: {error}
        </div>
      </div>
    );
  }

  // No tree data
  if (!tree) {
    return (
      <div
        className={`flex items-center justify-center p-4 ${className || ''}`}
        style={{ height }}
      >
        <div className="text-gray-500">No tree data available</div>
      </div>
    );
  }


  return (
    <div
      className={`card-tree-view overflow-y-auto ${className || ''}`}
      style={{ height }}
    >
      <CardTreeNode
        card={tree}
        depth={0}
        maxDepth={maxDepth}
        displayMode={displayMode}
        selectedCardId={selectedCardId}
        onSelectCard={handleSelectCard}
        isExpanded={isExpanded}
        isLoading={isLoading}
        onToggleExpansion={toggleExpansion}
      />
    </div>
  );
}