import React, { useState, useEffect, FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { getBillingPortalUrl, getBillingStatus } from "../api/billing";
import { exportUserData, deleteAccount } from "../api/account";
import { requestPasswordReset } from "../api/auth";
import { User, EditUserParams } from "../models/User";
import { useAuth } from "../contexts/AuthContext";
import { H6 } from "../components/Header";
import { TemplatesList } from "../components/templates/TemplatesList";
import { setDocumentTitle } from "../utils/title";
import { TagList } from "../components/tags/TagList";
import { StatusManagement } from "../components/settings/StatusManagement";
import { TimezoneSelector } from "../components/settings/TimezoneSelector";
import ToggleSlider from "../components/ToggleSlider";
import APIKeysManagement from "../components/settings/APIKeysManagement";
import { EntityPage } from "./EntityPage";
import { StatsPage } from "./StatsPage";

type Tab = "profile" | "templates" | "tags" | "statuses" | "apiKeys" | "entities" | "stats";

export function UserSettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>("profile");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [billingUrl, setBillingUrl] = useState<string | null>(null);
  const [billingEnabled, setBillingEnabled] = useState<boolean>(true);
  const [timezone, setTimezone] = useState<string>("UTC");
  const [showTasks, setShowTasks] = useState<boolean>(true);
  const [showRss, setShowRss] = useState<boolean>(true);

  const navigate = useNavigate();
  const { user, hasSubscription, updateUser, logoutUser } = useAuth();

  const [exporting, setExporting] = useState(false);
  const [exportMsg, setExportMsg] = useState<string | null>(null);
  const [exportError, setExportError] = useState<string | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  async function handleExport() {
    setExporting(true);
    setExportMsg(null);
    setExportError(null);
    try {
      await exportUserData();
      setExportMsg("Export downloaded.");
    } catch (err: any) {
      setExportError(err.message || "Export failed.");
    } finally {
      setExporting(false);
    }
  }

  async function handleDelete() {
    setDeleting(true);
    setDeleteError(null);
    try {
      await deleteAccount(deletePassword);
      // Account and its data are gone server-side; drop the local session.
      logoutUser();
      navigate("/");
    } catch (err: any) {
      setDeleteError(err.message || "Failed to delete account.");
      setDeleting(false);
    }
  }

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
      show_tasks: showTasks,
      show_rss: showRss,
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

  const subscriptionEnabled =
    import.meta.env.VITE_FEATURE_SUBSCRIPTION === "true";

  useEffect(() => {
    async function fetchBilling() {
      try {
        const status = await getBillingStatus();
        setBillingEnabled(status.enabled);
        if (status.enabled) {
          const response = await getBillingPortalUrl();
          setBillingUrl(response.url);
        }
      } catch (error) {
        console.error("Failed to fetch billing info:", error);
      }
    }

    setDocumentTitle("Settings");
    if (subscriptionEnabled) {
      fetchBilling();
    }
  }, [subscriptionEnabled]);

  useEffect(() => {
    if (user?.timezone) {
      setTimezone(user.timezone);
    }
    if (user?.show_tasks !== undefined) {
      setShowTasks(user.show_tasks);
    }
    if (user?.show_rss !== undefined) {
      setShowRss(user.show_rss);
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
                <div className="pt-2">
                  <h3 className="text-sm font-medium text-gray-700 mb-2">Sidebar Visibility</h3>
                  <div className="space-y-3">
                    <ToggleSlider
                      label="Show Tasks"
                      initialState={user?.show_tasks !== false}
                      onToggle={setShowTasks}
                    />
                    <ToggleSlider
                      label="Show RSS"
                      initialState={user?.show_rss !== false}
                      onToggle={setShowRss}
                    />
                  </div>
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

            {subscriptionEnabled && billingEnabled && (
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
                    <p>You don't have a subscription yet.</p>
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
                <div className="border border-gray-200 rounded-lg p-4">
                  <h3 className="text-lg font-medium text-gray-800 mb-2">Export your data</h3>
                  <p className="text-gray-600 text-sm mb-3">
                    Download a zip archive of your cards, tasks, files, tags,
                    entities, facts, and RSS data (as JSON, with Markdown/CSV
                    renderings of your cards and the original file bytes).
                  </p>
                  <button
                    onClick={handleExport}
                    disabled={exporting}
                    className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 disabled:opacity-50"
                  >
                    {exporting ? "Preparing export..." : "Export my data"}
                  </button>
                  {exportMsg && (
                    <p className="mt-2 text-sm text-green-600">{exportMsg}</p>
                  )}
                  {exportError && (
                    <p className="mt-2 text-sm text-red-600">{exportError}</p>
                  )}
                </div>

                <div className="border border-red-200 bg-red-50 rounded-lg p-4">
                  <h3 className="text-lg font-medium text-red-800 mb-2">Delete account</h3>
                  <p className="text-red-700 text-sm mb-3">
                    Permanently delete your account and all associated data
                    (cards, tasks, files, and more). This cannot be undone.
                  </p>
                  {!showDeleteConfirm ? (
                    <button
                      onClick={() => setShowDeleteConfirm(true)}
                      className="bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700"
                    >
                      Delete account
                    </button>
                  ) : (
                    <div className="space-y-3">
                      <p className="text-red-700 text-sm">
                        Type your password to confirm (GitHub/OIDC accounts can
                        leave it empty).
                      </p>
                      <input
                        type="password"
                        value={deletePassword}
                        onChange={(e) => setDeletePassword(e.target.value)}
                        placeholder="Password"
                        className="w-full max-w-xs px-3 py-2 border border-red-300 rounded focus:outline-none focus:ring-2 focus:ring-red-500"
                      />
                      <div className="flex gap-2">
                        <button
                          onClick={handleDelete}
                          disabled={deleting}
                          className="bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700 disabled:opacity-50"
                        >
                          {deleting ? "Deleting..." : "Permanently delete"}
                        </button>
                        <button
                          onClick={() => {
                            setShowDeleteConfirm(false);
                            setDeletePassword("");
                          }}
                          className="bg-gray-300 text-gray-700 px-4 py-2 rounded hover:bg-gray-400"
                        >
                          Cancel
                        </button>
                      </div>
                      {deleteError && (
                        <p className="text-sm text-red-700">{deleteError}</p>
                      )}
                    </div>
                  )}
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
      case "statuses":
        return <StatusManagement />;
      case "apiKeys":
        return <APIKeysManagement />;
      case "entities":
        return <EntityPage />;
      case "stats":
        return <StatsPage />;
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
          className={`px-4 py-2 text-sm font-medium ${activeTab === "entities" ? "border-b-2 border-blue-500 text-blue-600" : "text-gray-500 hover:text-gray-700"}`}
          onClick={() => setActiveTab("entities")}
        >
          Entities
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
