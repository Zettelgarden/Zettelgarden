import React, { useState, useEffect, ChangeEvent, KeyboardEvent } from "react";
import { Menu } from '@headlessui/react';
import { semanticSearchCards, semanticSearchCardsPaginated } from "../../api/cards";
import { fetchUserTags } from "../../api/tags";
import { SearchResult } from "../../models/Card";
import { Tag } from "../../models/Tags";
import { SearchConfig } from "../../models/StarredSearch";
import { sortCards } from "../../utils/cards";
import { Button } from "../../components/Button";
import { SearchResultList } from "../../components/cards/SearchResultList";
import { StarSearchDialog } from "../../components/search/StarSearchDialog";
import { getStarredSearches } from "../../api/starredSearches";
import { useTagContext } from "../../contexts/TagContext";
import { Entity } from "../../models/Card";
import { fetchEntityByName } from "../../api/entities";
import { setDocumentTitle } from "../../utils/title";

import { useShortcutContext } from "../../contexts/ShortcutContext";

interface SearchPageProps {
  searchTerm: string;
  setSearchTerm: (searchTerm: string) => void;
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;
  searchConfig: SearchConfig
  setSearchConfig: (config: any) => void;
}

export function SearchPage({
  searchTerm,
  setSearchTerm,
  searchResults,
  setSearchResults,
  searchConfig,
  setSearchConfig,
}: SearchPageProps) {
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<Error | null>(null);
  const { tags } = useTagContext();
  const [showStarSearchDialog, setShowStarSearchDialog] = useState<boolean>(false);
  const [message, setMessage] = useState<string>("");
  const latestRequestId = React.useRef(0);

  // Pagination state
  const [totalResults, setTotalResults] = useState<number>(0);
  const [totalPages, setTotalPages] = useState<number>(0);
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [perPage, setPerPage] = useState<number>(20);
  const {
    showEntityDialog,
    setShowEntityDialog,
    selectedEntity,
    setSelectedEntity,
  } = useShortcutContext();

  const params = new URLSearchParams(location.search);
  const starredId = params.get("starred");

  function handleSearchUpdate(e: ChangeEvent<HTMLInputElement>) {
    setSearchTerm(e.target.value);
  }

  async function handleSearch(searchTerm: string, config: SearchConfig, page: number = 1) {
    const requestId = ++latestRequestId.current;

    setIsLoading(true);
    setError(null);
    setSearchConfig({ ...config, currentPage: page });

    const term = searchTerm || "";
    console.log("searching for term:", term, "page:", page);

    try {
      const response = await semanticSearchCardsPaginated(
        term,
        config.useFullText,
        config.showEntities,
        config.showFacts,
        config.showCards,
        config.sortBy,
        config.searchType,
        config.rerank,
        page,
        perPage,
      );
      if (requestId === latestRequestId.current) {
        setSearchResults(response.results || []);
        setTotalResults(response.total);
        setTotalPages(response.total_pages);
        setCurrentPage(response.page);
      }
    } catch (error) {
      console.error("Search error:", error);
      if (requestId === latestRequestId.current) {
        setError(error);
      }
    } finally {
      if (requestId === latestRequestId.current) {
        setIsLoading(false);
      }
    }
  }

  useEffect(() => {
    const initializeSearch = async () => {
      setDocumentTitle("Search")
      const params = new URLSearchParams(location.search);
      const recent = params.get("recent");
      const term = params.get("term") || "";
      const starredId = params.get("starred");

      // Check if we're loading a starred search
      if (starredId) {
        try {
          const starredSearches = await getStarredSearches();
          const starredSearch = starredSearches.find(search => search.id === parseInt(starredId));

          console.log("search", starredSearch)
          if (starredSearch) {
            // Apply the starred search configuration
            setSearchTerm(starredSearch.searchTerm);
            setSearchConfig({
              ...searchConfig,
              ...starredSearch.searchConfig
            });

            // Execute the search with the starred configuration
            await handleSearch(starredSearch.searchTerm, starredSearch.searchConfig);
            return; // Exit early since we've handled the search
          }
        } catch (error) {
          console.error("Error loading starred search:", error);
          setMessage("Error loading starred search");
        }
      }

      // Regular search initialization if not a starred search
      if (recent !== null) {
        let config = { ...searchConfig, useClassicSearch: true }
        setSearchConfig(config);
        setSearchTerm("");
        await handleSearch("", config);
      } else if (term) {
        let config = { ...searchConfig, useClassicSearch: true }
        setSearchConfig(config);
        setSearchTerm(term);
        await handleSearch(term, config);
      } else {
        let config = { ...searchConfig, sortBy: "sortByRanking" }
        setSearchConfig(config);
        await handleSearch("", config);
      }
    };

    initializeSearch();
  }, [location.search]); // Re-run when the URL search parameters change

  function handleSortChange(e: ChangeEvent<HTMLSelectElement>) {
    setSearchConfig({ ...searchConfig, sortBy: e.target.value });
  }

  function handleTagClick(tagName: string) {
    setSearchTerm("#" + tagName);
    handleSearch(tagName, searchConfig);
  }

  async function handleEntityClick(entityName: string) {
    // Extract entity name from @[EntityName] format
    const cleanEntityName = entityName.replace('@[', '').replace(']', '');

    try {
      // Fetch the real entity data from the backend
      const entity = await fetchEntityByName(cleanEntityName);
      setSelectedEntity(entity);
      setShowEntityDialog(true);
    } catch (error) {
      console.error('Failed to fetch entity details:', error);
      // Fallback: still open dialog but with minimal entity data
      const fallbackEntity: Entity = {
        id: 0,
        user_id: 0,
        name: cleanEntityName,
        type: 'PERSON',
        description: '',
        created_at: new Date(),
        updated_at: new Date(),
        card_count: 0,
        card_pk: null,
      };
      setSelectedEntity(fallbackEntity);
      setShowEntityDialog(true);
    }
  }

  function getFilteredResults(): SearchResult[] {
    return searchResults
      .filter(result => !searchConfig.onlyParentCards || !result.id.includes("/"));
  }

  const handleOnlyParentCardsChange = (event) => {
    setSearchConfig({ ...searchConfig, onlyParentCards: event.target.checked, currentPage: 1 });
    setCurrentPage(1);
    handleSearch(searchTerm, { ...searchConfig, onlyParentCards: event.target.checked }, 1);
  };

  const handleShowPreviewChange = (event) => {
    setSearchConfig({ ...searchConfig, showPreview: event.target.checked });
  };

  const handleFullTextChange = (event) => {
    let config = { ...searchConfig, useFullText: event.target.checked }
    setSearchConfig(config);
    setCurrentPage(1);
    handleSearch(searchTerm, config, 1);
  };

  const handleShowEntitiesChange = (event) => {
    let config = { ...searchConfig, showEntities: event.target.checked };
    setSearchConfig(config);
    setCurrentPage(1);
    handleSearch(searchTerm, config, 1);
  };

  const handleShowFactsChange = (event) => {
    let config = { ...searchConfig, showFacts: event.target.checked };
    setSearchConfig(config);
    setCurrentPage(1);
    handleSearch(searchTerm, config, 1);
  };

  const handleShowCardsChange = (event) => {
    let config = { ...searchConfig, showCards: event.target.checked };
    setSearchConfig(config);
    setCurrentPage(1);
    handleSearch(searchTerm, config, 1);
  };

  const handleSearchTypeChange = (event) => {
    const newType = event.target.checked ? "typesense" : "classic";
    let config = { ...searchConfig, searchType: newType };
    setSearchConfig(config);
    setCurrentPage(1);
    handleSearch(searchTerm, config, 1);
  };

  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
    handleSearch(searchTerm, searchConfig, newPage);
  };

  return (
    <div>
      <div>
        <div className="bg-slate-100 p-4 border-b border-slate-300">
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
            {/* Search input section */}
            <div className="flex-grow flex items-center gap-2">
              <input
                type="text"
                id="title"
                value={searchTerm}
                placeholder="Search cards, entities, facts..."
                onChange={handleSearchUpdate}
                onKeyPress={(event: KeyboardEvent<HTMLInputElement>) => {
                  if (event.key === "Enter") {
                    handleSearch(searchTerm, searchConfig);
                  }
                }}
                className="flex-grow h-9 px-3 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              />
              <Button
                onClick={() => handleSearch(searchTerm, searchConfig)}
                className="h-9 bg-blue-600 hover:bg-blue-700 text-white rounded-md px-4 text-sm flex-shrink-0"
              >
                Search
              </Button>
            </div>

            {/* Controls section */}
            <div className="flex items-center gap-2 flex-shrink-0">
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
                          <Menu.Item key={option.value}>
                            {({ active }) => (
                              <button
                                onClick={() => setSearchConfig({ ...searchConfig, sortBy: option.value })}
                                className={`${
                                  searchConfig.sortBy === option.value
                                    ? 'bg-blue-50 text-blue-700 border-blue-200'
                                    : active
                                      ? 'bg-gray-100 text-gray-900'
                                      : 'text-gray-700'
                                } group flex rounded-md items-center w-full px-2 py-1.5 text-xs border ${
                                  searchConfig.sortBy === option.value ? 'border-blue-200' : 'border-transparent'
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
                      <div className="text-xs text-gray-500 mb-2 font-semibold">Search by Tag</div>
                      <div className="max-h-32 overflow-y-auto space-y-1">
                        {tags && tags.slice(0, 8).map((tag) => (
                          <Menu.Item key={tag.id}>
                            {({ active }) => (
                              <button
                                onClick={() => handleTagClick(tag.name)}
                                className={`${
                                  active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
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
                    <div className="text-xs text-gray-500 mb-2 font-semibold">Search Settings</div>
                    <div className="space-y-2">
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
                      <div className="hover:bg-gray-50 rounded px-2 py-1">
                        <label className="flex items-center text-sm cursor-pointer">
                          <input
                            type="checkbox"
                            checked={searchConfig.searchType === "typesense"}
                            onChange={handleSearchTypeChange}
                            className="mr-2"
                          />
                          New Search (Experimental)
                        </label>
                      </div>
                    </div>
                  </div>
                </div>
              </Menu.Items>
              </Menu>
            </div>
          </div>
        </div>
        {isLoading ? (
          <div className="flex justify-center w-full py-20">Loading</div>
        ) : (
          <div>
            {searchResults.length > 0 ? (
              <div>
                <SearchResultList
                  results={getFilteredResults()}
                  showPreview={searchConfig.showPreview}
                  onEntityClick={handleEntityClick}
                  onTagClick={handleTagClick}
                />
                <div className="flex justify-center items-center gap-4 mt-4 p-4">
                  <Button
                    onClick={() => handlePageChange(currentPage - 1)}
                    disabled={currentPage === 1}
                    children={"Previous"}
                  />
                  <span className="flex items-center text-sm text-gray-600">
                    Page {currentPage} of {totalPages} ({totalResults} results)
                  </span>
                  <Button
                    onClick={() => handlePageChange(currentPage + 1)}
                    disabled={currentPage === totalPages}
                    children={"Next"}
                  />
                </div>
              </div>
            ) : (
              <div>
                {error === null ? (
                  <div className="flex justify-center w-full py-20">
                    Search returned no results
                  </div>
                ) : (
                  <div className="flex justify-center w-full py-20">
                    Search returned an error: {error.message}
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Message display */}
      {message && (
        <div className="fixed bottom-4 right-4 bg-blue-500 text-white px-4 py-2 rounded shadow-lg z-50">
          {message}
        </div>
      )}

      {/* Star Search Dialog */}
      {showStarSearchDialog && (
        <StarSearchDialog
          searchTerm={searchTerm}
          searchConfig={searchConfig}
          onClose={() => setShowStarSearchDialog(false)}
          onStarSuccess={() => {
            // You might want to refresh something here
          }}
          setMessage={setMessage}
        />
      )}

    </div>
  );
}
