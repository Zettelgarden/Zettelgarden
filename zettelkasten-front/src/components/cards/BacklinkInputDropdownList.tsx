import React, { useState } from "react";
import { Combobox } from "@headlessui/react";
import { PartialCard } from "../../models/Card";
import { CardTag } from "./CardTag";
import { semanticSearchCards } from "../../api/cards";

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
  placeholder = "Search...",
  className = "",
  autoFocus = false,
  excludeCardId,
}: BacklinkInputDropdownListProps) {
  const [inputValue, setInputValue] = useState<string>("");

  const [searchResults, setSearchResults] = useState<PartialCard[]>([]);
  const latestRequestId = React.useRef(0);
  const [isLoading, setIsLoading] = useState(false);
  const inputRef = React.useRef<HTMLInputElement>(null);

  const debounceTimer = React.useRef<NodeJS.Timeout | null>(null);

  // Auto focus when component mounts if autoFocus is true
  React.useEffect(() => {
    if (autoFocus && inputRef.current) {
      inputRef.current.focus();
    }
  }, [autoFocus]);

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
        const results = await semanticSearchCards(value, true, false, false, true, "sortByRanking", "typesense", false);
        //const results = await semanticSearchCards(value, true, false, false);
        if (requestId === latestRequestId.current) {
          // Map SearchResult[] -> PartialCard[]
          const mapped: PartialCard[] = (results || []).map((r: any) => ({
            card_id: r.metadata.card_id,
            title: r.title,
            user_id: r.user_id ?? 0,
            parent_id: r.parent_id ?? null,
            id: Number(r.id),
            created_at: r.created_at ?? "",
            updated_at: r.updated_at ?? "",
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
        console.error("Search error:", err);
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
    setInputValue("");
    onSelect(card);
  }

  return (
    <div className={`relative w-full ${className}`}>
      <Combobox<PartialCard | null> value={null} onChange={handleSelect}>
        <div className="w-full">
          <div className="relative">
            <Combobox.Input
              ref={inputRef}
              className="w-full px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-200 transition-all duration-200"
              placeholder={placeholder}
              displayValue={() => inputValue}
              onChange={(e) => handleInputChange(e.target.value)}
            />
          </div>
          {(searchResults.length > 0 || inputValue.length > 0) && (
            <Combobox.Options className="w-full mt-1 overflow-hidden bg-white rounded-lg shadow-lg border border-gray-200 max-h-60 overflow-y-auto">
              {isLoading ? (
                <div className="p-3 text-gray-500">Loading...</div>
              ) : searchResults.length > 0 ? (
                searchResults.map((card) => (
                  <Combobox.Option
                    key={card.card_id}
                    value={card}
                    className={({ active }) =>
                      `cursor-pointer p-3 border-b border-gray-100 last:border-b-0 transition-colors duration-150 ${active ? "bg-blue-50" : ""
                      }`
                    }
                  >
                    <CardTag card={card} showTitle={true} />
                  </Combobox.Option>
                ))
              ) : (
                inputValue.length > 0 && (
                  <div className="p-3 text-gray-500">No results found</div>
                )
              )}
            </Combobox.Options>
          )}
        </div>
      </Combobox>
    </div>
  );
}
