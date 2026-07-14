// zettelkasten-front/src/components/cards/ViewMobileLayout.tsx
import React, { useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Card, PartialCard, Entity, RelatedCard } from "../../models/Card";
import { CategorizedReferences } from "../../api/cards";
import { SummarizeJobResponse } from "../../api/summarizer";
import { ViewMobileAccordion } from "./ViewMobileAccordion";
import { ViewNavigationSheet } from "./ViewNavigationSheet";
import { ViewCardContentSection } from "./ViewCardContentSection";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { RelatedCards } from "./RelatedCards";
import {
  TagsList,
  DetailsList,
  SourceArticleLink,
} from "./SideMetadataSections";
import { PersonIcon } from "../../assets/icons/PersonIcon";
import { CardStructuredDataDisplay } from "../schemas/CardStructuredDataDisplay";
import { RSSArticle } from "../../api/rss";
import { ViewMode } from "../../pages/cards/ViewPageContainer";

interface ViewMobileLayoutProps {
  viewingCard: Card;
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  linkedEntities: Entity[];
  categorizedReferences: CategorizedReferences;
  summaries: SummarizeJobResponse[];
  latestSummary: SummarizeJobResponse | null;
  relatedCards: RelatedCard[] | null;
  tags: any[];
  sourceArticle?: RSSArticle;
  onEditCard: () => void;
  onCreateChildCard: () => void;
  onToggleStar: () => void;
  toggleCreateTaskWindow: () => void;
  onTagClick: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  onAddBacklink: (selectedCard: PartialCard) => void;
  handleOpenEntity: (entity: Entity) => void;
  onResummarize: () => void;
  onRecategorize: () => void;
  refreshCard: () => void;
  setViewCard: (card: Card) => void;
  setError: (error: string) => void;
  fileUploadRef: React.RefObject<HTMLInputElement>;
  onSaveCard: (card: Card) => void;
  onMenuClick?: () => void;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
}

