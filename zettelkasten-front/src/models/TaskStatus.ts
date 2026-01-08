export interface TaskStatus {
  id: number;
  user_id: number;
  name: string;
  display_name: string;
  color: string;
  icon: string;
  position: number;
  is_default: boolean;
  is_complete_state: boolean;
  created_at: Date | string;
  updated_at: Date | string;
}

export interface CreateTaskStatusParams {
  name: string;
  display_name: string;
  color: string;
  icon: string;
  position: number;
  is_default: boolean;
  is_complete_state: boolean;
}

export interface UpdateTaskStatusParams {
  display_name?: string;
  color?: string;
  icon?: string;
  position?: number;
  is_default?: boolean;
  is_complete_state?: boolean;
}

export interface ReorderTaskStatusesParams {
  status_ids: number[];
}
