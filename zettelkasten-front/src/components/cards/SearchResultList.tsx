import React from "react";
import { SearchResult } from "../../models/Card";
import { CardIcon } from "../../assets/icons/CardIcon";
import { PersonIcon } from "../../assets/icons/PersonIcon";
import { Link } from "react-router-dom";
import { formatDate } from "../../utils/dates";
import { getFact } from "../../api/facts";
import { FactWithCard } from "../../models/Fact";
import { useState } from "react";

import { useShortcutContext } from "../../contexts/ShortcutContext";

interface SearchResultItemProps {
  result: SearchResult;
  showPreview: boolean;
  onEntityClick?: (entityName: string) => void;
  onTagClick?: (tagName: string) => void;
  onFactClick?: (factId: number) => void;
}

function SearchResultItem({ result, showPreview, onEntityClick, onTagClick, onFactClick }: SearchResultItemProps) {
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
      <div className="flex flex-col items-end text-xs text-gray-500">
        <div>{formatDate(result.created_at.toISOString())}</div>
        {!isClassicSearch && (
          <div className="mt-1">
            <span className="bg-gray-100 rounded px-2 py-0.5">
              {(result.score * 100).toFixed(0)}%
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

interface SearchResultListProps {
  results: SearchResult[];
  showPreview?: boolean;
  onEntityClick?: (entityName: string) => void;
  onTagClick?: (tagName: string) => void;
}

export function SearchResultList({
  results,
  showPreview = true,
  onEntityClick,
  onTagClick,
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
            />
          </li>
        ))}
      </ul>
    </>
  );
}
