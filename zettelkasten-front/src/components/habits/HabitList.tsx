import React, { useState, useEffect } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { HabitFormDialog } from './HabitFormDialog';
import { useToast } from '../toast/ToastContext';
import { formatDistanceToNow, parseISO } from 'date-fns';
import { ConfirmDialog } from '../tasks/ConfirmDialog';

export const HabitList: React.FC = () => {
  const { habits, selectedHabit, setSelectedHabit, deleteHabit, habitStats, fetchHabitStats, loading } = useHabits();
  const { showToast } = useToast();
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<{ id: number; title: string } | null>(null);

  // Fetch stats for all habits on mount
  useEffect(() => {
    habits.forEach(h => {
      fetchHabitStats(h.id).catch(() => {}); // Silently ignore if already fetched
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [habits]);

  const handleDelete = async () => {
    if (!deleteConfirm) return;
    try {
      await deleteHabit(deleteConfirm.id);
      setDeleteConfirm(null);
    } catch (e) {
      console.error(e);
      showToast('error', 'Delete Failed', 'Could not delete habit');
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
      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="p-3 rounded bg-gray-100 animate-pulse">
              <div className="flex items-center gap-2">
                <div className="w-6 h-6 bg-gray-200 rounded" />
                <div className="h-4 bg-gray-200 rounded w-32" />
              </div>
            </div>
          ))}
        </div>
      ) : !habits || habits.length === 0 ? (
        <div className="text-center py-12">
          <div className="text-4xl mb-3">🌱</div>
          <h3 className="text-lg font-medium text-gray-700 mb-2">Start Your Journey</h3>
          <p className="text-gray-500 mb-4 max-w-xs mx-auto">
            Habits help you build consistency. Create your first habit to begin tracking your progress.
          </p>
          <button
            className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-sm font-medium"
            onClick={() => setShowCreateDialog(true)}
          >
            Create Your First Habit
          </button>
        </div>
      ) : (
        <div className="space-y-2">
          {habits.map((h) => {
            const stats = habitStats[h.id];
            const streak = stats?.current_streak ?? 0;

            return (
              <div
                key={h.id}
                className={`flex rounded cursor-pointer overflow-hidden ${
                  selectedHabit?.id === h.id ? 'bg-blue-100' : 'hover:bg-gray-100'
                }`}
                onClick={() => setSelectedHabit(h)}
              >
                {/* Color indicator bar */}
                <div
                  className="w-1 flex-shrink-0"
                  style={{ backgroundColor: h.color || '#10b981' }}
                />
                <div className="p-3 flex-1">
                  <div className="flex justify-between items-center">
                    <div className="flex items-center gap-2">
                      {h.icon && <span className="text-xl">{h.icon}</span>}
                      <div className="flex flex-col">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">{h.title}</span>
                          {streak > 0 && (
                            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800">
                              🔥 {streak}
                            </span>
                          )}
                        </div>
                        {stats?.last_completed_at && (
                          <span className="text-xs text-gray-500">
                            Last: {formatDistanceToNow(parseISO(stats.last_completed_at), { addSuffix: true })}
                          </span>
                        )}
                      </div>
                    </div>
                    <button
                      className="text-red-500 hover:text-red-700 px-2"
                      onClick={(e) => { e.stopPropagation(); setDeleteConfirm({ id: h.id, title: h.title }); }}
                    >
                      ×
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
      {showCreateDialog && <HabitFormDialog onClose={() => setShowCreateDialog(false)} />}
      
      <ConfirmDialog
        isOpen={!!deleteConfirm}
        onClose={() => setDeleteConfirm(null)}
        onConfirm={handleDelete}
        title="Delete Habit"
        message={`Are you sure you want to delete "${deleteConfirm?.title}"? This will also delete all check-in history and cannot be undone.`}
        confirmText="Delete"
        cancelText="Cancel"
        variant="danger"
      />
    </div>
  );
};
