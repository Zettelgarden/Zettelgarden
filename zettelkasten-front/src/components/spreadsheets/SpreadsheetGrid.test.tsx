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
