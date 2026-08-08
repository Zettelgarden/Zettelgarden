import { expect, test } from 'vitest';
import {
  removeTagsFromTitle,
  parseTags,
  filterTasks,
  parseTaskQuery,
  filterTasksByDateView,
  formatAuditEvent,
} from './tasks';
import { sampleTaskData } from '../tests/data';
import { Task, TaskAuditEvent } from '../models/Task';

test('remove tags from title', () => {
  let title = 'This is a #test title with #tags';
  expect(removeTagsFromTitle(title)).toBe('This is a  title with');
});

test('remove no tags from title', () => {
  const title = 'This is a title without tags';
  expect(removeTagsFromTitle(title)).toBe('This is a title without tags');
});

test('parse tags from title with multiple tags', () => {
  const title = 'This is a #test title with #multiple #tags';
  expect(parseTags(title)).toEqual(['#test', '#multiple', '#tags']);
});

test('parse tags from title with no tags', () => {
  const title = 'This title has no tags';
  expect(parseTags(title)).toEqual([]);
});

test('parse tags from empty string', () => {
  const title = '';
  expect(parseTags(title)).toEqual([]);
});

test('parse tags from title with only tags', () => {
  const title = '#tag1 #tag2 #tag3';
  expect(parseTags(title)).toEqual(['#tag1', '#tag2', '#tag3']);
});

test('parse tags from title with tags and punctuation', () => {
  const title = 'This is a title with #tags, and punctuation!';
  expect(parseTags(title)).toEqual(['#tags,']);
});

test('parse tags from title with mixed content', () => {
  const title = 'Some text #tag1 more text #tag2';
  expect(parseTags(title)).toEqual(['#tag1', '#tag2']);
});

test('parse tags from title with consecutive tags', () => {
  const title = 'Some text #tag1#tag2 more text';
  expect(parseTags(title)).toEqual(['#tag1#tag2']);
});

test('skip parsing tags from middle of words', () => {
  const title = 'Some text#tag1#tag2 more text';
  expect(parseTags(title)).toEqual([]);
});

test('filter tasks by tags', () => {
  const results = filterTasks(sampleTaskData, '#work session');
  expect(results.length).toEqual(1);
});

test('filter tasks by negated tags', () => {
  const tasks: Task[] = [
    {
      id: 1,
      title: 'Task 1',
      tags: [{ name: 'work', id: 1, color: '#ff0000', user_id: 1 }],
      card_pk: 1,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
      sort_order: null,
    },
    {
      id: 2,
      title: 'Task 2',
      tags: [{ name: 'personal', id: 2, color: '#00ff00', user_id: 1 }],
      card_pk: 2,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
      sort_order: null,
    },
    {
      id: 3,
      title: 'Task 3',
      tags: [
        { name: 'work', id: 1, color: '#ff0000', user_id: 1 },
        { name: 'personal', id: 2, color: '#00ff00', user_id: 1 },
      ],
      card_pk: 3,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
      sort_order: null,
    },
  ];

  const results1 = filterTasks(tasks, '!#work');
  expect(results1.length).toEqual(1);
  expect(results1[0].id).toEqual(2);

  const results2 = filterTasks(tasks, '#work !#personal');
  expect(results2.length).toEqual(1);
  expect(results2[0].id).toEqual(1);
});

test('filter tasks by negated text', () => {
  const tasks: Task[] = [
    {
      id: 1,
      title: 'Meeting with team',
      tags: [],
      card_pk: 1,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
      sort_order: null,
    },
    {
      id: 2,
      title: 'Write documentation',
      tags: [],
      card_pk: 2,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
      sort_order: null,
    },
    {
      id: 3,
      title: 'Team lunch meeting',
      tags: [],
      card_pk: 3,
      user_id: 1,
      scheduled_date: new Date(),
      due_date: null,
      status: 'todo' as const,
      is_complete: false,
      created_at: new Date(),
      updated_at: new Date(),
      completed_at: null,
      is_deleted: false,
      priority: null,
      description: null,
      card: null,
      reminder_time: null,
      reminder_sent: false,
      blocked_by: [],
      blocks: [],
      parent_task_id: null,
      sort_order: null,
    },
  ];

  const results1 = filterTasks(tasks, '!meeting');
  expect(results1.length).toEqual(1);
  expect(results1[0].id).toEqual(2);

  const results2 = filterTasks(tasks, 'team !lunch');
  expect(results2.length).toEqual(1);
  expect(results2[0].id).toEqual(1);
});

test('parseTaskQuery extracts date:none', () => {
  const result = parseTaskQuery('date:none');
  expect(result.dateView).toEqual('no_date');
  expect(result.searchTerms).toEqual([]);
});

