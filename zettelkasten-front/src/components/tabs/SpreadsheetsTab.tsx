import React, { useState, useEffect, useCallback } from 'react';
import { Card } from '../../models/Card';
import { Button } from '../Button';
import { SpreadsheetGrid } from '../spreadsheets/SpreadsheetGrid';
import { Spreadsheet } from '../../models/Spreadsheet';
import { fetchSpreadsheets, createSpreadsheet, deleteSpreadsheet, updateSpreadsheet } from '../../api/spreadsheets';

interface SpreadsheetsTabProps {
  viewingCard: Card;
  setViewCard: (card: Card) => void;
  setError: (error: string) => void;
}

export function SpreadsheetsTab({ viewingCard, setViewCard, setError }: SpreadsheetsTabProps) {
  const [spreadsheets, setSpreadsheets] = useState<Spreadsheet[]>([]);
  const [selectedSpreadsheet, setSelectedSpreadsheet] = useState<Spreadsheet | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [spreadsheetToDelete, setSpreadsheetToDelete] = useState<Spreadsheet | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  // Load spreadsheets from the database
  const loadSpreadsheets = useCallback(async () => {
    try {
      setIsLoading(true);
      const data = await fetchSpreadsheets(viewingCard.id);
      setSpreadsheets(data);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Failed to load spreadsheets');
      }
    } finally {
      setIsLoading(false);
    }
  }, [viewingCard.id, setError]);

  // Load spreadsheets on mount
  useEffect(() => {
    loadSpreadsheets();
  }, [loadSpreadsheets]);

  // Create a new spreadsheet
  const handleCreateSpreadsheet = async () => {
    try {
      setIsCreating(true);

      // Generate a unique name
      const existingNames = new Set(spreadsheets.map(s => s.name));
      let newName = 'sheet1';
      let counter = 1;
      while (existingNames.has(newName)) {
        counter++;
        newName = `sheet${counter}`;
      }

      const newSpreadsheet = await createSpreadsheet(viewingCard.id, newName);
      setSpreadsheets(prev => [...prev, newSpreadsheet]);
      setSelectedSpreadsheet(newSpreadsheet);

      // Append spreadsheet reference to card body
      const updatedCard = {
        ...viewingCard,
        body: viewingCard.body.trim() + `\n\n{{spreadsheet:${newSpreadsheet.id}}}\n`
      };
      setViewCard(updatedCard);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Failed to create spreadsheet');
      }
    } finally {
      setIsCreating(false);
    }
  };

  // Handle spreadsheet data changes
  const handleSpreadsheetChange = async (spreadsheet: Spreadsheet) => {
    try {
      setIsSaving(true);
      const updated = await updateSpreadsheet(spreadsheet.id, spreadsheet.data);
      setSelectedSpreadsheet(updated);
      // Update the spreadsheet in the list as well
      setSpreadsheets(prev =>
        prev.map(s => s.id === updated.id ? updated : s)
      );
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Failed to save spreadsheet');
      }
    } finally {
      setIsSaving(false);
    }
  };

  // Initiate delete flow
  const handleDeleteClick = (spreadsheet: Spreadsheet, e: React.MouseEvent) => {
    e.stopPropagation();
    setSpreadsheetToDelete(spreadsheet);
    setShowDeleteConfirm(true);
  };

  // Confirm delete
  const handleConfirmDelete = async () => {
    if (!spreadsheetToDelete) return;

    try {
      setIsDeleting(true);
      await deleteSpreadsheet(spreadsheetToDelete.id);
      setSpreadsheets(prev => prev.filter(s => s.id !== spreadsheetToDelete.id));
      if (selectedSpreadsheet?.id === spreadsheetToDelete.id) {
        setSelectedSpreadsheet(null);
      }
      setShowDeleteConfirm(false);
      setSpreadsheetToDelete(null);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Failed to delete spreadsheet');
      }
    } finally {
      setIsDeleting(false);
    }
  };

  // Cancel delete
  const handleCancelDelete = () => {
    setShowDeleteConfirm(false);
    setSpreadsheetToDelete(null);
  };

  // Detail view: show the spreadsheet grid
  if (selectedSpreadsheet) {
    return (
      <div>
        <div className="flex items-center justify-between mb-4">
          <div>
            <h3 className="text-lg font-medium">{selectedSpreadsheet.name}</h3>
            <p className="text-sm text-gray-500">
              {selectedSpreadsheet.data.rows} rows x {selectedSpreadsheet.data.cols} columns
            </p>
          </div>
          <div className="flex gap-2">
            {isSaving && (
              <span className="text-sm text-gray-500 flex items-center">Saving...</span>
            )}
            <Button
              onClick={() => setSelectedSpreadsheet(null)}
              variant="outline"
              size="small"
            >
              Back to List
            </Button>
          </div>
        </div>
        <SpreadsheetGrid
          spreadsheet={selectedSpreadsheet}
          onChange={handleSpreadsheetChange}
          readOnly={false}
        />
      </div>
    );
  }

  // List view: show all spreadsheets
  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-medium">
          Spreadsheets ({spreadsheets.length})
        </h3>
        <Button
          onClick={handleCreateSpreadsheet}
          variant="primary"
          size="small"
          disabled={isCreating || isLoading}
        >
          {isCreating ? 'Creating...' : 'Add Spreadsheet'}
        </Button>
      </div>

      {/* Delete confirmation dialog */}
      {showDeleteConfirm && spreadsheetToDelete && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md mx-4 shadow-lg">
            <h3 className="text-lg font-medium mb-2">Delete Spreadsheet</h3>
            <p className="text-gray-600 mb-4">
              Are you sure you want to delete "{spreadsheetToDelete.name}"? This action cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <Button
                onClick={handleCancelDelete}
                variant="outline"
                size="small"
                disabled={isDeleting}
              >
                Cancel
              </Button>
              <Button
                onClick={handleConfirmDelete}
                variant="primary"
                size="small"
                disabled={isDeleting}
              >
                {isDeleting ? 'Deleting...' : 'Delete'}
              </Button>
            </div>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="text-center py-8 text-gray-500">
          Loading spreadsheets...
        </div>
      ) : spreadsheets.length === 0 ? (
        <div className="text-center py-8 text-gray-500">
          No spreadsheets found for this card.
          Click "Add Spreadsheet" to create one.
        </div>
      ) : (
        <div className="space-y-2">
          {spreadsheets.map((spreadsheet) => (
            <div
              key={spreadsheet.id}
              onClick={() => setSelectedSpreadsheet(spreadsheet)}
              className="p-3 border border-gray-200 rounded hover:bg-gray-50 cursor-pointer group"
            >
              <div className="flex justify-between items-start">
                <div className="flex-grow min-w-0">
                  <div className="font-medium text-gray-900">{spreadsheet.name}</div>
                  <div className="text-sm text-gray-500">
                    {spreadsheet.data.rows} rows x {spreadsheet.data.cols} columns
                    {spreadsheet.updated_at && (
                      <span className="ml-2">
                        Updated {new Date(spreadsheet.updated_at).toLocaleDateString()}
                      </span>
                    )}
                  </div>
                </div>
                <button
                  onClick={(e) => handleDeleteClick(spreadsheet, e)}
                  className="ml-2 p-2 min-w-[44px] min-h-[44px] flex items-center justify-center text-gray-400 hover:text-red-600 hover:bg-red-50 rounded opacity-0 group-hover:opacity-100 transition-all shrink-0"
                  title="Delete spreadsheet"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
                  </svg>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
