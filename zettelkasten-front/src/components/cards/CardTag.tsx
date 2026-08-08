import React from 'react';
import { PartialCard } from '../../models/Card';

interface CardTagProps {
  card: PartialCard;
  showTitle: boolean;
  displayText?: string;
}

export function CardTag({ card, showTitle, displayText }: CardTagProps) {
  const inner = displayText ? `${card.card_id} - ${displayText}` : card.card_id;
  return (
    <div className="flex items-center min-w-0 max-w-full text-sm">
      <span className="text-blue-600 hover:text-blue-800 flex-shrink-0">
        [{inner}]
      </span>
      {showTitle && (
        <span className="truncate ml-1 text-gray-700">- {card.title}</span>
      )}
    </div>
  );
}
