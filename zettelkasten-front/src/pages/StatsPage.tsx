import React, { useEffect, useState } from 'react';
import { setDocumentTitle } from '../utils/title';
import {
  fetchDailyStats,
  fetchTasksForDate,
  fetchCardsForDate,
} from '../api/stats';
import { DailyStatsResponse } from '../models/Stats';
import { Task } from '../models/Task';
import { PartialCard } from '../models/Card';
import { ActivityHeatMap } from '../components/stats/ActivityHeatMap';
import { DayTaskList } from '../components/stats/DayTaskList';
import { DayCardList } from '../components/stats/DayCardList';
import { CardItem } from '../components/cards/CardItem';
import { getOrphanCards } from '../api/cards';
import { MobileTopBar } from '../components/layout/MobileTopBar';
import { useUIState } from '../contexts/UIStateContext';

export function StatsPage() {
  const { toggleMobileSidebar } = useUIState();
  const [stats, setStats] = useState<DailyStatsResponse | null>(null);
  const [selectedDate, setSelectedDate] = useState<Date | null>(null);
  const [dayTasks, setDayTasks] = useState<Task[]>([]);
  const [dayCards, setDayCards] = useState<PartialCard[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isLoadingDay, setIsLoadingDay] = useState<boolean>(false);
  const [orphans, setOrphans] = useState<PartialCard[]>([]);
  const [orphansError, setOrphansError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDocumentTitle('Activity Stats');
    loadStats();
    loadOrphans();
  }, []);

  const loadOrphans = async () => {
    setOrphansError(null);
    try {
      const cards = await getOrphanCards();
      setOrphans(cards);
    } catch (err) {
      console.error('Error fetching orphan cards:', err);
      setOrphansError('Failed to load orphan cards');
    }
  };

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
      console.error('Error fetching stats:', error);
      setError('Failed to load activity stats. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDateClick = async (date: Date) => {
    setSelectedDate(date);
    setIsLoadingDay(true);

    try {
      const [tasks, cards] = await Promise.all([
        fetchTasksForDate(date),
        fetchCardsForDate(date),
      ]);
      setDayTasks(tasks);
      setDayCards(cards);
    } catch (error) {
      console.error('Error fetching data for date:', error);
      setDayTasks([]);
      setDayCards([]);
    } finally {
      setIsLoadingDay(false);
    }
  };

  const handleCloseDayView = () => {
    setSelectedDate(null);
    setDayTasks([]);
    setDayCards([]);
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
    <div>
      <MobileTopBar title="Activity Stats" onMenuClick={toggleMobileSidebar} />
      <div className="p-6 max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold text-gray-900 mb-6 md:block hidden">
          Activity Stats
        </h1>

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
            Click on any day to view cards created and tasks completed on that
            date.
          </p>

          {stats && stats.stats.length > 0 ? (
            <ActivityHeatMap
              stats={stats.stats}
              onDateClick={handleDateClick}
              selectedDate={selectedDate}
            />
          ) : (
            <div className="text-center py-8 text-gray-500">
              No activity data available yet. Start creating cards and
              completing tasks!
            </div>
          )}
        </div>

        {/* Selected Day Details */}
        {selectedDate && (
          <div>
            {isLoadingDay ? (
              <div className="bg-white rounded-lg shadow p-6">
                <div className="text-center text-gray-500">Loading...</div>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <DayCardList
                  cards={dayCards}
                  date={selectedDate}
                  onClose={handleCloseDayView}
                />
                <DayTaskList
                  tasks={dayTasks}
                  date={selectedDate}
                  onClose={handleCloseDayView}
                />
              </div>
            )}
          </div>
        )}

        {/* Orphan Cards */}
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between mb-2">
            <h2 className="text-xl font-semibold text-gray-800">
              Orphan Cards
            </h2>
            <button
              onClick={loadOrphans}
              className="text-sm text-blue-500 hover:text-blue-700"
            >
              Refresh
            </button>
          </div>
          <p className="text-sm text-gray-600 mb-4">
            Cards with no references, no children, and no shared entities or
            tags. These are prime candidates for building connections — open one
            to see related cards and unlinked mentions.
          </p>
          {orphansError ? (
            <div className="text-red-600 text-sm">{orphansError}</div>
          ) : orphans.length === 0 ? (
            <div className="text-gray-500 text-sm">
              No orphan cards — your vault is well connected!
            </div>
          ) : (
            <div className="max-h-80 overflow-y-auto">
              <ul className="space-y-1">
                {orphans.map((card) => (
                  <li key={card.id}>
                    <CardItem card={card} />
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
