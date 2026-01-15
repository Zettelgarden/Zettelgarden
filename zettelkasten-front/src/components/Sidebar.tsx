import React, { useState, useEffect, useMemo } from "react";
import { useLocation } from "react-router-dom";
import { useTaskContext } from "../contexts/TaskContext";
import { useChatContext } from "../contexts/ChatContext";
import { isTodayOrPast } from "../utils/dates";
import { usePartialCardContext } from "../contexts/CardContext";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { useShortcutContext } from "../contexts/ShortcutContext";

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
  const [message, setMessage] = useState<string>("");
  const { lastCard } = usePartialCardContext();
  const { tasks } = useTaskContext();
  const [isSidebarOpen, setIsSidebarOpen] = useState<boolean>(false);
  const { conversationId, setConversationId } = useChatContext();
  const [showAddArticleDialog, setShowAddArticleDialog] = useState(false);
  const [showStarCardDialog, setShowStarCardDialog] = useState(false);
  const { hasSubscription, user, updateUser } = useAuth();

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
    showTaskDialog,
    setShowTaskDialog,
    selectedTaskId,
  } = useShortcutContext();

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
        (task) => !task.is_complete && isTodayOrPast(task.scheduled_date),
      ),
    [tasks],
  );

  function handleNewStandardCard() {
    navigate("/app/card/new", { state: { cardType: "standard" } });
  }

  function handleNewChat() {
    setConversationId("");
    navigate("/app/chat");
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

  const handleKeyPress = (event: KeyboardEvent) => {
    // if this is true, the user is using a system shortcut, don't do anything with it
    if (event.metaKey) {
      return;
    }

    // these should only work if there isn't an input selected
    const focusedElement = document.activeElement;
    if (!focusedElement || !focusedElement.tagName.match(/^INPUT|TEXTAREA$/i)) {
      if (event.key === "t") {
        event.preventDefault();
        setShowQuickSearchWindow(false);
        setShowCreateTaskWindow(true);
      }
      if (event.key === "s") {
        event.preventDefault();
        setShowCreateTaskWindow(false);
        setShowQuickSearchWindow(true);
      }
    }
  };

  useEffect(() => {
    document.addEventListener("keydown", handleKeyPress);
    return () => {
      document.removeEventListener("keydown", handleKeyPress);
    };
  }, []);
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

          <StarredSearchesSection setMessage={setMessage} />
          <StarredCardsSection
            setMessage={setMessage}
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
        setMessage={setMessage}
        handleCloseGettingStarted={handleCloseGettingStarted}
      />
    </>
  );
}
