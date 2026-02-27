import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { PartialCard, SearchResult } from "../../models/Card";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { searchEmails, EmailSearchResult } from "../../api/email";
import { semanticSearchCards } from "../../api/cards";

interface QuickSearchWindowProps {
  setShowWindow: (showWindow: boolean) => void;
}

export function QuickSearchWindow({ setShowWindow }: QuickSearchWindowProps) {
  const navigate = useNavigate();
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [emailResults, setEmailResults] = useState<EmailSearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  function handleSelect(card: PartialCard) {
    setShowWindow(false);
    navigate(`/app/card/${card.id}`);
  }

  function handleEmailSelect(email: EmailSearchResult) {
    setShowWindow(false);
    navigate(`/app/emails/${email.id}`);
  }

  async function handleSearch(searchTerm: string) {
    setSearchQuery(searchTerm);
    if (!searchTerm.trim()) {
      setSearchResults([]);
      setEmailResults([]);
      return;
    }

    setIsSearching(true);
    try {
      // Search cards and emails in parallel
      const [cardsResponse, emailsResponse] = await Promise.all([
        semanticSearchCards(searchTerm, true, false, false, true, false, "sortByRanking", "typesense", false),
        searchEmails({ search_term: searchTerm, page: 1, per_page: 10 })
      ]);
      setSearchResults(cardsResponse);
      setEmailResults(emailsResponse.results ?? []);
    } catch (error) {
      console.error("Search failed:", error);
      setSearchResults([]);
      setEmailResults([]);
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
          placeholder="Search cards and emails..."
        />
      </div>
    </div>
  );
}
