import React, { useState } from "react";
import { Card } from "../../models/Card";
import { ViewPage } from "../../pages/cards/ViewPage";
import { useIsDesktop } from "../../hooks/useWindowSize";
import { PinErrorBoundary } from "../ErrorBoundary";
import { useUIState } from "../../contexts/UIStateContext";
import { PinIcon } from "../../assets/icons/PinIcon";

interface SplitViewLayoutProps {
  pinnedCard: Card;
  children: React.ReactNode;
}

export const SplitViewLayout: React.FC<SplitViewLayoutProps> = ({
  pinnedCard,
  children
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const isDesktop = useIsDesktop(1024);
  const { setPinnedCard } = useUIState();

  const handleUnpin = () => {
    setPinnedCard(null);
  };

  const handlePinError = () => {
    // Clear the pinned card on error to gracefully degrade to single-pane view
    setPinnedCard(null);
  };

  return (
    <div className="flex flex-col lg:flex-row h-full">
      {/* Main Content Pane - Left side on desktop, top on mobile */}
      <div className={`
        w-full lg:w-1/2
        border-b lg:border-b-0 lg:border-r border-gray-200
        overflow-y-auto
        ${isExpanded ? 'h-1/3 md:h-1/2 lg:h-full' : 'flex-1 lg:h-full'}
        transition-all duration-300 ease-in-out
      `}>
        <div className="h-full">
          {children}
        </div>
      </div>

      {/* Pinned Card Pane - Right side on desktop, collapsible bottom on mobile */}
      <div className={`
        w-full lg:w-1/2
        overflow-y-auto
        ${isExpanded ? 'h-2/3 md:h-1/2' : 'h-auto lg:h-full'}
        transition-all duration-300 ease-in-out
      `}>
        <div className="h-full bg-blue-50 flex flex-col">
          {/* Desktop Header */}
          <div className="hidden lg:flex bg-blue-100 px-3 py-2 border-b border-blue-200 items-center justify-between">
            <div className="flex items-center gap-2">
              <PinIcon className="h-4 w-4 text-blue-600" filled={true} />
              <span className="text-xs font-semibold uppercase tracking-wide text-blue-700">
                Pinned Card
              </span>
              <span className="text-blue-600 text-sm">
                [{pinnedCard.card_id}]
              </span>
              <span className="text-blue-700 text-sm font-medium truncate max-w-md">
                {pinnedCard.title}
              </span>
            </div>
            <button
              onClick={handleUnpin}
              className="text-blue-600 hover:text-blue-800 hover:bg-blue-200 px-2 py-1 rounded text-sm flex items-center gap-1"
              title="Unpin card"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
              Unpin
            </button>
          </div>

          {/* Mobile collapse/expand button */}
          <div className="lg:hidden bg-blue-100 p-2 border-b border-blue-200">
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="flex items-center justify-between w-full text-blue-700"
            >
              <div className="flex items-center gap-2">
                <PinIcon className="h-3 w-3 text-blue-600" filled={true} />
                <span className="text-xs font-medium uppercase tracking-wide">
                  Pinned: [{pinnedCard.card_id}]
                </span>
                <span className="text-sm font-medium truncate">
                  {pinnedCard.title}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={(e) => { e.stopPropagation(); handleUnpin(); }}
                  className="text-blue-600 hover:text-blue-800 p-1"
                  title="Unpin card"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
                <svg
                  className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </button>
          </div>

          {/* ViewPage content - conditionally shown on mobile */}
          {(isExpanded || isDesktop) && (
            <div className="flex-1 overflow-y-auto">
              <PinErrorBoundary onPinError={handlePinError}>
                <ViewPage cardId={pinnedCard.id.toString()} isPinnedView={true} />
              </PinErrorBoundary>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};