import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { Habit, HabitWithCheckin, HabitLog, HabitStats, CreateHabitParams, UpdateHabitParams, CheckinHabitParams } from '../models/habit';
import * as habitApi from '../api/habits';

interface HabitContextType {
  habits: Habit[];
  todaysHabits: HabitWithCheckin[];
  selectedHabit: Habit | null;
  habitStats: Record<number, HabitStats>;
  habitLogs: Record<number, HabitLog[]>;
  loading: boolean;
  error: string | null;
  fetchHabits: () => Promise<void>;
  fetchTodaysHabits: () => Promise<void>;
  fetchHabitStats: (id: number) => Promise<HabitStats>;
  fetchHabitLogs: (id: number, limit?: number) => Promise<HabitLog[]>;
  createHabit: (params: CreateHabitParams) => Promise<number>;
  updateHabit: (id: number, params: UpdateHabitParams) => Promise<Habit>;
  deleteHabit: (id: number) => Promise<void>;
  checkinHabit: (id: number, params?: CheckinHabitParams) => Promise<number>;
  undoCheckin: (habitId: number, logId: number) => Promise<void>;
  setSelectedHabit: (habit: Habit | null) => void;
}

const HabitContext = createContext<HabitContextType | undefined>(undefined);

export const HabitProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [habits, setHabits] = useState<Habit[]>([]);
  const [todaysHabits, setTodaysHabits] = useState<HabitWithCheckin[]>([]);
  const [selectedHabit, setSelectedHabit] = useState<Habit | null>(null);
  const [habitStats, setHabitStats] = useState<Record<number, HabitStats>>({});
  const [habitLogs, setHabitLogs] = useState<Record<number, HabitLog[]>>({});
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

  const fetchHabitLogs = useCallback(async (id: number, limit: number = 30): Promise<HabitLog[]> => {
    try {
      const logs = await habitApi.getHabitLogs(id, limit);
      setHabitLogs(prev => ({ ...prev, [id]: logs }));
      return logs;
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

  const updateHabit = useCallback(async (id: number, params: UpdateHabitParams): Promise<Habit> => {
    setLoading(true);
    try {
      const updated = await habitApi.updateHabit(id, params);
      await fetchHabits();
      await fetchTodaysHabits();
      // Update selectedHabit if it's the one we edited
      setSelectedHabit(prev => prev?.id === id ? updated : prev);
      return updated;
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
      return logId;
    } catch (err) {
      throw err;
    } finally {
      // Always refresh data after check-in attempt (success or failure)
      setLoading(false);
      await fetchTodaysHabits();
      await fetchHabitStats(id);
      await fetchHabitLogs(id);
    }
  }, [fetchTodaysHabits, fetchHabitStats, fetchHabitLogs]);

  const undoCheckin = useCallback(async (habitId: number, logId: number): Promise<void> => {
    setLoading(true);
    try {
      await habitApi.undoCheckin(habitId, logId);
      await fetchTodaysHabits();
      // Refresh stats and logs after undo
      await fetchHabitStats(habitId);
      await fetchHabitLogs(habitId);
    } catch (err) {
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchTodaysHabits, fetchHabitStats, fetchHabitLogs]);

  const value: HabitContextType = {
    habits, todaysHabits, selectedHabit, habitStats, habitLogs, loading, error,
    fetchHabits, fetchTodaysHabits, fetchHabitStats, fetchHabitLogs, createHabit, updateHabit, deleteHabit, checkinHabit, undoCheckin, setSelectedHabit,
  };

  return <HabitContext.Provider value={value}>{children}</HabitContext.Provider>;
};

export const useHabits = () => {
  const context = useContext(HabitContext);
  if (!context) throw new Error('useHabits must be used within HabitProvider');
  return context;
};
