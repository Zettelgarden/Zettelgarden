/**
 * Smoke test: SearchPage renders results and accepts input without crashing (a5q.6).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { SearchPage } from '../cards/SearchPage';
import { SearchConfig } from '../../models/StarredSearch';
import { SearchResult } from '../../models/Card';

const defaultConfig: SearchConfig = {
  useClassicSearch: true,
  searchType: 'classic',
  showEntities: false,
  showFacts: true,
  showCards: true,
  showEmails: false,
  useFullText: false,
  onlyParentCards: false,
  onlyEmptyCardId: false,
  rerank: true,
  sortBy: 'sortByRanking',
  showPreview: true,
  currentPage: 1,
};

const results: SearchResult[] = [
  {
    id: '1',
    type: 'card',
    title: 'Search Result Card',
    preview: 'A preview snippet',
    score: 0.9,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    tags: [],
    metadata: { id: 1, card_id: 'result-1', parent_id: 0 },
  },
];

vi.mock('../../api/cards', () => ({
  semanticSearchCards: vi.fn(),
  semanticSearchCardsPaginated: vi.fn(),
}));

vi.mock('../../api/tags', () => ({
  fetchUserTags: vi.fn(async () => []),
}));

vi.mock('../../api/starredSearches', () => ({
  getStarredSearches: vi.fn(async () => []),
}));

vi.mock('../../api/entities', () => ({
  fetchEntityByName: vi.fn(),
}));

import { mockEndpoint } from '../../tests/fetchMock';

const { semanticSearchCardsPaginated } = await import('../../api/cards');

function renderSearchPage() {
  const props = {
    searchTerm: '',
    setSearchTerm: vi.fn(),
    searchResults: results,
    setSearchResults: vi.fn(),
    searchConfig: defaultConfig,
    setSearchConfig: vi.fn(),
  };
  renderWithProviders(<SearchPage {...props} />);
  return props;
}

describe('SearchPage smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // SearchFiltersPanel fetches schemas on mount; stub explicitly so the
    // fetch settles instead of failing loudly (see subagent review M2).
    mockEndpoint('/schemas', []);
  });

  it('renders the search form and result titles', async () => {
    // SearchPage runs a search on mount; the mock must return a payload.
    vi.mocked(semanticSearchCardsPaginated).mockResolvedValue({
      results,
      page: 1,
      per_page: 20,
      total: 1,
      total_pages: 1,
    });

    renderSearchPage();

    expect(
      screen.getByPlaceholderText('Search cards, entities, facts...'),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByText('Search Result Card')).toBeInTheDocument(),
    );
  });

  it('accepts typing in the search box', async () => {
    vi.mocked(semanticSearchCardsPaginated).mockResolvedValue({
      results,
      page: 1,
      per_page: 20,
      total: 1,
      total_pages: 1,
    });

    const props = renderSearchPage();

    const input = screen.getByPlaceholderText(
      'Search cards, entities, facts...',
    );
    fireEvent.change(input, { target: { value: 'zettel' } });

    expect(props.setSearchTerm).toHaveBeenCalledWith('zettel');
  });
});
