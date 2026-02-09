import React, { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { SearchConfig as SearchConfigType, StarredSearch } from "../../models/StarredSearch";
import { Tag } from "../../models/Tags";
import { SchemaDefinition } from "../../models/Schema";
import { getStarredSearches, unstarSearch } from "../../api/starredSearches";
import { fetchSchemas } from "../../api/schemas";
import { useToast } from "../toast/ToastContext";

interface SearchFiltersPanelProps {
  searchTerm: string;
  searchConfig: SearchConfigType;
  setSearchConfig: (config: SearchConfigType) => void;
  tags: Tag[];
  starredId?: string | null;
  setShowStarSearchDialog: (show: boolean) => void;
  onTagClick: (tagName: string) => void;
  onSearchTrigger?: (config: SearchConfigType, resetPage?: boolean) => void;
  onSearchTermChange: (term: string) => void;
  onSearch: (searchTerm: string, config: SearchConfigType) => void;
  isLoading: boolean;
}

/**
 * Left sidebar panel showing search filters and options
 * This is the LEFT PANEL for the 3-column desktop layout.
 */
export function SearchFiltersPanel({
  searchTerm,
  searchConfig,
  setSearchConfig,
  tags,
  starredId,
  setShowStarSearchDialog,
  onTagClick,
  onSearchTrigger,
  onSearchTermChange,
  onSearch,
  isLoading,
}: SearchFiltersPanelProps) {
  const [starredSearches, setStarredSearches] = useState<StarredSearch[]>([]);
  const [schemas, setSchemas] = useState<SchemaDefinition[]>([]);
  const [schemasLoading, setSchemasLoading] = useState(true);
  const { showToast } = useToast();

  // Load starred searches
  const refreshStarredSearches = () => {
    getStarredSearches()
      .then((searches) => {
        setStarredSearches(searches);
      })
      .catch(error => {
        console.error("Error fetching starred searches:", error);
      });
  };

  useEffect(() => {
    refreshStarredSearches();
  }, []);

  // Load schemas
  useEffect(() => {
    const loadSchemas = async () => {
      try {
        const data = await fetchSchemas();
        setSchemas(data);
      } catch (error) {
        console.error("Failed to load schemas:", error);
      } finally {
        setSchemasLoading(false);
      }
    };
    loadSchemas();
  }, []);

  const handleUnstarSearch = (searchId: number) => {
    unstarSearch(searchId)
      .then(() => {
        refreshStarredSearches();
        showToast("success", "Search unstarred successfully");
      })
      .catch(error => {
        console.error("Error unstarring search:", error);
        showToast("error", "Failed to unstar search", "Please try again");
      });
  };
  const handleSchemaChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value;
    const schemaId = value === "" ? null : parseInt(value);
    const newConfig = { ...searchConfig, schemaId };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" && !isLoading) {
      onSearch(searchTerm, searchConfig);
    }
  };

  const handleFullTextChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newConfig = { ...searchConfig, useFullText: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleOnlyParentCardsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newConfig = { ...searchConfig, onlyParentCards: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleOnlyEmptyCardIdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newConfig = { ...searchConfig, onlyEmptyCardId: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleShowPreviewChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchConfig({ ...searchConfig, showPreview: event.target.checked });
  };

  const handleShowEntitiesChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newConfig = { ...searchConfig, showEntities: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleShowFactsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newConfig = { ...searchConfig, showFacts: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleShowCardsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newConfig = { ...searchConfig, showCards: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleSortByChange = (sortBy: string) => {
    const newConfig = { ...searchConfig, sortBy };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  return (
    <div className="hidden md:flex w-72 border-r border-gray-200 p-4 overflow-y-auto bg-gray-50 flex-shrink-0 flex-col">
      <div className="mb-4">
        <h2 className="text-lg font-semibold mb-3">Search</h2>
        <div className="space-y-2">
          <input
            type="text"
            value={searchTerm}
            placeholder="Search cards, entities, facts..."
            onChange={(e) => onSearchTermChange(e.target.value)}
            onKeyPress={handleKeyPress}
            disabled={isLoading}
            className="w-full h-9 px-3 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-50"
          />
          <button
            onClick={() => onSearch(searchTerm, searchConfig)}
            disabled={isLoading}
            className="w-full h-9 bg-blue-600 hover:bg-blue-700 text-white rounded-md px-4 text-sm flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            Search
          </button>
        </div>
      </div>

      {/* Star Search Option */}
      {!starredId && (
        <div className="mb-4">
          <button
            onClick={() => setShowStarSearchDialog(true)}
            className="w-full px-3 py-2 text-sm text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-md transition-colors flex items-center gap-2"
          >
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
            </svg>
            Star This Search
          </button>
        </div>
      )}

      {/* Starred Searches Section */}
      <div className="mb-4">
        <div className="text-xs text-gray-500 mb-2 font-semibold uppercase tracking-wider">Starred Searches</div>
        {starredSearches.length > 0 ? (
          <ul className="space-y-0.5">
            {starredSearches.map((search) => (
              <li key={search.id} className="px-2 py-0.5 text-sm group">
                <div className="flex items-center">
                  <Link
                    to={`/app/search?term=${encodeURIComponent(search.searchTerm)}&starred=${search.id}`}
                    className="flex-grow hover:bg-gray-100 rounded p-1 truncate"
                    title={search.title}
                  >
                    <span className="mr-1">•</span>
                    {search.title}
                  </Link>
                  <button
                    onClick={() => handleUnstarSearch(search.id)}
                    className="text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center px-1"
                    aria-label={`Unstar "${search.title}"`}
                    title="Unstar search"
                  >
                    ×
                  </button>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-xs text-gray-400 px-2">No starred searches yet</p>
        )}
      </div>

      {/* Sorting Section */}
      <div className="mb-4">
        <div className="text-xs text-gray-500 mb-2 font-semibold">Sort Results</div>
        <div className="space-y-1">
          {[
            { value: 'sortByRanking', label: 'Ranking Score' },
            { value: 'sortCreatedNewOld', label: 'Created (Newest)' },
            { value: 'sortCreatedOldNew', label: 'Created (Oldest)' },
            { value: 'sortNewOld', label: 'Updated (Newest)' },
            { value: 'sortOldNew', label: 'Updated (Oldest)' },
            { value: 'sortBigSmall', label: 'A to Z' },
            { value: 'sortSmallBig', label: 'Z to A' }
          ].map((option) => (
            <button
              key={option.value}
              onClick={() => handleSortByChange(option.value)}
              className={`w-full text-left px-2 py-1.5 text-xs rounded-md transition-colors ${
                searchConfig.sortBy === option.value
                  ? 'bg-blue-50 text-blue-700 border border-blue-200'
                  : 'text-gray-700 hover:bg-gray-100 border border-transparent'
              }`}
            >
              <div className="flex items-center justify-between">
                <span>{option.label}</span>
                {searchConfig.sortBy === option.value && (
                  <span className="text-blue-600">✓</span>
                )}
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Search Settings */}
      <div className="mb-4">
        <div className="text-xs text-gray-500 mb-2 font-semibold">Search Settings</div>
        <div className="space-y-2">
          {/* Schema Filter */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Filter by Schema
            </label>
            {schemasLoading ? (
              <div className="text-sm text-gray-500">Loading schemas...</div>
            ) : (
              <select
                value={searchConfig.schemaId ?? ""}
                onChange={handleSchemaChange}
                className="w-full text-sm border border-gray-300 rounded-md px-2 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">All Schemas</option>
                {schemas.map((schema) => (
                  <option key={schema.id} value={schema.id}>
                    {schema.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          <div className="hover:bg-gray-50 rounded px-2 py-1">
            <label className="flex items-center text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={searchConfig.useFullText}
                onChange={handleFullTextChange}
                className="mr-2"
              />
              Search Full Text
            </label>
          </div>
          <div className="hover:bg-gray-50 rounded px-2 py-1">
            <label className="flex items-center text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={searchConfig.onlyParentCards}
                onChange={handleOnlyParentCardsChange}
                className="mr-2"
              />
              Only Parent Cards
            </label>
          </div>
          <div className="hover:bg-gray-50 rounded px-2 py-1">
            <label className="flex items-center text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={searchConfig.onlyEmptyCardId}
                onChange={handleOnlyEmptyCardIdChange}
                className="mr-2"
              />
              Only Unsorted Cards
            </label>
          </div>
          <div className="hover:bg-gray-50 rounded px-2 py-1">
            <label className="flex items-center text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={searchConfig.showPreview}
                onChange={handleShowPreviewChange}
                className="mr-2"
              />
              Show Preview
            </label>
          </div>
          <div className="hover:bg-gray-50 rounded px-2 py-1">
            <label className="flex items-center text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={searchConfig.showEntities}
                onChange={handleShowEntitiesChange}
                className="mr-2"
              />
              Show Entities
            </label>
          </div>
          <div className="hover:bg-gray-50 rounded px-2 py-1">
            <label className="flex items-center text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={searchConfig.showFacts}
                onChange={handleShowFactsChange}
                className="mr-2"
              />
              Show Facts
            </label>
          </div>
          <div className="hover:bg-gray-50 rounded px-2 py-1">
            <label className="flex items-center text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={searchConfig.showCards}
                onChange={handleShowCardsChange}
                className="mr-2"
              />
              Show Cards
            </label>
          </div>
        </div>
      </div>

      {/* Tags Section */}
      {tags && tags.length > 0 && (
        <div className="mb-4">
          <div className="text-xs text-gray-500 mb-2 font-semibold">Search by Tag</div>
          <div className="max-h-48 overflow-y-auto space-y-1">
            {tags.slice(0, 10).map((tag) => (
              <button
                key={tag.id}
                onClick={() => onTagClick(tag.name)}
                className="w-full text-left px-2 py-1 text-xs text-gray-700 hover:bg-gray-100 rounded-md transition-colors flex items-center gap-1"
              >
                <span className="text-blue-500">#</span>
                {tag.name}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
