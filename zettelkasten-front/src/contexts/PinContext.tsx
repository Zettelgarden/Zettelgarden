import React, { createContext, useState, useEffect, useContext } from "react";
import { Card } from "../models/Card";

interface PinContextType {
  pinnedCard: Card | null;
  setPinnedCard: (card: Card | null) => void;
  isPinMode: boolean;
  setIsPinMode: (mode: boolean) => void;
}

const PinContext = createContext<PinContextType | undefined>(undefined);

export const PinProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [pinnedCard, setPinnedCard] = useState<Card | null>(null);
  const [isPinMode, setIsPinMode] = useState<boolean>(false);

  // Auto-enable pin mode when card is pinned
  useEffect(() => {
    setIsPinMode(pinnedCard !== null);
  }, [pinnedCard]);

  return (
    <PinContext.Provider
      value={{
        pinnedCard,
        setPinnedCard,
        isPinMode,
        setIsPinMode,
      }}
    >
      {children}
    </PinContext.Provider>
  );
};

export const usePinContext = () => {
  const context = useContext(PinContext);
  if (context === undefined) {
    throw new Error("usePinContext must be used within a PinProvider");
  }
  return context;
};