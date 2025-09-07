import React, { useState } from "react";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";
import { pinCard } from "../../api/cards";

interface PinCardDialogProps {
  onClose: () => void;
  onPinSuccess: () => void;
  setMessage: (message: string) => void;
}

export function PinCardDialog({ onClose, onPinSuccess, setMessage }: PinCardDialogProps) {
  function handleSearch(searchTerm: string) {
  }

  function handleSelect(card: PartialCard) {
    pinCard(card.id)
      .then(() => {
        setMessage(`Card "${card.title}" pinned successfully`);
        onPinSuccess(); // Refresh the pinned cards list
        onClose(); // Close the dialog
      })
      .catch(error => {
        console.error("Error pinning card:", error);
        setMessage(`Error pinning card: ${error.message}`);
      });
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[100]">
      <div className="bg-white p-4 rounded-lg shadow-lg w-full max-w-md mx-4">
        <h3 className="text-lg font-medium mb-4">Pin Existing Card</h3>
        <BacklinkInputDropdownList
          onSelect={handleSelect}
          onSearch={handleSearch}
          placeholder="Search for a card to pin..."
        />
        <div className="mt-4 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-gray-200 text-gray-800 rounded hover:bg-gray-300"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
