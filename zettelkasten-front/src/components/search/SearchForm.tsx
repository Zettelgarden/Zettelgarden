import React, { ChangeEvent, KeyboardEvent } from 'react';
import { Button } from '../../components/Button';
import { SearchConfig } from '../../models/StarredSearch';

interface SearchFormProps {
  searchTerm: string;
  setSearchTerm: (searchTerm: string) => void;
  searchConfig: SearchConfig;
  onSearch: (searchTerm: string, config: SearchConfig) => void;
  disabled?: boolean;
}

export function SearchForm({
  searchTerm,
  setSearchTerm,
  searchConfig,
  onSearch,
  disabled = false,
}: SearchFormProps) {
  function handleSearchUpdate(e: ChangeEvent<HTMLInputElement>) {
    setSearchTerm(e.target.value);
  }

  function handleKeyPress(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Enter' && !disabled) {
      onSearch(searchTerm, searchConfig);
    }
  }

  return (
    <div className="flex-grow flex items-center gap-2">
      <input
        type="text"
        id="title"
        value={searchTerm}
        placeholder="Search cards, entities, facts..."
        onChange={handleSearchUpdate}
        onKeyPress={handleKeyPress}
        disabled={disabled}
        className="flex-grow h-9 px-3 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:opacity-50"
      />
      <Button
        onClick={() => onSearch(searchTerm, searchConfig)}
        disabled={disabled}
        className="h-9 bg-blue-600 hover:bg-blue-700 text-white rounded-md px-4 text-sm flex-shrink-0 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Search
      </Button>
    </div>
  );
}
