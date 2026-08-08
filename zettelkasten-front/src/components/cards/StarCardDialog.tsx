import React, { useState } from 'react';
import { BacklinkInputDropdownList } from './BacklinkInputDropdownList';
import { PartialCard } from '../../models/Card';
import { starCard } from '../../api/cards';
import { useToast } from '../toast/ToastContext';

interface StarCardDialogProps {
  onClose: () => void;
  onStarSuccess: () => void;
}

export function StarCardDialog({
  onClose,
  onStarSuccess,
}: StarCardDialogProps) {
  const { showToast } = useToast();

  function handleSearch(searchTerm: string) {}

  function handleSelect(card: PartialCard) {
    starCard(card.id)
      .then(() => {
        showToast('success', `Card "${card.title}" starred successfully`);
        onStarSuccess(); // Refresh the starred cards list
        onClose(); // Close the dialog
      })
      .catch((error) => {
        console.error('Error starring card:', error);
        showToast('error', 'Failed to star card', error.message);
      });
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[100]">
      <div className="bg-white p-4 rounded-lg shadow-lg w-full max-w-md mx-4">
        <h3 className="text-lg font-medium mb-4">Star Existing Card</h3>
        <BacklinkInputDropdownList
          onSelect={handleSelect}
          onSearch={handleSearch}
          placeholder="Search for a card to star..."
        />
        <div className="mt-4 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-3 min-h-[44px] bg-gray-200 text-gray-800 rounded hover:bg-gray-300"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
