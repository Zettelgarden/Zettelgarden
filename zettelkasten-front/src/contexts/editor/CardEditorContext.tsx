import React, { createContext, useContext } from "react";
import { Card } from "../../models/Card";

interface CardEditorContextValue {
  editingCard: Card;
  setEditingCard: (card: Card | ((prevCard: Card) => Card)) => void;
}

const CardEditorContext = createContext<CardEditorContextValue | undefined>(undefined);

export function useCardEditorContext() {
  const context = useContext(CardEditorContext);
  if (!context) {
    throw new Error("useCardEditorContext must be used within CardEditorProvider");
  }
  return context;
}

interface CardEditorProviderProps {
  children: React.ReactNode;
  editingCard: Card;
  setEditingCard: (card: Card | ((prevCard: Card) => Card)) => void;
}

export function CardEditorProvider({ children, editingCard, setEditingCard }: CardEditorProviderProps) {
  return (
    <CardEditorContext.Provider value={{ editingCard, setEditingCard }}>
      {children}
    </CardEditorContext.Provider>
  );
}
