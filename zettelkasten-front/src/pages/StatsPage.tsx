import React, { useEffect, useState } from "react";
import { setDocumentTitle } from "../utils/title";
import { fetchDailyStats, fetchTasksForDate } from "../api/stats";
import { DailyStatsResponse } from "../models/Stats";
import { Task } from "../models/Task";
import { ActivityHeatMap } from "../components/stats/ActivityHeatMap";
import { DayTaskList } from "../components/stats/DayTaskList";

export function StatsPage() {
  const [stats, setStats] = useState<DailyStatsResponse | null>(null);
  const [selectedDate, setSelectedDate] = useState<Date | null>(null);
  const [dayTasks, setDayTasks] = useState<Task[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isLoadingTasks, setIsLoadingTasks] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDocumentTitle("Activity Stats");
    loadStats();
  }, []);

  const loadStats = async () => {
    setIsLoading(true);
    setError(null);

    try {
      // Calculate date range (last 365 days)
      const endDate = new Date();
      const startDate = new Date();
      startDate.setDate(startDate.getDate() - 365);

      const response = await fetchDailyStats(startDate, endDate);
      setStats(response);
    } catch (error) {
      console.error("Error fetching stats:", error);
      setError("Failed to load activity stats. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  const handleDateClick = async (date: Date) => {
    setSelectedDate(date);
    setIsLoadingTasks(true);

    try {
      const tasks = await fetchTasksForDate(date);
      setDayTasks(tasks);
    } catch (error) {
      console.error("Error fetching tasks for date:", error);
      setDayTasks([]);
    } finally {
      setIsLoadingTasks(false);
    }
  };

  const handleCloseTaskList = () => {
    setSelectedDate(null);
    setDayTasks([]);
  };

  if (isLoading) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-500">Loading activity stats...</div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-800">{error}</p>
          <button
            onClick={loadStats}
            className="mt-2 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <h1 className="text-3xl font-bold text-gray-900 mb-6">Activity Stats</h1>

      {/* Summary Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm text-gray-600 mb-1">Cards Created</div>
            <div className="text-3xl font-bold text-gray-900">
              {stats.total.cards_created}
            </div>
            <div className="text-xs text-gray-500 mt-1">Last 365 days</div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm text-gray-600 mb-1">Tasks Created</div>
            <div className="text-3xl font-bold text-gray-900">
              {stats.total.tasks_created}
            </div>
            <div className="text-xs text-gray-500 mt-1">Last 365 days</div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="text-sm text-gray-600 mb-1">Tasks Completed</div>
            <div className="text-3xl font-bold text-gray-900">
              {stats.total.tasks_completed}
            </div>
            <div className="text-xs text-gray-500 mt-1">Last 365 days</div>
          </div>
        </div>
      )}

      {/* Heat Map */}
      <div className="bg-white rounded-lg shadow p-6 mb-6 overflow-visible">
        <h2 className="text-xl font-semibold text-gray-800 mb-4">
          Activity Over Time
        </h2>
        <p className="text-sm text-gray-600 mb-4">
          Click on any day to view tasks completed on that date.
        </p>

        {stats && stats.stats.length > 0 ? (
          <ActivityHeatMap
            stats={stats.stats}
            onDateClick={handleDateClick}
            selectedDate={selectedDate}
          />
        ) : (
          <div className="text-center py-8 text-gray-500">
            No activity data available yet. Start creating cards and completing
            tasks!
          </div>
        )}
      </div>

      {/* Selected Day Tasks */}
      {selectedDate && (
        <div>
          {isLoadingTasks ? (
            <div className="bg-white rounded-lg shadow p-6">
              <div className="text-center text-gray-500">Loading tasks...</div>
            </div>
          ) : (
            <DayTaskList
              tasks={dayTasks}
              date={selectedDate}
              onClose={handleCloseTaskList}
            />
          )}
        </div>
      )}
    </div>
  );
}
