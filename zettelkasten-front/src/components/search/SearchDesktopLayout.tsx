import React from "react";
import { SearchResult } from "../../models/Card";
import { SearchConfig } from "../../models/StarredSearch";
import { Tag } from "../../models/Tags";
import { SearchFiltersPanel } from "./SearchFiltersPanel";
import { SearchResultsPanel } from "./SearchResultsPanel";
import { SearchCardDetailPanel } from "./SearchCardDetailPanel";

interface SearchDesktopLayoutProps {
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

  // Selected result for detail panel
  selectedResult: SearchResult | null;
  setSelectedResult: (result: SearchResult | null) => void;

  // Tags
  tags: Tag[];

  // Starred search
  starredId: string | null;
  setShowStarSearchDialog: (show: boolean) => void;

  // Event handlers
  onSearch: (searchTerm: string, config: SearchConfig) => void;
  onPageChange: (newPage: number) => void;
  onEntityClick: (entityName: string) => void;
  onTagClick: (tagName: string) => void;
  onEditCard?: (cardId: number) => void;
}

/**
 * Desktop three-panel layout for Search
 * Layout: Filters sidebar | Results panel | Card detail panel
 */
export function SearchDesktopLayout({
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
  selectedResult,
  setSelectedResult,
  tags,
  starredId,
  setShowStarSearchDialog,
  onSearch,
  onPageChange,
  onEntityClick,
  onTagClick,
  onEditCard,
}: SearchDesktopLayoutProps) {
  const handleConfigChangeWithSearch = (config: SearchConfig, resetPage?: boolean) => {
    if (resetPage) {
      onPageChange(1);
    }
    onSearch(searchTerm, config);
  };

  const handleResultClick = (result: SearchResult) => {
    setSelectedResult(result);
  };

  return (
    <div className="hidden md:flex flex-row h-screen overflow-hidden">
      {/* Left Panel: Filters */}
      <SearchFiltersPanel
        searchTerm={searchTerm}
        searchConfig={searchConfig}
        setSearchConfig={setSearchConfig}
        tags={tags}
        starredId={starredId}
        setShowStarSearchDialog={setShowStarSearchDialog}
        onTagClick={onTagClick}
        onSearchTrigger={handleConfigChangeWithSearch}
        onSearchTermChange={setSearchTerm}
        onSearch={onSearch}
        isLoading={isLoading}
      />

      {/* Middle Panel: Results */}
      <SearchResultsPanel
        searchTerm={searchTerm}
        setSearchTerm={setSearchTerm}
        searchConfig={searchConfig}
        setSearchConfig={setSearchConfig}
        searchResults={searchResults}
        totalResults={totalResults}
        totalPages={totalPages}
        currentPage={currentPage}
        isLoading={isLoading}
        error={error}
        onSearch={onSearch}
        onPageChange={onPageChange}
        onResultClick={handleResultClick}
        onEntityClick={onEntityClick}
        onTagClick={onTagClick}
        onResultsUpdate={setSearchResults}
        tags={tags.map(t => t.name)}
        starredId={starredId}
        setShowStarSearchDialog={setShowStarSearchDialog}
      />

      {/* Right Panel: Card Detail */}
      <SearchCardDetailPanel
        selectedCard={selectedResult}
        onEdit={onEditCard}
        onTagClick={onTagClick}
        onEntityClick={onEntityClick}
      />
    </div>
  );
}
