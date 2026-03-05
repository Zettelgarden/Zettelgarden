import React, { useEffect } from 'react';
import { useHabits } from '../contexts/HabitContext';
import { HabitList } from '../components/habits/HabitList';
import { HabitDetail } from '../components/habits/HabitDetail';

export const Habits: React.FC = () => {
  const { fetchHabits } = useHabits();
  useEffect(() => { fetchHabits(); }, [fetchHabits]);

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="md:col-span-1"><HabitList /></div>
        <div className="md:col-span-2"><HabitDetail /></div>
      </div>
    </div>
  );
};