test('parseTaskQuery extracts date:no_date', () => {
  const result = parseTaskQuery('date:no_date');
  expect(result.dateView).toEqual('no_date');
  expect(result.searchTerms).toEqual([]);
});

test('parseTaskQuery extracts date:today', () => {
  const result = parseTaskQuery('date:today');
  expect(result.dateView).toEqual('today');
  expect(result.searchTerms).toEqual([]);
});

test('parseTaskQuery extracts date:overdue', () => {
  const result = parseTaskQuery('date:overdue');
  expect(result.dateView).toEqual('overdue');
  expect(result.searchTerms).toEqual([]);
});

test('parseTaskQuery extracts date:this_week', () => {
  const result = parseTaskQuery('date:this_week');
  expect(result.dateView).toEqual('this_week');
  expect(result.searchTerms).toEqual([]);
});

test('parseTaskQuery extracts date:all', () => {
  const result = parseTaskQuery('date:all');
  expect(result.dateView).toEqual('all');
  expect(result.searchTerms).toEqual([]);
});

test('parseTaskQuery handles date filter with search terms', () => {
  const result = parseTaskQuery('date:none #work urgent');
  expect(result.dateView).toEqual('no_date');
  expect(result.searchTerms).toEqual(['#work', 'urgent']);
});

const DAY_MS = 24 * 60 * 60 * 1000;
const today = new Date();
const tomorrow = new Date(today.getTime() + DAY_MS);
const yesterday = new Date(today.getTime() - DAY_MS);
const nextWeek = new Date(today.getTime() + 8 * DAY_MS);

function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 1,
    card_pk: 101,
    user_id: 1,
    scheduled_date: null,
    due_date: null,
    created_at: today,
    updated_at: today,
    completed_at: null,
    title: 'Task',
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
    parent_task_id: null,
    sort_order: null,
    ...overrides,
  };
}

test('filterTasksByDateView "all" hides completed unless showCompleted', () => {
  expect(filterTasksByDateView(makeTask(), 'all', false)).toBe(true);
  expect(
    filterTasksByDateView(makeTask({ is_complete: true }), 'all', false),
  ).toBe(false);
  expect(
    filterTasksByDateView(makeTask({ is_complete: true }), 'all', true),
  ).toBe(true);
});

test('filterTasksByDateView "today" matches today-or-past incomplete tasks', () => {
  expect(
    filterTasksByDateView(makeTask({ scheduled_date: today }), 'today', false),
  ).toBe(true);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: yesterday }),
      'today',
      false,
    ),
  ).toBe(true);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: tomorrow }),
      'today',
      false,
    ),
  ).toBe(false);
});

test('filterTasksByDateView "today" shows completed tasks completed today', () => {
  const completedToday = makeTask({ is_complete: true, completed_at: today });
  expect(filterTasksByDateView(completedToday, 'today', true)).toBe(true);
  expect(filterTasksByDateView(completedToday, 'today', false)).toBe(false);
});

test('filterTasksByDateView "tomorrow" matches tasks scheduled tomorrow', () => {
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: tomorrow }),
      'tomorrow',
      false,
    ),
  ).toBe(true);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: today }),
      'tomorrow',
      false,
    ),
  ).toBe(false);
  expect(filterTasksByDateView(makeTask(), 'tomorrow', false)).toBe(false);
});

test('filterTasksByDateView "overdue" matches past incomplete tasks only', () => {
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: yesterday }),
      'overdue',
      false,
    ),
  ).toBe(true);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: tomorrow }),
      'overdue',
      false,
    ),
  ).toBe(false);
  // Completed overdue tasks are never shown
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: yesterday, is_complete: true }),
      'overdue',
      true,
    ),
  ).toBe(false);
});

test('filterTasksByDateView "this_week" matches the next 7 days', () => {
  const inThreeDays = new Date(today.getTime() + 3 * DAY_MS);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: today }),
      'this_week',
      false,
    ),
  ).toBe(true);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: inThreeDays }),
      'this_week',
      false,
    ),
  ).toBe(true);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: nextWeek }),
      'this_week',
      false,
    ),
  ).toBe(false);
  expect(filterTasksByDateView(makeTask(), 'this_week', false)).toBe(false);
});

test('filterTasksByDateView "no_date" matches tasks without a scheduled date', () => {
  expect(filterTasksByDateView(makeTask(), 'no_date', false)).toBe(true);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: today }),
      'no_date',
      false,
    ),
  ).toBe(false);
  expect(
    filterTasksByDateView(
      makeTask({ scheduled_date: null, is_complete: true }),
      'no_date',
      true,
    ),
  ).toBe(true);
});

test('filterTasksByDateView falls back to hiding completed for unknown views', () => {
  expect(filterTasksByDateView(makeTask(), 'mystery_view', false)).toBe(true);
  expect(
    filterTasksByDateView(
      makeTask({ is_complete: true }),
      'mystery_view',
      false,
    ),
  ).toBe(false);
});

