/**
 * Admin-specific types and API interfaces.
 *
 * This file contains all types related to admin functionality including
 * audit logs, statistics, and admin API responses.
 */

/**
 * Admin audit log entry - tracks all admin actions for security auditing
 */
export interface AdminAuditLog {
  id: number;
  admin_user_id: number;
  action: string;
  target_type: string;
  target_id: number | null;
  details: Record<string, unknown>;
  ip_address: string | null;
  user_agent: string | null;
  created_at: string;
}

/**
 * Parameters for querying audit logs
 */
export interface AuditLogQueryParams {
  limit?: number;
  offset?: number;
  action?: string;
  target_type?: string;
}

/**
 * Admin statistics for the dashboard
 */
export interface AdminStats {
  users: UserStats;
  subscriptions: SubscriptionStats;
  revenue: RevenueStats;
  content: ContentStats;
}

/**
 * User statistics
 */
export interface UserStats {
  total: number;
  active_this_week: number;
  active_this_month: number;
  new_this_week: number;
  new_this_month: number;
  total_admins: number;
}

/**
 * Subscription statistics
 */
export interface SubscriptionStats {
  active: number;
  trialing: number;
  free: number;
  past_due: number;
  canceled: number;
  total: number;
}

/**
 * Revenue statistics (in cents)
 */
export interface RevenueStats {
  total_revenue_cents: number;
  monthly_recurring_revenue_cents: number;
  annual_recurring_revenue_cents: number;
  revenue_this_month_cents: number;
  total_revenue: number;
  revenue_this_month: number;
  monthly_recurring_revenue: number;
}

/**
 * Content statistics
 */
export interface ContentStats {
  total_cards: number;
  total_tasks: number;
  total_files: number;
  total_chat_messages: number;
  total_entities: number;
  total_facts: number;
}

/**
 * Mailing list recipient (for message history)
 */
export interface MailingListRecipient {
  id: number;
  message_id: number;
  recipient_email: string;
  recipient_type: "to" | "bcc";
  sent_at: string;
}

/**
 * Response wrapper for admin API errors
 */
export interface AdminErrorResponse {
  error: boolean;
  message: string;
  details?: string;
}

/**
 * Type guard for admin API error responses
 */
export function isAdminErrorResponse(
  response: unknown
): response is AdminErrorResponse {
  return (
    typeof response === "object" &&
    response !== null &&
    "error" in response &&
    (response as AdminErrorResponse).error === true
  );
}

/**
 * Get all admin audit logs
 */
export async function getAdminAuditLogs(
  params: AuditLogQueryParams = {}
): Promise<AdminAuditLog[]> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");

  const queryParams = new URLSearchParams();
  if (params.limit) queryParams.append("limit", params.limit.toString());
  if (params.offset) queryParams.append("offset", params.offset.toString());
  if (params.action) queryParams.append("action", params.action);
  if (params.target_type) queryParams.append("target_type", params.target_type);

  const queryString = queryParams.toString();
  const url = `${base_url}/admin/audit-logs${queryString ? `?${queryString}` : ""}`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch audit logs: ${response.statusText}`);
  }

  return response.json() as Promise<AdminAuditLog[]>;
}

/**
 * Get admin statistics for the dashboard
 */
export async function getAdminStats(): Promise<AdminStats> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/stats`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch admin stats: ${response.statusText}`);
  }

  return response.json() as Promise<AdminStats>;
}

/**
 * Get mailing list recipients for a specific message
 */
export async function getMailingListRecipients(
  messageId: number
): Promise<MailingListRecipient[]> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/mailing-list/messages/recipients?message_id=${messageId}`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch recipients: ${response.statusText}`);
  }

  return response.json() as Promise<MailingListRecipient[]>;
}

/**
 * Job queue health statistics
 */
export interface JobQueueHealth {
  running: boolean;
  paused: boolean;
  worker_count: number;
  queue_depth: number;
  stats: WorkerStats;
}

/**
 * Worker statistics
 */
export interface WorkerStats {
  jobs_processed: number;
  jobs_succeeded: number;
  jobs_failed: number;
  jobs_retried: number;
}

/**
 * Worker pool statistics response
 */
export interface WorkerPoolStats {
  workers: Array<{
    worker_id: string;
    jobs_processed: number;
    jobs_succeeded: number;
    jobs_failed: number;
    jobs_retried: number;
  }>;
  total: WorkerStats;
}

/**
 * Job retry response
 */
export interface JobRetryResponse {
  message: string;
  job_id: number;
}

/**
 * Pause/Resume response
 */
export interface JobQueueControlResponse {
  message: string;
}

/**
 * Get job queue health status
 */
export async function getJobQueueHealth(): Promise<JobQueueHealth> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/jobs/health`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch job queue health: ${response.statusText}`);
  }

  return response.json() as Promise<JobQueueHealth>;
}

/**
 * Get worker pool statistics
 */
export async function getWorkerPoolStats(): Promise<WorkerPoolStats> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/jobs/workers`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch worker stats: ${response.statusText}`);
  }

  return response.json() as Promise<WorkerPoolStats>;
}

/**
 * Job information for admin view
 */
export interface AdminJob {
  id: number;
  user_id: number;
  username: string;
  job_type: string;
  status: string;
  priority: number;
  payload: Record<string, unknown>;
  result?: Record<string, unknown>;
  error_message?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  retry_count: number;
  max_retries: number;
  timeout_seconds: number;
}

/**
 * Response for getAllJobs admin endpoint
 */
export interface AdminJobsResponse {
  jobs: AdminJob[];
  total: number;
  limit: number;
  offset: number;
}

/**
 * Get all jobs (admin only)
 */
export async function getAllJobs(params?: {
  status?: string;
  limit?: number;
  offset?: number;
}): Promise<AdminJobsResponse> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");

  const queryParams = new URLSearchParams();
  if (params?.status) queryParams.set("status", params.status);
  if (params?.limit) queryParams.set("limit", params.limit.toString());
  if (params?.offset) queryParams.set("offset", params.offset.toString());

  const url = `${base_url}/admin/jobs${queryParams.toString() ? `?${queryParams}` : ""}`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get jobs: ${response.statusText}`);
  }

  return response.json() as Promise<AdminJobsResponse>;
}

