import React, { useState, useEffect, FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { getUserMemory } from "../api/users";
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

type Tab = "profile" | "templates" | "tags" | "files" | "statuses" | "apiKeys";

export function UserSettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>("profile");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [userMemory, setUserMemory] = useState<string | null>(null);
  const [billingUrl, setBillingUrl] = useState<string | null>(null);
  const [timezone, setTimezone] = useState<string>("UTC");

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
                <div className="pt-2">
                  <button
                    onClick={() => navigate('/app/schemas')}
                    className="text-blue-500 hover:underline text-sm"
                  >
                    Manage Schemas →
                  </button>
                </div>
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
      </div>
      <div className="mt-4">
        {renderTabContent()}
      </div>
    </div>
  );
}
