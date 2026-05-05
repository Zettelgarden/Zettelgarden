import React, { useEffect, useState } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { useNavigate, Link } from "react-router-dom";
import { MenuIcon } from "../../assets/icons/MenuIcon";

import { AdminUserIndex } from "./AdminUserIndex";
import { AdminUserDetailPage } from "./AdminUserDetailPage";
import { AdminEditUserPage } from "./AdminEditUserPage";
import { AdminMailingListPage } from "./AdminMailingListPage";
import { AdminMailingListSendPage } from "./AdminMailingListSendPage";
import { AdminMailingListHistoryPage } from "./AdminMailingListHistoryPage";
import { AdminJobQueuePage } from "./AdminJobQueuePage";
import { AdminSchedulerPage } from "./AdminSchedulerPage";
import { AdminDashboard } from "./AdminDashboard";

import { Routes, Route } from "react-router-dom";

export function Admin() {
  const { isAdmin, isLoading } = useAuth();
  const navigate = useNavigate();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  useEffect(() => {
    if (!isLoading && !isAdmin) {
      // Use window.location to redirect to the root page, since this is a
      // HashRouter app and navigate("/app") would produce /admin#/app instead
      // of /#/app when the user visits /admin directly.
      window.location.href = "/#/app";
    }
  }, [isAdmin, isLoading]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="animate-pulse">Loading...</div>
      </div>
    );
  }
  
  if (!isLoading && !isAdmin) {
    return <div></div>;
  }
  
  return (
    <div className="flex h-screen overflow-hidden">
      {/* Mobile Menu Button */}
      <button
        className="md:hidden fixed top-4 right-4 z-[60] p-3 bg-white rounded-lg shadow-lg hover:bg-gray-50 active:bg-gray-100 safe-top-fixed safe-right-fixed"
        onClick={() => setIsSidebarOpen(!isSidebarOpen)}
        aria-label="Toggle menu"
      >
        <MenuIcon />
      </button>

      {/* Mobile Backdrop */}
      {isSidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 md:hidden z-[45] animate-in fade-in"
          onClick={() => setIsSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <div
        className={`
          fixed md:relative
          w-64
          min-w-[16rem]
          max-w-[16rem]
          flex-shrink-0
          h-screen
          bg-gray-800
          flex flex-col
          transform
          ${isSidebarOpen ? "translate-x-0" : "-translate-x-full"}
          md:translate-x-0
          transition-transform
          duration-300
          ease-in-out
          z-[50]
        `}
      >
        <div className="h-full flex flex-col">
          {/* Sidebar header with close button on mobile */}
          <div className="flex items-center justify-between px-4 py-4 border-b border-gray-700 md:hidden">
            <span className="text-white font-semibold">Admin Menu</span>
            <button
              onClick={() => setIsSidebarOpen(false)}
              className="p-2 text-gray-400 hover:text-white"
              aria-label="Close menu"
            >
              ✕
            </button>
          </div>

          <div className="flex-1 overflow-y-auto py-4">
            <nav className="px-4">
              <ul className="space-y-2">
                <li>
                  <Link
                    to="/admin"
                    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
                    onClick={() => setIsSidebarOpen(false)}
                  >
                    📊 Dashboard
                  </Link>
                </li>
                <li>
                  <Link
                    to="/admin/users"
                    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
                    onClick={() => setIsSidebarOpen(false)}
                  >
                    👥 All Users
                  </Link>
                </li>
                <li>
                  <Link
                    to="/admin/job-queue"
                    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
                    onClick={() => setIsSidebarOpen(false)}
                  >
                    ⚙️ Job Queue
                  </Link>
                </li>
                <li>
                  <Link
                    to="/admin/scheduler"
                    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
                    onClick={() => setIsSidebarOpen(false)}
                  >
                    ⏰ Scheduled Jobs
                  </Link>
                </li>
                <li>
                  <Link
                    to="/admin/mailing-list"
                    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
                    onClick={() => setIsSidebarOpen(false)}
                  >
                    📧 Mailing List Subscribers
                  </Link>
                </li>
                <li>
                  <Link
                    to="/admin/mailing-list/send"
                    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
                    onClick={() => setIsSidebarOpen(false)}
                  >
                    ✉️ Send Message
                  </Link>
                </li>
                <li>
                  <Link
                    to="/admin/mailing-list/history"
                    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-300 hover:text-white transition-colors min-h-[44px] flex items-center"
                    onClick={() => setIsSidebarOpen(false)}
                  >
                    📜 Message History
                  </Link>
                </li>
                <li className="pt-4 border-t border-gray-700 mt-4">
                  <Link
                    to="/app"
                    className="block py-3 px-4 rounded-lg hover:bg-gray-700 text-gray-400 hover:text-white transition-colors min-h-[44px] flex items-center"
                    onClick={() => setIsSidebarOpen(false)}
                  >
                    ← Back to App
                  </Link>
                </li>
              </ul>
            </nav>
          </div>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-hidden">
        <div className="h-full overflow-y-auto">
          <div className="p-4">
            <Routes>
              <Route path="/" element={<AdminDashboard />} />
              <Route path="users" element={<AdminUserIndex />} />
              <Route path="user/:id" element={<AdminUserDetailPage />} />
              <Route path="user/:id/edit" element={<AdminEditUserPage />} />
              <Route path="job-queue" element={<AdminJobQueuePage />} />
              <Route path="scheduler" element={<AdminSchedulerPage />} />
              <Route path="mailing-list" element={<AdminMailingListPage />} />
              <Route path="mailing-list/send" element={<AdminMailingListSendPage />} />
              <Route path="mailing-list/history" element={<AdminMailingListHistoryPage />} />
            </Routes>
          </div>
        </div>
      </div>
    </div>
  );
}
