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

/**
 * Creates an empty spreadsheet with the specified dimensions
 */
export const createEmptySpreadsheet = (name: string = "sheet1", rows: number = 5, cols: number = 5): Spreadsheet => {
  const data: Record<string, SpreadsheetCell> = {};

  // Initialize all cells with empty values
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      // Convert column to letters (0=A, 1=B, ..., 26=AA, 27=AB, etc.)
      let colStr = '';
      let c = col + 1;
      while (c > 0) {
        c -= 1;
        colStr = String.fromCharCode(65 + (c % 26)) + colStr;
        c = Math.floor(c / 26);
      }
      const cellRef = `${colStr}${row + 1}`;
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

export const defaultSpreadsheet: Spreadsheet = {
  name: "",
  data: defaultSpreadsheetData
};
