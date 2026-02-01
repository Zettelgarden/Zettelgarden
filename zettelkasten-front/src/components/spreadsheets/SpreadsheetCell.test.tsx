import { describe, it, expect, vi } from 'vitest';
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

  it('displays computed value for formula cells', () => {
    const formulaCell: SpreadsheetCellModel = {
      value: '30',
      formula: 'A1+B1'
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
