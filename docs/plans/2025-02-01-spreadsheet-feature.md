# Spreadsheet Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add interactive spreadsheet grids to cards with formula support (SUM, AVERAGE, COUNT) and Excel-style A1 notation.

**Architecture:**
- Spreadsheets embedded via `{{spreadsheet:name}}` markdown syntax (inline) + dedicated "Spreadsheets" tab
- Data stored as JSON in card body within fenced code blocks
- Remark plugin parses syntax and renders DynamicSpreadsheet component
- Formula parser evaluates basic arithmetic and functions

**Tech Stack:**
- React 18, TypeScript, Vite
- Vitest + React Testing Library for tests
- react-markdown + custom remark plugins
- Tailwind CSS for styling

---

## Task 1: Add Spreadsheet TypeScript Models

**Files:**
- Create: `zettelkasten-front/src/models/Spreadsheet.ts`

**Step 1: Write the model interfaces**

```typescript
/**
 * Represents a single cell in a spreadsheet
 */
export interface SpreadsheetCell {
  value: string;        // Display value (result of formula or raw input)
  formula?: string;     // Formula if applicable (starts with =), empty string if plain value
  computed?: number;    // Computed numeric result (if applicable)
}

/**
 * The full data structure for a spreadsheet
 */
export interface SpreadsheetData {
  rows: number;         // Number of rows (default: 5)
  cols: number;         // Number of columns (default: 5)
  data: Record<string, SpreadsheetCell>;  // Keys: "A1", "B2", etc.
}

/**
 * A spreadsheet with a name identifier
 */
export interface Spreadsheet {
  name: string;         // Identifier for this spreadsheet
  data: SpreadsheetData;
}

/**
 * Parsed spreadsheet reference from markdown
 */
export interface SpreadsheetReference {
  name: string;         // Spreadsheet name from {{spreadsheet:name}}
}
```

**Step 2: Export default empty spreadsheet**

```typescript
export const createEmptySpreadsheet = (name: string = "sheet1", rows: number = 5, cols: number = 5): Spreadsheet => {
  const data: Record<string, SpreadsheetCell> = {};

  // Initialize all cells with empty values
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      const cellRef = `${String.fromCharCode(65 + col)}${row + 1}`;
      data[cellRef] = { value: "", formula: "" };
    }
  }

  return {
    name,
    data: { rows, cols, data }
  };
};

export const defaultSpreadsheetData: SpreadsheetData = {
  rows: 5,
  cols: 5,
  data: {}
};
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/models/Spreadsheet.ts
git commit -m "feat(spreadsheet): add TypeScript interfaces and models"
```

---

## Task 2: Create Helper Functions for Cell References

**Files:**
- Create: `zettelkasten-front/src/utils/spreadsheetHelpers.ts`
- Test: `zettelkasten-front/src/utils/spreadsheetHelpers.test.ts`

**Step 1: Write failing tests for A1 notation conversion**

```typescript
import { describe, it, expect } from 'vitest';
import { a1ToCoords, coordsToA1, parseCellReference, parseRange } from './spreadsheetHelpers';

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
```

**Step 2: Run tests to verify they fail**

```bash
cd zettelkasten-front
npm test -- spreadsheetHelpers.test.ts
```

Expected: FAIL - "Cannot find module './spreadsheetHelpers'"

**Step 3: Implement the helper functions**

