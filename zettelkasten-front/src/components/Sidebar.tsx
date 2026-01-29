import React, { useState, useEffect, useMemo, useCallback } from "react";
import { useLocation } from "react-router-dom";
import { useTaskContext } from "../contexts/TaskContext";
import { useUIState } from "../contexts/UIStateContext";
import { useDialogState } from "../contexts/DialogStateContext";
import { isTodayOrPast } from "../utils/dates";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { useKeyboardShortcuts } from "../hooks/useKeyboardShortcuts";
import { useToast } from "./toast/ToastContext";

import { PartialCard, Card, Entity } from "../models/Card";

import { SidebarHeader } from "./sidebar/SidebarHeader";
import { NavigationLinks } from "./sidebar/NavigationLinks";
import { SecondaryNavigationLinks } from "./sidebar/SecondaryNavigationLinks";
import { StarredSearchesSection } from "./sidebar/StarredSearchesSection";
import { StarredCardsSection } from "./sidebar/StarredCardsSection";
import { SidebarFooter } from "./sidebar/SidebarFooter";
import { SidebarModals } from "./sidebar/SidebarModals";
import { SidebarMobileMenu } from "./sidebar/SidebarMobileMenu";


export function Sidebar() {
  const navigate = useNavigate();
  const location = useLocation();
  const { lastCard, conversationId, setConversationId } = useUIState();
  const { showToast } = useToast();
  const { tasks } = useTaskContext();
  const [isSidebarOpen, setIsSidebarOpen] = useState<boolean>(false);
  const [showAddArticleDialog, setShowAddArticleDialog] = useState(false);
  const [showStarCardDialog, setShowStarCardDialog] = useState(false);
  const { hasSubscription, user, updateUser } = useAuth();

  const userTimezone = user?.timezone || "UTC";

  const [showGettingStarted, setShowGettingStarted] = useState(false);
  const [showEditEntityDialog, setShowEditEntityDialog] = useState(false);
  const [entityToEdit, setEntityToEdit] = useState<Entity | null>(null);
  const {
    showCreateTaskWindow,
    setShowCreateTaskWindow,
    showQuickSearchWindow,
    setShowQuickSearchWindow,
    showEntityDialog,
    setShowEntityDialog,
    selectedEntity,
    setSelectedEntity,
    showFactDialog,
    setShowFactDialog,
    selectedFact,
    setSelectedFact,
    showTaskDialog,
    setShowTaskDialog,
    selectedTaskId,
    setSelectedTaskId,
  } = useDialogState();

  const currentCard = useMemo(() => {
    const currentPath = location.pathname;
    const isCardPage = /^\/app\/card\/\d+$/.test(currentPath);
    if (isCardPage) {
      return lastCard;
    }
    return null;
  }, [location.pathname, lastCard]);

  const todayTasks = useMemo(
    () =>
      tasks.filter(
        (task) => !task.is_complete && isTodayOrPast(task.scheduled_date, userTimezone),
      ),
    [tasks, userTimezone],
  );

  function handleNewStandardCard() {
    navigate("/app/card/new", { state: { cardType: "standard" } });
  }

  function handleNewChat() {
    navigate("/app/chat?new=true");
  }

  function handleNewTask() {
    setShowCreateTaskWindow(true);
  }

  function handleAddArticle() {
    setShowAddArticleDialog(true);
  }


  async function handleCloseGettingStarted() {
    setShowGettingStarted(false);
    if (user) {
      user.has_seen_getting_started = true;
      updateUser(user);
    }
  }

  useEffect(() => {
    if (user && !user.has_seen_getting_started) {
      setShowGettingStarted(true);
    }
  }, [user]);

  const handleCreateTask = useCallback(() => {
    setShowQuickSearchWindow(false);
    setShowCreateTaskWindow(true);
  }, []);

  const handleQuickSearch = useCallback(() => {
    setShowCreateTaskWindow(false);
    setShowQuickSearchWindow(true);
  }, []);

  // Use custom hook for keyboard shortcuts
  useKeyboardShortcuts({
    onCreateTask: handleCreateTask,
    onQuickSearch: handleQuickSearch,
  });

  return (
    <>
      <SidebarMobileMenu
        isSidebarOpen={isSidebarOpen}
        setIsSidebarOpen={setIsSidebarOpen}
      />

      {/* Sidebar */}
      <div
        className={`
    fixed md:relative
    z-[50]
    w-72
    min-w-[18rem]
    max-w-[18rem]
    flex-shrink-0
    h-screen
    bg-white
    flex flex-col
    border-r
    transform
    ${isSidebarOpen ? "translate-x-0" : "-translate-x-full"}
    md:translate-x-0
    transition-transform
    duration-300
    ease-in-out
  `}
      >
        <SidebarHeader
          onNewStandardCard={handleNewStandardCard}
          onNewArticle={handleAddArticle}
          onNewTask={handleNewTask}
          onNewChat={handleNewChat}
        />

        {/* Scrollable Middle Section */}
        <div className="flex-1 overflow-y-auto">
          <NavigationLinks
            todayTasksCount={todayTasks.length}
            hasSubscription={hasSubscription}
          />
          <hr />
          <SecondaryNavigationLinks hasSubscription={hasSubscription} />

          <StarredSearchesSection />
          <StarredCardsSection
            onShowStarCardDialog={() => setShowStarCardDialog(true)}
          />
          <hr />
        </div>
        <SidebarFooter />
      </div>

      <SidebarModals
        showCreateTaskWindow={showCreateTaskWindow}
        setShowCreateTaskWindow={setShowCreateTaskWindow}
        showQuickSearchWindow={showQuickSearchWindow}
        setShowQuickSearchWindow={setShowQuickSearchWindow}
        showStarCardDialog={showStarCardDialog}
        setShowStarCardDialog={setShowStarCardDialog}
        showEntityDialog={showEntityDialog}
        setShowEntityDialog={setShowEntityDialog}
        selectedEntity={selectedEntity}
        setSelectedEntity={setSelectedEntity}
        showFactDialog={showFactDialog}
        setShowFactDialog={setShowFactDialog}
        showTaskDialog={showTaskDialog}
        setShowTaskDialog={setShowTaskDialog}
        selectedTaskId={selectedTaskId}
        showEditEntityDialog={showEditEntityDialog}
        setShowEditEntityDialog={setShowEditEntityDialog}
        entityToEdit={entityToEdit}
        setEntityToEdit={setEntityToEdit}
        showAddArticleDialog={showAddArticleDialog}
        setShowAddArticleDialog={setShowAddArticleDialog}
        showGettingStarted={showGettingStarted}
        setShowGettingStarted={setShowGettingStarted}
        currentCard={currentCard}
        handleCloseGettingStarted={handleCloseGettingStarted}
      />
    </>
  );
}
