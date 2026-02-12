import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { DynamicSpreadsheet, parseSpreadsheetFromBody, serializeSpreadsheetToBody } from './DynamicSpreadsheet';
import { SpreadsheetData } from '../../models/Spreadsheet';

// Mock the API module
vi.mock('../../api/spreadsheets', () => ({
  updateSpreadsheet: vi.fn()
}));

describe('DynamicSpreadsheet', () => {
  const mockInitialData: SpreadsheetData = {
    rows: 2,
    cols: 2,
    data: {
      'A1': { value: '10', formula: '' },
      'A2': { value: '20', formula: '' },
      'B1': { value: '30', formula: '' },
      'B2': { value: '40', formula: '' }
    }
  };

  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('renders spreadsheet with initial data', () => {
    render(<DynamicSpreadsheet id={1} initialData={mockInitialData} readOnly={false} />);

    // Should render the grid
    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
  });

  it('renders empty spreadsheet when initialData is empty', () => {
    const emptyData: SpreadsheetData = {
      rows: 5,
      cols: 5,
      data: {}
    };

    render(<DynamicSpreadsheet id={1} initialData={emptyData} readOnly={false} />);

    // Should still render a grid with default columns
    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('E')).toBeInTheDocument();
  });

  it('displays "Spreadsheet" as the spreadsheet label', () => {
    const { container } = render(<DynamicSpreadsheet id={1} initialData={mockInitialData} readOnly={false} />);
    // Component should render without error
    expect(container).toBeTruthy();
    // Should show "Spreadsheet" label (not a specific name)
    expect(screen.getByText('Spreadsheet')).toBeInTheDocument();
  });

  it('passes readOnly prop to SpreadsheetGrid', () => {
    const { container } = render(<DynamicSpreadsheet id={1} initialData={mockInitialData} readOnly={true} />);
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
