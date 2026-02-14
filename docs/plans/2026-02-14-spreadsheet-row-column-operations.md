# Spreadsheet Row/Column Operations Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add insert/delete row and column operations to the spreadsheet component with Excel-like data shifting and automatic formula reference updates.

**Architecture:** Client-side manipulation with formula reference updating, using right-click context menu for UI. Operations flow through existing `onChange` → debounced save pattern in `DynamicSpreadsheet`.

**Tech Stack:** React, TypeScript, Vitest, existing formula parser in `formulaParser.ts`

---

## Overview

This plan adds the ability to insert and delete rows and columns in the spreadsheet component. The implementation:

1. Creates helper functions for row/column operations with data shifting
2. Adds context menu UI for right-click interactions
3. Updates formula references when rows/columns are inserted or deleted
4. Adds confirmation dialog for deletions with data
5. Enforces soft limits (100 rows, 26 columns)

---

## Task 1: Create spreadsheet operations utility functions

**Files:**
- Create: `zettelkasten-front/src/utils/spreadsheetOperations.ts`
- Test: `zettelkasten-front/src/utils/spreadsheetOperations.test.ts`

**Step 1: Write tests for row insertion**

```typescript
// spreadsheetOperations.test.ts
import { describe, it, expect } from 'vitest';
import { Spreadsheet, SpreadsheetData } from '../models/Spreadsheet';
import { insertRow, deleteRow, insertColumn, deleteColumn } from './spreadsheetOperations';

const createTestSpreadsheet = (rows: number, cols: number): Spreadsheet => ({
  id: 1,
  user_id: 1,
  card_id: 1,
  name: 'test',
  data: {
    rows,
    cols,
    data: {}
  },
  created_at: new Date(),
  updated_at: new Date()
});

describe('insertRow', () => {
  it('should insert a row at the beginning and shift existing data down', () => {
    const spreadsheet = createTestSpreadsheet(3, 2);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'B1': { value: 'b1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'B2': { value: 'b2', formula: '' },
      'A3': { value: 'a3', formula: '' },
      'B3': { value: 'b3', formula: '' },
    };

    const result = insertRow(spreadsheet, 0);

    expect(result.data.rows).toBe(4);
    expect(result.data.data['A1']).toEqual({ value: '', formula: '' });
    expect(result.data.data['A2']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['A3']).toEqual({ value: 'a2', formula: '' });
    expect(result.data.data['A4']).toEqual({ value: 'a3', formula: '' });
  });

  it('should insert a row in the middle', () => {
    const spreadsheet = createTestSpreadsheet(3, 2);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'A3': { value: 'a3', formula: '' },
    };

    const result = insertRow(spreadsheet, 1);

    expect(result.data.rows).toBe(4);
    expect(result.data.data['A1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['A2']).toEqual({ value: '', formula: '' });
    expect(result.data.data['A3']).toEqual({ value: 'a2', formula: '' });
    expect(result.data.data['A4']).toEqual({ value: 'a3', formula: '' });
  });

  it('should insert a row at the end', () => {
    const spreadsheet = createTestSpreadsheet(3, 2);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'A3': { value: 'a3', formula: '' },
    };

    const result = insertRow(spreadsheet, 3);

    expect(result.data.rows).toBe(4);
    expect(result.data.data['A1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['A3']).toEqual({ value: 'a3', formula: '' });
    expect(result.data.data['A4']).toEqual({ value: '', formula: '' });
  });

  it('should return original spreadsheet if rows >= 100', () => {
    const spreadsheet = createTestSpreadsheet(100, 2);
    const result = insertRow(spreadsheet, 0);

    expect(result).toEqual(spreadsheet);
    expect(result.data.rows).toBe(100);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: FAIL with "insertRow is not defined"

**Step 3: Implement insertRow function**

```typescript
// spreadsheetOperations.ts
import { Spreadsheet, SpreadsheetCell } from '../models/Spreadsheet';

const MAX_ROWS = 100;
const MAX_COLS = 26;

function a1ToCoords(ref: string): { row: number; col: number } {
  const match = ref.match(/^([A-Z]+)(\d+)$/);
  if (!match) return { row: -1, col: -1 };

  const [, colStr, rowStr] = match;
  let col = 0;
  for (let i = 0; i < colStr.length; i++) {
    col = col * 26 + (colStr.charCodeAt(i) - 64);
  }
  return { row: parseInt(rowStr, 10) - 1, col: col - 1 };
}

function coordsToA1(row: number, col: number): string {
  let colStr = '';
  let c = col + 1;
  while (c > 0) {
    c -= 1;
    colStr = String.fromCharCode(65 + (c % 26)) + colStr;
    c = Math.floor(c / 26);
  }
  return `${colStr}${row + 1}`;
}

export function insertRow(spreadsheet: Spreadsheet, rowIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  // Check soft limit
  if (rows >= MAX_ROWS) {
    return spreadsheet;
  }

  // Clamp rowIndex to valid range [0, rows]
  const insertIndex = Math.max(0, Math.min(rowIndex, rows));

  const newData: Record<string, SpreadsheetCell> = { ...data };

  // Move cells down: start from bottom and work up to avoid overwriting
  for (let row = rows - 1; row >= insertIndex; row--) {
    for (let col = 0; col < cols; col++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row + 1, col);

      if (data[oldRef]) {
        newData[newRef] = { ...data[oldRef] };
        // Clean up old reference if it still exists
        delete newData[oldRef];
      }
    }
  }

  // Create empty cells for the new row
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
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: PASS for all insertRow tests

**Step 5: Write tests for deleteRow**

```typescript
// Add to spreadsheetOperations.test.ts

describe('deleteRow', () => {
  it('should delete a row at the beginning and shift data up', () => {
    const spreadsheet = createTestSpreadsheet(4, 2);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'B1': { value: 'b1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'B2': { value: 'b2', formula: '' },
      'A3': { value: 'a3', formula: '' },
      'B3': { value: 'b3', formula: '' },
      'A4': { value: 'a4', formula: '' },
      'B4': { value: 'b4', formula: '' },
    };

    const result = deleteRow(spreadsheet, 0);

    expect(result.data.rows).toBe(3);
    expect(result.data.data['A1']).toEqual({ value: 'a2', formula: '' });
    expect(result.data.data['A2']).toEqual({ value: 'a3', formula: '' });
    expect(result.data.data['A3']).toEqual({ value: 'a4', formula: '' });
    expect(result.data.data['A4']).toBeUndefined();
  });

  it('should delete a row in the middle', () => {
    const spreadsheet = createTestSpreadsheet(4, 2);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'A3': { value: 'a3', formula: '' },
      'A4': { value: 'a4', formula: '' },
    };

    const result = deleteRow(spreadsheet, 1);

    expect(result.data.rows).toBe(3);
    expect(result.data.data['A1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['A2']).toEqual({ value: 'a3', formula: '' });
    expect(result.data.data['A3']).toEqual({ value: 'a4', formula: '' });
  });

  it('should not allow deleting the last row', () => {
    const spreadsheet = createTestSpreadsheet(1, 2);
    const result = deleteRow(spreadsheet, 0);

    expect(result).toEqual(spreadsheet);
    expect(result.data.rows).toBe(1);
  });
});
```

