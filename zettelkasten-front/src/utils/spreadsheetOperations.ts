import type { Spreadsheet, SpreadsheetCell } from '../models/Spreadsheet';
import { coordsToA1, a1ToCoords } from './spreadsheetHelpers';

const MAX_ROWS = 100;
const MAX_COLS = 26;

export function insertRow(spreadsheet: Spreadsheet, rowIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  if (rows >= MAX_ROWS) {
    return spreadsheet;
  }

  const insertIndex = Math.max(0, Math.min(rowIndex, rows));

  const newData: Record<string, SpreadsheetCell> = { ...data };

  // Shift existing data down
  for (let row = rows - 1; row >= insertIndex; row--) {
    for (let col = 0; col < cols; col++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row + 1, col);

      if (data[oldRef]) {
        newData[newRef] = { ...data[oldRef] };
        delete newData[oldRef];
      }
    }
  }

  // Insert empty cells at the new row
  for (let col = 0; col < cols; col++) {
    const newRef = coordsToA1(insertIndex, col);
    newData[newRef] = { value: '', formula: '' };
  }

  return {
    ...spreadsheet,
    data: {
      rows: rows + 1,
      cols,
      data: newData
    }
  };
}

export function deleteRow(spreadsheet: Spreadsheet, rowIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  // Prevent deleting the last row
  if (rows <= 1) {
    return spreadsheet;
  }

  const deleteIndex = Math.max(0, Math.min(rowIndex, rows - 1));

  const newData: Record<string, SpreadsheetCell> = { ...data };

  // Remove the deleted row's cells
  for (let col = 0; col < cols; col++) {
    const ref = coordsToA1(deleteIndex, col);
    delete newData[ref];
  }

  // Shift data up
  for (let row = deleteIndex + 1; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row - 1, col);

      if (data[oldRef]) {
        newData[newRef] = { ...data[oldRef] };
        delete newData[oldRef];
      }
    }
  }

  return {
    ...spreadsheet,
    data: {
      rows: rows - 1,
      cols,
      data: newData
    }
  };
}

export function insertColumn(spreadsheet: Spreadsheet, colIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  if (cols >= MAX_COLS) {
    return spreadsheet;
  }

  const insertIndex = Math.max(0, Math.min(colIndex, cols));

  const newData: Record<string, SpreadsheetCell> = { ...data };

  // Shift existing data right
  for (let col = cols - 1; col >= insertIndex; col--) {
    for (let row = 0; row < rows; row++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row, col + 1);

      if (data[oldRef]) {
        newData[newRef] = { ...data[oldRef] };
        delete newData[oldRef];
      }
    }
  }

  // Insert empty cells at the new column
  for (let row = 0; row < rows; row++) {
    const newRef = coordsToA1(row, insertIndex);
    newData[newRef] = { value: '', formula: '' };
  }

  return {
    ...spreadsheet,
    data: {
      rows,
      cols: cols + 1,
      data: newData
    }
  };
}

export function deleteColumn(spreadsheet: Spreadsheet, colIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  // Prevent deleting the last column
  if (cols <= 1) {
    return spreadsheet;
  }

  const deleteIndex = Math.max(0, Math.min(colIndex, cols - 1));

  const newData: Record<string, SpreadsheetCell> = { ...data };

  // Remove the deleted column's cells
  for (let row = 0; row < rows; row++) {
    const ref = coordsToA1(row, deleteIndex);
    delete newData[ref];
  }

  // Shift data left
  for (let col = deleteIndex + 1; col < cols; col++) {
    for (let row = 0; row < rows; row++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row, col - 1);

      if (data[oldRef]) {
        newData[newRef] = { ...data[oldRef] };
        delete newData[oldRef];
      }
    }
  }

  return {
    ...spreadsheet,
    data: {
      rows,
      cols: cols - 1,
      data: newData
    }
  };
}
