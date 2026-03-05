import React, { useEffect, useState } from 'react';
import { useHabits } from '../../contexts/HabitContext';

export const TodaysHabitsWidget: React.FC = () => {
  const { todaysHabits, fetchTodaysHabits, checkinHabit } = useHabits();
  const [checkingIn, setCheckingIn] = useState<number | null>(null);

  useEffect(() => {
    fetchTodaysHabits();
    const interval = setInterval(fetchTodaysHabits, 60000);
    return () => clearInterval(interval);
  }, [fetchTodaysHabits]);

  const handleCheckin = async (habitId: number) => {
    setCheckingIn(habitId);
    try {
      await checkinHabit(habitId);
    } catch (e) {
      // Ignore "already checked in" errors
    } finally {
      setCheckingIn(null);
    }
  };

  if (!todaysHabits || todaysHabits.length === 0) return null;

  return (
    <div className="mb-4">
      <h3 className="text-sm font-semibold mb-2">Today's Habits</h3>
      <div className="space-y-1">
        {todaysHabits.map((h) => (
          <div key={h.id} className="flex items-center justify-between text-sm">
            <div className="flex items-center gap-2 overflow-hidden">
              {h.icon && <span>{h.icon}</span>}
              <span className="truncate">{h.title}</span>
            </div>
            <button
              className={`ml-2 px-2 py-0.5 rounded text-xs ${
                h.checked_in_today
                  ? 'bg-green-500 text-white'
                  : 'bg-gray-200 hover:bg-gray-300'
              }`}
              onClick={() => handleCheckin(h.id)}
              disabled={checkingIn === h.id || h.checked_in_today}
            >
              {h.checked_in_today ? '✓' : checkingIn === h.id ? '...' : '✓'}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};
