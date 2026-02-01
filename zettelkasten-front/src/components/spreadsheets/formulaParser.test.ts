import { describe, it, expect } from 'vitest';
import { evaluateFormula, tokenizeFormula } from './formulaParser';
import { SpreadsheetCell } from '../../models/Spreadsheet';

describe('formulaParser - tokenizeFormula', () => {
  it('tokenizes simple arithmetic', () => {
    expect(tokenizeFormula('A1+B1')).toEqual([
      { type: 'cell', value: 'A1' },
      { type: 'operator', value: '+' },
      { type: 'cell', value: 'B1' }
    ]);
  });

  it('tokenizes numbers and operators', () => {
    expect(tokenizeFormula('10+20*5')).toEqual([
      { type: 'number', value: '10' },
      { type: 'operator', value: '+' },
      { type: 'number', value: '20' },
      { type: 'operator', value: '*' },
      { type: 'number', value: '5' }
    ]);
  });

  it('tokenizes SUM function', () => {
    expect(tokenizeFormula('SUM(A1:A5)')).toEqual([
      { type: 'function', value: 'SUM' },
      { type: 'paren', value: '(' },
      { type: 'range', value: 'A1:A5' },
      { type: 'paren', value: ')' }
    ]);
  });

  it('tokenizes complex formula', () => {
    expect(tokenizeFormula('SUM(A1:A5)*AVERAGE(B1:B3)')).toEqual([
      { type: 'function', value: 'SUM' },
      { type: 'paren', value: '(' },
      { type: 'range', value: 'A1:A5' },
      { type: 'paren', value: ')' },
      { type: 'operator', value: '*' },
      { type: 'function', value: 'AVERAGE' },
      { type: 'paren', value: '(' },
      { type: 'range', value: 'B1:B3' },
      { type: 'paren', value: ')' }
    ]);
  });
});

describe('formulaParser - evaluateFormula', () => {
  const cellData: Record<string, SpreadsheetCell> = {
    'A1': { value: '10', formula: '' },
    'A2': { value: '20', formula: '' },
    'A3': { value: '30', formula: '' },
    'B1': { value: '5', formula: '' },
    'B2': { value: '15', formula: '' },
    'C1': { value: 'abc', formula: '' }, // Non-numeric
  };

  it('evaluates simple addition', () => {
    expect(evaluateFormula('A1+A2', cellData)).toBe(30);
  });

  it('evaluates subtraction', () => {
    expect(evaluateFormula('A2-A1', cellData)).toBe(10);
  });

  it('evaluates multiplication', () => {
    expect(evaluateFormula('A1*B1', cellData)).toBe(50);
  });

  it('evaluates division', () => {
    expect(evaluateFormula('A2/A1', cellData)).toBe(2);
  });

  it('evaluates SUM function', () => {
    expect(evaluateFormula('SUM(A1:A3)', cellData)).toBe(60);
  });

  it('evaluates AVERAGE function', () => {
    expect(evaluateFormula('AVERAGE(A1:A3)', cellData)).toBe(20);
  });

  it('evaluates COUNT function (numeric only)', () => {
    expect(evaluateFormula('COUNT(A1:A3)', cellData)).toBe(3);
  });

  it('COUNT ignores non-numeric values', () => {
    expect(evaluateFormula('COUNT(A1:C1)', cellData)).toBe(2); // 10, 5, abc -> counts 2
  });

  it('evaluates complex expression with parentheses', () => {
    expect(evaluateFormula('(A1+A2)*B1', cellData)).toBe(150); // (10+20)*5
  });

  it('evaluates operator precedence', () => {
    expect(evaluateFormula('A1+A2*B1', cellData)).toBe(110); // 10 + (20*5)
  });

  it('returns null for invalid cell reference', () => {
    expect(evaluateFormula('Z999', cellData)).toBeNull();
  });

  it('returns null for division by zero', () => {
    expect(evaluateFormula('A1/0', cellData)).toBeNull();
  });

  it('handles negative numbers', () => {
    expect(evaluateFormula('A1*-5', cellData)).toBe(-50);
  });

  it('handles decimal numbers', () => {
    expect(evaluateFormula('A1*1.5', cellData)).toBe(15);
  });
});
