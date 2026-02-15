import React, { useState, useEffect } from "react";
import { ExternalCalendar } from "../../models/ExternalEvent";
import { getExternalCalendars } from "../../api/externalEvents";
import { createEventOnCalendar } from "../../api/calendarEvents";

interface EventDialogProps {
  initialDate?: Date;
  onClose: () => void;
  onSuccess: () => void;
}

export function EventDialog({ initialDate, onClose, onSuccess }: EventDialogProps) {
  // Form state
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [startDate, setStartDate] = useState<Date>(initialDate || new Date());
  const [startTime, setStartTime] = useState("09:00");
  const [endDate, setEndDate] = useState<Date>(initialDate || new Date());
  const [endTime, setEndTime] = useState("10:00");
  const [allDay, setAllDay] = useState(false);
  const [location, setLocation] = useState("");

  // Calendar selection state
  const [calendars, setCalendars] = useState<ExternalCalendar[]>([]);
  const [selectedCalendar, setSelectedCalendar] = useState<number | null>(null);
  const [isLoadingCalendars, setIsLoadingCalendars] = useState(false);

  // Submit state
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load writable calendars on mount
  useEffect(() => {
    async function loadCalendars() {
      setIsLoadingCalendars(true);
      try {
        const allCalendars = await getExternalCalendars();
        // Filter writable calendars by username field presence
        const writableCalendars = allCalendars.filter(
          (c) => c.username && c.username.trim() !== ""
        );
        setCalendars(writableCalendars);
        // Auto-select first calendar if available
        if (writableCalendars.length > 0) {
          setSelectedCalendar(writableCalendars[0].id);
        }
      } catch (err) {
        console.error("Failed to load calendars:", err);
        setError("Failed to load calendars. Please try again.");
      } finally {
        setIsLoadingCalendars(false);
      }
    }

    loadCalendars();
  }, []);

  // Handle escape key
  useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleKeyPress);
    return () => document.removeEventListener("keydown", handleKeyPress);
  }, [onClose]);

  // Helper function to combine date and time
  const combineDateTime = (date: Date, time: string): Date => {
    const result = new Date(date);
    const [hours, minutes] = time.split(":").map(Number);
    result.setHours(hours, minutes, 0, 0);
    return result;
  };

  // Helper function to format date for input (YYYY-MM-DD)
  const formatDateForInput = (date: Date): string => {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
  };

  // Helper function to parse date from input string
  const parseDateFromInput = (dateStr: string): Date => {
    const [year, month, day] = dateStr.split("-").map(Number);
    return new Date(year, month - 1, day);
  };

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validation
    if (!selectedCalendar) {
      setError("Please select a calendar");
      return;
    }

    if (!title.trim()) {
      setError("Please enter an event title");
      return;
    }

    setIsSaving(true);
    try {
      // Combine date and time for start and end
      const startDateTime = allDay
        ? startDate
        : combineDateTime(startDate, startTime);
      const endDateTime = allDay
        ? endDate
        : combineDateTime(endDate, endTime);

      // Create event
      await createEventOnCalendar(selectedCalendar, {
        title: title.trim(),
        description: description.trim() || undefined,
        start_time: startDateTime.toISOString(),
        end_time: endDateTime.toISOString(),
        all_day: allDay,
        location: location.trim() || undefined,
      });

      // Call onSuccess and close
      onSuccess();
      onClose();
    } catch (err: any) {
      console.error("Failed to create event:", err);
      setError(err.message || "Failed to create event. Please try again.");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="event-dialog-title"
    >
      <div
        className="bg-white rounded-lg shadow-xl max-w-md w-full max-h-[90vh] overflow-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-4 border-b border-slate-200">
          <h3 id="event-dialog-title" className="text-lg font-semibold">
            Create Event
          </h3>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {/* Error message */}
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-md">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          {/* Title */}
          <div>
            <label htmlFor="event-title" className="block text-sm font-medium text-slate-700 mb-1">
              Title <span className="text-red-500">*</span>
            </label>
            <input
              id="event-title"
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Event title"
              disabled={isSaving}
              required
            />
          </div>

          {/* Calendar selection */}
          <div>
            <label htmlFor="event-calendar" className="block text-sm font-medium text-slate-700 mb-1">
              Calendar <span className="text-red-500">*</span>
            </label>
            {isLoadingCalendars ? (
              <div className="text-sm text-slate-500">Loading calendars...</div>
            ) : calendars.length === 0 ? (
              <div className="text-sm text-slate-500">
                No writable calendars available. Please add a calendar with authentication credentials.
              </div>
            ) : (
              <select
                id="event-calendar"
                value={selectedCalendar || ""}
                onChange={(e) => setSelectedCalendar(e.target.value ? Number(e.target.value) : null)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={isSaving}
                required
              >
                {calendars.map((cal) => (
                  <option key={cal.id} value={cal.id}>
                    {cal.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Date */}
          <div>
            <label htmlFor="event-start-date" className="block text-sm font-medium text-slate-700 mb-1">
              Start Date
            </label>
            <input
              id="event-start-date"
              type="date"
              value={formatDateForInput(startDate)}
              onChange={(e) => setStartDate(parseDateFromInput(e.target.value))}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={isSaving}
              required
            />
          </div>

          {/* All-day checkbox */}
          <div className="flex items-center">
            <input
              id="event-all-day"
              type="checkbox"
              checked={allDay}
              onChange={(e) => setAllDay(e.target.checked)}
              className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-slate-300 rounded"
              disabled={isSaving}
            />
            <label htmlFor="event-all-day" className="ml-2 block text-sm text-slate-700">
              All-day event
            </label>
          </div>

          {/* Start time */}
          {!allDay && (
            <div>
              <label htmlFor="event-start-time" className="block text-sm font-medium text-slate-700 mb-1">
                Start Time
              </label>
              <input
                id="event-start-time"
                type="time"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={isSaving}
                required
              />
            </div>
          )}

          {/* End date */}
          <div>
            <label htmlFor="event-end-date" className="block text-sm font-medium text-slate-700 mb-1">
              End Date
            </label>
            <input
              id="event-end-date"
              type="date"
              value={formatDateForInput(endDate)}
              onChange={(e) => setEndDate(parseDateFromInput(e.target.value))}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={isSaving}
              required
            />
          </div>

          {/* End time */}
          {!allDay && (
            <div>
              <label htmlFor="event-end-time" className="block text-sm font-medium text-slate-700 mb-1">
                End Time
              </label>
              <input
                id="event-end-time"
                type="time"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={isSaving}
                required
              />
            </div>
          )}

          {/* Location */}
          <div>
            <label htmlFor="event-location" className="block text-sm font-medium text-slate-700 mb-1">
              Location
            </label>
            <input
              id="event-location"
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Event location"
              disabled={isSaving}
            />
          </div>

          {/* Description */}
          <div>
            <label htmlFor="event-description" className="block text-sm font-medium text-slate-700 mb-1">
              Description
            </label>
            <textarea
              id="event-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
              placeholder="Event description"
              disabled={isSaving}
            />
          </div>

          {/* Buttons */}
          <div className="flex gap-2 pt-2">
            <button
              type="submit"
              disabled={isSaving || !selectedCalendar}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:bg-slate-300 disabled:cursor-not-allowed min-h-[44px]"
            >
              {isSaving ? "Creating..." : "Create Event"}
            </button>
            <button
              type="button"
              onClick={onClose}
              disabled={isSaving}
              className="px-4 py-2 border border-slate-300 text-slate-700 rounded-md hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:bg-slate-100 disabled:cursor-not-allowed min-h-[44px]"
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default EventDialog;
