import React, { useState } from "react";
import { useTagContext } from "../../contexts/TagContext";
import { TagListItem } from "./TagListItem";
import { createTag } from "../../api/tags";

interface TagListInterface { }

type SortOption = "name" | "name-desc" | "tasks-asc" | "tasks-desc" | "cards-asc" | "cards-desc";

export function TagList({ }: TagListInterface) {
  const { tags, setRefreshTags } = useTagContext();
  const [sortOption, setSortOption] = useState<SortOption>("name");
  const [filterText, setFilterText] = useState<string>("");
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newTagName, setNewTagName] = useState("");
  const [error, setError] = useState<string | null>(null);

  const handleCreateTag = async () => {
    try {
      setError(null);
      if (!newTagName.trim()) {
        setError("Tag name is required");
        return;
      }
      await createTag({ name: newTagName.trim(), color: "black" });
      setNewTagName("");
      setIsCreateModalOpen(false);
      setRefreshTags(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create tag");
    }
  };

  const filteredTags = (tags || []).filter(tag =>
    tag.name.toLowerCase().includes(filterText.toLowerCase())
  );

  const sortedTags = [...filteredTags].sort((a, b) => {
    switch (sortOption) {
      case "name":
        return a.name.localeCompare(b.name);
      case "name-desc":
        return b.name.localeCompare(a.name);
      default:
        return 0;
    }
  });

  return (
    <div className="p-4 max-w-full">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-2xl font-bold">Tags</h2>
        <button
          onClick={() => setIsCreateModalOpen(true)}
          className="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded"
        >
          Create Tag
        </button>
      </div>

      {isCreateModalOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full">
            <h3 className="text-xl font-bold mb-4">Create New Tag</h3>
            {error && <div className="text-red-500 mb-4">{error}</div>}
            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">Name</label>
              <input
                type="text"
                value={newTagName}
                onChange={(e) => setNewTagName(e.target.value)}
                className="border border-gray-300 rounded px-3 py-2 w-full"
                placeholder="Enter tag name"
              />
            </div>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setIsCreateModalOpen(false)}
                className="px-4 py-3 min-h-[44px] text-gray-600 hover:text-gray-800"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateTag}
                className="bg-blue-500 hover:bg-blue-600 text-white px-4 py-3 min-h-[44px] rounded"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="mb-4 flex flex-wrap items-center gap-4">
        <div className="flex-grow">
          <label className="mr-2">Filter:</label>
          <input
            type="text"
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            className="border border-gray-300 rounded px-2 py-1 w-full md:w-auto"
            placeholder="Filter by name"
          />
        </div>
        <div className="min-w-[150px]">
          <label className="mr-2">Sort by:</label>
          <select
            value={sortOption}
            onChange={(e) => setSortOption(e.target.value as SortOption)}
            className="border border-gray-300 rounded px-2 py-1 w-full"
          >
            <option value="name">A-Z</option>
            <option value="name-desc">Z-A</option>
            <option value="tasks-asc">Least Tasks</option>
            <option value="tasks-desc">Most Tasks</option>
            <option value="cards-asc">Least Cards</option>
            <option value="cards-desc">Most Cards</option>
          </select>
        </div>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {sortedTags.length > 0 ? (
          sortedTags.map((tag) => <TagListItem key={tag.id} tag={tag} />)
        ) : (
          <div className="text-center text-gray-500">No tags available</div>
        )}
      </div>
    </div>
  );
}