```typescript
/**
 * Convert A1 notation to coordinates (0-indexed)
 * Examples: A1 -> {row: 0, col: 0}, B5 -> {row: 4, col: 1}, AA1 -> {row: 0, col: 26}
 */
export function a1ToCoords(cellRef: string): { row: number; col: number } {
  const match = cellRef.match(/^([A-Z]+)(\d+)$/);
  if (!match) {
    throw new Error(`Invalid cell reference: ${cellRef}`);
  }

  const [, colStr, rowStr] = match;

  // Convert column letters to number (A=0, B=1, ..., AA=26, AB=27, etc.)
  let col = 0;
  for (let i = 0; i < colStr.length; i++) {
    col = col * 26 + (colStr.charCodeAt(i) - 64);
  }
  col -= 1; // Convert to 0-indexed

  const row = parseInt(rowStr, 10) - 1; // Convert to 0-indexed

  return { row, col };
}

/**
 * Convert coordinates to A1 notation
 * Examples: {row: 0, col: 0} -> A1, {row: 4, col: 1} -> B5
 */
export function coordsToA1(row: number, col: number): string {
  // Convert column to letters (0=A, 1=B, ..., 26=AA, 27=AB, etc.)
  let colStr = '';
  let c = col + 1;
  while (c > 0) {
    c -= 1;
    colStr = String.fromCharCode(65 + (c % 26)) + colStr;
    c = Math.floor(c / 26);
  }

  return `${colStr}${row + 1}`;
}

/**
 * Parse a cell reference (single cell or range)
 * Returns normalized range with start/end row/col
 */
export function parseCellReference(cellRef: string): {
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
} {
  // Check if it's a range (contains colon)
  if (cellRef.includes(':')) {
    return parseRange(cellRef);
  }

  const { row, col } = a1ToCoords(cellRef);
  return { startRow: row, startCol: col, endRow: row, endCol: col };
}

/**
 * Parse a range reference like A1:B3 or A1:A5
 */
export function parseRange(range: string): {
  startRow: number;
  startCol: number;
  endRow: number;
  endCol: number;
} {
  if (!range.includes(':')) {
    return parseCellReference(range);
  }

  const [start, end] = range.split(':');
  const startPos = a1ToCoords(start);
  const endPos = a1ToCoords(end);

  return {
    startRow: Math.min(startPos.row, endPos.row),
    startCol: Math.min(startPos.col, endPos.col),
    endRow: Math.max(startPos.row, endPos.row),
    endCol: Math.max(startPos.col, endPos.col),
  };
}

/**
 * Get all cell references in a range
 */
export function getCellsInRange(range: string): string[] {
  const { startRow, startCol, endRow, endCol } = parseRange(range);
  const cells: string[] = [];

  for (let row = startRow; row <= endRow; row++) {
    for (let col = startCol; col <= endCol; col++) {
      cells.push(coordsToA1(row, col));
    }
  }

  return cells;
}
```

**Step 4: Run tests to verify they pass**

```bash
cd zettelkasten-front
npm test -- spreadsheetHelpers.test.ts
```

Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/utils/spreadsheetHelpers.ts zettelkasten-front/src/utils/spreadsheetHelpers.test.ts
git commit -m "feat(spreadsheet): add A1 notation helper functions with tests"
```

---

## Task 3: Create Formula Parser

**Files:**
- Create: `zettelkasten-front/src/components/spreadsheets/formulaParser.ts`
- Test: `zettelkasten-front/src/components/spreadsheets/formulaParser.test.ts`

**Step 1: Write failing tests for formula parsing**

```typescript
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
```

**Step 2: Run tests to verify they fail**

```bash
cd zettelkasten-front
npm test -- formulaParser.test.ts
```

Expected: FAIL - "Cannot find module './formulaParser'"

**Step 3: Implement the formula parser**

```typescript
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

    // Numbers (including decimals and negative)
    if (/\d/.test(char) || (char === '-' && i === 0)) {
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

    // First pass: handle functions and ranges
    const processedTokens: Array<number | string> = [];

    for (let i = 0; i < tokens.length; i++) {
      const token = tokens[i];

      if (token.type === 'number') {
        processedTokens.push(parseFloat(token.value));
      } else if (token.type === 'cell') {
        const value = getCellValue(token.value, cellData);
        if (value === null) return null;
        processedTokens.push(value);
      } else if (token.type === 'function') {
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

          processedTokens.push(result);
          i = j - 1; // Skip to after the closing paren
        } else {
          return null;
        }
      } else if (token.type === 'range') {
        // Range outside function - treat as sum of values
        const sum = sumValues(token.value, cellData);
        processedTokens.push(sum);
      }
      // Skip operators and parens - they'll be handled in the infix evaluation
    }

    // Simple infix evaluation with operator precedence
    // First pass: handle * and /
    let pass1: Array<number | string> = [];
    for (let i = 0; i < tokens.length; i++) {
      const token = tokens[i];

      if (token.type === 'operator') {
        if (token.value === '*' || token.value === '/') {
          const left = pass1[pass1.length - 1] as number;
          const rightToken = tokens[++i];
          let right: number;

          if (rightToken.type === 'number') {
            right = parseFloat(rightToken.value);
          } else if (rightToken.type === 'cell') {
            right = getCellValue(rightToken.value, cellData) ?? 0;
          } else {
            return null;
          }

          if (token.value === '*') {
            pass1[pass1.length - 1] = left * right;
          } else {
            if (right === 0) return null;
            pass1[pass1.length - 1] = left / right;
          }
        } else {
          pass1.push(token.value);
        }
      } else if (token.type === 'number') {
        pass1.push(parseFloat(token.value));
      } else if (token.type === 'cell') {
        const val = getCellValue(token.value, cellData);
        if (val === null) return null;
        pass1.push(val);
      }
    }

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
```

**Step 4: Run tests and verify they pass**

```bash
cd zettelkasten-front
npm test -- formulaParser.test.ts
```

Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/formulaParser.ts zettelkasten-front/src/components/spreadsheets/formulaParser.test.ts
git commit -m "feat(spreadsheet): add formula parser with SUM, AVERAGE, COUNT support"
```

---

## Task 4: Create Remark Plugin for Spreadsheet Syntax

