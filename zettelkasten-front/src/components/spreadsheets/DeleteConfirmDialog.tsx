import React from 'react';

interface DeleteConfirmDialogProps {
  isOpen: boolean;
  itemType: 'row' | 'column';
  index: number;
  onConfirm: () => void;
  onCancel: () => void;
}

export function DeleteConfirmDialog({
  isOpen,
  itemType,
  index,
  onConfirm,
  onCancel
}: DeleteConfirmDialogProps) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
        <h2 className="text-lg font-semibold text-gray-900 mb-2">
          Delete {itemType} {index + 1}?
        </h2>
        <p className="text-sm text-gray-600 mb-6">
          This {itemType} contains data. Are you sure you want to delete it?
          This action cannot be undone.
        </p>
        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded hover:bg-red-700"
          >
            Delete {itemType}
          </button>
        </div>
      </div>
    </div>
  );
}
