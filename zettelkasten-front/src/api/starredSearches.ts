import { StarredSearch, SearchConfig } from '../models/StarredSearch';
import { apiClient, getData } from './client';

/**
 * Save a search configuration to starred searches
 * @param title The title for the starred search
 * @param searchTerm The search term
 * @param searchConfig The search configuration options
 * @returns A promise that resolves when the search is starred
 */
export function starSearch(
  title: string,
  searchTerm: string,
  searchConfig: SearchConfig,
): Promise<void> {
  return getData(
    apiClient.post<void>('/searches/star', {
      title,
      search_term: searchTerm,
      search_config: searchConfig,
    }),
  );
}

/**
 * Remove a starred search
 * @param id The ID of the starred search to remove
 * @returns A promise that resolves when the search is unstarred
 */
export function unstarSearch(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/searches/star/${id}`));
}

/**
 * Get all starred searches for the current user
 * @returns A promise that resolves to an array of starred searches
 */
export function getStarredSearches(): Promise<StarredSearch[]> {
  return getData(apiClient.get<StarredSearch[]>('/searches/starred'))
    .then((starredSearches) => starredSearches || [])
    .then((starredSearches) =>
      starredSearches.map((starredSearch) => {
        return {
          ...starredSearch,
          created_at: new Date(starredSearch.created_at),
        };
      }),
    );
}
