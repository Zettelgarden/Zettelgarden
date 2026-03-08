import React, { useState, useEffect } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { HabitFormDialog } from './HabitFormDialog';
import { useToast } from '../toast/ToastContext';

export const HabitList: React.FC = () => {
  const { habits, selectedHabit, setSelectedHabit, deleteHabit, habitStats, fetchHabitStats } = useHabits();
  const { showToast } = useToast();
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  // Fetch stats for all habits on mount
  useEffect(() => {
    habits.forEach(h => {
      fetchHabitStats(h.id).catch(() => {}); // Silently ignore if already fetched
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [habits]);

  const handleDelete = async (id: number) => {
    if (confirm('Delete this habit?')) {
      try {
        await deleteHabit(id);
      } catch (e) {
        console.error(e);
        showToast('error', 'Delete Failed', 'Could not delete habit');
      }
    }
  };

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">Habits</h2>
        <button className="px-3 py-1 bg-blue-500 text-white rounded hover:bg-blue-600" onClick={() => setShowCreateDialog(true)}>
          + New Habit
        </button>
      </div>
      {!habits || habits.length === 0 ? (
        <p className="text-gray-500 text-center py-8">No habits yet. Create one to start tracking!</p>
      ) : (
        <div className="space-y-2">
          {habits.map((h) => {
            const stats = habitStats[h.id];
            const streak = stats?.current_streak ?? 0;

            return (
              <div
                key={h.id}
                className={`p-3 rounded cursor-pointer ${
                  selectedHabit?.id === h.id ? 'bg-blue-100' : 'hover:bg-gray-100'
                }`}
                onClick={() => setSelectedHabit(h)}
              >
                <div className="flex justify-between items-center">
                  <div className="flex items-center gap-2">
                    {h.icon && <span className="text-xl">{h.icon}</span>}
                    <span className="font-medium">{h.title}</span>
                    {streak > 0 && (
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800">
                        🔥 {streak}
                      </span>
                    )}
                  </div>
                  <button
                    className="text-red-500 hover:text-red-700 px-2"
                    onClick={(e) => { e.stopPropagation(); handleDelete(h.id); }}
                  >
                    ×
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
      {showCreateDialog && <HabitFormDialog onClose={() => setShowCreateDialog(false)} />}
    </div>
  );
};
