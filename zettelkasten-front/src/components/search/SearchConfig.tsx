import React, { useState, useEffect } from 'react';
import { Menu } from '@headlessui/react';
import { SearchConfig as SearchConfigType } from '../../models/StarredSearch';
import { Tag } from '../../models/Tags';
import { fetchSchemas } from '../../api/schemas';
import { SchemaDefinition } from '../../models/Schema';

interface SearchConfigProps {
  searchTerm: string;
  searchConfig: SearchConfigType;
  setSearchConfig: (config: SearchConfigType) => void;
  tags: Tag[];
  starredId?: string | null;
  setShowStarSearchDialog: (show: boolean) => void;
  onTagClick: (tagName: string) => void;
  onSearchTrigger?: (config: SearchConfigType, resetPage?: boolean) => void;
}

export function SearchConfig({
  searchTerm,
  searchConfig,
  setSearchConfig,
  tags,
  starredId,
  setShowStarSearchDialog,
  onTagClick,
  onSearchTrigger,
}: SearchConfigProps) {
  const [schemas, setSchemas] = useState<SchemaDefinition[]>([]);
  const [schemasLoading, setSchemasLoading] = useState(true);

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
    loadSchemas();
  }, []);

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

  return (
    <Menu as="div" className="relative">
      <Menu.Button className="h-9 inline-flex items-center justify-center px-3 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-blue-500">
        Options
      </Menu.Button>
      <Menu.Items className="absolute right-0 mt-2 w-96 bg-white border border-gray-200 rounded-md shadow-lg py-3 z-10">
        <div className="flex">
          {/* Left Column */}
          <div className="w-1/2 px-4 border-r border-gray-200">
            {/* Star Search Option */}
            {!starredId && (
              <>
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => setShowStarSearchDialog(true)}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex rounded-md items-center w-full px-2 py-2 text-sm mb-2`}
                    >
                      Star This Search
                    </button>
                  )}
                </Menu.Item>
                <div className="border-t border-gray-100 my-2"></div>
              </>
            )}

            {/* Sorting Section */}
            <div className="mb-4">
              <div className="text-xs text-gray-500 mb-2 font-semibold">
                Sort Results
              </div>
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
                  <Menu.Item key={option.value}>
                    {({ active }) => (
                      <button
                        onClick={() => {
                          const newConfig = {
                            ...searchConfig,
                            sortBy: option.value,
                          };
                          setSearchConfig(newConfig);
                          onSearchTrigger?.(newConfig, true);
                        }}
                        className={`${
                          searchConfig.sortBy === option.value
                            ? 'bg-blue-50 text-blue-700 border-blue-200'
                            : active
                            ? 'bg-gray-100 text-gray-900'
                            : 'text-gray-700'
                        } group flex rounded-md items-center w-full px-2 py-1.5 text-xs border ${
                          searchConfig.sortBy === option.value
                            ? 'border-blue-200'
                            : 'border-transparent'
                        }`}
                      >
                        {option.label}
                        {searchConfig.sortBy === option.value && (
                          <span className="ml-auto text-blue-600">✓</span>
                        )}
                      </button>
                    )}
                  </Menu.Item>
                ))}
              </div>
            </div>

            {/* Tags Section */}
            <div>
              <div className="text-xs text-gray-500 mb-2 font-semibold">
                Search by Tag
              </div>
              <div className="max-h-32 overflow-y-auto space-y-1">
                {tags &&
                  tags.slice(0, 8).map((tag) => (
                    <Menu.Item key={tag.id}>
                      {({ active }) => (
                        <button
                          onClick={() => onTagClick(tag.name)}
                          className={`${
                            active
                              ? 'bg-gray-100 text-gray-900'
                              : 'text-gray-700'
                          } group flex rounded-md items-center w-full px-2 py-1 text-xs`}
                        >
                          #{tag.name}
                        </button>
                      )}
                    </Menu.Item>
                  ))}
              </div>
            </div>
          </div>

          {/* Right Column */}
          <div className="w-1/2 px-4">
            <div className="text-xs text-gray-500 mb-2 font-semibold">
              Search Settings
            </div>
            <div className="space-y-2">
              {/* Schema Filter */}
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Filter by Schema
                </label>
                {schemasLoading ? (
                  <div className="text-sm text-gray-500">
                    Loading schemas...
                  </div>
                ) : (
                  <select
                    value={searchConfig.schemaId ?? ''}
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
        </div>
      </Menu.Items>
    </Menu>
  );
}
