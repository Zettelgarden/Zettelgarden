import React from 'react';
import { HeaderSubSection } from '../Header';
import { CardItem } from './CardItem';
import { RelatedCard } from '../../models/Card';

interface RelatedCardsProps {
  relatedCards: RelatedCard[];
  onCardClick: (cardId: number) => void;
  onAddReference?: (card: RelatedCard) => void;
}

export function RelatedCards({
  relatedCards,
  onCardClick,
  onAddReference,
}: RelatedCardsProps) {
  if (relatedCards.length === 0) {
    return null;
  }

  return (
    <div>
      <HeaderSubSection text="Related Cards" />
      <ul className="mt-2 space-y-1">
        {relatedCards.map((rc) => (
          <li key={rc.card.id} className="cursor-pointer group">
            <div className="flex items-center justify-between">
              <div className="flex-1" onClick={() => onCardClick(rc.card.id)}>
                <CardItem card={rc.card} />
              </div>
              <div className="flex items-center gap-1 ml-2 shrink-0">
                {onAddReference && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onAddReference(rc);
                    }}
                    className="opacity-0 group-hover:opacity-100 text-xs text-blue-500 hover:text-blue-700 border border-blue-300 hover:border-blue-500 rounded px-1.5 py-0.5 transition-opacity"
                    title="Add as reference"
                  >
                    +Ref
                  </button>
                )}
                {rc.score > 0 && (
                  <span className="text-xs text-gray-400">
                    {rc.score.toFixed(1)}
                  </span>
                )}
              </div>
            </div>
            {rc.reasons && rc.reasons.length > 0 && (
              <div
                className="mt-1 flex flex-wrap gap-1 px-2"
                onClick={() => onCardClick(rc.card.id)}
              >
                {rc.reasons.map((reason) => (
                  <span
                    key={reason}
                    className="inline-flex items-center rounded-full bg-blue-50 text-blue-700 text-[10px] leading-4 px-2 py-0.5 border border-blue-100 max-w-full truncate"
                    title={reason}
                  >
                    {reason}
                  </span>
                ))}
              </div>
            )}
          </li>
        ))}
      </ul>
      <hr className="my-4" />
    </div>
  );
}
