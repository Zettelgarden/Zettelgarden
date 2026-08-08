/**
 * Tests for useCardData (src/hooks/useCardData.ts)
 *
 * useCardData fetches a card plus all of its sub-resources and manages
 * summary jobs. It depends on UIStateContext (for setLastCard), so tests
 * wrap in the real UIStateProvider and mock the API layer.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { UIStateProvider } from '../contexts/UIStateContext';
import { useCardData } from './useCardData';
import {
  getCard,
  getCardReferences,
  getCardChildren,
  getCardFiles,
  getCardTags,
  getCardTasks,
  getCardEntities,
  getLinkedEntitiesByCardPK,
} from '../api/cards';
import { fetchSummariesForCard } from '../api/summarizer';
import { setDocumentTitle } from '../utils/title';
import { convertCardToPartialCard } from '../utils/cards';
import { Card, PartialCard } from '../models/Card';

vi.mock('../api/cards', () => ({
  getCard: vi.fn(),
  getCardReferences: vi.fn(),
  getCardChildren: vi.fn(),
  getCardFiles: vi.fn(),
  getCardTags: vi.fn(),
  getCardTasks: vi.fn(),
  getCardEntities: vi.fn(),
  getLinkedEntitiesByCardPK: vi.fn(),
}));

vi.mock('../api/summarizer', () => ({
  fetchSummariesForCard: vi.fn(),
}));

vi.mock('../utils/title', () => ({
  setDocumentTitle: vi.fn(),
}));

vi.mock('../utils/cards', () => ({
  convertCardToPartialCard: vi.fn((card: Card) => ({
    id: card.id,
    card_id: card.card_id,
    user_id: card.user_id,
    title: card.title,
    parent_id: card.parent_id,
    created_at: card.created_at,
    updated_at: card.updated_at,
    tags: card.tags,
  })),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  return <UIStateProvider>{children}</UIStateProvider>;
}

function makeCard(overrides: Partial<Card> = {}): Card {
  return {
    id: 1,
    card_id: 'card-1',
    user_id: 1,
    title: 'Test Card',
    body: '',
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

function makePartialCard(overrides: Partial<PartialCard> = {}): PartialCard {
  return {
    id: 2,
    card_id: 'card-2',
    user_id: 1,
    title: 'Child',
    parent_id: 1,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    tags: [],
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useCardData', () => {
  it('fetches a card and all sub-resources on mount', async () => {
    const card = makeCard();
    vi.mocked(getCard).mockResolvedValue(card);
    const refs = {
      bidirectional: [makePartialCard()],
      outgoing: [],
      incoming: [],
    };
    vi.mocked(getCardReferences).mockResolvedValue(refs);
    vi.mocked(getCardChildren).mockResolvedValue([makePartialCard({ id: 3 })]);
    vi.mocked(getCardFiles).mockResolvedValue([{ id: 1 }]);
    vi.mocked(getCardTags).mockResolvedValue([{ id: 1, name: 'tag' }]);
    vi.mocked(getCardTasks).mockResolvedValue([{ id: 1 }]);
    vi.mocked(getCardEntities).mockResolvedValue([
      { id: 1, user_id: 1, name: 'E', description: '', type: 'concept', created_at: new Date(), updated_at: new Date(), card_count: 0, card_pk: null },
    ]);
    vi.mocked(getLinkedEntitiesByCardPK).mockResolvedValue([
      { id: 2, user_id: 1, name: 'Linked', description: '', type: 'concept', created_at: new Date(), updated_at: new Date(), card_count: 0, card_pk: null },
    ]);
    vi.mocked(fetchSummariesForCard).mockResolvedValue([]);

    const { result } = renderHook(() => useCardData('123'), { wrapper });

    await waitFor(() => expect(result.current.viewingCard).not.toBeNull());

    // Sub-resources are wired onto the viewing card
    expect(result.current.viewingCard!.references).toHaveLength(1);
    expect(result.current.viewingCard!.children).toHaveLength(1);
    expect(result.current.viewingCard!.files).toHaveLength(1);
    expect(result.current.viewingCard!.tags).toHaveLength(1);
    expect(result.current.viewingCard!.tasks).toHaveLength(1);
    expect(result.current.viewingCard!.entities).toHaveLength(1);
    expect(result.current.linkedEntities).toHaveLength(1);
    expect(result.current.categorizedReferences.bidirectional).toHaveLength(1);

    expect(getCard).toHaveBeenCalledWith('123');
    expect(fetchSummariesForCard).toHaveBeenCalledWith(123);
    expect(setDocumentTitle).toHaveBeenCalledWith('card-1 - Test Card');
    expect(convertCardToPartialCard).toHaveBeenCalled();
  });

  it('fetches the parent card and its children when the card has a parent', async () => {
    const child = makePartialCard({ id: 10 });
    const card = makeCard({
      parent_id: 7,
      parent: makePartialCard({
        id: 7,
        card_id: 'parent-7',
        title: 'Parent',
      }) as any,
    });
    const parentCard = makeCard({
      id: 7,
      card_id: 'parent-7',
      title: 'Parent',
    });

    vi.mocked(getCard)
      .mockResolvedValueOnce(card)
      .mockResolvedValueOnce(parentCard);
    vi.mocked(getCardReferences).mockResolvedValue({
      bidirectional: [],
      outgoing: [],
      incoming: [],
    });
    vi.mocked(getCardChildren).mockResolvedValue([child]);
    vi.mocked(getCardFiles).mockResolvedValue([]);
    vi.mocked(getCardTags).mockResolvedValue([]);
    vi.mocked(getCardTasks).mockResolvedValue([]);
    vi.mocked(getCardEntities).mockResolvedValue([]);
    vi.mocked(getLinkedEntitiesByCardPK).mockResolvedValue([]);
    vi.mocked(fetchSummariesForCard).mockResolvedValue([]);

    const { result } = renderHook(() => useCardData('card-1'), { wrapper });

    await waitFor(() => expect(result.current.parentCard).not.toBeNull());

    expect(getCard).toHaveBeenCalledTimes(2); // card + parent
    expect(result.current.parentCard!.id).toBe(7);
    expect(result.current.parentCard!.children).toHaveLength(1);
  });

  it('sets parentCard to null when the card has no parent', async () => {
    vi.mocked(getCard).mockResolvedValue(makeCard());
    vi.mocked(getCardReferences).mockResolvedValue({
      bidirectional: [],
      outgoing: [],
      incoming: [],
    });
    vi.mocked(getCardChildren).mockResolvedValue([]);
    vi.mocked(getCardFiles).mockResolvedValue([]);
    vi.mocked(getCardTags).mockResolvedValue([]);
    vi.mocked(getCardTasks).mockResolvedValue([]);
    vi.mocked(getCardEntities).mockResolvedValue([]);
    vi.mocked(getLinkedEntitiesByCardPK).mockResolvedValue([]);
    vi.mocked(fetchSummariesForCard).mockResolvedValue([]);

    const { result } = renderHook(() => useCardData('card-1'), { wrapper });

    await waitFor(() => expect(result.current.viewingCard).not.toBeNull());
    expect(result.current.parentCard).toBeNull();
  });

  it('handles an error response from getCard without crashing', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(getCard).mockResolvedValue({ error: 'Card not found' } as any);

    const { result } = renderHook(() => useCardData('missing'), { wrapper });

    await waitFor(() => expect(consoleSpy).toHaveBeenCalled());
    expect(result.current.viewingCard).toBeNull();

    consoleSpy.mockRestore();
  });

  it('handles a thrown error from getCard without crashing', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(getCard).mockRejectedValue(new Error('Network failure'));

    const { result } = renderHook(() => useCardData('missing'), { wrapper });

    await waitFor(() => expect(consoleSpy).toHaveBeenCalled());
    expect(result.current.viewingCard).toBeNull();

    consoleSpy.mockRestore();
  });

  it('loadSummaries stores completed summaries and derives latest', async () => {
    vi.mocked(fetchSummariesForCard).mockResolvedValue([
      { id: 1, status: 'processing' },
      { id: 2, status: 'complete', result: 'done' },
      { id: 3, status: 'complete', result: 'newer' },
    ]);

    const { result } = renderHook(() => useCardData(), { wrapper });

    await act(async () => {
      await result.current.loadSummaries(5);
    });

    expect(result.current.summaries).toHaveLength(3);
    expect(result.current.latestSummary?.id).toBe(3);
  });

  it('loadSummaries clears summaries when the card has none (expected error)', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(fetchSummariesForCard).mockRejectedValue(
      new Error('no rows in result set'),
    );

    const { result } = renderHook(() => useCardData(), { wrapper });

    await act(async () => {
      await result.current.loadSummaries(5);
    });

    expect(result.current.summaries).toBeNull();
    // The "no rows" case is expected — must not be logged as an error
    expect(consoleSpy).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it('setViewingCard updates the viewing card (optimistic update path)', () => {
    const { result } = renderHook(() => useCardData(), { wrapper });

    act(() => {
      result.current.setViewingCard(makeCard({ title: 'Optimistic' }));
    });

    expect(result.current.viewingCard?.title).toBe('Optimistic');
  });
});