function makeAuditEvent(
  overrides: Partial<TaskAuditEvent> = {},
): TaskAuditEvent {
  return {
    id: 1,
    user_id: 1,
    entity_id: 1,
    entity_type: 'task',
    action: 'update',
    details: { change_type: 'update', changes: {} },
    created_at: new Date(),
    ...overrides,
  } as TaskAuditEvent;
}

test('formatAuditEvent handles create and delete actions', () => {
  expect(formatAuditEvent(makeAuditEvent({ action: 'create' }))).toBe(
    'Task created',
  );
  expect(formatAuditEvent(makeAuditEvent({ action: 'delete' }))).toBe(
    'Task deleted',
  );
});

test('formatAuditEvent reports title changes', () => {
  const event = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { Title: { from: 'Old', to: 'New' } },
    },
  });
  expect(formatAuditEvent(event)).toBe('Changed title from "Old" to "New"');
});

test('formatAuditEvent reports completion changes', () => {
  const done = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { IsComplete: { from: false, to: true } },
    },
  });
  expect(formatAuditEvent(done)).toBe('Marked as complete');

  const reopened = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { IsComplete: { from: true, to: false } },
    },
  });
  expect(formatAuditEvent(reopened)).toBe('Marked as incomplete');
});

test('formatAuditEvent reports scheduled date changes', () => {
  const event = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { ScheduledDate: { from: null, to: '2024-03-15T12:00:00Z' } },
    },
  });
  expect(formatAuditEvent(event)).toBe(
    'Changed scheduled date to Mar 15, 2024',
  );

  const cleared = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { ScheduledDate: { from: '2024-03-15T12:00:00Z', to: null } },
    },
  });
  expect(formatAuditEvent(cleared)).toBe('Changed scheduled date to none');
});

test('formatAuditEvent reports card link changes', () => {
  const linked = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { CardPK: { from: 0, to: 42 } },
    },
  });
  expect(formatAuditEvent(linked)).toBe('Linked to card [42]');

  const unlinked = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { CardPK: { from: 42, to: 0 } },
    },
  });
  expect(formatAuditEvent(unlinked)).toBe('Unlinked from card [42]');

  const moved = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { CardPK: { from: 7, to: 42 } },
    },
  });
  expect(formatAuditEvent(moved)).toBe('Changed linked card from [7] to [42]');
});

test('formatAuditEvent reports priority changes', () => {
  const raised = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { Priority: { from: null, to: 'high' } },
    },
  });
  expect(formatAuditEvent(raised)).toBe(
    'Changed priority from No Priority to Priority high',
  );

  const lowered = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { Priority: { from: 'high', to: null } },
    },
  });
  expect(formatAuditEvent(lowered)).toBe(
    'Changed priority from Priority high to No Priority',
  );
});

test('formatAuditEvent reports description changes', () => {
  const added = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { Description: { from: null, to: 'New desc' } },
    },
  });
  expect(formatAuditEvent(added)).toBe('Added description');

  const removed = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { Description: { from: 'Old desc', to: null } },
    },
  });
  expect(formatAuditEvent(removed)).toBe('Removed description');

  const edited = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { Description: { from: 'Old', to: 'New' } },
    },
  });
  expect(formatAuditEvent(edited)).toBe('Updated description');
});

test('formatAuditEvent reports reminder changes', () => {
  const set = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { ReminderTime: { from: null, to: '2024-03-15T12:00:00Z' } },
    },
  });
  expect(formatAuditEvent(set)).toContain('Set reminder for');

  const removed = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: { ReminderTime: { from: '2024-03-15T12:00:00Z', to: null } },
    },
  });
  expect(formatAuditEvent(removed)).toBe('Removed reminder');

  const changed = makeAuditEvent({
    details: {
      change_type: 'update',
      changes: {
        ReminderTime: {
          from: '2024-03-15T12:00:00Z',
          to: '2024-03-16T12:00:00Z',
        },
      },
    },
  });
  expect(formatAuditEvent(changed)).toContain('Changed reminder to');
});

test('formatAuditEvent falls back for non-update change types and unknown actions', () => {
  const noChanges = makeAuditEvent();
  expect(formatAuditEvent(noChanges)).toBe('Task updated');

  const otherChangeType = makeAuditEvent({
    details: { change_type: 'snapshot', changes: {} },
  });
  expect(formatAuditEvent(otherChangeType)).toBe('Unknown change');

  const unknownAction = makeAuditEvent({ action: 'export' });
  expect(formatAuditEvent(unknownAction)).toBe('Unknown change');
});
