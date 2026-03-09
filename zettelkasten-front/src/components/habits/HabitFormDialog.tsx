import React, { useState, useEffect, useMemo } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { CreateHabitParams, UpdateHabitParams, Habit } from '../../models/habit';
import { HABIT_ICONS, HABIT_COLORS } from './constants';
import { ConfirmDialog } from '../tasks/ConfirmDialog';

interface HabitFormDialogProps {
  onClose: () => void;
  habit?: Habit; // If provided, we're editing. If not, we're creating.
}

export const HabitFormDialog: React.FC<HabitFormDialogProps> = ({ onClose, habit }) => {
  const { createHabit, updateHabit } = useHabits();
  const isEditing = !!habit;
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  const [title, setTitle] = useState(habit?.title || '');
  const [description, setDescription] = useState(habit?.description || '');
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'custom_days'>(
    habit?.frequency || 'daily'
  );
  const [customDays, setCustomDays] = useState<number[]>(() => {
    if (habit?.custom_days) {
      try {
        return JSON.parse(habit.custom_days);
      } catch {
        return [];
      }
    }
    return [];
  });
  const [icon, setIcon] = useState(habit?.icon || '');
  const [color, setColor] = useState(habit?.color || '#10b981');

  // Check if form has unsaved changes
  const hasChanges = useMemo(() => {
    if (isEditing && habit) {
      return (
        title !== (habit.title || '') ||
        description !== (habit.description || '') ||
        frequency !== habit.frequency ||
        JSON.stringify(customDays) !== (habit.custom_days || '[]') ||
        icon !== (habit.icon || '') ||
        color !== (habit.color || '#10b981')
      );
    }
    // For new habit, check if any field has been modified from defaults
    return (
      title !== '' ||
      description !== '' ||
      frequency !== 'daily' ||
      customDays.length > 0 ||
      icon !== '' ||
      color !== '#10b981'
    );
  }, [isEditing, habit, title, description, frequency, customDays, icon, color]);

  const handleClose = () => {
    if (hasChanges) {
      setShowCloseConfirm(true);
    } else {
      onClose();
    }
  };

  const confirmClose = () => {
    setShowCloseConfirm(false);
    onClose();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const customDaysStr = frequency === 'custom_days' && customDays.length > 0
      ? JSON.stringify(customDays)
      : undefined;

    if (isEditing && habit) {
      const params: UpdateHabitParams = {
        title,
        description: description || undefined,
        frequency,
        custom_days: customDaysStr,
        icon: icon || undefined,
        color,
      };
      try {
        await updateHabit(habit.id, params);
        onClose();
      } catch (err) {
        console.error(err);
      }
    } else {
      const params: CreateHabitParams = {
        title,
        description: description || undefined,
        frequency,
        custom_days: customDaysStr,
        icon: icon || undefined,
        color,
      };
      try {
        await createHabit(params);
        onClose();
      } catch (err) {
        console.error(err);
      }
    }
  };

  const toggleDay = (day: number) => {
    if (customDays.includes(day)) {
      setCustomDays(customDays.filter((d) => d !== day));
    } else {
      setCustomDays([...customDays, day]);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={handleClose}>
      <div className="bg-white rounded-lg p-6 w-full max-w-md" onClick={(e) => e.stopPropagation()}>
        <h2 className="text-xl font-bold mb-4">
          {isEditing ? 'Edit Habit' : 'Create New Habit'}
        </h2>
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Title *</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full border rounded px-3 py-2"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full border rounded px-3 py-2"
              rows={2}
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Frequency</label>
            <select
              value={frequency}
              onChange={(e) => setFrequency(e.target.value as any)}
              className="w-full border rounded px-3 py-2"
            >
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="custom_days">Specific Days</option>
            </select>
          </div>

          {frequency === 'custom_days' && (
            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">Select Days</label>
              <div className="flex gap-2 flex-wrap">
                {['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((day, i) => (
                  <button
                    key={day}
                    type="button"
                    className={`px-3 py-1 rounded text-sm ${
                      customDays.includes(i + 1) ? 'bg-blue-500 text-white' : 'bg-gray-200'
                    }`}
                    onClick={() => toggleDay(i + 1)}
                  >
                    {day}
                  </button>
                ))}
              </div>
            </div>
          )}

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Icon</label>
            <div className="flex gap-2 flex-wrap">
              {HABIT_ICONS.map((i) => (
                <button
                  key={i}
                  type="button"
                  className={`text-2xl p-2 rounded ${icon === i ? 'bg-gray-200' : ''}`}
                  onClick={() => setIcon(i)}
                >
                  {i}
                </button>
              ))}
            </div>
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Color</label>
            <div className="flex gap-2">
              {HABIT_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  className={`w-8 h-8 rounded ${color === c ? 'ring-2 ring-offset-2' : ''}`}
                  style={{ backgroundColor: c }}
                  onClick={() => setColor(c)}
                />
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-2">
            <button type="button" className="px-4 py-2 border rounded" onClick={handleClose}>
              Cancel
            </button>
            <button type="submit" className="px-4 py-2 bg-blue-500 text-white rounded">
              {isEditing ? 'Save Changes' : 'Create'}
            </button>
          </div>
        </form>
      </div>

      <ConfirmDialog
        isOpen={showCloseConfirm}
        onClose={() => setShowCloseConfirm(false)}
        onConfirm={confirmClose}
        title="Discard Changes?"
        message="You have unsaved changes. Are you sure you want to close without saving?"
        confirmText="Discard"
        cancelText="Keep Editing"
        variant="warning"
      />
    </div>
  );
};
