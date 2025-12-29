import { checkStatus } from "./common";
import { DailyStatsResponse } from "../models/Stats";
import { Task } from "../models/Task";
import { PartialCard } from "../models/Card";

const base_url = import.meta.env.VITE_URL;

export function fetchDailyStats(
  startDate?: Date,
  endDate?: Date
): Promise<DailyStatsResponse> {
  const token = localStorage.getItem("token");

  let url = `${base_url}/stats/daily`;
  const params = new URLSearchParams();

  if (startDate) {
    params.append("start_date", startDate.toISOString().split("T")[0]);
  }
  if (endDate) {
    params.append("end_date", endDate.toISOString().split("T")[0]);
  }

  if (params.toString()) {
    url += `?${params.toString()}`;
  }

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((data: any) => ({
          ...data,
          stats: data.stats.map((stat: any) => ({
            ...stat,
            date: new Date(stat.date),
          })),
        }));
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function fetchTasksForDate(date: Date): Promise<Task[]> {
  const token = localStorage.getItem("token");
  const dateStr = date.toISOString().split("T")[0];
  const url = `${base_url}/stats/day-tasks?date=${dateStr}`;

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((tasks: any[]) =>
          tasks.map((task) => ({
            ...task,
            scheduled_date: task.scheduled_date
              ? new Date(task.scheduled_date)
              : null,
            due_date: task.due_date ? new Date(task.due_date) : null,
            created_at: new Date(task.created_at),
            updated_at: new Date(task.updated_at),
            completed_at: task.completed_at
              ? new Date(task.completed_at)
              : null,
          }))
        );
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function fetchCardsForDate(date: Date): Promise<PartialCard[]> {
  const token = localStorage.getItem("token");
  const dateStr = date.toISOString().split("T")[0];
  const url = `${base_url}/stats/day-cards?date=${dateStr}`;

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((cards: any[]) =>
          cards.map((card) => ({
            ...card,
            created_at: new Date(card.created_at),
            updated_at: new Date(card.updated_at),
          }))
        );
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}
