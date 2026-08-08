import { DailyStatsResponse } from '../models/Stats';
import { Task } from '../models/Task';
import { PartialCard } from '../models/Card';
import { apiClient, getData } from './client';

function formatLocalDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export function fetchDailyStats(
  startDate?: Date,
  endDate?: Date,
): Promise<DailyStatsResponse> {
  const params: Record<string, string | undefined> = {};

  if (startDate) {
    params.start_date = formatLocalDate(startDate);
  }
  if (endDate) {
    params.end_date = formatLocalDate(endDate);
  }

  return getData(apiClient.get<any>('/stats/daily', { params })).then(
    (data) => ({
      ...data,
      stats: data.stats.map((stat: any) => ({
        ...stat,
        date: new Date(stat.date),
      })),
    }),
  );
}

export function fetchTasksForDate(date: Date): Promise<Task[]> {
  const params: Record<string, string> = {
    date: formatLocalDate(date),
  };

  return getData(apiClient.get<any[]>('/stats/day-tasks', { params })).then(
    (tasks) =>
      tasks.map((task) => ({
        ...task,
        scheduled_date: task.scheduled_date
          ? new Date(task.scheduled_date)
          : null,
        due_date: task.due_date ? new Date(task.due_date) : null,
        created_at: new Date(task.created_at),
        updated_at: new Date(task.updated_at),
        completed_at: task.completed_at ? new Date(task.completed_at) : null,
      })),
  );
}

export function fetchCardsForDate(date: Date): Promise<PartialCard[]> {
  const params: Record<string, string> = {
    date: formatLocalDate(date),
  };

  return getData(apiClient.get<any[]>('/stats/day-cards', { params })).then(
    (cards) =>
      cards.map((card) => ({
        ...card,
        created_at: new Date(card.created_at),
        updated_at: new Date(card.updated_at),
      })),
  );
}
