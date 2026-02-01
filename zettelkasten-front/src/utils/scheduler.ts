/**
 * Convert cron expression to human-readable description
 */
export function formatCronSchedule(cron: string): string {
  const parts = cron.trim().split(/\s+/);

  if (parts.length < 5 || parts.length > 6) {
    return cron;
  }

  const [seconds, minute, hour, dayOfMonth, month, dayOfWeek] =
    parts.length === 6 ? parts : ["0", ...parts];

  // Common patterns
  if (minute === "0" && hour === "2" && dayOfMonth === "*" && month === "*" && dayOfWeek === "*") {
    return "Daily at 2:00 AM";
  }
  if (minute === "0" && hour === "*" && dayOfMonth === "*" && month === "*" && dayOfWeek === "*") {
    return "Hourly";
  }
  if (cron === "0 0 * * 0") {
    return "Weekly (Sunday midnight)";
  }
  if (cron === "0 0 1 * *") {
    return "Monthly (1st at midnight)";
  }

  // Generic fallback
  const timeStr = `${hour}:${minute.padStart(2, "0")}`;
  if (dayOfMonth !== "*" && month !== "*") {
    return `Day ${dayOfMonth} of every month at ${timeStr}`;
  }
  if (dayOfWeek !== "*") {
    const days = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];
    const dayIndex = parseInt(dayOfWeek);
    const day = days[dayIndex] ?? dayOfWeek;
    return `Every ${day} at ${timeStr}`;
  }

  return `Cron: ${cron}`;
}

/**
 * Format relative time for timestamps
 */
export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
}

/**
 * Format duration in milliseconds to human readable string
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = Math.floor((ms % 60000) / 1000);
  return `${minutes}m ${seconds}s`;
}
