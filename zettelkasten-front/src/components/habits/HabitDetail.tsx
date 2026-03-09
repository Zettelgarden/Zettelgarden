import React, { useEffect, useState } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { useToast } from '../toast/ToastContext';
import { HabitCalendar } from './HabitCalendar';
import { HabitHistory } from './HabitHistory';
import { HabitFormDialog } from './HabitFormDialog';
import { ConfirmDialog } from '../tasks/ConfirmDialog';

type TabType = 'checkin' | 'history';

interface UndoConfirmState {
  logId: number;
  date: string;
}

export const HabitDetail: React.FC = () => {
  const { selectedHabit, checkinHabit, habitLogs, fetchHabitLogs, undoCheckin, habitStats, loading } = useHabits();
  const { showToast } = useToast();
  const [activeTab, setActiveTab] = useState<TabType>('checkin');
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [undoConfirm, setUndoConfirm] = useState<UndoConfirmState | null>(null);

  // Fetch logs when habit is selected
  useEffect(() => {
    if (selectedHabit) {
      fetchHabitLogs(selectedHabit.id, 50).catch(console.error);
    }
  }, [selectedHabit, fetchHabitLogs]);

  const handleCheckin = async () => {
    if (!selectedHabit) return;
    try {
      await checkinHabit(selectedHabit.id);
      showToast('success', 'Check-in complete!', `Great work on ${selectedHabit.title}`);
    } catch (e) {
      if (e instanceof Error && e.message.includes('already')) {
        showToast('warning', 'Already checked in', "You've already checked in today!");
      }
    }
  };

  const handleUndoCheckin = async (logId: number, date: string) => {
    if (!selectedHabit) return;
    setUndoConfirm({ logId, date });
  };

  const confirmUndoCheckin = async () => {
    if (!selectedHabit || !undoConfirm) return;
    try {
      await undoCheckin(selectedHabit.id, undoConfirm.logId);
      showToast('success', 'Check-in undone', 'The check-in has been removed');
      setUndoConfirm(null);
    } catch (e) {
      showToast('error', 'Failed to undo', 'Could not undo the check-in');
    }
  };

  if (!selectedHabit) {
    return (
      <div className="p-8 text-center">
        <div className="text-4xl mb-3">👈</div>
        <h3 className="text-lg font-medium text-gray-700 mb-2">Select a Habit</h3>
        <p className="text-gray-500 max-w-xs mx-auto">
          Choose a habit from the list to view details, check in, and see your progress.
        </p>
      </div>
    );
  }

  // Show loading state when fetching habit data
  const logsLoading = selectedHabit && !habitLogs[selectedHabit.id] && loading;
  if (logsLoading) {
    return (
      <div className="p-4">
        <div className="animate-pulse">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-gray-200 rounded" />
            <div className="h-6 bg-gray-200 rounded w-40" />
          </div>
          <div className="h-4 bg-gray-200 rounded w-full mb-4" />
          <div className="h-32 bg-gray-200 rounded" />
        </div>
      </div>
    );
  }

  const logs = habitLogs[selectedHabit.id] || [];
  const stats = habitStats[selectedHabit.id];

  return (
    <div className="p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          {/* Color indicator */}
          <div
            className="w-1 h-10 rounded-full"
            style={{ backgroundColor: selectedHabit.color || '#10b981' }}
          />
          {selectedHabit.icon && <span className="text-3xl">{selectedHabit.icon}</span>}
          <div>
            <h2 className="text-2xl font-bold">{selectedHabit.title}</h2>
            {stats && stats.current_streak > 0 && (
              <p className="text-sm text-amber-600">🔥 {stats.current_streak} day streak</p>
            )}
          </div>
        </div>
        <button
          onClick={() => setShowEditDialog(true)}
          className="px-3 py-1 text-sm border rounded hover:bg-gray-100"
        >
          Edit
        </button>
      </div>

      {selectedHabit.description && (
        <p className="text-gray-600 mb-4">{selectedHabit.description}</p>
      )}

      {/* Tabs */}
      <div className="flex border-b border-gray-200 mb-4">
        <button
          onClick={() => setActiveTab('checkin')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'checkin'
              ? 'border-blue-500 text-blue-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          Check In
        </button>
        <button
          onClick={() => setActiveTab('history')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
            activeTab === 'history'
              ? 'border-blue-500 text-blue-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          History
        </button>
      </div>

      {/* Tab Content */}
      {activeTab === 'checkin' && (
        <div className="space-y-4">
          <button
            className="w-full py-3 text-white rounded-lg font-medium transition-colors"
            style={{ backgroundColor: selectedHabit.color || '#10b981' }}
            onClick={handleCheckin}
          >
            Check In Today
          </button>

          {/* Mini Calendar */}
          <div>
            <h3 className="text-sm font-medium text-gray-700 mb-2">This Month</h3>
            <HabitCalendar logs={logs} />
          </div>
        </div>
      )}

      {activeTab === 'history' && (
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-2">Recent Check-ins</h3>
          <HabitHistory logs={logs} onUndoCheckin={handleUndoCheckin} maxItems={15} />
        </div>
      )}

      {/* Edit Dialog */}
      {showEditDialog && (
        <HabitFormDialog
          habit={selectedHabit}
          onClose={() => setShowEditDialog(false)}
        />
      )}

      {/* Undo Check-in Confirmation */}
      <ConfirmDialog
        isOpen={!!undoConfirm}
        onClose={() => setUndoConfirm(null)}
        onConfirm={confirmUndoCheckin}
        title="Undo Check-in"
        message={`Are you sure you want to undo the check-in from ${undoConfirm?.date}? This will affect your streak.`}
        confirmText="Undo"
        cancelText="Keep It"
        variant="warning"
      />
    </div>
  );
};
