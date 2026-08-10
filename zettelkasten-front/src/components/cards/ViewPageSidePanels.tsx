import React, { useState } from 'react';
import {
  Card,
  PartialCard,
  RelatedCard,
  UnlinkedMention,
} from '../../models/Card';
import { Entity } from '../../models/Card';
import { HeaderSubSection } from '../Header';
import { SearchTagDropdown } from '../tags/SearchTagDropdown';
import { EntitiesTab } from '../tabs/EntitiesTab';
import { FilesTab } from '../tabs/FilesTab';
import { HistoryTab } from '../tabs/HistoryTab';
import { RollbackConfirmDialog } from '../tabs/RollbackConfirmDialog';
import { removeEntityFromCard } from '../../api/entities';
import {
  saveExistingCard,
  getCardAuditEvents,
  restoreCardToAuditEvent,
} from '../../api/cards';
import { File } from '../../models/File';
import { CardStructuredDataDisplay } from '../schemas/CardStructuredDataDisplay';
import { RSSArticle } from '../../api/rss';
import { CategorizedReferences } from '../../api/cards';
import { RelatedCards } from './RelatedCards';
import { UnlinkedMentions } from './UnlinkedMentions';
import { ChildrenCards } from './ChildrenCards';
import { CardList } from './CardList';
import { BacklinkInput } from './BacklinkInput';
import { Collapsible } from '../Collapsible';
import { SortMethod, sortPartialCards } from '../../utils/cards';
import { SortControl as SortControlComponent } from './SortControl';
import { useUIState, RightPaneTab } from '../../contexts/UIStateContext';
import { useRightPaneTab } from '../../hooks/useRightPaneTab';
import {
  TagsList,
  DetailsList,
  SourceArticleLink,
} from './SideMetadataSections';

interface ViewPageSidePanelsProps {
  onOpenEntity: (entity: Entity) => void;
  viewingCard: Card;
  tags: any[];
  onTagClick: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  sourceArticle?: RSSArticle;
  relatedCards?: RelatedCard[];
  onRelatedCardClick?: (cardId: number) => void;
  onRelatedCardAddReference?: (card: RelatedCard) => void;
  unlinkedMentions?: UnlinkedMention[];
  onUnlinkedMentionClick?: (cardId: number) => void;
  onUnlinkedMentionAddLink?: (mention: UnlinkedMention) => void;
  onCreateChildCard: () => void;
  categorizedReferences: CategorizedReferences;
  onAddBacklink: (selectedCard: PartialCard) => void;
  setViewCard: (card: Card) => void;
  setError: (error: string) => void;
  fileUploadRef: React.RefObject<HTMLInputElement>;
}

const TABS: { id: RightPaneTab; label: string }[] = [
  { id: 'links', label: 'Links' },
  { id: 'metadata', label: 'Metadata' },
  { id: 'entities', label: 'Entities' },
  { id: 'files', label: 'Files' },
];

