import React from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { SearchPage } from "./cards/SearchPage";
import { UserSettingsPage } from "./UserSettings";
import { FileVault } from "./FileVault";
import { ViewPage } from "./cards/ViewPage";
import { EditPage } from "./cards/EditPage";
import Success from "./Success";
import Cancel from "./Cancel";
import SubscribePage from "./SubscribePage";
import { DashboardPage } from "./DashboardPage";
import { TaskPage } from "./tasks/TaskPage";
import { EntityPage } from "./EntityPage";
import { Summarizer } from "./Summarizer";
import { FactPage } from "./FactPage";
import { MemoryPage } from "./MemoryPage";
import { HelpPage } from "./HelpPage";
import { CalendarPage } from "./calendar/CalendarPage";
import { ChatPage } from "./ChatPage";
import { StatsPage } from "./StatsPage";
import { SchemaPage } from "./SchemaPage";
import { SchemaCreatePage } from "./SchemaCreatePage";
import { SchemaEditPage } from "./SchemaEditPage";
import { SchemaTableWrapper } from "./SchemaTableWrapper";
import { RssPage } from "./RssPage";
import { RssManagePage } from "./RssManagePage";
import { EmailInboxPage } from "./EmailInboxPage";
import { EmailDetailPage } from "./EmailDetailPage";
import { NotificationInboxPage } from "./NotificationInboxPage";
import { SearchConfig } from "../models/StarredSearch";
import { SearchResult } from "../models/Card";

interface AppRoutesProps {
  hasSubscription: boolean;
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;
  searchConfig: SearchConfig;
  setSearchConfig: (config: SearchConfig) => void;
  includeStats?: boolean;
}

/**
 * Consolidated route configuration for MainApp.
 * Eliminates duplication of the same route tree across different layout modes.
 */
export function AppRoutes({
  hasSubscription,
  searchTerm,
  setSearchTerm,
  searchResults,
  setSearchResults,
  searchConfig,
  setSearchConfig,
  includeStats = false,
}: AppRoutesProps) {
  return (
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
          <Route path="card/:id/edit" element={<EditPage newCard={false} />} />
          <Route path="card/new" element={<EditPage newCard={true} />} />
          <Route path="settings" element={<UserSettingsPage />} />
          <Route path="help" element={<HelpPage />} />
          <Route path="files" element={<FileVault />} />
          <Route path="tasks" element={<TaskPage />} />
          <Route path="calendar" element={<CalendarPage />} />
          <Route path="entities" element={<EntityPage />} />
          <Route path="summarizer" element={<Summarizer />} />
          <Route path="facts" element={<FactPage />} />
          <Route path="memory" element={<MemoryPage />} />
          {includeStats && <Route path="stats" element={<StatsPage />} />}
          <Route path="schemas" element={<SchemaPage />} />
          <Route path="schemas/new" element={<SchemaCreatePage />} />
          <Route path="schemas/:id/edit" element={<SchemaEditPage />} />
          <Route path="schemas/:id/table" element={<SchemaTableWrapper />} />
          <Route path="chat" element={<ChatPage />} />
          <Route path="rss" element={<RssPage />} />
          <Route path="rss/manage" element={<RssManagePage />} />
          <Route path="emails" element={<EmailInboxPage />} />
          <Route path="emails/:id" element={<EmailDetailPage />} />
          <Route path="inbox" element={<NotificationInboxPage />} />
          <Route path="*" element={<DashboardPage />} />
        </>
      ) : (
        <Route path="*" element={<Navigate to="/app/subscription" replace />} />
      )}
    </Routes>
  );
}
