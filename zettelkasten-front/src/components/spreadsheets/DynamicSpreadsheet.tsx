import React, { useState, useEffect } from 'react';
import { Spreadsheet, createEmptySpreadsheet } from '../../models/Spreadsheet';
import { SpreadsheetGrid } from './SpreadsheetGrid';

interface DynamicSpreadsheetProps {
  name: string;
}

export function DynamicSpreadsheet({ name }: DynamicSpreadsheetProps) {
  const [spreadsheet, setSpreadsheet] = useState<Spreadsheet>(() =>
    createEmptySpreadsheet(name)
  );
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // TODO: Load spreadsheet data from card body
    // For now, we'll use the default empty spreadsheet
    setIsLoading(false);
  }, [name]);

  const handleChange = (updated: Spreadsheet) => {
    setSpreadsheet(updated);
    // TODO: Update the card body with new spreadsheet data
  };

  if (isLoading) {
    return <div className="p-4 text-gray-500">Loading spreadsheet...</div>;
  }

  return (
    <div className="my-4 border-l-4 border-green-500 pl-4">
      <div className="text-sm font-medium text-gray-700 mb-2">
        Spreadsheet: {name}
      </div>
      <SpreadsheetGrid
        spreadsheet={spreadsheet}
        onChange={handleChange}
        readOnly={false}
      />
    </div>
  );
}
