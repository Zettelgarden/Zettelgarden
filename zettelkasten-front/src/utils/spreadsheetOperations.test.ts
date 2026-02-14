import { describe, it, expect } from 'vitest';
import type { Spreadsheet, SpreadsheetData } from '../models/Spreadsheet';
import { insertRow, deleteRow, insertColumn, deleteColumn, MAX_ROWS, MAX_COLS, hasDataInRow, hasDataInColumn, updateFormulaReferences } from './spreadsheetOperations';

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
    expect(result.data.data['A1']).toEqual({ value: '' });
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
    expect(result.data.data['A2']).toEqual({ value: '' });
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
    expect(result.data.data['A4']).toEqual({ value: '' });
  });

  it('should return original spreadsheet if rows >= MAX_ROWS', () => {
    const spreadsheet = createTestSpreadsheet(MAX_ROWS, 2);
    const result = insertRow(spreadsheet, 0);

    expect(result).toEqual(spreadsheet);
    expect(result.data.rows).toBe(MAX_ROWS);
  });
});

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
    expect(result.data.data['A4']).toBeUndefined();
  });

  it('should delete a row at the end', () => {
    const spreadsheet = createTestSpreadsheet(4, 2);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'A3': { value: 'a3', formula: '' },
      'A4': { value: 'a4', formula: '' },
    };

    const result = deleteRow(spreadsheet, 3);

    expect(result.data.rows).toBe(3);
    expect(result.data.data['A1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['A2']).toEqual({ value: 'a2', formula: '' });
    expect(result.data.data['A3']).toEqual({ value: 'a3', formula: '' });
    expect(result.data.data['A4']).toBeUndefined();
  });

  it('should return original spreadsheet when trying to delete the last row', () => {
    const spreadsheet = createTestSpreadsheet(1, 2);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'B1': { value: 'b1', formula: '' },
    };

    const result = deleteRow(spreadsheet, 0);

    expect(result).toEqual(spreadsheet);
    expect(result.data.rows).toBe(1);
  });
});

