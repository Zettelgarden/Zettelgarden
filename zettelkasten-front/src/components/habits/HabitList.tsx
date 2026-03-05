import React, { useState } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { CreateHabitDialog } from './CreateHabitDialog';

export const HabitList: React.FC = () => {
  const { habits, selectedHabit, setSelectedHabit, deleteHabit } = useHabits();
  const [showCreateDialog, setShowCreateDialog] = useState(false);

  const handleDelete = async (id: number) => {
    if (confirm('Delete this habit?')) {
      try {
        await deleteHabit(id);
      } catch (e) {
        console.error(e);
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
          {habits.map((h) => (
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
                </div>
                <button
                  className="text-red-500 hover:text-red-700 px-2"
                  onClick={(e) => { e.stopPropagation(); handleDelete(h.id); }}
                >
                  ×
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
      {showCreateDialog && <CreateHabitDialog onClose={() => setShowCreateDialog(false)} />}
    </div>
  );
};
