import React from "react";
import { Dialog } from "@headlessui/react";
import { Entity } from "../../models/Card";

interface EntityMergeDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  currentEntity: Entity | null;
  entityToMerge: Entity | null;
  mergeDirection: 'into-current' | 'from-current';
  isMerging: boolean;
  mergeError: string | null;
}

export function EntityMergeDialog({
  isOpen,
  onClose,
  onConfirm,
  currentEntity,
  entityToMerge,
  mergeDirection,
  isMerging,
  mergeError,
}: EntityMergeDialogProps) {
  if (!entityToMerge) return null;

  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      className="fixed inset-0 z-50 flex items-center justify-center"
    >
      <div className="fixed inset-0 bg-black bg-opacity-30" aria-hidden="true" />
      <Dialog.Panel className="bg-white p-6 rounded-lg max-w-md mx-auto relative">
        <Dialog.Title className="text-lg font-semibold mb-4">
          Confirm Merge
        </Dialog.Title>
        <div className="mb-4">
          <p className="font-medium text-green-600 mb-2">
            Primary Entity (will be kept):<br />
            {mergeDirection === 'into-current' ? currentEntity?.name : entityToMerge.name}
          </p>
          <p className="text-gray-600 mb-2">This entity will be merged into the primary:</p>
          <p className="text-gray-800">• {mergeDirection === 'into-current' ? entityToMerge.name : currentEntity?.name}</p>
        </div>
        <p className="text-red-600 text-sm mb-4">
          This action cannot be undone. The merged entity will be deleted.
        </p>
        {mergeError && <p className="text-red-600 mb-2">{mergeError}</p>}
        <div className="flex justify-end gap-4">
          <button
            onClick={onClose}
            className="px-4 py-2 text-gray-600 hover:text-gray-800"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={isMerging}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
          >
            {isMerging ? "Merging..." : "Merge"}
          </button>
        </div>
      </Dialog.Panel>
    </Dialog>
  );
}
