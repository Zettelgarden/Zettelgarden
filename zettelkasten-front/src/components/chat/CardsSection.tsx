import React, { useState } from 'react';
import { ChatCardData } from '../../models/Chat';
import { CardPreview } from './CardPreview';

interface CardsSectionProps {
  cards: ChatCardData[];
  onCardClick?: (cardId: string) => void;
}

export function CardsSection({ cards, onCardClick }: CardsSectionProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  if (cards.length === 0) return null;

  const toggleExpanded = () => {
    setIsExpanded(!isExpanded);
  };

  return (
    <div className="mt-4 p-3 bg-gray-50 border border-gray-200 rounded-lg">
      <button
        onClick={toggleExpanded}
        className="w-full flex items-center justify-between mb-2 hover:bg-gray-100 p-1 rounded transition-colors"
      >
        <div className="flex items-center gap-2">
          <svg className="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <h4 className="text-sm font-medium text-gray-800">
            {cards.length === 1 ? 'Referenced Card' : `Referenced Cards (${cards.length})`}
          </h4>
        </div>
        <svg
          className={`w-4 h-4 text-gray-500 transition-transform duration-200 ${isExpanded ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isExpanded && (
        <ul className="space-y-1 animate-fadeIn">
          {cards.map((card) => (
            <CardPreview
              key={card.id}
              card={card}
              onCardClick={onCardClick}
            />
          ))}
        </ul>
      )}
    </div>
  );
}