/**
 * Retry a failed job
 */
export async function retryJob(jobId: number): Promise<JobRetryResponse> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/jobs/${jobId}/retry`;

  const response = await fetch(url, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to retry job: ${response.statusText}`);
  }

  return response.json() as Promise<JobRetryResponse>;
}

/**
 * Pause job queue processing
 */
export async function pauseJobQueue(): Promise<JobQueueControlResponse> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/jobs/pause`;

  const response = await fetch(url, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to pause job queue: ${response.statusText}`);
  }

  return response.json() as Promise<JobQueueControlResponse>;
}

/**
 * Resume job queue processing
 */
export async function resumeJobQueue(): Promise<JobQueueControlResponse> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/jobs/resume`;

  const response = await fetch(url, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to resume job queue: ${response.statusText}`);
  }

  return response.json() as Promise<JobQueueControlResponse>;
}

/**
 * Information about a scheduled job
 */
export interface ScheduledJobInfo {
  name: string;
  schedule: string;
  next_run: string;
}

/**
 * Summary statistics for a scheduled job
 */
export interface JobSummary {
  job_name: string;
  last_run_status: string;
  last_run_at?: string;
  recent_stats: {
    total_runs: number;
    success_count: number;
    failure_count: number;
    success_rate: number;
  };
}

/**
 * Single execution record for a scheduled job
 */
export interface JobRun {
  id: number;
  job_name: string;
  started_at: string;
  completed_at?: string;
  status: string;
  error_message?: string;
  retry_count: number;
}

/**
 * Response from job history endpoint with pagination
 */
export interface JobHistoryResponse {
  runs: JobRun[];
  total: number;
  offset: number;
  limit: number;
  has_more: boolean;
}

/**
 * Scheduler health status
 */
export interface SchedulerHealth {
  running: boolean;
  jobs: string[];
}

/**
 * Get all scheduled jobs
 */
export async function getScheduledJobs(): Promise<{ jobs: ScheduledJobInfo[] }> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/scheduler/jobs`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get scheduled jobs: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Get job summary statistics
 */
export async function getJobSummary(jobName: string): Promise<JobSummary> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/scheduler/jobs/${encodeURIComponent(jobName)}/summary`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get job summary: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Get job execution history with pagination
 */
export async function getJobHistory(
  jobName: string,
  limit: number = 50,
  offset: number = 0
): Promise<JobHistoryResponse> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");

  const params = new URLSearchParams();
  params.set("limit", limit.toString());
  params.set("offset", offset.toString());

  const url = `${base_url}/admin/scheduler/jobs/${encodeURIComponent(jobName)}/history?${params}`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get job history: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Get scheduler health status
 */
export async function getSchedulerHealth(): Promise<SchedulerHealth> {
  const base_url = import.meta.env.VITE_URL;
  const token = localStorage.getItem("token");
  const url = `${base_url}/admin/scheduler/health`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error(`Failed to get scheduler health: ${response.statusText}`);
  }

  return response.json();
}
