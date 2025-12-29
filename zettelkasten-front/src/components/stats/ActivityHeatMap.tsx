import React from "react";
import { DailyStats } from "../../models/Stats";

interface ActivityHeatMapProps {
  stats: DailyStats[];
  onDateClick: (date: Date) => void;
  selectedDate: Date | null;
}

// Get activity level (0-4) based on total count
// A busy day is considered 50 activities
function getActivityLevel(count: number): number {
  if (count === 0) return 0;
  if (count <= 12) return 1;  // 0-25% of busy day
  if (count <= 25) return 2;  // 25-50% of busy day
  if (count <= 40) return 3;  // 50-80% of busy day
  return 4;                   // 80%+ of busy day
}

// Get background color class based on activity level
function getColorClass(level: number): string {
  const colors = [
    "bg-gray-200 hover:bg-gray-300",
    "bg-green-200 hover:bg-green-300",
    "bg-green-400 hover:bg-green-500",
    "bg-green-600 hover:bg-green-700",
    "bg-green-800 hover:bg-green-900",
  ];
  return colors[level];
}

// Format date as "Mon, Jan 1, 2024"
function formatDate(date: Date): string {
  return date.toLocaleDateString("en-US", {
    weekday: "short",
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function ActivityHeatMap({
  stats,
  onDateClick,
  selectedDate,
}: ActivityHeatMapProps) {
  // Group stats by week
  const weeks: DailyStats[][] = [];
  let currentWeek: DailyStats[] = [];

  // Start from the first day and pad to Sunday if needed
  if (stats.length > 0) {
    const firstDate = new Date(stats[0].date);
    const dayOfWeek = firstDate.getDay(); // 0 = Sunday, 6 = Saturday

    // Pad beginning of first week with empty cells
    for (let i = 0; i < dayOfWeek; i++) {
      currentWeek.push({
        date: new Date(0), // Placeholder date
        cards_created: 0,
        tasks_created: 0,
        tasks_completed: 0,
      });
    }
  }

  // Group into weeks
  stats.forEach((stat, index) => {
    currentWeek.push(stat);

    // If we've filled a week (7 days) or reached the end
    if (currentWeek.length === 7) {
      weeks.push([...currentWeek]);
      currentWeek = [];
    }
  });

  // Add remaining days as final week
  if (currentWeek.length > 0) {
    // Pad end of last week to complete it
    while (currentWeek.length < 7) {
      currentWeek.push({
        date: new Date(0), // Placeholder date
        cards_created: 0,
        tasks_created: 0,
        tasks_completed: 0,
      });
    }
    weeks.push(currentWeek);
  }

  return (
    <div className="w-full overflow-x-auto">
      <div className="inline-block min-w-full">
        {/* Day labels */}
        <div className="flex mb-2">
          <div className="w-8"></div>
          <div className="flex-1 flex gap-1">
            {weeks.map((_, weekIndex) => (
              <div key={weekIndex} className="flex-1 min-w-[12px]"></div>
            ))}
          </div>
        </div>

        {/* Heat map grid */}
        <div className="flex">
          {/* Day of week labels */}
          <div className="flex flex-col gap-1 mr-2 text-xs text-gray-600">
            <div className="h-3">Sun</div>
            <div className="h-3">Mon</div>
            <div className="h-3">Tue</div>
            <div className="h-3">Wed</div>
            <div className="h-3">Thu</div>
            <div className="h-3">Fri</div>
            <div className="h-3">Sat</div>
          </div>

          {/* Week columns */}
          <div className="flex gap-1">
            {weeks.map((week, weekIndex) => (
              <div key={weekIndex} className="flex flex-col gap-1">
                {week.map((stat, dayIndex) => {
                  // Tasks count as half of cards (opening + closing = 1 activity)
                  const totalActivity =
                    stat.cards_created +
                    (stat.tasks_created + stat.tasks_completed) * 0.5;
                  const level = getActivityLevel(totalActivity);
                  const isPlaceholder = stat.date.getTime() === 0;
                  const isSelected =
                    selectedDate &&
                    stat.date.toDateString() === selectedDate.toDateString();

                  // Show tooltip below for top 3 rows, above for bottom 4 rows
                  const showBelow = dayIndex < 3;
                  const tooltipPositionClass = showBelow
                    ? "top-full mt-2"
                    : "bottom-full mb-2";

                  return (
                    <div
                      key={dayIndex}
                      className={`w-3 h-3 rounded-sm relative group cursor-pointer transition-all ${
                        isPlaceholder
                          ? "bg-transparent cursor-default"
                          : getColorClass(level)
                      } ${isSelected ? "ring-2 ring-blue-500" : ""}`}
                      onClick={() => !isPlaceholder && onDateClick(stat.date)}
                    >
                      {!isPlaceholder && (
                        <div className={`absolute ${tooltipPositionClass} left-1/2 transform -translate-x-1/2 hidden group-hover:block z-[9999] pointer-events-none`}>
                          <div className="bg-gray-900 text-white text-xs rounded py-2 px-3 whitespace-nowrap shadow-lg">
                            <div className="font-semibold mb-1">
                              {formatDate(stat.date)}
                            </div>
                            <div>Cards created: {stat.cards_created}</div>
                            <div>Tasks created: {stat.tasks_created}</div>
                            <div>Tasks completed: {stat.tasks_completed}</div>
                            <div className="font-semibold mt-1">
                              Total: {totalActivity.toFixed(1)}
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            ))}
          </div>
        </div>

        {/* Legend */}
        <div className="mt-4 flex items-center gap-2 text-xs text-gray-600">
          <span>Less</span>
          {[0, 1, 2, 3, 4].map((level) => (
            <div
              key={level}
              className={`w-3 h-3 rounded-sm ${getColorClass(level)}`}
            ></div>
          ))}
          <span>More</span>
        </div>
      </div>
    </div>
  );
}
