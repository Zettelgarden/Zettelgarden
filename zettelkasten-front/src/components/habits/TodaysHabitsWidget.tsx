import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useHabits } from '../../contexts/HabitContext';
import { formatDistanceToNow, parseISO } from 'date-fns';

export const TodaysHabitsWidget: React.FC = () => {
  const { todaysHabits, fetchTodaysHabits, checkinHabit, habitStats, fetchHabitStats } = useHabits();
  const [checkingIn, setCheckingIn] = useState<number | null>(null);
  const navigate = useNavigate();

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
    <div className="mb-4 px-3">
      <h3 className="text-sm font-semibold mb-2 flex items-center justify-between">
        <span>Today's Habits</span>
        <button
          onClick={() => navigate('/app/habits')}
          className="text-xs font-normal text-blue-600 hover:text-blue-800"
        >
          View All →
        </button>
      </h3>
      <div className="space-y-1">
        {todaysHabits.map((h) => {
          const stats = habitStats[h.id];
          const showLastCompleted = !h.checked_in_today && stats?.last_completed_at;
          const habitColor = h.color || '#10b981';
          
          return (
            <div key={h.id} className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2 overflow-hidden">
                <div
                  className="w-1 h-4 rounded-full flex-shrink-0"
                  style={{ backgroundColor: habitColor }}
                />
                {h.icon && <span>{h.icon}</span>}
                <div className="flex flex-col">
                  <span className="truncate">{h.title}</span>
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
    </div>
  );
};
