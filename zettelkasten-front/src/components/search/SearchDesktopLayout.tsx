import React from 'react';
import { SearchResult } from '../../models/Card';
import { SearchConfig } from '../../models/StarredSearch';
import { Tag } from '../../models/Tags';
import { SearchSidebar } from './SearchSidebar';
import { SearchMainContent } from './SearchMainContent';

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
}

/**
 * Desktop two-panel layout for Search
 * Layout: Sidebar (navigation) | Main Content (search + results)
 * Clicking a result navigates to the card detail page
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
  tags,
  starredId,
  setShowStarSearchDialog,
  onSearch,
  onPageChange,
  onEntityClick,
  onTagClick,
}: SearchDesktopLayoutProps) {
  return (
    <div className="hidden md:flex flex-row h-screen overflow-hidden">
      {/* Left Panel: Sidebar with starred searches and tags 
      <SearchSidebar
        tags={tags}
        onTagClick={onTagClick}
      />
*/}
      {/* Right Panel: Main content with search header and results */}
      <SearchMainContent
        searchTerm={searchTerm}
        setSearchTerm={setSearchTerm}
        searchConfig={searchConfig}
        setSearchConfig={setSearchConfig}
        searchResults={searchResults}
        setSearchResults={setSearchResults}
        totalResults={totalResults}
        totalPages={totalPages}
        currentPage={currentPage}
        isLoading={isLoading}
        error={error}
        onSearch={onSearch}
        onPageChange={onPageChange}
        onEntityClick={onEntityClick}
        onTagClick={onTagClick}
        starredId={starredId}
        setShowStarSearchDialog={setShowStarSearchDialog}
      />
    </div>
  );
}
