/**
 * Smoke tests: EntityPage and FactPage render tables with data (a5q.6).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { EntityPage } from '../EntityPage';
import { FactPage } from '../FactPage';
import { Entity } from '../../models/Card';
import { FactWithCard } from '../../models/Fact';

vi.mock('../../api/entities', () => ({
  fetchEntities: vi.fn(),
  mergeEntities: vi.fn(),
  deleteEntity: vi.fn(),
}));

vi.mock('../../api/facts', () => ({
  getAllFacts: vi.fn(),
  mergeFacts: vi.fn(),
  deleteFact: vi.fn(),
}));

const { fetchEntities } = await import('../../api/entities');
const { getAllFacts } = await import('../../api/facts');

const entities: Entity[] = [
  {
    id: 1,
    user_id: 1,
    name: 'Zettelkasten Method',
    description: 'A note-taking system',
    type: 'concept',
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    card_count: 3,
    card_pk: null,
  },
];

const facts: FactWithCard[] = [
  {
    id: 1,
    fact: 'Zettelkasten means slip box in German',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    card: {
      id: 9,
      card_id: 'card-9',
      user_id: 1,
      title: 'Related card',
      parent_id: 0,
      created_at: new Date(),
      updated_at: new Date(),
      tags: [],
    },
  },
];

describe('EntityPage smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders entities from mocked data', async () => {
    vi.mocked(fetchEntities).mockResolvedValue({
      entities,
      total: 1,
      total_pages: 1,
      page: 1,
      per_page: 20,
    });

    renderWithProviders(<EntityPage />);

    await waitFor(() =>
      expect(screen.getByText('Zettelkasten Method')).toBeInTheDocument(),
    );
    expect(screen.getByText('Entities')).toBeInTheDocument();
  });

  it('renders an empty state when there are no entities', async () => {
    vi.mocked(fetchEntities).mockResolvedValue({
      entities: [],
      total: 0,
      total_pages: 1,
      page: 1,
      per_page: 20,
    });

    renderWithProviders(<EntityPage />);

    // The real empty state, not just the header.
    await waitFor(() =>
      expect(screen.getByText('No entities found')).toBeInTheDocument(),
    );
  });
});

describe('FactPage smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders facts from mocked data', async () => {
    vi.mocked(getAllFacts).mockResolvedValue({
      facts,
      total: 1,
      total_pages: 1,
      page: 1,
      per_page: 20,
    });

    renderWithProviders(<FactPage />);

    await waitFor(() =>
      expect(
        screen.getByText('Zettelkasten means slip box in German'),
      ).toBeInTheDocument(),
    );
    expect(screen.getByText('Facts')).toBeInTheDocument();
  });

  it('renders an empty state when there are no facts', async () => {
    vi.mocked(getAllFacts).mockResolvedValue({
      facts: [],
      total: 0,
      total_pages: 1,
      page: 1,
      per_page: 20,
    });

    renderWithProviders(<FactPage />);

    // The real empty state, not just the header.
    await waitFor(() =>
      expect(screen.getByText('No facts found')).toBeInTheDocument(),
    );
  });
});
