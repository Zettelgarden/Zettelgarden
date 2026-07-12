import React from "react";
import { Menu } from "@headlessui/react";
import { Card, PartialCard } from "../../models/Card";
import { HeaderSubSection } from "../Header";
import { BacklinkInputDropdownList } from "./BacklinkInputDropdownList";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { useNavigate } from "react-router-dom";
import { getNextRootId } from "../../api/cards";

interface CardMetadataProps {
  newCard: boolean;
  originalCard: Card;
  editingCard: Card;
  setEditingCard: (card: Card) => void;
  setShowCardIdDiscovery: (show: boolean) => void;
  handleClickFillCard: () => void;
  tags: any[];
  handleTagClick: (tagName: string) => void;
  handleRemoveTag: (tagName: string) => void;
  addBacklink: (selectedCard: PartialCard) => void;
  /** Which rail tab to render. Defaults to "metadata". */
  tab?: "metadata" | "links";
}

export function CardMetadata({
  newCard,
  originalCard,
  editingCard,
  setEditingCard,
  setShowCardIdDiscovery,
  handleClickFillCard,
  tags,
  handleTagClick,
  handleRemoveTag,
  addBacklink,
  tab = "metadata",
}: CardMetadataProps) {
  const navigate = useNavigate();

  return (
    <div className="bg-white rounded-lg p-4 shadow-sm">
      {tab === "links" ? (
        // Links tab: backlink input for appending [[id|*|]] references to the body.
        <>
          <HeaderSubSection text="References" />
          <BacklinkInputDropdownList
            onSelect={addBacklink}
            onSearch={() => { }}
            placeholder="Add Backlink"
            excludeCardId={editingCard.id}
          />
        </>
      ) : (
        // Metadata tab: Card ID, Tags, Source/Link, Details.
        <>
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <HeaderSubSection text="Card ID" />
              <Menu as="div" className="relative inline-block">
                <Menu.Button className="text-gray-400 hover:text-gray-600 p-2 min-w-[44px] min-h-[44px] flex items-center justify-center">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                  </svg>
                </Menu.Button>
                <Menu.Items className="absolute left-0 z-10 mt-1 w-80 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none p-4">
                  <div className="text-sm text-gray-700 space-y-2">
                    <p className="font-medium">Card IDs are unique identifiers that support hierarchical organization.</p>
                    <div>
                      <p className="font-medium mt-2">Examples:</p>
                      <ul className="list-disc ml-4 mt-1 space-y-1">
                        <li><code className="text-xs bg-gray-100 px-1 rounded">1</code> - root card</li>
                        <li><code className="text-xs bg-gray-100 px-1 rounded">1.1</code> - child of 1</li>
                        <li><code className="text-xs bg-gray-100 px-1 rounded">1.1.2</code> - child of 1.1</li>
                      </ul>
                    </div>
                    <p className="text-xs text-gray-500 mt-2">
                      We recommend using numbers for IDs. The + button assigns the next available number. Use search and tags to find cards.
                    </p>
                  </div>
                </Menu.Items>
              </Menu>
            </div>
            <div className="flex items-center gap-3">
              <div className="flex-1 relative">
                <input
                  type="text"
                  id="card_id"
                  value={editingCard.card_id}
                  onChange={(e) =>
                    setEditingCard({ ...editingCard, card_id: e.target.value })
                  }
                  placeholder="ID"
                  className="block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm pr-20"
                />
                {(newCard || (!newCard && editingCard.card_id === "")) && (
                  <div className="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
                    <button
                      onClick={async () => {
                        try {
                          const response = await getNextRootId();
                          setEditingCard({ ...editingCard, card_id: response.new_id });
                        } catch (error) {
                          console.error('Failed to get next root ID:', error);
                          // Silently fail - user can manually enter ID
                        }
                      }}
                      className="p-2 min-w-[44px] min-h-[44px] text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded flex items-center justify-center"
                      type="button"
                      title="Use next available root ID"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
                      </svg>
                    </button>
                    <button
                      onClick={() => setShowCardIdDiscovery(true)}
                      className="p-2 min-w-[44px] min-h-[44px] text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded flex items-center justify-center"
                      type="button"
                      title="Discover card ID from hierarchy"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clipRule="evenodd" />
                      </svg>
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>

          <hr className="my-4" />

          <div className="flex items-center justify-between">
            <HeaderSubSection text="Tags" />
            <SearchTagDropdown
              tags={tags}
              handleTagClick={handleTagClick}
            />
          </div>
          <div className="flex flex-wrap gap-1.5 mt-2">
            {editingCard.tags.map((tag) => (
              <span
                key={tag.name}
                className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
              >
                <span
                  className="cursor-pointer hover:bg-purple-100"
                  onClick={() => navigate(`/app/search?term=${encodeURIComponent('#' + tag.name)}`)}
                >
                  #{tag.name}
                </span>
                {editingCard.body.includes(`#${tag.name}`) && (
                  <button
                    onClick={() => handleRemoveTag(tag.name)}
                    className="ml-1.5 text-purple-400 hover:text-purple-600"
                  >
                    &times;
                  </button>
                )}
              </span>
            ))}
          </div>

          <hr className="my-4" />

          {/* Source/Link section (moved from the main column). */}
          <div className="space-y-2">
            <HeaderSubSection text="Link" />
            <div className="relative">
              <input
                type="text"
                id="link"
                value={editingCard.link}
                onChange={(e) =>
                  setEditingCard({ ...editingCard, link: e.target.value })
                }
                placeholder="Source"
                className="block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm pr-10"
              />
              <button
                onClick={handleClickFillCard}
                className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded"
                type="button"
                title="Fill card from URL"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </button>
            </div>
          </div>

          {/* Details Section */}
          {!newCard && (
            <div className="text-xs text-gray-600 space-y-1 pt-4 border-t mt-4">
              <div className="flex items-start">
                <span className="font-medium w-20">Created:</span>
                <span className="flex-1">{originalCard.created_at.toISOString()}</span>
              </div>
              <div className="flex items-start">
                <span className="font-medium w-20">Updated:</span>
                <span className="flex-1">{originalCard.updated_at.toISOString()}</span>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
