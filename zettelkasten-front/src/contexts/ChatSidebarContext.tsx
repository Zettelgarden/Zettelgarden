import React, { createContext, useState, useEffect, useContext } from "react";
import { Card } from "../models/Card";

interface ChatSidebarContextType {
  chatSidebarCard: Card | null;
  setChatSidebarCard: (card: Card | null) => void;
  isChatSidebarMode: boolean;
  setIsChatSidebarMode: (mode: boolean) => void;
}

const ChatSidebarContext = createContext<ChatSidebarContextType | undefined>(undefined);

export const ChatSidebarProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [chatSidebarCard, setChatSidebarCard] = useState<Card | null>(null);
  const [isChatSidebarMode, setIsChatSidebarMode] = useState<boolean>(false);

  // Auto-enable chat sidebar mode when card is set
  useEffect(() => {
    setIsChatSidebarMode(chatSidebarCard !== null);
  }, [chatSidebarCard]);

  return (
    <ChatSidebarContext.Provider
      value={{
        chatSidebarCard,
        setChatSidebarCard,
        isChatSidebarMode,
        setIsChatSidebarMode,
      }}
    >
      {children}
    </ChatSidebarContext.Provider>
  );
};

export const useChatSidebarContext = () => {
  const context = useContext(ChatSidebarContext);
  if (context === undefined) {
    throw new Error("useChatSidebarContext must be used within a ChatSidebarProvider");
  }
  return context;
};