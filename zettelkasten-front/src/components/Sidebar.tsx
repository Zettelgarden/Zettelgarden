import React, { useState, useEffect, ChangeEvent, useMemo } from "react";
import { CardItem } from "./cards/CardItem";
import { Link, useLocation } from "react-router-dom";
import { useTaskContext } from "../contexts/TaskContext";
import { useChatContext } from "../contexts/ChatContext";
import { isTodayOrPast } from "../utils/dates";
import { usePartialCardContext } from "../contexts/CardContext";
import { CreateTaskWindow } from "./tasks/CreateTaskWindow";
import { StarCardDialog } from "./cards/StarCardDialog";
import logo from "../assets/logo.png";
import { useNavigate } from "react-router-dom";
import { SidebarLink } from "./SidebarLink";
import { SearchIcon } from "../assets/icons/SearchIcon";
import { TasksIcon } from "../assets/icons/TasksIcon";
import { FileIcon } from "../assets/icons/FileIcon";
import { ChatIcon } from "../assets/icons/ChatIcon";
import { MenuIcon } from "../assets/icons/MenuIcon";
import { Button } from "./Button";
import { useAuth } from "../contexts/AuthContext";

import { GettingStartedPage } from "../pages/GettingStartedPage";

import { useShortcutContext } from "../contexts/ShortcutContext";
import { QuickSearchWindow } from "./cards/QuickSearchWindow";

import { PartialCard, Card, Entity } from "../models/Card";
import { getStarredCards, unstarCard } from "../api/cards";
import { StarredSearch } from "../models/StarredSearch";
import { getStarredSearches, unstarSearch } from "../api/starredSearches";
import { parseURL } from "../api/references";

import { defaultCard } from "../models/Card";
import { EntityIcon } from "../assets/icons/EntityIcon";
import { BookOpenIcon } from "../assets/icons/BookOpenIcon";
import { SettingsIcon } from "../assets/icons/SettingsIcon";
import { FactsIcon } from "../assets/icons/FactsIcon";
import { MemoryIcon } from "../assets/icons/MemoryIcon";

import { EntityDialog } from "./entities/EntityDialog";
import { EditEntityDialog } from "./entities/EditEntityDialog";
import { FactDialog } from "./facts/FactDialog";
import { AddArticleDialog } from "./cards/AddArticleDialog";
import { TaskDialog } from "./tasks/TaskDialog";


