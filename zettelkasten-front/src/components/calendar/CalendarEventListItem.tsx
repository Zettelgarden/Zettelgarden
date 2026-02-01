import React from "react";
import { ExternalEvent } from "../../models/ExternalEvent";
import { Link } from "react-router-dom";
import { unlinkEventFromCard } from "../../api/externalEvents";
import { format } from "date-fns-tz";
import { useAuth } from "../../contexts/AuthContext";

interface CalendarEventListItemProps {
  event: ExternalEvent;
  onUnlink?: () => void;
}

export function CalendarEventListItem({ event, onUnlink }: CalendarEventListItemProps) {
  const { user } = useAuth();
  const userTimezone = user?.timezone || "UTC";

  const handleUnlink = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (window.confirm("Unlink this event from the card?")) {
      try {
        await unlinkEventFromCard(event.id);
        if (onUnlink) {
          onUnlink();
        }
      } catch (error) {
        console.error("Failed to unlink event:", error);
      }
    }
  };

  const formatDate = (date: Date) => {
    if (event.all_day) {
      return format(date, "MMM d, yyyy", { timeZone: userTimezone });
    }
    return format(date, "MMM d, yyyy h:mm a", { timeZone: userTimezone });
  };

  const formatTimeRange = () => {
    const start = new Date(event.start_time);
    const end = new Date(event.end_time);

    if (event.all_day) {
      return formatDate(start);
    }

    // Same day
    if (start.toDateString() === end.toDateString()) {
      return `${formatDate(start)} - ${format(end, "h:mm a", { timeZone: userTimezone })}`;
    }

    // Different days
    return `${formatDate(start)} - ${formatDate(end)}`;
  };

  return (
    <div className="flex items-center bg-white group">
      {/* Event type indicator */}
      <div className="mr-2.5">
        <span
          className="inline-block w-3 h-3 rounded"
          style={{ backgroundColor: event.color || "#6366f1" }}
          title="Calendar event"
        />
      </div>

      <div className="flex-grow min-w-0">
        {/* Event title */}
        <div className="whitespace-nowrap overflow-hidden text-ellipsis">
          {event.external_url ? (
            <a
              href={event.external_url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-600 hover:text-blue-800"
            >
              {event.title}
            </a>
          ) : (
            <span>{event.title}</span>
          )}
        </div>

        {/* Event details */}
        <div className="flex text-sm text-gray-600 items-center gap-2">
          {/* Time range */}
          <span>{formatTimeRange()}</span>

          {/* Location */}
          {event.location && (
            <>
              <span>•</span>
              <span className="truncate max-w-[200px]">{event.location}</span>
            </>
          )}

          {/* Calendar name (if we had it) */}
          {event.external_calendar_id && (
            <>
              <span>•</span>
              <span className="text-xs text-gray-500">External</span>
            </>
          )}
        </div>

        {/* Description (truncated) */}
        {event.description && (
          <div className="text-xs text-gray-500 truncate mt-1">
            {event.description}
          </div>
        )}
      </div>

      {/* Card link */}
      {event.card && event.card.id > 0 && (
        <div className="ml-2.5">
          <Link
            to={`/app/card/${event.card.id}`}
            className="text-xs text-purple-600 hover:text-purple-800"
          >
            [{event.card.card_id}]
          </Link>
        </div>
      )}

      {/* Unlink button */}
      {event.card_pk != null && onUnlink && (
        <button
          onClick={handleUnlink}
          className="ml-2.5 bg-transparent border-0 cursor-pointer p-2.5 md:p-1 min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center hover:bg-gray-100 rounded transition-colors opacity-0 group-hover:opacity-100"
          aria-label="Unlink event from card"
        >
          ×
        </button>
      )}
    </div>
  );
}
