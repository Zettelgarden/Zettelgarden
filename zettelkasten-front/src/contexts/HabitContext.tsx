import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { Habit, HabitWithCheckin, HabitLog, HabitStats, CreateHabitParams, UpdateHabitParams, CheckinHabitParams } from '../models/habit';
import * as habitApi from '../api/habits';

interface HabitContextType {
  habits: Habit[];
  todaysHabits: HabitWithCheckin[];
  selectedHabit: Habit | null;
  habitStats: Record<number, HabitStats>;
  loading: boolean;
  error: string | null;
  fetchHabits: () => Promise<void>;
  fetchTodaysHabits: () => Promise<void>;
  fetchHabitStats: (id: number) => Promise<HabitStats>;
  createHabit: (params: CreateHabitParams) => Promise<number>;
  deleteHabit: (id: number) => Promise<void>;
  checkinHabit: (id: number, params?: CheckinHabitParams) => Promise<number>;
  setSelectedHabit: (habit: Habit | null) => void;
}

const HabitContext = createContext<HabitContextType | undefined>(undefined);

export const HabitProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [habits, setHabits] = useState<Habit[]>([]);
  const [todaysHabits, setTodaysHabits] = useState<HabitWithCheckin[]>([]);
  const [selectedHabit, setSelectedHabit] = useState<Habit | null>(null);
  const [habitStats, setHabitStats] = useState<Record<number, HabitStats>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHabits = useCallback(async () => {
    setLoading(true);
    try {
      const data = await habitApi.getHabits();
      setHabits(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchTodaysHabits = useCallback(async () => {
    setLoading(true);
    try {
      const data = await habitApi.getTodaysHabits();
      setTodaysHabits(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchHabitStats = useCallback(async (id: number): Promise<HabitStats> => {
    try {
      const stats = await habitApi.getHabitStats(id);
      setHabitStats(prev => ({ ...prev, [id]: stats }));
      return stats;
    } catch (err) {
      throw err;
    }
  }, []);

  const createHabit = useCallback(async (params: CreateHabitParams): Promise<number> => {
    setLoading(true);
    try {
      const id = await habitApi.createHabit(params);
      await fetchHabits();
      await fetchTodaysHabits();
      return id;
    } catch (err) {
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchHabits, fetchTodaysHabits]);

  const deleteHabit = useCallback(async (id: number) => {
    setLoading(true);
    try {
      await habitApi.deleteHabit(id);
      await fetchHabits();
      await fetchTodaysHabits();
    } catch (err) {
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchHabits, fetchTodaysHabits]);

  const checkinHabit = useCallback(async (id: number, params?: CheckinHabitParams): Promise<number> => {
    setLoading(true);
    try {
      const logId = await habitApi.checkinHabit(id, params);
      await fetchTodaysHabits();
      // Refresh stats after check-in
      await fetchHabitStats(id);
      return logId;
    } catch (err) {
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchTodaysHabits, fetchHabitStats]);

  const value: HabitContextType = {
    habits, todaysHabits, selectedHabit, habitStats, loading, error,
    fetchHabits, fetchTodaysHabits, fetchHabitStats, createHabit, deleteHabit, checkinHabit, setSelectedHabit,
  };

  return <HabitContext.Provider value={value}>{children}</HabitContext.Provider>;
};

export const useHabits = () => {
  const context = useContext(HabitContext);
  if (!context) throw new Error('useHabits must be used within HabitProvider');
  return context;
};
