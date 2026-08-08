import React from 'react';
import { FactWithCard } from '../../models/Fact';
import { CardTag } from '../cards/CardTag';

interface EntityFactsSectionProps {
  facts: FactWithCard[];
  isLoading: boolean;
  error: string | null;
  onFactClick: (fact: FactWithCard) => void;
}

export function EntityFactsSection({
  facts,
  isLoading,
  error,
  onFactClick,
}: EntityFactsSectionProps) {
  return (
    <>
      <h4 className="text-md font-medium text-gray-800 mt-4 border-t pt-3">
        Facts:
      </h4>
      <div className="min-h-[100px] max-h-[30vh] overflow-y-auto pr-2">
        {isLoading && <p>Loading facts...</p>}
        {error && <p className="text-red-600">{error}</p>}
        {!isLoading && !error && facts.length === 0 && (
          <p>No facts linked to this entity.</p>
        )}
        {!isLoading && !error && facts.length > 0 && (
          <ul className="space-y-2">
            {facts.map((f) => (
              <li
                key={f.id}
                onClick={() => onFactClick(f)}
                className="cursor-pointer hover:bg-gray-100 p-1 rounded"
              >
                <p className="text-sm text-gray-700">• {f.fact}</p>
                {f.card && (
                  <span className="text-xs text-blue-600">
                    <CardTag card={f.card} showTitle={true} />
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}