export function ViewPageSidePanels({
  onOpenEntity,
  viewingCard,
  tags,
  onTagClick,
  onRemoveTag,
  sourceArticle,
  relatedCards,
  onRelatedCardClick,
  onRelatedCardAddReference,
  unlinkedMentions,
  onUnlinkedMentionClick,
  onUnlinkedMentionAddLink,
  onCreateChildCard,
  categorizedReferences,
  onAddBacklink,
  setViewCard,
  setError,
  fileUploadRef,
}: ViewPageSidePanelsProps) {
  const { toggleRightPane, rightPaneTab, setRightPaneTab } = useUIState();

  const [childrenSortMethod, setChildrenSortMethod] =
    useState<SortMethod>('cardId');
  const [referencesSortMethod, setReferencesSortMethod] =
    useState<SortMethod>('cardId');
  const [entityFilterString, setEntityFilterString] = useState<string>('');
  const [showAddEntityDialog, setShowAddEntityDialog] =
    useState<boolean>(false);
  const [fileFilterString, setFileFilterString] = useState<string>('');
  const [auditEvents, setAuditEvents] = useState<any[]>([]);
  const [showRollbackDialog, setShowRollbackDialog] = useState<boolean>(false);
  const [pendingRestoreEvent, setPendingRestoreEvent] = useState<any>(null);
  const [isRestoring, setIsRestoring] = useState<boolean>(false);

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

  // Sync the active rail tab with ?pane=, picking a smart default on mount.
  useRightPaneTab({
    hasRelationships: sortedChildren.length > 0 || totalReferences > 0,
  });

  async function handleRemoveEntity(entityId: number) {
    try {
      await removeEntityFromCard(entityId, viewingCard.id);
      setViewCard({
        ...viewingCard,
        entities:
          viewingCard.entities?.filter((entity) => entity.id !== entityId) ||
          [],
      });
    } catch (error) {
      setError('Failed to remove entity from card');
    }
  }

  function handleEntityAdded(entity: Entity) {
    setViewCard({
      ...viewingCard,
      entities: [...(viewingCard.entities || []), entity],
    });
  }

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

  // Lazy-load audit events whenever the History collapsible is opened.
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
    <div className="w-full">
      {/* Tab strip + close affordance */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex flex-wrap">
          {TABS.map((tab) => (
            <span
              key={tab.id}
              onClick={() => setRightPaneTab(tab.id)}
              className={`
                cursor-pointer font-medium py-1 px-2 flex items-center text-sm
                ${
                  rightPaneTab === tab.id
                    ? 'text-blue-600 border-b-2 border-blue-600'
                    : 'text-gray-600 hover:text-gray-800 hover:bg-gray-100 rounded-md'
                }
              `}
            >
              {tab.label}
            </span>
          ))}
        </div>
        <button
          type="button"
          onClick={toggleRightPane}
          title="Close info pane"
          aria-label="Close info pane"
          className="p-1 rounded-md text-gray-400 hover:text-gray-700 hover:bg-gray-100 transition-colors"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>

      <div className="space-y-6">
        {rightPaneTab === 'links' && (
          <>
            {/* Children */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-3">
                  <HeaderSubSection text="Children" />
                  <SortControlComponent
                    sortMethod={childrenSortMethod}
                    onSortChange={setChildrenSortMethod}
                  />
                </div>
                <button
                  onClick={onCreateChildCard}
                  className="text-blue-500 hover:text-blue-700"
                  title="Add child"
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
                </button>
              </div>
              {sortedChildren.length > 0 ? (
                <ChildrenCards
                  allChildren={sortedChildren}
                  card={viewingCard}
                />
              ) : (
                <div className="text-gray-400 text-sm">No children yet.</div>
              )}
            </div>

            {/* Linked references */}
            <Collapsible
              title="Linked references"
              count={totalReferences}
              defaultOpen={totalReferences > 0}
              rightElement={
                <SortControlComponent
                  sortMethod={referencesSortMethod}
                  onSortChange={setReferencesSortMethod}
                />
              }
            >
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
            </Collapsible>

            {/* Related cards */}
            {relatedCards && onRelatedCardClick && (
              <RelatedCards
                relatedCards={relatedCards}
                onCardClick={onRelatedCardClick}
                onAddReference={onRelatedCardAddReference}
              />
            )}

            {/* Unlinked mentions */}
            {onUnlinkedMentionClick && onUnlinkedMentionAddLink && (
              <Collapsible
                title="Unlinked mentions"
                count={unlinkedMentions?.length ?? 0}
                defaultOpen={(unlinkedMentions?.length ?? 0) > 0}
              >
                <UnlinkedMentions
                  mentions={unlinkedMentions || []}
                  onCardClick={onUnlinkedMentionClick}
                  onAddLink={onUnlinkedMentionAddLink}
                />
              </Collapsible>
            )}
          </>
        )}

        {rightPaneTab === 'metadata' && (
          <>
            {/* Source Article Section */}
            {sourceArticle && (
              <div>
                <HeaderSubSection text="Source Article" />
                <div className="mt-2">
                  <SourceArticleLink sourceArticle={sourceArticle} />
                </div>
              </div>
            )}

            {/* Related cards live in the Links tab now. */}

            <div className="py-1">
              <CardStructuredDataDisplay
                schemaId={viewingCard.schema_id}
                structuredData={viewingCard.structured_data}
              />
            </div>

            {/* Tags Section */}
            <div>
              <div className="flex items-center justify-between">
                <HeaderSubSection text="Tags" />
                <SearchTagDropdown tags={tags} handleTagClick={onTagClick} />
              </div>
              <TagsList
                card={viewingCard}
                onRemoveTag={onRemoveTag}
                className="mt-2"
              />
            </div>

            {/* Details Section */}
            <DetailsList card={viewingCard} className="pt-4 border-t" />

            {/* History — collapsed by default; loads on expand. */}
            <Collapsible
              title="History"
              count={auditEvents.length}
              defaultOpen={false}
              onOpenChange={(open) => {
                if (open) loadAuditEvents();
              }}
            >
              <HistoryTab
                auditEvents={auditEvents}
                onRestore={handleRestoreClick}
              />
            </Collapsible>
          </>
        )}

        {rightPaneTab === 'entities' && (
          <EntitiesTab
            viewingCard={viewingCard}
            entityFilterString={entityFilterString}
            setEntityFilterString={setEntityFilterString}
            showAddEntityDialog={showAddEntityDialog}
            setShowAddEntityDialog={setShowAddEntityDialog}
            handleOpenEntity={onOpenEntity}
            handleRemoveEntity={handleRemoveEntity}
            handleEntityAdded={handleEntityAdded}
            setError={setError}
          />
        )}

        {rightPaneTab === 'files' && (
          <FilesTab
            viewingCard={viewingCard}
            fileUploadRef={fileUploadRef}
            handleDisplayFileOnCardClick={handleDisplayFileOnCardClick}
            fileFilterString={fileFilterString}
            setFileFilterString={setFileFilterString}
            setError={setError}
          />
        )}

        <RollbackConfirmDialog
          isOpen={showRollbackDialog}
          onClose={handleCancelRestore}
          onConfirm={handleConfirmRestore}
          cardTitle={
            viewingCard.title || viewingCard.card_id || 'Untitled Card'
          }
          auditEvent={pendingRestoreEvent}
          isLoading={isRestoring}
        />
      </div>
    </div>
  );
}
