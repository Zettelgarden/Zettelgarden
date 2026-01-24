import React from "react";
import { SortMethod, SORT_METHOD_LABELS } from "../../utils/cards";

interface SortControlProps {
  sortMethod: SortMethod;
  onSortChange: (method: SortMethod) => void;
  label?: string;
}

export function SortControl({ sortMethod, onSortChange, label }: SortControlProps) {
  return (
    <div className="flex items-center gap-2">
      {label && (
        <label className="text-xs font-medium text-gray-600 whitespace-nowrap">
          {label}
        </label>
      )}
      <select
        value={sortMethod}
        onChange={(e) => onSortChange(e.target.value as SortMethod)}
        className="text-xs border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 py-1 px-2"
      >
        {Object.entries(SORT_METHOD_LABELS).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>
    </div>
  );
}
