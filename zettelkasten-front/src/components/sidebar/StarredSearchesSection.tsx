import React, { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import { StarredSearch } from "../../models/StarredSearch";
import { getStarredSearches, unstarSearch } from "../../api/starredSearches";

interface StarredSearchesSectionProps {
  setMessage: (message: string) => void;
}

export function StarredSearchesSection({ setMessage }: StarredSearchesSectionProps) {
  const [starredSearches, setStarredSearches] = useState<StarredSearch[]>([]);

  const handleUnstarSearch = (searchId: number) => {
    unstarSearch(searchId)
      .then(() => {
        // Refresh the starred searches list after unstarring
        refreshStarredSearches();
        // Show a success message
        setMessage("Search unstarred successfully");
      })
      .catch(error => {
        console.error("Error unstarring search:", error);
        setMessage("Error unstarring search");
      });
  };

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

  return (
    <>
      <hr />
      <div className="p-2">
        <div className="flex items-center justify-between mb-2 px-2">
          <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
            Starred Searches
          </h3>
        </div>
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
                    className="text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity px-1"
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
    </>
  );
}