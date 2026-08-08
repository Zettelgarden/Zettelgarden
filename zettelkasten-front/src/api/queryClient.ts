/**
 * React Query setup and configuration for Zettelgarden
 *
 * This module provides:
 * - QueryClient instance with optimized defaults
 * - Query key factory for consistent cache management
 * - Mutation key factory for mutation tracking
 */

import { QueryClient } from '@tanstack/react-query';

/**
 * Create and configure the React Query client
 *
 * Configuration rationale:
 * - staleTime: 5 minutes - Data stays fresh for 5 minutes, reducing unnecessary refetches
 * - gcTime: 10 minutes - Keep unused data in cache for 10 minutes (previously cacheTime)
 * - retry: 1 - Retry failed requests once, but not indefinitely
 * - refetchOnWindowFocus: false - Don't refetch when window regains focus (reduces server load)
 * - defaultOptions.queries structuralSharing: true - Enable structural sharing for cache efficiency
 */
export const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 5 * 60 * 1000, // 5 minutes
        gcTime: 10 * 60 * 1000, // 10 minutes
        retry: 1,
        refetchOnWindowFocus: false,
        structuralSharing: true,
      },
      mutations: {
        retry: 1,
      },
    },
  });

/**
 * Query key factory
 *
 * Centralized query key management ensures:
 * - Consistent cache keys across the application
 * - Easy cache invalidation via queryClient.invalidateQueries()
 * - Type safety for query parameters
 *
 * Query key structure: [entity, operation, identifier, params]
 */
export const queryKeys = {
  // Auth queries
  auth: {
    all: ['auth'] as const,
    current: () => ['auth', 'current'] as const,
    admin: () => ['auth', 'admin'] as const,
    subscription: (userId: number) => ['auth', 'subscription', userId] as const,
  },

  // Task queries
  tasks: {
    all: ['tasks'] as const,
    lists: () => ['tasks', 'list'] as const,
    list: (filters: TaskListFilters) => ['tasks', 'list', filters] as const,
    details: () => ['tasks', 'detail'] as const,
    detail: (id: number) => ['tasks', 'detail', id] as const,
    auditEvents: (id: number) => ['tasks', 'audit', id] as const,
    dependencies: (id: number) => ['tasks', 'dependencies', id] as const,
  },

  // Card queries
  cards: {
    all: ['cards'] as const,
    lists: () => ['cards', 'list'] as const,
    list: (filters: CardListFilters) => ['cards', 'list', filters] as const,
    details: () => ['cards', 'detail'] as const,
    detail: (id: string | number) => ['cards', 'detail', String(id)] as const,
    search: (params: SearchParams) => ['cards', 'search', params] as const,
    starred: () => ['cards', 'starred'] as const,
    unsorted: (page: number, perPage: number) =>
      ['cards', 'unsorted', page, perPage] as const,
    references: (id: string) => ['cards', id, 'references'] as const,
    children: (id: string) => ['cards', id, 'children'] as const,
    files: (id: string) => ['cards', id, 'files'] as const,
    tags: (id: string) => ['cards', id, 'tags'] as const,
    tasks: (id: string) => ['cards', id, 'tasks'] as const,
    entities: (id: string) => ['cards', id, 'entities'] as const,
    linkedEntities: (id: string) => ['cards', id, 'linked-entities'] as const,
    auditEvents: (id: string) => ['cards', id, 'audit'] as const,
  },

  // Tag queries
  tags: {
    all: ['tags'] as const,
    lists: () => ['tags', 'list'] as const,
  },

  // Entity queries
  entities: {
    all: ['entities'] as const,
    lists: () => ['entities', 'list'] as const,
    detail: (id: number) => ['entities', 'detail', id] as const,
  },

  // Fact queries
  facts: {
    all: ['facts'] as const,
    lists: () => ['facts', 'list'] as const,
    detail: (id: number) => ['facts', 'detail', id] as const,
  },

  // File queries
  files: {
    all: ['files'] as const,
    upload: () => ['files', 'upload'] as const,
  },

  // Template queries
  templates: {
    all: ['templates'] as const,
    lists: () => ['templates', 'list'] as const,
    detail: (id: number) => ['templates', 'detail', id] as const,
  },

  // Starred searches
  starredSearches: {
    all: ['starred-searches'] as const,
  },

  // Stats queries
  stats: {
    all: ['stats'] as const,
  },

  // Task statuses
  taskStatuses: {
    all: ['task-statuses'] as const,
  },
};

/**
 * Type definitions for query key parameters
 */
export interface TaskListFilters {
  showCompleted?: boolean;
  scheduledDate?: Date | null;
  completedDate?: Date | null;
  status?: string | null;
}

export interface CardListFilters {
  page?: number;
  perPage?: number;
  onlyParentCards?: boolean;
}

export interface SearchParams {
  searchTerm: string;
  fullText?: boolean;
  showEntities?: boolean;
  showFacts?: boolean;
  showCards?: boolean;
  showEmails?: boolean;
  sortBy?: string;
  searchType?: string;
  rerank?: boolean;
  page?: number;
  perPage?: number;
  schemaId?: number;
  onlyEmptyCardId?: boolean;
}

/**
 * Mutation key factory for tracking mutations
 */
export const mutationKeys = {
  tasks: {
    create: () => ['tasks', 'create'] as const,
    update: (id: number) => ['tasks', 'update', id] as const,
    delete: (id: number) => ['tasks', 'delete', id] as const,
  },
  cards: {
    create: () => ['cards', 'create'] as const,
    update: (id: string | number) => ['cards', 'update', String(id)] as const,
    delete: (id: number) => ['cards', 'delete', id] as const,
    star: (id: number) => ['cards', 'star', id] as const,
    unstar: (id: number) => ['cards', 'unstar', id] as const,
  },
  tags: {
    create: () => ['tags', 'create'] as const,
    update: (id: number) => ['tags', 'update', id] as const,
    delete: (id: number) => ['tags', 'delete', id] as const,
  },
};
