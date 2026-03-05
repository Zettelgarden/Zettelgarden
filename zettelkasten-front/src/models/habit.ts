export interface Habit {
  id: number;
  user_id: number;
  title: string;
  description?: string;
  frequency: 'daily' | 'weekly' | 'custom_days';
  custom_days?: string;
  icon?: string;
  color?: string;
  position: number;
  linked_task_id?: number;
  created_at: string;
  updated_at: string;
  today_checked_in?: boolean;
  current_streak?: number;
}

export interface HabitWithCheckin extends Habit {
  is_due_today: boolean;
  checked_in_today: boolean;
  today_log_id?: number;
}

export interface HabitLog {
  id: number;
  habit_id: number;
  user_id: number;
  completed_at: string;
  notes?: string;
  created_at: string;
}

export interface HabitStats {
  current_streak: number;
  longest_streak: number;
  total_completions: number;
  completion_rate_7d: number;
  completion_rate_30d: number;
  last_completed_at?: string;
}

export interface CreateHabitParams {
  title: string;
  description?: string;
  frequency: 'daily' | 'weekly' | 'custom_days';
  custom_days?: number[];
  icon?: string;
  color?: string;
  linked_task_id?: number;
}

export interface UpdateHabitParams {
  title?: string;
  description?: string;
  frequency?: 'daily' | 'weekly' | 'custom_days';
  custom_days?: number[];
  icon?: string;
  color?: string;
  linked_task_id?: number;
}

export interface CheckinHabitParams {
  notes?: string;
}
