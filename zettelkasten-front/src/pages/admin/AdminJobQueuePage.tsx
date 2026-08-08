import React, { useEffect, useState } from 'react';
import { getAllJobs, retryJob, AdminJob } from '../../api/admin';
import { AdminErrorDisplay } from '../../components/admin/AdminErrorDisplay';

interface ErrorState {
  message: string;
  details?: string;
}

// Count of recent jobs to load for the audit log.
const PAGE_SIZE = 50;

const getJobTypeIcon = (jobType: string) => {
  switch (jobType) {
    case 'summarization':
      return '📝';
    case 'entity_extraction':
    case 'fact_entity_extraction':
      return '🏷️';
    case 'chat':
      return '💬';
    case 'file_text_extraction':
      return '📎';
    default:
      return '⚙️';
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case 'running':
      return 'bg-blue-100 text-blue-800';
    case 'completed':
      return 'bg-green-100 text-green-800';
    case 'failed':
      return 'bg-red-100 text-red-800';
    case 'cancelled':
      return 'bg-gray-100 text-gray-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
};

function CountCard({
  label,
  value,
  accent,
  icon,
}: {
  label: string;
  value: number;
  accent: string;
  icon: string;
}) {
  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-5">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-600 font-medium">{label}</p>
          <p className={`text-2xl font-bold mt-1 ${accent}`}>{value}</p>
        </div>
        <div className="text-3xl">{icon}</div>
      </div>
    </div>
  );
}

export function AdminJobQueuePage() {
  const [jobs, setJobs] = useState<AdminJob[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<ErrorState | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);
  const [retryingId, setRetryingId] = useState<number | null>(null);

  const fetchJobs = async (showRefreshing = false) => {
    if (showRefreshing) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }
    setError(null);
    try {
      const data = await getAllJobs({ limit: PAGE_SIZE });
      setJobs(data.jobs);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Failed to load job audit log';
      setError({
        message,
        details: err instanceof Error ? err.stack : undefined,
      });
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    fetchJobs();
    // Auto-refresh every 10 seconds.
    const interval = setInterval(() => fetchJobs(true), 10000);
    return () => clearInterval(interval);
  }, []);

  const handleRetry = async (jobId: number) => {
    setActionMessage(null);
    setRetryingId(jobId);
    try {
      await retryJob(jobId);
      setActionMessage(`Re-running job ${jobId}`);
      fetchJobs(true);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : `Failed to retry job ${jobId}`;
      setError({
        message,
        details: err instanceof Error ? err.stack : undefined,
      });
    } finally {
      setRetryingId(null);
    }
  };

  const runningCount = jobs.filter((j) => j.status === 'running').length;
  const completedCount = jobs.filter((j) => j.status === 'completed').length;
  const failedCount = jobs.filter((j) => j.status === 'failed').length;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-pulse text-gray-600">
          Loading job audit log...
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            LLM Job Audit Log
          </h1>
          <p className="text-gray-600 mt-1">
            Record of inline-processed background LLM work (summarization,
            entity extraction, etc.)
          </p>
        </div>
        <button
          onClick={() => fetchJobs()}
          disabled={isRefreshing}
          className="px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg font-medium transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          {isRefreshing ? (
            <>
              <span className="animate-spin">⟳</span>
              Refreshing...
            </>
          ) : (
            <>
              <span>🔄</span>
              Refresh
            </>
          )}
        </button>
      </div>

      {actionMessage && (
        <div className="bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded-lg flex items-center justify-between">
          <span className="flex items-center gap-2">
            <span>✓</span>
            {actionMessage}
          </span>
          <button
            onClick={() => setActionMessage(null)}
            className="text-green-600 hover:text-green-800"
          >
            ✕
          </button>
        </div>
      )}

      {error && (
        <AdminErrorDisplay
          message={error.message}
          details={error.details}
          severity="warning"
          onRetry={() => fetchJobs()}
          onDismiss={() => setError(null)}
        />
      )}

      {/* Counts (computed from the loaded page of recent jobs) */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <CountCard
          label="Running (recent)"
          value={runningCount}
          accent="text-blue-600"
          icon="▶"
        />
        <CountCard
          label="Completed (recent)"
          value={completedCount}
          accent="text-green-600"
          icon="✅"
        />
        <CountCard
          label="Failed (recent)"
          value={failedCount}
          accent="text-red-600"
          icon="❌"
        />
      </div>

      {/* Recent jobs table */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">Recent Jobs</h2>
          <span className="text-sm text-gray-500">
            Showing {jobs.length} most recent
          </span>
        </div>
        <div className="overflow-x-auto">
          {jobs.length > 0 ? (
            <table className="w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Job ID
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    User
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Type
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Duration
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Result
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {jobs.map((job) => {
                  const durationMs =
                    job.started_at && job.completed_at
                      ? new Date(job.completed_at).getTime() -
                        new Date(job.started_at).getTime()
                      : null;

                  const formatDuration = (ms: number) => {
                    if (ms < 1000) return `${ms}ms`;
                    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
                    return `${(ms / 60000).toFixed(1)}m`;
                  };

                  const getResultPreview = () => {
                    if (job.status === 'failed') {
                      return job.error_message || 'Failed';
                    }
                    if (job.result) {
                      if (job.result.entities_saved !== undefined) {
                        return `Saved ${job.result.entities_saved} entities`;
                      }
                      if (job.result.summarization_id) {
                        return 'Summary generated';
                      }
                      return 'Success';
                    }
                    return job.status === 'running'
                      ? 'In progress'
                      : 'Completed';
                  };

                  const canRetry =
                    job.status === 'failed' || job.status === 'cancelled';

                  return (
                    <tr key={job.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-sm font-medium text-gray-900">
                        {job.id}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-600">
                        {job.username || `User ${job.user_id}`}
                      </td>
                      <td className="px-4 py-3 text-sm">
                        <span className="flex items-center gap-1">
                          <span>{getJobTypeIcon(job.job_type)}</span>
                          <span className="capitalize">
                            {job.job_type.replace(/_/g, ' ')}
                          </span>
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm">
                        <span
                          className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(
                            job.status,
                          )}`}
                        >
                          {job.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-600">
                        {durationMs !== null ? formatDuration(durationMs) : '-'}
                      </td>
                      <td
                        className="px-4 py-3 text-sm text-gray-600 max-w-xs truncate"
                        title={getResultPreview()}
                      >
                        {getResultPreview()}
                      </td>
                      <td className="px-4 py-3 text-sm">
                        {canRetry && (
                          <button
                            onClick={() => handleRetry(job.id)}
                            disabled={retryingId === job.id}
                            className="px-3 py-1 bg-blue-500 hover:bg-blue-600 text-white rounded text-xs font-medium transition-colors disabled:opacity-50"
                          >
                            {retryingId === job.id ? 'Retrying...' : 'Retry'}
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : (
            <div className="px-6 py-12 text-center text-gray-500">
              <p className="text-sm">No jobs recorded yet</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
