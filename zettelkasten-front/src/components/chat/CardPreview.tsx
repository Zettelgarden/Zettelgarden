import React from 'react';
import { ChatCardData } from '../../models/Chat';
import { CardIcon } from '../../assets/icons/CardIcon';

interface CardPreviewProps {
  card: ChatCardData;
  onCardClick?: (cardId: string) => void;
}

export function CardPreview({ card, onCardClick }: CardPreviewProps) {
  return (
    <li className="py-1 px-2 hover:bg-gray-50 rounded-lg">
      <div className="flex items-center gap-2">
        <div className="text-gray-400">
          <CardIcon />
        </div>
        <div className="flex-grow">
          <div className="flex flex-col">
            <div className="flex items-center flex-wrap gap-1">
              <button
                onClick={() => onCardClick?.(String(card.id))}
                className="hover:underline flex-shrink-0"
                title={`Open card: ${card.title}`}
              >
                <span className="text-blue-600 hover:text-blue-800">[{card.card_id}]</span>
                <span className="mx-2 text-gray-400">-</span>
                <span>{card.title}</span>
              </button>
            </div>
            {card.body_preview && (
              <div className="mt-0.5 pl-2 text-sm italic text-gray-600">
                {card.body_preview.length > 200
                  ? `${card.body_preview.substring(0, 200)}...`
                  : card.body_preview}
              </div>
            )}
          </div>
        </div>
        <div className="flex flex-col items-end text-xs text-gray-500">
          <div>{new Date(card.updated_at).toLocaleDateString()}</div>
        </div>
      </div>
    </li>
  );
}