import React, { useState, useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { semanticSearchCards, semanticSearchCardsPaginated } from "../../api/cards";
import { fetchUserTags } from "../../api/tags";
import { SearchResult } from "../../models/Card";
import { Tag } from "../../models/Tags";
import { SearchConfig as SearchConfigType } from "../../models/StarredSearch";
import { SearchResultList } from "../../components/cards/SearchResultList";
import { StarSearchDialog } from "../../components/search/StarSearchDialog";
import { SearchForm } from "../../components/search/SearchForm";
import { SearchConfig } from "../../components/search/SearchConfig";
import { SearchResults } from "../../components/search/SearchResults";
import { SearchDesktopLayout } from "../../components/search/SearchDesktopLayout";
import { SearchMobileLayout } from "../../components/search/SearchMobileLayout";
import { getStarredSearches } from "../../api/starredSearches";
import { useTagContext } from "../../contexts/TagContext";
import { Entity } from "../../models/Card";
import { fetchEntityByName } from "../../api/entities";
import { setDocumentTitle } from "../../utils/title";
import { useResponsiveLayout } from "../../hooks/useResponsiveLayout";

import { useDialogState } from "../../contexts/DialogStateContext";
import { useUIState } from "../../contexts/UIStateContext";

interface SearchPageProps {
  searchTerm: string;
  setSearchTerm: (searchTerm: string) => void;
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;
  searchConfig: SearchConfigType
  setSearchConfig: (config: SearchConfigType) => void;
}

export function SearchPage({
  searchTerm,
  setSearchTerm,
  searchResults,
  setSearchResults,
  searchConfig,
  setSearchConfig,
}: SearchPageProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const { setHideGlobalSidebar } = useUIState();
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
  const [isInitializing, setIsInitializing] = useState<boolean>(true);
  const {
    showEntityDialog,
    setShowEntityDialog,
    selectedEntity,
    setSelectedEntity,
  } = useDialogState();

  const params = new URLSearchParams(location.search);
  const starredId = params.get("starred");

  // Responsive layout state
  const { isMobile, mobileView, setMobileView } = useResponsiveLayout();
  const [selectedCard, setSelectedCard] = useState<SearchResult | null>(null);

  // Hide global sidebar on desktop (SearchDesktopLayout has its own navigation)
  useEffect(() => {
    setHideGlobalSidebar(true);
    return () => {
      setHideGlobalSidebar(false);
    };
  }, [setHideGlobalSidebar]);

  // Handler for clicking on a search result
  const handleResultClick = (result: SearchResult) => {
    setSelectedCard(result);
    if (isMobile) {
      setMobileView('detail');
    }
  };

  // Function to update URL with current search state
  const updateURL = (term: string, config: SearchConfigType, page: number = 1) => {
    const params = new URLSearchParams();

    if (term.trim()) {
      params.set('term', term);
    }

    // Only add non-default search config parameters to keep URL clean
    if (config.searchType !== 'classic') {
      params.set('searchType', config.searchType);
    }
    if (config.showEntities) {
      params.set('showEntities', 'true');
    }
    if (config.showFacts) {
      params.set('showFacts', 'true');
    }
    if (!config.showCards) {
      params.set('showCards', 'false');
    }
    if (config.useFullText) {
      params.set('useFullText', 'true');
    }
    if (config.onlyParentCards) {
      params.set('onlyParentCards', 'true');
    }
    if (config.onlyEmptyCardId) {
      params.set('onlyEmptyCardId', 'true');
    }
    if (config.schemaId !== null && config.schemaId !== undefined) {
      params.set('schemaId', config.schemaId.toString());
    }
    if (page > 1) {
      params.set('page', page.toString());
    }

    const newURL = params.toString() ? `${location.pathname}?${params.toString()}` : location.pathname;
    navigate(newURL, { replace: true });
  };

  async function handleSearch(searchTerm: string, config: SearchConfigType, page: number = 1) {
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
        config.onlyEmptyCardId,
        config.schemaId ?? undefined,
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

    // Update URL to persist search state (but not during initialization)
    if (!isInitializing) {
      updateURL(searchTerm, config, page);
    }
  }

  useEffect(() => {
    setIsInitializing(true);
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
      // Read search configuration from URL parameters
      const page = parseInt(params.get("page") || "1", 10);
      const schemaId = params.get("schemaId");
      const urlConfig = {
        ...searchConfig,
        searchType: params.get("searchType") || "classic",
        showEntities: params.get("showEntities") === "true",
        showFacts: params.get("showFacts") === "true",
        showCards: params.get("showCards") !== "false", // default true
        useFullText: params.get("useFullText") === "true",
        onlyParentCards: params.get("onlyParentCards") === "true",
        onlyEmptyCardId: params.get("onlyEmptyCardId") === "true",
        useClassicSearch: params.get("searchType") !== "typesense",
        schemaId: schemaId ? parseInt(schemaId) : undefined
      };

      if (recent !== null) {
        const config = { ...urlConfig, useClassicSearch: true };
        setSearchConfig(config);
        setSearchTerm("");
        setCurrentPage(page);
        await handleSearch("", config, page);
      } else if (term) {
        setSearchConfig(urlConfig);
        setSearchTerm(term);
        setCurrentPage(page);
        await handleSearch(term, urlConfig, page);
      } else {
        const config = { ...urlConfig, sortBy: "sortByRanking" };
        setSearchConfig(config);
        setCurrentPage(page);
        await handleSearch("", config, page);
      }
    };

    initializeSearch().finally(() => {
      setIsInitializing(false);
    });
  }, [location.search]); // Re-run when the URL search parameters change


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


  const handlePageChange = (newPage: number) => {
    setCurrentPage(newPage);
    handleSearch(searchTerm, searchConfig, newPage);
  };

  const handleConfigChangeWithSearch = (config: SearchConfigType, resetPage?: boolean) => {
    if (resetPage) {
      setCurrentPage(1);
    }
    handleSearch(searchTerm, config, resetPage ? 1 : currentPage);
  };

  // Fetch schemas for filter panel
  const [schemas, setSchemas] = useState<any[]>([]);
  useEffect(() => {
    const loadSchemas = async () => {
      try {
        const { fetchSchemas } = await import("../../api/schemas");
        const data = await fetchSchemas();
        setSchemas(data);
      } catch (error) {
        console.error("Failed to load schemas:", error);
      }
    };
    loadSchemas();
  }, []);

  return (
    <div className="h-screen overflow-hidden">
      {isMobile ? (
        <SearchMobileLayout
          mobileView={mobileView}
          setMobileView={setMobileView}
          searchTerm={searchTerm}
          setSearchTerm={setSearchTerm}
          searchConfig={searchConfig}
          setSearchConfig={setSearchConfig}
          searchResults={searchResults}
          setSearchResults={setSearchResults}
          isLoading={isLoading}
          error={error}
          totalResults={totalResults}
          totalPages={totalPages}
          currentPage={currentPage}
          tags={tags}
          starredId={starredId}
          schemas={schemas}
          setShowStarSearchDialog={setShowStarSearchDialog}
          onSearch={handleSearch}
          handlePageChange={handlePageChange}
          handleEntityClick={handleEntityClick}
          handleTagClick={handleTagClick}
          onMenuClick={() => setMobileView('filters')}
        />
      ) : (
        <SearchDesktopLayout
          searchTerm={searchTerm}
          setSearchTerm={setSearchTerm}
          searchConfig={searchConfig}
          setSearchConfig={setSearchConfig}
          searchResults={searchResults}
          setSearchResults={setSearchResults}
          isLoading={isLoading}
          error={error}
          totalResults={totalResults}
          totalPages={totalPages}
          currentPage={currentPage}
          tags={tags}
          starredId={starredId}
          setShowStarSearchDialog={setShowStarSearchDialog}
          onSearch={handleSearch}
          onPageChange={handlePageChange}
          onEntityClick={handleEntityClick}
          onTagClick={handleTagClick}
          selectedResult={selectedCard}
          setSelectedResult={setSelectedCard}
          onEditCard={(cardId) => navigate(`/app/card/${cardId}`)}
        />
      )}

      {/* Message display */}
      {message && (
        <div className="fixed bottom-4 right-4 bg-blue-500 text-white px-4 py-2 rounded shadow-lg z-50 safe-bottom-fixed safe-right-fixed">
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
            // Refresh starred searches
          }}
          setMessage={setMessage}
        />
      )}

    </div>
  );
}
