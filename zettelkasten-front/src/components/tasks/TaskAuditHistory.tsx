import { format } from "date-fns";
import { TaskAuditEvent } from "../../models/Task";
import { formatAuditEvent } from "../../utils/tasks";

interface TaskAuditHistoryProps {
  events: TaskAuditEvent[];
}

export function TaskAuditHistory({ events }: TaskAuditHistoryProps) {
  return (
    <div className="mt-6 border-t pt-4">
      <h3 className="text-lg font-medium text-gray-900 mb-4">Task History</h3>
      <div className="space-y-3 max-h-[200px] overflow-y-auto">
        {events.length > 0 ? (
          events.map((event) => (
            <div
              key={event.id}
              className="flex items-start space-x-3 text-sm hover:bg-gray-50 p-2 rounded"
            >
              <div className="text-gray-500 min-w-[120px] font-medium">
                {format(event.created_at, "MMM d, HH:mm")}
              </div>
              <div className="flex-grow text-gray-700">{formatAuditEvent(event)}</div>
            </div>
          ))
        ) : (
          <div className="text-sm text-gray-500 text-center py-4">No history available</div>
        )}
      </div>
    </div>
  );
}
