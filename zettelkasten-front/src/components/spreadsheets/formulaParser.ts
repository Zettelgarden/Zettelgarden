import { SpreadsheetCell } from '../../models/Spreadsheet';
import { parseCellReference, getCellsInRange } from '../../utils/spreadsheetHelpers';

export type Token =
  | { type: 'cell'; value: string }
  | { type: 'range'; value: string }
  | { type: 'number'; value: string }
  | { type: 'operator'; value: string }
  | { type: 'function'; value: string }
  | { type: 'paren'; value: string };

/**
 * Tokenize a formula string into tokens
 */
export function tokenizeFormula(formula: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;

  while (i < formula.length) {
    const char = formula[i];

    // Skip whitespace
    if (/\s/.test(char)) {
      i++;
      continue;
    }

    // Numbers (including decimals and negative at start or after operator)
    const prevToken = tokens[tokens.length - 1];
    const isNegative = char === '-' && (i === 0 || (prevToken?.type === 'operator'));
    if (/\d/.test(char) || isNegative) {
      let num = char;
      i++;
      while (i < formula.length && /[\d.]/.test(formula[i])) {
        num += formula[i];
        i++;
      }
      tokens.push({ type: 'number', value: num });
      continue;
    }

    // Operators
    if (['+', '-', '*', '/', '='].includes(char)) {
      tokens.push({ type: 'operator', value: char });
      i++;
      continue;
    }

    // Parentheses
    if (char === '(' || char === ')') {
      tokens.push({ type: 'paren', value: char });
      i++;
      continue;
    }

    // Cell references, ranges, or functions
    if (/[A-Z]/.test(char)) {
      let ident = char;
      i++;
      while (i < formula.length && /[\w\d:]/.test(formula[i])) {
        ident += formula[i];
        i++;
      }

      // Check if it's a function (followed by paren)
      if (i < formula.length && formula[i] === '(') {
        tokens.push({ type: 'function', value: ident });
      } else if (ident.includes(':')) {
        tokens.push({ type: 'range', value: ident });
      } else if (/^[A-Z]+\d+$/.test(ident)) {
        tokens.push({ type: 'cell', value: ident });
      }
      continue;
    }

    // Unknown character
    i++;
  }

  return tokens;
}

/**
 * Get the numeric value of a cell, or null if not numeric
 */
function getCellValue(cellRef: string, cellData: Record<string, SpreadsheetCell>): number | null {
  const cell = cellData[cellRef];
  if (!cell) return null;

  const value = cell.value.trim();
  if (value === '') return null;

  const num = parseFloat(value);
  return isNaN(num) ? null : num;
}

/**
 * Count numeric values in a range
 */
function countValues(range: string, cellData: Record<string, SpreadsheetCell>): number {
  const cells = getCellsInRange(range);
  let count = 0;

  for (const cellRef of cells) {
    if (getCellValue(cellRef, cellData) !== null) {
      count++;
    }
  }

  return count;
}

/**
 * Sum values in a range
 */
function sumValues(range: string, cellData: Record<string, SpreadsheetCell>): number {
  const cells = getCellsInRange(range);
  let sum = 0;

  for (const cellRef of cells) {
    const value = getCellValue(cellRef, cellData);
    if (value !== null) {
      sum += value;
    }
  }

  return sum;
}

/**
 * Average values in a range
 */
function averageValues(range: string, cellData: Record<string, SpreadsheetCell>): number {
  const cells = getCellsInRange(range);
  let sum = 0;
  let count = 0;

  for (const cellRef of cells) {
    const value = getCellValue(cellRef, cellData);
    if (value !== null) {
      sum += value;
      count++;
    }
  }

  return count > 0 ? sum / count : 0;
}

/**
 * Evaluate a formula against cell data
 * Returns the computed number or null if error
 */
