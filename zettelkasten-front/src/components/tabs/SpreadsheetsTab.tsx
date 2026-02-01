import React, { useState, useMemo } from 'react';
import { Card } from '../../models/Card';
import { Button } from '../Button';
import { SpreadsheetGrid } from '../spreadsheets/SpreadsheetGrid';
import { Spreadsheet, createEmptySpreadsheet } from '../../models/Spreadsheet';

interface SpreadsheetsTabProps {
  viewingCard: Card;
  setViewCard: (card: Card) => void;
  setError: (error: string) => void;
}

// Extract spreadsheet names from card body using regex
function extractSpreadsheetNames(body: string): string[] {
  const regex = /\{\{spreadsheet(?::([^}\s]+))?\}\}/gi;
  const names = new Set<string>();
  let match;

  while ((match = regex.exec(body)) !== null) {
    const name = match[1] || 'sheet1';
    names.add(name);
  }

  return Array.from(names);
}

export function SpreadsheetsTab({ viewingCard, setViewCard, setError }: SpreadsheetsTabProps) {
  const [selectedSpreadsheet, setSelectedSpreadsheet] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);

  const spreadsheetNames = useMemo(() =>
    extractSpreadsheetNames(viewingCard.body),
    [viewingCard.body]
  );

  const handleCreateSpreadsheet = async () => {
    try {
      setIsCreating(true);

      // Generate a unique name
      const newName = `sheet${spreadsheetNames.length + 1}`;

      // Create empty spreadsheet data
      const newSpreadsheet = createEmptySpreadsheet(newName);

      // Insert the markdown syntax into the card body
      const updatedBody = `${viewingCard.body}\n\n{{spreadsheet:${newName}}}\n\n\`\`\`spreadsheet:${newName}\n${JSON.stringify(newSpreadsheet.data, null, 2)}\n\`\`\`\n`;

      setViewCard({
        ...viewingCard,
        body: updatedBody
      });

      setSelectedSpreadsheet(newName);
    } catch (err) {
      setError('Failed to create spreadsheet');
    } finally {
      setIsCreating(false);
    }
  };

  if (selectedSpreadsheet) {
    // Show the spreadsheet grid
    const spreadsheet = createEmptySpreadsheet(selectedSpreadsheet);

    return (
      <div>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium">{selectedSpreadsheet}</h3>
          <Button
            onClick={() => setSelectedSpreadsheet(null)}
            variant="outline"
            size="small"
          >
            Back to List
          </Button>
        </div>
        <SpreadsheetGrid
          spreadsheet={spreadsheet}
          onChange={() => {}}
          readOnly={false}
        />
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-medium">
          Spreadsheets ({spreadsheetNames.length})
        </h3>
        <Button
          onClick={handleCreateSpreadsheet}
          variant="primary"
          size="small"
          disabled={isCreating}
        >
          {isCreating ? 'Creating...' : 'Add Spreadsheet'}
        </Button>
      </div>

      {spreadsheetNames.length === 0 ? (
        <div className="text-center py-8 text-gray-500">
          No spreadsheets found in this card.
          Click "Add Spreadsheet" to create one.
        </div>
      ) : (
        <div className="space-y-2">
          {spreadsheetNames.map((name) => (
            <div
              key={name}
              onClick={() => setSelectedSpreadsheet(name)}
              className="p-3 border border-gray-200 rounded hover:bg-gray-50 cursor-pointer"
            >
              <div className="font-medium text-gray-900">{name}</div>
              <div className="text-sm text-gray-500">Click to edit</div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
