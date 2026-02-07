import React, { useState, useEffect } from "react";
import "../App.css";
import { Sidebar } from "../components/Sidebar";
import { useAuth } from "../contexts/AuthContext";
import { Navigate, useNavigate } from "react-router-dom";
import { Route, Routes } from "react-router-dom";
import { EmailValidationBanner } from "../components/EmailValidationBanner";
import { Card, PartialCard, SearchResult } from "../models/Card";
import { TaskProvider, useTaskContext } from "../contexts/TaskContext";
import { StatusProvider } from "../contexts/StatusContext";
import { TagProvider } from "../contexts/TagContext";
import { UIStateProvider } from "../contexts/UIStateContext";
import { DialogStateProvider } from "../contexts/DialogStateContext";
import { RSSProvider } from "../contexts/RSSContext";
import { SplitViewLayout } from "../components/cards/SplitViewLayout";
import { ChatSidebarLayout } from "../components/chat/ChatSidebarLayout";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { ToastProvider } from "../components/toast/ToastContext";
import { AppRoutes } from "./AppRoutes";
import { SearchConfig } from "../models/StarredSearch";
import { useUIState } from "../contexts/UIStateContext";

function MainAppContent() {
  const navigate = useNavigate();
  const [searchTerm, setSearchTerm] = useState("");
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searchConfig, setSearchConfig] = useState<SearchConfig>({
    sortBy: "sortRanking",
    currentPage: 1,
    useClassicSearch: false,
    useFullText: false,
    onlyParentCards: false,
    onlyEmptyCardId: false,
    showEntities: true,
    showPreview: true,
    showFacts: true,
    showCards: true,
    searchType: "typesense",
    rerank: true,
  });
  const {
    isAuthenticated,
    isLoading,
    hasSubscription,
    logoutUser,
    user,
    updateUser,
  } = useAuth();
  const { setRefreshTasks } = useTaskContext();
  const { pinnedCard, isPinMode, chatSidebarCard, isChatSidebarMode } = useUIState();

  // changing pages

  async function handleNewCard(cardType: string) {
    navigate("/app/card/new", { state: { cardType: cardType } });
  }

  useEffect(() => {
    if (!localStorage.getItem("token")) {
      logoutUser();
    }
  }, [isAuthenticated]);

  useEffect(() => {
    setRefreshTasks(true);
  }, []);

  if (isLoading) {
    return <div>Loading...</div>;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  return (
    <div>

      <div className="flex h-screen overflow-hidden">
        <Sidebar />
        <div className="flex-grow overflow-y-auto pb-16 md:pb-0 safe-bottom">
          {isPinMode && pinnedCard ? (
            <ErrorBoundary>
              <SplitViewLayout pinnedCard={pinnedCard}>
                <div className="">
                  <EmailValidationBanner />
                  <AppRoutes
                    hasSubscription={hasSubscription}
                    searchTerm={searchTerm}
                    setSearchTerm={setSearchTerm}
                    searchResults={searchResults}
                    setSearchResults={setSearchResults}
                    searchConfig={searchConfig}
                    setSearchConfig={setSearchConfig}
                  />
                </div>
              </SplitViewLayout>
            </ErrorBoundary>
          ) : isChatSidebarMode && chatSidebarCard ? (
            <ErrorBoundary>
              <ChatSidebarLayout chatSidebarCard={chatSidebarCard}>
                <div className="">
                  <EmailValidationBanner />
                  <AppRoutes
                    hasSubscription={hasSubscription}
                    searchTerm={searchTerm}
                    setSearchTerm={setSearchTerm}
                    searchResults={searchResults}
                    setSearchResults={setSearchResults}
                    searchConfig={searchConfig}
                    setSearchConfig={setSearchConfig}
                  />
                </div>
              </ChatSidebarLayout>
            </ErrorBoundary>
          ) : (
            <div className="">
              <EmailValidationBanner />
              <AppRoutes
                hasSubscription={hasSubscription}
                searchTerm={searchTerm}
                setSearchTerm={setSearchTerm}
                searchResults={searchResults}
                setSearchResults={setSearchResults}
                searchConfig={searchConfig}
                setSearchConfig={setSearchConfig}
                includeStats
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function MainApp() {
  return (
    <ToastProvider>
      <TagProvider>
        <TaskProvider>
          <StatusProvider>
            <UIStateProvider>
              <DialogStateProvider>
                <RSSProvider>
                  <MainAppContent />
                </RSSProvider>
              </DialogStateProvider>
            </UIStateProvider>
          </StatusProvider>
        </TaskProvider>
      </TagProvider>
    </ToastProvider>
  );
}

export default MainApp;
