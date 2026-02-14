import { describe, it, expect } from 'vitest';
import type { Spreadsheet, SpreadsheetData } from '../models/Spreadsheet';
import { insertRow, deleteRow, insertColumn, deleteColumn, MAX_ROWS, MAX_COLS } from './spreadsheetOperations';

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
