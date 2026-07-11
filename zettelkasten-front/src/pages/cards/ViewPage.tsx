import React, { useRef } from "react";
import { useNavigate } from "react-router-dom";
import { MobileTopBar } from "../../components/layout/MobileTopBar";
import { ViewPageHeader } from "../../components/cards/ViewPageHeader";
import { ViewCardContentSection } from "../../components/cards/ViewCardContentSection";
import { ViewPageSidePanels } from "../../components/cards/ViewPageSidePanels";
import { CardIdDiscoveryDialog } from "../../components/cards/CardIdDiscoveryDialog";
import { ViewSummaryView } from "../../components/cards/ViewSummaryView";
import { ViewAnalysisView } from "../../components/cards/ViewAnalysisView";
import { ViewMobileLayout } from "../../components/cards/ViewMobileLayout";
import { useViewPageContainer } from "./ViewPageContainer";
import { useTagContext } from "../../contexts/TagContext";
import { useUIState } from "../../contexts/UIStateContext";
import { Card } from "../../models/Card";
import { saveExistingCard, getCard } from "../../api/cards";
import { isErrorResponse } from "../../models/common";
import { useIsMobile } from "../../hooks/useIsMobile";
import { buildCardFromParent } from "../../utils/cards";

interface ViewPageProps {
  cardId?: string; // Optional card ID prop for pinned cards
  isPinnedView?: boolean; // Whether this ViewPage is in the pinned pane
}

export function ViewPage({ cardId, isPinnedView = false }: ViewPageProps) {
  const { tags } = useTagContext();
  const { toggleMobileSidebar, setPinnedCard } = useUIState();
  const navigate = useNavigate();
  const isMobile = useIsMobile();

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
    relatedCards,
    showIdDiscovery,
    isPinned,
    error,
    viewMode,
  } = data;

  const {
    setViewCard,
    setError,
    setViewMode,
  } = setters;

  const {
    onEditCard,
    onCreateChildCard,
    onToggleStar,
    onTogglePin,
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

  // Handle saving card
  const handleSaveCard = async (updatedCard: Card) => {
    try {
      await saveExistingCard(updatedCard);
      // Update the viewingCard state with the saved data
      setViewCard(updatedCard);
    } catch (err) {
      console.error("Failed to save card:", err);
      setError("Failed to save card");
    }
  };

  // Mobile layout
  if (isMobile && viewingCard) {
    return (
      <ViewMobileLayout
        viewingCard={viewingCard}
        parentCard={parentCard}
        prevSibling={prevSibling}
        nextSibling={nextSibling}
        linkedEntities={linkedEntities}
        categorizedReferences={categorizedReferences}
        summaries={summaries || []}
        latestSummary={latestSummary}
        analysis={analysis}
        relatedCards={relatedCards}
        tags={tags}
        sourceArticle={viewingCard.source_article}
        onEditCard={onEditCard}
        onCreateChildCard={onCreateChildCard}
        onToggleStar={onToggleStar}
        onTogglePin={onTogglePin}
        toggleCreateTaskWindow={toggleCreateTaskWindow}
        onTagClick={onTagClick}
        onRemoveTag={onRemoveTag}
        onAddBacklink={onAddBacklink}
        handleOpenEntity={handleOpenEntity}
        onResummarize={onResummarize}
        onRecategorize={onRecategorize}
        refreshCard={refreshCard}
        setViewCard={setViewCard}
        setError={setError}
        fileUploadRef={fileUploadRef}
        onSaveCard={handleSaveCard}
        onMenuClick={toggleMobileSidebar}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
      />
    );
  }

  // Desktop layout
  return (
    <div className="overflow-x-hidden">
      {viewingCard && (
        <MobileTopBar
          title={viewingCard.title || "Card"}
          onMenuClick={toggleMobileSidebar}
        />
      )}
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
            onEditCard={onEditCard}
            onToggleStar={onToggleStar}
            toggleCreateTaskWindow={toggleCreateTaskWindow}
            onResummarize={onResummarize}
            onRecategorize={onRecategorize}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
            onNavigateParent={viewingCard.parent ? () => setViewCard(
              buildCardFromParent(viewingCard.parent)
            ) : undefined}
          />

          <div className="flex flex-col md:flex-row gap-4">
            {/* Main Content Area - varies by view mode */}
            <div className="flex-1 md:w-2/3 space-y-4 overflow-y-auto">
              {viewMode === 'summary' && (
                <ViewSummaryView summary={latestSummary} />
              )}

              {viewMode === 'analysis' && (
                <ViewAnalysisView analysis={analysis} />
              )}

              {viewMode === 'normal' && (
                <ViewCardContentSection
                  viewingCard={viewingCard}
                  latestSummary={latestSummary}
                  analysis={analysis}
                  onCreateChildCard={onCreateChildCard}
                  categorizedReferences={categorizedReferences}
                  onAddBacklink={onAddBacklink}
                  setViewCard={setViewCard}
                  setError={setError}
                  handleOpenEntity={handleOpenEntity}
                  summaries={summaries}
                  fileUploadRef={fileUploadRef}
                  onSaveCard={handleSaveCard}
                />
              )}
            </div>

            {/* Side Panels - always visible */}
            <ViewPageSidePanels
              parentCard={parentCard}
              prevSibling={prevSibling}
              nextSibling={nextSibling}
              linkedEntities={linkedEntities}
              onOpenEntity={handleOpenEntity}
              viewingCard={viewingCard}
              tags={tags}
              onTagClick={onTagClick}
              onRemoveTag={onRemoveTag}
              sourceArticle={viewingCard.source_article}
              relatedCards={relatedCards || undefined}
              onRelatedCardClick={async (cardId) => {
                if (isPinnedView) {
                  const card = await getCard(cardId.toString());
                  if (card && !isErrorResponse(card)) {
                    setPinnedCard(card);
                  }
                } else {
                  navigate(`/app/card/${cardId}`);
                }
              }}
              onRelatedCardAddReference={(rc) => {
                if (viewingCard) {
                  onAddBacklink(rc.card);
                }
              }}
            />
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
