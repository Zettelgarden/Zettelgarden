import React, { useEffect, useState } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { useToast } from '../toast/ToastContext';
import { HabitCalendar } from './HabitCalendar';
import { HabitHistory } from './HabitHistory';
import { HabitFormDialog } from './HabitFormDialog';

type TabType = 'checkin' | 'history';

export const HabitDetail: React.FC = () => {
  const { selectedHabit, checkinHabit, habitLogs, fetchHabitLogs, undoCheckin, habitStats } = useHabits();
  const { showToast } = useToast();
  const [activeTab, setActiveTab] = useState<TabType>('checkin');
  const [showEditDialog, setShowEditDialog] = useState(false);

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

  const handleUndoCheckin = async (logId: number) => {
    if (!selectedHabit) return;
    if (!confirm('Undo this check-in?')) return;
    try {
      await undoCheckin(selectedHabit.id, logId);
      showToast('success', 'Check-in undone', 'The check-in has been removed');
    } catch (e) {
      showToast('error', 'Failed to undo', 'Could not undo the check-in');
    }
  };

  if (!selectedHabit) {
    return <div className="p-4 text-center text-gray-500">Select a habit to view details</div>;
  }

  const logs = habitLogs[selectedHabit.id] || [];
  const stats = habitStats[selectedHabit.id];

  return (
    <div className="p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
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
            className="w-full py-3 bg-green-500 text-white rounded-lg font-medium hover:bg-green-600 transition-colors"
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
    </div>
  );
};
