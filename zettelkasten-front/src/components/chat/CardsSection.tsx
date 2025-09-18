import React from 'react';
import { ChatCardData } from '../../models/Chat';
import { CardPreview } from './CardPreview';

interface CardsSectionProps {
  cards: ChatCardData[];
  onCardClick?: (cardId: string) => void;
}

export function CardsSection({ cards, onCardClick }: CardsSectionProps) {
  if (cards.length === 0) return null;

  return (
    <div className="mt-4 p-4 bg-gradient-to-br from-blue-50 to-indigo-50 border border-blue-200 rounded-lg">
      <div className="flex items-center gap-2 mb-3">
        <svg className="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
        <h4 className="text-sm font-semibold text-blue-900">
          {cards.length === 1 ? 'Referenced Card' : `Referenced Cards (${cards.length})`}
        </h4>
      </div>
      <div className="grid gap-3">
        {cards.map((card) => (
          <CardPreview
            key={card.id}
            card={card}
            onCardClick={onCardClick}
          />
        ))}
      </div>
    </div>
  );
}