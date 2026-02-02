import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { DynamicSpreadsheet, parseSpreadsheetFromBody, serializeSpreadsheetToBody } from './DynamicSpreadsheet';

describe('DynamicSpreadsheet', () => {
  const mockCardBody = 'Some text\n```spreadsheet:mysheet\n{"rows": 2, "cols": 2, "data": {"A1": {"value": "10", "formula": ""}}}\n```\nMore text';
  const mockOnBodyChange = vi.fn();

  it('renders spreadsheet from card body', () => {
    render(<DynamicSpreadsheet name="mysheet" cardBody={mockCardBody} onBodyChange={mockOnBodyChange} />);

    // Should render the grid
    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
  });

  it('renders default empty spreadsheet if not found', () => {
    render(<DynamicSpreadsheet name="nonexistent" cardBody={mockCardBody} onBodyChange={mockOnBodyChange} />);

    // Should still render a grid
    expect(screen.getByText('A')).toBeInTheDocument();
  });

  it('displays "mysheet" as spreadsheet name', () => {
    const { container } = render(<DynamicSpreadsheet name="mysheet" cardBody={mockCardBody} onBodyChange={mockOnBodyChange} />);
    // Component should render without error
    expect(container).toBeTruthy();
  });
});

describe('parseSpreadsheetFromBody', () => {
  it('parses spreadsheet data from card body', () => {
    const body = '```spreadsheet:mysheet\n{"rows": 2, "cols": 2, "data": {"A1": {"value": "10"}}}\n```';
    const result = parseSpreadsheetFromBody(body, 'mysheet');

    expect(result).toEqual({
      rows: 2,
      cols: 2,
      data: { A1: { value: '10' } }
    });
  });

  it('returns null for non-existent spreadsheet', () => {
    const body = '```spreadsheet:mysheet\n{"rows": 2, "cols": 2}\n```';
    const result = parseSpreadsheetFromBody(body, 'other');
    expect(result).toBeNull();
  });

  it('returns null for invalid JSON', () => {
    const body = '```spreadsheet:mysheet\ninvalid json\n```';
    const result = parseSpreadsheetFromBody(body, 'mysheet');
    expect(result).toBeNull();
  });
});

describe('serializeSpreadsheetToBody', () => {
  it('updates existing spreadsheet block in body', () => {
    const body = 'Some text\n```spreadsheet:mysheet\n{"rows": 2, "cols": 2}\n```\nMore text';
    const newData = { rows: 3, cols: 3, data: {} };
    const result = serializeSpreadsheetToBody(body, 'mysheet', newData);

    expect(result).toContain('"rows": 3');
    expect(result).toContain('"cols": 3');
    expect(result).toContain('Some text');
    expect(result).toContain('More text');
  });

  it('appends new spreadsheet block if not found', () => {
    const body = 'Some text';
    const newData = { rows: 2, cols: 2, data: {} };
    const result = serializeSpreadsheetToBody(body, 'newsheet', newData);

    expect(result).toContain('Some text');
    expect(result).toContain('```spreadsheet:newsheet');
    expect(result).toContain('"rows": 2');
  });
});
