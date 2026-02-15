import React, { createContext, useState, useEffect, useContext } from "react";
import { Card, PartialCard } from "../models/Card";

/**
 * UIStateContext - Consolidates simple UI state from multiple providers:
 * - ChatProvider (conversationId, showChat)
 * - PinProvider (pinnedCard) - isPinMode is derived: pinnedCard !== null
 * - ChatSidebarProvider (chatSidebarCard, isChatSidebarMode)
 * - ChatPanel (isChatOpen) - for sidebar chat without card context
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
  isPinMode: boolean; // Derived: pinnedCard !== null

  // Chat sidebar state (with card context)
  chatSidebarCard: Card | null;
  setChatSidebarCard: (card: Card | null) => void;
  isChatSidebarMode: boolean; // Derived: chatSidebarCard !== null

  // Chat panel state (without card context - from sidebar button)
  isChatOpen: boolean;
  setIsChatOpen: (open: boolean) => void;
  toggleChatOpen: () => void;

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
const PINNED_CARD_KEY = 'zettelgarden-pinned-card';
const CHAT_SIDEBAR_CARD_KEY = 'zettelgarden-chat-sidebar-card';
const CHAT_OPEN_KEY = 'zettelgarden-chat-open';

const getInitialSidebarState = (): boolean => {
  if (typeof window === 'undefined') return false;
  const stored = localStorage.getItem(SIDEBAR_COLLAPSED_KEY);
  return stored === 'true';
};

const getStoredCard = (key: string): Card | null => {
  if (typeof window === 'undefined') return null;
  try {
    const stored = localStorage.getItem(key);
    return stored ? JSON.parse(stored) : null;
  } catch {
    return null;
  }
};

const getInitialChatOpenState = (): boolean => {
  if (typeof window === 'undefined') return false;
  const stored = localStorage.getItem(CHAT_OPEN_KEY);
  return stored === 'true';
};

const getInitialSidebarState = (): boolean => {
  if (typeof window === 'undefined') return false;
  const stored = localStorage.getItem(SIDEBAR_COLLAPSED_KEY);
  return stored === 'true';
};

const getStoredCard = (key: string): Card | null => {
  if (typeof window === 'undefined') return null;
  try {
    const stored = localStorage.getItem(key);
    return stored ? JSON.parse(stored) : null;
  } catch {
    return null;
  }
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
  const [pinnedCard, setPinnedCard] = useState<Card | null>(() => getStoredCard(PINNED_CARD_KEY));

  // Chat sidebar state
  const [chatSidebarCard, setChatSidebarCard] = useState<Card | null>(() => getStoredCard(CHAT_SIDEBAR_CARD_KEY));

  // Chat panel state (without card context)
  const [isChatOpen, setIsChatOpenState] = useState<boolean>(getInitialChatOpenState);

  const setIsChatOpen = (open: boolean) => {
    setIsChatOpenState(open);
    if (typeof window !== 'undefined') {
      localStorage.setItem(CHAT_OPEN_KEY, String(open));
    }
  };

  const toggleChatOpen = () => {
    setIsChatOpenState(prev => {
      const newValue = !prev;
      if (typeof window !== 'undefined') {
        localStorage.setItem(CHAT_OPEN_KEY, String(newValue));
      }
      return newValue;
    });
  };

  // Partial card state
  const [lastCard, setLastCard] = useState<PartialCard | null>(null);
  const [nextCardId, setNextCardId] = useState<string | null>(null);

  // Card refresh state
  const [refreshTrigger, setRefreshTriggerState] = useState<string | null>(null);

  // File refresh state
  const [refreshFiles, setRefreshFiles] = useState(false);

  const setRefreshTrigger = (cardId: string) => {
    setRefreshTriggerState(cardId);
  };

  // Sync pinned card changes to localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      if (pinnedCard) {
        localStorage.setItem(PINNED_CARD_KEY, JSON.stringify(pinnedCard));
      } else {
        localStorage.removeItem(PINNED_CARD_KEY);
      }
    }
  }, [pinnedCard]);

  // Sync chat sidebar card changes to localStorage
  useEffect(() => {
    if (typeof window !== 'undefined') {
      if (chatSidebarCard) {
        localStorage.setItem(CHAT_SIDEBAR_CARD_KEY, JSON.stringify(chatSidebarCard));
      } else {
        localStorage.removeItem(CHAT_SIDEBAR_CARD_KEY);
      }
    }
  }, [chatSidebarCard]);

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
        isPinMode: pinnedCard !== null,

        // Chat sidebar
        chatSidebarCard,
        setChatSidebarCard,
        isChatSidebarMode: chatSidebarCard !== null,

        // Chat panel (without card)
        isChatOpen,
        setIsChatOpen,
        toggleChatOpen,

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
