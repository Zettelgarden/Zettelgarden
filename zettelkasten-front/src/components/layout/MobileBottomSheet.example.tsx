/**
 * MobileBottomSheet Usage Examples
 *
 * This file demonstrates how to use the MobileBottomSheet component.
 */

import React, { useState } from "react";
import { MobileBottomSheet } from "./MobileBottomSheet";

// Example 1: Basic usage with title and close button
export function BasicExample() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button onClick={() => setIsOpen(true)}>Open Bottom Sheet</button>

      <MobileBottomSheet
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="My Options"
      >
        <div className="p-4 space-y-2">
          <button className="w-full text-left px-4 py-3 bg-gray-100 rounded-lg hover:bg-gray-200">
            Option 1
          </button>
          <button className="w-full text-left px-4 py-3 bg-gray-100 rounded-lg hover:bg-gray-200">
            Option 2
          </button>
          <button className="w-full text-left px-4 py-3 bg-gray-100 rounded-lg hover:bg-gray-200">
            Option 3
          </button>
        </div>
      </MobileBottomSheet>
    </>
  );
}

// Example 2: Filter bottom sheet
export function FilterExample() {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedFilter, setSelectedFilter] = useState<string>("all");

  const filters = ["all", "unread", "starred", "recent"];

  return (
    <>
      <button onClick={() => setIsOpen(true)}>Filter Options</button>

      <MobileBottomSheet
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="Filter By"
      >
        <div className="p-4 space-y-2">
          {filters.map((filter) => (
            <button
              key={filter}
              onClick={() => {
                setSelectedFilter(filter);
                setIsOpen(false);
              }}
              className={`w-full text-left px-4 py-3 rounded-lg capitalize transition-colors ${
                selectedFilter === filter
                  ? "bg-blue-100 text-blue-900"
                  : "bg-gray-100 hover:bg-gray-200"
              }`}
            >
              {filter}
            </button>
          ))}
        </div>
      </MobileBottomSheet>
    </>
  );
}

// Example 3: Without title or close button (custom header)
export function CustomHeaderExample() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button onClick={() => setIsOpen(true)}>Open Custom Sheet</button>

      <MobileBottomSheet
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        showCloseButton={false}
      >
        {/* Custom header */}
        <div className="px-4 pb-3 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" />
                </svg>
              </div>
              <h2 className="text-lg font-semibold text-gray-900">Add Item</h2>
            </div>
            <button
              onClick={() => setIsOpen(false)}
              className="text-gray-500 hover:text-gray-700"
            >
              Done
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-4">
          <p>Your content here...</p>
        </div>
      </MobileBottomSheet>
    </>
  );
}

// Example 4: With custom max height
export function CustomHeightExample() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button onClick={() => setIsOpen(true)}>Open Tall Sheet</button>

      <MobileBottomSheet
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="Long List"
        maxHeight="90vh"
      >
        <div className="p-4 space-y-2">
          {Array.from({ length: 50 }).map((_, i) => (
            <div key={i} className="px-4 py-3 bg-gray-100 rounded-lg">
              Item {i + 1}
            </div>
          ))}
        </div>
      </MobileBottomSheet>
    </>
  );
}

// Example 5: RSS-style feeds bottom sheet
interface Feed {
  id: number;
  name: string;
}

export function FeedsExample() {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedFeed, setSelectedFeed] = useState<number | null>(null);

  const feeds: Feed[] = [
    { id: 1, name: "Tech News" },
    { id: 2, name: "Design Blog" },
    { id: 3, name: "Development Updates" },
  ];

  return (
    <>
      <button onClick={() => setIsOpen(true)}>Select Feed</button>

      <MobileBottomSheet
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        title="Feeds"
      >
        <div className="px-4 py-3 space-y-2">
          <button
            onClick={() => {
              setSelectedFeed(null);
              setIsOpen(false);
            }}
            className={`w-full text-left px-4 py-3 rounded-lg font-medium transition-colors ${
              selectedFeed === null
                ? "bg-blue-100 text-blue-900"
                : "hover:bg-gray-100 bg-gray-50"
            }`}
          >
            All Feeds
          </button>

          {feeds.map((feed) => (
            <button
              key={feed.id}
              onClick={() => {
                setSelectedFeed(feed.id);
                setIsOpen(false);
              }}
              className={`w-full text-left px-4 py-3 rounded-lg transition-colors ${
                selectedFeed === feed.id
                  ? "bg-blue-100 text-blue-900"
                  : "hover:bg-gray-100 bg-gray-50"
              }`}
            >
              {feed.name}
            </button>
          ))}
        </div>

        {/* Bottom Actions */}
        <div className="px-4 py-3 border-t border-gray-200">
          <button className="w-full bg-green-600 text-white px-4 py-3 rounded-lg hover:bg-green-700 transition-colors flex items-center justify-center gap-2 font-medium">
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                clipRule="evenodd"
              />
            </svg>
            Add Feed
          </button>
        </div>
      </MobileBottomSheet>
    </>
  );
}
