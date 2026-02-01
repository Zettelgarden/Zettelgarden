import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SpreadsheetsTab } from './SpreadsheetsTab';
import { Card } from '../../models/Card';

describe('SpreadsheetsTab', () => {
  const mockCard: Card = {
    id: 1,
    card_id: '1',
    user_id: 1,
    title: 'Test Card',
    body: 'Content with {{spreadsheet:budget}}',
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

  it('renders list of spreadsheets found in card', () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    expect(screen.getByText('budget')).toBeInTheDocument();
  });

  it('shows "No spreadsheets" message when none found', () => {
    const cardWithoutSpreadsheets = { ...mockCard, body: 'Just plain text' };

    render(
      <SpreadsheetsTab
        viewingCard={cardWithoutSpreadsheets}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    expect(screen.getByText(/no spreadsheets/i)).toBeInTheDocument();
  });

  it('has "Add Spreadsheet" button', () => {
    render(
      <SpreadsheetsTab
        viewingCard={mockCard}
        setViewCard={mockSetViewCard}
        setError={mockSetError}
      />
    );

    expect(screen.getByText(/add spreadsheet/i)).toBeInTheDocument();
  });
});
