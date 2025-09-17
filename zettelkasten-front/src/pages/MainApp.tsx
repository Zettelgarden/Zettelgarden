import React, { useState, useEffect } from "react";
import "../App.css";
import { SearchPage } from "./cards/SearchPage";
import { UserSettingsPage } from "./UserSettings";
import { FileVault } from "./FileVault";
import { ViewPage } from "./cards/ViewPage";
import { EditPage } from "./cards/EditPage";
import { Sidebar } from "../components/Sidebar";
import { useAuth } from "../contexts/AuthContext";
import { Navigate, useNavigate } from "react-router-dom";
import { Route, Routes } from "react-router-dom";
import { EmailValidationBanner } from "../components/EmailValidationBanner";
import Success from "./Success";
import Cancel from "./Cancel";
import SubscribePage from "./SubscribePage";
import { DashboardPage } from "./DashboardPage";
import { Card, PartialCard, SearchResult } from "../models/Card";
import { TaskPage } from "./tasks/TaskPage";
import { TaskProvider, useTaskContext } from "../contexts/TaskContext";
import { TagProvider } from "../contexts/TagContext";
import { ChatProvider, useChatContext } from "../contexts/ChatContext";
import {
  PartialCardProvider,
} from "../contexts/CardContext";
import { ShortcutProvider } from "../contexts/ShortcutContext";
import { FileProvider } from "../contexts/FileContext";
import { PinProvider, usePinContext } from "../contexts/PinContext";
import { SplitViewLayout } from "../components/cards/SplitViewLayout";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { EntityPage } from "./EntityPage";
import { CardRefreshProvider } from "../contexts/CardRefreshContext";
import { Summarizer } from "./Summarizer";
import { FactPage } from "./FactPage";
import { MemoryPage } from "./MemoryPage";
import { GettingStartedPage } from "../pages/GettingStartedPage";
import { ChatPage } from "./ChatPage";

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
                <Routes>
                  <Route path="subscription" element={<SubscribePage />} />
                  <Route path="settings/billing/success" element={<Success />} />
                  <Route path="settings/billing/cancel" element={<Cancel />} />
                  {hasSubscription ? (
                    <>
                      <Route
                        path="search"
                        element={
                          <SearchPage
                            searchTerm={searchTerm}
                            setSearchTerm={setSearchTerm}
                            searchResults={searchResults}
                            setSearchResults={setSearchResults}
                            searchConfig={searchConfig}
                            setSearchConfig={setSearchConfig}
                          />
                        }
                      />
                      <Route path="card/:id" element={<ViewPage />} />
                      <Route
                        path="card/:id/edit"
                        element={<EditPage newCard={false} />}
                      />

                      <Route path="card/new" element={<EditPage newCard={true} />} />
                      <Route path="settings" element={<UserSettingsPage />} />
                      <Route path="help" element={<GettingStartedPage />} />
                      <Route path="files" element={<FileVault />} />
                      <Route path="tasks" element={<TaskPage />} />
                      <Route path="entities" element={<EntityPage />} />
                      <Route path="summarizer" element={<Summarizer />} />
                      <Route path="facts" element={<FactPage />} />
                      <Route path="memory" element={<MemoryPage />} />
                      <Route path="chat" element={<ChatPage />} />
                      <Route path="*" element={<DashboardPage />} />
                    </>
                  ) : (
                    <Route
                      path="*"
                      element={<Navigate to="/app/subscription" replace />}
                    />
                  )}
                </Routes>
              </div>
              </SplitViewLayout>
            </ErrorBoundary>
          ) : (
            <div className="">
              <EmailValidationBanner />
              <Routes>
                <Route path="subscription" element={<SubscribePage />} />
                <Route path="settings/billing/success" element={<Success />} />
                <Route path="settings/billing/cancel" element={<Cancel />} />
                {hasSubscription ? (
                  <>
                    <Route
                      path="search"
                      element={
                        <SearchPage
                          searchTerm={searchTerm}
                          setSearchTerm={setSearchTerm}
                          searchResults={searchResults}
                          setSearchResults={setSearchResults}
                          searchConfig={searchConfig}
                          setSearchConfig={setSearchConfig}
                        />
                      }
                    />
                    <Route path="card/:id" element={<ViewPage />} />
                    <Route
                      path="card/:id/edit"
                      element={<EditPage newCard={false} />}
                    />

                    <Route path="card/new" element={<EditPage newCard={true} />} />
                    <Route path="settings" element={<UserSettingsPage />} />
                    <Route path="help" element={<GettingStartedPage />} />
                    <Route path="files" element={<FileVault />} />
                    <Route path="tasks" element={<TaskPage />} />
                    <Route path="entities" element={<EntityPage />} />
                    <Route path="summarizer" element={<Summarizer />} />
                    <Route path="facts" element={<FactPage />} />
                    <Route path="memory" element={<MemoryPage />} />
                    <Route path="chat" element={<ChatPage />} />
                    <Route path="*" element={<DashboardPage />} />
                  </>
                ) : (
                  <Route
                    path="*"
                    element={<Navigate to="/app/subscription" replace />}
                  />
                )}
              </Routes>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function MainApp() {
  return (
    <TagProvider>
      <ChatProvider>
        <PartialCardProvider>
          <TaskProvider>
            <ShortcutProvider>
              <FileProvider>
                <PinProvider>
                  <CardRefreshProvider>
                    <MainAppContent />
                  </CardRefreshProvider>
                </PinProvider>
              </FileProvider>
            </ShortcutProvider>
          </TaskProvider>
        </PartialCardProvider>
      </ChatProvider>
    </TagProvider>
  );
}

export default MainApp;
