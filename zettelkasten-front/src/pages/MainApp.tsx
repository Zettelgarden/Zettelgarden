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
import { ChatProvider, useChatContext } from "../contexts/ChatContext";
import {
  PartialCardProvider,
} from "../contexts/CardContext";
import { ShortcutProvider } from "../contexts/ShortcutContext";
import { FileProvider } from "../contexts/FileContext";
import { PinProvider, usePinContext } from "../contexts/PinContext";
import { ChatSidebarProvider, useChatSidebarContext } from "../contexts/ChatSidebarContext";
import { SplitViewLayout } from "../components/cards/SplitViewLayout";
import { ChatSidebarLayout } from "../components/chat/ChatSidebarLayout";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { CardRefreshProvider } from "../contexts/CardRefreshContext";
import { ToastProvider } from "../components/toast/ToastContext";
import { AppRoutes } from "./AppRoutes";

import { SearchConfig } from "../models/StarredSearch";

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
  const { pinnedCard, isPinMode } = usePinContext();
  const { chatSidebarCard, isChatSidebarMode } = useChatSidebarContext();

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
        <div className="flex-grow overflow-y-auto">
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
        <ChatProvider>
          <PartialCardProvider>
            <TaskProvider>
              <StatusProvider>
                <ShortcutProvider>
                  <FileProvider>
                    <PinProvider>
                      <ChatSidebarProvider>
                        <CardRefreshProvider>
                          <MainAppContent />
                        </CardRefreshProvider>
                      </ChatSidebarProvider>
                    </PinProvider>
                  </FileProvider>
                </ShortcutProvider>
              </StatusProvider>
            </TaskProvider>
          </PartialCardProvider>
        </ChatProvider>
      </TagProvider>
    </ToastProvider>
  );
}

export default MainApp;
