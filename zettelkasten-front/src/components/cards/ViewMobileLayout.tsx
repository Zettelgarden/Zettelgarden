// zettelkasten-front/src/components/cards/ViewMobileLayout.tsx
import React, { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Card,
  PartialCard,
  Entity,
  RelatedCard,
  UnlinkedMention,
} from '../../models/Card';
import { File } from '../../models/File';
import {
  CategorizedReferences,
  saveExistingCard,
  getCardAuditEvents,
  restoreCardToAuditEvent,
} from '../../api/cards';
import { SummarizeJobResponse } from '../../api/summarizer';
import { FilesTab } from '../tabs/FilesTab';
import { SummariesTab } from '../tabs/SummariesTab';
import { HistoryTab } from '../tabs/HistoryTab';
import { RollbackConfirmDialog } from '../tabs/RollbackConfirmDialog';
import { ViewMobileAccordion } from './ViewMobileAccordion';
import { ViewNavigationSheet } from './ViewNavigationSheet';
import { ViewCardContentSection } from './ViewCardContentSection';
import { SearchTagDropdown } from '../tags/SearchTagDropdown';
import { RelatedCards } from './RelatedCards';
import { UnlinkedMentions } from './UnlinkedMentions';
import { ChildrenCards } from './ChildrenCards';
import { CardList } from './CardList';
import { BacklinkInput } from './BacklinkInput';
import { SortControl as SortControlComponent } from './SortControl';
import { SortMethod, sortPartialCards } from '../../utils/cards';
import {
  TagsList,
  DetailsList,
  SourceArticleLink,
} from './SideMetadataSections';
import { PersonIcon } from '../../assets/icons/PersonIcon';
import { CardStructuredDataDisplay } from '../schemas/CardStructuredDataDisplay';
import { RSSArticle } from '../../api/rss';
import { ViewMode } from '../../pages/cards/ViewPageContainer';

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
  unlinkedMentions: UnlinkedMention[] | null;
  suggestions: RelatedCard[] | null;
  tags: any[];
  sourceArticle?: RSSArticle;
  onEditCard: () => void;
  onCreateChildCard: () => void;
  onToggleStar: () => void;
  toggleCreateTaskWindow: () => void;
  onTagClick: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  onAddBacklink: (selectedCard: PartialCard) => void;
  onUnlinkedMentionClick: (cardId: number) => void;
  onUnlinkedMentionAddLink: (mention: UnlinkedMention) => void;
  onSuggestionClick: (cardId: number) => void;
  onSuggestionAddReference: (card: RelatedCard) => void;
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
  unlinkedMentions,
  suggestions,
  tags,
  sourceArticle,
  onEditCard,
  onCreateChildCard,
  onToggleStar,
  toggleCreateTaskWindow,
  onTagClick,
  onRemoveTag,
  onAddBacklink,
  onUnlinkedMentionClick,
  onUnlinkedMentionAddLink,
  onSuggestionClick,
  onSuggestionAddReference,
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
  const [childrenSortMethod, setChildrenSortMethod] =
    useState<SortMethod>('cardId');
  const [referencesSortMethod, setReferencesSortMethod] =
    useState<SortMethod>('cardId');

  // Files + History (audit/restore) state, previously owned by the now-deleted
  // ViewCardTabbedDisplay. Mirrors the desktop rail's handling.
  const [fileFilterString, setFileFilterString] = useState<string>('');
  const [auditEvents, setAuditEvents] = useState<any[]>([]);
  const [showRollbackDialog, setShowRollbackDialog] = useState<boolean>(false);
  const [pendingRestoreEvent, setPendingRestoreEvent] = useState<any>(null);
  const [isRestoring, setIsRestoring] = useState<boolean>(false);

  const handleNavigate = (cardId: number) => {
    navigate(`/app/card/${cardId}`);
  };

  const sortedChildren = sortPartialCards(
    viewingCard.children,
    childrenSortMethod,
  );
  const sortedBidirectional = sortPartialCards(
    categorizedReferences.bidirectional,
    referencesSortMethod,
  );
  const sortedIncoming = sortPartialCards(
    categorizedReferences.incoming,
    referencesSortMethod,
  );
  const sortedOutgoing = sortPartialCards(
    categorizedReferences.outgoing,
    referencesSortMethod,
  );
  const totalReferences =
    sortedBidirectional.length + sortedIncoming.length + sortedOutgoing.length;

  const hasNavigation = parentCard || prevSibling || nextSibling;
  const hasEntities = linkedEntities && linkedEntities.length > 0;
  const hasRelatedCards = relatedCards && relatedCards.length > 0;
  const hasUnlinkedMentions = unlinkedMentions && unlinkedMentions.length > 0;
  const hasSuggestions = suggestions && suggestions.length > 0;

  // Append a file reference to the card body and save (mirrors the desktop rail).
  async function handleDisplayFileOnCardClick(file: File) {
    if (viewingCard === null) {
      return;
    }
    const editedCard = {
      ...viewingCard,
      body: viewingCard.body + '\n\n![](' + file.id + ')',
    };
    await saveExistingCard(editedCard);
    setViewCard(editedCard);
  }

  // Lazy-load audit events when the History accordion is first opened.
  function loadAuditEvents() {
    getCardAuditEvents(viewingCard.id.toString())
      .then((events) => setAuditEvents(events))
      .catch(() => setError('Failed to load audit events'));
  }

  const handleRestoreClick = (event: any) => {
    setPendingRestoreEvent(event);
    setShowRollbackDialog(true);
  };

  const handleConfirmRestore = async () => {
    if (!pendingRestoreEvent) return;
    setIsRestoring(true);
    try {
      const restoredCard = await restoreCardToAuditEvent(
        viewingCard.id.toString(),
        pendingRestoreEvent.id,
      );
      setViewCard(restoredCard);
      const events = await getCardAuditEvents(viewingCard.id.toString());
      setAuditEvents(events);
      setShowRollbackDialog(false);
      setPendingRestoreEvent(null);
    } catch (error) {
      setError('Failed to restore card');
    } finally {
      setIsRestoring(false);
    }
  };

  const handleCancelRestore = () => {
    setShowRollbackDialog(false);
    setPendingRestoreEvent(null);
  };

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
              {viewingCard.title || 'Card'}
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
                {(['normal', 'summary'] as ViewMode[]).map((mode) => (
                  <button
                    key={mode}
                    onClick={() => {
                      onViewModeChange(mode);
                      setShowMenu(false);
                    }}
                    className={`w-full px-3 py-2 text-left text-sm hover:bg-gray-50 ${
                      viewMode === mode
                        ? 'text-blue-600 font-medium'
                        : 'text-gray-700'
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
            latestSummary={latestSummary}
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

          {/* Children */}
          <ViewMobileAccordion
            title="Children"
            defaultExpanded={sortedChildren.length > 0}
            rightElement={
              <span
                role="button"
                tabIndex={0}
                onClick={onCreateChildCard}
                className="text-blue-500 hover:text-blue-700 cursor-pointer"
                title="Add child"
                aria-label="Add child"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className="h-5 w-5"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path
                    fillRule="evenodd"
                    d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                    clipRule="evenodd"
                  />
                </svg>
              </span>
            }
          >
            <div className="flex justify-end mb-2">
              <SortControlComponent
                sortMethod={childrenSortMethod}
                onSortChange={setChildrenSortMethod}
              />
            </div>
            {sortedChildren.length > 0 ? (
              <ChildrenCards allChildren={sortedChildren} card={viewingCard} />
            ) : (
              <div className="text-gray-400 text-sm">No children yet.</div>
            )}
          </ViewMobileAccordion>

          {/* Linked references */}
          <ViewMobileAccordion
            title="Linked references"
            defaultExpanded={totalReferences > 0}
          >
            <div className="flex justify-end mb-2">
              <SortControlComponent
                sortMethod={referencesSortMethod}
                onSortChange={setReferencesSortMethod}
              />
            </div>
            {sortedBidirectional.length > 0 && (
              <div className="mb-3">
                <h3 className="text-xs font-medium text-gray-600 mb-1.5">
                  Two-way Links ({sortedBidirectional.length})
                </h3>
                <CardList cards={sortedBidirectional} />
              </div>
            )}
            {sortedIncoming.length > 0 && (
              <div className="mb-3">
                <h3 className="text-xs font-medium text-gray-600 mb-1.5">
                  Incoming Links ({sortedIncoming.length})
                </h3>
                <CardList cards={sortedIncoming} />
              </div>
            )}
            {sortedOutgoing.length > 0 && (
              <div className="mb-3">
                <h3 className="text-xs font-medium text-gray-600 mb-1.5">
                  Outgoing Links ({sortedOutgoing.length})
                </h3>
                <CardList cards={sortedOutgoing} />
              </div>
            )}
            {totalReferences === 0 && (
              <div className="text-gray-400 text-sm">No references yet.</div>
            )}
            <div className="mt-4">
              <BacklinkInput
                addBacklink={onAddBacklink}
                excludeCardId={viewingCard.id}
              />
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

          {/* Unlinked Mentions */}
          {hasUnlinkedMentions && (
            <ViewMobileAccordion title="Unlinked mentions">
              <UnlinkedMentions
                mentions={unlinkedMentions!}
                onCardClick={onUnlinkedMentionClick}
                onAddLink={onUnlinkedMentionAddLink}
              />
            </ViewMobileAccordion>
          )}

          {/* Suggestions */}
          {hasSuggestions && (
            <ViewMobileAccordion title="Suggestions">
              <RelatedCards
                title="Suggestions"
                relatedCards={suggestions!}
                onCardClick={onSuggestionClick}
                onAddReference={onSuggestionAddReference}
              />
            </ViewMobileAccordion>
          )}

          {/* Source Article */}
          {sourceArticle && (
            <ViewMobileAccordion title="Source Article">
              <SourceArticleLink sourceArticle={sourceArticle} />
            </ViewMobileAccordion>
          )}

          {/* Files */}
          <ViewMobileAccordion title="Files">
            <FilesTab
              viewingCard={viewingCard}
              fileUploadRef={fileUploadRef}
              handleDisplayFileOnCardClick={handleDisplayFileOnCardClick}
              fileFilterString={fileFilterString}
              setFileFilterString={setFileFilterString}
              setError={setError}
            />
          </ViewMobileAccordion>

          {/* Summaries */}
          <ViewMobileAccordion title="Summaries">
            <SummariesTab summaries={summaries} />
          </ViewMobileAccordion>

          {/* History — lazy-loads audit events on first open. */}
          <ViewMobileAccordion
            title="History"
            onOpenChange={(open) => {
              if (open) loadAuditEvents();
            }}
          >
            <HistoryTab
              auditEvents={auditEvents}
              onRestore={handleRestoreClick}
            />
          </ViewMobileAccordion>

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

      <RollbackConfirmDialog
        isOpen={showRollbackDialog}
        onClose={handleCancelRestore}
        onConfirm={handleConfirmRestore}
        cardTitle={viewingCard.title || viewingCard.card_id || 'Untitled Card'}
        auditEvent={pendingRestoreEvent}
        isLoading={isRestoring}
      />

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
