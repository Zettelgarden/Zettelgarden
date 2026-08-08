import React from 'react';
import { Fact, FactWithCard } from '../../models/Fact';

interface FactsTabProps {
  facts: Fact[];
  factFilterString: string;
  setFactFilterString: (value: string) => void;
  setSelectedFact: (fact: FactWithCard | null) => void;
  setShowFactDialog: (show: boolean) => void;
}

export function FactsTab({
  facts,
  factFilterString,
  setFactFilterString,
  setSelectedFact,
  setShowFactDialog,
}: FactsTabProps) {
  return (
    <div className="p-4">
      <div className="mb-4">
        <input
          type="text"
          placeholder="Filter facts..."
          value={factFilterString}
          onChange={(e) => setFactFilterString(e.target.value)}
          className="w-full p-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>
      {facts && facts.length > 0 ? (
        <div className="max-h-[600px] overflow-y-auto border rounded-md p-2 pb-4">
          {facts
            .filter((fact) =>
              fact.fact.toLowerCase().includes(factFilterString.toLowerCase()),
            )
            .map((fact) => (
              <div
                key={fact.id}
                className="border-b pb-2 cursor-pointer hover:bg-gray-50"
                onClick={() => {
                  setSelectedFact(fact as FactWithCard);
                  setShowFactDialog(true);
                }}
              >
                <div className="text-sm text-gray-700">{fact.fact}</div>
              </div>
            ))}
        </div>
      ) : (
        <div className="text-gray-500">No facts available</div>
      )}
    </div>
  );
}
