import React from "react";
import { Card } from "../../models/Card";
import { ViewPage } from "../../pages/cards/ViewPage";
import { PinErrorBoundary } from "../ErrorBoundary";
import { useUIState } from "../../contexts/UIStateContext";
import { PinIcon } from "../../assets/icons/PinIcon";
import { SidePanelLayout } from "../layout/SidePanelLayout";

interface SplitViewLayoutProps {
  pinnedCard: Card;
  children: React.ReactNode;
}

export const SplitViewLayout: React.FC<SplitViewLayoutProps> = ({
  pinnedCard,
  children
}) => {
  const { setPinnedCard } = useUIState();

  const handleClose = () => {
    setPinnedCard(null);
  };

  const handleError = () => {
    // Clear the pinned card on error to gracefully degrade to single-pane view
    setPinnedCard(null);
  };

  return (
    <SidePanelLayout
      theme="blue"
      title="Pinned Card"
      subtitle={`[${pinnedCard.card_id}] ${pinnedCard.title}`}
      onClose={handleClose}
      panelContent={
        <PinErrorBoundary onPinError={handleError}>
          <ViewPage cardId={pinnedCard.id.toString()} isPinnedView={true} />
        </PinErrorBoundary>
      }
    >
      {children}
    </SidePanelLayout>
  );
};
