/**
 * Tests for card data hooks (src/hooks/queries/useCards.ts)
 *
 * Follows the proven useTasks.test.tsx pattern: mock the api module,
 * wrap in a test QueryClient (retry off), renderHook + waitFor.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useCard,
  useCardReferences,
  useCardChildren,
  useCardFiles,
  useCardTags,
  useCardTasks,
  useCardEntities,
  useLinkedEntities,
  useCardAuditEvents,
  useCardSearch,
  useStarredCards,
  useUnsortedCards,
  useCreateCard,
  useUpdateCard,
  useDeleteCard,
  useStarCard,
  useUnstarCard,
  useRestoreCardToAuditEvent,
  useCreateArticle,
} from './useCards';
import {
  getCard,
  getCardReferences,
  getCardChildren,
  getCardFiles,
  getCardTags,
  getCardTasks,
  getCardEntities,
  getLinkedEntitiesByCardPK,
  getCardAuditEvents,
  semanticSearchCardsPaginated,
  starCard,
  unstarCard,
  getStarredCards,
  getUnsortedCards,
  saveNewCard,
  saveExistingCard,
  deleteCard,
  restoreCardToAuditEvent,
  createArticle,
} from '../../api/cards';
import { Card, PartialCard, SearchResult } from '../../models/Card';

vi.mock('../../api/cards', () => ({
  getCard: vi.fn(),
  getCardReferences: vi.fn(),
  getCardChildren: vi.fn(),
  getCardFiles: vi.fn(),
  getCardTags: vi.fn(),
  getCardTasks: vi.fn(),
  getCardEntities: vi.fn(),
  getLinkedEntitiesByCardPK: vi.fn(),
  getCardAuditEvents: vi.fn(),
  semanticSearchCardsPaginated: vi.fn(),
  starCard: vi.fn(),
  unstarCard: vi.fn(),
  getStarredCards: vi.fn(),
  getUnsortedCards: vi.fn(),
  saveNewCard: vi.fn(),
  saveExistingCard: vi.fn(),
  deleteCard: vi.fn(),
  restoreCardToAuditEvent: vi.fn(),
  createArticle: vi.fn(),
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: 0 },
      mutations: { retry: false },
    },
  });
}

function wrapper({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={createTestQueryClient()}>
      {children}
    </QueryClientProvider>
  );
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
    title: 'Child Card',
    parent_id: 1,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    tags: [],
    ...overrides,
  };
}

describe('useCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches a card by id', async () => {
    const card = makeCard();
    vi.mocked(getCard).mockResolvedValue(card);

    const { result } = renderHook(() => useCard('card-1'), { wrapper });

    expect(result.current.isLoading).toBe(true);
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data).toEqual(card);
    expect(getCard).toHaveBeenCalledWith('card-1');
  });

  it('does not fetch when disabled', async () => {
    const { result } = renderHook(() => useCard('card-1', false), { wrapper });

    await waitFor(() => expect(result.current.isFetching).toBe(false));
    expect(getCard).not.toHaveBeenCalled();
    expect(result.current.data).toBeUndefined();
  });

  it('surfaces fetch errors', async () => {
    const error = new Error('Card not found');
    vi.mocked(getCard).mockRejectedValue(error);

    const { result } = renderHook(() => useCard('missing'), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toEqual(error);
  });
});

describe('card sub-resource hooks', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const cases = [
    {
      hook: useCardReferences,
      mock: getCardReferences,
      url: '/cards/c1/references',
    },
    { hook: useCardChildren, mock: getCardChildren, url: '/cards/c1/children' },
    { hook: useCardFiles, mock: getCardFiles, url: '/cards/c1/files' },
    { hook: useCardTags, mock: getCardTags, url: '/cards/c1/tags' },
    { hook: useCardTasks, mock: getCardTasks, url: '/cards/c1/tasks' },
    { hook: useCardEntities, mock: getCardEntities, url: '/cards/c1/entities' },
    {
      hook: useLinkedEntities,
      mock: getLinkedEntitiesByCardPK,
      url: '/cards/c1/linked-entities',
    },
    {
      hook: useCardAuditEvents,
      mock: getCardAuditEvents,
      url: '/cards/c1/audit',
    },
  ] as const;

  it.each(cases.map((c) => [c.hook.name, c]))(
    '%s fetches data for the card id',
    async (_name, c) => {
      const payload =
        c.hook === useLinkedEntities
          ? [
              {
                id: 1,
                user_id: 1,
                name: 'Entity',
                description: '',
                type: 'concept',
                created_at: new Date(),
                updated_at: new Date(),
                card_count: 0,
                card_pk: null,
              },
            ]
          : [makePartialCard()];
      vi.mocked(c.mock).mockResolvedValue(payload as any);

      const { result } = renderHook(() => c.hook('c1'), { wrapper });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(c.mock).toHaveBeenCalledWith('c1');
      expect(result.current.data).toEqual(payload);
    },
  );

  it('does not fetch sub-resources without a card id', async () => {
    const { result } = renderHook(() => useCardChildren(''), { wrapper });

    await waitFor(() => expect(result.current.isFetching).toBe(false));
    expect(getCardChildren).not.toHaveBeenCalled();
  });
});

describe('useCardSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('maps search params with defaults and fetches', async () => {
    vi.mocked(semanticSearchCardsPaginated).mockResolvedValue({
      results: [] as SearchResult[],
      page: 1,
      per_page: 50,
      total: 0,
      total_pages: 0,
    });

    const { result } = renderHook(
      () => useCardSearch({ searchTerm: 'zettel', onlyEmptyCardId: false }),
      { wrapper },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(semanticSearchCardsPaginated).toHaveBeenCalledWith(
      'zettel', // searchTerm
      false, // fullText
      false, // showEntities
      true, // showFacts
      true, // showCards
      false, // showEmails
      'sortByRanking', // sortBy
      'classic', // searchType
      true, // rerank
      1, // page
      50, // perPage
      false, // onlyEmptyCardId
      undefined, // schemaId
    );
  });

  it('is disabled when searchTerm is empty and onlyEmptyCardId is false', async () => {
    const { result } = renderHook(() => useCardSearch({ searchTerm: '' }), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isFetching).toBe(false));
    expect(semanticSearchCardsPaginated).not.toHaveBeenCalled();
  });

  it('fetches when onlyEmptyCardId is set even with empty term', async () => {
    vi.mocked(semanticSearchCardsPaginated).mockResolvedValue({
      results: [],
      page: 1,
      per_page: 50,
      total: 0,
      total_pages: 0,
    });

    const { result } = renderHook(
      () => useCardSearch({ searchTerm: '', onlyEmptyCardId: true }),
      {
        wrapper,
      },
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(semanticSearchCardsPaginated).toHaveBeenCalled();
  });

  it('refetches when search params change (stable query key)', async () => {
    vi.mocked(semanticSearchCardsPaginated).mockResolvedValue({
      results: [],
      page: 1,
      per_page: 50,
      total: 0,
      total_pages: 0,
    });

    const { result, rerender } = renderHook(
      ({ term }) => useCardSearch({ searchTerm: term }),
      { wrapper, initialProps: { term: 'first' } },
    );

    await waitFor(() =>
      expect(semanticSearchCardsPaginated).toHaveBeenCalledTimes(1),
    );

    rerender({ term: 'second' });

    await waitFor(() =>
      expect(semanticSearchCardsPaginated).toHaveBeenCalledTimes(2),
    );
    expect(semanticSearchCardsPaginated).toHaveBeenLastCalledWith(
      'second',
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
      expect.anything(),
      undefined,
    );
  });
});

describe('useStarredCards', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches starred cards', async () => {
    const cards = [makeCard()];
    vi.mocked(getStarredCards).mockResolvedValue(cards);

    const { result } = renderHook(() => useStarredCards(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(cards);
  });
});

describe('useUnsortedCards', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches unsorted cards with page params', async () => {
    const response = {
      cards: [makePartialCard()],
      page: 2,
      per_page: 25,
      total: 1,
      total_pages: 1,
    };
    vi.mocked(getUnsortedCards).mockResolvedValue(response as any);

    const { result } = renderHook(() => useUnsortedCards(2, 25), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(getUnsortedCards).toHaveBeenCalledWith(2, 25);
    expect(result.current.data).toEqual(response);
  });
});

describe('useCreateCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates a card successfully', async () => {
    const card = makeCard();
    vi.mocked(saveNewCard).mockResolvedValue(card);

    const queryClient = createTestQueryClient();
    const { result } = renderHook(() => useCreateCard(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    result.current.mutate(card);

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(saveNewCard).toHaveBeenCalledWith(card, expect.any(Object));
  });

  it('handles create errors', async () => {
    const error = new Error('Create failed');
    vi.mocked(saveNewCard).mockRejectedValue(error);

    const { result } = renderHook(() => useCreateCard(), { wrapper });

    result.current.mutate(makeCard());

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toEqual(error);
  });
});

describe('useUpdateCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('updates a card and applies optimistic cache write', async () => {
    const card = makeCard();
    const updated = { ...card, title: 'Updated Title' };
    let resolveUpdate: (value: Card) => void;
    const updatePromise = new Promise<Card>((resolve) => {
      resolveUpdate = resolve;
    });
    vi.mocked(saveExistingCard).mockReturnValue(updatePromise as any);

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(['cards', 'detail', '1'], card);

    const { result } = renderHook(() => useUpdateCard(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    result.current.mutate(updated);

    await waitFor(() => {
      expect(queryClient.getQueryData(['cards', 'detail', '1'])).toEqual(
        updated,
      );
    });

    resolveUpdate!(updated);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it('rolls back the optimistic write when the update fails', async () => {
    const card = makeCard();
    const error = new Error('Update failed');
    vi.mocked(saveExistingCard).mockRejectedValue(error);

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(['cards', 'detail', '1'], card);

    const { result } = renderHook(() => useUpdateCard(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    result.current.mutate({ ...card, title: 'Wont Stick' });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(queryClient.getQueryData(['cards', 'detail', '1'])).toEqual(card);
  });
});

describe('useDeleteCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('deletes a card successfully', async () => {
    vi.mocked(deleteCard).mockResolvedValue(null);

    const { result } = renderHook(() => useDeleteCard(), { wrapper });

    result.current.mutate(1);

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(deleteCard).toHaveBeenCalledWith(1, expect.any(Object));
  });

  it('handles delete errors', async () => {
    const error = new Error('Delete failed');
    vi.mocked(deleteCard).mockRejectedValue(error);

    const { result } = renderHook(() => useDeleteCard(), { wrapper });

    result.current.mutate(1);

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toEqual(error);
  });
});

describe('useStarCard / useUnstarCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('star applies optimistic is_pinned and rolls back on error', async () => {
    const card = makeCard({ is_pinned: false } as any);
    vi.mocked(starCard).mockRejectedValue(new Error('Star failed'));

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(['cards', 'detail', '1'], card);

    const { result } = renderHook(() => useStarCard(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    result.current.mutate(1);

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(queryClient.getQueryData(['cards', 'detail', '1'])).toEqual(card);
    expect(starCard).toHaveBeenCalledWith(1);
  });

  it('unstar applies optimistic is_pinned false', async () => {
    const card = makeCard({ is_pinned: true } as any);
    let resolveUnstar: (value: void) => void;
    const unstarPromise = new Promise<void>((resolve) => {
      resolveUnstar = resolve;
    });
    vi.mocked(unstarCard).mockReturnValue(unstarPromise as any);

    const queryClient = createTestQueryClient();
    queryClient.setQueryData(['cards', 'detail', '1'], card);

    const { result } = renderHook(() => useUnstarCard(), {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      ),
    });

    result.current.mutate(1);

    await waitFor(() => {
      expect(
        (queryClient.getQueryData(['cards', 'detail', '1']) as any).is_pinned,
      ).toBe(false);
    });

    resolveUnstar!(undefined);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe('useRestoreCardToAuditEvent', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('restores a card to an audit event state', async () => {
    const card = makeCard();
    vi.mocked(restoreCardToAuditEvent).mockResolvedValue(card);

    const { result } = renderHook(() => useRestoreCardToAuditEvent(), {
      wrapper,
    });

    result.current.mutate({ cardId: 'c1', auditEventId: 99 });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(restoreCardToAuditEvent).toHaveBeenCalledWith('c1', 99);
  });
});

describe('useCreateArticle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('creates an article from a URL', async () => {
    const card = makeCard();
    vi.mocked(createArticle).mockResolvedValue(card);

    const { result } = renderHook(() => useCreateArticle(), { wrapper });

    result.current.mutate({
      url: 'https://example.com/a',
      cardId: 'c1',
      tags: 't1',
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(createArticle).toHaveBeenCalledWith(
      'https://example.com/a',
      'c1',
      't1',
    );
  });
});
