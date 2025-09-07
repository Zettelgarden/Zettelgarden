import React, { createContext, useState, useEffect, useContext } from "react";
import { PartialCard, Card } from "../models/Card";



interface PartialCardContextType {
  lastCard: PartialCard | null;
  setLastCard: (card: PartialCard) => void;
  nextCardId: string | null;
  setNextCardId: (id: string | null) => void;
}
const PartialCardContext = createContext<PartialCardContextType | undefined>(
  undefined,
);

export const PartialCardProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [lastCard, setLastCard] = useState<PartialCard | null>(null);
  const [nextCardId, setNextCardId] = useState<string | null>(null);
  useState<boolean>(false);

  return (
    <PartialCardContext.Provider
      value={{
        lastCard,
        setLastCard,
        nextCardId,
        setNextCardId,
      }}
    >
      {children}
    </PartialCardContext.Provider>
  );
};

export const usePartialCardContext = () => {
  const context = useContext(PartialCardContext);
  if (context === undefined) {
    throw new Error(
      "usePartialCardContext must be used wtihin a PartialCardProvider",
    );
  }
  return context;
};
