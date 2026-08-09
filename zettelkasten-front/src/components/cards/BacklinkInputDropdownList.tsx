import React, { useState } from 'react';
import { Combobox } from '../ui/Combobox';
import { PartialCard } from '../../models/Card';
import { CardTag } from './CardTag';
import { semanticSearchCards } from '../../api/cards';

interface BacklinkInputDropdownListProps {
  onSelect: (card: PartialCard) => void;
  onSearch: (searchTerm: string) => void;
  placeholder?: string;
  className?: string;
  autoFocus?: boolean;
  excludeCardId?: number; // ID of card to exclude from results
}

export function BacklinkInputDropdownList({
  onSelect,
  onSearch,
  placeholder = 'Search...',
  className = '',
  autoFocus = false,
  excludeCardId,
}: BacklinkInputDropdownListProps) {
  const [inputValue, setInputValue] = useState<string>('');

  const [searchResults, setSearchResults] = useState<PartialCard[]>([]);
  const latestRequestId = React.useRef(0);
  const [isLoading, setIsLoading] = useState(false);

  const debounceTimer = React.useRef<NodeJS.Timeout | null>(null);

  function handleInputChange(value: string) {
    setInputValue(value);
    onSearch(value);

    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current);
    }

    if (!value) {
      setSearchResults([]);
      return;
    }

    debounceTimer.current = setTimeout(async () => {
      const requestId = ++latestRequestId.current;
      setIsLoading(true);
      try {
        const results = await semanticSearchCards(
          value,
          true,
          false,
          false,
          true,
          false,
          'sortByRanking',
          'typesense',
          false,
        );
        //const results = await semanticSearchCards(value, true, false, false);
        if (requestId === latestRequestId.current) {
          // Map SearchResult[] -> PartialCard[]
          const mapped: PartialCard[] = (results || []).map((r: any) => ({
            card_id: r.metadata.card_id,
            title: r.title,
            user_id: r.user_id ?? 0,
            parent_id: r.parent_id ?? null,
            id: r.pk != null ? Number(r.pk) : Number(r.id), // Use pk (internal DB ID) if available
            created_at: r.created_at ?? '',
            updated_at: r.updated_at ?? '',
            tags: r.tags ?? [],
          }));

          // Filter out the current card if excludeCardId is provided
          const filtered = excludeCardId
            ? mapped.filter((card) => card.id !== excludeCardId)
            : mapped;

          // Reorder results so exact card_id match comes first
          const ordered = filtered.sort((a, b) => {
            if (a.card_id === value) return -1;
            if (b.card_id === value) return 1;
            return 0;
          });

          setSearchResults(ordered);
        }
      } catch (err) {
        console.error('Search error:', err);
        if (requestId === latestRequestId.current) {
          setSearchResults([]);
        }
      } finally {
        if (requestId === latestRequestId.current) {
          setIsLoading(false);
        }
      }
    }, 500); // 500ms debounce delay
  }

  function handleSelect(card: PartialCard) {
    setInputValue('');
    onSelect(card);
  }

  return (
    <Combobox<PartialCard>
      value={null}
      onChange={handleSelect}
      inputValue={inputValue}
      onInputChange={handleInputChange}
      displayValue={() => inputValue}
      placeholder={placeholder}
      autoFocus={autoFocus}
      className={className}
      inputStyle={{ fontSize: '16px' }}
      isLoading={isLoading}
      options={searchResults}
      getOptionKey={(card) => card.card_id}
      renderOption={(card, active) => (
        <div
          className={`cursor-pointer px-4 py-3 min-h-[44px] border-b border-gray-100 last:border-b-0 transition-colors duration-150 ${
            active ? 'bg-blue-50' : ''
          }`}
        >
          <CardTag card={card} showTitle={true} />
        </div>
      )}
    />
  );
}
