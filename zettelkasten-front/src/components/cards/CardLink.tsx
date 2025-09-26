import React from "react";
import { PartialCard } from "../../models/Card";
import { Link } from "react-router-dom";
import { CardTag } from "./CardTag";

interface CardLinkProps {
  card: PartialCard;
  handleViewBacklink: (id: number) => void;
  showTitle: boolean;
  showTags?: boolean;
}

export function CardLink({ card, showTitle, showTags = false }: CardLinkProps) {
  return (
    <Link to={`/app/card/${card.id}`} className="flex items-center gap-2 min-w-0 overflow-hidden">
      <div className="flex-shrink min-w-0">
        <CardTag card={card} showTitle={showTitle} />
      </div>
      {/* Display tags */}
      {showTags && card.tags && card.tags.length > 0 && (
        <div className="flex flex-wrap gap-1 flex-shrink min-w-0">
          {card.tags.map((tag, index) => (
            <span
              key={index}
              className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
            >
              {tag.name}
            </span>
          ))}
        </div>
      )}
    </Link>
  );
}
