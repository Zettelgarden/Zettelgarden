import React, { useState } from "react";
import { Card } from "../../models/Card";
import { ChatSidebar } from "./ChatSidebar";
import { useIsDesktop } from "../../hooks/useWindowSize";
import { useUIState } from "../../contexts/UIStateContext";

interface ChatSidebarLayoutProps {
  chatSidebarCard: Card;
  children: React.ReactNode;
}

export const ChatSidebarLayout: React.FC<ChatSidebarLayoutProps> = ({
  chatSidebarCard,
  children
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const isDesktop = useIsDesktop(1024);
  const { setChatSidebarCard } = useUIState();

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

      {/* Chat Sidebar Pane - Right side on desktop, collapsible bottom on mobile */}
      <div className={`
        w-full lg:w-1/2
        ${isExpanded ? 'h-2/3 md:h-1/2' : 'h-auto lg:h-full'}
        transition-all duration-300 ease-in-out
      `}>
        <div className="h-full bg-green-50 flex flex-col">
          {/* Mobile collapse/expand button */}
          <div className="lg:hidden bg-green-100 p-2 border-b border-green-200 flex-shrink-0">
            <button
              onClick={() => setIsExpanded(!isExpanded)}
              className="flex items-center justify-between w-full text-green-700"
            >
              <div className="flex items-center gap-2">
                <span className="text-xs font-medium uppercase tracking-wide">
                  Chat: [{chatSidebarCard.card_id}]
                </span>
                <span className="text-sm font-medium truncate">
                  {chatSidebarCard.title}
                </span>
              </div>
              <svg
                className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </button>
          </div>

          {/* ChatSidebar content - conditionally shown on mobile */}
          {(isExpanded || isDesktop) && (
            <div className="flex-1 min-h-0">
              <ChatSidebar card={chatSidebarCard} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};