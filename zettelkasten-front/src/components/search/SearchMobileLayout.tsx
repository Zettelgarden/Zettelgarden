import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { SearchResult } from '../../models/Card';
import { Tag } from '../../models/Tags';
import { SearchConfig as SearchConfigType } from '../../models/StarredSearch';
import { SchemaDefinition } from '../../models/Schema';
import { SearchResultList } from '../../components/cards/SearchResultList';
import { MobileView } from '../../hooks/useResponsiveLayout';

interface SearchMobileLayoutProps {
  mobileView: MobileView;
  setMobileView: (view: MobileView) => void;
  searchTerm: string;
  setSearchTerm: (searchTerm: string) => void;
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;
  searchConfig: SearchConfigType;
  setSearchConfig: (config: SearchConfigType) => void;
  tags: Tag[];
  isLoading: boolean;
  error: Error | null;
  totalResults: number;
  totalPages: number;
  currentPage: number;
  onSearch: (searchTerm: string, config: SearchConfigType) => void;
  handlePageChange: (newPage: number) => void;
  handleEntityClick: (entityName: string) => void;
  handleTagClick: (tagName: string) => void;
  onMenuClick: () => void;
  schemas?: SchemaDefinition[];
  starredId?: string | null;
  setShowStarSearchDialog?: (show: boolean) => void;
}

/**
 * Mobile layout for Search with results list and detail views
 */
