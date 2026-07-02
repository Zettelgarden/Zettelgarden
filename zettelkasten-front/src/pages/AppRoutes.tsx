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
import { StatsPage } from "./StatsPage";
import { SchemaPage } from "./SchemaPage";
import { SchemaCreatePage } from "./SchemaCreatePage";
import { SchemaEditPage } from "./SchemaEditPage";
import { SchemaTableWrapper } from "./SchemaTableWrapper";
import { RssPage } from "./RssPage";
import { RssManagePage } from "./RssManagePage";
import { Habits } from "./Habits";
import { NotificationInboxPage } from "./NotificationInboxPage";
import { Paywall } from "../components/Paywall";
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
 *
 * Free-tier users can access: dashboard, search, cards, tasks, settings, help.
 * Pro features (AI, entities, summarizer, etc.) show a paywall.
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
  /** Wrap a pro-only element with a paywall for free-tier users. */
  const proOnly = (element: React.ReactNode, feature: string) =>
    hasSubscription ? element : <Paywall feature={feature} />;

  return (
    <Routes>
      {/* Always accessible */}
      <Route path="subscription" element={<SubscribePage />} />
      <Route path="settings/billing/success" element={<Success />} />
      <Route path="settings/billing/cancel" element={<Cancel />} />

      {/* Free-tier routes */}
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
      <Route path="tasks" element={<TaskPage />} />

      {/* Pro-only routes */}
      <Route path="files" element={proOnly(<FileVault />, "File Vault")} />
      <Route path="entities" element={proOnly(<EntityPage />, "Entities")} />
      <Route path="summarizer" element={proOnly(<Summarizer />, "Summarizer")} />
      <Route path="facts" element={proOnly(<FactPage />, "Facts")} />
      <Route path="memory" element={proOnly(<MemoryPage />, "Memory")} />
      {includeStats && <Route path="stats" element={proOnly(<StatsPage />, "Stats")} />}
      <Route path="schemas" element={proOnly(<SchemaPage />, "Schemas")} />
      <Route path="schemas/new" element={proOnly(<SchemaCreatePage />, "Schemas")} />
      <Route path="schemas/:id/edit" element={proOnly(<SchemaEditPage />, "Schemas")} />
      <Route path="schemas/:id/table" element={proOnly(<SchemaTableWrapper />, "Schemas")} />
      <Route path="habits" element={proOnly(<Habits />, "Habits")} />
      <Route path="rss" element={proOnly(<RssPage />, "RSS")} />
      <Route path="rss/manage" element={proOnly(<RssManagePage />, "RSS")} />
      <Route path="inbox" element={proOnly(<NotificationInboxPage />, "Inbox")} />

      {/* Default route */}
      <Route path="*" element={<DashboardPage />} />
    </Routes>
  );
}
