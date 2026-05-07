import React from "react";
import { PartialCard } from "../../models/Card";
import { Link } from "react-router-dom";
import { CardTag } from "./CardTag";

interface CardLinkProps {
  card: PartialCard;
  handleViewBacklink: (id: number) => void;
  showTitle: boolean;
  displayText?: string;
  showTags?: boolean;
  onRemoveTag?: (tagName: string) => void;
  showTagRemoval?: boolean;
}

export function CardLink({ card, showTitle, displayText, showTags = false, onRemoveTag, showTagRemoval = false }: CardLinkProps) {
  return (
    <Link to={`/app/card/${card.id}`} className="flex items-center gap-2 min-w-0 overflow-hidden">
      <span className="inline-flex items-center flex-shrink min-w-0">
        <CardTag card={card} showTitle={showTitle} displayText={displayText} />
      </span>
      {/* Display tags */}
      {showTags && card.tags && card.tags.length > 0 && (
        <div className="flex flex-wrap gap-1 flex-shrink min-w-0">
          {card.tags.map((tag, index) => (
            <span
              key={index}
              className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
            >
              <span>#{tag.name}</span>
              {showTagRemoval && onRemoveTag && (
                <button
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    onRemoveTag(tag.name);
                  }}
                  className="ml-1.5 text-purple-400 hover:text-purple-600 min-w-[44px] md:min-w-[24px] min-h-[44px] md:min-h-[24px] flex items-center justify-center"
                >
                  &times;
                </button>
              )}
            </span>
          ))}
        </div>
      )}
    </Link>
  );
}
