import type { SortField, SortDirection, ViewMode } from '../types/taskPage';

/**
 * A user-saved task search, synced across devices via the backend.
 * Captures the full state of the Tasks page filter bar so a saved search
 * can be re-applied exactly: the filter string plus sort and view mode.
 */
export interface TaskSavedSearch {
  id: number;
  user_id: number;
  name: string;
  filter_string: string;
  sort_field: SortField;
  sort_direction: SortDirection;
  view_mode: ViewMode;
  created_at: string;
  updated_at: string;
}

export interface CreateTaskSavedSearchParams {
  name: string;
  filter_string: string;
  sort_field: SortField;
  sort_direction: SortDirection;
  view_mode: ViewMode;
}

export interface UpdateTaskSavedSearchParams {
  name?: string;
  filter_string?: string;
  sort_field?: SortField;
  sort_direction?: SortDirection;
  view_mode?: ViewMode;
}
