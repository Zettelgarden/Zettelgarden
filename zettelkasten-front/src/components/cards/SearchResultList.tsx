import React from "react";
import { SearchResult, Card } from "../../models/Card";
import { CardIcon } from "../../assets/icons/CardIcon";
import { PersonIcon } from "../../assets/icons/PersonIcon";
import { Link, useNavigate } from "react-router-dom";
import { formatDate } from "../../utils/dates";
import { getFact } from "../../api/facts";
import { FactWithCard } from "../../models/Fact";
import { useState } from "react";
import { Menu } from "@headlessui/react";
import { getCard, saveExistingCard } from "../../api/cards";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { useTagContext } from "../../contexts/TagContext";

import { useShortcutContext } from "../../contexts/ShortcutContext";

interface SearchResultItemProps {
  result: SearchResult;
  showPreview: boolean;
  onEntityClick?: (entityName: string) => void;
  onTagClick?: (tagName: string) => void;
  onFactClick?: (factId: number) => void;
  onResultUpdate?: (updatedResult: SearchResult) => void;
}

function SearchResultItem({ result, showPreview, onEntityClick, onTagClick, onFactClick, onResultUpdate }: SearchResultItemProps) {
  const navigate = useNavigate();
  const { tags } = useTagContext();
  const isClassicSearch = result.type === "card" && result.score === 1.0;
  const isEntity = result.type === "entity";
  const isCard = result.type === "card";
  const isFact = result.type === "fact";
  const cardId = Number(result.metadata?.id) || 0;
  const linkedCard = isEntity ? result.metadata?.linked_card : null;

  const handleClick = (e: React.MouseEvent) => {
    if (isEntity && onEntityClick) {
      e.preventDefault();
      onEntityClick(`@[${result.title}]`);
    }
    if (isFact && onFactClick) {
      e.preventDefault();
      console.log("going to fetch", result)
      onFactClick(Number(result.id));
    }
  };

  const handleEditClick = () => {
    navigate(`/app/card/${cardId}/edit`);
  };

  const handleAddTag = async (tagName: string) => {
    if (!isCard || !cardId) return;

    try {
      // Fetch the full card data
      const fullCard = await getCard(cardId.toString());

      // Add the tag to the card body
      const editedCard: Card = {
        ...fullCard,
        body: fullCard.body + "\n\n#" + tagName,
      };

      // Save the updated card
      await saveExistingCard(editedCard);

      // Soft update the search result to show the new tag immediately
      if (onResultUpdate) {
        const newTag = { id: Date.now(), name: tagName, count: 1 }; // Create temporary tag object
        const updatedResult: SearchResult = {
          ...result,
          tags: result.tags ? [...result.tags, newTag] : [newTag]
        };
        onResultUpdate(updatedResult);
      }

      console.log(`Tag #${tagName} added to card ${cardId}`);
    } catch (error) {
      console.error("Failed to add tag to card:", error);
    }
  };

  return (
    <div className="flex items-center gap-2">
      {isCard && (
        <div className="text-gray-400">
          <CardIcon />
        </div>
      )}
      {isEntity && (
        <div className="text-gray-400">
          <PersonIcon />
        </div>
      )}
      <div className="flex-grow">
        <div className="flex flex-col">
          <div className="flex items-center flex-wrap gap-1">
            <Link
              to={isEntity ? "#" : isFact ? "#" : `/app/card/${cardId}`}
              onClick={handleClick}
              className="hover:underline flex-shrink-0"
            >
              {!isEntity && !isFact && (
                <>
                  <span className="text-blue-600 hover:text-blue-800">[{result.metadata.card_id}]</span>
                  <span className="mx-2 text-gray-400">-</span>
                </>
              )}
              {isFact ? (
                <>
                  <span className="text-green-600">[Fact]</span>
                  {result.metadata && (
                    <>
                      <span className="mx-2 text-gray-400">→</span>
                      <Link
                        to={`/app/card/${result.metadata.linked_card_pk}`}
                        className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800 hover:underline"
                      >
                        <div className="w-3 h-3 mr-1 text-gray-400">
                          <CardIcon />
                        </div>
                        [{result.metadata.linked_card_id}] {result.metadata.linked_card_title}
                      </Link>
                    </>
                  )}
                </>
              ) : (
                <span>{result.title}</span>
              )}
            </Link>
            {/* Show linked card for entities */}
            {isEntity && (
              <>
                <span className="mx-2 text-gray-400">→</span>
                <Link
                  to={`/app/card/${result.metadata.linked_card_pk}`}
                  className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800 hover:underline"
                >
                  <div className="w-3 h-3 mr-1 text-gray-400">
                    <CardIcon />
                  </div>
                  [{result.metadata.linked_card_id}] {result.metadata.linked_card_title}
                </Link>
              </>
            )}
            {/* Parse preview text for hashtags */}
            {result.preview && (
              <>
                <span className="mx-2"></span>
                {result.tags && result.tags.map((tag, index) => (
                  <span
                    key={index}
                    className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full cursor-pointer hover:bg-purple-100"
                    onClick={() => onTagClick && onTagClick(tag.name)}
                  >
                    {tag.name}
                  </span>
                ))}
              </>
            )}
          </div>
          {showPreview && result.preview && (
            <div className="mt-0.5 pl-2 text-sm italic text-gray-600">
              {result.preview.length > 200
                ? `${result.preview.substring(0, 200)}...`
                : result.preview}
            </div>
          )}
        </div>
      </div>

      {/* Menu for cards only */}
      {isCard && (
        <Menu as="div" className="relative flex-shrink-0 w-6 mr-2">
          <Menu.Button className="rounded hover:bg-gray-100 transition-colors">
            <svg
              className="w-4 h-4 text-gray-500"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
              />
            </svg>
          </Menu.Button>

          <Menu.Items className="absolute right-0 z-10 mt-1 w-32 origin-top-right bg-white border border-gray-200 rounded-md shadow-lg focus:outline-none">
            <div className="py-1">
              <Menu.Item>
                {({ active }) => (
                  <button
                    onClick={handleEditClick}
                    className={`${active ? 'bg-gray-100' : ''
                      } flex w-full items-center px-3 py-2 text-sm text-gray-700 hover:bg-gray-100`}
                  >
                    <svg
                      className="w-4 h-4 mr-2"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                      />
                    </svg>
                    Edit
                  </button>
                )}
              </Menu.Item>
              <Menu.Item>
                {({ active }) => (
                  <div className={`${active ? 'bg-gray-100' : ''} flex w-full items-center px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 border-t border-gray-100`}>
                    <svg
                      className="w-4 h-4 mr-2"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
                      />
                    </svg>
                    <span className="mr-2">Add Tag</span>
                    <SearchTagDropdown
                      tags={tags}
                      handleTagClick={handleAddTag}
                    />
                  </div>
                )}
              </Menu.Item>
            </div>
          </Menu.Items>
        </Menu>
      )}

      <div className="flex flex-col items-end text-xs text-gray-500">
        <div>{formatDate(result.created_at.toISOString())}</div>
      </div>
    </div>
  );
}

