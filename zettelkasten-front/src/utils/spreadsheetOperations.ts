import type { Spreadsheet, SpreadsheetCell } from '../models/Spreadsheet';
import { coordsToA1, a1ToCoords } from './spreadsheetHelpers';

export const MAX_ROWS = 100;
export const MAX_COLS = 26;

/**
 * Inserts a new row at the specified index in the spreadsheet.
 * Existing rows from the specified index onwards are shifted down.
 * If the spreadsheet already has MAX_ROWS, no change is made.
 *
 * @param spreadsheet - The spreadsheet to modify
 * @param rowIndex - The index at which to insert the new row (0-based)
 * @returns A new spreadsheet with the row inserted, or the original spreadsheet if MAX_ROWS is reached
 *
 * @example
 * ```ts
 * const spreadsheet = { id: 1, data: { rows: 3, cols: 2, data: {} } };
 * const result = insertRow(spreadsheet, 1);
 * console.log(result.data.rows); // 4
 * ```
 */
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
    newData[newRef] = { value: '' };
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

/**
 * Deletes a row at the specified index from the spreadsheet.
 * Rows below the deleted row are shifted up.
 * If the spreadsheet has only one row, no change is made.
 *
 * @param spreadsheet - The spreadsheet to modify
 * @param rowIndex - The index of the row to delete (0-based)
 * @returns A new spreadsheet with the row deleted, or the original spreadsheet if only one row exists
 *
 * @example
 * ```ts
 * const spreadsheet = { id: 1, data: { rows: 4, cols: 2, data: {} } };
 * const result = deleteRow(spreadsheet, 1);
 * console.log(result.data.rows); // 3
 * ```
 */
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

/**
 * Inserts a new column at the specified index in the spreadsheet.
 * Existing columns from the specified index onwards are shifted right.
 * If the spreadsheet already has MAX_COLS, no change is made.
 *
 * @param spreadsheet - The spreadsheet to modify
 * @param colIndex - The index at which to insert the new column (0-based)
 * @returns A new spreadsheet with the column inserted, or the original spreadsheet if MAX_COLS is reached
 *
 * @example
 * ```ts
 * const spreadsheet = { id: 1, data: { rows: 2, cols: 3, data: {} } };
 * const result = insertColumn(spreadsheet, 1);
 * console.log(result.data.cols); // 4
 * ```
 */
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
    newData[newRef] = { value: '' };
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

/**
 * Deletes a column at the specified index from the spreadsheet.
 * Columns to the right of the deleted column are shifted left.
 * If the spreadsheet has only one column, no change is made.
 *
 * @param spreadsheet - The spreadsheet to modify
 * @param colIndex - The index of the column to delete (0-based)
 * @returns A new spreadsheet with the column deleted, or the original spreadsheet if only one column exists
 *
 * @example
 * ```ts
 * const spreadsheet = { id: 1, data: { rows: 2, cols: 4, data: {} } };
 * const result = deleteColumn(spreadsheet, 1);
 * console.log(result.data.cols); // 3
 * ```
 */
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