export function ViewMobileLayout({
  viewingCard,
  parentCard,
  prevSibling,
  nextSibling,
  linkedEntities,
  categorizedReferences,
  summaries,
  latestSummary,
  relatedCards,
  tags,
  sourceArticle,
  onEditCard,
  onCreateChildCard,
  onToggleStar,
  toggleCreateTaskWindow,
  onTagClick,
  onRemoveTag,
  onAddBacklink,
  handleOpenEntity,
  onResummarize,
  onRecategorize,
  refreshCard,
  setViewCard,
  setError,
  fileUploadRef,
  onSaveCard,
  onMenuClick,
  viewMode,
  onViewModeChange,
}: ViewMobileLayoutProps) {
  const navigate = useNavigate();
  const [showNavSheet, setShowNavSheet] = useState(false);
  const [showMenu, setShowMenu] = useState(false);

  const handleNavigate = (cardId: number) => {
    navigate(`/app/card/${cardId}`);
  };

  const hasNavigation = parentCard || prevSibling || nextSibling;
  const hasEntities = linkedEntities && linkedEntities.length > 0;
  const hasRelatedCards = relatedCards && relatedCards.length > 0;

  return (
    <div className="flex flex-col h-full overflow-hidden md:hidden">
      {/* Top Bar */}
      <div className="sticky top-0 bg-white border-b border-gray-200 z-20">
        <div className="flex items-center justify-between px-4 py-3">
          <div className="flex items-center flex-1 min-w-0">
            {onMenuClick && (
              <button
                onClick={onMenuClick}
                className="p-2 -ml-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors shrink-0"
                aria-label="Open menu"
              >
                <svg
                  className="w-6 h-6"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 6h16M4 12h16M4 18h16"
                  />
                </svg>
              </button>
            )}
            <h1 className="text-lg font-semibold text-gray-900 truncate ml-2">
              {viewingCard.title || "Card"}
            </h1>
          </div>
          <div className="relative shrink-0">
            <button
              onClick={() => setShowMenu(!showMenu)}
              className="p-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
              aria-label="More options"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
                />
              </svg>
            </button>
            {showMenu && (
              <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
                <div className="px-3 py-2 text-xs text-gray-500 font-medium">
                  View Mode
                </div>
                {(["normal", "summary"] as ViewMode[]).map(
                  (mode) => (
                    <button
                      key={mode}
                      onClick={() => {
                        onViewModeChange(mode);
                        setShowMenu(false);
                      }}
                      className={`w-full px-3 py-2 text-left text-sm hover:bg-gray-50 ${
                        viewMode === mode
                          ? "text-blue-600 font-medium"
                          : "text-gray-700"
                      }`}
                    >
                      {mode.charAt(0).toUpperCase() + mode.slice(1)}
                    </button>
                  ),
                )}
                <hr className="my-1" />
                <button
                  onClick={() => {
                    onEditCard();
                    setShowMenu(false);
                  }}
                  className="w-full px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50"
                >
                  Edit
                </button>
                <button
                  onClick={() => {
                    onToggleStar();
                    setShowMenu(false);
                  }}
                  className="w-full px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50"
                >
                  {viewingCard.is_starred ? "Unstar" : "Star"}
                </button>
                <button
                  onClick={() => {
                    setShowNavSheet(true);
                    setShowMenu(false);
                  }}
                  className="w-full px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50"
                >
                  Navigate...
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Scrollable Content */}
      <div className="flex-1 overflow-y-auto">
        {/* Main Content */}
        <div className="p-4">
          <ViewCardContentSection
            viewingCard={viewingCard}
            showingSummary={viewMode === "summary"}
            latestSummary={latestSummary}
            onCreateChildCard={onCreateChildCard}
            showRelationships
            showTabbedDisplay
            categorizedReferences={categorizedReferences}
            onAddBacklink={onAddBacklink}
            setViewCard={setViewCard}
            setError={setError}
            summaries={summaries}
            fileUploadRef={fileUploadRef}
            onSaveCard={onSaveCard}
          />
        </div>

        {/* Accordion Sections */}
        <div className="border-t border-gray-200">
          {/* Tags - expanded by default */}
          <ViewMobileAccordion
            title="Tags"
            defaultExpanded={true}
            rightElement={
              <SearchTagDropdown tags={tags} handleTagClick={onTagClick} />
            }
          >
            <TagsList card={viewingCard} onRemoveTag={onRemoveTag} />
          </ViewMobileAccordion>

          {/* Navigation */}
          {hasNavigation && (
            <ViewMobileAccordion title="Navigation">
              <div className="space-y-2">
                {parentCard && (
                  <button
                    onClick={() => handleNavigate(parentCard.id)}
                    className="w-full p-2 bg-gray-50 rounded text-left hover:bg-gray-100"
                  >
                    <div className="text-xs text-gray-500">Parent</div>
                    <div className="font-medium">{parentCard.title}</div>
                  </button>
                )}
                <div className="flex gap-2">
                  {prevSibling && (
                    <button
                      onClick={() => handleNavigate(prevSibling.id)}
                      className="flex-1 p-2 bg-gray-50 rounded hover:bg-gray-100"
                    >
                      Prev
                    </button>
                  )}
                  {nextSibling && (
                    <button
                      onClick={() => handleNavigate(nextSibling.id)}
                      className="flex-1 p-2 bg-gray-50 rounded hover:bg-gray-100"
                    >
                      Next
                    </button>
                  )}
                </div>
              </div>
            </ViewMobileAccordion>
          )}

          {/* Linked Entities */}
          {hasEntities && (
            <ViewMobileAccordion title="Linked Entities">
              <ul className="space-y-1">
                {linkedEntities.map((entity) => (
                  <li
                    key={entity.id}
                    className="py-1 px-2 hover:bg-gray-50 rounded cursor-pointer"
                    onClick={() => handleOpenEntity(entity)}
                  >
                    <div className="flex items-center gap-2 text-xs">
                      <div className="text-gray-400 shrink-0">
                        <PersonIcon />
                      </div>
                      <span className="text-blue-600">{entity.name}</span>
                      <span className="text-gray-300">-</span>
                      <span className="text-gray-500">{entity.type}</span>
                    </div>
                  </li>
                ))}
              </ul>
            </ViewMobileAccordion>
          )}

          {/* Related Cards */}
          {hasRelatedCards && (
            <ViewMobileAccordion title="Related Cards">
              <RelatedCards
                relatedCards={relatedCards!}
                onCardClick={handleNavigate}
                onAddReference={(rc) => onAddBacklink(rc.card)}
              />
            </ViewMobileAccordion>
          )}

          {/* Source Article */}
          {sourceArticle && (
            <ViewMobileAccordion title="Source Article">
              <SourceArticleLink sourceArticle={sourceArticle} />
            </ViewMobileAccordion>
          )}

          {/* Structured Data */}
          {viewingCard.schema_id && viewingCard.structured_data && (
            <ViewMobileAccordion title="Data">
              <CardStructuredDataDisplay
                schemaId={viewingCard.schema_id}
                structuredData={viewingCard.structured_data}
              />
            </ViewMobileAccordion>
          )}

          {/* Details */}
          <ViewMobileAccordion title="Details">
            <DetailsList card={viewingCard} />
          </ViewMobileAccordion>
        </div>
      </div>

      {/* Navigation Bottom Sheet */}
      <ViewNavigationSheet
        isOpen={showNavSheet}
        onClose={() => setShowNavSheet(false)}
        parentCard={parentCard}
        prevSibling={prevSibling}
        nextSibling={nextSibling}
        viewingCard={viewingCard}
        onNavigate={handleNavigate}
      />
    </div>
  );
}
