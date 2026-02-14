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

  // Collect deleted references (empty for insert)
  const deletedRefs = new Set<string>();

  // Move cells down and update formulas
  for (let row = rows - 1; row >= insertIndex; row--) {
    for (let col = 0; col < cols; col++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row + 1, col);

      if (data[oldRef]) {
        const cell = { ...data[oldRef] };
        // Update formula references
        if (cell.formula) {
          cell.formula = updateFormulaReferences(cell.formula, 1, 0, insertIndex + 1, deletedRefs);
        }
        newData[newRef] = cell;
        delete newData[oldRef];
      }
    }
  }

  // Create empty cells for the new row
  for (let col = 0; col < cols; col++) {
    const newRef = coordsToA1(insertIndex, col);
    newData[newRef] = { value: '' };
  }

  // Update formulas in cells that were not moved
  for (let row = 0; row < insertIndex; row++) {
    for (let col = 0; col < cols; col++) {
      const ref = coordsToA1(row, col);
      if (newData[ref] && newData[ref].formula) {
        newData[ref] = {
          ...newData[ref],
          formula: updateFormulaReferences(newData[ref].formula!, 1, 0, insertIndex + 1, deletedRefs)
        };
      }
    }
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

  // Track deleted references
  const deletedRefs = new Set<string>();
  for (let col = 0; col < cols; col++) {
    deletedRefs.add(coordsToA1(deleteIndex, col));
  }

  // Delete cells at the target row
  for (let col = 0; col < cols; col++) {
    const ref = coordsToA1(deleteIndex, col);
    delete newData[ref];
  }

  // Process all existing cells to shift up and update formulas
  const cellsToProcess: Array<{ oldRef: string; newRef: string; cell: SpreadsheetCell }> = [];
  const movedRefs = new Set<string>();

  for (const ref in data) {
    try {
      const coords = a1ToCoords(ref);
      if (coords.row > deleteIndex) {
        const newRef = coordsToA1(coords.row - 1, coords.col);
        const cell = { ...data[ref] };
        if (cell.formula) {
          cell.formula = updateFormulaReferences(cell.formula, -1, 0, deleteIndex + 1, deletedRefs);
        }
        cellsToProcess.push({ oldRef: ref, newRef, cell });
        movedRefs.add(newRef);
      }
    } catch {
      // Invalid ref, skip
    }
  }

  // Move cells
  for (const { oldRef, newRef, cell } of cellsToProcess) {
    newData[newRef] = cell;
    delete newData[oldRef];
  }

  // Update formulas in cells that were not moved
  for (const ref in newData) {
    if (!movedRefs.has(ref) && newData[ref].formula) {
      newData[ref] = {
        ...newData[ref],
        formula: updateFormulaReferences(newData[ref].formula!, -1, 0, deleteIndex + 1, deletedRefs)
      };
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

  // Collect deleted references (empty for insert)
  const deletedRefs = new Set<string>();

  // Move cells right and update formulas
  for (let col = cols - 1; col >= insertIndex; col--) {
    for (let row = 0; row < rows; row++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row, col + 1);

      if (data[oldRef]) {
        const cell = { ...data[oldRef] };
        // Update formula references
        if (cell.formula) {
          cell.formula = updateFormulaReferences(cell.formula, 0, 1, insertIndex + 1, deletedRefs);
        }
        newData[newRef] = cell;
        delete newData[oldRef];
      }
    }
  }

  // Create empty cells for the new column
  for (let row = 0; row < rows; row++) {
    const newRef = coordsToA1(row, insertIndex);
    newData[newRef] = { value: '' };
  }

  // Update formulas in cells that were not moved
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < insertIndex; col++) {
      const ref = coordsToA1(row, col);
      if (newData[ref] && newData[ref].formula) {
        newData[ref] = {
          ...newData[ref],
          formula: updateFormulaReferences(newData[ref].formula!, 0, 1, insertIndex + 1, deletedRefs)
        };
      }
    }
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

  // Track deleted references
  const deletedRefs = new Set<string>();
  for (let row = 0; row < rows; row++) {
    deletedRefs.add(coordsToA1(row, deleteIndex));
  }

  // Delete cells at the target column
  for (let row = 0; row < rows; row++) {
    const ref = coordsToA1(row, deleteIndex);
    delete newData[ref];
  }

  // Process all existing cells to shift left and update formulas
  const cellsToProcess: Array<{ oldRef: string; newRef: string; cell: SpreadsheetCell }> = [];
  const movedRefs = new Set<string>();

  for (const ref in data) {
    try {
      const coords = a1ToCoords(ref);
      if (coords.col > deleteIndex) {
        const newRef = coordsToA1(coords.row, coords.col - 1);
        const cell = { ...data[ref] };
        if (cell.formula) {
          cell.formula = updateFormulaReferences(cell.formula, 0, -1, deleteIndex + 1, deletedRefs);
        }
        cellsToProcess.push({ oldRef: ref, newRef, cell });
        movedRefs.add(newRef);
      }
    } catch {
      // Invalid ref, skip
    }
  }

  // Move cells
  for (const { oldRef, newRef, cell } of cellsToProcess) {
    newData[newRef] = cell;
    delete newData[oldRef];
  }

  // Update formulas in cells that were not moved
  for (const ref in newData) {
    if (!movedRefs.has(ref) && newData[ref].formula) {
      newData[ref] = {
        ...newData[ref],
        formula: updateFormulaReferences(newData[ref].formula!, 0, -1, deleteIndex + 1, deletedRefs)
      };
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

/**
 * Check if a row contains any non-empty cell values
 * @param spreadsheet - The spreadsheet to check
 * @param rowIndex - The row index to check (0-based)
 * @returns true if any cell in the row has a non-empty value
 */
export function hasDataInRow(spreadsheet: Spreadsheet, rowIndex: number): boolean {
  const { rows, cols, data } = spreadsheet.data;
  if (rowIndex < 0 || rowIndex >= rows) return false;

  for (let col = 0; col < cols; col++) {
    const ref = coordsToA1(rowIndex, col);
    const cell = data[ref];
    if (cell && cell.value !== '') {
      return true;
    }
  }
  return false;
}

/**
 * Check if a column contains any non-empty cell values
 * @param spreadsheet - The spreadsheet to check
 * @param colIndex - The column index to check (0-based)
 * @returns true if any cell in the column has a non-empty value
 */
export function hasDataInColumn(spreadsheet: Spreadsheet, colIndex: number): boolean {
  const { rows, cols, data } = spreadsheet.data;
  if (colIndex < 0 || colIndex >= cols) return false;

  for (let row = 0; row < rows; row++) {
    const ref = coordsToA1(row, colIndex);
    const cell = data[ref];
    if (cell && cell.value !== '') {
      return true;
    }
  }
  return false;
}

/**
 * Update cell references in a formula after row/column insert/delete
 * @param formula - The formula string (without leading =)
 * @param rowDelta - Row offset (+1 for insert, -1 for delete, 0 for no change)
 * @param colDelta - Column offset (+1 for insert, -1 for delete, 0 for no change)
 * @param beforePosition - Only update refs with row/col >= this position (for insert) or > this (for delete)
 * @param deletedRefs - Set of cell references that were deleted (optional)
 * @returns The updated formula string
 */
export function updateFormulaReferences(
  formula: string,
  rowDelta: number,
  colDelta: number,
  beforePosition: number,
  deletedRefs?: Set<string>
): string {
  if (!formula) return formula;

  // Match cell references like A1, B2, AA10, and ranges like A1:A5
  const cellRefRegex = /([A-Z]+)(\d+)(?::([A-Z]+)(\d+))?/g;

  return formula.replace(cellRefRegex, (match, colStr, rowStr, rangeColStr, rangeRowStr) => {
    // Handle range reference
    if (rangeColStr && rangeRowStr) {
      const startRef = updateSingleReference(colStr, rowStr, rowDelta, colDelta, beforePosition, deletedRefs);
      const endRef = updateSingleReference(rangeColStr, rangeRowStr, rowDelta, colDelta, beforePosition, deletedRefs);
      return `${startRef}:${endRef}`;
    }

    // Handle single reference
    return updateSingleReference(colStr, rowStr, rowDelta, colDelta, beforePosition, deletedRefs);
  });
}

function updateSingleReference(
  colStr: string,
  rowStr: string,
  rowDelta: number,
  colDelta: number,
  beforePosition: number,
  deletedRefs?: Set<string>
): string {
  const ref = `${colStr}${rowStr}`;

  // Check if this reference was deleted
  if (deletedRefs?.has(ref)) {
    return '#REF!';
  }

  let coords;
  try {
    coords = a1ToCoords(ref);
  } catch {
    return ref; // Invalid ref, return as-is
  }

  if (coords.row < 0 || coords.col < 0) {
    return ref; // Invalid ref, return as-is
  }

  let newCol = coords.col;
  let newRow = coords.row;

  // Convert beforePosition from 1-indexed to 0-indexed
  // beforePosition=2 means "before row 2" which is index 1
  const beforePosition0Indexed = Math.max(0, beforePosition - 1);

  // Update row if it's at or after the position
  // For insert (rowDelta > 0): update rows >= beforePosition
  // For delete (rowDelta < 0): update rows > beforePosition
  if (rowDelta !== 0) {
    const rowThreshold = rowDelta > 0 ? beforePosition0Indexed : beforePosition0Indexed + 1;
    if (coords.row >= rowThreshold) {
      newRow = coords.row + rowDelta;
    }
  }

  // Update column if it's at or after the position
  // For insert (colDelta > 0): update cols >= beforePosition
  // For delete (colDelta < 0): update cols > beforePosition
  if (colDelta !== 0) {
    const colThreshold = colDelta > 0 ? beforePosition0Indexed : beforePosition0Indexed + 1;
    if (coords.col >= colThreshold) {
      newCol = coords.col + colDelta;
    }
  }

  // Validate new position is within bounds
  if (newRow < 0 || newRow >= MAX_ROWS || newCol < 0 || newCol >= MAX_COLS) {
    return '#REF!';
  }

  return coordsToA1(newRow, newCol);
}
