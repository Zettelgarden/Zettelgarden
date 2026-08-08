import { PartialCard } from './Card';
import { Tag } from './Tags';

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

export interface PartialTask {
  id: number;
  title: string;
  is_complete: boolean;
  status: string;
}

export interface Task {
  id: number;
  card_pk: number;
  user_id: number;
  scheduled_date: Date | null;
  due_date: Date | null;
  created_at: Date;
  updated_at: Date;
  completed_at: Date | null;
  title: string;
  description: string | null;
  priority: string | null;
  status: string; // Dynamic status based on user configuration
  is_complete: boolean;
  is_deleted: boolean;
  reminder_time: Date | null;
  reminder_sent: boolean;
  card: PartialCard | null;
  tags: Tag[];
  blocked_by: PartialTask[];
  blocks: PartialTask[];
  parent_task_id: number | null;
  sort_order: number | null;
  subtasks?: Task[];
  parent_title?: string;
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
  due_date: null,
  scheduled_date: new Date(),
  completed_at: null,
  title: '',
  description: null,
  priority: null,
  status: 'todo',
  is_complete: false,
  is_deleted: false,
  reminder_time: null,
  reminder_sent: false,
  card: null,
  tags: [],
  blocked_by: [],
  blocks: [],
  parent_task_id: null,
  sort_order: null,
  subtasks: [],
  parent_title: undefined,
};
