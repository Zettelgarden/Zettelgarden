import React from "react";

export interface AuditChange {
  field: string;
  from: any;
  to: any;
}

export interface AuditEventWithChanges {
  id: number;
  user_id: number;
  entity_id: number;
  entity_type: string;
  action: string;
  details: {
    change_type: string;
    changes: Record<string, { from: any; to: any }>;
    custom_data?: Record<string, any>;
  };
  created_at: Date;
  updated_at: Date;
}

export function formatFieldName(field: string): string {
  return field.charAt(0).toUpperCase() + field.slice(1).replace(/([A-Z])/g, " $1");
}

/**
 * Generate a human-readable summary of changes
 * e.g., "Updated title and body", "Updated title", "Updated 3 fields"
 */
export function generateChangeSummary(changes: AuditChange[], eventType: string): string {
  if (changes.length === 0) {
    switch (eventType.toLowerCase()) {
      case 'create':
        return 'Card created';
      case 'delete':
        return 'Card deleted';
      default:
        return 'No changes';
    }
  }

  if (changes.length === 1) {
    return `Updated ${formatFieldName(changes[0].field).toLowerCase()}`;
  }

  if (changes.length === 2) {
    return `Updated ${formatFieldName(changes[0].field).toLowerCase()} and ${formatFieldName(changes[1].field).toLowerCase()}`;
  }

  return `Updated ${changes.length} fields`;
}

/**
 * Group events by date category: Today, Yesterday, This Week, Older
 */
export function groupEventsByDate(events: any[]): Record<string, any[]> {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);
  const thisWeek = new Date(today);
  thisWeek.setDate(thisWeek.getDate() - 7);

  const groups: Record<string, any[]> = {
    Today: [],
    Yesterday: [],
    'This Week': [],
    Older: []
  };

  for (const event of events) {
    const eventDate = new Date(event.created_at);
    const eventDay = new Date(eventDate.getFullYear(), eventDate.getMonth(), eventDate.getDate());

    if (eventDay.getTime() === today.getTime()) {
      groups.Today.push(event);
    } else if (eventDay.getTime() === yesterday.getTime()) {
      groups.Yesterday.push(event);
    } else if (eventDay >= thisWeek) {
      groups['This Week'].push(event);
    } else {
      groups.Older.push(event);
    }
  }

  return groups;
}

/**
 * Render inline diff with word-level highlighting for string changes
 */
export function renderInlineDiff(from: string, to: string, maxLength: number = 200): React.ReactNode {
  const truncate = (text: string) => {
    if (text.length <= maxLength) return text;
    return text.slice(0, maxLength) + '...';
  };

  const fromText = truncate(from || '(empty)');
  const toText = truncate(to || '(empty)');

  // For simplicity, just show the old/new with styling
  // A full word-level diff would require a diffing library
  return (
    <div className="flex flex-col space-y-1">
      <div className="text-red-600 bg-red-50 px-2 py-1 rounded break-words">
        {fromText}
      </div>
      <div className="text-green-600 bg-green-50 px-2 py-1 rounded break-words">
        {toText}
      </div>
    </div>
  );
}

export function renderAuditDiff(change: AuditChange) {
  const fieldName = formatFieldName(change.field);

  if (typeof change.from === "string" && typeof change.to === "string") {
    return (
      <div className="flex flex-col space-y-1">
        <div className="text-sm font-medium text-gray-700">{fieldName}</div>
        <div className="flex flex-col space-y-1 pl-4">
          <div className="text-red-600 line-through bg-red-50 px-2 py-1 rounded">
            {change.from || "(empty)"}
          </div>
          <div className="text-green-600 bg-green-50 px-2 py-1 rounded">
            {change.to || "(empty)"}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col space-y-1">
      <div className="text-sm font-medium text-gray-700">{fieldName}</div>
      <div className="text-gray-600 pl-4">
        Changed from{" "}
        <code className="bg-gray-100 px-1 rounded">{JSON.stringify(change.from)}</code>
        {" "}to{" "}
        <code className="bg-gray-100 px-1 rounded">{JSON.stringify(change.to)}</code>
      </div>
    </div>
  );
}

export function parseAuditEvent(event: any): AuditChange[] {
  const changes: AuditChange[] = [];

  if (event.details?.changes) {
    Object.entries(event.details.changes).forEach(([field, values]: [string, any]) => {
      if (typeof values === 'object' && values !== null) {
        if ('from' in values && 'to' in values) {
          changes.push({
            field,
            from: values.from,
            to: values.to
          });
        } else {
          Object.entries(values).forEach(([subField, subValues]: [string, any]) => {
            if (typeof subValues === 'object' && subValues !== null && 'from' in subValues && 'to' in subValues) {
              changes.push({
                field: `${field}.${subField}`,
                from: subValues.from,
                to: subValues.to
              });
            }
          });
        }
      }
    });
  }

  return changes;
}

export function getEventIcon(eventType: string) {
  switch (eventType.toLowerCase()) {
    case 'update':
      return (
        <svg className="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
        </svg>
      );
    case 'create':
      return (
        <svg className="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
        </svg>
      );
    case 'delete':
      return (
        <svg className="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
      );
    default:
      return (
        <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
  }
}

export function formatDate(date: Date | string) {
  const d = typeof date === "string" ? new Date(date) : date;
  return new Intl.DateTimeFormat('default', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(d);
}