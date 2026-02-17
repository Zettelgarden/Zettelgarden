import React, { useState } from "react";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";
import { useToast } from "../toast/ToastContext";

interface BacklinkDialogProps {
  onClose: () => void;
  onSelect: (card: PartialCard) => void;
  excludeCardId?: number;
}

export function BacklinkDialog({ onClose, onSelect, excludeCardId }: BacklinkDialogProps) {
  const [searchResults, setSearchResults] = useState<PartialCard[]>([]);
  const { showToast } = useToast();

  function handleSearch(searchTerm: string) {
  }

  function handleSelect(card: PartialCard) {
    onSelect(card);
    showToast("success", "Backlink Added", `Backlink to "${card.title}" added successfully`);
    onClose(); // Close the dialog
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[100]">
      <div className="bg-white p-3 rounded-lg shadow-lg w-full max-w-md mx-4">
        <h3 className="text-base font-medium mb-3">Add Backlink to Card</h3>
        <BacklinkInputDropdownList
          onSelect={handleSelect}
          onSearch={handleSearch}
          placeholder="Search for a card to link..."
          excludeCardId={excludeCardId}
        />
        <div className="mt-3 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-3 min-h-[44px] bg-gray-200 text-gray-800 rounded hover:bg-gray-300 text-sm"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
