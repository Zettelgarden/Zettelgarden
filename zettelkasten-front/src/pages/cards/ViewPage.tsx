import React, { useRef } from "react";
import { ViewPageHeader } from "../../components/cards/ViewPageHeader";
import { ViewCardContentSection } from "../../components/cards/ViewCardContentSection";
import { ViewPageSidePanels } from "../../components/cards/ViewPageSidePanels";
import { CardIdDiscoveryDialog } from "../../components/cards/CardIdDiscoveryDialog";
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
          />

          <div className="">
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
