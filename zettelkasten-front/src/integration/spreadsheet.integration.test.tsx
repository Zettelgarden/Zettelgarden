import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { useState } from 'react';
import { CardBody } from '../components/cards/CardBody';
import { Card } from '../models/Card';
import { MemoryRouter } from 'react-router-dom';
import { CardEditorProvider } from '../contexts/editor';
import { Spreadsheet } from '../models/Spreadsheet';

// Mock DialogStateContext
vi.mock('../contexts/DialogStateContext', () => ({
  useDialogState: () => ({
    showEntityDialog: false,
    setShowEntityDialog: vi.fn(),
    selectedEntity: null,
    setSelectedEntity: vi.fn(),
  }),
}));

// Mock file download API
vi.mock('../api/files', () => ({
  downloadFile: vi.fn(() => Promise.resolve('mocked-blob-url')),
}));

// Mock entity API
vi.mock('../api/entities', () => ({
  fetchEntityById: vi.fn(() => Promise.resolve({
    id: 1,
    user_id: 1,
    name: 'Test Entity',
    type: 'PERSON',
    description: '',
    created_at: new Date(),
    updated_at: new Date(),
    card_count: 0,
    card_pk: null,
  })),
}));

// Mock spreadsheets API
vi.mock('../api/spreadsheets', () => ({
  fetchSpreadsheets: vi.fn((cardId: number): Promise<Spreadsheet[]> => {
    return Promise.resolve([{
      id: 1,
      user_id: 1,
      card_id: cardId,
      name: 'budget',
      data: {
        rows: 3,
        cols: 2,
        data: {
          'A1': { value: '100', formula: '' },
          'A2': { value: '200', formula: '' },
          'A3': { value: '300', formula: '=SUM(A1:A2)' }
        }
      },
      created_at: new Date(),
      updated_at: new Date()
    }]);
  }),
  fetchSpreadsheet: vi.fn(),
  createSpreadsheet: vi.fn(),
  updateSpreadsheet: vi.fn(),
  deleteSpreadsheet: vi.fn(),
}));

describe('Spreadsheet Integration', () => {
  const cardWithSpreadsheet: Card = {
    id: 1,
    card_id: '1',
    user_id: 1,
    title: 'Test Card with Spreadsheet',
    body: 'My budget:\n\n{{spreadsheet:1}}\n\nTotal is calculated.',
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

  // Wrapper for the test with CardEditorProvider
  function TestWrapper({ children }: { children: React.ReactNode }) {
    const [editingCard, setEditingCard] = useState(cardWithSpreadsheet);
    return (
      <CardEditorProvider editingCard={editingCard} setEditingCard={setEditingCard}>
        <MemoryRouter>
          {children}
        </MemoryRouter>
      </CardEditorProvider>
    );
  }

  it('renders card with embedded spreadsheet', () => {
    render(
      <TestWrapper>
        <CardBody viewingCard={cardWithSpreadsheet} />
      </TestWrapper>
    );

    expect(screen.getByText('My budget:')).toBeInTheDocument();
    expect(screen.getByText('Total is calculated.')).toBeInTheDocument();
  });

  it('renders spreadsheet grid', async () => {
    render(
      <TestWrapper>
        <CardBody viewingCard={cardWithSpreadsheet} />
      </TestWrapper>
    );

    // Wait for the async fetchSpreadsheets to complete and grid to render
    await waitFor(() => {
      expect(screen.getByText('A')).toBeInTheDocument();
      expect(screen.getByText('B')).toBeInTheDocument();
    });
  });
});
