import React, { useState } from "react";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";

interface BacklinkDialogProps {
  onClose: () => void;
  onSelect: (card: PartialCard) => void;
  setMessage: (message: string) => void;
}

export function BacklinkDialog({ onClose, onSelect, setMessage }: BacklinkDialogProps) {
  const [searchResults, setSearchResults] = useState<PartialCard[]>([]);

  function handleSearch(searchTerm: string) {
  }

  function handleSelect(card: PartialCard) {
    onSelect(card);
    setMessage(`Backlink to "${card.title}" added successfully`);
    onClose(); // Close the dialog
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[100]">
      <div className="bg-white p-4 rounded-lg shadow-lg w-full max-w-md mx-4">
        <h3 className="text-lg font-medium mb-4">Add Backlink to Card</h3>
        <BacklinkInputDropdownList
          onSelect={handleSelect}
          onSearch={handleSearch}
          placeholder="Search for a card to link..."
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
