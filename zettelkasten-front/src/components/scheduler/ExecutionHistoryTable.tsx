import React, { useState } from "react";
import { JobRun } from "../../api/admin";
import { formatDuration } from "../../utils/scheduler";

interface ExecutionHistoryTableProps {
  runs: JobRun[];
}

export function ExecutionHistoryTable({ runs }: ExecutionHistoryTableProps) {
  const [expandedError, setExpandedError] = useState<number | null>(null);

  if (runs.length === 0) {
    return (
      <div className="py-8 text-center text-gray-500 text-sm">
        This job has never run
      </div>
    );
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  const getStatusBadge = (status: string) => {
    const styles = {
      completed: "bg-green-100 text-green-800",
      failed: "bg-red-100 text-red-800",
      running: "bg-yellow-100 text-yellow-800",
    };
    return (
      <span
        className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${styles[status as keyof typeof styles] || styles.running}`}
      >
        {status}
      </span>
    );
  };

  return (
    <table className="w-full text-sm">
      <thead className="bg-gray-50">
        <tr>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Started</th>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Duration</th>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
          <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">Retries</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-200">
        {runs.map((run) => {
          const duration =
            run.completed_at && run.started_at
              ? new Date(run.completed_at).getTime() - new Date(run.started_at).getTime()
              : null;

          return (
            <React.Fragment key={run.id}>
              <tr className="hover:bg-gray-50">
                <td className="px-3 py-2 text-gray-900">{run.id}</td>
                <td className="px-3 py-2 text-gray-600">{formatDate(run.started_at)}</td>
                <td className="px-3 py-2 text-gray-600">
                  {duration !== null ? formatDuration(duration) : "-"}
                </td>
                <td className="px-3 py-2">{getStatusBadge(run.status)}</td>
                <td className="px-3 py-2 text-gray-600">{run.retry_count}</td>
              </tr>
              {run.status === "failed" && run.error_message && (
                <tr>
                  <td colSpan={5} className="px-3 py-2 bg-red-50">
                    <button
                      onClick={() => setExpandedError(expandedError === run.id ? null : run.id)}
                      className="text-xs text-red-700 hover:text-red-900 font-medium"
                    >
                      {expandedError === run.id ? "Hide" : "Show"} error message
                    </button>
                    {expandedError === run.id && (
                      <pre className="mt-2 text-xs text-red-800 whitespace-pre-wrap font-mono bg-red-100 p-2 rounded">
                        {run.error_message}
                      </pre>
                    )}
                  </td>
                </tr>
              )}
            </React.Fragment>
          );
        })}
      </tbody>
    </table>
  );
}
