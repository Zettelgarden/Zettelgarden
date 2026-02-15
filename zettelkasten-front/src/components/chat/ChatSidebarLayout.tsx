import React from "react";
import { Card } from "../../models/Card";
import { ChatSidebar } from "./ChatSidebar";
import { useUIState } from "../../contexts/UIStateContext";
import { SidePanelLayout } from "../layout/SidePanelLayout";

interface ChatSidebarLayoutProps {
  chatSidebarCard: Card;
  children: React.ReactNode;
}

export const ChatSidebarLayout: React.FC<ChatSidebarLayoutProps> = ({
  chatSidebarCard,
  children
}) => {
  const { setChatSidebarCard } = useUIState();

  const handleClose = () => {
    setChatSidebarCard(null);
  };

  return (
    <SidePanelLayout
      theme="green"
      title="Chat"
      subtitle={`[${chatSidebarCard.card_id}] ${chatSidebarCard.title}`}
      onClose={handleClose}
      panelContent={<ChatSidebar card={chatSidebarCard} />}
    >
      {children}
    </SidePanelLayout>
  );
};
