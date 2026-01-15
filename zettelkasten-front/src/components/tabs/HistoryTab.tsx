import React from "react";
import {
  AuditChange,
  parseAuditEvent,
  getEventIcon,
  formatDate,
  renderAuditDiff,
} from "../../utils/audit";

interface HistoryTabProps {
  auditEvents: any[];
}

export function HistoryTab({ auditEvents }: HistoryTabProps) {
  return (
    <div className="p-4">
      <div className="space-y-4 mt-4">
        {auditEvents.map((event, index) => {
          const changes: AuditChange[] = parseAuditEvent(event);
          const eventType = event.details?.change_type || 'unknown';
          return (
            <div key={index} className="bg-white border border-gray-200 p-4 rounded-lg shadow-sm">
              <div className="flex items-start space-x-3">
                <div className="flex-shrink-0 mt-1">
                  {getEventIcon(eventType)}
                </div>
                <div className="flex-grow">
                  <div className="flex justify-between items-start">
                    <div>
                      <span className="font-medium text-gray-900 capitalize">
                        {eventType.toLowerCase()}
                      </span>
                      <span className="text-gray-600 ml-2">by User {event.user_id}</span>
                    </div>
                    <span className="text-sm text-gray-500">{formatDate(event.created_at)}</span>
                  </div>
                  {changes.length > 0 && (
                    <div className="mt-3 space-y-3">
                      {changes.map((change, idx) => (
                        <div key={idx}>
                          {renderAuditDiff(change)}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          );
        })}
        {auditEvents.length === 0 && (
          <div className="text-center text-gray-500 py-8">
            No audit events found
          </div>
        )}
      </div>
    </div>
  );
}