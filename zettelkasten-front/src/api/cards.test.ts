/**
 * Contract tests for src/api/cards.ts
 *
 * These assert the request contract (URL, method, body params) and the
 * response processing (date conversion, empty handling, defaults) for every
 * card API function. They protect against client/server contract drift.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  semanticSearchCardsPaginated,
  semanticSearchCards,
  getCard,
  saveNewCard,
  saveExistingCard,
  deleteCard,
  getCardAuditEvents,
  getCardFiles,
  getCardChildren,
  getCardTags,
  getLinkedEntitiesByCardPK,
  getCardEntities,
  getCardTasks,
  getCardReferences,
  getNextRootId,
  starCard,
  unstarCard,
  getStarredCards,
  getUnsortedCards,
  restoreCardToAuditEvent,
  createArticle,
} from './cards';
import { apiClient } from './client';

vi.mock('./client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    fetchResponse: vi.fn(),
  },
  getData: vi.fn(
    async (promise: Promise<{ data: unknown }>) => (await promise).data,
  ),
}));

const mockedGet = vi.mocked(apiClient.get);
const mockedPost = vi.mocked(apiClient.post);
const mockedPut = vi.mocked(apiClient.put);
const mockedDelete = vi.mocked(apiClient.delete);
const mockedFetchResponse = vi.mocked(apiClient.fetchResponse);

function mockApiResponse<T>(data: T, status = 200) {
  return { data, response: { status, ok: status < 400 } as Response };
}

describe('semanticSearchCardsPaginated', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('POSTs search params to /search with snake_case mapping', async () => {
    mockedPost.mockResolvedValue(
      mockApiResponse({
        results: [],
        page: 1,
        per_page: 50,
        total: 0,
        total_pages: 0,
      }),
    );

    await semanticSearchCardsPaginated(
      'zettelkasten',
      true, // fullText
      true, // showEntities
      false, // showFacts
      true, // showCards
      true, // showEmails
      'sortCreatedNewOld',
      'classic',
      false, // rerank
      2,
      25,
      false,
      7, // schemaId
    );

    expect(mockedPost).toHaveBeenCalledWith('/search', {
      search_term: 'zettelkasten',
      search_type: 'classic',
      full_text: true,
      show_entities: true,
      show_facts: false,
      show_cards: true,
      only_empty_card_id: false,
      schema_id: 7,
      sort: 'sortCreatedNewOld',
      rerank: false,
      page: 2,
      per_page: 25,
    });
  });

  it('applies defaults when called with no arguments', async () => {
    mockedPost.mockResolvedValue(
      mockApiResponse({
        results: [],
        page: 1,
        per_page: 50,
        total: 0,
        total_pages: 0,
      }),
    );

    await semanticSearchCardsPaginated();

    expect(mockedPost).toHaveBeenCalledWith('/search', {
      search_term: '',
      search_type: 'classic',
      full_text: false,
      show_entities: false,
      show_facts: true,
      show_cards: true,
      only_empty_card_id: false,
      schema_id: undefined,
      sort: 'sortByRanking',
      rerank: true,
      page: 1,
      per_page: 50,
    });
  });

  it('converts result date strings to Date objects', async () => {
    mockedPost.mockResolvedValue(
      mockApiResponse({
        results: [
          {
            id: '1',
            type: 'card',
            title: 'Result',
            preview: '...',
            score: 0.9,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-02T00:00:00Z',
            tags: [],
            metadata: {},
          },
        ],
        page: 1,
        per_page: 50,
        total: 1,
        total_pages: 1,
      }),
    );

    const result = await semanticSearchCardsPaginated('query');

    expect(result.results[0].created_at).toBeInstanceOf(Date);
    expect(result.results[0].updated_at).toBeInstanceOf(Date);
  });

  it('returns empty pagination when response has no results', async () => {
    mockedPost.mockResolvedValue(mockApiResponse(null));

    const result = await semanticSearchCardsPaginated('query');

    expect(result).toEqual({
      results: [],
      page: 1,
      per_page: 50,
      total: 0,
      total_pages: 0,
    });
  });
});

describe('semanticSearchCards (legacy wrapper)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('delegates to paginated search with page 1 and perPage 1000', async () => {
    mockedPost.mockResolvedValue(
      mockApiResponse({
        results: [
          {
            id: '1',
            type: 'card',
            title: 'Result',
            preview: '...',
            score: 0.5,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
            tags: [],
            metadata: {},
          },
        ],
        page: 1,
        per_page: 1000,
        total: 1,
        total_pages: 1,
      }),
    );

    const results = await semanticSearchCards('hello');

    expect(mockedPost).toHaveBeenCalledWith(
      '/search',
      expect.objectContaining({
        search_term: 'hello',
        page: 1,
        per_page: 1000,
      }),
    );
    expect(results).toHaveLength(1);
    expect(results[0].title).toBe('Result');
  });
});

describe('getCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('GETs /cards/:id and converts dates', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse({
        id: 1,
        card_id: 'abc-123',
        user_id: 1,
        title: 'Test',
        body: '',
        link: '',
        is_deleted: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-02T00:00:00Z',
        parent_id: 0,
        parent: null,
        files: [],
        children: [
          {
            id: 2,
            card_id: 'child',
            user_id: 1,
            title: 'Child',
            parent_id: 1,
            created_at: '2024-01-03T00:00:00Z',
            updated_at: '2024-01-03T00:00:00Z',
            tags: [],
          },
        ],
        references: [],
        tags: [],
        tasks: [
          {
            id: 9,
            title: 'Task',
            scheduled_date: '2024-02-01T00:00:00Z',
            due_date: null,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
            completed_at: null,
          },
        ],
        entities: [],
      }),
    );

    const card = await getCard('abc-123');

    expect(mockedGet).toHaveBeenCalledWith('/cards/abc-123');
    expect(card.created_at).toBeInstanceOf(Date);
    expect(card.updated_at).toBeInstanceOf(Date);
    expect(card.children[0].created_at).toBeInstanceOf(Date);
    expect(card.tasks[0].scheduled_date).toBeInstanceOf(Date);
  });

  it('URL-encodes the card id', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse({
        id: 1,
        card_id: 'x',
        user_id: 1,
        title: 'X',
        body: '',
        link: '',
        is_deleted: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        parent_id: 0,
        parent: null,
        files: [],
        children: [],
        references: [],
        tags: [],
        tasks: [],
        entities: [],
      }),
    );
    await getCard('has space/and-slash');
    expect(mockedGet).toHaveBeenCalledWith('/cards/has%20space%2Fand-slash');
  });
});

describe('saveNewCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('POSTs to /cards with trimmed card_id', async () => {
    mockedPost.mockResolvedValue(
      mockApiResponse({
        id: 1,
        card_id: 'new-card',
        user_id: 1,
        title: 'New',
        body: '',
        link: '',
        is_deleted: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        parent_id: 0,
        parent: null,
        files: [],
        children: [],
        references: [],
        tags: [],
        tasks: [],
        entities: [],
      }),
    );

    const card = {
      id: 0,
      card_id: '  new-card  ',
      user_id: 1,
      title: 'New',
      body: '',
      link: '',
      is_deleted: false,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      parent_id: 0,
      parent: null,
      files: [],
      children: [],
      references: [],
      tags: [],
      tasks: [],
      entities: [],
    };
    await saveNewCard(card as any);

    expect(mockedPost).toHaveBeenCalledWith(
      '/cards',
      expect.objectContaining({ card_id: 'new-card' }),
    );
  });
});

describe('saveExistingCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('PUTs to /cards/:id', async () => {
    mockedPut.mockResolvedValue(
      mockApiResponse({
        id: 42,
        card_id: 'c',
        user_id: 1,
        title: 'Updated',
        body: '',
        link: '',
        is_deleted: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        parent_id: 0,
        parent: null,
        files: [],
        children: [],
        references: [],
        tags: [],
        tasks: [],
        entities: [],
      }),
    );

    await saveExistingCard({ id: 42, title: 'Updated' } as any);

    expect(mockedPut).toHaveBeenCalledWith(
      '/cards/42',
      expect.objectContaining({ id: 42 }),
    );
  });
});

describe('deleteCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('DELETEs /cards/:id and returns data on 200', async () => {
    mockedDelete.mockResolvedValue(
      mockApiResponse({
        id: 1,
        card_id: 'x',
        user_id: 1,
        title: 'Gone',
        body: '',
        link: '',
        is_deleted: true,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        parent_id: 0,
        parent: null,
        files: [],
        children: [],
        references: [],
        tags: [],
        tasks: [],
        entities: [],
      }),
    );

    const result = await deleteCard(1);

    expect(mockedDelete).toHaveBeenCalledWith('/cards/1');
    expect(result?.is_deleted).toBe(true);
  });

  it('returns null on 204 No Content', async () => {
    mockedDelete.mockResolvedValue(mockApiResponse(undefined, 204));

    const result = await deleteCard(7);

    expect(result).toBeNull();
  });
});

describe('card sub-resource fetchers', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getCardAuditEvents GETs /cards/:id/audit and converts dates', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse([
        {
          id: 1,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ]),
    );

    const events = await getCardAuditEvents('c1');
    expect(mockedGet).toHaveBeenCalledWith('/cards/c1/audit');
    expect(events[0].created_at).toBeInstanceOf(Date);
  });

  it('getCardFiles GETs /cards/:id/files', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse([
        {
          id: 1,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ]),
    );
    const files = await getCardFiles('c1');
    expect(mockedGet).toHaveBeenCalledWith('/cards/c1/files');
    expect(files[0].created_at).toBeInstanceOf(Date);
  });

  it('getCardChildren GETs /cards/:id/children', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse([
        {
          id: 2,
          card_id: 'kid',
          user_id: 1,
          title: 'Kid',
          parent_id: 1,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          tags: [],
        },
      ]),
    );
    const children = await getCardChildren('c1');
    expect(mockedGet).toHaveBeenCalledWith('/cards/c1/children');
    expect(children[0].created_at).toBeInstanceOf(Date);
  });

  it('getCardTags GETs /cards/:id/tags', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse([{ id: 1, name: 'tag', color: 'black' }]),
    );
    const tags = await getCardTags('c1');
    expect(mockedGet).toHaveBeenCalledWith('/cards/c1/tags');
    expect(tags[0].name).toBe('tag');
  });

  it('getCardEntities GETs /cards/:id/entities', async () => {
    mockedGet.mockResolvedValue(mockApiResponse([{ id: 1, name: 'Entity' }]));
    const entities = await getCardEntities('c1');
    expect(mockedGet).toHaveBeenCalledWith('/cards/c1/entities');
    expect(entities[0].name).toBe('Entity');
  });

  it('getCardTasks GETs /cards/:id/tasks and converts dates', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse([
        {
          id: 1,
          title: 'T',
          scheduled_date: '2024-01-01T00:00:00Z',
          due_date: null,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          completed_at: null,
        },
      ]),
    );
    const tasks = await getCardTasks('c1');
    expect(mockedGet).toHaveBeenCalledWith('/cards/c1/tasks');
    expect(tasks[0].scheduled_date).toBeInstanceOf(Date);
  });

  it('getLinkedEntitiesByCardPK uses fetchResponse and handles 204', async () => {
    mockedFetchResponse.mockResolvedValue({
      status: 204,
      json: vi.fn(),
    } as any);

    const entities = await getLinkedEntitiesByCardPK('c1');
    expect(mockedFetchResponse).toHaveBeenCalledWith(
      '/cards/c1/linked-entities',
    );
    expect(entities).toEqual([]);
  });

  it('getLinkedEntitiesByCardPK parses entities from response body', async () => {
    mockedFetchResponse.mockResolvedValue({
      status: 200,
      json: vi.fn().mockResolvedValue([{ id: 1, name: 'E' }]),
    } as any);

    const entities = await getLinkedEntitiesByCardPK('c1');
    expect(entities).toEqual([{ id: 1, name: 'E' }]);
  });

  it('getCardReferences GETs /cards/:id/references and categorizes', async () => {
    const ref = {
      id: 3,
      card_id: 'r',
      user_id: 1,
      title: 'Ref',
      parent_id: 1,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      tags: [],
    };
    mockedGet.mockResolvedValue(
      mockApiResponse({ bidirectional: [ref], outgoing: [], incoming: [ref] }),
    );

    const refs = await getCardReferences('c1');
    expect(mockedGet).toHaveBeenCalledWith('/cards/c1/references');
    expect(refs.bidirectional).toHaveLength(1);
    expect(refs.outgoing).toHaveLength(0);
    expect(refs.incoming[0].created_at).toBeInstanceOf(Date);
  });

  it('getCardReferences returns empty categories when response is null', async () => {
    mockedGet.mockResolvedValue(mockApiResponse(null));
    const refs = await getCardReferences('c1');
    expect(refs).toEqual({ bidirectional: [], outgoing: [], incoming: [] });
  });

  it('sub-resource fetchers return [] on null response', async () => {
    mockedGet.mockResolvedValue(mockApiResponse(null));
    expect(await getCardAuditEvents('c1')).toEqual([]);
    expect(await getCardFiles('c1')).toEqual([]);
    expect(await getCardChildren('c1')).toEqual([]);
    expect(await getCardTags('c1')).toEqual([]);
    expect(await getCardEntities('c1')).toEqual([]);
    expect(await getCardTasks('c1')).toEqual([]);
  });
});

describe('star / unstar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('starCard POSTs to /cards/:id/star', async () => {
    mockedPost.mockResolvedValue(mockApiResponse(null));
    await starCard(5);
    expect(mockedPost).toHaveBeenCalledWith('/cards/5/star', undefined);
  });

  it('unstarCard DELETEs /cards/:id/star', async () => {
    mockedDelete.mockResolvedValue(mockApiResponse(null));
    await unstarCard(5);
    expect(mockedDelete).toHaveBeenCalledWith('/cards/5/star');
  });
});

describe('getStarredCards', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('GETs /cards/starred and marks cards as pinned with dates converted', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse([
        {
          card: {
            id: 1,
            card_id: 's',
            user_id: 1,
            title: 'Starred',
            body: '',
            link: '',
            is_deleted: false,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
            parent_id: 0,
            children: [],
            references: [],
            tasks: [],
            tags: [],
          },
        },
      ]),
    );

    const cards = await getStarredCards();
    expect(mockedGet).toHaveBeenCalledWith('/cards/starred');
    expect((cards[0] as any).is_pinned).toBe(true);
    expect(cards[0].created_at).toBeInstanceOf(Date);
  });

  it('returns [] when response is null', async () => {
    mockedGet.mockResolvedValue(mockApiResponse(null));
    expect(await getStarredCards()).toEqual([]);
  });
});

describe('getUnsortedCards', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('GETs /cards/unsorted with page params and converts dates', async () => {
    mockedGet.mockResolvedValue(
      mockApiResponse({
        cards: [
          {
            id: 1,
            card_id: 'u',
            user_id: 1,
            title: 'U',
            parent_id: 0,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
            tags: [],
          },
        ],
        page: 2,
        per_page: 25,
        total: 1,
        total_pages: 1,
      }),
    );

    const result = await getUnsortedCards(2, 25);
    expect(mockedGet).toHaveBeenCalledWith(
      '/cards/unsorted?page=2&per_page=25',
    );
    expect(result.cards[0].created_at).toBeInstanceOf(Date);
    expect(result.page).toBe(2);
  });
});

describe('restoreCardToAuditEvent', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('POSTs to /cards/:id/audit/:eventId/restore', async () => {
    mockedPost.mockResolvedValue(
      mockApiResponse({
        id: 1,
        card_id: 'c',
        user_id: 1,
        title: 'Restored',
        body: '',
        link: '',
        is_deleted: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        parent_id: 0,
        parent: null,
        files: [],
        children: [],
        references: [],
        tags: [],
        tasks: [],
        entities: [],
      }),
    );

    const card = await restoreCardToAuditEvent('c1', 99);
    expect(mockedPost).toHaveBeenCalledWith(
      '/cards/c1/audit/99/restore',
      undefined,
    );
    expect(card.created_at).toBeInstanceOf(Date);
  });
});

describe('createArticle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('POSTs to /articles with url, card_id and tags', async () => {
    mockedPost.mockResolvedValue(
      mockApiResponse({
        id: 1,
        card_id: 'a',
        user_id: 1,
        title: 'Article',
        body: '',
        link: '',
        is_deleted: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        parent_id: 0,
        parent: null,
        files: [],
        children: [],
        references: [],
        tags: [],
        tasks: [],
        entities: [],
      }),
    );

    await createArticle(
      'https://example.com/article',
      'parent-card',
      'tag1,tag2',
    );
    expect(mockedPost).toHaveBeenCalledWith('/articles', {
      url: 'https://example.com/article',
      card_id: 'parent-card',
      tags: 'tag1,tag2',
    });
  });
});

describe('getNextRootId', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('GETs /cards/next-root-id and returns data via getData', async () => {
    mockedGet.mockResolvedValue(mockApiResponse({ next_id: 123 }));

    const result = await getNextRootId();
    expect(mockedGet).toHaveBeenCalledWith('/cards/next-root-id');
    expect(result).toEqual({ next_id: 123 });
  });
});
