/**
 * Smoke test: ViewPage renders a card with its data without crashing (a5q.6).
 *
 * The page pulls all data through useViewPageContainer -> useCardData, which
 * calls the api/cards + api/summarizer functions we mock here. This guards
 * the render path (header, content section, side panels) with realistic data.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { ViewPage } from '../cards/ViewPage';
import { Card } from '../../models/Card';

vi.mock('../../api/cards', () => ({
  getCard: vi.fn(),
  getCardReferences: vi.fn(),
  getCardChildren: vi.fn(),
  getCardFiles: vi.fn(),
  getCardTags: vi.fn(),
  getCardTasks: vi.fn(),
  getCardEntities: vi.fn(),
  getLinkedEntitiesByCardPK: vi.fn(),
  getRelatedCards: vi.fn(),
  saveExistingCard: vi.fn(),
}));

vi.mock('../../api/summarizer', () => ({
  fetchSummariesForCard: vi.fn(),
}));

const {
  getCard,
  getCardReferences,
  getCardChildren,
  getCardFiles,
  getCardTags,
  getCardTasks,
  getCardEntities,
  getLinkedEntitiesByCardPK,
  getRelatedCards,
} = await import('../../api/cards');
const { fetchSummariesForCard } = await import('../../api/summarizer');

function makeCard(overrides: Partial<Card> = {}): Card {
  return {
    id: 1,
    card_id: 'viewed-card',
    user_id: 1,
    title: 'Viewed Card Title',
    body: 'Card body content',
    link: '',
    is_deleted: false,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    parent_id: 0,
    parent: null as any,
    files: [],
    children: [],
    references: [],
    tags: [],
    tasks: [],
    entities: [],
    ...overrides,
  };
}

const emptyRefs = { bidirectional: [], outgoing: [], incoming: [] };

describe('ViewPage smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the card title and content for a loaded card', async () => {
    const card = makeCard();
    vi.mocked(getCard).mockResolvedValue(card);
    vi.mocked(getCardReferences).mockResolvedValue(emptyRefs);
    vi.mocked(getCardChildren).mockResolvedValue([]);
    vi.mocked(getCardFiles).mockResolvedValue([]);
    vi.mocked(getCardTags).mockResolvedValue([]);
    vi.mocked(getCardTasks).mockResolvedValue([]);
    vi.mocked(getCardEntities).mockResolvedValue([]);
    vi.mocked(getLinkedEntitiesByCardPK).mockResolvedValue([]);
    vi.mocked(getRelatedCards).mockResolvedValue([]);
    vi.mocked(fetchSummariesForCard).mockResolvedValue([]);

    renderWithProviders(<ViewPage cardId="viewed-card" />);

    await waitFor(() =>
      expect(screen.getAllByText('Viewed Card Title').length).toBeGreaterThan(
        0,
      ),
    );
    expect(screen.getByText(/Card body content/)).toBeInTheDocument();
  });

  it('renders without crashing when the card fails to load', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(getCard).mockRejectedValue(new Error('not found'));

    renderWithProviders(<ViewPage cardId="missing" />);

    await waitFor(() => expect(consoleSpy).toHaveBeenCalled());
    // Failure path must not render the card content.
    expect(screen.queryByText(/Viewed Card Title/)).toBeNull();

    consoleSpy.mockRestore();
  });
});
