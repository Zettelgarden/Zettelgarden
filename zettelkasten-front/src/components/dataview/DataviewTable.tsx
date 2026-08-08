import React, { useState } from 'react';
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell,
} from '../table/TableComponents';

interface DataviewTableProps {
  content: string;
  onSave?: (newContent: string) => void;
}

export const DataviewTable: React.FC<DataviewTableProps> = ({
  content,
  onSave,
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editableData, setEditableData] = useState<{
    headers: string[];
    rows: string[][];
  }>(() => parseCSV(content));

  // Parse CSV content into structured data
  function parseCSV(csvContent: string) {
    const lines = csvContent.trim().split('\n');
    if (lines.length === 0) {
      return { headers: [], rows: [] };
    }

    const headers = lines[0].split(',').map((h) => h.trim());
    const rows = lines
      .slice(1)
      .map((line) => line.split(',').map((cell) => cell.trim()));

    return { headers, rows };
  }

  // Convert structured data back to CSV
  function toCSV(data: { headers: string[]; rows: string[][] }) {
    return [
      data.headers.join(','),
      ...data.rows.map((row) => row.join(',')),
    ].join('\n');
  }

  // Handle cell value changes
  const updateCell = (rowIndex: number, colIndex: number, value: string) => {
    setEditableData((prev) => {
      const newData = { ...prev };
      if (rowIndex === -1) {
        // Update header
        newData.headers = [...prev.headers];
        newData.headers[colIndex] = value;
      } else {
        // Update cell
        newData.rows = [...prev.rows.map((row) => [...row])];
        newData.rows[rowIndex][colIndex] = value;
      }
      return newData;
    });
  };

  // Add a new row
  const addRow = () => {
    setEditableData((prev) => ({
      ...prev,
      rows: [...prev.rows, prev.headers.map(() => '')],
    }));
  };

  // Delete a row
  const deleteRow = (index: number) => {
    setEditableData((prev) => ({
      ...prev,
      rows: prev.rows.filter((_, i) => i !== index),
    }));
  };

  // Save changes
  const handleSave = () => {
    const csvContent = toCSV(editableData);
    if (onSave) {
      onSave(csvContent);
    }
    setIsEditing(false);
  };

  // Cancel editing
  const handleCancel = () => {
    setEditableData(parseCSV(content));
    setIsEditing(false);
  };

  // Toggle edit mode
  const toggleEditMode = () => {
    if (isEditing) {
      handleCancel();
    } else {
      setIsEditing(true);
    }
  };

  // If there's no data, show a placeholder
  if (editableData.headers.length === 0) {
    return <div className="text-gray-500">Empty dataview</div>;
  }

  // Render edit mode
  if (isEditing) {
    return (
      <div className="dataview-editor my-4">
        <Table>
          <TableHead>
            <TableRow>
              {editableData.headers.map((header, i) => (
                <TableHeader key={`header-${i}`}>
                  <input
                    value={header}
                    onChange={(e) => updateCell(-1, i, e.target.value)}
                    className="w-full px-2 py-1 border"
                  />
                </TableHeader>
              ))}
              <TableHeader>Actions</TableHeader>
            </TableRow>
          </TableHead>
          <TableBody>
            {editableData.rows.map((row, rowIdx) => (
              <TableRow key={`row-${rowIdx}`}>
                {row.map((cell, colIdx) => (
                  <TableCell key={`cell-${rowIdx}-${colIdx}`}>
                    <input
                      value={cell}
                      onChange={(e) =>
                        updateCell(rowIdx, colIdx, e.target.value)
                      }
                      className="w-full px-2 py-1 border"
                    />
                  </TableCell>
                ))}
                <TableCell>
                  <button
                    onClick={() => deleteRow(rowIdx)}
                    className="text-red-500 px-2 py-1 rounded hover:bg-red-50"
                  >
                    Delete
                  </button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <div className="flex mt-2 space-x-2">
          <button
            onClick={addRow}
            className="px-3 py-1 bg-blue-500 text-white rounded hover:bg-blue-600"
          >
            Add Row
          </button>
          <button
            onClick={handleSave}
            className="px-3 py-1 bg-green-500 text-white rounded hover:bg-green-600"
          >
            Save
          </button>
          <button
            onClick={handleCancel}
            className="px-3 py-1 bg-gray-500 text-white rounded hover:bg-gray-600"
          >
            Cancel
          </button>
        </div>
      </div>
    );
  }

  // Render view mode
  return (
    <div className="dataview-table my-4">
      <Table>
        <TableHead>
          <TableRow>
            {editableData.headers.map((header, i) => (
              <TableHeader key={`header-${i}`}>{header}</TableHeader>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {editableData.rows.map((row, rowIdx) => (
            <TableRow key={`row-${rowIdx}`}>
              {row.map((cell, colIdx) => (
                <TableCell key={`cell-${rowIdx}-${colIdx}`}>{cell}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {onSave && (
        <button
          onClick={toggleEditMode}
          className="mt-2 px-3 py-1 bg-gray-200 rounded text-sm hover:bg-gray-300"
        >
          Edit Table
        </button>
      )}
    </div>
  );
};
