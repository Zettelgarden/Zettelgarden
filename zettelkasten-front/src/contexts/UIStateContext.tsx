import React, { createContext, useState, useEffect, useContext } from "react";
import { Card, PartialCard } from "../models/Card";

/**
 * UIStateContext - Consolidates simple UI state from multiple providers:
 * - ChatProvider (conversationId, showChat)
 * - PinProvider (pinnedCard, isPinMode)
 * - ChatSidebarProvider (chatSidebarCard, isChatSidebarMode)
 * - PartialCardProvider (lastCard, nextCardId)
 * - CardRefreshProvider (refreshTrigger)
 * - FileProvider (refreshFiles)
 */

interface UIStateContextType {
  // Chat state
  conversationId: string;
  setConversationId: (id: string) => void;
  showChat: boolean;
  setShowChat: (show: boolean) => void;

  // Sidebar state
  isSidebarCollapsed: boolean;
  setIsSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebarCollapsed: () => void;
  isMobileSidebarOpen: boolean;
  setIsMobileSidebarOpen: (open: boolean) => void;
  toggleMobileSidebar: () => void;

  // Pin state
  pinnedCard: Card | null;
  setPinnedCard: (card: Card | null) => void;
  isPinMode: boolean;
  setIsPinMode: (mode: boolean) => void;

  // Chat sidebar state
  chatSidebarCard: Card | null;
  setChatSidebarCard: (card: Card | null) => void;
  isChatSidebarMode: boolean;
  setIsChatSidebarMode: (mode: boolean) => void;

  // Partial card state
  lastCard: PartialCard | null;
  setLastCard: (card: PartialCard) => void;
  nextCardId: string | null;
  setNextCardId: (id: string | null) => void;

  // Card refresh state
  setRefreshTrigger: (cardId: string) => void;
  refreshTrigger: string | null;

  // File refresh state
  refreshFiles: boolean;
  setRefreshFiles: (refresh: boolean) => void;
}

const UIStateContext = createContext<UIStateContextType | undefined>(undefined);

const SIDEBAR_COLLAPSED_KEY = 'zettelgarden-sidebar-collapsed';

const getInitialSidebarState = (): boolean => {
  if (typeof window === 'undefined') return false;
  const stored = localStorage.getItem(SIDEBAR_COLLAPSED_KEY);
  return stored === 'true';
};

export const UIStateProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  // Chat state
  const [conversationId, setConversationId] = useState<string>("");
  const [showChat, setShowChat] = useState<boolean>(false);

  // Sidebar state
  const [isSidebarCollapsed, setIsSidebarCollapsedState] = useState<boolean>(getInitialSidebarState);
  const [isMobileSidebarOpen, setIsMobileSidebarOpenState] = useState<boolean>(false);

  const setIsSidebarCollapsed = (collapsed: boolean) => {
    setIsSidebarCollapsedState(collapsed);
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(collapsed));
  };

  const toggleSidebarCollapsed = () => {
    setIsSidebarCollapsedState(prev => {
      const newValue = !prev;
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(newValue));
      return newValue;
    });
  };

  const setIsMobileSidebarOpen = (open: boolean) => {
    setIsMobileSidebarOpenState(open);
  };

  const toggleMobileSidebar = () => {
    setIsMobileSidebarOpenState(prev => !prev);
  };

  // Pin state
  const [pinnedCard, setPinnedCard] = useState<Card | null>(null);
  const [isPinMode, setIsPinMode] = useState<boolean>(false);

  // Chat sidebar state
  const [chatSidebarCard, setChatSidebarCard] = useState<Card | null>(null);
  const [isChatSidebarMode, setIsChatSidebarMode] = useState<boolean>(false);

  // Partial card state
  const [lastCard, setLastCard] = useState<PartialCard | null>(null);
  const [nextCardId, setNextCardId] = useState<string | null>(null);

  // Card refresh state
  const [refreshTrigger, setRefreshTriggerState] = useState<string | null>(null);

  // File refresh state
  const [refreshFiles, setRefreshFiles] = useState(false);

  // Auto-enable pin mode when card is pinned
  useEffect(() => {
    setIsPinMode(pinnedCard !== null);
  }, [pinnedCard]);

  // Auto-enable chat sidebar mode when card is set
  useEffect(() => {
    setIsChatSidebarMode(chatSidebarCard !== null);
  }, [chatSidebarCard]);

  const setRefreshTrigger = (cardId: string) => {
    setRefreshTriggerState(cardId);
  };

  return (
    <UIStateContext.Provider
      value={{
        // Chat
        conversationId,
        setConversationId,
        showChat,
        setShowChat,

        // Sidebar
        isSidebarCollapsed,
        setIsSidebarCollapsed,
        toggleSidebarCollapsed,
        isMobileSidebarOpen,
        setIsMobileSidebarOpen,
        toggleMobileSidebar,

        // Pin
        pinnedCard,
        setPinnedCard,
        isPinMode,
        setIsPinMode,

        // Chat sidebar
        chatSidebarCard,
        setChatSidebarCard,
        isChatSidebarMode,
        setIsChatSidebarMode,

        // Partial card
        lastCard,
        setLastCard,
        nextCardId,
        setNextCardId,

        // Card refresh
        setRefreshTrigger,
        refreshTrigger,

        // File refresh
        refreshFiles,
        setRefreshFiles,
      }}
    >
      {children}
    </UIStateContext.Provider>
  );
};

export const useUIState = () => {
  const context = useContext(UIStateContext);
  if (context === undefined) {
    throw new Error("useUIState must be used within a UIStateProvider");
  }
  return context;
};

// Backwards compatibility exports - these can be removed after migrating all consumers
export const usePinContext = useUIState;
export const useChatSidebarContext = useUIState;
export const useChatContext = useUIState;
export const useCardRefresh = useUIState;
export const useFileContext = useUIState;
