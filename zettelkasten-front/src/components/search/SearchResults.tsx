import React from "react";
import { SearchResult } from "../../models/Card";
import { SearchConfig } from "../../models/StarredSearch";
import { Button } from "../../components/Button";
import { SearchResultList } from "../../components/cards/SearchResultList";

interface SearchResultsProps {
  searchResults: SearchResult[];
  isLoading: boolean;
  error: Error | null;
  searchConfig: SearchConfig;
  totalResults: number;
  totalPages: number;
  currentPage: number;
  handlePageChange: (newPage: number) => void;
  handleEntityClick: (entityName: string) => void;
  handleTagClick: (tagName: string) => void;
  setSearchResults: (results: SearchResult[]) => void;
}

export function SearchResults({
  searchResults,
  isLoading,
  error,
  searchConfig,
  totalResults,
  totalPages,
  currentPage,
  handlePageChange,
  handleEntityClick,
  handleTagClick,
  setSearchResults,
}: SearchResultsProps) {
  function getFilteredResults(): SearchResult[] {
    return searchResults
      .filter(result => !searchConfig.onlyParentCards || !result.id.includes("/"));
  }

  return (
    <>
      {isLoading ? (
        <div className="flex justify-center w-full py-20">Loading</div>
      ) : (
        <div>
          {getFilteredResults().length > 0 ? (
            <div>
              <SearchResultList
                results={getFilteredResults()}
                showPreview={searchConfig.showPreview}
                onEntityClick={handleEntityClick}
                onTagClick={handleTagClick}
                onResultsUpdate={setSearchResults}
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
    </>
  );
}