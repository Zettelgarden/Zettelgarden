import React, { useState, useRef, useEffect, KeyboardEvent } from 'react';
import { SpreadsheetCell as SpreadsheetCellModel } from '../../models/Spreadsheet';

interface SpreadsheetCellProps {
  cellRef: string;
  cell: SpreadsheetCellModel;
  onChange: (cellRef: string, cell: SpreadsheetCellModel) => void;
  readOnly?: boolean;
}

export function SpreadsheetCell({ cellRef, cell, onChange, readOnly = false }: SpreadsheetCellProps) {
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

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      inputRef.current?.blur();
    } else if (e.key === 'Escape') {
      setEditValue((cell.formula ? '=' + cell.formula : cell.value) || '');
      setIsEditing(false);
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
