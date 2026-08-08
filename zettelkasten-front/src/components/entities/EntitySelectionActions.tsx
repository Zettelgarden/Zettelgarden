import React from 'react';

interface EntitySelectionActionsProps {
  selectedCount: number;
  onMerge: () => void;
  onDelete: () => void;
  isMerging: boolean;
  isDeleting: boolean;
}

export function EntitySelectionActions({
  selectedCount,
  onMerge,
  onDelete,
  isMerging,
  isDeleting,
}: EntitySelectionActionsProps) {
  if (selectedCount === 0) return null;

  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-gray-600">{selectedCount} selected</span>

      <button
        onClick={onMerge}
        disabled={selectedCount < 2 || isMerging}
        className="px-3 py-1 text-sm bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isMerging ? 'Merging...' : 'Merge'}
      </button>

      <button
        onClick={onDelete}
        disabled={isDeleting}
        className="px-3 py-1 text-sm bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isDeleting ? 'Deleting...' : 'Delete'}
      </button>
    </div>
  );
}
