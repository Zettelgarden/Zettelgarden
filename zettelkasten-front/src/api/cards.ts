import {
  Card,
  PartialCard,
  NextIdResponse,
  Entity,
  SearchResult,
  defaultCard,
  RelatedCard,
} from '../models/Card';
import { apiClient, getData } from './client';
import { getDataProvider } from '../data/provider';
import { processCardFromAPI as httpProcessCardFromAPI } from '../data/httpProvider';
import { graceful, EMPTY_REFERENCES } from '../data/offline';

/**
 * Process card data from API (shared by online-only card reads; the
 * offline-writable reads route through the data provider).
 */
function processCardFromAPI(card: Card): Card {
  return httpProcessCardFromAPI(card);
}

// Utility function to escape entity names for search queries
export function escapeEntityNameForSearch(entityName: string): string {
  // Escape characters that could break the @[...] syntax
  return entityName.replace(/[\[\]\\]/g, '\\$&');
}

interface SearchRequestParams {
  search_term: string;
  sort?: string;
  full_text?: boolean;
  show_entities?: boolean;
  show_facts?: boolean;
  show_cards?: boolean;
  only_empty_card_id?: boolean;
  schema_id?: number;
  search_type?: string; // classic or typesense
  rerank?: boolean;
  page?: number;
  per_page?: number;
}

