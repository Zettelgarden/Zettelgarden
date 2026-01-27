import React, { useState, useEffect } from "react";
import {
  getJobQueueHealth,
  getWorkerPoolStats,
  getAllJobs,
  pauseJobQueue,
  resumeJobQueue,
  JobQueueHealth,
  WorkerPoolStats,
  AdminJob,
} from "../../api/admin";
import { AdminErrorDisplay } from "../../components/admin/AdminErrorDisplay";

interface ErrorState {
  message: string;
  details?: string;
}

interface StatusBadgeProps {
  running: boolean;
  paused: boolean;
}

function StatusBadge({ running, paused }: StatusBadgeProps) {
  if (!running) {
    return (
      <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-gray-100 text-gray-800">
        <span className="w-2 h-2 bg-gray-500 rounded-full mr-2"></span>
        Stopped
      </span>
    );
  }
  if (paused) {
    return (
      <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-yellow-100 text-yellow-800">
        <span className="w-2 h-2 bg-yellow-500 rounded-full mr-2 animate-pulse"></span>
        Paused
      </span>
    );
  }
  return (
    <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800">
      <span className="w-2 h-2 bg-green-500 rounded-full mr-2 animate-pulse"></span>
      Running
    </span>
  );
}

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: string;
}

function StatCard({ title, value, subtitle, icon = "📊" }: StatCardProps) {
  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-5">
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <p className="text-sm text-gray-600 font-medium">{title}</p>
          <p className="text-2xl font-bold text-gray-900 mt-1">{value}</p>
          {subtitle && (
            <p className="text-sm text-gray-500 mt-1">{subtitle}</p>
          )}
        </div>
        <div className="text-3xl ml-4">{icon}</div>
      </div>
    </div>
  );
}

interface WorkerRowProps {
  workerId: string;
  stats: {
    jobs_processed: number;
    jobs_succeeded: number;
    jobs_failed: number;
    jobs_retried: number;
  };
}

function WorkerRow({ workerId, stats }: WorkerRowProps) {
  const successRate =
    stats.jobs_processed > 0
      ? ((stats.jobs_succeeded / stats.jobs_processed) * 100).toFixed(1)
      : "0.0";

  return (
    <tr className="hover:bg-gray-50">
      <td className="px-4 py-3 text-sm font-medium text-gray-900">{workerId}</td>
      <td className="px-4 py-3 text-sm text-gray-600">{stats.jobs_processed}</td>
      <td className="px-4 py-3 text-sm text-green-600">{stats.jobs_succeeded}</td>
      <td className="px-4 py-3 text-sm text-red-600">{stats.jobs_failed}</td>
      <td className="px-4 py-3 text-sm text-blue-600">{stats.jobs_retried}</td>
      <td className="px-4 py-3 text-sm text-gray-900">{successRate}%</td>
    </tr>
  );
}

// Helper functions for job display
const getJobTypeIcon = (jobType: string) => {
  switch (jobType) {
    case "embedding":
      return "🔤";
    case "summarization":
      return "📝";
    case "entity_extraction":
    case "fact_entity_extraction":
      return "🏷️";
    case "chat":
      return "💬";
    case "memory":
      return "🧠";
    case "email":
      return "📧";
    default:
      return "⚙️";
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case "pending":
      return "bg-yellow-100 text-yellow-800";
    case "running":
      return "bg-blue-100 text-blue-800";
    case "completed":
      return "bg-green-100 text-green-800";
    case "failed":
      return "bg-red-100 text-red-800";
    case "cancelled":
      return "bg-gray-100 text-gray-800";
    default:
      return "bg-gray-100 text-gray-800";
  }
};

interface RunningJobsRowProps {
  job: AdminJob;
}

