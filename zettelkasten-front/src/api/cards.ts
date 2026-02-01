import {
  Card,
  PartialCard,
  NextIdResponse,
  Entity,
  SearchResult,
  CardWithDescendants,
  processCardWithDescendants,
  defaultCard,
} from "../models/Card";
import { apiClient, getData } from "./client";

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
  searchTerm = "",
  fullText = false,
  showEntities = false,
  showFacts = true,
  showCards = true,
  sortBy = "sortByRanking",
  searchType = "classic",
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

  const { data: paginatedResponse } = await apiClient.post<PaginatedSearchResponse>(
    "/search",
    params
  );

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
  searchTerm = "",
  fullText = false,
  showEntities = false,
  showFacts = true,
  showCards = true,
  sortBy = "sortByRanking",
  searchType = "classic",
  rerank = true,
): Promise<SearchResult[]> {
  const response = await semanticSearchCardsPaginated(
    searchTerm,
    fullText,
    showEntities,
    showFacts,
    showCards,
    sortBy,
    searchType,
    rerank,
    1,
    1000 // Large page size to get most results
  );
  return response.results;
}

/**
 * Process card data from API
 */
function processCardFromAPI(card: Card): Card {
  return {
    ...card,
    created_at: new Date(card.created_at),
    updated_at: new Date(card.updated_at),
    children: card.children?.map((child) => ({
      ...child,
      created_at: new Date(child.created_at),
      updated_at: new Date(child.updated_at),
    })) || [],
    references: card.references?.map((ref) => ({
      ...ref,
      created_at: new Date(ref.created_at),
      updated_at: new Date(ref.updated_at),
    })) || [],
    tasks: card.tasks?.map((task) => ({
      ...task,
      scheduled_date: task.scheduled_date ? new Date(task.scheduled_date) : null,
      due_date: task.due_date ? new Date(task.due_date) : null,
      created_at: new Date(task.created_at),
      updated_at: new Date(task.updated_at),
      completed_at: task.completed_at ? new Date(task.completed_at) : null,
    })) || [],
  };
}

/**
 * Get a single card by ID
 */
export async function getCard(id: string): Promise<Card> {
  const encoded = encodeURIComponent(id);
  const { data: card } = await apiClient.get<Card>(`/cards/${encoded}`);
  return processCardFromAPI(card);
}

/**
 * Save a new card
 */
export async function saveNewCard(card: Card): Promise<Card> {
  card.card_id = card.card_id.trim();
  const { data } = await apiClient.post<Card>("/cards", card);
  return processCardFromAPI(data);
}

/**
 * Save an existing card
 */
export async function saveExistingCard(card: Card): Promise<Card> {
  const { data } = await apiClient.put<Card>(`/cards/${encodeURIComponent(card.id)}`, card);
  return processCardFromAPI(data);
}

/**
 * Delete a card
 */
export async function deleteCard(id: number): Promise<Card | null> {
  const encodedId = encodeURIComponent(id);
  const { response, data } = await apiClient.delete<Card>(`/cards/${encodedId}`);

  if (response.status === 204) {
    return null;
  }
  return data;
}

/**
 * Get audit events for a card
 */
export async function getCardAuditEvents(cardId: string): Promise<any[]> {
  const { data: events } = await apiClient.get<any[]>(
    `/cards/${encodeURIComponent(cardId)}/audit`
  );

  if (!events) {
    return [];
  }

  return events.map(event => ({
    ...event,
    created_at: new Date(event.created_at),
    updated_at: new Date(event.updated_at),
  }));
}

/**
 * Get files attached to a card
 */
export async function getCardFiles(cardId: string): Promise<any[]> {
  const { data: files } = await apiClient.get<any[]>(
    `/cards/${encodeURIComponent(cardId)}/files`
  );

  if (!files) {
    return [];
  }

  return files.map((file) => ({
    ...file,
    created_at: new Date(file.created_at),
    updated_at: new Date(file.updated_at),
  }));
}

/**
 * Get children of a card
 */
export async function getCardChildren(cardId: string): Promise<PartialCard[]> {
  const { data: children } = await apiClient.get<PartialCard[]>(
    `/cards/${encodeURIComponent(cardId)}/children`
  );

  if (!children) {
    return [];
  }

  return children.map((child) => ({
    ...child,
    created_at: new Date(child.created_at),
    updated_at: new Date(child.updated_at),
  }));
}

/**
 * Get tags for a card
 */
export async function getCardTags(cardId: string): Promise<any[]> {
  const { data: tags } = await apiClient.get<any[]>(
    `/cards/${encodeURIComponent(cardId)}/tags`
  );

  if (!tags) {
    return [];
  }
  return tags;
}

/**
 * Get entities linked to a card
 */