**Step 6: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: FAIL for deleteRow tests

**Step 7: Implement deleteRow function**

```typescript
// Add to spreadsheetOperations.ts

export function deleteRow(spreadsheet: Spreadsheet, rowIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  // Don't allow deleting the last row
  if (rows <= 1) {
    return spreadsheet;
  }

  // Clamp rowIndex to valid range [0, rows-1]
  const deleteIndex = Math.max(0, Math.min(rowIndex, rows - 1));

  const newData: Record<string, SpreadsheetCell> = { ...data };

  // Delete cells at the target row
  for (let col = 0; col < cols; col++) {
    const ref = coordsToA1(deleteIndex, col);
    delete newData[ref];
  }

  // Shift cells up
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
```

**Step 8: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: PASS for all deleteRow tests

**Step 9: Write tests for insertColumn and deleteColumn**

```typescript
// Add to spreadsheetOperations.test.ts

describe('insertColumn', () => {
  it('should insert a column at the beginning and shift data right', () => {
    const spreadsheet = createTestSpreadsheet(2, 3);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'B1': { value: 'b1', formula: '' },
      'C1': { value: 'c1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'B2': { value: 'b2', formula: '' },
      'C2': { value: 'c2', formula: '' },
    };

    const result = insertColumn(spreadsheet, 0);

    expect(result.data.cols).toBe(4);
    expect(result.data.data['A1']).toEqual({ value: '', formula: '' });
    expect(result.data.data['B1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['C1']).toEqual({ value: 'b1', formula: '' });
    expect(result.data.data['D1']).toEqual({ value: 'c1', formula: '' });
  });

  it('should return original spreadsheet if cols >= 26', () => {
    const spreadsheet = createTestSpreadsheet(5, 26);
    const result = insertColumn(spreadsheet, 0);

    expect(result).toEqual(spreadsheet);
    expect(result.data.cols).toBe(26);
  });
});

describe('deleteColumn', () => {
  it('should delete a column at the beginning and shift data left', () => {
    const spreadsheet = createTestSpreadsheet(2, 4);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'B1': { value: 'b1', formula: '' },
      'C1': { value: 'c1', formula: '' },
      'D1': { value: 'd1', formula: '' },
    };

    const result = deleteColumn(spreadsheet, 0);

    expect(result.data.cols).toBe(3);
    expect(result.data.data['A1']).toEqual({ value: 'b1', formula: '' });
    expect(result.data.data['B1']).toEqual({ value: 'c1', formula: '' });
    expect(result.data.data['C1']).toEqual({ value: 'd1', formula: '' });
    expect(result.data.data['D1']).toBeUndefined();
  });

  it('should not allow deleting the last column', () => {
    const spreadsheet = createTestSpreadsheet(2, 1);
    const result = deleteColumn(spreadsheet, 0);

    expect(result).toEqual(spreadsheet);
    expect(result.data.cols).toBe(1);
  });
});
```

**Step 10: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: FAIL for column tests

**Step 11: Implement insertColumn and deleteColumn functions**

```typescript
// Add to spreadsheetOperations.ts

export function insertColumn(spreadsheet: Spreadsheet, colIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  // Check soft limit
  if (cols >= MAX_COLS) {
    return spreadsheet;
  }

  // Clamp colIndex to valid range [0, cols]
  const insertIndex = Math.max(0, Math.min(colIndex, cols));

  const newData: Record<string, SpreadsheetCell> = { ...data };

  // Move cells right: start from rightmost and work left
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

  // Create empty cells for the new column
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

  // Don't allow deleting the last column
  if (cols <= 1) {
    return spreadsheet;
  }

  // Clamp colIndex to valid range [0, cols-1]
  const deleteIndex = Math.max(0, Math.min(colIndex, cols - 1));

  const newData: Record<string, SpreadsheetCell> = { ...data };

  // Delete cells at the target column
  for (let row = 0; row < rows; row++) {
    const ref = coordsToA1(row, deleteIndex);
    delete newData[ref];
  }

  // Shift cells left
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
```

**Step 12: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: PASS for all tests

**Step 13: Commit**

```bash
git add zettelkasten-front/src/utils/spreadsheetOperations.ts zettelkasten-front/src/utils/spreadsheetOperations.test.ts
git commit -m "feat: add spreadsheet row/column insert/delete operations

- Add insertRow, deleteRow, insertColumn, deleteColumn functions
- Support data shifting when rows/columns are inserted or deleted
- Enforce soft limits: 100 rows, 26 columns
- Prevent deletion of last row/column
- Add comprehensive unit tests
"
```

---

## Task 2: Add helper functions for data detection and formula updates

**Files:**
- Modify: `zettelkasten-front/src/utils/spreadsheetOperations.ts`
- Test: `zettelkasten-front/src/utils/spreadsheetOperations.test.ts`

**Step 1: Write tests for data detection helpers**

```typescript
// Add to spreadsheetOperations.test.ts

import { hasDataInRow, hasDataInColumn } from './spreadsheetOperations';

describe('hasDataInRow', () => {
  it('should return true if row has non-empty cells', () => {
    const spreadsheet = createTestSpreadsheet(3, 2);
    spreadsheet.data.data = {
      'A1': { value: '', formula: '' },
      'B1': { value: 'data', formula: '' },
      'A2': { value: '', formula: '' },
      'B2': { value: '', formula: '' },
    };

    expect(hasDataInRow(spreadsheet, 0)).toBe(true);
    expect(hasDataInRow(spreadsheet, 1)).toBe(false);
  });

  it('should return false if all cells in row are empty', () => {
    const spreadsheet = createTestSpreadsheet(2, 2);
    spreadsheet.data.data = {
      'A1': { value: '', formula: '' },
      'B1': { value: '', formula: '' },
    };

    expect(hasDataInRow(spreadsheet, 0)).toBe(false);
  });
});

describe('hasDataInColumn', () => {
  it('should return true if column has non-empty cells', () => {
    const spreadsheet = createTestSpreadsheet(2, 3);
    spreadsheet.data.data = {
      'A1': { value: '', formula: '' },
      'A2': { value: 'data', formula: '' },
      'B1': { value: '', formula: '' },
      'B2': { value: '', formula: '' },
    };

    expect(hasDataInColumn(spreadsheet, 0)).toBe(true);
    expect(hasDataInColumn(spreadsheet, 1)).toBe(false);
  });

  it('should return false if all cells in column are empty', () => {
    const spreadsheet = createTestSpreadsheet(2, 2);
    spreadsheet.data.data = {
      'A1': { value: '', formula: '' },
      'A2': { value: '', formula: '' },
    };

    expect(hasDataInColumn(spreadsheet, 0)).toBe(false);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: FAIL for hasDataInRow/Column tests

**Step 3: Implement data detection helpers**

```typescript
// Add to spreadsheetOperations.ts

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
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: PASS

