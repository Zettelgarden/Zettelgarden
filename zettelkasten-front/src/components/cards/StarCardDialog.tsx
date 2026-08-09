import React, { useState } from 'react';
import { BacklinkInputDropdownList } from './BacklinkInputDropdownList';
import { Modal } from '../ui/Modal';
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
    <Modal
      open
      onClose={onClose}
      size="md"
      dialogClassName="z-[100]"
      className="p-4"
    >
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
    </Modal>
  );
}
