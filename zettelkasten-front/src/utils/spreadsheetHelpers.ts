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
