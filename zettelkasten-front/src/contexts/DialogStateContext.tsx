import React, { createContext, ReactNode, useContext, useState } from "react";
import { Entity } from "../models/Card";
import { Fact, FactWithCard } from "../models/Fact";
import { Task } from "../models/Task";

/**
 * DialogStateContext - Manages UI state for keyboard shortcut-triggered dialogs.
 * Renamed from ShortcutProvider for clarity - it specifically handles dialog state,
 * not keyboard shortcut registration.
 */

interface ChildrenProviderProps {
  children: ReactNode;
}

interface DialogStateContextType {
  showCreateTaskWindow: boolean;
  setShowCreateTaskWindow: (show: boolean) => void;
  showQuickSearchWindow: boolean;
  setShowQuickSearchWindow: (show: boolean) => void;
  showEntityDialog: boolean;
  setShowEntityDialog: (show: boolean) => void;
  showFactDialog: boolean;
  setShowFactDialog: (show: boolean) => void;
  showTaskDialog: boolean;
  setShowTaskDialog: (show: boolean) => void;
  selectedEntity: Entity | null;
  setSelectedEntity: (entity: Entity | null) => void;
  selectedFact: FactWithCard | null;
  setSelectedFact: (fact: FactWithCard | null) => void;
  selectedTaskId: number | null;
  setSelectedTaskId: (taskId: number | null) => void;
}

const DialogStateContext = createContext<DialogStateContextType>({
  showCreateTaskWindow: false,
  setShowCreateTaskWindow: () => { },
  showQuickSearchWindow: false,
  setShowQuickSearchWindow: () => { },
  showEntityDialog: false,
  setShowEntityDialog: () => { },
  showFactDialog: false,
  setShowFactDialog: () => { },
  showTaskDialog: false,
  setShowTaskDialog: () => { },
  selectedEntity: null,
  setSelectedEntity: (entity: Entity | null) => { },
  selectedFact: null,
  setSelectedFact: (fact: FactWithCard | null) => { },
  selectedTaskId: null,
  setSelectedTaskId: (taskId: number | null) => { },
});

export const DialogStateProvider = ({ children }: ChildrenProviderProps) => {
  const [showCreateTaskWindow, setShowCreateTaskWindow] = useState(false);
  const [showQuickSearchWindow, setShowQuickSearchWindow] = useState(false);
  const [selectedEntity, setSelectedEntity] = useState<Entity | null>(null);
  const [showEntityDialog, setShowEntityDialog] = useState(false);
  const [selectedFact, setSelectedFact] = useState<FactWithCard | null>(null);
  const [showFactDialog, setShowFactDialog] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<number | null>(null);
  const [showTaskDialog, setShowTaskDialog] = useState(false);

  return (
    <DialogStateContext.Provider
      value={{
        showCreateTaskWindow,
        setShowCreateTaskWindow,
        showQuickSearchWindow,
        setShowQuickSearchWindow,
        showEntityDialog,
        setShowEntityDialog,
        showFactDialog,
        setShowFactDialog,
        showTaskDialog,
        setShowTaskDialog,
        selectedEntity,
        setSelectedEntity,
        selectedFact,
        setSelectedFact,
        selectedTaskId,
        setSelectedTaskId,
      }}
    >
      {children}
    </DialogStateContext.Provider>
  );
};

export const useDialogState = () => {
  const context = useContext(DialogStateContext);
  if (context === undefined) {
    throw new Error("useDialogState must be used within a DialogStateProvider");
  }
  return context;
};

// Backwards compatibility export - can be removed after migrating all consumers
export const useShortcutContext = useDialogState;
