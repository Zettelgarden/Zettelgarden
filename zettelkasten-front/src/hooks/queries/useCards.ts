/**
 * React Query hooks for card data fetching and mutations
 *
 * This module provides hooks for card operations with:
 * - Automatic caching and refetching
 * - Optimistic updates for star/unstar
 * - Proper error handling
 */

import { useQuery, useMutation, useQueryClient, UseQueryResult } from '@tanstack/react-query';
import { queryKeys, mutationKeys, SearchParams } from '../../api/queryClient';
import {
  getCard,
  saveNewCard,
  saveExistingCard,
  deleteCard,
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
  restoreCardToAuditEvent,
  createArticle,
} from '../../api/cards';
import { Card, PartialCard, Entity, SearchResult } from '../../models/Card';
import { CategorizedReferences, PaginatedSearchResponse } from '../../api/cards';

/**
 * Hook to fetch a single card by ID
 *
 * @param id - Card ID (string or number)
 * @param enabled - Whether to enable the query (default: true)
 * @returns Query result with card data
 *
 * @example
 * ```tsx
 * function CardView({ cardId }) {
 *   const { data: card, isLoading, error } = useCard(cardId);
 *
 *   if (isLoading) return <div>Loading...</div>;
 *   if (error) return <div>Error: {error.message}</div>;
 *
 *   return <div>{card?.title}</div>;
 * }
 * ```
 */
export function useCard(id: string | number, enabled = true): UseQueryResult<Card, Error> {
  return useQuery({
    queryKey: queryKeys.cards.detail(String(id)),
    queryFn: () => getCard(String(id)),
    enabled: !!id && enabled,
    staleTime: 5 * 60 * 1000, // 5 minutes - cards don't change that often
  });
}

/**
 * Hook to fetch card references (categorized)
 *
 * @param cardId - Card ID
 * @returns Query result with categorized references
 */
export function useCardReferences(cardId: string): UseQueryResult<CategorizedReferences, Error> {
  return useQuery({
    queryKey: queryKeys.cards.references(cardId),
    queryFn: () => getCardReferences(cardId),
    enabled: !!cardId,
  });
}

/**
 * Hook to fetch card children
 *
 * @param cardId - Card ID
 * @returns Query result with children array
 */
export function useCardChildren(cardId: string): UseQueryResult<PartialCard[], Error> {
  return useQuery({
    queryKey: queryKeys.cards.children(cardId),
    queryFn: () => getCardChildren(cardId),
    enabled: !!cardId,
  });
}

/**
 * Hook to fetch card files
 *
 * @param cardId - Card ID
 * @returns Query result with files array
 */
export function useCardFiles(cardId: string): UseQueryResult<any[], Error> {
  return useQuery({
    queryKey: queryKeys.cards.files(cardId),
    queryFn: () => getCardFiles(cardId),
    enabled: !!cardId,
  });
}

/**
 * Hook to fetch card tags
 *
 * @param cardId - Card ID
 * @returns Query result with tags array
 */
export function useCardTags(cardId: string): UseQueryResult<any[], Error> {
  return useQuery({
    queryKey: queryKeys.cards.tags(cardId),
    queryFn: () => getCardTags(cardId),
    enabled: !!cardId,
  });
}

/**
 * Hook to fetch card tasks
 *
 * @param cardId - Card ID
 * @returns Query result with tasks array
 */
export function useCardTasks(cardId: string): UseQueryResult<any[], Error> {
  return useQuery({
    queryKey: queryKeys.cards.tasks(cardId),
    queryFn: () => getCardTasks(cardId),
    enabled: !!cardId,
  });
}

/**
 * Hook to fetch card entities
 *
 * @param cardId - Card ID
 * @returns Query result with entities array
 */
export function useCardEntities(cardId: string): UseQueryResult<any[], Error> {
  return useQuery({
    queryKey: queryKeys.cards.entities(cardId),
    queryFn: () => getCardEntities(cardId),
    enabled: !!cardId,
  });
}

/**
 * Hook to fetch linked entities for a card
 *
 * @param cardId - Card ID
 * @returns Query result with linked entities array
 */
export function useLinkedEntities(cardId: string): UseQueryResult<Entity[], Error> {
  return useQuery({
    queryKey: queryKeys.cards.linkedEntities(cardId),
    queryFn: () => getLinkedEntitiesByCardPK(cardId),
    enabled: !!cardId,
  });
}

/**
 * Hook to fetch card audit events
 *
 * @param cardId - Card ID
 * @returns Query result with audit events
 */
export function useCardAuditEvents(cardId: string): UseQueryResult<any[], Error> {
  return useQuery({
    queryKey: queryKeys.cards.auditEvents(cardId),
    queryFn: () => getCardAuditEvents(cardId),
    enabled: !!cardId,
  });
}

/**
 * Hook to search cards
 *
 * @param params - Search parameters
 * @param enabled - Whether to enable the query
 * @returns Query result with paginated search results
 */
