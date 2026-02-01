import React from "react";
import { formatCronSchedule, formatRelativeTime } from "../../utils/scheduler";

interface ScheduleDisplayProps {
  schedule: string;
  nextRun: string;
}

export function ScheduleDisplay({ schedule, nextRun }: ScheduleDisplayProps) {
  const humanReadable = formatCronSchedule(schedule);
  const relativeTime = formatRelativeTime(nextRun);

  return (
    <div className="group relative">
      <div className="text-sm text-gray-900">{humanReadable}</div>
      <div className="text-xs text-gray-500">
        Next: {relativeTime}
      </div>
      {/* Tooltip with raw cron */}
      <div className="absolute bottom-full left-0 mb-2 hidden group-hover:block bg-gray-900 text-white text-xs px-2 py-1 rounded whitespace-nowrap z-10">
        {schedule}
      </div>
    </div>
  );
}