**Step 5: Write tests for formula reference updates**

```typescript
// Add to spreadsheetOperations.test.ts

import { updateFormulaReferences } from './spreadsheetOperations';

describe('updateFormulaReferences', () => {
  it('should update row references when row inserted above', () => {
    const formula = 'A1+A2';
    const result = updateFormulaReferences(formula, 1, 0, 0);
    expect(result).toBe('A2+A3');
  });

  it('should update row references when row deleted', () => {
    const formula = 'A1+A3';
    const result = updateFormulaReferences(formula, -1, 0, 1);
    expect(result).toBe('A1+A2');
  });

  it('should update column references when column inserted left', () => {
    const formula = 'A1+B1';
    const result = updateFormulaReferences(formula, 0, 1, 0);
    expect(result).toBe('B1+C1');
  });

  it('should update column references when column deleted', () => {
    const formula = 'A1+C1';
    const result = updateFormulaReferences(formula, 0, -1, 1);
    expect(result).toBe('A1+B1');
  });

  it('should handle range references', () => {
    const formula = 'SUM(A1:A5)';
    const result = updateFormulaReferences(formula, 1, 0, 2);
    expect(result).toBe('SUM(A1:A6)');
  });

  it('should return #REF! for deleted references', () => {
    const formula = 'A1+A2';
    const result = updateFormulaReferences(formula, 0, 0, 1, new Set(['A1']));
    expect(result).toBe('#REF!+A2');
  });

  it('should handle function calls with multiple arguments', () => {
    const formula = 'SUM(A1,B2)+C3';
    const result = updateFormulaReferences(formula, 1, 1, 0);
    expect(result).toBe('SUM(A2,B3)+D4');
  });
});
```

**Step 6: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: FAIL for updateFormulaReferences tests

**Step 7: Implement formula reference update function**

```typescript
// Add to spreadsheetOperations.ts

/**
 * Update cell references in a formula after row/column insert/delete
 * @param formula - The formula string (without leading =)
 * @param rowDelta - Row offset (+1 for insert, -1 for delete, 0 for no change)
 * @param colDelta - Column offset (+1 for insert, -1 for delete, 0 for no change)
 * @param beforePosition - Only update refs with row/col >= this position (for insert) or > this (for delete)
 * @param deletedRefs - Set of cell references that were deleted (optional)
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

  const coords = a1ToCoords(ref);
  if (coords.row < 0 || coords.col < 0) {
    return ref; // Invalid ref, return as-is
  }

  let newCol = coords.col;
  let newRow = coords.row;

  // Update row if it's at or after the position
  const rowThreshold = rowDelta >= 0 ? beforePosition : beforePosition + 1;
  if (coords.row >= rowThreshold) {
    newRow = coords.row + rowDelta;
  }

  // Update column if it's at or after the position
  const colThreshold = colDelta >= 0 ? beforePosition : beforePosition + 1;
  if (coords.col >= colThreshold) {
    newCol = coords.col + colDelta;
  }

  // Validate new position is within bounds
  if (newRow < 0 || newRow >= 100 || newCol < 0 || newCol >= 26) {
    return '#REF!';
  }

  return coordsToA1(newRow, newCol);
}
```

**Step 8: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: PASS

**Step 9: Commit**

```bash
git add zettelkasten-front/src/utils/spreadsheetOperations.ts zettelkasten-front/src/utils/spreadsheetOperations.test.ts
git commit -m "feat: add data detection and formula update helpers

- Add hasDataInRow and hasDataInColumn for checking non-empty cells
- Add updateFormulaReferences for updating cell refs after insert/delete
- Handle single and range references (A1:B5)
- Return #REF! for deleted references
- Add comprehensive tests
"
```

---

## Task 3: Update row/column operations to handle formula updates

**Files:**
- Modify: `zettelkasten-front/src/utils/spreadsheetOperations.ts`
- Test: `zettelkasten-front/src/utils/spreadsheetOperations.test.ts`

**Step 1: Write tests for formula updates during insert/delete**

```typescript
// Add to spreadsheetOperations.test.ts

describe('insertRow with formulas', () => {
  it('should update formula references when inserting row', () => {
    const spreadsheet = createTestSpreadsheet(3, 2);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'A2': { value: '2', formula: '' },
      'A3': { value: '3', formula: '' },
      'B1': { value: '', formula: 'A1+A2' }, // should become A2+A3
    };

    const result = insertRow(spreadsheet, 0);

    expect(result.data.data['B2']).toEqual({
      value: '3',
      formula: 'A2+A3'
    });
  });

  it('should update formulas when inserting row in middle', () => {
    const spreadsheet = createTestSpreadsheet(3, 2);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'A2': { value: '2', formula: '' },
      'A3': { value: '3', formula: '' },
      'B1': { value: '', formula: 'A1+A3' },
    };

    const result = insertRow(spreadsheet, 1);

    // A3 becomes A4, so formula should update
    expect(result.data.data['B1']).toEqual({
      value: '',
      formula: 'A1+A4'
    });
  });
});

describe('deleteRow with formulas', () => {
  it('should update formula references and mark deleted refs', () => {
    const spreadsheet = createTestSpreadsheet(3, 2);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'A2': { value: '2', formula: '' },
      'A3': { value: '3', formula: '' },
      'B1': { value: '', formula: 'A1+A2+A3' },
    };

    const result = deleteRow(spreadsheet, 1); // Delete row 2 (index 1)

    expect(result.data.data['B1']).toEqual({
      value: '',
      formula: 'A1+#REF!+A2' // A3 becomes A2, A2 was deleted
    });
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: FAIL for formula update tests

**Step 3: Update insertRow to handle formulas**

```typescript
// Modify insertRow function in spreadsheetOperations.ts

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
          cell.formula = updateFormulaReferences(cell.formula, 1, 0, insertIndex, deletedRefs);
          // Recalculate value
          // Note: We'll need to import evaluateFormula from formulaParser
          cell.value = cell.formula; // Placeholder - will be recalculated by grid
        }
        newData[newRef] = cell;
        delete newData[oldRef];
      }
    }
  }

  // Create empty cells for the new row
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
```

**Step 4: Update deleteRow to handle formulas**

```typescript
// Modify deleteRow function in spreadsheetOperations.ts

