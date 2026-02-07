import React, { createContext, useContext, useState, useCallback, ReactNode } from "react";

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
