import React from "react";

interface EntityListToolbarProps {
  filterText: string;
  onFilterChange: (value: string) => void;
  sortBy: "name" | "cards" | "created_at";
  sortDirection: "asc" | "desc";
  onSortChange: (sortBy: "name" | "cards" | "created_at", direction: "asc" | "desc") => void;
  onSearch: () => void;
  onKeyPress: (e: React.KeyboardEvent) => void;
}

export function EntityListToolbar({
  filterText,
  onFilterChange,
  sortBy,
  sortDirection,
  onSortChange,
  onSearch,
  onKeyPress,
}: EntityListToolbarProps) {
  return (
    <div className="mb-4 flex gap-2">
      <input
        type="text"
        placeholder="Search entities... (Press Enter to search)"
        value={filterText}
        onChange={(e) => onFilterChange(e.target.value)}
        onKeyPress={onKeyPress}
        className="flex-1 p-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
      <button
        onClick={onSearch}
        className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        Search
      </button>
      <select
        value={`${sortBy}-${sortDirection}`}
        onChange={(e) => {
          const [newSortBy, newDirection] = e.target.value.split("-") as [
            "name" | "cards" | "created_at",
            "asc" | "desc"
          ];
          onSortChange(newSortBy, newDirection);
        }}
        className="p-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        <option value="name-asc">Name (A-Z)</option>
        <option value="name-desc">Name (Z-A)</option>
        <option value="cards-desc">Most Cards</option>
        <option value="cards-asc">Least Cards</option>
        <option value="created_at-desc">Newest First</option>
        <option value="created_at-asc">Oldest First</option>
      </select>
    </div>
  );
} 