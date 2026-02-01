import { describe, it, expect } from 'vitest';
import { a1ToCoords, coordsToA1, parseCellReference, parseRange, getCellsInRange, type CellRange } from './spreadsheetHelpers';

describe('spreadsheetHelpers - a1ToCoords', () => {
  it('converts A1 to {row: 0, col: 0}', () => {
    expect(a1ToCoords('A1')).toEqual({ row: 0, col: 0 });
  });

  it('converts B5 to {row: 4, col: 1}', () => {
    expect(a1ToCoords('B5')).toEqual({ row: 4, col: 1 });
  });

  it('converts Z100 to {row: 99, col: 25}', () => {
    expect(a1ToCoords('Z100')).toEqual({ row: 99, col: 25 });
  });

  it('handles multi-letter columns like AA1', () => {
    expect(a1ToCoords('AA1')).toEqual({ row: 0, col: 26 });
  });

  it('throws on invalid format', () => {
    expect(() => a1ToCoords('INVALID')).toThrow();
  });
});

describe('spreadsheetHelpers - coordsToA1', () => {
  it('converts {row: 0, col: 0} to A1', () => {
    expect(coordsToA1(0, 0)).toBe('A1');
  });

  it('converts {row: 4, col: 1} to B5', () => {
    expect(coordsToA1(4, 1)).toBe('B5');
  });

  it('converts to multi-letter columns when needed', () => {
    expect(coordsToA1(0, 26)).toBe('AA1');
    expect(coordsToA1(0, 27)).toBe('AB1');
  });
});

describe('spreadsheetHelpers - parseCellReference', () => {
  it('parses single cell reference A1', () => {
    expect(parseCellReference('A1')).toEqual({ startRow: 0, startCol: 0, endRow: 0, endCol: 0 });
  });

  it('parses single cell reference Z99', () => {
    expect(parseCellReference('Z99')).toEqual({ startRow: 98, startCol: 25, endRow: 98, endCol: 25 });
  });
});

describe('spreadsheetHelpers - parseRange', () => {
  it('parses range A1:A5', () => {
    expect(parseRange('A1:A5')).toEqual({ startRow: 0, startCol: 0, endRow: 4, endCol: 0 });
  });

  it('parses range A1:B3', () => {
    expect(parseRange('A1:B3')).toEqual({ startRow: 0, startCol: 0, endRow: 2, endCol: 1 });
  });

  it('parses range B5:D10', () => {
    expect(parseRange('B5:D10')).toEqual({ startRow: 4, startCol: 1, endRow: 9, endCol: 3 });
  });

  it('handles single cell as range', () => {
    expect(parseRange('A1')).toEqual({ startRow: 0, startCol: 0, endRow: 0, endCol: 0 });
  });
});

describe('spreadsheetHelpers - getCellsInRange', () => {
  it('returns single cell for A1:A1', () => {
    expect(getCellsInRange('A1:A1')).toEqual(['A1']);
  });

  it('returns cells in row range A1:C1', () => {
    expect(getCellsInRange('A1:C1')).toEqual(['A1', 'B1', 'C1']);
  });

  it('returns cells in 2D range A1:B2', () => {
    expect(getCellsInRange('A1:B2')).toEqual(['A1', 'B1', 'A2', 'B2']);
  });

  it('handles reverse range B2:A1', () => {
    expect(getCellsInRange('B2:A1')).toEqual(['A1', 'B1', 'A2', 'B2']);
  });
});
