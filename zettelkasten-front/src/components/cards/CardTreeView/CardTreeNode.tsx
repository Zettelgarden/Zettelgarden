import React, { memo } from "react";
import { ProcessedCardWithDescendants } from "../../../models/Card";
import { CardTreeItem } from "./CardTreeItem";
import { TriangleRightIcon } from "../../../assets/icons/TriangleRight";
import { TriangleDownIcon } from "../../../assets/icons/TriangleDown";

interface CardTreeNodeProps {
  card: ProcessedCardWithDescendants;
  depth: number;
  maxDepth?: number;
  displayMode: 'compact' | 'full' | 'minimal';
  selectedCardId: string | null;
  onSelectCard: (card: ProcessedCardWithDescendants) => void;
  onToggleExpansion: (cardId: string) => void;
  isExpanded: (cardId: string) => boolean;
  isLoading: (cardId: string) => boolean;
}

const CardTreeNode = memo(function CardTreeNode({
  card,
  depth,
  maxDepth,
  displayMode,
  selectedCardId,
  onSelectCard,
  onToggleExpansion,
  isExpanded,
  isLoading
}: CardTreeNodeProps) {
  console.log(`CardTreeNode rendering: ${card.title} (${card.card_id}), depth ${depth}, hasDescendants: ${card.descendants?.length || 0}`);
  const hasChildren = card.descendants && card.descendants.length > 0;
  const nodeIsExpanded = depth === 0 ? true : isExpanded(card.card_id); // Root is always expanded
  const nodeIsLoading = isLoading(card.card_id);
  const shouldRenderChildren = hasChildren && nodeIsExpanded;
  const isSelected = card.card_id === selectedCardId;
  console.log(`Setting shouldRenderChildren: ${shouldRenderChildren} (hasChildren: ${hasChildren}, nodeIsExpanded: ${nodeIsExpanded})`);

  // Calculate depth-based styling
  const depthPadding = 6 * depth; // 6px per depth level, starting from 0
  const depthBorderWidth = Math.min(depth + 1, 4); // Max 4px border

  // Only render if we haven't exceeded maxDepth (if specified)
  if (maxDepth !== undefined && depth > maxDepth) {
    return null;
  }

  return (
    <div className="tree-node-container">
      {/* Main node item */}
      <div
        className={`flex items-start gap-2 py-1.5 px-2 hover:bg-gray-50 rounded transition-colors duration-150 ${isSelected ? 'bg-blue-50 border-l-4 border-l-blue-500' : ''}`}
        style={{
          paddingLeft: `${depthPadding}px`,
          ...(depth > 0 && {
            borderLeftWidth: `${depthBorderWidth}px`,
            borderLeftStyle: 'solid',
            borderLeftColor: depth <= 1 ? '#e5e7eb' : depth <= 2 ? '#d1d5db' : '#9ca3af',
          }),
        }}
      >
        {/* Expansion toggle icon */}
        <div
          className={`flex-shrink-0 w-4 h-4 flex items-center justify-center transition-transform duration-200 ${depth === 0 ? 'cursor-default' : 'cursor-pointer'}`}
          onClick={(e) => {
            e.stopPropagation();
            if (depth > 0 && hasChildren) {
              onToggleExpansion(card.card_id);
            }
          }}
        >
          {depth > 0 && hasChildren ? (
            nodeIsExpanded ? (
              <span className="text-gray-500">
                <TriangleDownIcon />
              </span>
            ) : (
              <span className="text-gray-500">
                <TriangleRightIcon />
              </span>
            )
          ) : null}
        </div>

        {/* Card content */}
        <CardTreeItem
          card={card}
          displayMode={displayMode}
          isSelected={isSelected}
          onClick={() => onSelectCard(card)}
        />
      </div>

      {/* Child nodes */}
      {shouldRenderChildren && (
        <>
          {console.log(`Rendering ${card.descendants.length} children for ${card.card_id}`)}
          <div className="ml-2">
            {nodeIsLoading && (
              <div className="text-gray-500 text-sm py-1 pl-1">
                Loading...
              </div>
            )}
            {!nodeIsLoading && card.descendants.map((child) => (
            <CardTreeNode
              key={child.card_id}
              card={child}
              depth={depth + 1}
              maxDepth={maxDepth}
              displayMode={displayMode}
              selectedCardId={selectedCardId}
              onSelectCard={onSelectCard}
              isExpanded={isExpanded}
              isLoading={isLoading}
              onToggleExpansion={onToggleExpansion}
            />
          ))}
          </div>
        </>
      )}
    </div>
  );
});

export { CardTreeNode };