import React, { createContext, useContext, useState, useCallback, useEffect, ReactNode } from "react";
import { getUnreadCounts } from "../api/rss";

interface RSSContextType {
  unreadCount: number;
  setUnreadCount: (count: number) => void;
}

const RSSContext = createContext<RSSContextType | undefined>(undefined);

export function RSSProvider({ children }: { children: ReactNode }) {
  const [unreadCount, setUnreadCount] = useState(0);

  const setUnreadCountCallback = useCallback((count: number) => {
    setUnreadCount(count);
  }, []);

  // Fetch unread count on mount so it's available before visiting RSS page
  useEffect(() => {
    getUnreadCounts()
      .then((counts) => {
        const total = Object.values(counts.feeds).reduce((sum, count) => sum + count, 0);
        setUnreadCount(total);
      })
      .catch(() => {
        // Silently fail - count will update when RSS page is visited
      });
  }, []);

  return (
    <RSSContext.Provider value={{ unreadCount, setUnreadCount: setUnreadCountCallback }}>
      {children}
    </RSSContext.Provider>
  );
}

export function useRSS() {
  const context = useContext(RSSContext);
  if (context === undefined) {
    throw new Error("useRSS must be used within an RSSProvider");
  }
  return context;
}
