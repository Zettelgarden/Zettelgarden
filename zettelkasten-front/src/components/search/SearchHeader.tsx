import React, { useState } from "react";
import { SearchConfig } from "../../models/StarredSearch";

interface SearchHeaderProps {
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  searchConfig: SearchConfig;
  onSearch: (searchTerm: string, config: SearchConfig) => void;
  onToggleFilters: () => void;
  setShowStarSearchDialog: (show: boolean) => void;
  starredId: string | null;
  totalResults: number;
  currentPage: number;
  totalPages: number;
  isLoading: boolean;
  showFilters: boolean;
}

/**
 * Prominent search header with primary search input and actions
 * Replaces the duplicated search bars from filters panel and results panel
 */
export function SearchHeader({
  searchTerm,
  setSearchTerm,
  searchConfig,
  onSearch,
  onToggleFilters,
  setShowStarSearchDialog,
  starredId,
  totalResults,
  currentPage,
  totalPages,
  isLoading,
  showFilters,
}: SearchHeaderProps) {

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" && !isLoading) {
      onSearch(searchTerm, searchConfig);
    }
  };

  return (
    <div className="flex-shrink-0 border-b border-gray-200 p-4">
      {/* Primary Search Bar */}
      <div className="flex items-center gap-3 mb-3">
        <div className="flex-grow relative">
          <input
            type="text"
            value={searchTerm}
            placeholder="Search cards, entities, facts..."
            onChange={(e) => setSearchTerm(e.target.value)}
            onKeyPress={handleKeyPress}
            disabled={isLoading}
            className="w-full h-12 px-4 pr-12 border border-slate-300 rounded-lg text-base focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-50 shadow-sm"
          />
          <div className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
        </div>
        <button
          onClick={() => onSearch(searchTerm, searchConfig)}
          disabled={isLoading}
          className="h-12 px-6 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-base font-medium flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors shadow-sm"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          Search
        </button>
      </div>

      {/* Action Buttons */}
      <div className="flex items-center gap-2">
        {/* Advanced Filters Toggle */}
        <button
          onClick={onToggleFilters}
          className={`px-3 py-2 text-sm font-medium rounded-md transition-colors flex items-center gap-2 ${
            showFilters
              ? 'bg-blue-50 text-blue-700 border border-blue-200'
              : 'text-gray-700 hover:bg-gray-100 border border-transparent'
          }`}
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4" />
          </svg>
          Options
          {showFilters && (
            <svg className="w-4 h-4 transform rotate-180" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          )}
        </button>

        {/* Star Search Button */}
        {!starredId && (
          <button
            onClick={() => setShowStarSearchDialog(true)}
            className="px-3 py-2 text-sm text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-md transition-colors flex items-center gap-2"
          >
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
            </svg>
            Star This Search
          </button>
        )}
      </div>

      {/* Results count indicator */}
      {!isLoading && totalResults > 0 && (
        <div className="mt-3 text-sm text-gray-600">
          Found {totalResults} result{totalResults !== 1 ? 's' : ''}
          {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
        </div>
      )}
    </div>
  );
}
