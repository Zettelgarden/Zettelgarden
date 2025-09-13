import React, { createContext, ReactNode, useContext, useState } from "react";
import { Entity } from "../models/Card";
import { Fact, FactWithCard } from "../models/Fact";
import { Task } from "../models/Task";

interface ChildrenProviderProps {
  children: ReactNode;
}

interface ShortcutProviderType {
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
  selectedTask: Task | null;
  setSelectedTask: (task: Task | null) => void;
}

const ShortcutContext = createContext<ShortcutProviderType>({
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
  selectedTask: null,
  setSelectedTask: (task: Task | null) => { },
});

export const ShortcutProvider = ({ children }: ChildrenProviderProps) => {
  const [showCreateTaskWindow, setShowCreateTaskWindow] = useState(false);
  const [showQuickSearchWindow, setShowQuickSearchWindow] = useState(false);
  const [selectedEntity, setSelectedEntity] = useState<Entity | null>(null);
  const [showEntityDialog, setShowEntityDialog] = useState(false);
  const [selectedFact, setSelectedFact] = useState<FactWithCard | null>(null);
  const [showFactDialog, setShowFactDialog] = useState(false);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [showTaskDialog, setShowTaskDialog] = useState(false);


  return (
    <ShortcutContext.Provider
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
        selectedTask,
        setSelectedTask,
      }}
    >
      {children}
    </ShortcutContext.Provider>
  );
};

export const useShortcutContext = () => {
  const context = useContext(ShortcutContext);
  if (context === undefined) {
    throw new Error("useShortcutContext must be used within a ShorcutProvider");
  }
  return context;
};