export function useCardSearch(
  params: SearchParams,
  enabled = true
): UseQueryResult<PaginatedSearchResponse, Error> {
  // Create a stable query key from params
  const queryParams = {
    searchTerm: params.searchTerm,
    fullText: params.fullText ?? false,
    showEntities: params.showEntities ?? false,
    showFacts: params.showFacts ?? true,
    showCards: params.showCards ?? true,
    showEmails: params.showEmails ?? false,
    sortBy: params.sortBy ?? 'sortByRanking',
    searchType: params.searchType ?? 'classic',
    rerank: params.rerank ?? true,
    page: params.page ?? 1,
    perPage: params.perPage ?? 50,
    schemaId: params.schemaId,
    onlyEmptyCardId: params.onlyEmptyCardId ?? false,
  };

  return useQuery({
    queryKey: queryKeys.cards.search(queryParams),
    queryFn: () =>
      semanticSearchCardsPaginated(
        queryParams.searchTerm,
        queryParams.fullText,
        queryParams.showEntities,
        queryParams.showFacts,
        queryParams.showCards,
        queryParams.showEmails,
        queryParams.sortBy,
        queryParams.searchType,
        queryParams.rerank,
        queryParams.page,
        queryParams.perPage,
        queryParams.onlyEmptyCardId,
        queryParams.schemaId
      ),
    enabled: enabled && (queryParams.searchTerm.length > 0 || queryParams.onlyEmptyCardId),
    staleTime: 2 * 60 * 1000, // 2 minutes - search results can become stale
  });
}

/**
 * Hook to fetch starred cards
 *
 * @returns Query result with starred cards array
 */
export function useStarredCards(): UseQueryResult<Card[], Error> {
  return useQuery({
    queryKey: queryKeys.cards.starred(),
    queryFn: getStarredCards,
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to fetch unsorted cards (paginated)
 *
 * @param page - Page number
 * @param perPage - Items per page
 * @returns Query result with unsorted cards
 */
export function useUnsortedCards(
  page = 1,
  perPage = 10
): UseQueryResult<{ cards: PartialCard[]; page: number; per_page: number; total: number; total_pages: number }, Error> {
  return useQuery({
    queryKey: queryKeys.cards.unsorted(page, perPage),
    queryFn: () => getUnsortedCards(page, perPage),
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Mutation hook to create a new card
 *
 * @returns Mutation with status and handlers
 */
export function useCreateCard() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: mutationKeys.cards.create(),
    mutationFn: saveNewCard,

    onSuccess: () => {
      // Invalidate card lists and searches
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.lists() });
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.search({} as any) });
    },
  });
}

/**
 * Mutation hook to update an existing card
 *
 * @returns Mutation with status and handlers
 */
export function useUpdateCard() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: saveExistingCard,

    onMutate: async (updatedCard: Card) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.cards.detail(String(updatedCard.id)) });

      const previousCard = queryClient.getQueryData(queryKeys.cards.detail(String(updatedCard.id)));

      queryClient.setQueryData(queryKeys.cards.detail(String(updatedCard.id)), updatedCard);

      return { previousCard };
    },

    onError: (error, variables, context) => {
      if (context?.previousCard) {
        queryClient.setQueryData(
          queryKeys.cards.detail(String(variables.id)),
          context.previousCard
        );
      }
    },

    onSettled: (newCard) => {
      if (newCard) {
        queryClient.invalidateQueries({ queryKey: queryKeys.cards.detail(String(newCard.id)) });
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.lists() });
    },
  });
}

/**
 * Mutation hook to delete a card
 *
 * @returns Mutation with status and handlers
 */
export function useDeleteCard() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteCard,

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.lists() });
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.starred() });
    },
  });
}

/**
 * Mutation hook to star a card
 *
 * Supports optimistic updates.
 *
 * @returns Mutation with status and handlers
 */
export function useStarCard() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (cardId: number) => starCard(cardId),

    onMutate: async (cardId: number) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.cards.detail(String(cardId)) });

      const previousCard = queryClient.getQueryData<Card>(queryKeys.cards.detail(String(cardId)));

      if (previousCard) {
        queryClient.setQueryData(queryKeys.cards.detail(String(cardId)), {
          ...previousCard,
          is_pinned: true,
        });
      }

      return { previousCard };
    },

    onError: (error, variables, context) => {
      if (context?.previousCard) {
        queryClient.setQueryData(
          queryKeys.cards.detail(String(variables)),
          context.previousCard
        );
      }
    },

    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.starred() });
    },
  });
}

/**
 * Mutation hook to unstar a card
 *
 * Supports optimistic updates.
 *
 * @returns Mutation with status and handlers
 */
export function useUnstarCard() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (cardId: number) => unstarCard(cardId),

    onMutate: async (cardId: number) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.cards.detail(String(cardId)) });

      const previousCard = queryClient.getQueryData<Card>(queryKeys.cards.detail(String(cardId)));

      if (previousCard) {
        queryClient.setQueryData(queryKeys.cards.detail(String(cardId)), {
          ...previousCard,
          is_pinned: false,
        });
      }

      return { previousCard };
    },

    onError: (error, variables, context) => {
      if (context?.previousCard) {
        queryClient.setQueryData(
          queryKeys.cards.detail(String(variables)),
          context.previousCard
        );
      }
    },

    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.starred() });
    },
  });
}

/**
 * Mutation hook to restore a card to a previous audit event state
 *
 * @returns Mutation with status and handlers
 */
export function useRestoreCardToAuditEvent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ cardId, auditEventId }: { cardId: string; auditEventId: number }) =>
      restoreCardToAuditEvent(cardId, auditEventId),

    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.detail(variables.cardId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.auditEvents(variables.cardId) });
    },
  });
}

/**
 * Mutation hook to create an article from a URL
 *
 * @returns Mutation with status and handlers
 */
export function useCreateArticle() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ url, cardId, tags }: { url: string; cardId?: string; tags?: string }) =>
      createArticle(url, cardId, tags),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.cards.lists() });
    },
  });
}