describe('insertColumn', () => {
  it('should insert a column at the beginning and shift existing data right', () => {
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
    expect(result.data.data['A1']).toEqual({ value: '' });
    expect(result.data.data['B1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['C1']).toEqual({ value: 'b1', formula: '' });
    expect(result.data.data['D1']).toEqual({ value: 'c1', formula: '' });
  });

  it('should insert a column in the middle', () => {
    const spreadsheet = createTestSpreadsheet(2, 3);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'B1': { value: 'b1', formula: '' },
      'C1': { value: 'c1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'B2': { value: 'b2', formula: '' },
      'C2': { value: 'c2', formula: '' },
    };

    const result = insertColumn(spreadsheet, 1);

    expect(result.data.cols).toBe(4);
    expect(result.data.data['A1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['B1']).toEqual({ value: '' });
    expect(result.data.data['C1']).toEqual({ value: 'b1', formula: '' });
    expect(result.data.data['D1']).toEqual({ value: 'c1', formula: '' });
  });

  it('should insert a column at the end', () => {
    const spreadsheet = createTestSpreadsheet(2, 3);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'C1': { value: 'c1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'C2': { value: 'c2', formula: '' },
    };

    const result = insertColumn(spreadsheet, 3);

    expect(result.data.cols).toBe(4);
    expect(result.data.data['A1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['C1']).toEqual({ value: 'c1', formula: '' });
    expect(result.data.data['D1']).toEqual({ value: '' });
  });

  it('should return original spreadsheet if cols >= MAX_COLS', () => {
    const spreadsheet = createTestSpreadsheet(2, MAX_COLS);
    const result = insertColumn(spreadsheet, 0);

    expect(result).toEqual(spreadsheet);
    expect(result.data.cols).toBe(MAX_COLS);
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
      'A2': { value: 'a2', formula: '' },
      'B2': { value: 'b2', formula: '' },
      'C2': { value: 'c2', formula: '' },
      'D2': { value: 'd2', formula: '' },
    };

    const result = deleteColumn(spreadsheet, 0);

    expect(result.data.cols).toBe(3);
    expect(result.data.data['A1']).toEqual({ value: 'b1', formula: '' });
    expect(result.data.data['B1']).toEqual({ value: 'c1', formula: '' });
    expect(result.data.data['C1']).toEqual({ value: 'd1', formula: '' });
    expect(result.data.data['D1']).toBeUndefined();
  });

  it('should delete a column in the middle', () => {
    const spreadsheet = createTestSpreadsheet(2, 4);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'B1': { value: 'b1', formula: '' },
      'C1': { value: 'c1', formula: '' },
      'D1': { value: 'd1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'B2': { value: 'b2', formula: '' },
      'C2': { value: 'c2', formula: '' },
      'D2': { value: 'd2', formula: '' },
    };

    const result = deleteColumn(spreadsheet, 1);

    expect(result.data.cols).toBe(3);
    expect(result.data.data['A1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['B1']).toEqual({ value: 'c1', formula: '' });
    expect(result.data.data['C1']).toEqual({ value: 'd1', formula: '' });
    expect(result.data.data['D1']).toBeUndefined();
  });

  it('should delete a column at the end', () => {
    const spreadsheet = createTestSpreadsheet(2, 4);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'B1': { value: 'b1', formula: '' },
      'C1': { value: 'c1', formula: '' },
      'D1': { value: 'd1', formula: '' },
      'A2': { value: 'a2', formula: '' },
      'B2': { value: 'b2', formula: '' },
      'C2': { value: 'c2', formula: '' },
      'D2': { value: 'd2', formula: '' },
    };

    const result = deleteColumn(spreadsheet, 3);

    expect(result.data.cols).toBe(3);
    expect(result.data.data['A1']).toEqual({ value: 'a1', formula: '' });
    expect(result.data.data['B1']).toEqual({ value: 'b1', formula: '' });
    expect(result.data.data['C1']).toEqual({ value: 'c1', formula: '' });
    expect(result.data.data['D1']).toBeUndefined();
  });

  it('should return original spreadsheet when trying to delete the last column', () => {
    const spreadsheet = createTestSpreadsheet(2, 1);
    spreadsheet.data.data = {
      'A1': { value: 'a1', formula: '' },
      'A2': { value: 'a2', formula: '' },
    };

    const result = deleteColumn(spreadsheet, 0);

    expect(result).toEqual(spreadsheet);
    expect(result.data.cols).toBe(1);
  });
});

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
      value: '',
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

    expect(result.data.data['B1']).toEqual({
      value: '',
      formula: 'A1+A4'
    });
  });

  it('should update formulas when inserting row at end', () => {
    const spreadsheet = createTestSpreadsheet(3, 2);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'A2': { value: '2', formula: '' },
      'A3': { value: '3', formula: '' },
      'B1': { value: '', formula: 'SUM(A1:A3)' },
    };

    const result = insertRow(spreadsheet, 3);

    expect(result.data.data['B1']).toEqual({
      value: '',
      formula: 'SUM(A1:A3)'
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

  it('should update formulas when deleting first row', () => {
    const spreadsheet = createTestSpreadsheet(4, 2);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'A2': { value: '2', formula: '' },
      'A3': { value: '3', formula: '' },
      'A4': { value: '4', formula: '' },
      'B2': { value: '', formula: 'SUM(A1:A4)' },
    };

    const result = deleteRow(spreadsheet, 0);

    expect(result.data.data['B1']).toEqual({
      value: '',
      formula: 'SUM(#REF!:A3)'
    });
  });

  it('should update formulas when deleting last row', () => {
    const spreadsheet = createTestSpreadsheet(4, 2);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'A2': { value: '2', formula: '' },
      'A3': { value: '3', formula: '' },
      'A4': { value: '4', formula: '' },
      'B1': { value: '', formula: 'SUM(A1:A4)' },
    };

    const result = deleteRow(spreadsheet, 3);

    expect(result.data.data['B1']).toEqual({
      value: '',
      formula: 'SUM(A1:#REF!)'
    });
  });
});

describe('insertColumn with formulas', () => {
  it('should update formula references when inserting column', () => {
    const spreadsheet = createTestSpreadsheet(2, 3);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'B1': { value: '2', formula: '' },
      'C1': { value: '3', formula: '' },
      'A2': { value: '', formula: 'A1+B1' }, // should become B1+C1
    };

    const result = insertColumn(spreadsheet, 0);

    expect(result.data.data['B2']).toEqual({
      value: '',
      formula: 'B1+C1'
    });
  });

  it('should update formulas when inserting column in middle', () => {
    const spreadsheet = createTestSpreadsheet(2, 3);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'B1': { value: '2', formula: '' },
      'C1': { value: '3', formula: '' },
      'A2': { value: '', formula: 'A1+C1' },
    };

    const result = insertColumn(spreadsheet, 1);

    expect(result.data.data['A2']).toEqual({
      value: '',
      formula: 'A1+D1'
    });
  });
});

describe('deleteColumn with formulas', () => {
  it('should update formula references and mark deleted refs', () => {
    const spreadsheet = createTestSpreadsheet(2, 3);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'B1': { value: '2', formula: '' },
      'C1': { value: '3', formula: '' },
      'A2': { value: '', formula: 'A1+B1+C1' },
    };

    const result = deleteColumn(spreadsheet, 1); // Delete column B (index 1)

    expect(result.data.data['A2']).toEqual({
      value: '',
      formula: 'A1+#REF!+B1' // C1 becomes B1, B1 was deleted
    });
  });

  it('should update formulas when deleting first column', () => {
    const spreadsheet = createTestSpreadsheet(2, 3);
    spreadsheet.data.data = {
      'A1': { value: '1', formula: '' },
      'B1': { value: '2', formula: '' },
      'C1': { value: '3', formula: '' },
      'D1': { value: '', formula: 'SUM(A1:C1)' },
    };

    const result = deleteColumn(spreadsheet, 0);

    expect(result.data.data['C1']).toEqual({
      value: '',
      formula: 'SUM(#REF!:B1)'
    });
  });
});

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

  it('should return false for invalid row index', () => {
    const spreadsheet = createTestSpreadsheet(2, 2);
    spreadsheet.data.data = {
      'A1': { value: 'data', formula: '' },
    };

    expect(hasDataInRow(spreadsheet, -1)).toBe(false);
    expect(hasDataInRow(spreadsheet, 5)).toBe(false);
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

  it('should return false for invalid column index', () => {
    const spreadsheet = createTestSpreadsheet(2, 2);
    spreadsheet.data.data = {
      'A1': { value: 'data', formula: '' },
    };

    expect(hasDataInColumn(spreadsheet, -1)).toBe(false);
    expect(hasDataInColumn(spreadsheet, 5)).toBe(false);
  });
});

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
    expect(result).toBe('SUM(B2,C3)+D4');
  });

  it('should handle both row and column deltas', () => {
    const formula = 'A1+B2';
    const result = updateFormulaReferences(formula, 1, 1, 0);
    expect(result).toBe('B2+C3');
  });

  it('should not update references before position when inserting', () => {
    const formula = 'A1+B2';
    const result = updateFormulaReferences(formula, 1, 0, 2);
    expect(result).toBe('A1+B3');
  });

  it('should not update references at or before position when deleting', () => {
    const formula = 'A1+B2';
    const result = updateFormulaReferences(formula, -1, 0, 1);
    expect(result).toBe('A1+B1');
  });

  it('should return #REF! for references that go out of bounds', () => {
    const formula = 'A100';
    const result = updateFormulaReferences(formula, 1, 0, 100);
    expect(result).toBe('#REF!');
  });

  it('should handle empty formula', () => {
    const result = updateFormulaReferences('', 1, 0, 0);
    expect(result).toBe('');
  });

  it('should handle mixed operations in formula', () => {
    const formula = '(A1+B2)*C3-SUM(D4:E5)';
    const result = updateFormulaReferences(formula, 1, 1, 0);
    expect(result).toBe('(B2+C3)*D4-SUM(E5:F6)');
  });
});
