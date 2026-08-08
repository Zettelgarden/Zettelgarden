import React, { useState, useMemo } from 'react';
import {
  AuditChange,
  parseAuditEvent,
  getEventIcon,
  formatDate,
  renderAuditDiff,
  generateChangeSummary,
  groupEventsByDate,
} from '../../utils/audit';

interface HistoryTabProps {
  auditEvents: any[];
  onRestore?: (event: any) => void;
}

interface HistoryEventProps {
  event: any;
  changes: AuditChange[];
  isExpanded: boolean;
  onToggleExpand: () => void;
  onRestore?: (event: any) => void;
}

function HistoryEvent({
  event,
  changes,
  isExpanded,
  onToggleExpand,
  onRestore,
}: HistoryEventProps) {
  const eventType = event.details?.change_type || 'unknown';
  const changeSummary = generateChangeSummary(changes, eventType);

  return (
    <div className="relative">
      {/* Timeline connector line */}
      <div className="absolute left-[19px] -bottom-6 w-0.5 h-6 bg-gray-200 last:hidden" />

      <div className="flex items-start space-x-4">
        {/* Icon with circular background */}
        <div className="flex-shrink-0 relative z-10">
          <div className="w-10 h-10 rounded-full bg-white border-2 border-gray-200 flex items-center justify-center shadow-sm">
            {getEventIcon(eventType)}
          </div>
        </div>

        {/* Event content */}
        <div className="flex-grow min-w-0 pb-6">
          <div
            className={`bg-white border rounded-lg shadow-sm overflow-hidden transition-all ${
              isExpanded ? 'border-gray-300' : 'border-gray-200'
            }`}
          >
            {/* Always-visible summary header */}
            <div
              className="p-3 cursor-pointer hover:bg-gray-50 transition-colors"
              onClick={onToggleExpand}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <span className="font-medium text-gray-900 capitalize">
                    {eventType.toLowerCase()}
                  </span>
                  <span className="text-sm text-gray-600">{changeSummary}</span>
                  {changes.length > 0 && (
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
                      {changes.length}{' '}
                      {changes.length === 1 ? 'change' : 'changes'}
                    </span>
                  )}
                </div>
                <div className="flex items-center space-x-3">
                  <span className="text-xs text-gray-500 whitespace-nowrap">
                    {formatDate(event.created_at)}
                  </span>
                  <button className="text-gray-400 hover:text-gray-600">
                    {isExpanded ? (
                      <svg
                        className="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M5 15l7-7 7 7"
                        />
                      </svg>
                    ) : (
                      <svg
                        className="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M19 9l-7 7-7-7"
                        />
                      </svg>
                    )}
                  </button>
                </div>
              </div>
            </div>

            {/* Expandable details section */}
            {isExpanded && (
              <div className="border-t border-gray-100 p-4 space-y-4">
                {changes.length > 0 ? (
                  changes.map((change, idx) => (
                    <div key={idx} className="pl-3 border-l-2 border-gray-200">
                      {renderAuditDiff(change)}
                    </div>
                  ))
                ) : (
                  <div className="text-sm text-gray-500 italic">
                    {eventType.toLowerCase() === 'create'
                      ? 'Initial card creation'
                      : eventType.toLowerCase() === 'delete'
                      ? 'Card was deleted'
                      : 'No field changes recorded'}
                  </div>
                )}

                {/* Restore button for non-create events */}
                {onRestore &&
                  eventType.toLowerCase() !== 'create' &&
                  changes.length > 0 && (
                    <div className="pt-2 mt-2 border-t border-gray-100">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onRestore(event);
                        }}
                        className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 hover:text-gray-900 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                      >
                        <svg
                          className="w-4 h-4 mr-1.5"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"
                          />
                        </svg>
                        Restore to this version
                      </button>
                    </div>
                  )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export function HistoryTab({ auditEvents, onRestore }: HistoryTabProps) {
  const [expandedEvents, setExpandedEvents] = useState<Set<number>>(new Set());

  const groupedEvents = useMemo(
    () => groupEventsByDate(auditEvents),
    [auditEvents],
  );

  const toggleExpand = (eventId: number) => {
    setExpandedEvents((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(eventId)) {
        newSet.delete(eventId);
      } else {
        newSet.add(eventId);
      }
      return newSet;
    });
  };

  const expandAll = () => {
    setExpandedEvents(new Set(auditEvents.map((e) => e.id)));
  };

  const collapseAll = () => {
    setExpandedEvents(new Set());
  };

  // Get all non-empty group names in order
  const groupNames: Array<'Today' | 'Yesterday' | 'This Week' | 'Older'> = [
    'Today',
    'Yesterday',
    'This Week',
    'Older',
  ];

  if (auditEvents.length === 0) {
    return (
      <div className="p-4">
        <div className="text-center text-gray-500 py-8">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <p className="mt-2">No audit events found</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-4">
      {/* Header with expand/collapse controls */}
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Card History</h3>
        <div className="flex space-x-2">
          <button
            onClick={expandAll}
            className="text-sm text-blue-600 hover:text-blue-700"
          >
            Expand all
          </button>
          <span className="text-gray-300">|</span>
          <button
            onClick={collapseAll}
            className="text-sm text-blue-600 hover:text-blue-700"
          >
            Collapse all
          </button>
        </div>
      </div>

      {/* Timeline events grouped by date */}
      <div className="space-y-6">
        {groupNames.map((groupName) => {
          const events = groupedEvents[groupName];
          if (events.length === 0) return null;

          return (
            <div key={groupName}>
              {/* Group header */}
              <h4 className="text-sm font-medium text-gray-500 uppercase tracking-wide mb-3">
                {groupName}
              </h4>

              {/* Events in this group */}
              <div className="space-y-0">
                {events.map((event) => {
                  const changes: AuditChange[] = parseAuditEvent(event);
                  return (
                    <HistoryEvent
                      key={event.id}
                      event={event}
                      changes={changes}
                      isExpanded={expandedEvents.has(event.id)}
                      onToggleExpand={() => toggleExpand(event.id)}
                      onRestore={onRestore}
                    />
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
