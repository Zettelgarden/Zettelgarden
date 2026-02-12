import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Spreadsheet, createEmptySpreadsheet, SpreadsheetData } from '../../models/Spreadsheet';
import { SpreadsheetGrid } from './SpreadsheetGrid';
import { updateSpreadsheet } from '../../api/spreadsheets';

interface DynamicSpreadsheetProps {
  /** Database ID of the spreadsheet */
  id: number;
  /** Initial spreadsheet data (loaded from API) */
  initialData: SpreadsheetData;
  /** Whether the spreadsheet is read-only */
  readOnly?: boolean;
}

// Regex to match spreadsheet code blocks in markdown
const SPREADSHEET_BLOCK_REGEX = /```spreadsheet:(\w+)\n([\s\S]*?)\n```/g;

/**
 * Parse spreadsheet data from a card body
 * Searches for ```spreadsheet:name``` code blocks and extracts the JSON data
 * @deprecated This function is deprecated. Spreadsheets are now stored in the database,
 * not embedded in markdown. Use the API functions instead.
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
 * @deprecated This function is deprecated. Spreadsheets are now stored in the database,
 * not embedded in markdown. Use the API functions instead.
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

export function DynamicSpreadsheet({ id, initialData, readOnly = false }: DynamicSpreadsheetProps) {
  // State management for spreadsheet data
  const [spreadsheetData, setSpreadsheetData] = useState<SpreadsheetData>(initialData);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const saveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingUpdateRef = useRef<SpreadsheetData | null>(null);

  // Update local state when initialData changes (e.g., if parent reloads data)
  useEffect(() => {
    setSpreadsheetData(initialData);
  }, [id, initialData]);

  // Debounced save function using API
  const saveSpreadsheet = useCallback(async (data: SpreadsheetData) => {
    // Clear any existing timeout
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current);
    }

    // Store the pending update
    pendingUpdateRef.current = data;

    // Debounce with 500ms delay
    saveTimeoutRef.current = setTimeout(async () => {
      if (!pendingUpdateRef.current) return;

      setIsSaving(true);
      setSaveError(null);

      try {
        // Save via API - this updates the database directly
        await updateSpreadsheet(id, pendingUpdateRef.current);
        pendingUpdateRef.current = null;
      } catch (error) {
        console.error('Failed to save spreadsheet:', error);
        setSaveError('Failed to save changes');
      } finally {
        setIsSaving(false);
      }
    }, 500);
  }, [id]);

  const handleChange = useCallback((updated: Spreadsheet) => {
    // Update local state immediately for responsive UI
    setSpreadsheetData(updated.data);
    // Trigger debounced save
    saveSpreadsheet(updated.data);
  }, [saveSpreadsheet]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
    };
  }, []);

  // Build a minimal Spreadsheet object for SpreadsheetGrid
  const spreadsheetForGrid: Spreadsheet = {
    id,
    user_id: 0, // Not used by SpreadsheetGrid
    card_id: 0, // Not used by SpreadsheetGrid
    name: '',   // Not used by SpreadsheetGrid
    data: spreadsheetData,
    created_at: new Date(),
    updated_at: new Date()
  };

  return (
    <div className="my-4 border-l-4 border-green-500 pl-4">
      <div className="flex items-center justify-between mb-2">
        <div className="text-sm font-medium text-gray-700">
          Spreadsheet
        </div>
        {(isSaving || saveError) && (
          <div className={`text-xs ${saveError ? 'text-red-500' : 'text-gray-500'}`}>
            {saveError || 'Saving...'}
          </div>
        )}
      </div>
      <SpreadsheetGrid
        spreadsheet={spreadsheetForGrid}
        onChange={handleChange}
        readOnly={readOnly}
      />
    </div>
  );
}
