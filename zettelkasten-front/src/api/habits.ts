import { apiClient } from './client';
import { Habit, HabitWithCheckin, HabitLog, HabitStats, CreateHabitParams, UpdateHabitParams, CheckinHabitParams } from '../models/habit';

/**
 * Get all habits for the current user
 */
export async function getHabits(): Promise<Habit[]> {
  const { data } = await apiClient.get<Habit[]>('/habits');
  return data;
}

/**
 * Get habits due today with check-in status
 */
export async function getTodaysHabits(): Promise<HabitWithCheckin[]> {
  const { data } = await apiClient.get<HabitWithCheckin[]>('/habits/today');
  return data;
}

/**
 * Get a single habit by ID
 */
export async function getHabit(id: number): Promise<Habit> {
  const { data } = await apiClient.get<Habit>(`/habits/${id}`);
  return data;
}

/**
 * Get statistics for a specific habit
 */
export async function getHabitStats(id: number): Promise<HabitStats> {
  const { data } = await apiClient.get<HabitStats>(`/habits/${id}/stats`);
  return data;
}

/**
 * Get check-in logs for a specific habit
 */
export async function getHabitLogs(id: number, limit?: number, offset?: number): Promise<HabitLog[]> {
  const params: Record<string, number | undefined> = {};
  if (limit !== undefined) params.limit = limit;
  if (offset !== undefined) params.offset = offset;

  const { data } = await apiClient.get<HabitLog[]>(`/habits/${id}/logs`, { params });
  return data;
}

/**
 * Create a new habit
 */
export async function createHabit(params: CreateHabitParams): Promise<number> {
  const { data } = await apiClient.post<{ id: number }>('/habits', params);
  return data.id;
}

/**
 * Update an existing habit
 */
export async function updateHabit(id: number, params: UpdateHabitParams): Promise<Habit> {
  const { data } = await apiClient.put<Habit>(`/habits/${id}`, params);
  return data;
}

/**
 * Delete a habit
 */
export async function deleteHabit(id: number): Promise<void> {
  await apiClient.delete(`/habits/${id}`);
}

/**
 * Check in to a habit (mark as complete for today)
 */
export async function checkinHabit(id: number, params?: CheckinHabitParams): Promise<number> {
  const { data } = await apiClient.post<{ id: number }>(`/habits/${id}/checkin`, params || {});
  return data.id;
}

/**
 * Undo a habit check-in
 */
export async function undoCheckin(id: number, logId: number): Promise<void> {
  await apiClient.delete(`/habits/${id}/checkin/${logId}`);
}
