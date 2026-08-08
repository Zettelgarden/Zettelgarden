import React from 'react';
import { SearchResult } from '../../models/Card';
import { SearchConfig } from '../../models/StarredSearch';
import { Button } from '../../components/Button';
import { SearchForm } from './SearchForm';
import { SearchConfig as SearchConfigComponent } from './SearchConfig';
import { SearchResultList } from '../../components/cards/SearchResultList';

interface SearchResultsPanelProps {
  // Search input state
  searchTerm: string;
  setSearchTerm: (searchTerm: string) => void;

  // Search configuration
  searchConfig: SearchConfig;
  setSearchConfig: (config: SearchConfig) => void;

  // Results data
  searchResults: SearchResult[];
  totalResults: number;
  totalPages: number;
  currentPage: number;

  // Loading and error states
  isLoading: boolean;
  error: Error | null;

  // Event handlers
  onSearch: (searchTerm: string, config: SearchConfig) => void;
  onPageChange: (newPage: number) => void;
  onResultClick: (result: SearchResult) => void;
  onEntityClick: (entityName: string) => void;
  onTagClick: (tagName: string) => void;
  onResultsUpdate: (results: SearchResult[]) => void;

  // Search options
  tags: string[];
  starredId?: string | null;
  setShowStarSearchDialog: (show: boolean) => void;
}

export function SearchResultsPanel({
  searchTerm,
  setSearchTerm,
  searchConfig,
  setSearchConfig,
  searchResults,
  totalResults,
  totalPages,
  currentPage,
  isLoading,
  error,
  onSearch,
  onPageChange,
  onResultClick,
  onEntityClick,
  onTagClick,
  onResultsUpdate,
  tags = [],
  starredId,
  setShowStarSearchDialog,
}: SearchResultsPanelProps) {
  function getFilteredResults(): SearchResult[] {
    return searchResults.filter(
      (result) => !searchConfig.onlyParentCards || !result.id.includes('/'),
    );
  }

  const handleResultClick = (result: SearchResult) => {
    onResultClick(result);
  };

  const handleSearchTrigger = (config: SearchConfig, resetPage?: boolean) => {
    onSearch(searchTerm, config);
  };

  return (
    <div className="flex flex-col h-full bg-white">
      {/* Search Header Section */}
      <div className="flex-shrink-0 border-b border-gray-200 p-4">
        <div className="flex items-center gap-2 mb-3">
          <SearchForm
            searchTerm={searchTerm}
            setSearchTerm={setSearchTerm}
            searchConfig={searchConfig}
            onSearch={onSearch}
            disabled={isLoading}
          />
          <SearchConfigComponent
            searchTerm={searchTerm}
            searchConfig={searchConfig}
            setSearchConfig={setSearchConfig}
            tags={tags.map((name, id) => ({ id, name, color: '', user_id: 0 }))}
            starredId={starredId}
            setShowStarSearchDialog={setShowStarSearchDialog}
            onTagClick={onTagClick}
            onSearchTrigger={handleSearchTrigger}
          />
        </div>

        {/* Results count indicator */}
        {!isLoading && searchResults.length > 0 && (
          <div className="text-sm text-gray-600">
            Found {totalResults} result{totalResults !== 1 ? 's' : ''}
            {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
          </div>
        )}
      </div>

      {/* Results List Section */}
      <div className="flex-grow overflow-y-auto">
        {isLoading ? (
          <div className="flex flex-col items-center justify-center h-full py-20">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
            <p className="text-gray-600">Searching...</p>
          </div>
        ) : (
          <div className="p-4">
            {getFilteredResults().length > 0 ? (
              <div>
                <SearchResultList
                  results={getFilteredResults()}
                  showPreview={searchConfig.showPreview}
                  onEntityClick={onEntityClick}
                  onTagClick={onTagClick}
                  onResultsUpdate={onResultsUpdate}
                />

                {/* Pagination Controls */}
                {totalPages > 1 && (
                  <div className="flex justify-center items-center gap-4 mt-6 p-4 border-t border-gray-200">
                    <Button
                      onClick={() => onPageChange(currentPage - 1)}
                      disabled={currentPage === 1}
                      variant="outline"
                      size="small"
                    >
                      Previous
                    </Button>
                    <span className="flex items-center text-sm text-gray-600">
                      Page {currentPage} of {totalPages}
                    </span>
                    <Button
                      onClick={() => onPageChange(currentPage + 1)}
                      disabled={currentPage === totalPages}
                      variant="outline"
                      size="small"
                    >
                      Next
                    </Button>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-20">
                {error === null ? (
                  <>
                    <div className="text-gray-400 mb-4">
                      <svg
                        className="w-16 h-16 mx-auto"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                        />
                      </svg>
                    </div>
                    <p className="text-gray-600 text-center">
                      {searchTerm
                        ? 'No results found'
                        : 'Enter a search term to find cards, entities, and facts'}
                    </p>
                  </>
                ) : (
                  <div className="text-center">
                    <div className="text-red-400 mb-4">
                      <svg
                        className="w-16 h-16 mx-auto"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                        />
                      </svg>
                    </div>
                    <p className="text-red-600">
                      Search returned an error: {error.message}
                    </p>
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
