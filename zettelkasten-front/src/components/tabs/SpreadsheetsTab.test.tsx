import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, waitForElementToBeRemoved } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SpreadsheetsTab } from './SpreadsheetsTab';
import { Card } from '../../models/Card';
import * as spreadsheetsApi from '../../api/spreadsheets';
import { Spreadsheet } from '../../models/Spreadsheet';

// Mock the spreadsheets API
vi.mock('../../api/spreadsheets', () => ({
  fetchSpreadsheets: vi.fn(),
  createSpreadsheet: vi.fn(),
  deleteSpreadsheet: vi.fn(),
  updateSpreadsheet: vi.fn(),
}));

// Mock SpreadsheetGrid
vi.mock('../spreadsheets/SpreadsheetGrid', () => ({
  SpreadsheetGrid: ({ spreadsheet, onChange }: { spreadsheet: any; onChange: any }) => (
    <div data-testid="spreadsheet-grid">
      <div>Spreadsheet: {spreadsheet.name}</div>
      <button onClick={() => onChange(spreadsheet)}>Modify</button>
    </div>
  ),
}));

describe('SpreadsheetsTab', () => {
  const mockCard: Card = {
    id: 1,
    card_id: '1',
    user_id: 1,
    title: 'Test Card',
    body: 'Test content',
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
    external_events: [],
    entities: []
  };

  const mockSetViewCard = vi.fn();
  const mockSetError = vi.fn();

  const mockSpreadsheets: Spreadsheet[] = [
    {
      id: 1,
      user_id: 1,
      card_id: 1,
      name: 'budget',
      data: { rows: 5, cols: 5, data: {} },
      created_at: new Date('2024-01-01'),
      updated_at: new Date('2024-01-01'),
    },
    {
      id: 2,
      user_id: 1,
      card_id: 1,
      name: 'sheet1',
      data: { rows: 5, cols: 5, data: {} },
      created_at: new Date('2024-01-02'),
      updated_at: new Date('2024-01-02'),
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    // Default mock implementation for fetchSpreadsheets
    vi.mocked(spreadsheetsApi.fetchSpreadsheets).mockResolvedValue(mockSpreadsheets);
  });

  it('renders list of spreadsheets from API', async () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('budget')).toBeInTheDocument();
      expect(screen.getByText('sheet1')).toBeInTheDocument();
    });

    expect(spreadsheetsApi.fetchSpreadsheets).toHaveBeenCalledWith(1);
  });

  it('shows loading state while fetching', () => {
    vi.mocked(spreadsheetsApi.fetchSpreadsheets).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    expect(screen.getByText(/loading spreadsheets/i)).toBeInTheDocument();
  });

  it('shows "No spreadsheets" message when none found', async () => {
    vi.mocked(spreadsheetsApi.fetchSpreadsheets).mockResolvedValue([]);

    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText(/no spreadsheets/i)).toBeInTheDocument();
    });
  });

  it('has "Add Spreadsheet" button', async () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText(/add spreadsheet/i)).toBeInTheDocument();
    });
  });

  it('creates new spreadsheet when Add Spreadsheet is clicked', async () => {
    const newSpreadsheet: Spreadsheet = {
      id: 3,
      user_id: 1,
      card_id: 1,
      name: 'sheet2', // budget and sheet1 already exist, so sheet2 is next
      data: { rows: 5, cols: 5, data: {} },
      created_at: new Date(),
      updated_at: new Date(),
    };

    vi.mocked(spreadsheetsApi.createSpreadsheet).mockResolvedValue(newSpreadsheet);

    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    // Wait for loading to complete and button to be enabled
    await waitFor(() => {
      const addButton = screen.getByRole('button', { name: /add spreadsheet/i });
      expect(addButton).not.toBeDisabled();
    });

    const addButton = screen.getByRole('button', { name: /add spreadsheet/i });
    await userEvent.click(addButton);

    // Component starts with 'sheet1', sees it exists, so creates 'sheet2'
    expect(spreadsheetsApi.createSpreadsheet).toHaveBeenCalledWith(1, 'sheet2');
  });

  it('shows spreadsheet grid when spreadsheet is clicked', async () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    const budgetLink = await screen.findByText('budget');
    await userEvent.click(budgetLink);

    expect(screen.getByTestId('spreadsheet-grid')).toBeInTheDocument();
    expect(screen.getByText('Spreadsheet: budget')).toBeInTheDocument();
  });

  it('shows delete button on hover', async () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('budget')).toBeInTheDocument();
    });

    // Find the delete button (hidden by default, visible on hover/group-hover)
    const deleteButtons = screen.getAllByTitle('Delete spreadsheet');
    expect(deleteButtons.length).toBe(2);
  });

  it('shows delete confirmation dialog when delete is clicked', async () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('budget')).toBeInTheDocument();
    });

    const deleteButtons = screen.getAllByTitle('Delete spreadsheet');
    await userEvent.click(deleteButtons[0]);

    expect(screen.getByText(/delete spreadsheet/i)).toBeInTheDocument();
    expect(screen.getByText(/are you sure you want to delete "budget"/i)).toBeInTheDocument();
  });

  it('deletes spreadsheet when confirmed', async () => {
    vi.mocked(spreadsheetsApi.deleteSpreadsheet).mockResolvedValue(undefined);

    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('budget')).toBeInTheDocument();
    });

    // Click delete
    const deleteButtons = screen.getAllByTitle('Delete spreadsheet');
    await userEvent.click(deleteButtons[0]);

    // Confirm deletion
    const confirmButton = screen.getByText('Delete');
    await userEvent.click(confirmButton);

    await waitFor(() => {
      expect(spreadsheetsApi.deleteSpreadsheet).toHaveBeenCalledWith(1);
    });
  });

  it('cancels delete when cancel is clicked', async () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('budget')).toBeInTheDocument();
    });

    // Click delete
    const deleteButtons = screen.getAllByTitle('Delete spreadsheet');
    await userEvent.click(deleteButtons[0]);

    // Cancel deletion
    const cancelButton = screen.getByText('Cancel');
    await userEvent.click(cancelButton);

    // Dialog should be closed
    expect(screen.queryByText(/are you sure you want to delete/i)).not.toBeInTheDocument();
    // budget should still be visible
    expect(screen.getByText('budget')).toBeInTheDocument();
  });

  it('handles API error when fetching spreadsheets', async () => {
    const error = new Error('Failed to fetch');
    vi.mocked(spreadsheetsApi.fetchSpreadsheets).mockRejectedValue(error);

    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(mockSetError).toHaveBeenCalledWith('Failed to fetch');
    });
  });

  it('handles API error when creating spreadsheet', async () => {
    const error = new Error('Failed to create');
    vi.mocked(spreadsheetsApi.createSpreadsheet).mockRejectedValue(error);

    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    // Wait for loading to complete and button to be enabled
    await waitFor(() => {
      const addButton = screen.getByRole('button', { name: /add spreadsheet/i });
      expect(addButton).not.toBeDisabled();
    });

    const addButton = screen.getByRole('button', { name: /add spreadsheet/i });
    await userEvent.click(addButton);

    // Wait for async operations to complete
    await waitFor(() => {
      expect(mockSetError).toHaveBeenCalledWith('Failed to create');
    });
  });

  it('navigates back to list when Back to List is clicked', async () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('budget')).toBeInTheDocument();
    });

    // Click on a spreadsheet to enter detail view
    await userEvent.click(screen.getByText('budget'));

    expect(screen.getByTestId('spreadsheet-grid')).toBeInTheDocument();

    // Click back to list
    await userEvent.click(screen.getByText('Back to List'));

    // Should be back in list view
    expect(screen.getByText('budget')).toBeInTheDocument();
    expect(screen.queryByTestId('spreadsheet-grid')).not.toBeInTheDocument();
  });

  it('displays spreadsheet count in header', async () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('Spreadsheets (2)')).toBeInTheDocument();
    });
  });

  it('generates unique name when creating spreadsheet with existing names', async () => {
    const newSpreadsheet: Spreadsheet = {
      id: 3,
      user_id: 1,
      card_id: 1,
      name: 'sheet2', // Should skip sheet1 since it exists
      data: { rows: 5, cols: 5, data: {} },
      created_at: new Date(),
      updated_at: new Date(),
    };

    vi.mocked(spreadsheetsApi.createSpreadsheet).mockResolvedValue(newSpreadsheet);

    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    // Wait for loading to complete and button to be enabled
    await waitFor(() => {
      const addButton = screen.getByRole('button', { name: /add spreadsheet/i });
      expect(addButton).not.toBeDisabled();
    });

    const addButton = screen.getByRole('button', { name: /add spreadsheet/i });
    await userEvent.click(addButton);

    // Should request 'sheet2' since 'sheet1' already exists
    expect(spreadsheetsApi.createSpreadsheet).toHaveBeenCalledWith(1, 'sheet2');
  });
});
