import React, { useState, useEffect, useMemo, useCallback } from "react";
import { useLocation } from "react-router-dom";
import { useTaskContext } from "../contexts/TaskContext";
import { useUIState } from "../contexts/UIStateContext";
import { useDialogState } from "../contexts/DialogStateContext";
import { useRSS } from "../contexts/RSSContext";
import { isTodayOrPast } from "../utils/dates";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";
import { useKeyboardShortcuts } from "../hooks/useKeyboardShortcuts";
import { useToast } from "./toast/ToastContext";
import { listFolders } from "../api/rss";
import { RSSFolder, RSSFeed } from "../api/rss";

import { PartialCard, Card, Entity } from "../models/Card";

import { SidebarHeader } from "./sidebar/SidebarHeader";
import { NavigationLinks } from "./sidebar/NavigationLinks";
import { StarredSearchesSection } from "./sidebar/StarredSearchesSection";
import { StarredCardsSection } from "./sidebar/StarredCardsSection";
import { SidebarFooter } from "./sidebar/SidebarFooter";
import { SidebarModals } from "./sidebar/SidebarModals";
import { MobileBottomNav } from "./mobile/MobileBottomNav";
import { RssAddFeedDialog } from "./rss/RssAddFeedDialog";
import { SidebarSearchBar } from "./sidebar/SidebarSearchBar";

export function Sidebar() {
  const navigate = useNavigate();
  const location = useLocation();
  const { lastCard, isSidebarCollapsed, toggleSidebarCollapsed, toggleRightPane, isMobileSidebarOpen, setIsMobileSidebarOpen } = useUIState();
  const { showToast } = useToast();
  const { tasks } = useTaskContext();
  const { unreadCount: unreadRssCount } = useRSS();
  const [showAddArticleDialog, setShowAddArticleDialog] = useState(false);
  const [showAddFeedDialog, setShowAddFeedDialog] = useState(false);
  const [rssFolders, setRssFolders] = useState<RSSFolder[]>([]);
  const [showStarCardDialog, setShowStarCardDialog] = useState(false);
  const { user, updateUser } = useAuth();

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

  function handleNewTask() {
    setShowCreateTaskWindow(true);
  }

  function handleAddArticle() {
    setShowAddArticleDialog(true);
  }

  function handleAddFeed() {
    setShowAddFeedDialog(true);
  }

  function handleFeedAdded(feed: RSSFeed) {
    // Feed added successfully - dialog will close itself
    // No additional action needed - RSS page will show the new feed when visited
    console.log("Feed added:", feed);
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

  useEffect(() => {
    async function fetchFolders() {
      try {
        const folders = await listFolders();
        setRssFolders(folders);
      } catch (error) {
        console.error("Failed to fetch RSS folders:", error);
      }
    }
    fetchFolders();
  }, []);

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
    onToggleRightPane: toggleRightPane,
  });

  return (
    <>
      {/* Mobile Backdrop */}
      {isMobileSidebarOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 md:hidden z-[45] safe-all"
          onClick={() => setIsMobileSidebarOpen(false)}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              setIsMobileSidebarOpen(false);
            }
          }}
          tabIndex={0}
          role="button"
          aria-label="Close sidebar menu"
        />
      )}

      {/* Sidebar */}
      <div
        className={`
    fixed md:relative
    z-[50]
    flex-shrink-0
    h-screen
    bg-white
    flex flex-col
    border-r
    transform
    ${isMobileSidebarOpen ? "translate-x-0" : "-translate-x-full"}
    md:translate-x-0
    transition-all
    duration-300
    ease-in-out
    ${isSidebarCollapsed ? "w-16 min-w-[4rem] max-w-[4rem]" : "w-72 min-w-[18rem] max-w-[18rem]"}
  `}
      >
        <SidebarHeader
          onNewStandardCard={handleNewStandardCard}
          onNewArticle={handleAddArticle}
          onNewTask={handleNewTask}
          onAddFeed={handleAddFeed}
          isCollapsed={isSidebarCollapsed}
          onToggleCollapse={toggleSidebarCollapsed}
        />

        <SidebarSearchBar isCollapsed={isSidebarCollapsed} />

        {/* Scrollable Middle Section */}
        <div className="flex-1 overflow-y-auto" style={{ paddingBottom: 'env(safe-area-inset-bottom, 0)' }}>
          <NavigationLinks
            todayTasksCount={todayTasks.length}
            unreadRssCount={unreadRssCount}
            isCollapsed={isSidebarCollapsed}
          />

          {!isSidebarCollapsed && (
            <>
              <hr />
              <StarredSearchesSection />
              <StarredCardsSection
                onShowStarCardDialog={() => setShowStarCardDialog(true)}
              />
              <hr />
            </>
          )}
        </div>
        <SidebarFooter
          isCollapsed={isSidebarCollapsed}
          onToggleCollapse={toggleSidebarCollapsed}
        />
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

      {showAddFeedDialog && (
        <RssAddFeedDialog
          isOpen={showAddFeedDialog}
          onClose={() => setShowAddFeedDialog(false)}
          folders={rssFolders}
          onFeedAdded={handleFeedAdded}
        />
      )}
    </>
  );
}