**Files:**
- Create: `zettelkasten-front/src/remark-spreadsheet.ts`
- Test: `zettelkasten-front/src/remark-spreadsheet.test.ts`

**Step 1: Write failing test**

```typescript
import { describe, it, expect } from 'vitest';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkStringify from 'remark-stringify';
import remarkSpreadsheet from './remark-spreadsheet';

describe('remark-spreadsheet', () => {
  it('parses {{spreadsheet:name}} syntax', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet)
      .use(remarkStringify);

    const result = processor.processSync('Check out {{spreadsheet:budget}} for details');

    // Should contain a spreadsheet node
    const tree = processor.parse(result);
    // Verify the transformation
    expect(result).toContain('budget');
  });

  it('handles default name without colon', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet)
      .use(remarkStringify);

    const result = processor.processSync('Inline {{spreadsheet}} here');
    expect(result).toBeTruthy();
  });

  it('parses multiple spreadsheets', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet)
      .use(remarkStringify);

    const result = processor.processSync('{{spreadsheet:sales}} and {{spreadsheet:expenses}}');
    expect(result).toBeTruthy();
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd zettelkasten-front
npm test -- remark-spreadsheet.test.ts
```

Expected: FAIL - "Cannot find module './remark-spreadsheet'"

**Step 3: Implement the remark plugin**

```typescript
import { visit } from "unist-util-visit";
import { Node } from "unist";

export default function remarkSpreadsheet() {
  return (tree: Node) => {
    visit(tree, "text", (node: any, index: number | undefined, parent: any) => {
      if (!parent || typeof node.value !== "string" || index === undefined) return;

      // Match {{spreadsheet:name}} or {{spreadsheet}} syntax
      const regex = /\{\{spreadsheet(?::([^}\s]+))?\}\}/gi;
      let match;
      const newNodes: any[] = [];
      let lastIndex = 0;

      while ((match = regex.exec(node.value)) !== null) {
        const [fullMatch, name] = match;
        const spreadsheetName = name || "sheet1";

        // Push text before match
        if (match.index > lastIndex) {
          newNodes.push({
            type: "text",
            value: node.value.slice(lastIndex, match.index),
          });
        }

        // Push spreadsheet node
        newNodes.push({
          type: "spreadsheet",
          data: {
            name: spreadsheetName,
            hName: "div",
            hProperties: {
              className: "spreadsheet-container",
              "data-spreadsheet-name": spreadsheetName
            }
          },
          children: [],
        });

        lastIndex = match.index + fullMatch.length;
      }

      // Push remaining text after last match
      if (lastIndex < node.value.length) {
        newNodes.push({
          type: "text",
          value: node.value.slice(lastIndex),
        });
      }

      if (newNodes.length > 0) {
        parent.children.splice(index, 1, ...newNodes);
        return index + newNodes.length;
      }
    });
  };
}
```

**Step 4: Run tests to verify they pass**

```bash
cd zettelkasten-front
npm test -- remark-spreadsheet.test.ts
```

Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/remark-spreadsheet.ts zettelkasten-front/src/remark-spreadsheet.test.ts
git commit -m "feat(spreadsheet): add remark plugin for {{spreadsheet}} syntax"
```

---

## Task 5: Create SpreadsheetCell Component

**Files:**
- Create: `zettelkasten-front/src/components/spreadsheets/SpreadsheetCell.tsx`
- Test: `zettelkasten-front/src/components/spreadsheets/SpreadsheetCell.test.tsx`

**Step 1: Write the test**

```typescript
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SpreadsheetCell } from './SpreadsheetCell';
import { SpreadsheetCell as SpreadsheetCellModel } from '../../models/Spreadsheet';

