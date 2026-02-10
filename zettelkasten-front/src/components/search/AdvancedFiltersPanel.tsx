import React, { useEffect, useState } from "react";
import { SearchConfig } from "../../models/StarredSearch";
import { SchemaDefinition } from "../../models/Schema";
import { fetchSchemas } from "../../api/schemas";

interface AdvancedFiltersPanelProps {
  searchConfig: SearchConfig;
  setSearchConfig: (config: SearchConfig) => void;
  onApply: () => void;
  isOpen: boolean;
}

/**
 * Collapsible panel with advanced search configuration options
 * Consolidates all search filters into one place with progressive disclosure
 */
export function AdvancedFiltersPanel({
  searchConfig,
  setSearchConfig,
  onApply,
  isOpen,
}: AdvancedFiltersPanelProps) {
  const [schemas, setSchemas] = useState<SchemaDefinition[]>([]);
  const [schemasLoading, setSchemasLoading] = useState(true);

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

  const handleSchemaChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value;
    const schemaId = value === "" ? null : parseInt(value);
    const newConfig = { ...searchConfig, schemaId };
    setSearchConfig(newConfig);
    onApply();
  };

  const handleFullTextChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchConfig({ ...searchConfig, useFullText: event.target.checked });
    onApply();
  };

  const handleOnlyParentCardsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchConfig({ ...searchConfig, onlyParentCards: event.target.checked });
    onApply();
  };

  const handleOnlyEmptyCardIdChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchConfig({ ...searchConfig, onlyEmptyCardId: event.target.checked });
    onApply();
  };

  const handleShowPreviewChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchConfig({ ...searchConfig, showPreview: event.target.checked });
  };

  const handleShowEntitiesChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchConfig({ ...searchConfig, showEntities: event.target.checked });
    onApply();
  };

  const handleShowFactsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchConfig({ ...searchConfig, showFacts: event.target.checked });
    onApply();
  };

  const handleShowCardsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchConfig({ ...searchConfig, showCards: event.target.checked });
    onApply();
  };

  const handleSortByChange = (sortBy: string) => {
    setSearchConfig({ ...searchConfig, sortBy });
    onApply();
  };

  if (!isOpen) {
    return null;
  }

  return (
    <div className="flex-shrink-0 border-b border-gray-200 bg-gray-50">
      <div className="p-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Sorting Section */}
          <div>
            <div className="text-xs text-gray-500 mb-2 font-semibold uppercase tracking-wider">Sort Results</div>
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
          <div>
            <div className="text-xs text-gray-500 mb-2 font-semibold uppercase tracking-wider">Search Settings</div>
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

              <div className="hover:bg-gray-100 rounded px-2 py-1">
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
              <div className="hover:bg-gray-100 rounded px-2 py-1">
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
              <div className="hover:bg-gray-100 rounded px-2 py-1">
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
              <div className="hover:bg-gray-100 rounded px-2 py-1">
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
              <div className="hover:bg-gray-100 rounded px-2 py-1">
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
              <div className="hover:bg-gray-100 rounded px-2 py-1">
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
              <div className="hover:bg-gray-100 rounded px-2 py-1">
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
      </div>
    </div>
  );
}
