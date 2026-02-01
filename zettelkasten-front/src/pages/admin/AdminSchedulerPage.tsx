import React, { useState, useEffect } from "react";
import {
  getScheduledJobs,
  getSchedulerHealth,
  getJobSummary,
  ScheduledJobInfo,
  SchedulerHealth,
  JobSummary,
} from "../../api/admin";
import { AdminErrorDisplay } from "../../components/admin/AdminErrorDisplay";
import { JobStatusBadge } from "../../components/scheduler/JobStatusBadge";
import { ScheduleDisplay } from "../../components/scheduler/ScheduleDisplay";
import { RecentStatsSummary } from "../../components/scheduler/RecentStatsSummary";
import { ExpandableHistory } from "../../components/scheduler/ExpandableHistory";

interface SchedulerJobData extends ScheduledJobInfo {
  summary?: JobSummary;
}

interface ErrorState {
  message: string;
  details?: string;
}

export function AdminSchedulerPage() {
  const [jobs, setJobs] = useState<SchedulerJobData[]>([]);
  const [health, setHealth] = useState<SchedulerHealth | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<ErrorState | null>(null);
  const [expandedJobs, setExpandedJobs] = useState<Set<string>>(new Set());

  const fetchAllData = async (showRefreshing = false) => {
    if (showRefreshing) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }
    setError(null);

    try {
      const [jobsData, healthData] = await Promise.all([
        getScheduledJobs(),
        getSchedulerHealth(),
      ]);

      setHealth(healthData);

      // Fetch summaries for each job
      const jobsWithSummaries = await Promise.all(
        jobsData.jobs.map(async (job) => {
          try {
            const summary = await getJobSummary(job.name);
            return { ...job, summary };
          } catch {
            return { ...job, summary: undefined };
          }
        })
      );

      setJobs(jobsWithSummaries);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to load scheduler data";
      setError({ message, details: err instanceof Error ? err.stack : undefined });
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    fetchAllData();
  }, []);

  const handleRefresh = () => {
    fetchAllData(true);
  };

  const toggleExpanded = (jobName: string) => {
    setExpandedJobs((prev) => {
      const next = new Set(prev);
      if (next.has(jobName)) {
        next.delete(jobName);
      } else {
        next.add(jobName);
      }
      return next;
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-pulse text-gray-600">Loading scheduled jobs...</div>
      </div>
    );
  }

  if (error && !health) {
    return (
      <AdminErrorDisplay
        message={error.message}
        details={error.details}
        severity="error"
        onRetry={() => fetchAllData()}
        onDismiss={() => setError(null)}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Scheduled Jobs</h1>
          <p className="text-gray-600 mt-1">
            Monitor and manage cron-based scheduled jobs
          </p>
        </div>
        <div className="flex items-center gap-3">
          {health && (
            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${
              health.running ? "bg-green-100 text-green-800" : "bg-gray-100 text-gray-800"
            }`}>
              <span className={`w-2 h-2 rounded-full mr-2 ${health.running ? "bg-green-500" : "bg-gray-500"}`} />
              {health.running ? "Running" : "Stopped"}
            </span>
          )}
          <button
            onClick={handleRefresh}
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

      {error && health && (
        <AdminErrorDisplay
          message={error.message}
          details={error.details}
          severity="warning"
          onDismiss={() => setError(null)}
        />
      )}

      {/* Jobs Table */}
      {jobs.length === 0 ? (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
          <p className="text-gray-500">No scheduled jobs configured</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Job Name
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Schedule
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Recent Stats
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {jobs.map((job) => {
                const isExpanded = expandedJobs.has(job.name);
                const lastStatus = job.summary?.last_run_status ?? "never";

                return (
                  <React.Fragment key={job.name}>
                    <tr className="hover:bg-gray-50">
                      <td className="px-6 py-4 text-sm font-medium text-gray-900">
                        {job.name}
                      </td>
                      <td className="px-6 py-4">
                        <ScheduleDisplay schedule={job.schedule} nextRun={job.next_run} />
                      </td>
                      <td className="px-6 py-4">
                        <JobStatusBadge status={lastStatus as any} />
                      </td>
                      <td className="px-6 py-4">
                        <RecentStatsSummary summary={job.summary} />
                      </td>
                      <td className="px-6 py-4">
                        <button
                          onClick={() => toggleExpanded(job.name)}
                          className="text-sm text-blue-600 hover:text-blue-800 font-medium"
                        >
                          {isExpanded ? "Hide" : "View"} History
                        </button>
                      </td>
                    </tr>
                    {isExpanded && (
                      <tr>
                        <td colSpan={5} className="px-6 py-0">
                          <ExpandableHistory jobName={job.name} isExpanded={isExpanded} />
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
