import React from 'react';
import { Spreadsheet, SpreadsheetCell } from '../../models/Spreadsheet';
import { SpreadsheetCell as SpreadsheetCellComponent } from './SpreadsheetCell';
import { evaluateFormula } from './formulaParser';

interface SpreadsheetGridProps {
  spreadsheet: Spreadsheet;
  onChange: (spreadsheet: Spreadsheet) => void;
  readOnly?: boolean;
}

export function SpreadsheetGrid({ spreadsheet, onChange, readOnly = false }: SpreadsheetGridProps) {
  const { rows, cols, data } = spreadsheet.data;

  const handleCellChange = (cellRef: string, newCell: SpreadsheetCell) => {
    const newData = { ...data };

    // Update the cell
    newData[cellRef] = newCell;

    // Recalculate all formulas that depend on this cell
    // For now, we'll do a simple recalculation of all formula cells
    // A more sophisticated approach would track dependencies
    for (const [ref, cell] of Object.entries(newData)) {
      if (cell.formula) {
        const result = evaluateFormula(cell.formula, newData);
        newData[ref] = {
          ...cell,
          value: result !== null ? result.toString() : '#ERROR'
        };
      }
    }

    onChange({
      ...spreadsheet,
      data: {
        ...spreadsheet.data,
        data: newData
      }
    });
  };

  const columnHeaders = Array.from({ length: cols }, (_, i) =>
    String.fromCharCode(65 + i)
  );

  const rowHeaders = Array.from({ length: rows }, (_, i) => (i + 1).toString());

  return (
    <div className="overflow-x-auto">
      <table className="border-collapse border border-gray-400">
        <thead>
          <tr>
            <th className="border border-gray-300 bg-gray-100 px-2 py-1 w-8"></th>
            {columnHeaders.map((col) => (
              <th key={col} className="border border-gray-300 bg-gray-100 px-2 py-1 min-w-[80px] font-semibold text-sm">
                {col}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rowHeaders.map((row) => (
            <tr key={row}>
              <td className="border border-gray-300 bg-gray-100 px-2 py-1 text-center font-semibold text-sm">
                {row}
              </td>
              {columnHeaders.map((col) => {
                const cellRef = `${col}${row}`;
                const cell = data[cellRef] || { value: '', formula: '' };

                return (
                  <SpreadsheetCellComponent
                    key={cellRef}
                    cellRef={cellRef}
                    cell={cell}
                    onChange={handleCellChange}
                    readOnly={readOnly}
                  />
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
