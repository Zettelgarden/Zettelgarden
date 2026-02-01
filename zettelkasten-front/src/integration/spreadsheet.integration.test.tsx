import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CardBody } from '../components/cards/CardBody';
import { Card } from '../models/Card';
import { MemoryRouter } from 'react-router-dom';

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

describe('Spreadsheet Integration', () => {
  const cardWithSpreadsheet: Card = {
    id: 1,
    card_id: '1',
    user_id: 1,
    title: 'Test Card with Spreadsheet',
    body: 'My budget:\n\n{{spreadsheet:budget}}\n\n```spreadsheet:budget\n{\n  "rows": 3,\n  "cols": 2,\n  "data": {\n    "A1": {"value": "100", "formula": ""},\n    "A2": {"value": "200", "formula": ""},\n    "A3": {"value": "300", "formula": "=SUM(A1:A2)"}\n  }\n}\n```\n\nTotal is calculated.',
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

  it('renders card with embedded spreadsheet', () => {
    render(
      <MemoryRouter>
        <CardBody viewingCard={cardWithSpreadsheet} />
      </MemoryRouter>
    );

    expect(screen.getByText('My budget:')).toBeInTheDocument();
    expect(screen.getByText('Total is calculated.')).toBeInTheDocument();
  });

  it('renders spreadsheet grid', () => {
    render(
      <MemoryRouter>
        <CardBody viewingCard={cardWithSpreadsheet} />
      </MemoryRouter>
    );

    expect(screen.getByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
  });
});
