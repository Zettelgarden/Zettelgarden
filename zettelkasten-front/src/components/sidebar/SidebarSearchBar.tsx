import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';

interface SidebarSearchBarProps {
  isCollapsed: boolean;
}

export function SidebarSearchBar({ isCollapsed }: SidebarSearchBarProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const navigate = useNavigate();

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      const trimmed = searchTerm.trim();
      if (trimmed) {
        navigate(`/app/search?term=${encodeURIComponent(trimmed)}`);
        setSearchTerm(''); // Clear input after navigation
      }
    }
  };

  if (isCollapsed) {
    return null;
  }

  return (
    <div className="px-3 pb-3">
      <div className="relative">
        {/* Search icon */}
        <svg
          className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
          />
        </svg>
        <input
          type="text"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Search cards..."
          className="w-full pl-9 pr-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>
    </div>
  );
}
