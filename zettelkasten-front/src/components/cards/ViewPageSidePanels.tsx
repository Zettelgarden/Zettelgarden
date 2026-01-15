import React from "react";
import { useNavigate } from "react-router-dom";
import { Card, PartialCard } from "../../models/Card";
import { Entity } from "../../models/Card";
import { SectionAnalysis, SummarizeJobResponse } from "../../api/summarizer";
import { HeaderSubSection } from "../Header";
import { Button } from "../Button";
import { CardItem } from "./CardItem";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { linkifyWithDefaultOptions } from "../../utils/strings";

interface ViewPageSidePanelsProps {
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  linkedEntities: Entity[];
  onOpenEntity: (entity: Entity) => void;
  showingSummary: boolean;
  showingAnalysis: boolean;
  latestSummary: SummarizeJobResponse | null;
  analysis: SectionAnalysis[] | null;
  setShowingSummary: (showing: boolean) => void;
  setShowingAnalysis: (showing: boolean) => void;
  viewingCard: Card;
  tags: any[];
  onTagClick: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
}

export function ViewPageSidePanels({
  parentCard,
  prevSibling,
  nextSibling,
  linkedEntities,
  onOpenEntity,
  showingSummary,
  showingAnalysis,
  latestSummary,
  analysis,
  setShowingSummary,
  setShowingAnalysis,
  viewingCard,
  tags,
  onTagClick,
  onRemoveTag
}: ViewPageSidePanelsProps) {
  const navigate = useNavigate();

  return (
    <div className="md:w-1/3 bg-white rounded-lg p-4 shadow-sm space-y-4">
      {/* Parent Card Section */}
      {parentCard && (
        <div>
          <HeaderSubSection text="Parent" />
          <div className="mt-2">
            <CardItem card={parentCard} />
          </div>

          {prevSibling && (
            <Button
              onClick={() => navigate(`/app/card/${prevSibling.id}`)}
              variant="secondary"
            >
              ← Prev
            </Button>
          )}
          {parentCard && (
            <Button
              onClick={() => navigate(`/app/card/${parentCard.id}`)}
              variant="secondary"
            >
              ↑ Up
            </Button>
          )}
          {nextSibling && (
            <Button
              onClick={() => navigate(`/app/card/${nextSibling.id}`)}
              variant="secondary"
            >
              Next →
            </Button>
          )}
          <hr className="my-4" />
        </div>
      )}

      {/* Linked Entities Section */}
      {linkedEntities && linkedEntities.length > 0 && (
        <div>
          <HeaderSubSection text="Linked Entities" />
          <div className="mt-2 space-y-2">
            {linkedEntities.map(entity => (
              <div
                key={entity.id}
                className="text-xs text-blue-600 cursor-pointer hover:underline"
                onClick={() => onOpenEntity(entity)}
              >
                {entity.name}
              </div>
            ))}
          </div>
          <hr className="my-4" />
        </div>
      )}

      {/* Card Views Section */}
      {(latestSummary || (analysis && analysis.length > 0)) && (
        <div>
          <HeaderSubSection text="Card Views" />
          <div className="">
            <button
              onClick={() => {
                setShowingSummary(false);
                setShowingAnalysis(false);
              }}
              className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${!showingSummary && !showingAnalysis
                ? 'bg-blue-100 text-blue-800 font-medium'
                : 'text-gray-600 hover:bg-gray-100'
                }`}
            >
              📄 Show Card
            </button>
            {latestSummary && (
              <button
                onClick={() => {
                  setShowingSummary(!showingSummary);
                  if (showingAnalysis) setShowingAnalysis(false);
                }}
                className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${showingSummary
                  ? 'bg-yellow-100 text-yellow-800 font-medium'
                  : 'text-gray-600 hover:bg-gray-100'
                  }`}
              >
                📝 Show Summary
              </button>
            )}
            {analysis && analysis.length > 0 && (
              <button
                onClick={() => {
                  setShowingAnalysis(!showingAnalysis);
                  if (showingSummary) setShowingSummary(false);
                }}
                className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${showingAnalysis
                  ? 'bg-blue-100 text-blue-800 font-medium'
                  : 'text-gray-600 hover:bg-gray-100'
                  }`}
              >
                🔍 Show Analysis
              </button>
            )}
          </div>
          <hr className="my-4" />
        </div>
      )}

      {/* Tags Section */}
      <div>
        <div className="flex items-center justify-between">
          <HeaderSubSection text="Tags" />
          <SearchTagDropdown
            tags={tags}
            handleTagClick={onTagClick}
          />
        </div>
        <div className="flex flex-wrap gap-1.5 mt-2">
          {viewingCard.tags.map((tag) => (
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
              {viewingCard.body.includes(`#${tag.name}`) && (
                <button
                  onClick={() => onRemoveTag(tag.name)}
                  className="ml-1.5 text-purple-400 hover:text-purple-600"
                >
                  &times;
                </button>
              )}
            </span>
          ))}
        </div>
        <hr className="my-4" />
      </div>

      {/* Details Section */}
      <div className="text-xs text-gray-600 space-y-1 pt-4 border-t">
        {viewingCard.link && (
          <div className="flex items-start">
            <span className="font-medium w-20">Link:</span>
            <div
              className="flex-1 break-all"
              dangerouslySetInnerHTML={{ __html: linkifyWithDefaultOptions(viewingCard.link) }}
            />
          </div>
        )}
        <div className="flex items-start">
          <span className="font-medium w-20">Created:</span>
          <span className="flex-1">{viewingCard.created_at.toISOString()}</span>
        </div>
        <div className="flex items-start">
          <span className="font-medium w-20">Updated:</span>
          <span className="flex-1">{viewingCard.updated_at.toISOString()}</span>
        </div>
      </div>
    </div>
  );
}