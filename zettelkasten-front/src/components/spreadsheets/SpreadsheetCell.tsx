import React, { useState, useRef, useEffect, KeyboardEvent, useCallback } from 'react';
import { SpreadsheetCell as SpreadsheetCellModel } from '../../models/Spreadsheet';
import { a1ToCoords, coordsToA1 } from '../../utils/spreadsheetHelpers';

interface SpreadsheetCellProps {
  cellRef: string;
  cell: SpreadsheetCellModel;
  onChange: (cellRef: string, cell: SpreadsheetCellModel) => void;
  readOnly?: boolean;
  rows?: number;
  cols?: number;
  onNavigate?: (cellRef: string) => void;
}

export function SpreadsheetCell({
  cellRef,
  cell,
  onChange,
  readOnly = false,
  rows = 5,
  cols = 5,
  onNavigate
}: SpreadsheetCellProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState((cell.formula ? '=' + cell.formula : cell.value) || '');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

  const handleDoubleClick = () => {
    if (!readOnly) {
      setIsEditing(true);
      setEditValue((cell.formula ? '=' + cell.formula : cell.value) || '');
    }
  };

  const handleBlur = () => {
    setIsEditing(false);

    // Determine if this is a formula or plain value
    const trimmed = editValue.trim();
    const newCell: SpreadsheetCellModel = {
      value: trimmed.startsWith('=') ? '' : trimmed,
      formula: trimmed.startsWith('=') ? trimmed.slice(1) : ''
    };

    onChange(cellRef, newCell);
  };

  const navigateTo = useCallback((direction: 'up' | 'down' | 'left' | 'right') => {
    const { row, col } = a1ToCoords(cellRef);
    let newRow = row;
    let newCol = col;

    switch (direction) {
      case 'up':
        newRow = Math.max(0, row - 1);
        break;
      case 'down':
        newRow = Math.min(rows - 1, row + 1);
        break;
      case 'left':
        newCol = Math.max(0, col - 1);
        break;
      case 'right':
        newCol = Math.min(cols - 1, col + 1);
        break;
    }

    const newCellRef = coordsToA1(newRow, newCol);
    onNavigate?.(newCellRef);
  }, [cellRef, rows, cols, onNavigate]);

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      // On Enter, save and move down
      inputRef.current?.blur();
      navigateTo('down');
    } else if (e.key === 'Tab') {
      e.preventDefault();
      if (e.shiftKey) {
        // Shift+Tab moves left
        inputRef.current?.blur();
        navigateTo('left');
      } else {
        // Tab moves right
        inputRef.current?.blur();
        navigateTo('right');
      }
    } else if (e.key === 'Escape') {
      setEditValue((cell.formula ? '=' + cell.formula : cell.value) || '');
      setIsEditing(false);
    } else if (isEditing) {
      // Arrow keys during editing - allow normal navigation within input
      const input = inputRef.current;
      if (input) {
        const { selectionStart, selectionEnd } = input;
        const valueLength = input.value.length;

        // Only navigate to another cell if at the edge of the text
        if (e.key === 'ArrowUp' && (selectionStart === 0 && selectionEnd === 0)) {
          e.preventDefault();
          inputRef.current?.blur();
          navigateTo('up');
        } else if (e.key === 'ArrowDown' && (selectionStart === valueLength && selectionEnd === valueLength)) {
          e.preventDefault();
          inputRef.current?.blur();
          navigateTo('down');
        } else if (e.key === 'ArrowLeft' && (selectionStart === 0 && selectionEnd === 0)) {
          e.preventDefault();
          inputRef.current?.blur();
          navigateTo('left');
        } else if (e.key === 'ArrowRight' && (selectionStart === valueLength && selectionEnd === valueLength)) {
          e.preventDefault();
          inputRef.current?.blur();
          navigateTo('right');
        }
      }
    }
  };

  const displayValue = isEditing ? '' : (cell.value || '\u00A0');

  return (
    <td
      className="border border-gray-300 px-2 py-1 min-w-[80px] h-8 text-sm"
      onDoubleClick={handleDoubleClick}
      style={{ cursor: readOnly ? 'default' : 'pointer' }}
    >
      {isEditing ? (
        <input
          ref={inputRef}
          type="text"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onBlur={handleBlur}
          onKeyDown={handleKeyDown}
          className="w-full h-full outline-none bg-blue-50"
        />
      ) : (
        <span className={cell.formula ? 'text-blue-600' : 'text-gray-900'}>
          {displayValue}
        </span>
      )}
    </td>
  );
}