export function SearchMobileLayout({
  mobileView,
  setMobileView,
  searchTerm,
  setSearchTerm,
  searchResults,
  setSearchResults,
  searchConfig,
  setSearchConfig,
  tags,
  isLoading,
  error,
  totalResults,
  totalPages,
  currentPage,
  onSearch,
  handlePageChange,
  handleEntityClick,
  handleTagClick,
  onMenuClick,
  schemas = [],
  starredId,
  setShowStarSearchDialog,
}: SearchMobileLayoutProps) {
  const navigate = useNavigate();
  const bottomSheetRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);

  // Handle escape key and backdrop click for filters
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && mobileView === 'filters') {
        setMobileView('list');
      }
    };

    if (mobileView === 'filters') {
      document.addEventListener('keydown', handleEscape);
      // Prevent body scroll when filters sheet is open
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [mobileView, setMobileView]);

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === backdropRef.current) {
      setMobileView('list');
    }
  };

  const handleConfigChange = (newConfig: SearchConfigType) => {
    setSearchConfig(newConfig);
    onSearch(searchTerm, newConfig);
    setMobileView('list');
  };

  // Mobile List View
  if (mobileView === 'list') {
    return (
      <div className="md:hidden flex flex-col flex-1 h-full">
        {/* Mobile Top Bar */}
        <div className="sticky top-0 z-40 bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between">
          {/* Left: Hamburger menu */}
          <button
            onClick={onMenuClick}
            className="p-2 -ml-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
            aria-label="Open menu"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 6h16M4 12h16M4 18h16"
              />
            </svg>
          </button>

          {/* Center: Title */}
          <div className="flex items-center gap-2">
            <h1 className="text-lg font-semibold text-gray-900">Search</h1>
          </div>

          {/* Right: Filters button */}
          <button
            onClick={() => setMobileView('filters')}
            className="p-2 -mr-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
            aria-label="Open filters"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
              />
            </svg>
          </button>
        </div>

        {/* Search Input Area */}
        <div className="bg-slate-100 p-4 border-b border-slate-300">
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={searchTerm}
              placeholder="Search cards, entities, facts..."
              onChange={(e) => setSearchTerm(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter' && !isLoading) {
                  onSearch(searchTerm, searchConfig);
                }
              }}
              disabled={isLoading}
              className="flex-1 h-9 px-3 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-50"
            />
            <button
              onClick={() => onSearch(searchTerm, searchConfig)}
              disabled={isLoading}
              className="h-9 bg-blue-600 hover:bg-blue-700 text-white rounded-md px-4 text-sm flex-shrink-0 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Search
            </button>
          </div>
        </div>

        {/* Results content */}
        <div className="flex-1 bg-white flex flex-col overflow-y-auto">
          {isLoading ? (
            <div className="flex justify-center items-center py-20">
              <div className="text-gray-500">Loading...</div>
            </div>
          ) : searchResults.length > 0 ? (
            <>
              <div className="flex-1 overflow-y-auto">
                <SearchResultList
                  results={searchResults.filter(
                    (result) =>
                      !searchConfig.onlyParentCards || !result.id.includes('/'),
                  )}
                  showPreview={searchConfig.showPreview}
                  onEntityClick={handleEntityClick}
                  onTagClick={handleTagClick}
                  onResultsUpdate={setSearchResults}
                />
              </div>

              {/* Pagination for mobile */}
              {totalPages > 1 && (
                <div className="p-4 border-t border-gray-200 bg-gray-50">
                  <div className="flex justify-center items-center gap-4">
                    <button
                      onClick={() => handlePageChange(currentPage - 1)}
                      disabled={currentPage === 1}
                      className="px-4 py-2 bg-white border border-gray-300 rounded-lg text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
                    >
                      Previous
                    </button>
                    <span className="text-sm text-gray-600">
                      {currentPage} / {totalPages}
                    </span>
                    <button
                      onClick={() => handlePageChange(currentPage + 1)}
                      disabled={currentPage === totalPages}
                      className="px-4 py-2 bg-white border border-gray-300 rounded-lg text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
                    >
                      Next
                    </button>
                  </div>
                  <div className="text-center text-xs text-gray-500 mt-2">
                    {totalResults} results
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center">
              <div className="text-center p-8">
                {error === null ? (
                  <>
                    <p className="text-gray-500 mb-2">No results found</p>
                    <p className="text-sm text-gray-400">
                      Try adjusting your search or filters
                    </p>
                  </>
                ) : (
                  <p className="text-red-500">Search error: {error.message}</p>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }

  // Mobile Detail View (for card detail)
  if (mobileView === 'detail') {
    // This would be handled by navigation to the card view page
    // The component just renders null as navigation happens via React Router
    return null;
  }

  // Mobile Filters View (Bottom Sheet)
  if (mobileView === 'filters') {
    return (
      <>
        {/* Background content - shows list behind the filter overlay */}
        <div className="md:hidden flex flex-col flex-1 h-full opacity-50">
          <div className="sticky top-0 z-40 bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between">
            <h1 className="text-lg font-semibold text-gray-900">Search</h1>
          </div>
          <div className="flex-1 overflow-y-auto">
            <SearchResultList
              results={searchResults.filter(
                (result) =>
                  !searchConfig.onlyParentCards || !result.id.includes('/'),
              )}
              showPreview={searchConfig.showPreview}
              onEntityClick={handleEntityClick}
              onTagClick={handleTagClick}
              onResultsUpdate={setSearchResults}
            />
          </div>
        </div>

        {/* Bottom Sheet Overlay */}
        <div
          ref={backdropRef}
          onClick={handleBackdropClick}
          className="fixed inset-0 bg-black/50 z-50 md:hidden"
          style={{ animation: 'fade-in 0.2s ease-out' }}
        >
          <div
            ref={bottomSheetRef}
            className="fixed bottom-0 left-0 right-0 bg-white rounded-t-2xl shadow-2xl max-h-[80vh] flex flex-col"
            style={{ animation: 'slide-up 0.3s ease-out' }}
          >
            {/* Drag Handle */}
            <div className="flex justify-center pt-3 pb-2 px-4 flex-shrink-0">
              <div className="w-12 h-1.5 bg-gray-300 rounded-full" />
            </div>

            {/* Header */}
            <div className="px-4 pb-3 border-b border-gray-200 flex-shrink-0">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold text-gray-900">
                  Filters & Options
                </h2>
                <button
                  onClick={() => setMobileView('list')}
                  className="p-2 -mr-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg"
                  aria-label="Close"
                >
                  <svg
                    className="w-5 h-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </div>
            </div>

            {/* Scrollable Content */}
            <div className="flex-1 overflow-y-auto px-4 py-3">
              {/* Sorting Section */}
              <div className="mb-6">
                <h3 className="text-sm font-semibold text-gray-900 mb-3">
                  Sort Results
                </h3>
                <div className="space-y-1">
                  {[
                    { value: 'sortByRanking', label: 'Ranking Score' },
                    { value: 'sortCreatedNewOld', label: 'Created (Newest)' },
                    { value: 'sortCreatedOldNew', label: 'Created (Oldest)' },
                    { value: 'sortNewOld', label: 'Updated (Newest)' },
                    { value: 'sortOldNew', label: 'Updated (Oldest)' },
                    { value: 'sortBigSmall', label: 'A to Z' },
                    { value: 'sortSmallBig', label: 'Z to A' },
                  ].map((option) => (
                    <button
                      key={option.value}
                      onClick={() =>
                        handleConfigChange({
                          ...searchConfig,
                          sortBy: option.value,
                        })
                      }
                      className={`w-full text-left px-4 py-3 rounded-lg transition-colors ${
                        searchConfig.sortBy === option.value
                          ? 'bg-blue-100 text-blue-900 font-medium'
                          : 'hover:bg-gray-100 bg-gray-50 text-gray-700'
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span>{option.label}</span>
                        {searchConfig.sortBy === option.value && (
                          <svg
                            className="w-5 h-5 text-blue-600"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                        )}
                      </div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Schema Filter */}
              {schemas.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-sm font-semibold text-gray-900 mb-3">
                    Filter by Schema
                  </h3>
                  <select
                    value={searchConfig.schemaId ?? ''}
                    onChange={(e) => {
                      const value = e.target.value;
                      const schemaId = value === '' ? null : parseInt(value);
                      handleConfigChange({ ...searchConfig, schemaId });
                    }}
                    className="w-full text-sm border border-gray-300 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                  >
                    <option value="">All Schemas</option>
                    {schemas.map((schema) => (
                      <option key={schema.id} value={schema.id}>
                        {schema.name}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Search Settings */}
              <div className="mb-6">
                <h3 className="text-sm font-semibold text-gray-900 mb-3">
                  Search Settings
                </h3>
                <div className="space-y-2">
                  <label className="flex items-center p-3 rounded-lg hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={searchConfig.useFullText}
                      onChange={(e) =>
                        handleConfigChange({
                          ...searchConfig,
                          useFullText: e.target.checked,
                        })
                      }
                      className="mr-3 w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">
                      Search Full Text
                    </span>
                  </label>
                  <label className="flex items-center p-3 rounded-lg hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={searchConfig.onlyParentCards}
                      onChange={(e) =>
                        handleConfigChange({
                          ...searchConfig,
                          onlyParentCards: e.target.checked,
                        })
                      }
                      className="mr-3 w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">
                      Only Parent Cards
                    </span>
                  </label>
                  <label className="flex items-center p-3 rounded-lg hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={searchConfig.onlyEmptyCardId}
                      onChange={(e) =>
                        handleConfigChange({
                          ...searchConfig,
                          onlyEmptyCardId: e.target.checked,
                        })
                      }
                      className="mr-3 w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">
                      Only Unsorted Cards
                    </span>
                  </label>
                  <label className="flex items-center p-3 rounded-lg hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={searchConfig.showPreview}
                      onChange={(e) =>
                        setSearchConfig({
                          ...searchConfig,
                          showPreview: e.target.checked,
                        })
                      }
                      className="mr-3 w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">Show Preview</span>
                  </label>
                  <label className="flex items-center p-3 rounded-lg hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={searchConfig.showEntities}
                      onChange={(e) =>
                        handleConfigChange({
                          ...searchConfig,
                          showEntities: e.target.checked,
                        })
                      }
                      className="mr-3 w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">Show Entities</span>
                  </label>
                  <label className="flex items-center p-3 rounded-lg hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={searchConfig.showFacts}
                      onChange={(e) =>
                        handleConfigChange({
                          ...searchConfig,
                          showFacts: e.target.checked,
                        })
                      }
                      className="mr-3 w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">Show Facts</span>
                  </label>
                  <label className="flex items-center p-3 rounded-lg hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={searchConfig.showCards}
                      onChange={(e) =>
                        handleConfigChange({
                          ...searchConfig,
                          showCards: e.target.checked,
                        })
                      }
                      className="mr-3 w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500"
                    />
                    <span className="text-sm text-gray-700">Show Cards</span>
                  </label>
                </div>
              </div>

              {/* Tags Section */}
              {tags.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-sm font-semibold text-gray-900 mb-3">
                    Search by Tag
                  </h3>
                  <div className="flex flex-wrap gap-2">
                    {tags.slice(0, 12).map((tag) => (
                      <button
                        key={tag.id}
                        onClick={() => {
                          handleTagClick(tag.name);
                          setMobileView('list');
                        }}
                        className="px-3 py-1.5 bg-purple-50 text-purple-600 rounded-full text-sm hover:bg-purple-100 transition-colors"
                      >
                        #{tag.name}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Star Search Option */}
              {!starredId && setShowStarSearchDialog && (
                <div className="pt-4 border-t border-gray-200">
                  <button
                    onClick={() => {
                      setShowStarSearchDialog(true);
                      setMobileView('list');
                    }}
                    className="w-full px-4 py-3 bg-yellow-50 text-yellow-700 rounded-lg hover:bg-yellow-100 transition-colors font-medium flex items-center justify-center gap-2"
                  >
                    <svg
                      className="w-5 h-5"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                    </svg>
                    Star This Search
                  </button>
                </div>
              )}
            </div>

            <style>{`
              @keyframes fade-in {
                from {
                  opacity: 0;
                }
                to {
                  opacity: 1;
                }
              }
              @keyframes slide-up {
                from {
                  transform: translateY(100%);
                }
                to {
                  transform: translateY(0);
                }
              }
            `}</style>
          </div>
        </div>
      </>
    );
  }

  return null;
}
