import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { PartialCard } from "../../models/Card";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";

interface QuickSearchWindowProps {
  setShowWindow: (showWindow: boolean) => void;
}

export function QuickSearchWindow({ setShowWindow }: QuickSearchWindowProps) {
  const navigate = useNavigate();

  function handleSelect(card: PartialCard) {
    setShowWindow(false);
    navigate(`/app/card/${card.id}`);
  }

  async function handleSearch(searchTerm: string) {
  }

  return (
    <div
      className="fixed top-0 left-0 w-full h-full bg-black/50 flex justify-center items-center z-[1000]"
      onClick={() => setShowWindow(false)}
    >
      <div
        className="bg-white p-4 rounded-lg shadow-lg max-w-[672px] w-[95%] max-h-[90vh] overflow-y-visible sm:p-6 sm:w-[90%]"
        onClick={(e) => e.stopPropagation()}
      >
        <BacklinkInputDropdownList
          onSelect={handleSelect}
          onSearch={handleSearch}
          placeholder="Search cards..."
        />
      </div>
    </div>
  );
}
