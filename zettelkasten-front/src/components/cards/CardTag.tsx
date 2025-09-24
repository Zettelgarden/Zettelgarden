import React from "react";
import { PartialCard } from "../../models/Card";

interface CardTagProps {
  card: PartialCard;
  showTitle: boolean;
}

export function CardTag({ card, showTitle }: CardTagProps) {
  return (
    <div className="flex items-center min-w-0 max-w-full">
      <span className="text-blue-600 hover:text-blue-800 flex-shrink-0">
        [{card.card_id}]
      </span>
      {showTitle && (
        <span className="truncate ml-1">
          - {card.title}
        </span>
      )}
    </div>
  );
}
