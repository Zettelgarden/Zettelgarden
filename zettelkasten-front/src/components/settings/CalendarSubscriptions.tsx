import React, { useState, useEffect } from "react";
import { FaPlus, FaSync, FaTrash, FaLock } from "react-icons/fa";
import {
  ExternalCalendar,
  CreateExternalCalendarRequest,
} from "../../models/ExternalEvent";
import {
  getExternalCalendars,
  createExternalCalendar,
  updateExternalCalendar,
  deleteExternalCalendar,
  syncExternalCalendar,
} from "../../api/externalEvents";

interface CalendarSubscriptionsProps {
  onCalendarChange?: () => void;
}

export function CalendarSubscriptions({ onCalendarChange }: CalendarSubscriptionsProps) {
  const [calendars, setCalendars] = useState<ExternalCalendar[]>([]);
  const [showAddForm, setShowAddForm] = useState(false);
  const [syncing, setSyncing] = useState<Set<number>>(new Set());
  const [deleting, setDeleting] = useState<Set<number>>(new Set());
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    loadCalendars();
  }, []);

  // Auto-dismiss success messages
  useEffect(() => {
    if (success) {
      const timer = setTimeout(() => setSuccess(null), 3000);
      return () => clearTimeout(timer);
    }
  }, [success]);

  async function loadCalendars() {
    try {
      setError(null);
      const data = await getExternalCalendars();
      setCalendars(data);
    } catch (err) {
      setError("Failed to load calendar subscriptions");
      console.error(err);
    }
  }

  async function handleAdd(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const usernameInput = form.elements.namedItem("username") as HTMLInputElement;
    const passwordInput = form.elements.namedItem("password") as HTMLInputElement;
    const data: CreateExternalCalendarRequest = {
      name: (form.elements.namedItem("name") as HTMLInputElement).value,
      url: (form.elements.namedItem("url") as HTMLInputElement).value,
      username: usernameInput.value || undefined,
      password: passwordInput.value || undefined,
      color: (form.elements.namedItem("color") as HTMLInputElement).value || "#6366f1",
    };

    try {
      setError(null);
      await createExternalCalendar(data);
      setShowAddForm(false);
      setSuccess("Calendar subscription added successfully");
      await loadCalendars();
      onCalendarChange?.();
    } catch (err: any) {
      setError(err.message || "Failed to add calendar subscription");
    }
  }

  async function handleSync(id: number) {
    setSyncing(prev => new Set(prev).add(id));
    setError(null);
    try {
      await syncExternalCalendar(id);
      setSuccess("Calendar synced successfully");
      await loadCalendars();
      onCalendarChange?.();
    } catch (err: any) {
      setError(err.message || "Failed to sync calendar");
    } finally {
      setSyncing(prev => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  }

  async function handleDelete(id: number) {
    if (!confirm("Are you sure you want to remove this calendar subscription? All imported events will be removed.")) {
      return;
    }

    setDeleting(prev => new Set(prev).add(id));
    setError(null);
    try {
      await deleteExternalCalendar(id);
      setSuccess("Calendar subscription removed");
      await loadCalendars();
      onCalendarChange?.();
    } catch (err: any) {
      setError(err.message || "Failed to remove calendar");
    } finally {
      setDeleting(prev => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  }

  function formatDate(dateStr: string | undefined) {
    if (!dateStr) return null;
    const d = new Date(dateStr);
    return d.toLocaleString();
  }

  return (
    <div className="bg-white border border-slate-300 rounded-lg overflow-hidden">
      <div className="bg-slate-100 px-4 py-3 border-b border-slate-300 flex justify-between items-center">
        <h2 className="text-lg font-semibold text-slate-800">Calendar Subscriptions</h2>
        <button
          onClick={() => setShowAddForm(true)}
          className="px-3 py-1.5 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 flex items-center gap-2 min-h-[36px]"
        >
          <FaPlus size={14} aria-hidden="true" />
          Add Calendar
        </button>
      </div>

      <div className="p-4">
        {/* Error/Success Messages */}
        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded">
            {error}
            <button onClick={() => setError(null)} className="float-right hover:underline">Dismiss</button>
          </div>
        )}
        {success && (
          <div className="mb-4 p-3 bg-green-50 border border-green-200 text-green-700 rounded">
            {success}
          </div>
        )}

        {/* Add Form */}
        {showAddForm && (
          <form onSubmit={handleAdd} className="mb-6 p-4 border rounded bg-slate-50">
            <h3 className="font-medium mb-3">Subscribe to Calendar</h3>
            <div className="space-y-3">
              <div>
                <label htmlFor="name" className="block text-sm font-medium text-slate-700 mb-1">Name</label>
                <input
                  name="name"
                  placeholder="e.g., Google Calendar"
                  required
                  className="w-full px-3 py-2 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label htmlFor="url" className="block text-sm font-medium text-slate-700 mb-1">iCal URL</label>
                <input
                  name="url"
                  placeholder="https://calendar.google.com/calendar/ical/..."
                  type="url"
                  required
                  className="w-full px-3 py-2 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-slate-500 mt-1">
                  Enter the public iCal feed URL from your calendar provider
                </p>
              </div>
              <div>
                <label htmlFor="username" className="block text-sm font-medium text-slate-700 mb-1">Username (optional)</label>
                <input
                  name="username"
                  placeholder="Username for authentication"
                  type="text"
                  className="w-full px-3 py-2 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-slate-500 mt-1">
                  Only required for password-protected calendars
                </p>
              </div>
              <div>
                <label htmlFor="password" className="block text-sm font-medium text-slate-700 mb-1">Password (optional)</label>
                <input
                  name="password"
                  placeholder="Password for authentication"
                  type="password"
                  className="w-full px-3 py-2 border border-slate-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex items-center gap-2">
                <input
                  name="color"
                  type="color"
                  defaultValue="#6366f1"
                  className="w-16 h-10 border border-slate-300 rounded cursor-pointer"
                />
                <span className="text-sm text-slate-700">Event color</span>
              </div>
              <div className="flex gap-2">
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                >
                  Subscribe
                </button>
                <button
                  type="button"
                  onClick={() => setShowAddForm(false)}
                  className="px-4 py-2 border border-slate-300 rounded hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                >
                  Cancel
                </button>
              </div>
            </div>
          </form>
        )}

        {/* Calendar List */}
        <div className="space-y-3">
          {calendars && calendars.map(cal => (
            <div key={cal.id} className="flex items-center justify-between p-3 border border-slate-200 rounded">
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <div
                  className="w-4 h-4 rounded flex-shrink-0"
                  style={{ backgroundColor: cal.color }}
                  aria-hidden="true"
                />
                <div className="min-w-0 flex-1">
                  <div className="font-medium text-slate-800 truncate flex items-center gap-2">
                    {cal.name}
                    {cal.username && (
                      <FaLock size={12} className="text-slate-500" title="Password protected" aria-label="Password protected calendar" />
                    )}
                  </div>
                  <div className="text-sm text-slate-500 truncate">{cal.url}</div>
                  {cal.last_synced_at && (
                    <div className="text-xs text-slate-400">
                      Last synced: {formatDate(cal.last_synced_at)}
                    </div>
                  )}
                  {cal.last_error && (
                    <div className="text-xs text-red-500 truncate" title={cal.last_error}>
                      Error: {cal.last_error}
                    </div>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-2 ml-4">
                <button
                  onClick={() => handleSync(cal.id)}
                  disabled={syncing.has(cal.id)}
                  className="p-2 text-blue-600 hover:bg-blue-50 rounded min-h-[36px] min-w-[36px] focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  title={syncing.has(cal.id) ? "Syncing..." : "Sync now"}
                  aria-label={syncing.has(cal.id) ? "Syncing calendar" : "Sync calendar"}
                >
                  <FaSync size={14} className={syncing.has(cal.id) ? "animate-spin" : ""} aria-hidden="true" />
                </button>
                <button
                  onClick={() => handleDelete(cal.id)}
                  disabled={deleting.has(cal.id)}
                  className="p-2 text-red-600 hover:bg-red-50 rounded min-h-[36px] min-w-[36px] focus:outline-none focus:ring-2 focus:ring-red-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  title="Remove subscription"
                  aria-label="Remove calendar subscription"
                >
                  <FaTrash size={14} aria-hidden="true" />
                  <span className="sr-only">Remove</span>
                </button>
              </div>
            </div>
          ))}

          {calendars && calendars.length === 0 && !showAddForm && (
            <div className="text-center py-8 text-slate-500">
              <p className="mb-2">No calendar subscriptions yet.</p>
              <p className="text-sm">Subscribe to your Google Calendar, Outlook, or other iCal feeds to see events alongside your tasks.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
