import React, { useEffect, useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useHabits } from '../../contexts/HabitContext';
import { formatDistanceToNow, parseISO } from 'date-fns';

export const TodaysHabitsWidget: React.FC = () => {
  const { todaysHabits, fetchTodaysHabits, checkinHabit, habitStats, fetchHabitStats } = useHabits();
  const [checkingIn, setCheckingIn] = useState<number | null>(null);
  const [isCollapsed, setIsCollapsed] = useState(true);
  const navigate = useNavigate();

  const overdueCount = useMemo(
    () => todaysHabits.filter(h => h.is_overdue).length,
    [todaysHabits],
  );

  const outstandingCount = useMemo(
    () => todaysHabits.filter(h => !h.checked_in_today).length,
    [todaysHabits],
  );

  useEffect(() => {
    fetchTodaysHabits();
    const interval = setInterval(fetchTodaysHabits, 60000);
    return () => clearInterval(interval);
  }, [fetchTodaysHabits]);

  // Fetch stats for habits when they're loaded
  useEffect(() => {
    todaysHabits.forEach(h => {
      if (!habitStats[h.id]) {
        fetchHabitStats(h.id).catch(() => {});
      }
    });
  }, [todaysHabits, habitStats, fetchHabitStats]);

  const handleCheckin = async (habitId: number) => {
    setCheckingIn(habitId);
    try {
      await checkinHabit(habitId);
    } catch (e) {
      // Even on error (e.g., "already checked in"), refresh to get current state
      await fetchTodaysHabits();
    } finally {
      setCheckingIn(null);
    }
  };

  if (!todaysHabits || todaysHabits.length === 0) return null;

  return (
    <div className="p-2">
      <button
        className="flex items-center justify-between w-full mb-2 px-2 group"
        onClick={() => setIsCollapsed(!isCollapsed)}
      >
        <div className="flex items-center gap-2">
          <svg
            className={`w-3 h-3 text-gray-400 transition-transform ${isCollapsed ? '' : 'rotate-90'}`}
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path fillRule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clipRule="evenodd" />
          </svg>
          <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
            Today's Habits
          </h3>
        </div>
        <div className="flex items-center gap-2">
          {outstandingCount > 0 && (
            <span className={`inline-flex items-center justify-center px-1.5 py-0.5 text-xs font-bold leading-none text-white rounded-full ${overdueCount > 0 ? 'bg-red-500' : 'bg-amber-500'}`}>
              {outstandingCount}
            </span>
          )}
          {outstandingCount === 0 && todaysHabits.length > 0 && (
            <span className="text-xs text-green-500">✓</span>
          )}
          <span
            onClick={(e) => { e.stopPropagation(); navigate('/app/habits'); }}
            className="text-xs text-gray-400 hover:text-blue-600"
          >
            View All →
          </span>
        </div>
      </button>
      {!isCollapsed && (
      <div className="space-y-1">
        {todaysHabits.map((h) => {
          const stats = habitStats[h.id];
          const showLastCompleted = !h.checked_in_today && stats?.last_completed_at;
          const habitColor = h.color || '#10b981';
          
          return (
            <div key={h.id} className={`flex items-center justify-between text-sm ${h.is_overdue ? 'bg-red-50 rounded px-1 -mx-1' : ''}`}>
              <div className="flex items-center gap-2 overflow-hidden">
                <div
                  className="w-1 h-4 rounded-full flex-shrink-0"
                  style={{ backgroundColor: h.is_overdue ? '#ef4444' : habitColor }}
                />
                {h.icon && <span>{h.icon}</span>}
                <div className="flex flex-col">
                  <span className="truncate">
                    {h.title}
                    {h.is_overdue && <span className="text-xs text-red-500 ml-1">overdue</span>}
                  </span>
                  {showLastCompleted && (
                    <span className="text-xs text-gray-400 truncate">
                      Last: {formatDistanceToNow(parseISO(stats.last_completed_at!), { addSuffix: true })}
                    </span>
                  )}
                </div>
              </div>
              <button
                className="ml-2 px-2 py-0.5 rounded text-xs flex-shrink-0 text-white"
                style={{
                  backgroundColor: h.checked_in_today ? habitColor : '#d1d5db',
                }}
                onClick={() => handleCheckin(h.id)}
                disabled={checkingIn === h.id || h.checked_in_today}
              >
                {h.checked_in_today ? '✓' : checkingIn === h.id ? '...' : '✓'}
              </button>
            </div>
          );
        })}
      </div>
      )}
    </div>
  );
};