export function Sidebar() {
  const navigate = useNavigate();
  const location = useLocation();
  const [message, setMessage] = useState<string>("");
  const { lastCard } = usePartialCardContext();
  const { tasks } = useTaskContext();
  const username = localStorage.getItem("username");
  const [isNewDropdownOpen, setIsNewDropdownOpen] = useState(false);
  const [showStarCardDialog, setShowStarCardDialog] = useState(false);
  const [starredCards, setStarredCards] = useState<Card[]>([]);
  const [starredSearches, setStarredSearches] = useState<StarredSearch[]>([]);
  const [isSidebarOpen, setIsSidebarOpen] = useState<boolean>(false);
  const { conversationId, setConversationId } = useChatContext();
  const [showAddArticleDialog, setShowAddArticleDialog] = useState(false);
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
    selectedTask,
  } = useShortcutContext();

  function getCurrentCard(): PartialCard | Card | null {
    const location = useLocation();
    const currentPath = location.pathname;
    const isCardPage = /^\/app\/card\/\d+$/.test(currentPath);
    if (isCardPage) {
      return lastCard;
    }
    return null;
  }
  function handleNewStandardCard() {
    toggleNewDropdown();
    navigate("/app/card/new", { state: { cardType: "standard" } });
  }
  function handleNewChat() {
    toggleNewDropdown();
    setConversationId("");
    navigate("/app/chat");
  }
  function handleNewTask() {
    toggleNewDropdown();
    setShowCreateTaskWindow(true);
  }

  const toggleNewDropdown = () => {
    console.log("?");
    setIsNewDropdownOpen(!isNewDropdownOpen);
    console.log(isNewDropdownOpen);
  };

  const todayTasks = useMemo(
    () =>
      tasks.filter(
        (task) => !task.is_complete && isTodayOrPast(task.scheduled_date),
      ),
    [tasks],
  );

  function handleAddArticle() {
    toggleNewDropdown();
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
      if (event.key === "c") {
        navigate("/app/card/new", { state: { cardType: "standard" } });
      }
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

  // Function to unstar a card
  const handleUnstarCard = (cardId: number) => {
    unstarCard(cardId)
      .then(() => {
        // Refresh the starred cards list after unstarring
        refreshStarredCards();
        // Show a success message
        setMessage("Card unstarred successfully");
      })
      .catch(error => {
        console.error("Error unstarring card:", error);
        setMessage("Error unstarring card");
      });
  };

  // Function to unstar a search
  const handleUnstarSearch = (searchId: number) => {
    unstarSearch(searchId)
      .then(() => {
        // Refresh the starred searches list after unstarring
        refreshStarredSearches();
        // Show a success message
        setMessage("Search unstarred successfully");
      })
      .catch(error => {
        console.error("Error unstarring search:", error);
        setMessage("Error unstarring search");
      });
  };

  // Function to refresh starred cards
  const refreshStarredCards = () => {
    getStarredCards()
      .then((cards) => {
        setStarredCards(cards);
      })
      .catch(error => {
        console.error("Error fetching starred cards:", error);
      });
  };

  // Function to refresh starred searches
  const refreshStarredSearches = () => {
    getStarredSearches()
      .then((searches) => {
        setStarredSearches(searches);
      })
      .catch(error => {
        console.error("Error fetching starred searches:", error);
      });
  };

  useEffect(() => {
    // getUserConversations().then((conversations) => {
    //   setChatConversations(conversations);
    // });

    // Fetch starred cards and searches
    refreshStarredCards();
    refreshStarredSearches();
  }, []);

  // Refresh starred items when location changes (navigation occurs)
  useEffect(() => {
    refreshStarredCards();
    refreshStarredSearches();
  }, [location.pathname]);
  return (
    <>
      {/* Mobile Menu Button */}
      <button
        className="md:hidden fixed top-4 right-4 z-[60] p-2 bg-white rounded shadow"
        onClick={() => setIsSidebarOpen(!isSidebarOpen)}
      >
        <MenuIcon />
      </button>

      {/* Mobile Backdrop */}
      {isSidebarOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 md:hidden z-[45]"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <div
        className={`
    fixed md:relative
    z-[50]
    w-72
    min-w-[18rem]    // Increased minimum width
    max-w-[18rem]    // Increased maximum width
    flex-shrink-0    // Add this to prevent shrinking
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
        {/* Upper Section */}
        <div className="flex items-center p-4 border-b">
          <Link to="/app" className="flex-shrink-0">
            <img
              src={logo}
              alt="Company Logo"
              className="h-8 w-auto rounded-md"
            />
          </Link>
          <div className="flex-grow mx-2 min-w-0">
            <Link to="/app/settings">
              <span className="text-sm font-medium hover:text-gray-700 truncate block">
                {username}
              </span>
            </Link>
          </div>
          <div className="relative flex-shrink-0">
            <Button
              onClick={toggleNewDropdown}
              className="w-8 h-8 flex items-center justify-center rounded-full bg-blue-500 text-white hover:bg-blue-600"
            >
              +
            </Button>
            {isNewDropdownOpen && (
              <div className="absolute right-0 mt-2 w-48 bg-white rounded-md shadow-lg py-1 z-[70] border">
                <button
                  onClick={handleNewStandardCard}
                  className="w-full text-left px-4 py-2 hover:bg-gray-100"
                >
                  Create Card
                </button>
                <button
                  onClick={handleAddArticle}
                  className="w-full text-left px-4 py-2 hover:bg-gray-100"
                >
                  Add Article (Card)
                </button>
                <button
                  onClick={handleNewTask}
                  className="w-full text-left px-4 py-2 hover:bg-gray-100"
                >
                  Create Task
                </button>
                {hasSubscription && (
                  <button
                    onClick={handleNewChat}
                    className="w-full text-left px-4 py-2 hover:bg-gray-100"
                  >
                    New Chat
                  </button>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Scrollable Middle Section */}
        <div className="flex-1 overflow-y-auto">
          {/* Navigation Links */}
          <div className="p-2">

            <ul className="space-y-1">
              <SidebarLink to="/app/search?recent=true">
                <SearchIcon />
                <span className="px-2 flex-grow">Search</span>
              </SidebarLink>

              <SidebarLink to="/app/tasks">
                <TasksIcon />
                <span className="px-2 flex-grow">Tasks</span>
                <span className="px-2 py-1 text-xs bg-blue-100 rounded-full">
                  {todayTasks.length}
                </span>
              </SidebarLink>

              <SidebarLink to="/app/chat">
                <ChatIcon />
                <span className="px-2 flex-grow">Chat</span>
                {!hasSubscription && (
                  <span className="ml-2 bg-purple-500 text-white text-xs font-semibold px-2 py-0.5 rounded-full">PRO</span>
                )}
              </SidebarLink>
            </ul>
          </div>
          <hr />
          <div className="p-2">
            <ul className="space-y-1">

              <SidebarLink to="/app/entities">
                <EntityIcon />
                <span className="px-2 flex-grow">Entities</span>
                {!hasSubscription && (
                  <span className="ml-2 bg-purple-500 text-white text-xs font-semibold px-2 py-0.5 rounded-full">PRO</span>
                )}
              </SidebarLink>
              <SidebarLink to="/app/facts">
                <FactsIcon />
                <span className="px-2 flex-grow">Facts</span>
                {!hasSubscription && (
                  <span className="ml-2 bg-purple-500 text-white text-xs font-semibold px-2 py-0.5 rounded-full">PRO</span>
                )}
              </SidebarLink>
              <SidebarLink to="/app/memory">
                <MemoryIcon />
                <span className="px-2 flex-grow">Memory</span>
              </SidebarLink>
            </ul>
          </div>

          {/* Starred Searches Section */}
          <>
            <hr />
            <div className="p-2">
              <div className="flex items-center justify-between mb-2 px-2">
                <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
                  Starred Searches
                </h3>
              </div>
              {starredSearches.length > 0 ? (
                <ul className="space-y-0.5">
                  {starredSearches.map((search) => (
                    <li key={search.id} className="px-2 py-0.5 text-sm group">
                      <div className="flex items-center">
                        <Link
                          to={`/app/search?term=${encodeURIComponent(search.searchTerm)}&starred=${search.id}`}
                          className="flex-grow hover:bg-gray-100 rounded p-1 truncate"
                          title={search.title}
                          onClick={() => {
                            // This will be handled in SearchPage.tsx when it detects the starred parameter
                          }}
                        >
                          <span className="mr-1">•</span>
                          {search.title}
                        </Link>
                        <button
                          onClick={() => handleUnstarSearch(search.id)}
                          className="text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity px-1"
                          title="Unstar search"
                        >
                          ×
                        </button>
                      </div>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-xs text-gray-400 px-2">No starred searches yet</p>
              )}
            </div>
          </>

          {/* Starred Cards Section */}
          <>
            <hr />
            <div className="p-2">
              <div className="flex items-center justify-between mb-2 px-2">
                <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
                  Starred Cards
                </h3>
                <button
                  onClick={() => setShowStarCardDialog(true)}
                  className="w-5 h-5 flex items-center justify-center text-gray-400 hover:text-blue-500 rounded-full"
                  title="Star a card"
                >
                  +
                </button>
              </div>
              {starredCards.length > 0 ? (
                <ul className="space-y-0.5">
                  {starredCards.map((card) => (
                    <li key={card.id} className="text-sm group relative">
                      <div className="flex items-center">
                        <div className="flex-grow min-w-0">
                          <CardItem card={card} />
                        </div>
                        <button
                          onClick={() => handleUnstarCard(card.id)}
                          className="absolute right-2 text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-opacity"
                          title="Unstar card"
                        >
                          ×
                        </button>
                      </div>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-xs text-gray-400 px-2">No starred cards yet</p>
              )}
            </div>
          </>
          <hr />
        </div>
        {/* Bottom icons section - Help and Settings */}
        <div className="p-2 border-t">
          <div className="flex justify-end space-x-4 pr-2">
            <SidebarLink to="/app/help">
              <BookOpenIcon />
            </SidebarLink>
            <SidebarLink to="/app/settings">
              <SettingsIcon />
            </SidebarLink>
          </div>
        </div>
      </div>
      {/* Modal Windows */}
      {showCreateTaskWindow && (
        <CreateTaskWindow
          currentCard={getCurrentCard()}
          setShowTaskWindow={setShowCreateTaskWindow}
        />
      )}

      {showQuickSearchWindow && (
        <QuickSearchWindow setShowWindow={setShowQuickSearchWindow} />
      )}

      {showStarCardDialog && (
        <StarCardDialog
          onClose={() => setShowStarCardDialog(false)}
          onStarSuccess={refreshStarredCards}
          setMessage={setMessage}
        />
      )}
      <EntityDialog
        onClose={() => { setShowEntityDialog(false) }}
        onEdit={(entity) => {
          setEntityToEdit(entity);
          setShowEditEntityDialog(true);
        }}
      />
      <FactDialog
        onClose={() => setShowFactDialog(false)}
        onFactDeleted={() => setShowFactDialog(false)}
      />
      <TaskDialog
        task={selectedTask}
        isOpen={showTaskDialog}
        onClose={() => setShowTaskDialog(false)}
        onTagClick={(tag: string) => {
          // Handle tag click if needed - navigate to tasks filtered by tag
          navigate(`/app/tasks?tag=${encodeURIComponent(tag)}`);
        }}
      />
      <EditEntityDialog
        entity={entityToEdit}
        isOpen={showEditEntityDialog}
        onClose={() => {
          setShowEditEntityDialog(false);
          setEntityToEdit(null);
        }}
        onSuccess={() => {
          // Refresh the entity dialog if it's still open
          if (selectedEntity && entityToEdit && selectedEntity.id === entityToEdit.id) {
            // Force refresh of the entity dialog by toggling it
            setShowEntityDialog(false);
            setTimeout(() => {
              setSelectedEntity(entityToEdit);
              setShowEntityDialog(true);
            }, 100);
          }
        }}
        onDelete={(entity) => {
          // Close edit dialog and entity dialog
          setShowEditEntityDialog(false);
          setShowEntityDialog(false);
          setEntityToEdit(null);
          setSelectedEntity(null);
        }}
      />
      <AddArticleDialog
        show={showAddArticleDialog}
        onClose={() => setShowAddArticleDialog(false)}
        setMessage={setMessage}
      />
      {showGettingStarted && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center z-[1000]"
          onClick={() => handleCloseGettingStarted()}
        >
          <div
            className="bg-white p-5 rounded-md shadow-lg max-w-4xl w-[90%] max-h-[90vh] overflow-y-auto"
            onClick={(e) => e.stopPropagation()}
          >
            <GettingStartedPage setShowGettingStarted={setShowGettingStarted} />
          </div>
        </div>
      )}
    </>
  );
}
