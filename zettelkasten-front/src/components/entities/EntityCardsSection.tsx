import React from 'react';
import { EntityCard } from '../../models/Card';
import { CardItem } from '../cards/CardItem';

interface EntityCardsSectionProps {
  cards: EntityCard[];
  isLoading: boolean;
  error: string | null;
}

export function EntityCardsSection({
  cards,
  isLoading,
  error,
}: EntityCardsSectionProps) {
  return (
    <>
      <h4 className="text-md font-medium text-gray-800 mb-2 border-t pt-3">
        Associated Cards ({cards.length})
      </h4>
      <div className="min-h-[150px] max-h-[50vh] overflow-y-auto pr-2">
        {isLoading && <p>Loading cards...</p>}
        {error && <p className="text-red-600">{error}</p>}
        {!isLoading && !error && cards.length === 0 && (
          <p>No cards found for this entity.</p>
        )}
        {!isLoading && !error && cards.length > 0 && (
          <ul className="space-y-1">
            {cards.map((ec) => (
              <li key={ec.card.id} className="flex items-center gap-2">
                <div className="flex-grow min-w-0">
                  <CardItem card={ec.card} />
                </div>
                <span
                  className="shrink-0 text-[10px] text-gray-500 bg-gray-100 rounded-full px-2 py-0.5"
                  title={`${ec.entity_count} entit${
                    ec.entity_count === 1 ? 'y' : 'ies'
                  } on this card`}
                >
                  {ec.entity_count} ent
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}
