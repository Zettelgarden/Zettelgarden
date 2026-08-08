import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { PartialCard, SearchResult } from '../../models/Card';
import { BacklinkInputDropdownList } from './BacklinkInputDropdownList';
import { semanticSearchCards } from '../../api/cards';

interface QuickSearchWindowProps {
  setShowWindow: (showWindow: boolean) => void;
}

export function QuickSearchWindow({ setShowWindow }: QuickSearchWindowProps) {
  const navigate = useNavigate();
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  function handleSelect(card: PartialCard) {
    setShowWindow(false);
    navigate(`/app/card/${card.id}`);
  }

  async function handleSearch(searchTerm: string) {
    setSearchQuery(searchTerm);
    if (!searchTerm.trim()) {
      setSearchResults([]);
      return;
    }

    setIsSearching(true);
    try {
      const cardsResponse = await semanticSearchCards(
        searchTerm,
        true,
        false,
        false,
        true,
        false,
        'sortByRanking',
        'typesense',
        false,
      );
      setSearchResults(cardsResponse);
    } catch (error) {
      console.error('Search failed:', error);
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  }

  return (
    <div
      className="fixed top-0 left-0 w-full h-full bg-black/50 flex justify-center items-center z-[1000]"
      onClick={() => setShowWindow(false)}
    >
      <div
        className="bg-white p-3 rounded-lg shadow-lg max-w-[672px] w-[95%] max-h-[90vh] overflow-y-visible sm:p-4 sm:w-[90%]"
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
