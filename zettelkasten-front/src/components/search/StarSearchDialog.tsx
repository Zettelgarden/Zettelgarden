import React, { useState } from "react";
import { starSearch } from "../../api/starredSearches";
import { SearchConfig } from "../../models/StarredSearch";

interface StarSearchDialogProps {
  searchTerm: string;
  searchConfig: SearchConfig;
  onClose: () => void;
  onStarSuccess: () => void;
  setMessage: (message: string) => void;
}

export function StarSearchDialog({
  searchTerm,
  searchConfig,
  onClose,
  onStarSuccess,
  setMessage
}: StarSearchDialogProps) {
  const [title, setTitle] = useState<string>(searchTerm || "Untitled Search");

  function handleSave() {
    if (!title.trim()) {
      setMessage("Please enter a title for the starred search");
      return;
    }

    starSearch(title, searchTerm, searchConfig)
      .then(() => {
        setMessage(`Search "${title}" starred successfully`);
        onStarSuccess();
        onClose();
      })
      .catch(error => {
        console.error("Error starring search:", error);
        setMessage(`Error starring search: ${error.message}`);
      });
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[100]">
      <div className="bg-white p-4 rounded-lg shadow-lg w-full max-w-md mx-4">
        <h3 className="text-lg font-medium mb-4">Star Current Search</h3>

        <div className="mb-4">
          <label htmlFor="search-title" className="block text-sm font-medium text-gray-700 mb-1">
            Title
          </label>
          <input
            id="search-title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full p-2 border border-gray-300 rounded"
            placeholder="Enter a title for this search"
          />
        </div>

        <div className="mb-4">
          <h4 className="text-sm font-medium text-gray-700 mb-1">Search Details</h4>
          <div className="bg-gray-50 p-3 rounded text-sm">
            <p><strong>Search Term:</strong> {searchTerm || "(empty)"}</p>
            <p><strong>Search Type:</strong> {searchConfig.useClassicSearch ? "Classic" : "Semantic"}</p>
            <p><strong>Sort By:</strong> {searchConfig.sortBy}</p>
            <p><strong>Full Text:</strong> {searchConfig.useFullText ? "Yes" : "No"}</p>
            <p><strong>Only Parent Cards:</strong> {searchConfig.onlyParentCards ? "Yes" : "No"}</p>
            <p><strong>Show Entities:</strong> {searchConfig.showEntities ? "Yes" : "No"}</p>
            <p><strong>Show Facts:</strong> {searchConfig.showFacts ? "Yes" : "No"}</p>
          </div>
          <p className="text-xs text-gray-500 mt-1">
            These search settings will be saved and applied when you click on this starred search.
          </p>
        </div>

        <div className="flex justify-end space-x-2">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-200 text-gray-800 rounded hover:bg-gray-300"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
