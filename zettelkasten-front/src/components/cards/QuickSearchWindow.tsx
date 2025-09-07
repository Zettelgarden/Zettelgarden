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
      className="create-task-popup-overlay"
      onClick={() => setShowWindow(false)}
    >
      <div
        className="create-task-popup-content"
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
