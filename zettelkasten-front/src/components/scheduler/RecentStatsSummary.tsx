import React from "react";
import { JobSummary } from "../../api/admin";

interface RecentStatsSummaryProps {
  summary?: JobSummary;
}

export function RecentStatsSummary({ summary }: RecentStatsSummaryProps) {
  if (!summary) {
    return <span className="text-sm text-gray-400 italic">Stats unavailable</span>;
  }

  const { total_runs, success_count, failure_count, success_rate } = summary.recent_stats;

  if (total_runs === 0) {
    return <span className="text-sm text-gray-400 italic">No recent runs</span>;
  }

  // Determine color based on success rate
  const getColor = () => {
    if (success_rate >= 95) return "bg-green-500";
    if (success_rate >= 70) return "bg-yellow-500";
    return "bg-red-500";
  };

  return (
    <div className="group relative flex items-center gap-2">
      {/* Success rate bar */}
      <div className="w-16 h-2 bg-gray-200 rounded-full overflow-hidden">
        <div
          className={`h-full ${getColor()} transition-all`}
          style={{ width: `${success_rate}%` }}
        />
      </div>
      <span className="text-sm text-gray-600">{success_rate.toFixed(0)}%</span>

      {/* Tooltip with detailed stats */}
      <div className="absolute bottom-full left-0 mb-2 hidden group-hover:block bg-gray-900 text-white text-xs px-3 py-2 rounded z-10">
        <div className="font-medium mb-1">Last 7 days</div>
        <div>Total runs: {total_runs}</div>
        <div className="text-green-400">Success: {success_count}</div>
        <div className="text-red-400">Failed: {failure_count}</div>
      </div>
    </div>
  );
}