export function deleteRow(spreadsheet: Spreadsheet, rowIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

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

  // Shift cells up and update formulas
  for (let row = deleteIndex + 1; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row - 1, col);

      if (data[oldRef]) {
        const cell = { ...data[oldRef] };
        // Update formula references
        if (cell.formula) {
          cell.formula = updateFormulaReferences(cell.formula, -1, 0, deleteIndex, deletedRefs);
          cell.value = cell.formula; // Placeholder - will be recalculated
        }
        newData[newRef] = cell;
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
```

**Step 5: Update insertColumn and deleteColumn similarly**

```typescript
// Modify insertColumn in spreadsheetOperations.ts

export function insertColumn(spreadsheet: Spreadsheet, colIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  if (cols >= MAX_COLS) {
    return spreadsheet;
  }

  const insertIndex = Math.max(0, Math.min(colIndex, cols));
  const newData: Record<string, SpreadsheetCell> = { ...data };
  const deletedRefs = new Set<string>();

  for (let col = cols - 1; col >= insertIndex; col--) {
    for (let row = 0; row < rows; row++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row, col + 1);

      if (data[oldRef]) {
        const cell = { ...data[oldRef] };
        if (cell.formula) {
          cell.formula = updateFormulaReferences(cell.formula, 0, 1, insertIndex, deletedRefs);
          cell.value = cell.formula;
        }
        newData[newRef] = cell;
        delete newData[oldRef];
      }
    }
  }

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

// Modify deleteColumn in spreadsheetOperations.ts

export function deleteColumn(spreadsheet: Spreadsheet, colIndex: number): Spreadsheet {
  const { rows, cols, data } = spreadsheet.data;

  if (cols <= 1) {
    return spreadsheet;
  }

  const deleteIndex = Math.max(0, Math.min(colIndex, cols - 1));
  const newData: Record<string, SpreadsheetCell> = { ...data };

  const deletedRefs = new Set<string>();
  for (let row = 0; row < rows; row++) {
    deletedRefs.add(coordsToA1(row, deleteIndex));
  }

  for (let row = 0; row < rows; row++) {
    const ref = coordsToA1(row, deleteIndex);
    delete newData[ref];
  }

  for (let col = deleteIndex + 1; col < cols; col++) {
    for (let row = 0; row < rows; row++) {
      const oldRef = coordsToA1(row, col);
      const newRef = coordsToA1(row, col - 1);

      if (data[oldRef]) {
        const cell = { ...data[oldRef] };
        if (cell.formula) {
          cell.formula = updateFormulaReferences(cell.formula, 0, -1, deleteIndex, deletedRefs);
          cell.value = cell.formula;
        }
        newData[newRef] = cell;
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
```

**Step 6: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- spreadsheetOperations.test.ts`
Expected: PASS

**Step 7: Commit**

```bash
git add zettelkasten-front/src/utils/spreadsheetOperations.ts zettelkasten-front/src/utils/spreadsheetOperations.test.ts
git commit -m "feat: update formulas when inserting/deleting rows and columns

- Modify insertRow/deleteRow/insertColumn/deleteColumn to update formulas
- Update cell references in formulas based on insert/delete position
- Mark deleted references with #REF!
- Shift formulas correctly when rows/columns are moved
"
```

---

## Task 4: Create context menu component

**Files:**
- Create: `zettelkasten-front/src/components/spreadsheets/SpreadsheetContextMenu.tsx`
- Test: `zettelkasten-front/src/components/spreadsheets/SpreadsheetContextMenu.test.tsx`

**Step 1: Write the context menu component**

```typescript
// SpreadsheetContextMenu.tsx
import React, { useEffect, useRef } from 'react';

export interface ContextMenuPosition {
  x: number;
  y: number;
}

export interface ContextMenuAction {
  label: string;
  action: () => void;
  disabled?: boolean;
}

interface SpreadsheetContextMenuProps {
  position: ContextMenuPosition | null;
  actions: ContextMenuAction[];
  onClose: () => void;
}

export function SpreadsheetContextMenu({ position, actions, onClose }: SpreadsheetContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    if (position) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleEscape);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [position, onClose]);

  if (!position) return null;

  // Calculate position to avoid viewport edge clipping
  const menuStyle: React.CSSProperties = {
    position: 'fixed',
    left: position.x,
    top: position.y,
    zIndex: 1000,
  };

  return (
    <div
      ref={menuRef}
      className="absolute bg-white border border-gray-300 rounded shadow-lg py-1 min-w-[160px]"
      style={menuStyle}
    >
      {actions.map((action, index) => (
        <button
          key={index}
          onClick={() => {
            if (!action.disabled) {
              action.action();
              onClose();
            }
          }}
          disabled={action.disabled}
          className={`
            w-full text-left px-4 py-2 text-sm
            ${action.disabled
              ? 'text-gray-400 cursor-not-allowed'
              : 'text-gray-700 hover:bg-gray-100 cursor-pointer'
            }
          `}
        >
          {action.label}
        </button>
      ))}
    </div>
  );
}
```

**Step 2: Write tests for context menu**

```typescript
// SpreadsheetContextMenu.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SpreadsheetContextMenu } from './SpreadsheetContextMenu';

describe('SpreadsheetContextMenu', () => {
  it('should render menu items when position is provided', () => {
    const actions = [
      { label: 'Insert Row', action: vi.fn() },
      { label: 'Delete Row', action: vi.fn() },
    ];

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={actions}
        onClose={() => {}}
      />
    );

    expect(screen.getByText('Insert Row')).toBeInTheDocument();
    expect(screen.getByText('Delete Row')).toBeInTheDocument();
  });

  it('should not render when position is null', () => {
    const { container } = render(
      <SpreadsheetContextMenu
        position={null}
        actions={[]}
        onClose={() => {}}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it('should call action and onClose when item clicked', () => {
    const action = vi.fn();
    const onClose = vi.fn();

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={[{ label: 'Test', action }]}
        onClose={onClose}
      />
    );

    fireEvent.click(screen.getByText('Test'));
    expect(action).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('should not call action for disabled item', () => {
    const action = vi.fn();
    const onClose = vi.fn();

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={[{ label: 'Disabled', action, disabled: true }]}
        onClose={onClose}
      />
    );

    fireEvent.click(screen.getByText('Disabled'));
    expect(action).not.toHaveBeenCalled();
  });

  it('should close when clicking outside', () => {
    const onClose = vi.fn();

    render(
      <>
        <SpreadsheetContextMenu
          position={{ x: 100, y: 100 }}
          actions={[{ label: 'Test', action: vi.fn() }]}
          onClose={onClose}
        />
        <div data-testid="outside">Outside</div>
      </>
    );

    fireEvent.click(screen.getByTestId('outside'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('should close when Escape key pressed', () => {
    const onClose = vi.fn();

    render(
      <SpreadsheetContextMenu
        position={{ x: 100, y: 100 }}
        actions={[{ label: 'Test', action: vi.fn() }]}
        onClose={onClose}
      />
    );

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
```

**Step 3: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- SpreadsheetContextMenu.test.tsx`
Expected: PASS

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/SpreadsheetContextMenu.tsx zettelkasten-front/src/components/spreadsheets/SpreadsheetContextMenu.test.tsx
git commit -m "feat: add spreadsheet context menu component

- Add SpreadsheetContextMenu for right-click actions
- Support for enabled/disabled menu items
- Close on outside click or Escape key
- Position to avoid viewport edge clipping
- Add unit tests
"
```

---

## Task 5: Create delete confirmation dialog component

**Files:**
- Create: `zettelkasten-front/src/components/spreadsheets/DeleteConfirmDialog.tsx`
- Test: `zettelkasten-front/src/components/spreadsheets/DeleteConfirmDialog.test.tsx`

**Step 1: Write the delete confirmation dialog component**

```typescript
// DeleteConfirmDialog.tsx
import React from 'react';

interface DeleteConfirmDialogProps {
  isOpen: boolean;
  itemType: 'row' | 'column';
  index: number;
  onConfirm: () => void;
  onCancel: () => void;
}

export function DeleteConfirmDialog({
  isOpen,
  itemType,
  index,
  onConfirm,
  onCancel
}: DeleteConfirmDialogProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
        <h2 className="text-lg font-semibold text-gray-900 mb-2">
          Delete {itemType} {index + 1}?
        </h2>
        <p className="text-sm text-gray-600 mb-6">
          This {itemType} contains data. Are you sure you want to delete it?
          This action cannot be undone.
        </p>
        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded hover:bg-red-700"
          >
            Delete {itemType}
          </button>
        </div>
      </div>
    </div>
  );
}
```

**Step 2: Write tests for delete confirmation dialog**

```typescript
// DeleteConfirmDialog.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DeleteConfirmDialog } from './DeleteConfirmDialog';

describe('DeleteConfirmDialog', () => {
  it('should render dialog when isOpen is true', () => {
    render(
      <DeleteConfirmDialog
        isOpen={true}
        itemType="row"
        index={2}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText('Delete row 3?')).toBeInTheDocument();
    expect(screen.getByText(/This row contains data/)).toBeInTheDocument();
  });

  it('should not render when isOpen is false', () => {
    const { container } = render(
      <DeleteConfirmDialog
        isOpen={false}
        itemType="column"
        index={1}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it('should call onConfirm when Delete button clicked', () => {
    const onConfirm = vi.fn();

    render(
      <DeleteConfirmDialog
        isOpen={true}
        itemType="row"
        index={0}
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /Delete row/i }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('should call onCancel when Cancel button clicked', () => {
    const onCancel = vi.fn();

    render(
      <DeleteConfirmDialog
        isOpen={true}
        itemType="column"
        index={5}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
```

**Step 3: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- DeleteConfirmDialog.test.tsx`
Expected: PASS

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/DeleteConfirmDialog.tsx zettelkasten-front/src/components/spreadsheets/DeleteConfirmDialog.test.tsx
git commit -m "feat: add delete confirmation dialog for spreadsheets

- Add DeleteConfirmDialog modal component
- Show warning when deleting rows/columns with data
- Provide Cancel and Confirm buttons
- Add unit tests
"
```

---

## Task 6: Integrate context menu into SpreadsheetGrid

**Files:**
- Modify: `zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.tsx`
- Test: `zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.test.tsx` (create if not exists)

**Step 1: Add context menu state and handlers to SpreadsheetGrid**

```typescript
// Add to SpreadsheetGrid.tsx imports
import { useState, useCallback } from 'react';
import { SpreadsheetContextMenu, ContextMenuPosition, ContextMenuAction } from './SpreadsheetContextMenu';

// Add inside SpreadsheetGrid component, after existing state
const [contextMenu, setContextMenu] = useState<{
  position: ContextMenuPosition;
  type: 'row' | 'column';
  index: number;
} | null>(null);

// Add context menu close handler
const closeContextMenu = useCallback(() => {
  setContextMenu(null);
}, []);

// Add row header right-click handler
const handleRowHeaderContextMenu = useCallback((e: React.MouseEvent, rowIndex: number) => {
  e.preventDefault();

  if (readOnly) return;

  setContextMenu({
    position: { x: e.clientX, y: e.clientY },
    type: 'row',
    index: rowIndex
  });
}, [readOnly]);

// Add column header right-click handler
const handleColumnHeaderContextMenu = useCallback((e: React.MouseEvent, colIndex: number) => {
  e.preventDefault();

  if (readOnly) return;

  setContextMenu({
    position: { x: e.clientX, y: e.clientY },
    type: 'column',
    index: colIndex
  });
}, [readOnly]);

// Add handlers for insert/delete operations (these will call parent onChange)
const handleInsertRowAbove = useCallback(() => {
  if (!contextMenu) return;
  // We'll implement this after updating onChange signature
  closeContextMenu();
}, [contextMenu, closeContextMenu]);

const handleInsertRowBelow = useCallback(() => {
  if (!contextMenu) return;
  closeContextMenu();
}, [contextMenu, closeContextMenu]);

const handleDeleteRow = useCallback(() => {
  if (!contextMenu) return;
  closeContextMenu();
}, [contextMenu, closeContextMenu]);

const handleInsertColumnLeft = useCallback(() => {
  if (!contextMenu) return;
  closeContextMenu();
}, [contextMenu, closeContextMenu]);

const handleInsertColumnRight = useCallback(() => {
  if (!contextMenu) return;
  closeContextMenu();
}, [contextMenu, closeContextMenu]);

const handleDeleteColumn = useCallback(() => {
  if (!contextMenu) return;
  closeContextMenu();
}, [contextMenu, closeContextMenu]);
```

**Step 2: Add context menu actions based on selection**

```typescript
// Add after the handlers, inside SpreadsheetGrid component
const contextMenuActions: ContextMenuAction[] = contextMenu ? (
  contextMenu.type === 'row' ? [
    { label: 'Insert Row Above', action: handleInsertRowAbove },
    { label: 'Insert Row Below', action: handleInsertRowBelow },
    { label: 'Delete Row', action: handleDeleteRow },
  ] : [
    { label: 'Insert Column Left', action: handleInsertColumnLeft },
    { label: 'Insert Column Right', action: handleInsertColumnRight },
    { label: 'Delete Column', action: handleDeleteColumn },
  ]
) : [];
```

**Step 3: Add onContextMenu to row and column headers**

```typescript
// Modify the thead section to add onContextMenu
<thead>
  <tr>
    <th className="border border-gray-300 bg-gray-100 px-2 py-1 w-8"></th>
    {columnHeaders.map((col, colIndex) => (
      <th
        key={col}
        className="border border-gray-300 bg-gray-100 px-2 py-1 min-w-[80px] font-semibold text-sm"
        onContextMenu={(e) => handleColumnHeaderContextMenu(e, colIndex)}
      >
        {col}
      </th>
    ))}
  </tr>
</thead>

// Modify the row header cell to add onContextMenu
<td
  className="border border-gray-300 bg-gray-100 px-2 py-1 text-center font-semibold text-sm"
  onContextMenu={(e) => handleRowHeaderContextMenu(e, parseInt(row, 10) - 1)}
>
  {row}
</td>
```

**Step 4: Add context menu to the render**

```typescript
// Add before the closing return in SpreadsheetGrid
return (
  <div className="overflow-x-auto relative">
    <table className="border-collapse border border-gray-400">
      {/* existing table content */}
    </table>
    <SpreadsheetContextMenu
      position={contextMenu?.position || null}
      actions={contextMenuActions}
      onClose={closeContextMenu}
    />
  </div>
);
```

**Step 5: Write tests for context menu integration**

```typescript
// SpreadsheetGrid.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SpreadsheetGrid } from './SpreadsheetGrid';
import { Spreadsheet } from '../../models/Spreadsheet';

const createMockSpreadsheet = (): Spreadsheet => ({
  id: 1,
  user_id: 1,
  card_id: 1,
  name: 'test',
  data: {
    rows: 3,
    cols: 3,
    data: {
      'A1': { value: 'test', formula: '' },
    }
  },
  created_at: new Date(),
  updated_at: new Date()
});

describe('SpreadsheetGrid context menu', () => {
  it('should show context menu on row header right-click', () => {
    const onChange = vi.fn();
    render(
      <SpreadsheetGrid
        spreadsheet={createMockSpreadsheet()}
        onChange={onChange}
        readOnly={false}
      />
    );

    const rowHeader = screen.getByText('1');
    fireEvent.contextMenu(rowHeader);

    expect(screen.getByText('Insert Row Above')).toBeInTheDocument();
    expect(screen.getByText('Insert Row Below')).toBeInTheDocument();
    expect(screen.getByText('Delete Row')).toBeInTheDocument();
  });

  it('should show context menu on column header right-click', () => {
    const onChange = vi.fn();
    render(
      <SpreadsheetGrid
        spreadsheet={createMockSpreadsheet()}
        onChange={onChange}
        readOnly={false}
      />
    );

    const colHeader = screen.getByText('A');
    fireEvent.contextMenu(colHeader);

    expect(screen.getByText('Insert Column Left')).toBeInTheDocument();
    expect(screen.getByText('Insert Column Right')).toBeInTheDocument();
    expect(screen.getByText('Delete Column')).toBeInTheDocument();
  });

  it('should not show context menu in read-only mode', () => {
    const onChange = vi.fn();
    render(
      <SpreadsheetGrid
        spreadsheet={createMockSpreadsheet()}
        onChange={onChange}
        readOnly={true}
      />
    );

    const rowHeader = screen.getByText('1');
    fireEvent.contextMenu(rowHeader);

    expect(screen.queryByText('Insert Row Above')).not.toBeInTheDocument();
  });

  it('should close context menu when clicking outside', () => {
    const onChange = vi.fn();
    const { container } = render(
      <SpreadsheetGrid
        spreadsheet={createMockSpreadsheet()}
        onChange={onChange}
        readOnly={false}
      />
    );

    const rowHeader = screen.getByText('1');
    fireEvent.contextMenu(rowHeader);

    expect(screen.getByText('Insert Row Above')).toBeInTheDocument();

    // Click outside
    fireEvent.mouseDown(document);

    expect(screen.queryByText('Insert Row Above')).not.toBeInTheDocument();
  });
});
```

**Step 6: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- SpreadsheetGrid.test.tsx`
Expected: PASS

**Step 7: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.tsx zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.test.tsx
git commit -m "feat: integrate context menu into SpreadsheetGrid

- Add right-click handlers to row and column headers
- Show context menu with insert/delete options
- Hide context menu in read-only mode
- Close menu on outside click
- Add unit tests
"
```

---

## Task 7: Add operation handlers to DynamicSpreadsheet

**Files:**
- Modify: `zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.tsx`
- Modify: `zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.tsx` (to pass handlers as props)

**Step 1: Add state for delete confirmation dialog in DynamicSpreadsheet**

```typescript
// Add to DynamicSpreadsheet.tsx imports
import { useState, useCallback } from 'react';
import { DeleteConfirmDialog } from './DeleteConfirmDialog';
import { insertRow, deleteRow, insertColumn, deleteColumn, hasDataInRow, hasDataInColumn } from '../../utils/spreadsheetOperations';

// Add inside DynamicSpreadsheet component, after existing state
const [deleteDialog, setDeleteDialog] = useState<{
  type: 'row' | 'column';
  index: number;
} | null>(null);

// Add handlers for insert/delete operations
const handleInsertRow = useCallback((rowIndex: number, above: boolean) => {
  const insertIndex = above ? rowIndex : rowIndex + 1;
  const updated = insertRow(
    { id, user_id: 0, card_id: 0, name: '', data: spreadsheetData, created_at: new Date(), updated_at: new Date() },
    insertIndex
  );

  // Call handleChange to update state and trigger save
  handleChange(updated);
}, [id, spreadsheetData, handleChange]);

const handleDeleteRow = useCallback((rowIndex: number) => {
  // Check if row has data
  const tempSpreadsheet = { id, user_id: 0, card_id: 0, name: '', data: spreadsheetData, created_at: new Date(), updated_at: new Date() };

  if (hasDataInRow(tempSpreadsheet, rowIndex)) {
    // Show confirmation dialog
    setDeleteDialog({ type: 'row', index: rowIndex });
  } else {
    // Delete immediately
    const updated = deleteRow(tempSpreadsheet, rowIndex);
    handleChange(updated);
  }
}, [id, spreadsheetData, handleChange]);

const handleConfirmDelete = useCallback(() => {
  if (!deleteDialog) return;

  const tempSpreadsheet = { id, user_id: 0, card_id: 0, name: '', data: spreadsheetData, created_at: new Date(), updated_at: new Date() };
  let updated;

  if (deleteDialog.type === 'row') {
    updated = deleteRow(tempSpreadsheet, deleteDialog.index);
  } else {
    updated = deleteColumn(tempSpreadsheet, deleteDialog.index);
  }

  handleChange(updated);
  setDeleteDialog(null);
}, [deleteDialog, id, spreadsheetData, handleChange]);

const handleInsertColumn = useCallback((colIndex: number, left: boolean) => {
  const insertIndex = left ? colIndex : colIndex + 1;
  const updated = insertColumn(
    { id, user_id: 0, card_id: 0, name: '', data: spreadsheetData, created_at: new Date(), updated_at: new Date() },
    insertIndex
  );

  handleChange(updated);
}, [id, spreadsheetData, handleChange]);

const handleDeleteColumn = useCallback((colIndex: number) => {
  const tempSpreadsheet = { id, user_id: 0, card_id: 0, name: '', data: spreadsheetData, created_at: new Date(), updated_at: new Date() };

  if (hasDataInColumn(tempSpreadsheet, colIndex)) {
    setDeleteDialog({ type: 'column', index: colIndex });
  } else {
    const updated = deleteColumn(tempSpreadsheet, colIndex);
    handleChange(updated);
  }
}, [id, spreadsheetData, handleChange]);
```

**Step 2: Update SpreadsheetGrid props interface and pass handlers**

```typescript
// Update SpreadsheetGridProps in SpreadsheetGrid.tsx
interface SpreadsheetGridProps {
  spreadsheet: Spreadsheet;
  onChange: (spreadsheet: Spreadsheet) => void;
  readOnly?: boolean;
  // Add operation handlers
  onInsertRow?: (rowIndex: number, above: boolean) => void;
  onDeleteRow?: (rowIndex: number) => void;
  onInsertColumn?: (colIndex: number, left: boolean) => void;
  onDeleteColumn?: (colIndex: number) => void;
}

// Update the component signature
export function SpreadsheetGrid({
  spreadsheet,
  onChange,
  readOnly = false,
  onInsertRow,
  onDeleteRow,
  onInsertColumn,
  onDeleteColumn
}: SpreadsheetGridProps) {
  // ... existing code ...

  // Update the handler implementations to call the props
  const handleInsertRowAbove = useCallback(() => {
    if (!contextMenu || onInsertRow) return;
    onInsertRow(contextMenu.index, true);
    closeContextMenu();
  }, [contextMenu, onInsertRow, closeContextMenu]);

  const handleInsertRowBelow = useCallback(() => {
    if (!contextMenu || onInsertRow) return;
    onInsertRow(contextMenu.index, false);
    closeContextMenu();
  }, [contextMenu, onInsertRow, closeContextMenu]);

  const handleDeleteRow = useCallback(() => {
    if (!contextMenu || onDeleteRow) return;
    onDeleteRow(contextMenu.index);
    closeContextMenu();
  }, [contextMenu, onDeleteRow, closeContextMenu]);

  const handleInsertColumnLeft = useCallback(() => {
    if (!contextMenu || onInsertColumn) return;
    onInsertColumn(contextMenu.index, true);
    closeContextMenu();
  }, [contextMenu, onInsertColumn, closeContextMenu]);

  const handleInsertColumnRight = useCallback(() => {
    if (!contextMenu || onInsertColumn) return;
    onInsertColumn(contextMenu.index, false);
    closeContextMenu();
  }, [contextMenu, onInsertColumn, closeContextMenu]);

  const handleDeleteColumn = useCallback(() => {
    if (!contextMenu || onDeleteColumn) return;
    onDeleteColumn(contextMenu.index);
    closeContextMenu();
  }, [contextMenu, onDeleteColumn, closeContextMenu]);
```

**Step 3: Update DynamicSpreadsheet to pass handlers to SpreadsheetGrid**

```typescript
// Update the SpreadsheetGrid usage in DynamicSpreadsheet.tsx
<SpreadsheetGrid
  spreadsheet={spreadsheetForGrid}
  onChange={handleChange}
  readOnly={readOnly}
  onInsertRow={handleInsertRow}
  onDeleteRow={handleDeleteRow}
  onInsertColumn={handleInsertColumn}
  onDeleteColumn={handleDeleteColumn}
/>
```

**Step 4: Add DeleteConfirmDialog to DynamicSpreadsheet render**

```typescript
// Add before the closing div in DynamicSpreadsheet
return (
  <div className="my-4 border-l-4 border-green-500 pl-4">
    {/* existing content */}
    <DeleteConfirmDialog
      isOpen={deleteDialog !== null}
      itemType={deleteDialog?.type || 'row'}
      index={deleteDialog?.index || 0}
      onConfirm={handleConfirmDelete}
      onCancel={() => setDeleteDialog(null)}
    />
  </div>
);
```

**Step 5: Write integration test for the full flow**

```typescript
// DynamicSpreadsheet.test.tsx (create new file)
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { DynamicSpreadsheet } from './DynamicSpreadsheet';
import * as api from '../../api/spreadsheets';

vi.mock('../../api/spreadsheets');

const mockInitialData = {
  rows: 3,
  cols: 3,
  data: {
    'A1': { value: 'test', formula: '' },
    'B2': { value: 'data', formula: '' },
  }
};

describe('DynamicSpreadsheet row/column operations', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.updateSpreadsheet).mockResolvedValue(undefined);
  });

  it('should insert a row above when context menu action selected', async () => {
    render(
      <DynamicSpreadsheet
        id={1}
        initialData={mockInitialData}
        readOnly={false}
      />
    );

    // Right-click on row header
    const rowHeader = screen.getByText('2');
    fireEvent.contextMenu(rowHeader);

    // Click "Insert Row Above"
    fireEvent.click(screen.getByText('Insert Row Above'));

    // Wait for state update
    await waitFor(() => {
      expect(screen.getByText('4')).toBeInTheDocument(); // Should now have 4 rows
    });
  });

  it('should show confirmation dialog when deleting row with data', async () => {
    render(
      <DynamicSpreadsheet
        id={1}
        initialData={mockInitialData}
        readOnly={false}
      />
    );

    // Right-click on row header (row 2 has data at B2)
    const rowHeader = screen.getByText('3'); // Row 3 (index 2) where B2 is
    fireEvent.contextMenu(rowHeader);

    // Click "Delete Row"
    fireEvent.click(screen.getByText('Delete Row'));

    // Should show confirmation dialog
    await waitFor(() => {
      expect(screen.getByText(/This row contains data/)).toBeInTheDocument();
    });
  });
});
```

**Step 6: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- DynamicSpreadsheet.test.tsx`
Expected: PASS

**Step 7: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.tsx zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.test.tsx zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.tsx
git commit -m "feat: connect row/column operations to DynamicSpreadsheet

- Add operation handlers in DynamicSpreadsheet
- Insert/delete rows and columns with data shifting
- Show confirmation dialog for deletions with data
- Pass handlers to SpreadsheetGrid via props
- Add integration tests
"
```

---

## Task 8: Add toast notification for soft limit warnings

**Files:**
- Create: `zettelkasten-front/src/components/Toast.tsx` (if not exists)
- Modify: `zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.tsx`

**Step 1: Create or update Toast component**

```typescript
// Toast.tsx (create if not exists, or update existing)
import React, { useEffect, useState } from 'react';

interface ToastProps {
  message: string;
  duration?: number;
  onClose: () => void;
}

export function Toast({ message, duration = 3000, onClose }: ToastProps) {
  const [isVisible, setIsVisible] = useState(true);

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsVisible(false);
      setTimeout(onClose, 300); // Wait for fade out animation
    }, duration);

    return () => clearTimeout(timer);
  }, [duration, onClose]);

  if (!isVisible) return null;

  return (
    <div
      className={`fixed bottom-4 right-4 bg-gray-800 text-white px-4 py-2 rounded shadow-lg transition-opacity duration-300 ${
        isVisible ? 'opacity-100' : 'opacity-0'
      }`}
    >
      {message}
    </div>
  );
}
```

**Step 2: Add toast state to DynamicSpreadsheet**

```typescript
// Add to DynamicSpreadsheet imports
import { Toast } from '../Toast';

// Add inside DynamicSpreadsheet component
const [toast, setToast] = useState<string | null>(null);

// Modify operation handlers to show toast when limit reached
const handleInsertRow = useCallback((rowIndex: number, above: boolean) => {
  const insertIndex = above ? rowIndex : rowIndex + 1;
  const tempSpreadsheet = { id, user_id: 0, card_id: 0, name: '', data: spreadsheetData, created_at: new Date(), updated_at: new Date() };

  if (spreadsheetData.rows >= 100) {
    setToast('Maximum rows reached (100)');
    return;
  }

  const updated = insertRow(tempSpreadsheet, insertIndex);

  // Check if insertRow returned the same spreadsheet (limit reached)
  if (updated === tempSpreadsheet) {
    setToast('Maximum rows reached (100)');
    return;
  }

  handleChange(updated);
}, [id, spreadsheetData, handleChange]);

// Similar updates for handleInsertColumn
const handleInsertColumn = useCallback((colIndex: number, left: boolean) => {
  const insertIndex = left ? colIndex : colIndex + 1;
  const tempSpreadsheet = { id, user_id: 0, card_id: 0, name: '', data: spreadsheetData, created_at: new Date(), updated_at: new Date() };

  if (spreadsheetData.cols >= 26) {
    setToast('Maximum columns reached (26)');
    return;
  }

  const updated = insertColumn(tempSpreadsheet, insertIndex);

  if (updated === tempSpreadsheet) {
    setToast('Maximum columns reached (26)');
    return;
  }

  handleChange(updated);
}, [id, spreadsheetData, handleChange]);
```

**Step 3: Add Toast to DynamicSpreadsheet render**

```typescript
// Add to DynamicSpreadsheet return
return (
  <div className="my-4 border-l-4 border-green-500 pl-4">
    {/* existing content */}
    <DeleteConfirmDialog
      isOpen={deleteDialog !== null}
      itemType={deleteDialog?.type || 'row'}
      index={deleteDialog?.index || 0}
      onConfirm={handleConfirmDelete}
      onCancel={() => setDeleteDialog(null)}
    />
    {toast && <Toast message={toast} onClose={() => setToast(null)} />}
  </div>
);
```

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.tsx zettelkasten-front/src/components/Toast.tsx
git commit -m "feat: add toast notifications for soft limits

- Show toast when reaching 100 row or 26 column limit
- Auto-dismiss after 3 seconds
- Fade-out animation
"
```

---

## Task 9: Manual testing and final verification

**Files:**
- None (manual testing)

**Step 1: Run all tests**

```bash
cd zettelkasten-front
npm test
```

Expected: All tests pass

**Step 2: Start dev server**

```bash
cd zettelkasten-front
npm start
```

**Step 3: Manual testing checklist**

Open a spreadsheet with existing data and test:

1. **Row Operations:**
   - [ ] Right-click row header → context menu appears
   - [ ] Click "Insert Row Above" → row added, data shifted down
   - [ ] Click "Insert Row Below" → row added below selected row
   - [ ] Edit cell, then insert row above → cell value moves down
   - [ ] Create formula =A1+A2, insert row at 0 → formula updates to =A2+A3

2. **Column Operations:**
   - [ ] Right-click column header → context menu appears
   - [ ] Click "Insert Column Left" → column added, data shifted right
   - [ ] Click "Insert Column Right" → column added right of selected
   - [ ] Edit cell, then insert column → cell value moves right

3. **Delete Operations:**
   - [ ] Delete empty row → row removed immediately
   - [ ] Delete row with data → confirmation dialog appears
   - [ ] Confirm deletion → row removed, formulas updated
   - [ ] Cancel deletion → row remains

4. **Limits:**
   - [ ] Try to insert 100th row → toast "Maximum rows reached"
   - [ ] Try to insert 26th column → toast "Maximum columns reached"
   - [ ] Try to delete last row → nothing happens (or error toast)

5. **Persistence:**
   - [ ] Insert row → wait for "Saving..." → changes saved
   - [ ] Refresh page → changes persist

6. **Read-only mode:**
   - [ ] Open read-only spreadsheet → context menu doesn't appear

**Step 4: Commit**

```bash
git commit --allow-empty -m "test: complete manual testing of spreadsheet row/column operations

All tests passing:
- Insert/delete rows with data shifting
- Insert/delete columns with data shifting
- Formula reference updates
- Confirmation dialogs for data deletion
- Soft limit enforcement (100 rows, 26 columns)
- Persistence to backend
- Read-only mode respected
"
```

---

## Summary

This implementation plan breaks down the spreadsheet row/column operations feature into 9 tasks:

1. **Core operations** - insertRow, deleteRow, insertColumn, deleteColumn with data shifting
2. **Helper functions** - data detection and formula reference updates
3. **Formula integration** - update formulas when structure changes
4. **Context menu** - right-click UI for operations
5. **Delete confirmation** - safety dialog for data deletion
6. **Grid integration** - connect context menu to SpreadsheetGrid
7. **Parent integration** - connect handlers in DynamicSpreadsheet
8. **User feedback** - toast notifications for limits
9. **Verification** - comprehensive testing and validation

Each task follows TDD: write failing test, implement, verify passing, commit.
