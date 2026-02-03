import React, { useState, useEffect, FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { getUserMemory, regenerateCalDAVToken } from "../api/users";
import { getBillingPortalUrl } from "../api/billing";
import { requestPasswordReset } from "../api/auth";
import { User, EditUserParams } from "../models/User";
import { useAuth } from "../contexts/AuthContext";
import { H6 } from "../components/Header";
import { TemplatesList } from "../components/templates/TemplatesList";
import { setDocumentTitle } from "../utils/title";
import { TagList } from "../components/tags/TagList";
import { FileVault } from "./FileVault";
import { StatusManagement } from "../components/settings/StatusManagement";
import { TimezoneSelector } from "../components/settings/TimezoneSelector";
import APIKeysManagement from "../components/settings/APIKeysManagement";
import { CalendarSubscriptions } from "../components/settings/CalendarSubscriptions";
import { MemoryPage } from "./MemoryPage";
import { SchemaPage } from "./SchemaPage";
import { StatsPage } from "./StatsPage";
import { ModelSelector } from "../components/settings/ModelSelector";

type Tab = "profile" | "templates" | "tags" | "files" | "statuses" | "apiKeys" | "calendars" | "memory" | "schemas" | "stats" | "chat";

export function UserSettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>("profile");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [userMemory, setUserMemory] = useState<string | null>(null);
  const [billingUrl, setBillingUrl] = useState<string | null>(null);
  const [timezone, setTimezone] = useState<string>("UTC");
  const [caldavUrl, setCaldavUrl] = useState<string>("");
  const [copiedToClipboard, setCopiedToClipboard] = useState(false);
  const [chatModel, setChatModel] = useState(() =>
    localStorage.getItem('chatSelectedModel') || "google/gemini-2.5-flash"
  );

  // Get the base URL for API calls
  const getApiBaseUrl = () => {
    return import.meta.env.VITE_API_URL || window.location.origin;
  };

  // Get the calendar feed URL
  const getCalendarFeedUrl = () => {
    const token = user?.caldav_token;
    if (!token) return null;
    return `${getApiBaseUrl()}/api/user/calendar.ics?token=${token}`;
  };

  const navigate = useNavigate();
  const { user, hasSubscription, updateUser, logoutUser } = useAuth();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault(); // Prevent the default form submit action

    // Get the form data
    const formData = new FormData(event.currentTarget as HTMLFormElement);
    const updatedUsername = formData.get("username");
    const updatedEmail = formData.get("email");

    if (!user) {
      setError("User data is not loaded");
      return;
    }

    // Prepare the data to be updated
    const updatedUser = {
      ...user,
      username: updatedUsername,
      email: updatedEmail,
      timezone: timezone,
      caldav_url: caldavUrl || null,
    };

    try {
      await updateUser(updatedUser as User);
      alert("User updated successfully");
      localStorage.setItem("username", updatedUser.username as string);
    } catch (error: any) {
      console.error("Failed to update user:", error);
      setError(error.message);
    }
  }

  const handlePasswordReset = async () => {
    if (!user?.email) return;

    try {
      setIsLoading(true);
      const response = await requestPasswordReset(user.email);
      if (response.error) {
        setError(response.message);
      } else {
        setSuccess("Password reset link has been sent to your email address.");
      }
    } catch (error) {
      setError("Failed to initiate password reset.");
    } finally {
      setIsLoading(false);
    }
  };

  const handleRegenerateToken = async () => {
    if (!confirm("Are you sure? This will invalidate your existing calendar feed URL and any subscribed calendars will need to be updated.")) {
      return;
    }

    try {
      setIsLoading(true);
      const response = await regenerateCalDAVToken();
      // Update the user object with the new token
      if (user) {
        const updatedUser = { ...user, caldav_token: response.token };
        // Force a user refresh by calling the auth context update
        await updateUser(updatedUser as User);
      }
      setSuccess("Calendar feed token regenerated successfully");
    } catch (error: any) {
      setError(error.message || "Failed to regenerate token");
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopyToClipboard = async () => {
    const url = getCalendarFeedUrl();
    if (!url) return;

    try {
      await navigator.clipboard.writeText(url);
      setCopiedToClipboard(true);
      setTimeout(() => setCopiedToClipboard(false), 2000);
    } catch (error) {
      console.error("Failed to copy to clipboard:", error);
    }
  };

  const handleModelChange = (model: string) => {
    setChatModel(model);
    localStorage.setItem('chatSelectedModel', model);
    // Dispatch event for other components (useChat hook) to react to model change
    window.dispatchEvent(new CustomEvent('chatModelChange', { detail: model }));
  };

  const subscriptionEnabled =
    import.meta.env.VITE_FEATURE_SUBSCRIPTION === "true";

  useEffect(() => {
    async function fetchBillingUrl() {
      try {
        const response = await getBillingPortalUrl();
        setBillingUrl(response.url);
      } catch (error) {
        console.error("Failed to fetch billing URL:", error);
      }
    }

    async function fetchUserMemory() {
      try {
        const memory = await getUserMemory();
        setUserMemory(memory.memory);
      } catch (error) {
        console.error("Failed to fetch LLM providers:", error);
      }
    }

    setDocumentTitle("Settings");
    if (subscriptionEnabled) {
      fetchBillingUrl();
    }
    fetchUserMemory();
  }, [subscriptionEnabled]);

  useEffect(() => {
    if (user?.timezone) {
      setTimezone(user.timezone);
    }
    if (user?.caldav_url) {
      setCaldavUrl(user.caldav_url);
    } else {
      setCaldavUrl("");
    }
  }, [user]);

  const renderTabContent = () => {
    switch (activeTab) {
      case "profile":
        return (
          <div className="space-y-6">
            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold mb-4">Profile Settings</h2>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    Username:
                    <input
                      type="text"
                      name="username"
                      defaultValue={user?.username}
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    />
                  </label>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    Email:
                    <input
                      type="email"
                      name="email"
                      defaultValue={user?.email}
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    />
                  </label>
                </div>
                <div>
                  <TimezoneSelector
                    value={timezone}
                    onChange={setTimezone}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    CalDAV URL:
                    <input
                      type="url"
                      name="caldav_url"
                      value={caldavUrl}
                      onChange={(e) => setCaldavUrl(e.target.value)}
                      placeholder="https://calendar.google.com/dav/user@example.com/user"
                      className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    />
                    <p className="mt-1 text-xs text-gray-500">
                      Optional: Enter your CalDAV server URL to sync tasks with external calendars (Google Calendar, Outlook, etc.)
                    </p>
                  </label>
                </div>
                <button
                  type="submit"
                  className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 disabled:opacity-50"
                >
                  Save Changes
                </button>
              </form>
            </div>

            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold mb-4">Calendar Feed (iCal)</h2>
              <p className="text-gray-600 mb-4">
                Subscribe to your tasks in external calendar apps like Google Calendar, Apple Calendar, Outlook, or any other calendar app that supports iCal feeds.
              </p>

              {user?.caldav_token ? (
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Your Calendar Feed URL:
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        readOnly
                        value={getCalendarFeedUrl() || ""}
                        className="flex-1 px-3 py-2 border border-gray-300 rounded-md bg-gray-50 text-sm font-mono"
                      />
                      <button
                        onClick={handleCopyToClipboard}
                        className="px-4 py-2 bg-gray-100 border border-gray-300 rounded-md hover:bg-gray-200 text-sm font-medium"
                      >
                        {copiedToClipboard ? "Copied!" : "Copy"}
                      </button>
                    </div>
                  </div>

                  <div>
                    <button
                      onClick={handleRegenerateToken}
                      disabled={isLoading}
                      className="text-sm text-red-600 hover:text-red-700 underline"
                    >
                      Regenerate Feed Token
                    </button>
                    <p className="mt-1 text-xs text-gray-500">
                      Regenerating will invalidate your existing feed URL. You'll need to resubscribe in your calendar app.
                    </p>
                  </div>

                  <div className="mt-6 pt-6 border-t">
                    <h3 className="text-sm font-semibold text-gray-700 mb-3">Setup Instructions:</h3>
                    <div className="space-y-3 text-sm text-gray-600">
                      <div>
                        <strong className="text-gray-700">Google Calendar:</strong>
                        <ol className="ml-4 mt-1 list-decimal space-y-1">
                          <li>Go to Settings → Add calendar → From URL</li>
                          <li>Paste the feed URL above</li>
                          <li>Click "Add calendar"</li>
                        </ol>
                      </div>
                      <div>
                        <strong className="text-gray-700">Apple Calendar (macOS/iOS):</strong>
                        <ol className="ml-4 mt-1 list-decimal space-y-1">
                          <li>File → New Calendar Subscription… (or "Add Calendar" on iOS)</li>
                          <li>Paste the feed URL above</li>
                          <li>Choose your refresh frequency and click OK</li>
                        </ol>
                      </div>
                      <div>
                        <strong className="text-gray-700">Outlook:</strong>
                        <ol className="ml-4 mt-1 list-decimal space-y-1">
                          <li>Go to Calendar → Add calendar → Subscribe from web</li>
                          <li>Paste the feed URL above</li>
                          <li>Enter a name and click Import</li>
                        </ol>
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-gray-500">
                  <p className="mb-4">Generate a calendar feed token to enable external calendar subscriptions.</p>
                  <button
                    onClick={handleRegenerateToken}
                    disabled={isLoading}
                    className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 disabled:opacity-50"
                  >
                    {isLoading ? "Generating..." : "Generate Feed Token"}
                  </button>
                </div>
              )}
            </div>

            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold mb-4">Password Settings</h2>
              <p className="text-gray-600 mb-4">
                To change your password, we'll send a password reset link to your email address.
              </p>
              <button
                onClick={handlePasswordReset}
                disabled={isLoading}
                className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 disabled:opacity-50"
              >
                {isLoading ? "Sending..." : "Send Password Reset Link"}
              </button>
              {success && <div className="mt-2 text-green-600 text-sm">{success}</div>}
              {error && <div className="mt-2 text-red-600 text-sm">{error}</div>}
            </div>

            {subscriptionEnabled && (
              <div className="bg-white rounded-lg shadow p-6">
                <h2 className="text-xl font-semibold mb-4">Subscription</h2>
                {hasSubscription ? (
                  <div className="space-y-2">
                    <p>
                      Status:{" "}
                      <span className="font-medium">active</span>
                    </p>
                    <a
                      href={billingUrl || "#"}
                      className="text-blue-500 hover:underline"
                    >
                      Manage Subscription →
                    </a>
                  </div>
                ) : (
                  <div className="space-y-2">
                    <p>You don’t have a subscription yet.</p>
                    <button
                      onClick={() => navigate("/app/subscription")}
                      className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
                    >
                      Upgrade to Pro →
                    </button>
                  </div>
                )}
              </div>
            )}

            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold mb-4">Data & Privacy</h2>
              <div className="space-y-4">
                <div className="border border-orange-200 bg-orange-50 rounded-lg p-4">
                  <h3 className="text-lg font-medium text-orange-800 mb-2">Account Deletion & Data Export</h3>
                  <p className="text-orange-700 text-sm mb-3">
                    Need to delete your account or export your data? We're happy to help! Our team can assist you with:
                  </p>
                  <ul className="text-orange-700 text-sm mb-3 ml-4 list-disc space-y-1">
                    <li>Complete account deletion</li>
                    <li>Data export in various formats</li>
                    <li>Selective data removal</li>
                  </ul>
                  <p className="text-orange-700 text-sm mb-3">
                    Please email us at{" "}
                    <a
                      href="mailto:info@zettelgarden.com"
                      className="font-medium text-orange-800 hover:underline"
                    >
                      info@zettelgarden.com
                    </a>{" "}
                    with your request, and we'll take care of it promptly.
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold mb-4">Account Actions</h2>
              <div className="space-y-2">
                <button
                  onClick={() => {
                    logoutUser();
                    navigate('/');
                  }}
                  className="bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600"
                >
                  Logout
                </button>
              </div>
            </div>
          </div>
        );
      case "templates":
        return (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-xl font-semibold mb-4">Card Templates</h2>
            <TemplatesList />
          </div>
        );
      case "tags":
        return <TagList />;
      case "files":
        return <FileVault />;
      case "statuses":
        return <StatusManagement />;
      case "apiKeys":
        return <APIKeysManagement />;
      case "calendars":
        return <CalendarSubscriptions />;
      case "memory":
        return <MemoryPage />;
      case "schemas":
        return <SchemaPage />;
      case "stats":
        return <StatsPage />;
      case "chat":
        return (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-xl font-semibold mb-4">Chat Settings</h2>
            <ModelSelector
              currentModel={chatModel}
              onModelChange={handleModelChange}
            />
          </div>
        );
    }
  };


  return (
    <div className="p-6">
      <H6 children="Settings" />
      <div className="flex border-b">
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "profile" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("profile")}
        >
          Profile
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "templates" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("templates")}
        >
          Templates
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "tags" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("tags")}
        >
          Tags
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "files" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("files")}
        >
          Files
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "statuses" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("statuses")}
        >
          Task Statuses
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "apiKeys" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("apiKeys")}
        >
          API Keys
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "calendars" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("calendars")}
        >
          Calendars
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "chat" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("chat")}
        >
          Chat
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "memory" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("memory")}
        >
          Memory
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "schemas" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("schemas")}
        >
          Schemas
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium ${activeTab === "stats" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("stats")}
        >
          Stats
        </button>
      </div>
      <div className="mt-4">
        {renderTabContent()}
      </div>
    </div>
  );
}
