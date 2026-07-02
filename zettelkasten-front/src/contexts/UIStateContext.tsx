import React, { createContext, useState, useEffect, useContext } from "react";
import { Card, PartialCard } from "../models/Card";

/**
 * UIStateContext - Consolidates simple UI state from multiple providers:
 * - PinProvider (pinnedCard) - isPinMode is derived: pinnedCard !== null
 * - PartialCardProvider (lastCard, nextCardId)
 * - CardRefreshProvider (refreshTrigger)
 * - FileProvider (refreshFiles)
 */

interface UIStateContextType {
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

  return (
    <UIStateContext.Provider
      value={{
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
export const useCardRefresh = useUIState;
export const useFileContext = useUIState;
