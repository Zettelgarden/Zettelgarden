import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Spreadsheet, createEmptySpreadsheet, SpreadsheetData } from '../../models/Spreadsheet';
import { SpreadsheetGrid } from './SpreadsheetGrid';

interface DynamicSpreadsheetProps {
  name: string;
  cardBody?: string;
  onBodyChange?: (newBody: string) => void;
}

// Regex to match spreadsheet code blocks in markdown
const SPREADSHEET_BLOCK_REGEX = /```spreadsheet:(\w+)\n([\s\S]*?)\n```/g;

/**
 * Parse spreadsheet data from a card body
 * Searches for ```spreadsheet:name``` code blocks and extracts the JSON data
 */
export function parseSpreadsheetFromBody(body: string, name: string): SpreadsheetData | null {
  const regex = new RegExp(`\`\`\`spreadsheet:${name}\\n([\\s\\S]*?)\\n\`\`\``);
  const match = body.match(regex);

  if (!match || !match[1]) {
    return null;
  }

  try {
    const data = JSON.parse(match[1].trim());
    // Validate the structure
    if (typeof data === 'object' && data !== null && 'rows' in data && 'cols' in data && 'data' in data) {
      return data as SpreadsheetData;
    }
  } catch (e) {
    console.warn(`Failed to parse spreadsheet data for ${name}:`, e);
  }

  return null;
}

/**
 * Serialize spreadsheet data and update the card body
 * If the spreadsheet block exists, it will be updated; otherwise, a new block will be appended
 */
export function serializeSpreadsheetToBody(body: string, name: string, data: SpreadsheetData): string {
  const blockRegex = new RegExp(`\`\`\`spreadsheet:${name}\\n[\\s\\S]*?\\n\`\`\``);
  const newBlock = `\`\`\`spreadsheet:${name}\n${JSON.stringify(data, null, 2)}\n\`\`\``;

  if (blockRegex.test(body)) {
    // Replace existing block
    return body.replace(blockRegex, newBlock);
  } else {
    // Append new block at the end
    return `${body.trim()}\n\n${newBlock}\n`;
  }
}

export function DynamicSpreadsheet({ name, cardBody = '', onBodyChange }: DynamicSpreadsheetProps) {
  const [spreadsheet, setSpreadsheet] = useState<Spreadsheet>(() =>
    createEmptySpreadsheet(name)
  );
  const [isLoading, setIsLoading] = useState(true);
  const saveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Load spreadsheet data from card body on mount
  useEffect(() => {
    const data = parseSpreadsheetFromBody(cardBody, name);
    if (data) {
      setSpreadsheet({
        name,
        data
      });
    } else {
      setSpreadsheet(createEmptySpreadsheet(name));
    }
    setIsLoading(false);
  }, [name, cardBody]);

  // Debounced save function
  const saveSpreadsheet = useCallback((updated: Spreadsheet) => {
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current);
    }

    saveTimeoutRef.current = setTimeout(() => {
      if (onBodyChange) {
        const newBody = serializeSpreadsheetToBody(cardBody, name, updated.data);
        onBodyChange(newBody);
      }
    }, 500); // 500ms debounce
  }, [cardBody, name, onBodyChange]);

  const handleChange = (updated: Spreadsheet) => {
    setSpreadsheet(updated);
    saveSpreadsheet(updated);
  };

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
    };
  }, []);

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
