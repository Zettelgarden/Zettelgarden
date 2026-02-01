import { Task } from "../models/Task";
import { Tag } from "../models/Tags";
import { Card, PartialCard } from "../models/Card";
import { Entity } from "../models/Card";

export function sampleTasks(): Task[] {
  return sampleTaskData;
}
export function sampleCards(): Card[] {
  return sampleCardData;
}
export function samplePartialCards(): PartialCard[] {
  return samplePartialCardData;
}
export function sampleTags(): Tag[] {
  return sampleTagData;
}

export const sampleTagData: Tag[] = [
  {
    id: 1,
    name: "report",
    color: "black",
    user_id: 1,
  },
];


export const sampleTaskData: Task[] = [
  {
    id: 1,
    card_pk: 101,
    user_id: 1001,
    scheduled_date: new Date(new Date().setDate(new Date().getDate() + 1)), // Tomorrow
    due_date: null, // Assuming dueDate is not provided in Swift data
    created_at: new Date(),
    updated_at: new Date(),
    completed_at: null,
    title: "Daily Standup Meeting #is #hi http://google.com",
    description: null,
    status: 'todo' as const,
    is_complete: false,
    is_deleted: false,
    card: null, // Or provide a mock PartialCard if needed
    tags: [],
    priority: null,
    reminder_time: null,
    reminder_sent: false,
    blocked_by: [],
    blocks: [],
  },
  {
    id: 2,
    card_pk: 102,
    user_id: 1001,
    scheduled_date: new Date(), // Today
    due_date: null,
    created_at: new Date(),
    updated_at: new Date(),
    completed_at: null,
    title: "Weekly Sync-up #recurring",
    description: null,
    status: 'todo' as const,
    is_complete: false,
    is_deleted: false,
    card: null,
    tags: [],
    priority: null,
    reminder_time: null,
    reminder_sent: false,
    blocked_by: [],
    blocks: [],
  },
  {
    id: 3,
    card_pk: 103,
    user_id: 1002,
    scheduled_date: new Date(new Date().setDate(new Date().getDate() - 2)), // 2 days ago
    due_date: null,
    created_at: new Date(),
    updated_at: new Date(),
    completed_at: null,
    title: "Write Quarterly Work Report #report",
    description: null,
    status: 'todo' as const,
    is_complete: false,
    is_deleted: false,
    card: null,
    tags: [],
    priority: null,
    reminder_time: null,
    reminder_sent: false,
    blocked_by: [],
    blocks: [],
  },
  {
    id: 4,
    card_pk: 104,
    user_id: 1003,
    scheduled_date: new Date(new Date().setDate(new Date().getDate() - 7)), // 7 days ago
    due_date: null,
    created_at: new Date(),
    updated_at: new Date(),
    completed_at: new Date(), // Completed
    title: "Submit Expense Reports #work #task",
    description: null,
    status: 'done' as const,
    is_complete: true,
    is_deleted: false,
    card: null,
    priority: null,
    reminder_time: null,
    reminder_sent: false,
    tags: [
      {
        id: 2,
        name: "work",
        color: "black",
        user_id: 1,
      },
      {
        id: 4,
        name: "task",
        color: "black",
        user_id: 1,
      },
    ],
    blocked_by: [],
    blocks: [],
  },
  {
    id: 5,
    card_pk: 105,
    user_id: 1004,
    scheduled_date: null, // No scheduled date
    due_date: null,
    created_at: new Date(),
    updated_at: new Date(),
    completed_at: null,
    title: "Brainstorm Session #work #todo",
    description: null,
    status: 'todo' as const,
    is_complete: false,
    is_deleted: false,
    card: null,
    priority: null,
    reminder_time: null,
    reminder_sent: false,
    tags: [
      {
        id: 2,
        name: "work",
        color: "black",
        user_id: 1,
      },
      {
        id: 3,
        name: "todo",
        color: "black",
        user_id: 1,
      },
    ],
    blocked_by: [],
    blocks: [],
  },
];
export const sampleEntityData: Entity[] = [
  {
    id: 1,
    user_id: 1,
    name: "Entity One",
    description: "Description for entity one",
    type: "Type A",
    created_at: new Date(),
    updated_at: new Date(),
    card_count: 1,
    card_pk: null
  },
  {
    id: 2,
    user_id: 1,
    name: "Entity Two",
    description: "Description for entity two",
    type: "Type B",
    created_at: new Date(),
    updated_at: new Date(),
    card_count: 2,
    card_pk: null
  },
];

const samplePartialCardData: PartialCard[] = [
  {
    id: 1,
    card_id: "A.1",
    user_id: 1,
    title: "Introduction to Machine Learning",
    parent_id: 0,
    created_at: new Date("2024-01-15T10:00:00Z"),
    updated_at: new Date("2024-01-15T10:00:00Z"),
    tags: [
      {
        id: 1,
        name: "ML",
        color: "blue",
        user_id: 1,
      },
      {
        id: 2,
        name: "study",
        color: "green",
        user_id: 1,
      },
    ],
  },
  {
    id: 2,
    card_id: "A.1/A",
    user_id: 1,
    title: "Supervised Learning Algorithms",
    parent_id: 1,
    created_at: new Date("2024-01-16T11:00:00Z"),
    updated_at: new Date("2024-01-16T11:00:00Z"),
    tags: [],
  },
  {
    id: 3,
    card_id: "B.1",
    user_id: 1,
    title: "Card with Long Title That Should Test Text Wrapping and Display",
    parent_id: 0,
    created_at: new Date("2024-01-17T12:00:00Z"),
    updated_at: new Date("2024-01-17T12:00:00Z"),
    tags: [
      {
        id: 3,
        name: "test",
        color: "red",
        user_id: 1,
      },
    ],
  },
];

const sampleCardData: Card[] = [
  {
    id: 1,
    card_id: "1",
    user_id: 1,
    title: "hello world",
    body: "this is a test of the emergency response system",
    link: "",
    created_at: new Date(),
    updated_at: new Date(),
    parent_id: 1,
    parent: samplePartialCardData[0],
    children: [],
    references: [],
    files: [],
    is_deleted: false,
    tags: [],
    tasks: [],
    external_events: [],
    entities: sampleEntityData,
  },
  {
    id: 2,
    card_id: "1/A",
    user_id: 1,
    title: "update",
    body: "this is another test of the emergency response system",
    link: "",
    created_at: new Date(),
    updated_at: new Date(),
    parent_id: 2,
    parent: samplePartialCardData[1],
    children: [],
    references: [],
    files: [],
    is_deleted: false,
    tags: [],
    tasks: [],
    external_events: [],
    entities: sampleEntityData,
  },
];


