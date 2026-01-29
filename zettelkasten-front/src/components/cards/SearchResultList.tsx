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
import { CardListMenu } from "./CardListMenu";

import { useDialogState } from "../../contexts/DialogStateContext";

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
        const newTag = {
          id: Date.now(),
          name: tagName,
          color: "",
          user_id: 0
        }; // Create temporary tag object with all required properties
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

  const handleRemoveTag = async (tagName: string) => {
    if (!isCard || !cardId) return;

    try {
      // Fetch the full card data
      const fullCard = await getCard(cardId.toString());

      // Remove the tag from the card body using regex
      const tagRegex = new RegExp(`\\n*#${tagName}\\b`, 'g');
      const editedCard: Card = {
        ...fullCard,
        body: fullCard.body.replace(tagRegex, ''),
      };

      // Save the updated card
      await saveExistingCard(editedCard);

      // Soft update the search result to remove the tag immediately
      if (onResultUpdate) {
        const updatedResult: SearchResult = {
          ...result,
          tags: result.tags ? result.tags.filter(tag => tag.name !== tagName) : []
        };
        onResultUpdate(updatedResult);
      }

      console.log(`Tag #${tagName} removed from card ${cardId}`);
    } catch (error) {
      console.error("Failed to remove tag from card:", error);
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
                    className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
                  >
                    <span
                      className="cursor-pointer hover:bg-purple-100"
                      onClick={() => onTagClick && onTagClick(tag.name)}
                    >
                      #{tag.name}
                    </span>
                    {isCard && (
                      <button
                        onClick={() => handleRemoveTag(tag.name)}
                        className="ml-1.5 text-purple-400 hover:text-purple-600"
                      >
                        &times;
                      </button>
                    )}
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
        <div className="mr-2">
          <CardListMenu
            cardId={cardId}
            onEditClick={handleEditClick}
            onAddTag={handleAddTag}
            tags={tags}
          />
        </div>
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
  } = useDialogState();

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
