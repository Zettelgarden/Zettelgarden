import { PartialCard } from "./Card";
import { Tag } from "./Tags";

interface AuditChange {
  from: any;
  to: any;
}

interface AuditDetails {
  change_type: string;
  changes: {
    [key: string]: AuditChange;
  };
}

export interface TaskAuditEvent {
  id: number;
  user_id: number;
  entity_id: number;
  entity_type: string;
  action: string;
  details: AuditDetails;
  created_at: string | Date;
}

export type TaskStatus = 'todo' | 'in_progress' | 'blocked' | 'done';

export interface Task {
  id: number;
  card_pk: number;
  user_id: number;
  scheduled_date: Date | null;
  dueDate: Date | null;
  created_at: Date;
  updated_at: Date;
  completed_at: Date | null;
  title: string;
  priority: string | null;
  status: TaskStatus;
  is_complete: boolean;
  is_deleted: boolean;
  card: PartialCard | null;
  tags: Tag[];
}

export interface TasksResponse {
  tasks: Task[];
  total: number;
  limit: number;
  offset: number;
}

export const emptyTask: Task = {
  id: 0,
  card_pk: 0,
  user_id: 0,
  created_at: new Date(0),
  updated_at: new Date(0),
  dueDate: null,
  scheduled_date: new Date(),
  completed_at: null,
  title: "",
  priority: null,
  status: 'todo',
  is_complete: false,
  is_deleted: false,
  card: null,
  tags: [],
};
