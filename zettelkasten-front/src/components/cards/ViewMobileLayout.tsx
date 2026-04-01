// zettelkasten-front/src/components/cards/ViewMobileLayout.tsx
import React, { useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Card, PartialCard, Entity, RelatedCard } from "../../models/Card";
import { CategorizedReferences } from "../../api/cards";
import { SummarizeJobResponse, SectionAnalysis } from "../../api/summarizer";
import { ViewMobileAccordion } from "./ViewMobileAccordion";
import { ViewNavigationSheet } from "./ViewNavigationSheet";
import { ViewCardContentSection } from "./ViewCardContentSection";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { RelatedCards } from "./RelatedCards";
import { linkifyWithDefaultOptions } from "../../utils/strings";
import { PersonIcon } from "../../assets/icons/PersonIcon";
import { CardStructuredDataDisplay } from "../schemas/CardStructuredDataDisplay";
import { RSSArticle } from "../../api/rss";

interface ViewMobileLayoutProps {
  viewingCard: Card;
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  linkedEntities: Entity[];
  categorizedReferences: CategorizedReferences;
  summaries: SummarizeJobResponse[];
  latestSummary: SummarizeJobResponse | null;
  analysis: SectionAnalysis[] | null;
  relatedCards: RelatedCard[] | null;
  tags: any[];
  sourceArticle?: RSSArticle;
  onEditCard: () => void;
  onCreateChildCard: () => void;
  onToggleStar: () => void;
  onTogglePin: () => void;
  onOpenChatSidebar: () => void;
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
  setSelectedFact: (fact: any) => void;
  setShowFactDialog: (show: boolean) => void;
  fileUploadRef: React.RefObject<HTMLInputElement>;
  onSaveCard: (card: Card) => void;
  onMenuClick?: () => void;
}

type ViewMode = 'normal' | 'tree' | 'summary' | 'analysis';

export function ViewMobileLayout({
  viewingCard,
  parentCard,
  prevSibling,
  nextSibling,
  linkedEntities,
  categorizedReferences,
  summaries,
  latestSummary,
  analysis,
  relatedCards,
  tags,
  sourceArticle,
  onEditCard,
  onCreateChildCard,
  onToggleStar,
  onTogglePin,
  onOpenChatSidebar,
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
  setSelectedFact,
  setShowFactDialog,
  fileUploadRef,
  onSaveCard,
  onMenuClick,
}: ViewMobileLayoutProps) {
  const navigate = useNavigate();
  const [showNavSheet, setShowNavSheet] = useState(false);
  const [showMenu, setShowMenu] = useState(false);
  const [viewMode, setViewMode] = useState<ViewMode>('normal');

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
                <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
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
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
              </svg>
            </button>
            {showMenu && (
              <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
                <div className="px-3 py-2 text-xs text-gray-500 font-medium">View Mode</div>
                {(['normal', 'tree', 'summary', 'analysis'] as ViewMode[]).map((mode) => (
                  <button
                    key={mode}
                    onClick={() => {
                      setViewMode(mode);
                      setShowMenu(false);
                    }}
                    className={`w-full px-3 py-2 text-left text-sm hover:bg-gray-50 ${
                      viewMode === mode ? 'text-blue-600 font-medium' : 'text-gray-700'
                    }`}
                  >
                    {mode.charAt(0).toUpperCase() + mode.slice(1)}
                  </button>
                ))}
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
                  {viewingCard.is_starred ? 'Unstar' : 'Star'}
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
            showingSummary={viewMode === 'summary'}
            showingAnalysis={viewMode === 'analysis'}
            latestSummary={latestSummary}
            analysis={analysis}
            onCreateChildCard={onCreateChildCard}
            categorizedReferences={categorizedReferences}
            onAddBacklink={onAddBacklink}
            setViewCard={setViewCard}
            setError={setError}
            handleOpenEntity={handleOpenEntity}
            summaries={summaries}
            setSelectedFact={setSelectedFact}
            setShowFactDialog={setShowFactDialog}
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
              <SearchTagDropdown
                tags={tags}
                handleTagClick={onTagClick}
              />
            }
          >
            <div className="flex flex-wrap gap-1.5">
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
                      x
                    </button>
                  )}
                </span>
              ))}
            </div>
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
                {linkedEntities.map(entity => (
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
              />
            </ViewMobileAccordion>
          )}

          {/* Source Article */}
          {sourceArticle && (
            <ViewMobileAccordion title="Source Article">
              <button
                onClick={() => navigate('/app/rss', { state: { selectedArticleId: sourceArticle.id } })}
                className="w-full text-left p-2 rounded hover:bg-gray-50"
              >
                <div className="flex items-start gap-2">
                  <svg className="w-4 h-4 text-green-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                  </svg>
                  <div>
                    <p className="text-sm font-medium text-blue-600">{sourceArticle.title}</p>
                    <p className="text-xs text-gray-500 mt-1">
                      RSS Feed - {new Date(sourceArticle.fetched_at).toLocaleDateString()}
                    </p>
                  </div>
                </div>
              </button>
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
            <div className="text-xs text-gray-600 space-y-1">
              <div className="flex items-start">
                <span className="font-medium w-20">ID:</span>
                <span className="flex-1 text-blue-600 font-mono">[{viewingCard.card_id}]</span>
              </div>
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
                <span className="flex-1">{viewingCard.created_at.toLocaleDateString()}</span>
              </div>
              <div className="flex items-start">
                <span className="font-medium w-20">Updated:</span>
                <span className="flex-1">{viewingCard.updated_at.toLocaleDateString()}</span>
              </div>
            </div>
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
