# Spreadsheet Row/Column Operations Design

**Date:** 2026-02-14
**Status:** Approved

## Overview

Add the ability to insert and delete rows and columns in the spreadsheet component, with Excel-like data shifting and automatic formula reference updates.

## Requirements

- Users can insert/delete rows and columns via right-click context menu
- Data shifts appropriately when rows/columns are inserted or deleted
- Formula cell references update automatically
- Soft limits: 26 columns (A-Z), 100 rows
- Confirmation dialog when deleting rows/columns with data
- Respect read-only mode

## Architecture

### New Components

1. **`SpreadsheetContextMenu`**
   - Reusable context menu component
   - Shows options based on what was clicked (row/column/cell)
   - Smart positioning to avoid viewport edge clipping

2. **Helper Functions** (in `spreadsheetOperations.ts`)
   - `insertRow(spreadsheet, rowIndex)`
   - `deleteRow(spreadsheet, rowIndex)`
   - `insertColumn(spreadsheet, columnIndex)`
   - `deleteColumn(spreadsheet, columnIndex)`
   - `updateFormulaReferences(formula, rowDelta, colDelta, beforePosition)`
   - `hasDataInRow(spreadsheet, rowIndex)`
   - `hasDataInColumn(spreadsheet, columnIndex)`

3. **`DeleteConfirmDialog`**
   - Modal for confirming deletion of rows/columns with data
   - Shows affected cells preview
   - Cancel/Confirm buttons

### Component Changes

- **`SpreadsheetGrid`**: Add right-click handlers to row/column headers
- **`DynamicSpreadsheet`**: Wrap grid with context menu provider, handle operations through existing `onChange` → debounced save flow

## Data Flow

### Insert Row Operation
```
1. Validate: rows < 100
2. For all cells with row >= rowIndex:
   - Move data[row][col] → data[row+1][col]
   - Update cell reference key (e.g., "A3" → "A4")
3. Create empty cells for new row at rowIndex
4. Increment rows count
5. For all formulas: update references if row >= rowIndex (rowDelta = +1)
```

### Delete Row Operation
```
1. Check hasDataInRow - if yes, show confirmation dialog
2. For all cells with row > rowIndex:
   - Move data[row][col] → data[row-1][col]
   - Update cell reference key
3. Delete cells at rowIndex
4. Decrement rows count
5. For all formulas: update references if row > rowIndex (rowDelta = -1)
6. References to deleted cells become "#REF!"
```

### Formula Reference Updates
- Parse formula to extract cell references
- For each reference: calculate row/column position
- Apply rowDelta/colDelta based on position relative to insert/delete
- Handle ranges: "A1:A5" → update both bounds
- References to deleted row/column → "#REF!"

### Context Menu Options

| Click Target | Options |
|--------------|---------|
| Row header | Insert Row Above, Insert Row Below, Delete Row |
| Column header | Insert Column Left, Insert Column Right, Delete Column |
| Cell | Show both row and column options (for that cell's position) |

## Error Handling

| Case | Behavior |
|------|----------|
| Soft limit reached (100 rows, 26 cols) | Show toast/notification |
| Delete last row/column | Disable option, show "Cannot delete last row/column" |
| Circular reference after operation | Display "#CIRCULAR" (existing behavior) |
| Formula parsing error | Mark with "#ERROR", log to console |
| Read-only spreadsheet | Hide add/remove options |

## Edge Cases

1. **Delete row/column referenced by formula** → Replace with "#REF!"
2. **Insert in middle of range references** → Expand range appropriately
3. **Read-only mode** → Hide context menu options
4. **During save operation** → Allow, cancel pending timeout, create new one

## Testing

### Unit Tests
- Formula reference updates (insert/delete row/col)
- Data structure operations (insert/delete at various positions)
- Edge cases (soft limits, last row/column, read-only)

### Integration Tests
- Context menu flow (right-click → action → verify result)
- Delete confirmation flow
- Persist to backend via debounced save

### Manual Testing Checklist
- Insert 10 rows rapidly → verify all saved
- Insert row, edit cell, delete row → verify no stale data
- Complex formula chain → insert row → verify recalculation
- Delete column referenced by formula → verify "#REF!" appears