interface SearchResultListProps {
  results: SearchResult[];
  showPreview?: boolean;
  onEntityClick?: (entityName: string) => void;
  onTagClick?: (tagName: string) => void;
  onResultsUpdate?: (updatedResults: SearchResult[]) => void;
}

export function SearchResultList({
  results,
  showPreview = true,
  onEntityClick,
  onTagClick,
  onResultsUpdate,
}: SearchResultListProps) {

  const {
    showEntityDialog,
    setShowEntityDialog,
    selectedEntity,
    showFactDialog,
    setShowFactDialog,
    selectedFact,
    setSelectedFact,
  } = useShortcutContext();

  const handleFactClick = async (factId: number) => {
    const fact = await getFact(factId);
    setSelectedFact(fact);
    setShowFactDialog(true);
  };

  const handleResultUpdate = (updatedResult: SearchResult) => {
    if (onResultsUpdate) {
      const updatedResults = results.map(result =>
        result.id === updatedResult.id ? updatedResult : result
      );
      onResultsUpdate(updatedResults);
    }
  };

  return (
    <>
      <ul className="space-y-1">
        {results.map((result) => (
          <li key={result.id} className="py-1 px-2 hover:bg-gray-50 rounded-lg">
            <SearchResultItem
              result={result}
              showPreview={showPreview}
              onEntityClick={onEntityClick}
              onTagClick={onTagClick}
              onFactClick={handleFactClick}
              onResultUpdate={handleResultUpdate}
            />
          </li>
        ))}
      </ul>
    </>
  );
}
