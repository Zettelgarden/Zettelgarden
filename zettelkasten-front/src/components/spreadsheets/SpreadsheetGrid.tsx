import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Spreadsheet, SpreadsheetCell } from '../../models/Spreadsheet';
import { SpreadsheetCell as SpreadsheetCellComponent } from './SpreadsheetCell';
import { evaluateFormula, extractCellReferences } from './formulaParser';
import { coordsToA1 } from '../../utils/spreadsheetHelpers';
import { SpreadsheetContextMenu, ContextMenuPosition, ContextMenuAction } from './SpreadsheetContextMenu';

interface SpreadsheetGridProps {
  spreadsheet: Spreadsheet;
  onChange: (spreadsheet: Spreadsheet) => void;
  readOnly?: boolean;
  onInsertRow?: (rowIndex: number, above: boolean) => void;
  onDeleteRow?: (rowIndex: number) => void;
  onInsertColumn?: (colIndex: number, left: boolean) => void;
  onDeleteColumn?: (colIndex: number) => void;
}

export function SpreadsheetGrid({ spreadsheet, onChange, readOnly = false, onInsertRow, onDeleteRow, onInsertColumn, onDeleteColumn }: SpreadsheetGridProps) {
  const { rows, cols, data } = spreadsheet.data;
  const [selectedCell, setSelectedCell] = useState<string | null>(null);
  const focusedCellRef = useRef<string | null>(null);
  const [contextMenu, setContextMenu] = useState<{
    position: ContextMenuPosition;
    type: 'row' | 'column';
    index: number;
  } | null>(null);

  // Get all cell references that a formula depends on
  function getDependencies(cellRef: string): Set<string> {
    const cell = data[cellRef];
    if (!cell?.formula) return new Set();

    const refs = extractCellReferences(cell.formula);
    return new Set(refs);
  }

  // Build dependency graph: maps each cell to the cells that depend on it
  function buildDependentsMap(): Map<string, Set<string>> {
    const dependents = new Map<string, Set<string>>();

    for (const [ref, cell] of Object.entries(data)) {
      if (cell.formula) {
        const deps = getDependencies(ref);
        for (const dep of deps) {
          if (!dependents.has(dep)) {
            dependents.set(dep, new Set());
          }
          dependents.get(dep)!.add(ref);
        }
      }
    }

    return dependents;
  }

  // Detect circular references using DFS
  function detectCircular(cellRef: string, visited: Set<string>, recStack: Set<string>): boolean {
    visited.add(cellRef);
    recStack.add(cellRef);

    const deps = getDependencies(cellRef);
    for (const dep of deps) {
      if (!visited.has(dep)) {
        if (detectCircular(dep, visited, recStack)) {
          return true;
        }
      } else if (recStack.has(dep)) {
        return true; // Circular reference detected
      }
    }

    recStack.delete(cellRef);
    return false;
  }

  // Topological sort for formula recalculation
  function topologicalSort(formulaCells: string[]): string[] {
    const inDegree = new Map<string, number>();
    const adjList = new Map<string, Set<string>>();

    // Initialize
    for (const ref of formulaCells) {
      inDegree.set(ref, 0);
      adjList.set(ref, new Set());
    }

    // Build graph
    for (const ref of formulaCells) {
      const deps = getDependencies(ref);
      for (const dep of deps) {
        if (formulaCells.includes(dep)) {
          adjList.get(dep)!.add(ref);
          inDegree.set(ref, (inDegree.get(ref) || 0) + 1);
        }
      }
    }

    // Kahn's algorithm
    const queue: string[] = [];
    const result: string[] = [];

    for (const ref of formulaCells) {
      if (inDegree.get(ref) === 0) {
        queue.push(ref);
      }
    }

    while (queue.length > 0) {
      const current = queue.shift()!;
      result.push(current);

      for (const dependent of adjList.get(current) || []) {
        inDegree.set(dependent, (inDegree.get(dependent) || 0) - 1);
        if (inDegree.get(dependent) === 0) {
          queue.push(dependent);
        }
      }
    }

    return result;
  }

  const handleCellChange = (cellRef: string, newCell: SpreadsheetCell) => {
    const newData = { ...data };

    // Update the cell
    newData[cellRef] = newCell;

    // Check for circular reference in the new formula
    if (newCell.formula) {
      const visited = new Set<string>();
      const recStack = new Set<string>();
      if (detectCircular(cellRef, visited, recStack)) {
        // Circular reference detected - mark the cell
        newData[cellRef] = {
          ...newCell,
          value: '#CIRCULAR'
        };
        onChange({
          ...spreadsheet,
          data: {
            ...spreadsheet.data,
            data: newData
          }
        });
        return;
      }
    }

    // Get all formula cells
    const formulaCells = Object.keys(newData).filter(ref => newData[ref].formula);

    // Sort using topological order for correct recalculation
    const sortedCells = topologicalSort(formulaCells);

    // Recalculate formulas in dependency order
    for (const ref of sortedCells) {
      const cell = newData[ref];
      if (cell.formula) {
        const result = evaluateFormula(cell.formula, newData);
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
    coordsToA1(0, i).replace(/\d+$/, '')
  );

  const rowHeaders = Array.from({ length: rows }, (_, i) => (i + 1).toString());

  // Focus management for keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!selectedCell || readOnly) return;

      const { row, col } = (() => {
        try {
          return coordsToA1.name === 'coordsToA1'
            ? (() => {
                // This is a workaround - we need to parse the cell ref
                const match = selectedCell.match(/^([A-Z]+)(\d+)$/);
                if (!match) return { row: 0, col: 0 };
                const [, colStr, rowStr] = match;
                let col = 0;
                for (let i = 0; i < colStr.length; i++) {
                  col = col * 26 + (colStr.charCodeAt(i) - 64);
                }
                return { row: parseInt(rowStr, 10) - 1, col: col - 1 };
              })()
            : { row: 0, col: 0 };
        } catch {
          return { row: 0, col: 0 };
        }
      })();

      let newRow = row;
      let newCol = col;

      switch (e.key) {
        case 'ArrowUp':
          if (row > 0) newRow = row - 1;
          break;
        case 'ArrowDown':
          if (row < rows - 1) newRow = row + 1;
          break;
        case 'ArrowLeft':
          if (col > 0) newCol = col - 1;
          break;
        case 'ArrowRight':
          if (col < cols - 1) newCol = col + 1;
          break;
        default:
          return;
      }

      if (newRow !== row || newCol !== col) {
        e.preventDefault();
        // Convert back to A1 notation
        let colStr = '';
        let c = newCol + 1;
        while (c > 0) {
          c -= 1;
          colStr = String.fromCharCode(65 + (c % 26)) + colStr;
          c = Math.floor(c / 26);
        }
        setSelectedCell(`${colStr}${newRow + 1}`);
      }
    };

    if (selectedCell) {
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [selectedCell, rows, cols, readOnly]);

  const handleCellNavigate = (cellRef: string) => {
    setSelectedCell(cellRef);
    focusedCellRef.current = cellRef;
  };

  const closeContextMenu = useCallback(() => {
    setContextMenu(null);
  }, []);

  const handleRowHeaderContextMenu = useCallback((e: React.MouseEvent, rowIndex: number) => {
    e.preventDefault();

    if (readOnly) return;

    setContextMenu({
      position: { x: e.clientX, y: e.clientY },
      type: 'row',
      index: rowIndex
    });
  }, [readOnly]);

  const handleColumnHeaderContextMenu = useCallback((e: React.MouseEvent, colIndex: number) => {
    e.preventDefault();

    if (readOnly) return;

    setContextMenu({
      position: { x: e.clientX, y: e.clientY },
      type: 'column',
      index: colIndex
    });
  }, [readOnly]);

  const handleInsertRowAbove = useCallback(() => {
    if (!contextMenu || !onInsertRow) return;
    onInsertRow(contextMenu.index, true);
  }, [contextMenu, onInsertRow]);

  const handleInsertRowBelow = useCallback(() => {
    if (!contextMenu || !onInsertRow) return;
    onInsertRow(contextMenu.index, false);
  }, [contextMenu, onInsertRow]);

  const handleDeleteRow = useCallback(() => {
    if (!contextMenu || !onDeleteRow) return;
    onDeleteRow(contextMenu.index);
  }, [contextMenu, onDeleteRow]);

  const handleInsertColumnLeft = useCallback(() => {
    if (!contextMenu || !onInsertColumn) return;
    onInsertColumn(contextMenu.index, true);
  }, [contextMenu, onInsertColumn]);

  const handleInsertColumnRight = useCallback(() => {
    if (!contextMenu || !onInsertColumn) return;
    onInsertColumn(contextMenu.index, false);
  }, [contextMenu, onInsertColumn]);

  const handleDeleteColumn = useCallback(() => {
    if (!contextMenu || !onDeleteColumn) return;
    onDeleteColumn(contextMenu.index);
  }, [contextMenu, onDeleteColumn]);

  const contextMenuActions: ContextMenuAction[] = contextMenu ? (
    contextMenu.type === 'row' ? [
      { label: 'Insert Row Above', action: handleInsertRowAbove },
      { label: 'Insert Row Below', action: handleInsertRowBelow },
      { label: 'Delete Row', action: handleDeleteRow },
    ] : [
      { label: 'Insert Column Left', action: handleInsertColumnLeft },
      { label: 'Insert Column Right', action: handleInsertColumnRight },
      { label: 'Delete Column', action: handleDeleteColumn },
    ]
  ) : [];

  return (
    <div className="overflow-x-auto relative">
      <table className="border-collapse border border-gray-400">
        <thead>
          <tr>
            <th className="border border-gray-300 bg-gray-100 px-2 py-1 w-8"></th>
            {columnHeaders.map((col, colIndex) => (
              <th
                key={col}
                className="border border-gray-300 bg-gray-100 px-2 py-1 min-w-[80px] font-semibold text-sm"
                onContextMenu={(e) => handleColumnHeaderContextMenu(e, colIndex)}
              >
                {col}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rowHeaders.map((row) => (
            <tr key={row}>
              <td
                className="border border-gray-300 bg-gray-100 px-2 py-1 text-center font-semibold text-sm"
                onContextMenu={(e) => handleRowHeaderContextMenu(e, parseInt(row, 10) - 1)}
              >
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
                    rows={rows}
                    cols={cols}
                    onNavigate={handleCellNavigate}
                  />
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
      <SpreadsheetContextMenu
        position={contextMenu?.position || null}
        actions={contextMenuActions}
        onClose={closeContextMenu}
      />
    </div>
  );
}
