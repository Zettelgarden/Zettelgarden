/**
 * Smoke test: DashboardPage renders with data and does not crash.
 * Crash-guard for the home page (a5q.6).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { renderWithProviders, settle } from '../../tests/utils';
import { DashboardPage } from '../DashboardPage';
import { PartialCard } from '../../models/Card';

vi.mock('../../api/cards', () => ({
  semanticSearchCardsPaginated: vi.fn(),
  getUnsortedCards: vi.fn(),
}));

const { semanticSearchCardsPaginated, getUnsortedCards } = await import(
  '../../api/cards'
);

function makePartialCard(overrides: Partial<PartialCard> = {}): PartialCard {
  return {
    id: 1,
    card_id: 'recent-1',
    user_id: 1,
    title: 'Recent Card Title',
    parent_id: 0,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    tags: [],
    ...overrides,
  };
}

describe('DashboardPage smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the welcome header and search box', async () => {
    vi.mocked(semanticSearchCardsPaginated).mockResolvedValue({
      results: [],
      page: 1,
      per_page: 10,
      total: 0,
      total_pages: 0,
    });
    vi.mocked(getUnsortedCards).mockResolvedValue({
      cards: [],
      page: 1,
      per_page: 10,
      total: 0,
      total_pages: 0,
    });

    renderWithProviders(<DashboardPage />);

    expect(screen.getByText(/Welcome to Zettelgarden/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Search cards...')).toBeInTheDocument();

    await settle();
  });

  it('renders recent and unsorted cards from mocked data', async () => {
    vi.mocked(semanticSearchCardsPaginated).mockResolvedValue({
      results: [
        {
          id: '1',
          type: 'card',
          title: 'Recent Card Title',
          preview: 'preview text',
          score: 1,
          created_at: new Date('2024-01-01T00:00:00Z'),
          updated_at: new Date('2024-01-02T00:00:00Z'),
          tags: [],
          metadata: { id: 1, card_id: 'recent-1', parent_id: 0 },
        },
      ],
      page: 1,
      per_page: 10,
      total: 1,
      total_pages: 1,
    });
    vi.mocked(getUnsortedCards).mockResolvedValue({
      cards: [
        makePartialCard({
          id: 2,
          card_id: 'unsorted-1',
          title: 'Unsorted Card Title',
        }),
      ],
      page: 1,
      per_page: 10,
      total: 1,
      total_pages: 1,
    });

    renderWithProviders(<DashboardPage />);

    await waitFor(() =>
      expect(screen.getByText(/- Recent Card Title/)).toBeInTheDocument(),
    );
    expect(screen.getByText(/- Unsorted Card Title/)).toBeInTheDocument();
    expect(screen.getByText('Recent Cards')).toBeInTheDocument();
    expect(screen.getByText('Unsorted Cards')).toBeInTheDocument();
  });

  it('does not crash when the APIs fail', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(semanticSearchCardsPaginated).mockRejectedValue(
      new Error('down'),
    );
    vi.mocked(getUnsortedCards).mockRejectedValue(new Error('down'));

    renderWithProviders(<DashboardPage />);

    await waitFor(() =>
      expect(screen.getByText('Recent Cards')).toBeInTheDocument(),
    );
    expect(screen.getByText('Welcome to Zettelgarden 🌱')).toBeInTheDocument();

    consoleSpy.mockRestore();
  });
});
