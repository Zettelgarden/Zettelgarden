import React from "react";
import { Link } from "react-router-dom";
import { ProcessedCardWithDescendants } from "../../../models/Card";

interface CardTreeItemProps {
  card: ProcessedCardWithDescendants;
  displayMode: 'compact' | 'full' | 'minimal';
  isSelected: boolean;
  onClick: () => void;
}

export function CardTreeItem({ card, displayMode, isSelected, onClick }: CardTreeItemProps) {
  const truncatedBody = card.body && card.body.length > 100
    ? card.body.substring(0, 100) + "..."
    : card.body || "";

  return (
    <div
      className={`flex items-start gap-2 py-1.5 px-2 hover:bg-gray-50 rounded transition-colors duration-150 ${isSelected ? 'bg-blue-50 border-l-4 border-l-blue-500' : ''}`}
      onClick={onClick}
    >
      {displayMode !== 'minimal' && (
        <div className="flex-grow min-w-0">
          <div className="font-medium text-gray-900 truncate">
            <Link
              to={`/app/card/${card.id}`}
              className="hover:text-blue-600"
              onClick={(e) => e.stopPropagation()}
            >
              {card.title || "Untitled Card"}
            </Link>
          </div>
          {displayMode === 'full' && card.body && (
            <div className="text-sm text-gray-600 mt-0.5 line-clamp-2">
              {truncatedBody}
            </div>
          )}
        </div>
      )}

      {displayMode === 'minimal' && (
        <Link
          to={`/app/card/${card.id}`}
          className="text-sm text-blue-600 font-mono hover:text-blue-800"
          onClick={(e) => e.stopPropagation()}
        >
          {card.card_id}
        </Link>
      )}

      <div className="text-xs text-gray-400 ml-2 flex-shrink-0">
        #{card.card_id}
      </div>
    </div>
  );
}