import React from "react";
import { Link } from "react-router-dom";
import { ProcessedCardWithDescendants } from "../../../models/Card";

interface CardTreeItemProps {
  card: ProcessedCardWithDescendants;
  displayMode: 'compact' | 'full' | 'minimal';
  isSelected: boolean;
  onClick: () => void;
}

// Smart truncation utility for card body content
const truncateBody = (body: string, maxLength: number, mode: string): string => {
  if (!body || body.length <= maxLength) {
    return body || "";
  }

  // Try to truncate at word boundary if possible
  let truncated = body.substring(0, maxLength);
  const lastSpace = truncated.lastIndexOf(' ');

  // If we found a space and it's close to our limit, truncate there
  if (lastSpace > maxLength * 0.7) {
    truncated = body.substring(0, lastSpace);
  }

  // For minimal mode, prefer complete sentences if reasonably short
  if (mode === 'minimal') {
    const firstSentence = body.split(/[.!?]+/)[0];
    if (firstSentence && firstSentence.length <= maxLength + 10) {
      return firstSentence + (body.length > firstSentence.length ? '.' : '');
    }
  }

  return truncated + "...";
};

export function CardTreeItem({ card, displayMode, isSelected, onClick }: CardTreeItemProps) {
  // Configure truncation based on display mode
  const getTruncationConfig = (mode: string): { maxLength: number } => {
    switch (mode) {
      case 'minimal':
        return { maxLength: 30 };
      case 'compact':
        return { maxLength: 60 };
      case 'full':
        return { maxLength: 1000 };
      default:
        return { maxLength: 60 };
    }
  };

  const { maxLength } = getTruncationConfig(displayMode);
  const truncatedBody = truncateBody(card.body || "", maxLength, displayMode);

  return (
    <div
      className={`flex flex-1 items-start gap-2 py-1.5 px-2 hover:bg-gray-50 rounded transition-colors duration-150 ${isSelected ? 'bg-blue-50 border-l-4 border-l-blue-500' : ''}`}
      onClick={onClick}
    >
      {displayMode !== 'minimal' && (
        <div className="flex-1 min-w-0">
          <div className="font-medium text-gray-900 truncate">
            <Link
              to={`/app/card/${card.id}`}
              className="hover:text-blue-600"
              onClick={(e) => e.stopPropagation()}
            >
              {card.title || "Untitled Card"}
            </Link>
          </div>
          {card.body && (
            <div className={`text-gray-600 mt-0.5 line-clamp-2 ${
              displayMode === 'full' ? 'text-sm' : 'text-xs'
            }`}>
              {truncatedBody}
            </div>
          )}
        </div>
      )}

      {displayMode === 'minimal' && (
        <div className="flex-1 min-w-0">
          <Link
            to={`/app/card/${card.id}`}
            className="text-sm text-blue-600 font-mono hover:text-blue-800 block"
            onClick={(e) => e.stopPropagation()}
          >
            {card.card_id}
          </Link>
          {card.body && (
            <div className="text-xs text-gray-500 mt-0.5 truncate">
              {truncatedBody}
            </div>
          )}
        </div>
      )}

      <div className="text-xs text-gray-400 ml-2 flex-shrink-0 whitespace-nowrap">
        #{card.card_id}
      </div>
    </div>
  );
}