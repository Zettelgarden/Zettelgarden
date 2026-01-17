import React, { useRef, useState } from "react";
import { ViewPageHeader } from "../../components/cards/ViewPageHeader";
import { ViewCardContentSection } from "../../components/cards/ViewCardContentSection";
import { ViewPageSidePanels } from "../../components/cards/ViewPageSidePanels";
import { CardIdDiscoveryDialog } from "../../components/cards/CardIdDiscoveryDialog";
import { CardTreeView } from "../../components/cards/CardTreeView/CardTreeView";
import { useViewPageContainer } from "./ViewPageContainer";
import { useTagContext } from "../../contexts/TagContext";
import { usePinContext } from "../../contexts/PinContext";
import { useShortcutContext } from "../../contexts/ShortcutContext";
import { Card, PartialCard } from "../../models/Card";
import { saveExistingCard } from "../../api/cards";

interface ViewPageProps {
  cardId?: string; // Optional card ID prop for pinned cards
}

export function ViewPage({ cardId }: ViewPageProps) {
  const { tags } = useTagContext();
  const { pinnedCard } = usePinContext();

  const fileUploadRef = useRef<HTMLInputElement>(null);
  const [viewMode, setViewMode] = useState<'normal' | 'tree'>('normal');

  const {
    data,
    setters,
    actions
  } = useViewPageContainer({ cardId });

  const {
    viewingCard,
    parentCard,
    prevSibling,
    nextSibling,
    linkedEntities,
    categorizedReferences,
    summaries,
    latestSummary,
    analysis,
    showingSummary,
    showingAnalysis,
    showIdDiscovery,
    error,
  } = data;

  const {
    setViewCard,
    setError,
    setShowingSummary,
    setShowingAnalysis,
  } = setters;

  const {
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
    onCloseIdDiscovery,
    refreshCard,
  } = actions;

  // We need to get these from shortcut context
  const {
    setSelectedFact,
    setShowFactDialog,
  } = useShortcutContext();


  const isPinned = pinnedCard && viewingCard && pinnedCard.id === viewingCard.id;

  return (
    <div className="overflow-x-hidden">
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-4">
          <div className="text-red-700">{error}</div>
        </div>
      )}
      {viewingCard && (
        <div className="space-y-6">
          <ViewPageHeader
            viewingCard={viewingCard}
            isPinned={!!isPinned}
            onTogglePin={onTogglePin}
            onOpenChatSidebar={onOpenChatSidebar}
            onEditCard={onEditCard}
            onToggleStar={onToggleStar}
            toggleCreateTaskWindow={toggleCreateTaskWindow}
            onResummarize={onResummarize}
            onRecategorize={onRecategorize}
            showIdDiscovery={showIdDiscovery}
            viewMode={viewMode}
            onToggleViewMode={() => setViewMode(viewMode === 'normal' ? 'tree' : 'normal')}
          />

          <div className="">
            {viewMode === 'tree' ? (
              <CardTreeView
                rootCardId={viewingCard.id}
                displayMode="compact"
                height="600px"
                onCardSelect={(selectedCard) => {
                  // Navigate to the selected card in tree view
                  // Convert ProcessedCardWithDescendants to Card format
                  setViewCard({
                    id: selectedCard.id,
                    card_id: selectedCard.card_id,
                    user_id: selectedCard.user_id,
                    title: selectedCard.title || "",
                    body: selectedCard.body || "",
                    link: selectedCard.link || "",
                    is_deleted: false,
                    created_at: selectedCard.created_at,
                    updated_at: selectedCard.updated_at,
                    parent_id: selectedCard.parent_id,
                    parent: viewingCard.parent, // Reuse existing parent data
                    files: [], // Tree data doesn't include files
                    children: selectedCard.descendants?.map(d => ({
                      id: d.id,
                      card_id: d.card_id,
                      user_id: d.user_id,
                      title: d.title || "",
                      parent_id: d.parent_id,
                      created_at: d.created_at,
                      updated_at: d.updated_at,
                      tags: [] // Tree data doesn't include tags
                    })) || [],
                    references: [],
                    tags: [],
                    tasks: [], // Tree data doesn't include tasks
                    entities: [], // Tree data doesn't include entities
                    is_starred: false
                  });
                  // Note: We could trigger a fetch of the full card data here if needed
                  refreshCard();
                }}
              />
            ) : (
              <div className="flex flex-col md:flex-row gap-4">
                <ViewCardContentSection
                  viewingCard={viewingCard}
                  showingSummary={showingSummary}
                  showingAnalysis={showingAnalysis}
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
                />

                <ViewPageSidePanels
                  parentCard={parentCard}
                  prevSibling={prevSibling}
                  nextSibling={nextSibling}
                  linkedEntities={linkedEntities}
                  onOpenEntity={handleOpenEntity}
                  showingSummary={showingSummary}
                  showingAnalysis={showingAnalysis}
                  latestSummary={latestSummary}
                  analysis={analysis}
                  setShowingSummary={setShowingSummary}
                  setShowingAnalysis={setShowingAnalysis}
                  viewingCard={viewingCard}
                  tags={tags}
                  onTagClick={onTagClick}
                  onRemoveTag={onRemoveTag}
                />
              </div>
            )}
          </div>
        </div>
      )}
      {showIdDiscovery && (
        <CardIdDiscoveryDialog
          onClose={onCloseIdDiscovery}
          onSelectId={(cardId) => {
            if (viewingCard) {
              const updatedCard = { ...viewingCard, card_id: cardId };
              saveExistingCard(updatedCard).then(() => {
                // Trigger a refresh to update the card data
                refreshCard();
              });
              onCloseIdDiscovery();
            }
          }}
        />
      )}
    </div>
  );
}