export function evaluateFormula(
  formula: string,
  cellData: Record<string, SpreadsheetCell>
): number | null {
  try {
    const tokens = tokenizeFormula(formula);

    // First pass: replace functions and ranges with their computed values
    const evaluated: Array<Token | { type: 'computed'; value: number }> = [];

    for (let i = 0; i < tokens.length; i++) {
      const token = tokens[i];

      if (token.type === 'function') {
        const nextToken = tokens[i + 1];
        if (nextToken?.type === 'paren' && nextToken.value === '(') {
          // Find the closing paren and extract range
          let j = i + 2;
          let rangeContent = '';
          let parenDepth = 1;

          while (j < tokens.length && parenDepth > 0) {
            if (tokens[j].type === 'paren') {
              if (tokens[j].value === '(') parenDepth++;
              else if (tokens[j].value === ')') parenDepth--;
            } else if (parenDepth === 1) {
              rangeContent += tokens[j].value;
            }
            j++;
          }

          // Evaluate function
          let result: number;
          switch (token.value.toUpperCase()) {
            case 'SUM':
              result = sumValues(rangeContent, cellData);
              break;
            case 'AVERAGE':
              result = averageValues(rangeContent, cellData);
              break;
            case 'COUNT':
              result = countValues(rangeContent, cellData);
              break;
            default:
              return null; // Unknown function
          }

          evaluated.push({ type: 'computed', value: result });
          i = j - 1; // Skip to after the closing paren
        } else {
          return null;
        }
      } else if (token.type === 'range') {
        // Range outside function - treat as sum of values
        const sum = sumValues(token.value, cellData);
        evaluated.push({ type: 'computed', value: sum });
      } else {
        evaluated.push(token);
      }
    }

    // Evaluate parentheses first (recursive descent)
    function evaluateSubexpression(tokens: typeof evaluated): number | null {
      // Find and evaluate parenthesized sub-expressions first
      let parenOpen = -1;
      for (let i = 0; i < tokens.length; i++) {
        if (tokens[i].type === 'paren' && (tokens[i] as Token).value === '(') {
          parenOpen = i;
        } else if (tokens[i].type === 'paren' && (tokens[i] as Token).value === ')') {
          if (parenOpen >= 0) {
            const subTokens = tokens.slice(parenOpen + 1, i);
            const result = evaluateWithPrecedence(subTokens);
            if (result === null) return null;
            // Replace parenthesized expression with result
            const newTokens = tokens.slice(0, parenOpen).concat({ type: 'computed', value: result }).concat(tokens.slice(i + 1));
            return evaluateSubexpression(newTokens);
          }
        }
      }
      return evaluateWithPrecedence(tokens);
    }

    function evaluateWithPrecedence(tokens: typeof evaluated): number | null {
      // First pass: handle * and /
      let pass1: Array<number | string> = [];
      for (let i = 0; i < tokens.length; i++) {
        const token = tokens[i];

        if (token.type === 'computed') {
          pass1.push(token.value);
        } else if (token.type === 'number') {
          pass1.push(parseFloat(token.value));
        } else if (token.type === 'cell') {
          const val = getCellValue(token.value, cellData);
          if (val === null) return null;
          pass1.push(val);
        } else if (token.type === 'operator') {
          if (token.value === '*' || token.value === '/') {
            const left = pass1[pass1.length - 1] as number;
            const nextToken = tokens[++i];
            let right: number | null = null;

            if (nextToken.type === 'computed') {
              right = nextToken.value;
            } else if (nextToken.type === 'number') {
              right = parseFloat(nextToken.value);
            } else if (nextToken.type === 'cell') {
              const val = getCellValue(nextToken.value, cellData);
              if (val === null) return null;
              right = val;
            } else if (nextToken.type === 'operator' && nextToken.value === '-' && i + 1 < tokens.length) {
              // Handle negative number after operator
              const nextNextToken = tokens[++i];
              if (nextNextToken.type === 'number') {
                right = -parseFloat(nextNextToken.value);
              } else if (nextNextToken.type === 'cell') {
                const val = getCellValue(nextNextToken.value, cellData);
                if (val === null) return null;
                right = -val;
              }
            }

            if (right === null) return null;

            if (token.value === '*') {
              pass1[pass1.length - 1] = left * right;
            } else {
              if (right === 0) return null;
              pass1[pass1.length - 1] = left / right;
            }
          } else {
            // Check for unary minus
            if (token.value === '-' && pass1.length === 0) {
              pass1.push('0'); // Add zero for unary minus
            }
            pass1.push(token.value);
          }
        }
      }

      if (pass1.length === 0) return null;

      // Second pass: handle + and -
      let result = pass1[0] as number;
      for (let i = 1; i < pass1.length; i += 2) {
        const op = pass1[i] as string;
        const val = pass1[i + 1] as number;

        if (op === '+') {
          result += val;
        } else if (op === '-') {
          result -= val;
        }
      }

      return result;
    }

    return evaluateSubexpression(evaluated);
  } catch {
    return null;
  }
}

/**
 * Parse a cell value (determine if it's a formula or plain value)
 */
export function parseCellValue(input: string): { value: string; formula: string } {
  const trimmed = input.trim();

  if (trimmed.startsWith('=')) {
    return {
      value: '', // Will be computed
      formula: trimmed.slice(1)
    };
  }

  return {
    value: trimmed,
    formula: ''
  };
}

/**
 * Extract all cell references from a formula string
 * Returns array of cell references like ['A1', 'B2']
 */
export function extractCellReferences(formula: string): string[] {
  const tokens = tokenizeFormula(formula);
  const refs: string[] = [];

  for (const token of tokens) {
    if (token.type === 'cell') {
      refs.push(token.value);
    } else if (token.type === 'range') {
      // Expand range to individual cells
      const cells = getCellsInRange(token.value);
      refs.push(...cells);
    }
  }

  return refs;
}
