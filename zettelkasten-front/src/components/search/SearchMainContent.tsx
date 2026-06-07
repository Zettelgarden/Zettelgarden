import React, { useState } from "react";
import { SearchResult } from "../../models/Card";
import { SearchConfig } from "../../models/StarredSearch";
import { Button } from "../../components/Button";
import { SearchResultList } from "../../components/cards/SearchResultList";
import { SearchHeader } from "./SearchHeader";
import { AdvancedFiltersPanel } from "./AdvancedFiltersPanel";

interface SearchMainContentProps {
  // Search input state
  searchTerm: string;
  setSearchTerm: (searchTerm: string) => void;

  // Search configuration
  searchConfig: SearchConfig;
  setSearchConfig: (config: SearchConfig) => void;

  // Results data
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;
  totalResults: number;
  totalPages: number;
  currentPage: number;

  // Loading and error states
  isLoading: boolean;
  error: Error | null;

  // Event handlers
  onSearch: (searchTerm: string, config: SearchConfig) => void;
  onPageChange: (newPage: number) => void;
  onEntityClick: (entityName: string) => void;
  onTagClick: (tagName: string) => void;

  // Starred search
  starredId: string | null;
  setShowStarSearchDialog: (show: boolean) => void;
}

/**
 * Main content area for desktop search
 * Contains the prominent search header, collapsible filters, and results list
 * Clicking a result navigates to the card detail page
 */
export function SearchMainContent({
  searchTerm,
  setSearchTerm,
  searchConfig,
  setSearchConfig,
  searchResults,
  setSearchResults,
  totalResults,
  totalPages,
  currentPage,
  isLoading,
  error,
  onSearch,
  onPageChange,
  onEntityClick,
  onTagClick,
  starredId,
  setShowStarSearchDialog,
}: SearchMainContentProps) {
  const [showFilters, setShowFilters] = useState(false);

  function getFilteredResults(): SearchResult[] {
    return searchResults
      .filter(result => !searchConfig.onlyParentCards || !result.id.includes("/"));
  }

  const handleToggleFilters = () => {
    setShowFilters(!showFilters);
  };

  const handleApplyFilters = (newConfig?: SearchConfig) => {
    onSearch(searchTerm, newConfig ?? searchConfig);
  };

  return (
    <div className="flex flex-col flex-grow h-full bg-white overflow-hidden">
      {/* Prominent Search Header */}
      <SearchHeader
        searchTerm={searchTerm}
        setSearchTerm={setSearchTerm}
        searchConfig={searchConfig}
        onSearch={onSearch}
        onToggleFilters={handleToggleFilters}
        setShowStarSearchDialog={setShowStarSearchDialog}
        starredId={starredId}
        totalResults={totalResults}
        currentPage={currentPage}
        totalPages={totalPages}
        isLoading={isLoading}
        showFilters={showFilters}
      />

      {/* Collapsible Advanced Filters Panel */}
      <AdvancedFiltersPanel
        searchConfig={searchConfig}
        setSearchConfig={setSearchConfig}
        onApply={handleApplyFilters}
        isOpen={showFilters}
      />

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
                  onResultsUpdate={setSearchResults}
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
                      <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                      </svg>
                    </div>
                    <p className="text-gray-600 text-center">
                      {searchTerm ? 'No results found' : 'Enter a search term to find cards, entities, and facts'}
                    </p>
                  </>
                ) : (
                  <div className="text-center">
                    <div className="text-red-400 mb-4">
                      <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    </div>
                    <p className="text-red-600">Search returned an error: {error.message}</p>
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
