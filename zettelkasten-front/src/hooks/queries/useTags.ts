/**
 * React Query hooks for tag data fetching and mutations
 *
 * This module provides hooks for tag operations.
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys, mutationKeys } from '../../api/queryClient';
import { fetchUserTags } from '../../api/tags';
import { Tag } from '../../models/Tags';

/**
 * Hook to fetch all user tags
 *
 * Replaces TagContext with automatic caching and refetching.
 *
 * @returns Query result with tags array (sorted alphabetically)
 *
 * @example
 * ```tsx
 * function TagList() {
 *   const { data: tags, isLoading, error } = useTags();
 *
 *   if (isLoading) return <div>Loading...</div>;
 *   if (error) return <div>Error: {error.message}</div>;
 *
 *   return <ul>{tags?.map(tag => <li key={tag.id}>{tag.name}</li>)}</ul>;
 * }
 * ```
 */
export function useTags() {
  return useQuery({
    queryKey: queryKeys.tags.all,
    queryFn: async () => {
      const tags = await fetchUserTags();
      return tags.sort((a, b) => a.name.localeCompare(b.name));
    },
    staleTime: 10 * 60 * 1000, // 10 minutes - tags change infrequently
  });
}