export async function getLinkedEntitiesByCardPK(cardId: string | number): Promise<Entity[]> {
  const response = await apiClient.fetchResponse(
    `/cards/${encodeURIComponent(cardId)}/linked-entities`
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
}

/**
 * Get entities for a card
 */
export async function getCardEntities(cardId: string | number): Promise<any[]> {
  const { data: entities } = await apiClient.get<any[]>(
    `/cards/${encodeURIComponent(cardId)}/entities`
  );

  if (!entities) {
    return [];
  }
  return entities;
}

/**
 * Get tasks for a card
 */
export async function getCardTasks(cardId: string | number): Promise<any[]> {
  const { data: tasks } = await apiClient.get<any[]>(
    `/cards/${encodeURIComponent(cardId)}/tasks`
  );

  if (!tasks) {
    return [];
  }

  return tasks.map((task) => ({
    ...task,
    scheduled_date: task.scheduled_date ? new Date(task.scheduled_date) : null,
    due_date: task.due_date ? new Date(task.due_date) : null,
    created_at: new Date(task.created_at),
    updated_at: new Date(task.updated_at),
    completed_at: task.completed_at ? new Date(task.completed_at) : null,
  }));
}

/**
 * Get external events for a card
 */
export async function getCardExternalEvents(cardId: string | number): Promise<any[]> {
  const { data: events } = await apiClient.get<any[]>(
    `/cards/${encodeURIComponent(cardId)}/external-events`
  );

  if (!events) {
    return [];
  }

  return events.map((event) => ({
    ...event,
    start_time: new Date(event.start_time),
    end_time: new Date(event.end_time),
    created_at: new Date(event.created_at),
    updated_at: new Date(event.updated_at),
  }));
}

export interface CategorizedReferences {
  bidirectional: PartialCard[]; // Two-way links (mutual references)
  outgoing: PartialCard[];      // One-way links (this card references them)
  incoming: PartialCard[];      // One-way links (they reference this card)
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
export async function getCardReferences(cardId: string): Promise<CategorizedReferences> {
  const { data: refs } = await apiClient.get<CategorizedReferences>(
    `/cards/${encodeURIComponent(cardId)}/references`
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
}

/**
 * Get the next root card ID
 */
export async function getNextRootId(): Promise<NextIdResponse> {
  return getData(apiClient.get<NextIdResponse>("/cards/next-root-id"));
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
  const { data: starredCards } = await apiClient.get<any[]>("/cards/starred");

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
      children: card.children?.map((child: any) => ({
        ...child,
        created_at: new Date(child.created_at),
        updated_at: new Date(child.updated_at),
      })) || [],
      references: card.references?.map((ref: any) => ({
        ...ref,
        created_at: new Date(ref.created_at),
        updated_at: new Date(ref.updated_at),
      })) || [],
      tasks: card.tasks?.map((task: any) => ({
        ...task,
        scheduled_date: task.scheduled_date ? new Date(task.scheduled_date) : null,
        due_date: task.due_date ? new Date(task.due_date) : null,
        created_at: new Date(task.created_at),
        updated_at: new Date(task.updated_at),
        completed_at: task.completed_at ? new Date(task.completed_at) : null,
      })) || [],
      is_pinned: true, // Mark as starred since it's coming from the starred cards endpoint
    };
  });
}

/**
 * Suggest a title for a card based on its body content using AI
 */
export async function suggestCardTitle(body: string): Promise<string> {
  const { data } = await apiClient.post<{ suggested_title: string }>(
    "/cards/suggest-title",
    { body }
  );
  return data.suggested_title;
}

/**
 * Get the hierarchical card structure with depth information for CardTreeView component
 */
export async function getCardWithDescendants(cardId: string | number): Promise<CardWithDescendants> {
  const { data: card } = await apiClient.get<CardWithDescendants>(
    `/cards/${encodeURIComponent(cardId)}/tree`
  );
  // Recursively process dates and prepare for frontend use
  return processCardWithDescendants(card);
}

/**
 * Get card tree with limited depth
 */
export async function getCardWithDescendantsLimited(
  cardId: string | number,
  maxDepth: number
): Promise<CardWithDescendants> {
  const { data: card } = await apiClient.get<CardWithDescendants>(
    `/cards/${encodeURIComponent(cardId)}/tree/depth/${maxDepth}`
  );
  // Recursively process dates and prepare for frontend use
  return processCardWithDescendants(card);
}

interface UnsortedCardsResponse {
  cards: PartialCard[];
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

/**
 * Fetch unsorted cards (cards with empty card_id)
 */
export async function getUnsortedCards(page = 1, perPage = 10): Promise<UnsortedCardsResponse> {
  const { data } = await apiClient.get<UnsortedCardsResponse>(
    `/cards/unsorted?page=${page}&per_page=${perPage}`
  );

  const cards = data.cards.map((card) => ({
    ...card,
    created_at: new Date(card.created_at),
    updated_at: new Date(card.updated_at),
  }));

  return {
    ...data,
    cards,
  };
}

/**
 * Restore a card to the state it was in at the time of the audit event
 */
export async function restoreCardToAuditEvent(
  cardId: string,
  auditEventId: number
): Promise<Card> {
  const { data: card } = await apiClient.post<Card>(
    `/cards/${encodeURIComponent(cardId)}/audit/${auditEventId}/restore`,
    undefined
  );
  return processCardFromAPI(card);
}

/**
 * Create an article card from a URL
 */
export async function createArticle(
  url: string,
  cardId?: string,
  tags?: string
): Promise<Card> {
  const { data: card } = await apiClient.post<Card>("/articles", {
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
    body: cardData.body || "",
    link: cardData.link || "",
    parent_id: cardData.parent_id || 0,
    tags: cardData.tags?.map(tag => ({ id: 0, name: tag, color: "black", user_id: 1 })) || [],
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
  page = 1
): Promise<SearchResult[]> {
  const response = await semanticSearchCardsPaginated(
    query,
    fullText,
    false, // showEntities
    true,  // showFacts
    true,  // showCards
    "sortByRanking",
    "classic",
    true,  // rerank
    page,
    perPage
  );
  return response.results;
}
