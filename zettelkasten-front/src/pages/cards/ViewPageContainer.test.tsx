/**
 * Tests for useViewPageContainer (src/pages/cards/ViewPageContainer.tsx)
 *
 * Verifies that the Related Cards list excludes cards that already appear in
 * the card's Linked references, even when the related-cards fetch happens
 * before the references are known.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { UIStateProvider } from '../../contexts/UIStateContext';
import { DialogStateProvider } from '../../contexts/DialogStateContext';
import { TaskProvider } from '../../contexts/TaskContext';
import { TagProvider } from '../../contexts/TagContext';
import { useViewPageContainer } from './ViewPageContainer';
import {
  getCard,
  getCardReferences,
  getCardChildren,
  getCardFiles,
  getCardTags,
  getCardTasks,
  getCardEntities,
  getLinkedEntitiesByCardPK,
  getRelatedCards,
  getUnlinkedMentions,
  getCardSuggestions,
} from '../../api/cards';
import { fetchSummariesForCard } from '../../api/summarizer';
import { Card, PartialCard } from '../../models/Card';

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
  getUnlinkedMentions: vi.fn(),
  getCardSuggestions: vi.fn(),
  saveExistingCard: vi.fn(),
}));

vi.mock('../../api/summarizer', () => ({
  fetchSummariesForCard: vi.fn(),
}));

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

function makePartialCard(id: number, cardId: string): PartialCard {
  return {
    id,
    card_id: cardId,
    user_id: 1,
    title: `Card ${cardId}`,
    parent_id: 0,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    tags: [],
  };
}

function wrapper({ children }: { children: React.ReactNode }) {
  return (
    <BrowserRouter>
      <TagProvider testing={true} testTags={[]}>
        <TaskProvider testing={true} testTasks={[]}>
          <UIStateProvider>
            <DialogStateProvider>{children}</DialogStateProvider>
          </UIStateProvider>
        </TaskProvider>
      </TagProvider>
    </BrowserRouter>
  );
}

describe('useViewPageContainer related cards filtering', () => {
  beforeEach(() => {
    vi.clearAllMocks();
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
    vi.mocked(getUnlinkedMentions).mockResolvedValue([]);
    vi.mocked(getCardSuggestions).mockResolvedValue([]);
    vi.mocked(fetchSummariesForCard).mockResolvedValue([]);
  });

  it('excludes related cards that appear in the Linked references', async () => {
    const referencedCard = makePartialCard(10, 'REF-10');
    const standaloneRelated = makePartialCard(11, 'REL-11');

    // Related cards are fetched first (before references are known).
    vi.mocked(getRelatedCards).mockResolvedValue([
      { card: referencedCard, score: 8, reasons: ['similarity'] },
      { card: standaloneRelated, score: 5, reasons: ['tags'] },
    ]);
    // References arrive later and include one of the related cards.
    vi.mocked(getCardReferences).mockResolvedValue({
      bidirectional: [],
      outgoing: [referencedCard],
      incoming: [],
    });

    const { result } = renderHook(() => useViewPageContainer({ cardId: '1' }), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.data.relatedCards).not.toBeNull();
    });

    const ids = result.current.data.relatedCards!.map((rc) => rc.card.id);
    expect(ids).toContain(11);
    expect(ids).not.toContain(10);
  });

  it('refetches related cards after refreshRelatedCards invalidates the cache', async () => {
    const staleCard = makePartialCard(11, 'REL-11');
    const freshCard = makePartialCard(12, 'REL-12');

    // First fetch returns the stale list; the invalidation triggers a second
    // fetch that returns the fresh list.
    vi.mocked(getRelatedCards)
      .mockResolvedValueOnce([{ card: staleCard, score: 5, reasons: ['tags'] }])
      .mockResolvedValueOnce([
        { card: freshCard, score: 6, reasons: ['tags'] },
      ]);

    const { result } = renderHook(() => useViewPageContainer({ cardId: '1' }), {
      wrapper,
    });

    await waitFor(() => {
      expect(result.current.data.relatedCards?.map((rc) => rc.card.id)).toEqual(
        [11],
      );
    });

    act(() => {
      result.current.actions.refreshRelatedCards();
    });

    await waitFor(() => {
      expect(result.current.data.relatedCards?.map((rc) => rc.card.id)).toEqual(
        [12],
      );
    });
    expect(getRelatedCards).toHaveBeenCalledTimes(2);
  });

  it('fetches unlinked mentions and removes one after adding a link', async () => {
    const mentionCard = makePartialCard(20, 'MENTION-20');
    vi.mocked(getUnlinkedMentions).mockResolvedValue([
      {
        card: mentionCard,
        mention_count: 1,
        context_snippet: '...mentions viewed-card here...',
      },
    ]);

    const { result } = renderHook(() => useViewPageContainer({ cardId: '1' }), {
      wrapper,
    });

    await waitFor(() => {
      expect(
        result.current.data.unlinkedMentions?.map((m) => m.card.id),
      ).toEqual([20]);
    });
    expect(getUnlinkedMentions).toHaveBeenCalledTimes(1);

    // Adding a link drops the mention from the list (viewing card is id 1).
    await act(async () => {
      await result.current.actions.addUnlinkedMentionLink(
        result.current.data.unlinkedMentions![0],
      );
    });

    expect(result.current.data.unlinkedMentions).toEqual([]);
  });
});