function RunningJobsRow({ job }: RunningJobsRowProps) {
  const formatDate = (dateStr?: string) => {
    if (!dateStr) return "-";
    return new Date(dateStr).toLocaleTimeString();
  };

  return (
    <tr className="hover:bg-gray-50">
      <td className="px-4 py-3 text-sm font-medium text-gray-900">{job.id}</td>
      <td className="px-4 py-3 text-sm text-gray-600">{job.username || `User ${job.user_id}`}</td>
      <td className="px-4 py-3 text-sm">
        <span className="flex items-center gap-1">
          <span>{getJobTypeIcon(job.job_type)}</span>
          <span className="capitalize">{job.job_type.replace(/_/g, " ")}</span>
        </span>
      </td>
      <td className="px-4 py-3 text-sm">
        <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(job.status)}`}>
          {job.status}
        </span>
      </td>
      <td className="px-4 py-3 text-sm text-gray-600">{formatDate(job.started_at)}</td>
      <td className="px-4 py-3 text-sm text-gray-600">{formatDate(job.completed_at)}</td>
      <td className="px-4 py-3 text-sm text-gray-500">{job.retry_count}/{job.max_retries}</td>
    </tr>
  );
}

export function AdminJobQueuePage() {
  const [health, setHealth] = useState<JobQueueHealth | null>(null);
  const [workerStats, setWorkerStats] = useState<WorkerPoolStats | null>(null);
  const [jobs, setJobs] = useState<AdminJob[]>([]);
  const [completedJobs, setCompletedJobs] = useState<AdminJob[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<ErrorState | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);

  const fetchData = async (showRefreshing = false) => {
    if (showRefreshing) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }
    setError(null);
    try {
      const [healthData, workersData, jobsData, completedJobsData] = await Promise.all([
        getJobQueueHealth(),
        getWorkerPoolStats(),
        getAllJobs({ status: "running", limit: 20 }),
        getAllJobs({ status: "completed", limit: 20 }),
      ]);
      setHealth(healthData);
      setWorkerStats(workersData);
      setJobs(jobsData.jobs);
      setCompletedJobs(completedJobsData.jobs);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load job queue data";
      setError({ message, details: err instanceof Error ? err.stack : undefined });
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    fetchData();
    // Auto-refresh every 5 seconds
    const interval = setInterval(() => fetchData(true), 5000);
    return () => clearInterval(interval);
  }, []);

  const handlePause = async () => {
    setActionMessage(null);
    try {
      await pauseJobQueue();
      setActionMessage("Job queue paused successfully");
      fetchData(true);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to pause job queue";
      setError({ message, details: err instanceof Error ? err.stack : undefined });
    }
  };

  const handleResume = async () => {
    setActionMessage(null);
    try {
      await resumeJobQueue();
      setActionMessage("Job queue resumed successfully");
      fetchData(true);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to resume job queue";
      setError({ message, details: err instanceof Error ? err.stack : undefined });
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-pulse text-gray-600">Loading job queue data...</div>
      </div>
    );
  }

  if (error && !health) {
    return (
      <AdminErrorDisplay
        message={error.message}
        details={error.details}
        severity="error"
        onRetry={() => fetchData()}
        onDismiss={() => setError(null)}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Job Queue Management</h1>
          <p className="text-gray-600 mt-1">
            Monitor and control the background job processing system
          </p>
        </div>
        <div className="flex items-center gap-3">
          {health && <StatusBadge running={health.running} paused={health.paused} />}
          <button
            onClick={() => fetchData()}
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
      </div>

      {/* Action Messages */}
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

      {/* Error Display */}
      {error && health && (
        <AdminErrorDisplay
          message={error.message}
          details={error.details}
          severity="warning"
          onDismiss={() => setError(null)}
        />
      )}

      {health && (
        <>
          {/* Control Buttons */}
          <div className="flex gap-3">
            {health.running && !health.paused && (
              <button
                onClick={handlePause}
                className="px-4 py-2 bg-yellow-500 hover:bg-yellow-600 text-white rounded-lg font-medium transition-colors flex items-center gap-2"
              >
                <span>⏸</span>
                Pause Queue
              </button>
            )}
            {health.running && health.paused && (
              <button
                onClick={handleResume}
                className="px-4 py-2 bg-green-500 hover:bg-green-600 text-white rounded-lg font-medium transition-colors flex items-center gap-2"
              >
                <span>▶</span>
                Resume Queue
              </button>
            )}
          </div>

          {/* Overview Stats */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              title="Worker Count"
              value={health.worker_count}
              subtitle="Active workers"
              icon="👷"
            />
            <StatCard
              title="Queue Depth"
              value={health.queue_depth}
              subtitle="Pending jobs"
              icon="📥"
            />
            <StatCard
              title="Jobs Processed"
              value={health.stats.jobs_processed.toLocaleString()}
              subtitle="Total processed"
              icon="⚙️"
            />
            <StatCard
              title="Success Rate"
              value={
                health.stats.jobs_processed > 0
                  ? `${((health.stats.jobs_succeeded / health.stats.jobs_processed) * 100).toFixed(1)}%`
                  : "N/A"
              }
              subtitle={`${health.stats.jobs_succeeded}/${health.stats.jobs_processed} succeeded`}
              icon="✓"
            />
          </div>

          {/* Detailed Stats */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              title="Succeeded"
              value={health.stats.jobs_succeeded.toLocaleString()}
              icon="✅"
            />
            <StatCard
              title="Failed"
              value={health.stats.jobs_failed.toLocaleString()}
              icon="❌"
            />
            <StatCard
              title="Retried"
              value={health.stats.jobs_retried.toLocaleString()}
              icon="🔄"
            />
            <StatCard
              title="Status"
              value={health.paused ? "Paused" : health.running ? "Running" : "Stopped"}
              subtitle={health.running ? "Processing jobs" : "Not processing"}
              icon={health.paused ? "⏸" : health.running ? "▶" : "⏹"}
            />
          </div>

          {/* Worker Statistics Table */}
          {workerStats && workerStats.workers.length > 0 && (
            <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
              <div className="px-6 py-4 border-b border-gray-200">
                <h2 className="text-lg font-semibold text-gray-900">Per-Worker Statistics</h2>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Worker ID
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Processed
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Succeeded
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Failed
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Retried
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Success Rate
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {workerStats.workers.map((worker) => (
                      <WorkerRow
                        key={worker.worker_id}
                        workerId={worker.worker_id}
                        stats={{
                          jobs_processed: worker.jobs_processed,
                          jobs_succeeded: worker.jobs_succeeded,
                          jobs_failed: worker.jobs_failed,
                          jobs_retried: worker.jobs_retried,
                        }}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Running Jobs Table */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
            <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900">Running Jobs</h2>
              <span className="text-sm text-gray-500">
                {jobs.length} active {jobs.length === 1 ? "job" : "jobs"}
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
                        Started
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Completed
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Retries
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {jobs.map((job) => (
                      <RunningJobsRow key={job.id} job={job} />
                    ))}
                  </tbody>
                </table>
              ) : (
                <div className="px-6 py-12 text-center text-gray-500">
                  <p className="text-sm">No active jobs</p>
                </div>
              )}
            </div>
          </div>

          {/* Completed Jobs Table */}
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
            <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900">Recently Completed Jobs</h2>
              <span className="text-sm text-gray-500">
                Last {completedJobs.length} completed
              </span>
            </div>
            <div className="overflow-x-auto">
              {completedJobs.length > 0 ? (
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
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {completedJobs.map((job) => {
                      // Calculate duration if we have both timestamps
                      const durationMs = job.started_at && job.completed_at
                        ? new Date(job.completed_at).getTime() - new Date(job.started_at).getTime()
                        : null;

                      const formatDuration = (ms: number) => {
                        if (ms < 1000) return `${ms}ms`;
                        if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
                        return `${(ms / 60000).toFixed(1)}m`;
                      };

                      // Get result preview
                      const getResultPreview = () => {
                        if (job.status === "failed") {
                          return job.error_message || "Failed";
                        }
                        if (job.result) {
                          // Try to extract a meaningful preview from result
                          if (job.result.entities_saved !== undefined) {
                            return `Saved ${job.result.entities_saved} entities`;
                          }
                          if (job.result.fact_entity_extraction) {
                            return "Entity extraction";
                          }
                          if (job.result.summarization_id) {
                            return "Summary generated";
                          }
                          return "Success";
                        }
                        return "Completed";
                      };

                      return (
                        <tr key={job.id} className="hover:bg-gray-50">
                          <td className="px-4 py-3 text-sm font-medium text-gray-900">{job.id}</td>
                          <td className="px-4 py-3 text-sm text-gray-600">{job.username || `User ${job.user_id}`}</td>
                          <td className="px-4 py-3 text-sm">
                            <span className="flex items-center gap-1">
                              <span>{getJobTypeIcon(job.job_type)}</span>
                              <span className="capitalize">{job.job_type.replace(/_/g, " ")}</span>
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                              job.status === "completed" ? "bg-green-100 text-green-800" :
                              job.status === "failed" ? "bg-red-100 text-red-800" :
                              "bg-gray-100 text-gray-800"
                            }`}>
                              {job.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm text-gray-600">
                            {durationMs !== null ? formatDuration(durationMs) : "-"}
                          </td>
                          <td className="px-4 py-3 text-sm text-gray-600 max-w-xs truncate" title={getResultPreview()}>
                            {getResultPreview()}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              ) : (
                <div className="px-6 py-12 text-center text-gray-500">
                  <p className="text-sm">No completed jobs</p>
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
