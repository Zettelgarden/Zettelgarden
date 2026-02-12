/**
 * Represents a single cell in a spreadsheet
 */
export interface SpreadsheetCell {
  value: string;        // Display value (result of formula or raw input)
  formula?: string;     // Formula WITHOUT leading = (e.g., "A1+B1" not "=A1+B1"), empty string if plain value
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
  id: number;          // Database ID
  user_id: number;     // Owner user ID
  card_id: number;     // Parent card ID
  name: string;        // Identifier for this spreadsheet
  data: SpreadsheetData;
  created_at: Date;    // Creation timestamp
  updated_at: Date;    // Last update timestamp
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
    id: 0,              // Placeholder - will be set by backend
    user_id: 0,         // Placeholder - will be set by backend
    card_id: 0,         // Placeholder - will be set by backend
    name,
    data: { rows, cols, data },
    created_at: new Date(),
    updated_at: new Date()
  };
};

export const defaultSpreadsheetData: SpreadsheetData = {
  rows: 5,
  cols: 5,
  data: {}
};

export const defaultSpreadsheet: Spreadsheet = {
  id: 0,
  user_id: 0,
  card_id: 0,
  name: "",
  data: defaultSpreadsheetData,
  created_at: new Date(),
  updated_at: new Date()
};
