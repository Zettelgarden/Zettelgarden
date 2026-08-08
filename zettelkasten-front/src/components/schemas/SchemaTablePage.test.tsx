import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { SchemaTablePage } from './SchemaTablePage';
import { SchemaDefinition } from '../../models/Schema';
import { Card } from '../../models/Card';
import { mockEndpoint } from '../../tests/fetchMock';

const { mockNavigate } = vi.hoisted(() => ({ mockNavigate: vi.fn() }));
vi.mock('react-router-dom', async () => {
  const actual =
    await vi.importActual<typeof import('react-router-dom')>(
      'react-router-dom',
    );
  return { ...actual, useNavigate: () => mockNavigate };
});

const { fetchSchema } = vi.hoisted(() => ({ fetchSchema: vi.fn() }));
vi.mock('../../api/schemas', () => ({ fetchSchema }));

// Keep the real schemaCardsToCsv (so column construction is exercised) but
// stub the actual browser download.
const { downloadCsv } = vi.hoisted(() => ({ downloadCsv: vi.fn() }));
vi.mock('../../utils/schemaCsv', async () => {
  const actual = await vi.importActual<typeof import('../../utils/schemaCsv')>(
    '../../utils/schemaCsv',
  );
  return { ...actual, downloadCsv };
});

const now = new Date();

const schema: SchemaDefinition = {
  id: 1,
  name: 'Book Review',
  slug: 'book-review',
  owner_id: 1,
  fields: [
    { name: 'Author', type: 'text', required: true },
    { name: 'Rating', type: 'number', required: false },
  ],
  created_at: now,
  updated_at: now,
  is_deleted: false,
};

function makeCard(overrides: Partial<Card>): Card {
  return {
    id: 1,
    card_id: '20240101120000',
    user_id: 1,
    title: 'Dune',
    body: '',
    link: '',
    is_deleted: false,
    created_at: '2024-01-01T00:00:00Z' as unknown as Date,
    updated_at: '2024-01-02T00:00:00Z' as unknown as Date,
    parent_id: -1,
    parent: {
      id: -1,
      card_id: '',
      user_id: -1,
      title: '',
      parent_id: -1,
      created_at: now,
      updated_at: now,
      tags: [],
    },
    files: [],
    children: [],
    references: [],
    tags: [],
    tasks: [],
    entities: [],
    ...overrides,
  };
}

const cards: Card[] = [
  makeCard({
    id: 1,
    card_id: '20240101120000',
    title: 'Dune',
    structured_data: { Author: 'Frank Herbert', Rating: 5 },
  }),
  makeCard({
    id: 2,
    card_id: '20240102120000',
    title: 'Foundation',
    structured_data: { Author: 'Isaac Asimov', Rating: 4 },
  }),
];

function renderPage(initialEntries: string[] = ['/table']) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <SchemaTablePage schemaId={1} onBack={() => {}} />
    </MemoryRouter>,
  );
}

describe('SchemaTablePage actions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (fetchSchema as ReturnType<typeof vi.fn>).mockResolvedValue(schema);
    mockEndpoint('/schemas/1/cards', cards);
  });

  it('renders the Add Card and Export CSV actions', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText(/Dune/)).toBeInTheDocument());

    expect(
      screen.getByRole('button', { name: /add card/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /export csv/i }),
    ).toBeInTheDocument();
  });

  it('navigates to card/new with the schema pre-selected on Add Card', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText(/Dune/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /add card/i }));

    expect(mockNavigate).toHaveBeenCalledWith('/app/card/new?schema=1');
  });

  it('exports CSV with card_id, title, and structured field columns', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText(/Dune/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /export csv/i }));

    expect(downloadCsv).toHaveBeenCalledTimes(1);
    const [filename, csv] = downloadCsv.mock.calls[0] as [string, string];
    expect(filename).toBe('book-review-cards.csv');

    const lines = csv.split('\n');
    expect(lines[0]).toBe('card_id,title,Author,Rating');
    expect(lines).toHaveLength(3); // header + 2 cards
    expect(lines[1]).toBe('20240101120000,Dune,Frank Herbert,5');
    expect(lines[2]).toBe('20240102120000,Foundation,Isaac Asimov,4');
  });

  it('exports only the filtered rows, honoring active filters', async () => {
    const authorFilter = encodeURIComponent(
      JSON.stringify({ type: 'text', operator: 'contains', value: 'Herbert' }),
    );
    renderPage([`/table?filter_Author=${authorFilter}`]);
    await waitFor(() => expect(screen.getByText(/Dune/)).toBeInTheDocument());
    // Foundation is filtered out of the visible table
    expect(screen.queryByText(/Foundation/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /export csv/i }));

    const [filename, csv] = downloadCsv.mock.calls[0] as [string, string];
    expect(filename).toBe('book-review-cards.csv');
    const lines = csv.split('\n');
    expect(lines).toHaveLength(2); // header + 1 filtered card
    expect(lines[1]).toContain('Dune');
    expect(lines[1]).toContain('Frank Herbert');
    expect(csv).not.toContain('Foundation');
  });
});
