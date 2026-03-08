import React, { useEffect } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { useToast } from '../toast/ToastContext';

export const HabitDetail: React.FC = () => {
  const { selectedHabit, checkinHabit } = useHabits();
  const { showToast } = useToast();

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

  if (!selectedHabit) {
    return <div className="p-4 text-center text-gray-500">Select a habit to view details</div>;
  }

  return (
    <div className="p-4">
      <div className="flex items-center gap-3 mb-4">
        {selectedHabit.icon && <span className="text-3xl">{selectedHabit.icon}</span>}
        <h2 className="text-2xl font-bold">{selectedHabit.title}</h2>
      </div>
      {selectedHabit.description && <p className="text-gray-600 mb-4">{selectedHabit.description}</p>}
      <button
        className="w-full py-3 bg-green-500 text-white rounded-lg font-medium hover:bg-green-600"
        onClick={handleCheckin}
      >
        Check In
      </button>
    </div>
  );
};
