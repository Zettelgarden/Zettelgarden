import React, { useRef, useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { MobileTopBar } from "../../components/layout/MobileTopBar";
import { ViewPageHeader } from "../../components/cards/ViewPageHeader";
import { ViewCardContentSection } from "../../components/cards/ViewCardContentSection";
import { ViewPageSidePanels } from "../../components/cards/ViewPageSidePanels";
import { CardIdDiscoveryDialog } from "../../components/cards/CardIdDiscoveryDialog";
import { CardTreeView } from "../../components/cards/CardTreeView/CardTreeView";
import { ViewSummaryView } from "../../components/cards/ViewSummaryView";
import { ViewAnalysisView } from "../../components/cards/ViewAnalysisView";
import { ViewMobileLayout } from "../../components/cards/ViewMobileLayout";
import { useViewPageContainer } from "./ViewPageContainer";
import { useTagContext } from "../../contexts/TagContext";
import { useUIState } from "../../contexts/UIStateContext";
import { useDialogState } from "../../contexts/DialogStateContext";
import { defaultPartialCard } from "../../models/Card";
import { Card, PartialCard } from "../../models/Card";
import { saveExistingCard } from "../../api/cards";
import { getCard } from "../../api/cards";
import { isErrorResponse } from "../../models/common";

interface ViewPageProps {
  cardId?: string; // Optional card ID prop for pinned cards
  isPinnedView?: boolean; // Whether this ViewPage is in the pinned pane
}

export function ViewPage({ cardId, isPinnedView = false }: ViewPageProps) {
  const { tags } = useTagContext();
  const { toggleMobileSidebar, setPinnedCard } = useUIState();
  const navigate = useNavigate();

  const fileUploadRef = useRef<HTMLInputElement>(null);
  type ViewMode = 'normal' | 'tree' | 'summary' | 'analysis';
  const [viewMode, setViewMode] = useState<ViewMode>('normal');

  // Mobile detection
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window !== 'undefined') {
      return window.innerWidth < 768;
    }
    return false;
  });

  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth < 768);
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

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
  } = data;

  const {
    setViewCard,
    setError,
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

  // We need to get these from dialog state context
  const {
  } = useDialogState();

  // Handle saving card when spreadsheet is edited
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
        onOpenChatSidebar={onOpenChatSidebar}
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
            onOpenChatSidebar={onOpenChatSidebar}
            onEditCard={onEditCard}
            onToggleStar={onToggleStar}
            toggleCreateTaskWindow={toggleCreateTaskWindow}
            onResummarize={onResummarize}
            onRecategorize={onRecategorize}
            showIdDiscovery={showIdDiscovery}
            viewMode={viewMode}
            onViewModeChange={(mode) => setViewMode(mode)}
            onNavigateParent={viewingCard.parent ? () => setViewCard({
              id: viewingCard.parent!.id,
              card_id: viewingCard.parent!.card_id,
              user_id: viewingCard.parent!.user_id,
              title: viewingCard.parent!.title || "",
              body: "", // Parent data doesn't include body
              link: "", // Parent data doesn't include link
              is_deleted: false,
              created_at: viewingCard.parent!.created_at,
              updated_at: viewingCard.parent!.updated_at,
              parent_id: viewingCard.parent!.parent_id,
              parent: defaultPartialCard, // Use default for missing nested parent data
              files: [], // Parent data doesn't include files
              children: [], // We'll repopulate when full card loads
              references: [],
              tags: viewingCard.parent!.tags || [],
              tasks: [], // Parent data doesn't include tasks
              entities: [], // Parent data doesn't include entities
              is_starred: false
            }) : undefined}
          />

          <div className="flex flex-col md:flex-row gap-4">
            {/* Main Content Area - varies by view mode */}
            <div className="flex-1 md:w-2/3 space-y-4 overflow-y-auto">
              {viewMode === 'tree' && (
                <CardTreeView
                  rootCardId={viewingCard.id}
                  displayMode="full"
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
              )}

              {viewMode === 'summary' && (
                <ViewSummaryView summary={latestSummary} />
              )}

              {viewMode === 'analysis' && (
                <ViewAnalysisView analysis={analysis} />
              )}

              {viewMode === 'normal' && (
                <ViewCardContentSection
                  viewingCard={viewingCard}
                  showingSummary={false}
                  showingAnalysis={false}
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
                  // In pinned view, update the pinned card instead of navigating
                  const card = await getCard(cardId.toString());
                  if (card && !isErrorResponse(card)) {
                    setPinnedCard(card);
                  }
                } else {
                  navigate(`/app/card/${cardId}`);
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
