export interface DailyStats {
  date: Date;
  cards_created: number;
  tasks_created: number;
  tasks_completed: number;
}

export interface DailyStatsResponse {
  stats: DailyStats[];
  total: {
    cards_created: number;
    tasks_created: number;
    tasks_completed: number;
  };
}
