import React, { useState } from "react";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { PartialCard } from "../../models/Card";

interface BacklinkInputProps {
  addBacklink: (selectedCard: PartialCard) => void;
}

export function BacklinkInput({ addBacklink }: BacklinkInputProps) {

  function handleSearch(searchTerm: string) {
  }

  function handleSelect(card: PartialCard) {
    addBacklink(card);
  }

  return (
    <BacklinkInputDropdownList
      onSelect={handleSelect}
      onSearch={handleSearch}
      placeholder="Add Backlink"
      className="max-w-md"
    />
  );
}
