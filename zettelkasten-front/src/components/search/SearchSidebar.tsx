import React, { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { StarredSearch } from "../../models/StarredSearch";
import { Tag } from "../../models/Tags";
import { getStarredSearches, unstarSearch } from "../../api/starredSearches";
import { useToast } from "../toast/ToastContext";

interface SearchSidebarProps {
  tags: Tag[];
  onTagClick: (tagName: string) => void;
}

/**
 * Left sidebar panel showing starred searches and tags
 * Navigation-only component - no search input or configuration
 */
export function SearchSidebar({
  tags,
  onTagClick,
}: SearchSidebarProps) {
  const [starredSearches, setStarredSearches] = useState<StarredSearch[]>([]);
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

  return (
    <div className="hidden md:flex w-72 border-r border-gray-200 p-4 overflow-y-auto bg-gray-50 flex-shrink-0 flex-col">
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

      {/* Tags Section */}
      {tags && tags.length > 0 && (
        <div className="mb-4">
          <div className="text-xs text-gray-500 mb-2 font-semibold uppercase tracking-wider">Search by Tag</div>
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
