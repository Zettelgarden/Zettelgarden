import React from 'react';
import { PartialCard } from '../../models/Card';
import { CardList } from '../cards/CardList';

interface EntityCardsSectionProps {
  cards: PartialCard[];
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
        Associated Cards:
      </h4>
      <div className="min-h-[150px] max-h-[50vh] overflow-y-auto pr-2">
        {isLoading && <p>Loading cards...</p>}
        {error && <p className="text-red-600">{error}</p>}
        {!isLoading && !error && cards.length === 0 && (
          <p>No cards found for this entity.</p>
        )}
        {!isLoading && !error && cards.length > 0 && (
          <CardList cards={cards} showAddButton={false} />
        )}
      </div>
    </>
  );
}
