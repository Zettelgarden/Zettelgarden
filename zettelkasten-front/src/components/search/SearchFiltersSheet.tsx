import React, { useState, useEffect } from 'react';
import { MobileBottomSheet } from '../layout/MobileBottomSheet';
import { SearchConfig as SearchConfigType } from '../../models/StarredSearch';
import { Tag } from '../../models/Tags';
import { fetchSchemas } from '../../api/schemas';
import { SchemaDefinition } from '../../models/Schema';
import { getStarredSearches, unstarSearch } from '../../api/starredSearches';
import { StarredSearch } from '../../models/StarredSearch';
import { useToast } from '../toast/ToastContext';

interface SearchFiltersSheetProps {
  isOpen: boolean;
  onClose: () => void;
  searchTerm: string;
  searchConfig: SearchConfigType;
  setSearchConfig: (config: SearchConfigType) => void;
  tags: Tag[];
  onTagClick: (tagName: string) => void;
  onSearchTrigger?: (config: SearchConfigType, resetPage?: boolean) => void;
  onStarSearch?: () => void;
  starredId?: string | null;
  currentUserId?: number;
}

export function SearchFiltersSheet({
  isOpen,
  onClose,
  searchTerm,
  searchConfig,
  setSearchConfig,
  tags,
  onTagClick,
  onSearchTrigger,
  onStarSearch,
  starredId,
  currentUserId,
}: SearchFiltersSheetProps) {
  const [schemas, setSchemas] = useState<SchemaDefinition[]>([]);
  const [schemasLoading, setSchemasLoading] = useState(true);
  const [starredSearches, setStarredSearches] = useState<StarredSearch[]>([]);
  const { showToast } = useToast();

  useEffect(() => {
    const loadSchemas = async () => {
      try {
        const data = await fetchSchemas();
        setSchemas(data);
      } catch (error) {
        console.error('Failed to load schemas:', error);
      } finally {
        setSchemasLoading(false);
      }
    };
    if (isOpen) {
      loadSchemas();
      loadStarredSearches();
    }
  }, [isOpen]);

  const loadStarredSearches = async () => {
    try {
      const searches = await getStarredSearches();
      setStarredSearches(searches);
    } catch (error) {
      console.error('Failed to load starred searches:', error);
    }
  };

  const handleSchemaChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value;
    const schemaId = value === '' ? null : parseInt(value);
    const newConfig = { ...searchConfig, schemaId };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleFullTextChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newConfig = { ...searchConfig, useFullText: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleOnlyParentCardsChange = (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const newConfig = {
      ...searchConfig,
      onlyParentCards: event.target.checked,
    };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleOnlyEmptyCardIdChange = (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const newConfig = {
      ...searchConfig,
      onlyEmptyCardId: event.target.checked,
    };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleShowPreviewChange = (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    setSearchConfig({ ...searchConfig, showPreview: event.target.checked });
  };

  const handleShowEntitiesChange = (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const newConfig = { ...searchConfig, showEntities: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleShowFactsChange = (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const newConfig = { ...searchConfig, showFacts: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleShowCardsChange = (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const newConfig = { ...searchConfig, showCards: event.target.checked };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleSortChange = (sortBy: string) => {
    const newConfig = { ...searchConfig, sortBy };
    setSearchConfig(newConfig);
    onSearchTrigger?.(newConfig, true);
  };

  const handleUnstarSearch = (searchId: number) => {
    unstarSearch(searchId)
      .then(() => {
        loadStarredSearches();
        showToast('success', 'Search unstarred successfully');
      })
      .catch((error) => {
        console.error('Error unstarring search:', error);
        showToast('error', 'Failed to unstar search', 'Please try again');
      });
  };

  const sortOptions = [
    { value: 'sortByRanking', label: 'Ranking Score' },
    { value: 'sortCreatedNewOld', label: 'Created (Newest)' },
    { value: 'sortCreatedOldNew', label: 'Created (Oldest)' },
    { value: 'sortNewOld', label: 'Updated (Newest)' },
    { value: 'sortOldNew', label: 'Updated (Oldest)' },
    { value: 'sortBigSmall', label: 'A to Z' },
    { value: 'sortSmallBig', label: 'Z to A' },
  ];

  return (
    <MobileBottomSheet
      isOpen={isOpen}
      onClose={onClose}
      title="Filters"
      showCloseButton={true}
      maxHeight="85vh"
    >
      <div className="px-4 py-4 space-y-6">
        {/* Starred Searches Section */}
        {starredSearches.length > 0 && (
          <section>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">
              Starred Searches
            </h3>
            <div className="space-y-2 max-h-40 overflow-y-auto">
              {starredSearches.map((search) => (
                <div
                  key={search.id}
                  className="flex items-center justify-between p-3 bg-gray-50 rounded-lg touch-manipulation"
                >
                  <button
                    onClick={() => {
                      setSearchConfig({
                        ...searchConfig,
                        ...search.searchConfig,
                      });
                      onSearchTrigger?.(
                        {
                          ...searchConfig,
                          ...search.searchConfig,
                        },
                        true,
                      );
                      onClose();
                    }}
                    className="flex-grow text-left touch-manipulation min-h-[44px] flex items-center"
                  >
                    <span className="text-sm text-gray-800">
                      {search.title}
                    </span>
                  </button>
                  <button
                    onClick={() => handleUnstarSearch(search.id)}
                    className="ml-2 text-gray-400 hover:text-red-500 p-2 touch-manipulation min-h-[44px] min-w-[44px] flex items-center justify-center"
                    aria-label={`Unstar "${search.title}"`}
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
              ))}
            </div>
          </section>
        )}

        {/* Star Current Search Section */}
        {!starredId && onStarSearch && (
          <section>
            <button
              onClick={onStarSearch}
              className="w-full p-3 bg-blue-50 text-blue-700 rounded-lg font-medium text-sm touch-manipulation min-h-[44px] flex items-center justify-center"
            >
              <svg
                className="w-5 h-5 mr-2"
                fill="currentColor"
                viewBox="0 0 20 20"
              >
                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
              </svg>
              Star Current Search
            </button>
          </section>
        )}

        {/* Sort By Section */}
        <section>
          <h3 className="text-sm font-semibold text-gray-700 mb-3">
            Sort Results
          </h3>
          <div className="grid grid-cols-2 gap-2">
            {sortOptions.map((option) => (
              <button
                key={option.value}
                onClick={() => handleSortChange(option.value)}
                className={`p-3 text-sm rounded-lg touch-manipulation min-h-[44px] text-left transition-colors ${
                  searchConfig.sortBy === option.value
                    ? 'bg-blue-600 text-white border-2 border-blue-600'
                    : 'bg-gray-100 text-gray-700 border-2 border-transparent hover:bg-gray-200'
                }`}
              >
                {option.label}
              </button>
            ))}
          </div>
        </section>

        {/* Tags Section */}
        {tags && tags.length > 0 && (
          <section>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">
              Search by Tag
            </h3>
            <div className="flex flex-wrap gap-2 max-h-32 overflow-y-auto">
              {tags.slice(0, 12).map((tag) => (
                <button
                  key={tag.id}
                  onClick={() => {
                    onTagClick(tag.name);
                    onClose();
                  }}
                  className="px-3 py-2 bg-gray-100 text-gray-700 rounded-full text-sm touch-manipulation min-h-[44px] hover:bg-gray-200 transition-colors"
                >
                  #{tag.name}
                </button>
              ))}
            </div>
          </section>
        )}

        {/* Search Settings Section */}
        <section>
          <h3 className="text-sm font-semibold text-gray-700 mb-3">
            Search Settings
          </h3>
          <div className="space-y-3">
            {/* Schema Filter */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Filter by Schema
              </label>
              {schemasLoading ? (
                <div className="text-sm text-gray-500 p-3">
                  Loading schemas...
                </div>
              ) : (
                <select
                  value={searchConfig.schemaId ?? ''}
                  onChange={handleSchemaChange}
                  className="w-full p-3 text-sm border border-gray-300 rounded-lg bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[44px] touch-manipulation"
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

            {/* Checkbox Options */}
            <div className="space-y-1">
              <label className="flex items-center p-3 hover:bg-gray-50 rounded-lg cursor-pointer touch-manipulation">
                <input
                  type="checkbox"
                  checked={searchConfig.useFullText}
                  onChange={handleFullTextChange}
                  className="w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-blue-500 touch-manipulation"
                />
                <span className="ml-3 text-sm text-gray-700">
                  Search Full Text
                </span>
              </label>

              <label className="flex items-center p-3 hover:bg-gray-50 rounded-lg cursor-pointer touch-manipulation">
                <input
                  type="checkbox"
                  checked={searchConfig.onlyParentCards}
                  onChange={handleOnlyParentCardsChange}
                  className="w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-blue-500 touch-manipulation"
                />
                <span className="ml-3 text-sm text-gray-700">
                  Only Parent Cards
                </span>
              </label>

              <label className="flex items-center p-3 hover:bg-gray-50 rounded-lg cursor-pointer touch-manipulation">
                <input
                  type="checkbox"
                  checked={searchConfig.onlyEmptyCardId}
                  onChange={handleOnlyEmptyCardIdChange}
                  className="w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-blue-500 touch-manipulation"
                />
                <span className="ml-3 text-sm text-gray-700">
                  Only Unsorted Cards
                </span>
              </label>

              <label className="flex items-center p-3 hover:bg-gray-50 rounded-lg cursor-pointer touch-manipulation">
                <input
                  type="checkbox"
                  checked={searchConfig.showPreview}
                  onChange={handleShowPreviewChange}
                  className="w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-blue-500 touch-manipulation"
                />
                <span className="ml-3 text-sm text-gray-700">Show Preview</span>
              </label>

              <label className="flex items-center p-3 hover:bg-gray-50 rounded-lg cursor-pointer touch-manipulation">
                <input
                  type="checkbox"
                  checked={searchConfig.showEntities}
                  onChange={handleShowEntitiesChange}
                  className="w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-blue-500 touch-manipulation"
                />
                <span className="ml-3 text-sm text-gray-700">
                  Show Entities
                </span>
              </label>

              <label className="flex items-center p-3 hover:bg-gray-50 rounded-lg cursor-pointer touch-manipulation">
                <input
                  type="checkbox"
                  checked={searchConfig.showFacts}
                  onChange={handleShowFactsChange}
                  className="w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-blue-500 touch-manipulation"
                />
                <span className="ml-3 text-sm text-gray-700">Show Facts</span>
              </label>

              <label className="flex items-center p-3 hover:bg-gray-50 rounded-lg cursor-pointer touch-manipulation">
                <input
                  type="checkbox"
                  checked={searchConfig.showCards}
                  onChange={handleShowCardsChange}
                  className="w-5 h-5 text-blue-600 border-gray-300 rounded focus:ring-blue-500 touch-manipulation"
                />
                <span className="ml-3 text-sm text-gray-700">Show Cards</span>
              </label>
            </div>
          </div>
        </section>

        {/* Action Buttons */}
        <div className="flex gap-3 pt-2 pb-safe">
          <button
            onClick={onClose}
            className="flex-1 py-4 bg-gray-200 text-gray-800 rounded-lg font-medium text-sm touch-manipulation min-h-[48px] active:bg-gray-300 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </MobileBottomSheet>
  );
}
