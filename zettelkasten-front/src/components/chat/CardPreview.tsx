import React from 'react';
import { ChatCardData } from '../../models/Chat';

interface CardPreviewProps {
  card: ChatCardData;
  onCardClick?: (cardId: string) => void;
}

export function CardPreview({ card, onCardClick }: CardPreviewProps) {
  return (
    <div className="bg-white border border-gray-200 rounded-lg p-3 shadow-sm hover:shadow-md transition-shadow duration-200">
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <h3 className="font-medium text-gray-900 text-sm">
            <span className="text-blue-600 font-mono">[{card.card_id}]</span> - {card.title}
          </h3>
        </div>
        <button
          onClick={() => onCardClick?.(String(card.id))}
          className="ml-2 px-2 py-1 text-xs font-medium text-blue-700 bg-blue-50 border border-blue-200 rounded hover:bg-blue-100 hover:border-blue-300 transition-colors duration-200 flex items-center gap-1"
          title={`Open card: ${card.title}`}
        >
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
          Open
        </button>
      </div>
    </div>
  );
}