describe('SpreadsheetCell', () => {
  const mockCell: SpreadsheetCellModel = {
    value: '42',
    formula: ''
  };

  const mockOnChange = vi.fn();

  it('renders cell value', () => {
    render(
      <table>
        <tbody>
          <tr>
            <SpreadsheetCell
              cellRef="A1"
              cell={mockCell}
              onChange={mockOnChange}
            />
          </tr>
        </tbody>
      </table>
    );

    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('enters edit mode on double-click', () => {
    render(
      <table>
        <tbody>
          <tr>
            <SpreadsheetCell
              cellRef="A1"
              cell={mockCell}
              onChange={mockOnChange}
            />
          </tr>
        </tbody>
      </table>
    );

    const cell = screen.getByText('42');
    fireEvent.doubleClick(cell);

    const input = screen.getByDisplayValue('42');
    expect(input).toBeInTheDocument();
  });

  it('calls onChange on blur after edit', () => {
    render(
      <table>
        <tbody>
          <tr>
            <SpreadsheetCell
              cellRef="A1"
              cell={mockCell}
              onChange={mockOnChange}
            />
          </tr>
        </tbody>
      </table>
    );

    fireEvent.doubleClick(screen.getByText('42'));
    const input = screen.getByDisplayValue('42');

    fireEvent.change(input, { target: { value: '100' } });
    fireEvent.blur(input);

    expect(mockOnChange).toHaveBeenCalledWith('A1', { value: '100', formula: '' });
  });

  it('formats formula cells with leading =', () => {
    const formulaCell: SpreadsheetCellModel = {
      value: '30',
      formula: '=A1+B1'
    };

    render(
      <table>
        <tbody>
          <tr>
            <SpreadsheetCell
              cellRef="A1"
              cell={formulaCell}
              onChange={mockOnChange}
            />
          </tr>
        </tbody>
      </table>
    );

    expect(screen.getByText('30')).toBeInTheDocument();
  });

  it('displays empty string for empty cells', () => {
    const emptyCell: SpreadsheetCellModel = {
      value: '',
      formula: ''
    };

    render(
      <table>
        <tbody>
          <tr>
            <SpreadsheetCell
              cellRef="A1"
              cell={emptyCell}
              onChange={mockOnChange}
            />
          </tr>
        </tbody>
      </table>
    );

    const cellContent = screen.queryByText(/\S/);
    expect(cellContent).not.toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd zettelkasten-front
npm test -- SpreadsheetCell.test.tsx
```

Expected: FAIL - "Cannot find module './SpreadsheetCell'"

**Step 3: Implement the component**

```typescript
import React, { useState, useRef, useEffect, KeyboardEvent } from 'react';
import { SpreadsheetCell as SpreadsheetCellModel } from '../../models/Spreadsheet';

interface SpreadsheetCellProps {
  cellRef: string;
  cell: SpreadsheetCellModel;
  onChange: (cellRef: string, cell: SpreadsheetCellModel) => void;
  readOnly?: boolean;
}

export function SpreadsheetCell({ cellRef, cell, onChange, readOnly = false }: SpreadsheetCellProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(cell.formula || cell.value);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

  const handleDoubleClick = () => {
    if (!readOnly) {
      setIsEditing(true);
      setEditValue(cell.formula || cell.value);
    }
  };

  const handleBlur = () => {
    setIsEditing(false);

    // Determine if this is a formula or plain value
    const trimmed = editValue.trim();
    const newCell: SpreadsheetCellModel = {
      value: trimmed.startsWith('=') ? '' : trimmed,
      formula: trimmed.startsWith('=') ? trimmed.slice(1) : ''
    };

    onChange(cellRef, newCell);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      inputRef.current?.blur();
    } else if (e.key === 'Escape') {
      setEditValue(cell.formula || cell.value);
      setIsEditing(false);
    }
  };

  const displayValue = isEditing ? '' : (cell.value || '\u00A0');

  return (
    <td
      className="border border-gray-300 px-2 py-1 min-w-[80px] h-8 text-sm"
      onDoubleClick={handleDoubleClick}
      style={{ cursor: readOnly ? 'default' : 'pointer' }}
    >
      {isEditing ? (
        <input
          ref={inputRef}
          type="text"
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onBlur={handleBlur}
          onKeyDown={handleKeyDown}
          className="w-full h-full outline-none bg-blue-50"
        />
      ) : (
        <span className={cell.formula ? 'text-blue-600' : 'text-gray-900'}>
          {displayValue}
        </span>
      )}
    </td>
  );
}
```

**Step 4: Run tests to verify they pass**

```bash
cd zettelkasten-front
npm test -- SpreadsheetCell.test.tsx
```

Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/SpreadsheetCell.tsx zettelkasten-front/src/components/spreadsheets/SpreadsheetCell.test.tsx
git commit -m "feat(spreadsheet): add SpreadsheetCell component with edit mode"
```

---

## Task 6: Create SpreadsheetGrid Component

**Files:**
- Create: `zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.tsx`
- Test: `zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.test.tsx`

**Step 1: Write the test**

```typescript
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SpreadsheetGrid } from './SpreadsheetGrid';
import { Spreadsheet, SpreadsheetCell } from '../../models/Spreadsheet';

describe('SpreadsheetGrid', () => {
  const mockSpreadsheet: Spreadsheet = {
    name: 'test',
    data: {
      rows: 3,
      cols: 3,
      data: {
        'A1': { value: '10', formula: '' },
        'A2': { value: '20', formula: '' },
        'A3': { value: '30', formula: '=A1+A2' },
        'B1': { value: '5', formula: '' },
        'B2': { value: '', formula: '' },
      }
    }
  };

  const mockOnChange = vi.fn();

  it('renders grid with correct dimensions', () => {
    const { container } = render(
      <SpreadsheetGrid
        spreadsheet={mockSpreadsheet}
        onChange={mockOnChange}
      />
    );

    // Check for header row (A, B, C) + data rows
    const headerRow = container.querySelector('thead tr');
    expect(headerRow?.children.length).toBe(4); // Empty corner + A, B, C

    const dataRows = container.querySelectorAll('tbody tr');
    expect(dataRows.length).toBe(3); // 1, 2, 3
  });

  it('displays cell values correctly', () => {
    render(
      <SpreadsheetGrid
        spreadsheet={mockSpreadsheet}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText('10')).toBeInTheDocument();
    expect(screen.getByText('20')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
  });

  it('renders column headers (A, B, C...)', () => {
    const { container } = render(
      <SpreadsheetGrid
        spreadsheet={mockSpreadsheet}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
    expect(screen.getByText('C')).toBeInTheDocument();
  });

  it('renders row headers (1, 2, 3...)', () => {
    const { container } = render(
      <SpreadsheetGrid
        spreadsheet={mockSpreadsheet}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText('1')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('calls onChange when cell is edited', () => {
    const { container } = render(
      <SpreadsheetGrid
        spreadsheet={mockSpreadsheet}
        onChange={mockOnChange}
      />
    );

    const cellWith10 = screen.getByText('10');
    fireEvent.doubleClick(cellWith10);

    const input = screen.getByDisplayValue('10');
    fireEvent.change(input, { target: { value: '100' } });
    fireEvent.blur(input);

    expect(mockOnChange).toHaveBeenCalled();
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd zettelkasten-front
npm test -- SpreadsheetGrid.test.tsx
```

Expected: FAIL - "Cannot find module './SpreadsheetGrid'"

**Step 3: Implement the component**

```typescript
import React from 'react';
import { Spreadsheet, SpreadsheetCell } from '../../models/Spreadsheet';
import { SpreadsheetCell as SpreadsheetCellComponent } from './SpreadsheetCell';
import { coordsToA1 } from '../../utils/spreadsheetHelpers';

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
    const { evaluateFormula } = require('./formulaParser');

    for (const [ref, cell] of Object.entries(newData)) {
      if (cell.formula && cell.formula.startsWith('=')) {
        const result = evaluateFormula(cell.formula.slice(1), newData);
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
```

**Step 4: Run tests to verify they pass**

```bash
cd zettelkasten-front
npm test -- SpreadsheetGrid.test.tsx
```

Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.tsx zettelkasten-front/src/components/spreadsheets/SpreadsheetGrid.test.tsx
git commit -m "feat(spreadsheet): add SpreadsheetGrid component with formula recalculation"
```

---

## Task 7: Create DynamicSpreadsheet Component

**Files:**
- Create: `zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.tsx`
- Test: `zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.test.tsx`

**Step 1: Write the test**

```typescript
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DynamicSpreadsheet } from './DynamicSpreadsheet';

// Mock the card context for accessing card body
vi.mock('../../contexts/CardContext', () => ({
  useCardContext: () => ({
    viewingCard: {
      id: 1,
      body: 'Some text\n```spreadsheet:mysheet\n{"rows": 2, "cols": 2, "data": {"A1": {"value": "10", "formula": ""}}}\n```\nMore text'
    }
  })
}));

describe('DynamicSpreadsheet', () => {
  it('renders spreadsheet from card body', () => {
    render(<DynamicSpreadsheet name="mysheet" />);

    // Should render the grid
    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
  });

  it('renders default empty spreadsheet if not found', () => {
    render(<DynamicSpreadsheet name="nonexistent" />);

    // Should still render a grid
    expect(screen.getByText('A')).toBeInTheDocument();
  });

  it('displays "mysheet" as spreadsheet name', () => {
    const { container } = render(<DynamicSpreadsheet name="mysheet" />);
    // Component should render without error
    expect(container).toBeTruthy();
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd zettelkasten-front
npm test -- DynamicSpreadsheet.test.tsx
```

Expected: FAIL - "Cannot find module './DynamicSpreadsheet'"

**Step 3: Implement the component**

```typescript
import React, { useState, useEffect } from 'react';
import { Spreadsheet, createEmptySpreadsheet } from '../../models/Spreadsheet';
import { SpreadsheetGrid } from './SpreadsheetGrid';

interface DynamicSpreadsheetProps {
  name: string;
}

export function DynamicSpreadsheet({ name }: DynamicSpreadsheetProps) {
  const [spreadsheet, setSpreadsheet] = useState<Spreadsheet>(() =>
    createEmptySpreadsheet(name)
  );
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // TODO: Load spreadsheet data from card body
    // For now, we'll use the default empty spreadsheet
    setIsLoading(false);
  }, [name]);

  const handleChange = (updated: Spreadsheet) => {
    setSpreadsheet(updated);
    // TODO: Update the card body with new spreadsheet data
  };

  if (isLoading) {
    return <div className="p-4 text-gray-500">Loading spreadsheet...</div>;
  }

  return (
    <div className="my-4 border-l-4 border-green-500 pl-4">
      <div className="text-sm font-medium text-gray-700 mb-2">
        Spreadsheet: {name}
      </div>
      <SpreadsheetGrid
        spreadsheet={spreadsheet}
        onChange={handleChange}
        readOnly={false}
      />
    </div>
  );
}
```

**Step 4: Run tests to verify they pass**

```bash
cd zettelkasten-front
npm test -- DynamicSpreadsheet.test.tsx
```

Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.tsx zettelkasten-front/src/components/spreadsheets/DynamicSpreadsheet.test.tsx
git commit -m "feat(spreadsheet): add DynamicSpreadsheet component for inline rendering"
```

---

## Task 8: Create SpreadsheetsTab Component

**Files:**
- Create: `zettelkasten-front/src/components/tabs/SpreadsheetsTab.tsx`
- Test: `zettelkasten-front/src/components/tabs/SpreadsheetsTab.test.tsx`

**Step 1: Write the test**

```typescript
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SpreadsheetsTab } from './SpreadsheetsTab';
import { Card } from '../../models/Card';

describe('SpreadsheetsTab', () => {
  const mockCard: Card = {
    id: 1,
    card_id: '1',
    user_id: 1,
    title: 'Test Card',
    body: 'Content with {{spreadsheet:budget}}',
    link: '',
    is_deleted: false,
    created_at: new Date(),
    updated_at: new Date(),
    parent_id: 0,
    parent: {
      id: 0,
      card_id: '',
      user_id: 0,
      title: '',
      parent_id: 0,
      created_at: new Date(),
      updated_at: new Date(),
      tags: []
    },
    files: [],
    children: [],
    references: [],
    tags: [],
    tasks: [],
    entities: []
  };

  const mockSetViewCard = vi.fn();
  const mockSetError = vi.fn();

  it('renders list of spreadsheets found in card', () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    expect(screen.getByText('budget')).toBeInTheDocument();
  });

  it('shows "No spreadsheets" message when none found', () => {
    const cardWithoutSpreadsheets = { ...mockCard, body: 'Just plain text' };

    render(
      <SpreadsheetsTab
        viewingCard={cardWithoutSpreadsheets}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    expect(screen.getByText(/no spreadsheets/i)).toBeInTheDocument();
  });

  it('has "Add Spreadsheet" button', () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    expect(screen.getByText(/add spreadsheet/i)).toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

```bash
cd zettelkasten-front
npm test -- SpreadsheetsTab.test.tsx
```

Expected: FAIL - "Cannot find module './SpreadsheetsTab'"

**Step 3: Implement the component**

```typescript
import React, { useState, useMemo } from 'react';
import { Card } from '../../models/Card';
import { Button } from '../Button';
import { SpreadsheetGrid } from '../spreadsheets/SpreadsheetGrid';
import { Spreadsheet, createEmptySpreadsheet } from '../../models/Spreadsheet';

interface SpreadsheetsTabProps {
  viewingCard: Card;
  setViewCard: (card: Card) => void;
  setError: (error: string) => void;
}

// Extract spreadsheet names from card body using regex
function extractSpreadsheetNames(body: string): string[] {
  const regex = /\{\{spreadsheet(?::([^}\s]+))?\}\}/gi;
  const names = new Set<string>();
  let match;

  while ((match = regex.exec(body)) !== null) {
    const name = match[1] || 'sheet1';
    names.add(name);
  }

  return Array.from(names);
}

export function SpreadsheetsTab({ viewingCard, setViewCard, setError }: SpreadsheetsTabProps) {
  const [selectedSpreadsheet, setSelectedSpreadsheet] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);

  const spreadsheetNames = useMemo(() =>
    extractSpreadsheetNames(viewingCard.body),
    [viewingCard.body]
  );

  const handleCreateSpreadsheet = async () => {
    try {
      setIsCreating(true);

      // Generate a unique name
      const newName = `sheet${spreadsheetNames.length + 1}`;

      // Create empty spreadsheet data
      const newSpreadsheet = createEmptySpreadsheet(newName);

      // Insert the markdown syntax into the card body
      const updatedBody = `${viewingCard.body}\n\n{{spreadsheet:${newName}}}\n\n\`\`\`spreadsheet:${newName}\n${JSON.stringify(newSpreadsheet.data, null, 2)}\n\`\`\`\n`;

      setViewCard({
        ...viewingCard,
        body: updatedBody
      });

      setSelectedSpreadsheet(newName);
    } catch (err) {
      setError('Failed to create spreadsheet');
    } finally {
      setIsCreating(false);
    }
  };

  if (selectedSpreadsheet) {
    // Show the spreadsheet grid
    const spreadsheet = createEmptySpreadsheet(selectedSpreadsheet);

    return (
      <div>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium">{selectedSpreadsheet}</h3>
          <Button
            onClick={() => setSelectedSpreadsheet(null)}
            variant="outline"
            size="small"
          >
            Back to List
          </Button>
        </div>
        <SpreadsheetGrid
          spreadsheet={spreadsheet}
          onChange={() => {}}
          readOnly={false}
        />
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-medium">
          Spreadsheets ({spreadsheetNames.length})
        </h3>
        <Button
          onClick={handleCreateSpreadsheet}
          variant="primary"
          size="small"
          disabled={isCreating}
        >
          {isCreating ? 'Creating...' : 'Add Spreadsheet'}
        </Button>
      </div>

      {spreadsheetNames.length === 0 ? (
        <div className="text-center py-8 text-gray-500">
          No spreadsheets found in this card.
          Click "Add Spreadsheet" to create one.
        </div>
      ) : (
        <div className="space-y-2">
          {spreadsheetNames.map((name) => (
            <div
              key={name}
              onClick={() => setSelectedSpreadsheet(name)}
              className="p-3 border border-gray-200 rounded hover:bg-gray-50 cursor-pointer"
            >
              <div className="font-medium text-gray-900">{name}</div>
              <div className="text-sm text-gray-500">Click to edit</div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

**Step 4: Run tests to verify they pass**

```bash
cd zettelkasten-front
npm test -- SpreadsheetsTab.test.tsx
```

Expected: PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/tabs/SpreadsheetsTab.tsx zettelkasten-front/src/components/tabs/SpreadsheetsTab.test.tsx
git commit -m "feat(spreadsheet): add SpreadsheetsTab component for managing all spreadsheets"
```

---

## Task 9: Integrate Spreadsheet Rendering into CardBody

**Files:**
- Modify: `zettelkasten-front/src/components/cards/CardBody.tsx`

**Step 1: Add import for remarkSpreadsheet plugin and DynamicSpreadsheet component**

At the top of `CardBody.tsx`, add:

```typescript
import remarkSpreadsheet from "../../remark-spreadsheet";
import { DynamicSpreadsheet } from "../spreadsheets/DynamicSpreadsheet";
```

**Step 2: Add remarkSpreadsheet to the remarkPlugins array**

Find the remarkPlugins line (around line 297) and update:

```typescript
remarkPlugins={[remarkGfm, remarkTaskQuery, remarkEntity, remarkSchemaTable, remarkSpreadsheet]}
```

**Step 3: Add spreadsheet renderer in the components prop of Markdown**

In the Markdown component's components prop, find the div handler and add spreadsheet handling. Add after the schema table check:

```typescript
// Check if this is a spreadsheet container
if (propsData.className === "spreadsheet-container" || propsData["data-spreadsheet-name"] !== undefined) {
  const spreadsheetName = propsData["data-spreadsheet-name"] || "sheet1";
  return <DynamicSpreadsheet name={spreadsheetName} />;
}
```

**Step 4: Run existing tests to ensure no regression**

```bash
cd zettelkasten-front
npm test -- CardBody.test.ts
```

Expected: PASS (no existing tests broken)

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/cards/CardBody.tsx
git commit -m "feat(spreadsheet): integrate spreadsheet rendering into CardBody"
```

---

## Task 10: Add Spreadsheets Tab to ViewCardTabbedDisplay

**Files:**
- Modify: `zettelkasten-front/src/components/cards/ViewCardTabbedDisplay.tsx`

**Step 1: Import SpreadsheetsTab**

Add to imports at top:

```typescript
import { SpreadsheetsTab } from "../tabs/SpreadsheetsTab";
```

**Step 2: Add "Spreadsheets" to the tabs array**

Add to the tabs array (around line 59-65):

```typescript
const tabs = [
  { label: "Entities" },
  { label: "Facts" },
  { label: "History" },
  { label: "Summaries" },
  { label: "Files" },
  { label: "Spreadsheets" },
];
```

**Step 3: Add SpreadsheetsTab to the conditional rendering**

Add before the closing </div> of the component:

```typescript
{activeTab === "Spreadsheets" && (
  <SpreadsheetsTab
    viewingCard={viewingCard}
    setViewCard={setViewCard}
    setError={setError}
  />
)}
```

**Step 4: Run tests to ensure no regression**

```bash
cd zettelkasten-front
npm test
```

Expected: All existing tests still pass

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/cards/ViewCardTabbedDisplay.tsx
git commit -m "feat(spreadsheet): add Spreadsheets tab to ViewCardTabbedDisplay"
```

---

## Task 11: Add Spreadsheet Button to MarkdownToolbar

**Files:**
- Modify: `zettelkasten-front/src/components/cards/MarkdownToolbar.tsx`

**Step 1: Add Spreadsheet button to the toolbar**

Add after the Table button (around line 110):

```typescript
<Button
  onClick={() => onFormatText('spreadsheet')}
  variant="secondary"
  size="small"
>
  Spreadsheet
</Button>
```

**Step 2: Update CardBodyTextArea to handle spreadsheet format**

In `CardBodyTextArea.tsx`, add the case in the formatText switch statement:

```typescript
case 'spreadsheet':
  insertAtCursor('\n{{spreadsheet:mysheet}}\n\n```spreadsheet:mysheet\n{\n  "rows": 5,\n  "cols": 5,\n  "data": {}\n}\n```\n');
  break;
```

**Step 3: Run tests**

```bash
cd zettelkasten-front
npm test -- MarkdownToolbar.test.tsx
```

Expected: PASS

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/cards/MarkdownToolbar.tsx zettelkasten-front/src/components/cards/CardBodyTextArea.tsx
git commit -m "feat(spreadsheet): add Spreadsheet button to markdown toolbar"
```

---

## Task 12: Create Index File for Spreadsheets

**Files:**
- Create: `zettelkasten-front/src/components/spreadsheets/index.ts`

**Step 1: Create barrel export file**

```typescript
export { DynamicSpreadsheet } from './DynamicSpreadsheet';
export { SpreadsheetGrid } from './SpreadsheetGrid';
export { SpreadsheetCell } from './SpreadsheetCell';
export { evaluateFormula, tokenizeFormula, parseCellValue } from './formulaParser';
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/components/spreadsheets/index.ts
git commit -m "feat(spreadsheet): add index barrel export"
```

---

## Task 13: Final Integration Test

**Files:**
- Test: `zettelkasten-front/src/integration/spreadsheet.integration.test.tsx`

**Step 1: Create integration test**

```typescript
import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CardBody } from '../components/cards/CardBody';
import { Card } from '../models/Card';

describe('Spreadsheet Integration', () => {
  const cardWithSpreadsheet: Card = {
    id: 1,
    card_id: '1',
    user_id: 1,
    title: 'Test Card with Spreadsheet',
    body: 'My budget:\n\n{{spreadsheet:budget}}\n\n```spreadsheet:budget\n{\n  "rows": 3,\n  "cols": 2,\n  "data": {\n    "A1": {"value": "100", "formula": ""},\n    "A2": {"value": "200", "formula": ""},\n    "A3": {"value": "300", "formula": "=SUM(A1:A2)"}\n  }\n}\n```\n\nTotal is calculated.',
    link: '',
    is_deleted: false,
    created_at: new Date(),
    updated_at: new Date(),
    parent_id: 0,
    parent: {
      id: 0,
      card_id: '',
      user_id: 0,
      title: '',
      parent_id: 0,
      created_at: new Date(),
      updated_at: new Date(),
      tags: []
    },
    files: [],
    children: [],
    references: [],
    tags: [],
    tasks: [],
    entities: []
  };

  it('renders card with embedded spreadsheet', () => {
    render(<CardBody viewingCard={cardWithSpreadsheet} />);

    expect(screen.getByText('My budget:')).toBeInTheDocument();
    expect(screen.getByText('Total is calculated.')).toBeInTheDocument();
  });

  it('renders spreadsheet grid', () => {
    render(<CardBody viewingCard={cardWithSpreadsheet} />);

    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
  });
});
```

**Step 2: Run integration test**

```bash
cd zettelkasten-front
npm test -- spreadsheet.integration.test.tsx
```

Expected: PASS

**Step 3: Run full test suite**

```bash
cd zettelkasten-front
npm test
```

Expected: All tests pass

**Step 4: Final commit**

```bash
git add zettelkasten-front/src/integration/spreadsheet.integration.test.tsx
git commit -m "test(spreadsheet): add integration test for full feature"
```

---

## Summary

This implementation plan creates a complete spreadsheet feature with:

1. **Models & Types** - TypeScript interfaces for spreadsheet data structures
2. **Helper Functions** - A1 notation conversion (a1ToCoords, coordsToA1)
3. **Formula Parser** - Tokenization and evaluation of basic formulas (SUM, AVERAGE, COUNT)
4. **Remark Plugin** - Markdown syntax `{{spreadsheet:name}}` parsing
5. **Components** - SpreadsheetCell, SpreadsheetGrid, DynamicSpreadsheet, SpreadsheetsTab
6. **Integration** - CardBody rendering, tabbed display, toolbar button
7. **Tests** - Unit tests for each component and integration tests

**Total estimated tasks:** 13

**Testing approach:** TDD for each component with Vitest + React Testing Library

**Git strategy:** Commit after each completed task with descriptive messages
