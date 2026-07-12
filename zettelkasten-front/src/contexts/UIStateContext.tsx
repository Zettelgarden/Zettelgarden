import React, { createContext, useState, useContext } from "react";
import { PartialCard } from "../models/Card";

/**
 * UIStateContext - Consolidates simple UI state from multiple providers:
 * - PartialCardProvider (lastCard, nextCardId)
 * - CardRefreshProvider (refreshTrigger)
 * - FileProvider (refreshFiles)
 */

/** Which tab is active in the card view's right info rail. */
export type RightPaneTab = "links" | "metadata" | "entities";

interface UIStateContextType {
  // Sidebar state
  isSidebarCollapsed: boolean;
  setIsSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebarCollapsed: () => void;
  isMobileSidebarOpen: boolean;
  setIsMobileSidebarOpen: (open: boolean) => void;
  toggleMobileSidebar: () => void;

  // Right pane (card view info rail) state
  rightPaneOpen: boolean;
  setRightPaneOpen: (open: boolean) => void;
  toggleRightPane: () => void;
  rightPaneTab: RightPaneTab;
  setRightPaneTab: (tab: RightPaneTab) => void;

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

const SIDEBAR_COLLAPSED_KEY = "zettelgarden-sidebar-collapsed";
const RIGHT_PANE_OPEN_KEY = "zettelgarden-right-pane-open";

const getInitialSidebarState = (): boolean => {
  if (typeof window === "undefined") return false;
  const stored = localStorage.getItem(SIDEBAR_COLLAPSED_KEY);
  return stored === "true";
};

const getInitialRightPaneState = (): boolean => {
  if (typeof window === "undefined") return true;
  const stored = localStorage.getItem(RIGHT_PANE_OPEN_KEY);
  // Defaults to open (true) unless explicitly closed.
  return stored !== "false";
};

export const UIStateProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  // Sidebar state
  const [isSidebarCollapsed, setIsSidebarCollapsedState] = useState<boolean>(
    getInitialSidebarState,
  );
  const [isMobileSidebarOpen, setIsMobileSidebarOpenState] =
    useState<boolean>(false);

  // Right pane state
  const [rightPaneOpen, setRightPaneOpenState] = useState<boolean>(
    getInitialRightPaneState,
  );
  const [rightPaneTab, setRightPaneTab] = useState<RightPaneTab>("links");

  const setIsSidebarCollapsed = (collapsed: boolean) => {
    setIsSidebarCollapsedState(collapsed);
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(collapsed));
  };

  const toggleSidebarCollapsed = () => {
    setIsSidebarCollapsedState((prev) => {
      const newValue = !prev;
      localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(newValue));
      return newValue;
    });
  };

  const setIsMobileSidebarOpen = (open: boolean) => {
    setIsMobileSidebarOpenState(open);
  };

  const toggleMobileSidebar = () => {
    setIsMobileSidebarOpenState((prev) => !prev);
  };

  const setRightPaneOpen = (open: boolean) => {
    setRightPaneOpenState(open);
    localStorage.setItem(RIGHT_PANE_OPEN_KEY, String(open));
  };

  const toggleRightPane = () => {
    setRightPaneOpenState((prev) => {
      const newValue = !prev;
      localStorage.setItem(RIGHT_PANE_OPEN_KEY, String(newValue));
      return newValue;
    });
  };

  // Partial card state
  const [lastCard, setLastCard] = useState<PartialCard | null>(null);
  const [nextCardId, setNextCardId] = useState<string | null>(null);

  // Card refresh state
  const [refreshTrigger, setRefreshTriggerState] = useState<string | null>(
    null,
  );

  // File refresh state
  const [refreshFiles, setRefreshFiles] = useState(false);

  const setRefreshTrigger = (cardId: string) => {
    setRefreshTriggerState(cardId);
  };

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

        // Right pane
        rightPaneOpen,
        setRightPaneOpen,
        toggleRightPane,
        rightPaneTab,
        setRightPaneTab,

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
export const useCardRefresh = useUIState;
export const useFileContext = useUIState;