export interface PaginatedSearchResponse {
  results: SearchResult[];
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

/**
 * Process search results to convert date strings to Date objects
 */
function processSearchResults(results: SearchResult[]): SearchResult[] {
  return results.map((result) => ({
    ...result,
    created_at: new Date(result.created_at),
    updated_at: new Date(result.updated_at),
  }));
}

/**
 * Semantic search with pagination
 */
export async function semanticSearchCardsPaginated(
  searchTerm = '',
  fullText = false,
  showEntities = false,
  showFacts = true,
  showCards = true,
  showEmails = false,
  sortBy = 'sortByRanking',
  searchType = 'classic',
  rerank = true,
  page = 1,
  perPage = 50,
  onlyEmptyCardId = false,
  schemaId?: number,
): Promise<PaginatedSearchResponse> {
  const params: SearchRequestParams = {
    search_term: searchTerm,
    search_type: searchType,
    full_text: fullText,
    show_entities: showEntities,
    show_facts: showFacts,
    show_cards: showCards,
    only_empty_card_id: onlyEmptyCardId,
    schema_id: schemaId,
    sort: sortBy,
    rerank: rerank,
    page: page,
    per_page: perPage,
  };

  const response = await graceful<{ data: PaginatedSearchResponse | null }>(
    { data: null },
    () => apiClient.post<PaginatedSearchResponse>('/search', params),
  );
  const paginatedResponse = response?.data;

  if (!paginatedResponse || !paginatedResponse.results) {
    return {
      results: [],
      page: page,
      per_page: perPage,
      total: 0,
      total_pages: 0,
    };
  }

  return {
    ...paginatedResponse,
    results: processSearchResults(paginatedResponse.results),
  };
}

/**
 * Semantic search (backward compatible wrapper)
 */
export async function semanticSearchCards(
  searchTerm = '',
  fullText = false,
  showEntities = false,
  showFacts = true,
  showCards = true,
  showEmails = false,
  sortBy = 'sortByRanking',
  searchType = 'classic',
  rerank = true,
): Promise<SearchResult[]> {
  const response = await semanticSearchCardsPaginated(
    searchTerm,
    fullText,
    showEntities,
    showFacts,
    showCards,
    showEmails,
    sortBy,
    searchType,
    rerank,
    1,
    1000, // Large page size to get most results
  );
  return response.results;
}

/**
 * Get a single card by ID. Desktop: resolved from the local mirror (instant,
 * offline). Web: GET /cards/:id.
 */
export async function getCard(id: string): Promise<Card> {
  return getDataProvider().getCard(id);
}

/**
 * Save a new card. Desktop: writes the local mirror + outbox (offline-safe).
 */
export async function saveNewCard(card: Card): Promise<Card> {
  return getDataProvider().saveNewCard(card);
}

/**
 * Save an existing card. Desktop: local mirror + outbox (offline-safe).
 */
export async function saveExistingCard(card: Card): Promise<Card> {
  return getDataProvider().saveExistingCard(card);
}

/**
 * Delete a card. Desktop: queues a local delete, reconciles on reconnect.
 */
export async function deleteCard(id: number): Promise<Card | null> {
  return getDataProvider().deleteCard(id);
}

/**
 * Get audit events for a card
 */
export async function getCardAuditEvents(cardId: string): Promise<any[]> {
  return graceful<any[]>([] as any[], async () => {
    const { data: events } = await apiClient.get<any[]>(
      `/cards/${encodeURIComponent(cardId)}/audit`,
    );

    if (!events) {
      return [];
    }

    return events.map((event) => ({
      ...event,
      created_at: new Date(event.created_at),
      updated_at: new Date(event.updated_at),
    }));
  });
}
/**
 * Get files attached to a card
 */
export async function getCardFiles(cardId: string): Promise<any[]> {
  return graceful<any[]>([] as any[], async () => {
    const { data: files } = await apiClient.get<any[]>(
      `/cards/${encodeURIComponent(cardId)}/files`,
    );

    if (!files) {
      return [];
    }

    return files.map((file) => ({
      ...file,
      created_at: new Date(file.created_at),
      updated_at: new Date(file.updated_at),
    }));
  });
}
/**
 * Get children of a card. Desktop: computed from the local mirror.
 */
export async function getCardChildren(cardId: string): Promise<PartialCard[]> {
  return getDataProvider().getCardChildren(cardId);
}

/**
 * Get tags for a card. Desktop: derived from the body + mirror tags.
 */
export async function getCardTags(cardId: string): Promise<any[]> {
  return getDataProvider().getCardTags(cardId);
}

/**
 * Get entities linked to a card
 */
export async function getLinkedEntitiesByCardPK(
  cardId: string | number,
): Promise<Entity[]> {
  return graceful<Entity[]>([] as Entity[], async () => {
    const response = await apiClient.fetchResponse(
      `/cards/${encodeURIComponent(cardId)}/linked-entities`,
    );

    // Handle no content responses gracefully
    if (response.status === 204) {
      return [];
    }

    try {
      const entities: Entity[] | null = await response.json();
      if (!entities) {
        return [];
      }
      return entities;
    } catch (err) {
      // In case of empty body or invalid JSON
      return [];
    }
  });
}
/**
 * Get entities for a card
 */
export async function getCardEntities(cardId: string | number): Promise<any[]> {
  return graceful<any[]>([] as any[], async () => {
    const { data: entities } = await apiClient.get<any[]>(
      `/cards/${encodeURIComponent(cardId)}/entities`,
    );

    if (!entities) {
      return [];
    }
    return entities;
  });
}
/**
 * Get tasks for a card. Desktop: computed from the local mirror.
 */
export async function getCardTasks(cardId: string | number): Promise<any[]> {
  return getDataProvider().getCardTasks(cardId);
}

export interface CategorizedReferences {
  bidirectional: PartialCard[]; // Two-way links (mutual references)
  outgoing: PartialCard[]; // One-way links (this card references them)
  incoming: PartialCard[]; // One-way links (they reference this card)
}

function processPartialCards(cards: PartialCard[]): PartialCard[] {
  return cards.map((ref) => ({
    ...ref,
    created_at: new Date(ref.created_at),
    updated_at: new Date(ref.updated_at),
  }));
}

/**
 * Get references for a card
 */
export async function getCardReferences(
  cardId: string,
): Promise<CategorizedReferences> {
  return graceful<CategorizedReferences>(EMPTY_REFERENCES as CategorizedReferences, async () => {
    const { data: refs } = await apiClient.get<CategorizedReferences>(
      `/cards/${encodeURIComponent(cardId)}/references`,
    );

    if (!refs) {
      return {
        bidirectional: [],
        outgoing: [],
        incoming: [],
      };
    }

    return {
      bidirectional: processPartialCards(refs.bidirectional),
      outgoing: processPartialCards(refs.outgoing),
      incoming: processPartialCards(refs.incoming),
    };
  });
}
/**
 * Get the next root card ID. Desktop: computed from the local mirror.
 */
export async function getNextRootId(): Promise<NextIdResponse> {
  return getDataProvider().getNextRootId();
}

/**
 * Star a card
 */
export async function starCard(cardId: number): Promise<void> {
  await apiClient.post(`/cards/${cardId}/star`, undefined);
}

/**
 * Unstar a card
 */
export async function unstarCard(cardId: number): Promise<void> {
  await apiClient.delete(`/cards/${cardId}/star`);
}

/**
 * Get all starred cards
 */
export async function getStarredCards(): Promise<Card[]> {
  return graceful<Card[]>([] as Card[], async () => {
    const { data: starredCards } = await apiClient.get<any[]>('/cards/starred');

    if (!starredCards) {
      return [];
    }

    // Transform the response into Card objects
    return starredCards.map((starredCard) => {
      const card = starredCard.card;

      return {
        ...card,
        created_at: new Date(card.created_at),
        updated_at: new Date(card.updated_at),
        children:
          card.children?.map((child: any) => ({
            ...child,
            created_at: new Date(child.created_at),
            updated_at: new Date(child.updated_at),
          })) || [],
        references:
          card.references?.map((ref: any) => ({
            ...ref,
            created_at: new Date(ref.created_at),
            updated_at: new Date(ref.updated_at),
          })) || [],
        tasks:
          card.tasks?.map((task: any) => ({
            ...task,
            scheduled_date: task.scheduled_date
              ? new Date(task.scheduled_date)
              : null,
            due_date: task.due_date ? new Date(task.due_date) : null,
            created_at: new Date(task.created_at),
            updated_at: new Date(task.updated_at),
            completed_at: task.completed_at ? new Date(task.completed_at) : null,
          })) || [],
        is_pinned: true, // Mark as starred since it's coming from the starred cards endpoint
      };
    });
  });
}
/**
 * Suggest a title for a card based on its body content using AI
 */
export async function suggestCardTitle(body: string): Promise<string> {
  const { data } = await apiClient.post<{ suggested_title: string }>(
    '/cards/suggest-title',
    { body },
  );
  return data.suggested_title;
}

interface UnsortedCardsResponse {
  cards: PartialCard[];
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

/**
 * Fetch unsorted cards (cards with empty card_id). Desktop: local mirror.
 */
export async function getUnsortedCards(
  page = 1,
  perPage = 10,
): Promise<UnsortedCardsResponse> {
  return getDataProvider().getUnsortedCards(page, perPage);
}

/**
 * Restore a card to the state it was in at the time of the audit event
 */
export async function restoreCardToAuditEvent(
  cardId: string,
  auditEventId: number,
): Promise<Card> {
  const { data: card } = await apiClient.post<Card>(
    `/cards/${encodeURIComponent(cardId)}/audit/${auditEventId}/restore`,
    undefined,
  );
  return processCardFromAPI(card);
}

/**
 * Create an article card from a URL
 */
export async function createArticle(
  url: string,
  cardId?: string,
  tags?: string,
): Promise<Card> {
  const { data: card } = await apiClient.post<Card>('/articles', {
    url,
    card_id: cardId,
    tags,
  });

  return {
    ...card,
    created_at: new Date(card.created_at),
    updated_at: new Date(card.updated_at),
  };
}

/**
 * Create a card - convenience wrapper for saveNewCard
 * Accepts partial card data for creating a new card
 */
export async function createCard(cardData: {
  title: string;
  body?: string;
  link?: string;
  parent_id?: number;
  tags?: string[];
}): Promise<Card> {
  const newCard: Card = {
    ...defaultCard,
    title: cardData.title,
    body: cardData.body || '',
    link: cardData.link || '',
    parent_id: cardData.parent_id || 0,
    tags:
      cardData.tags?.map((tag) => ({
        id: 0,
        name: tag,
        color: 'black',
        user_id: 1,
      })) || [],
  };
  return saveNewCard(newCard);
}

/**
 * Search cards - convenience wrapper for semanticSearchCardsPaginated
 * Returns simplified search results
 * @param query - Search query string
 * @param fullText - Whether to use full-text search (default false)
 * @param perPage - Number of results per page (default 10)
 * @param page - Page number (default 1)
 */
export async function searchCards(
  query: string,
  fullText = false,
  perPage = 10,
  page = 1,
): Promise<SearchResult[]> {
  const response = await semanticSearchCardsPaginated(
    query,
    fullText,
    false, // showEntities
    true, // showFacts
    true, // showCards
    false, // showEmails
    'sortByRanking',
    'classic',
    true, // rerank
    page,
    perPage,
  );
  return response.results;
}

/**
 * Get related cards for a given card
 */
export async function getRelatedCards(cardId: string): Promise<RelatedCard[]> {
  const { data: relatedCards } = await apiClient.get<RelatedCard[]>(
    `/cards/${encodeURIComponent(cardId)}/related`,
  );
  return relatedCards;
}
