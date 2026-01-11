import { describe, it, expect } from 'vitest';
import {
  processTaskFromAPI,
  convertTaskDates,
  normalizeTaskProperties,
} from './taskDataProcessing';

describe('taskDataProcessing', () => {
  describe('processTaskFromAPI', () => {
    it('converts raw API task to internal Task format', () => {
      const rawTask: any = {
        id: 123,
        title: 'Test Task',
        description: 'Test description',
        scheduled_date: '2024-01-15',
        due_date: null,
        created_at: '2024-01-01T10:00:00Z',
        updated_at: '2024-01-02T11:00:00Z',
        completed_at: null,
        reminder_time: '2024-01-14T09:00:00Z',
        status: 'todo',
        priority: 'high',
        is_complete: false,
        is_deleted: false,
        reminder_sent: false,
        tags: [
          { id: 1, name: 'urgent' },
          { id: 2, name: 'project' }
        ],
        card: null,
        blocked_by: [],
        blocks: []
      };

      const processedTask = processTaskFromAPI(rawTask);

      // Check date conversions
      expect(processedTask.scheduled_date).toEqual(new Date('2024-01-15'));
      expect(processedTask.due_date).toBeNull();
      expect(processedTask.created_at).toEqual(new Date('2024-01-01T10:00:00Z'));
      expect(processedTask.updated_at).toEqual(new Date('2024-01-02T11:00:00Z'));
      expect(processedTask.completed_at).toBeNull();
      expect(processedTask.reminder_time).toEqual(new Date('2024-01-14T09:00:00Z'));

      // Check property normalization
      expect(processedTask.description).toBe('Test description');
      expect(processedTask.tags).toEqual([
        { id: 1, name: 'urgent' },
        { id: 2, name: 'project' }
      ]);
    });

    it('handles null/undefined values correctly', () => {
      const rawTask: any = {
        id: 123,
        title: 'Test Task',
        description: null,
        scheduled_date: null,
        due_date: undefined,
        created_at: '2024-01-01T10:00:00Z',
        updated_at: '2024-01-02T11:00:00Z',
        completed_at: undefined,
        reminder_time: null,
        status: 'todo',
        priority: null,
        is_complete: false,
        is_deleted: false,
        reminder_sent: false,
        tags: undefined,
        card: null,
        blocked_by: [],
        blocks: []
      };

      const processedTask = processTaskFromAPI(rawTask);

      expect(processedTask.scheduled_date).toBeNull();
      expect(processedTask.due_date).toBeNull();
      expect(processedTask.completed_at).toBeNull();
      expect(processedTask.reminder_time).toBeNull();
      expect(processedTask.description).toBeNull();
      expect(processedTask.tags).toEqual([]);
    });

    it('normalizes empty string description to null', () => {
      const rawTask: any = {
        id: 123,
        title: 'Test Task',
        description: '',
        tags: [],
        scheduled_date: null,
        due_date: null,
        created_at: '2024-01-01T10:00:00Z',
        updated_at: '2024-01-01T10:00:00Z',
        completed_at: null,
        reminder_time: null,
        status: 'todo',
        priority: null,
        is_complete: false,
        is_deleted: false,
        reminder_sent: false,
        card: null,
        blocked_by: [],
        blocks: []
      };

      const processedTask = processTaskFromAPI(rawTask);
      expect(processedTask.description).toBeNull();
    });

    it('handles invalid date strings gracefully', () => {
      const rawTask: any = {
        id: 123,
        title: 'Test Task',
        description: null,
        scheduled_date: 'invalid-date',
        tags: [],
        created_at: '2024-01-01T10:00:00Z',
        updated_at: '2024-01-01T10:00:00Z',
        completed_at: null,
        reminder_time: null,
        status: 'todo',
        priority: null,
        is_complete: false,
        is_deleted: false,
        reminder_sent: false,
        card: null,
        blocked_by: [],
        blocks: []
      };

      // This should not throw - new Date('invalid-date') creates an Invalid Date object
      expect(() => processTaskFromAPI(rawTask)).not.toThrow();
      const processedTask = processTaskFromAPI(rawTask);
      expect(processedTask.scheduled_date?.toString()).toBe('Invalid Date');
    });
  });

  describe('convertTaskDates', () => {
    it('converts date fields in partial task', () => {
      const partialTask = {
        scheduled_date: '2024-01-15',
        due_date: null,
        reminder_time: '2024-01-14T09:00:00Z',
        title: 'Test'
      };

      const converted = convertTaskDates(partialTask);

      expect(converted.scheduled_date).toEqual(new Date('2024-01-15'));
      expect(converted.due_date).toBeNull();
      expect(converted.reminder_time).toEqual(new Date('2024-01-14T09:00:00Z'));
      expect(converted.title).toBe('Test');
    });

    it('handles cases where date fields already exist but are undefined', () => {
      const partialTask = {
        scheduled_date: undefined,
        due_date: null,
        reminder_time: undefined,
        title: 'Test'
      };

      const converted = convertTaskDates(partialTask);

      expect(converted.scheduled_date).toBeNull();
      expect(converted.due_date).toBeNull();
      expect(converted.reminder_time).toBeNull();
      expect(converted.title).toBe('Test');
    });
  });

  describe('normalizeTaskProperties', () => {
    it('normalizes description and tags in partial task', () => {
      const partialTask = {
        description: '',
        tags: undefined,
        title: 'Test'
      };

      const normalized = normalizeTaskProperties(partialTask);

      expect(normalized.description).toBeNull();
      expect(normalized.tags).toEqual([]);
      expect(normalized.title).toBe('Test');
    });

    it('handles various tag array types', () => {
      const testCases = [
        { input: [{ id: 1, name: 'tag1' }], expected: [{ id: 1, name: 'tag1' }] },
        { input: [], expected: [] },
        { input: null, expected: [] },
        { input: undefined, expected: [] }
      ];

      testCases.forEach(({ input, expected }) => {
        const partialTask = { tags: input, title: 'Test' };
        const normalized = normalizeTaskProperties(partialTask);
        expect(normalized.tags).toEqual(expected);
      });
    });

    it('handles various description types', () => {
      const testCases = [
        { input: 'Valid description', expected: 'Valid description' },
        { input: '', expected: null },
        { input: '   ', expected: '   ' }, // Whitespace-only strings are preserved
        { input: null, expected: null },
        { input: undefined, expected: null }
      ];

      testCases.forEach(({ input, expected }) => {
        const partialTask = { description: input, title: 'Test' };
        const normalized = normalizeTaskProperties(partialTask);
        expect(normalized.description).toBe(expected);
      });
    });
  });

  describe('integration scenarios', () => {
    it('handles complete task data from API', () => {
      const fullApiResponse: any = {
        id: 456,
        title: 'Complete Test Task',
        description: 'A complete description',
        scheduled_date: '2024-03-15',
        due_date: '2024-03-20',
        created_at: '2024-01-01T12:00:00Z',
        updated_at: '2024-01-02T14:30:00Z',
        completed_at: '2024-01-03T16:45:00Z',
        reminder_time: '2024-03-14T08:00:00Z',
        status: 'completed',
        priority: 'medium',
        is_complete: true,
        is_deleted: false,
        reminder_sent: true,
        tags: [
          { id: 10, name: 'feature' },
          { id: 11, name: 'ui' }
        ],
        card: { id: 789, title: 'Related Card' },
        blocked_by: [{ id: 111, title: 'Blocked by this', is_complete: false }],
        blocks: [{ id: 222, title: 'Blocks that', is_complete: true }]
      };

      const processed = processTaskFromAPI(fullApiResponse);

      // Verify all date conversions worked
      expect(processed.scheduled_date).toEqual(new Date('2024-03-15'));
      expect(processed.due_date).toEqual(new Date('2024-03-20'));
      expect(processed.created_at).toEqual(new Date('2024-01-01T12:00:00Z'));
      expect(processed.updated_at).toEqual(new Date('2024-01-02T14:30:00Z'));
      expect(processed.completed_at).toEqual(new Date('2024-01-03T16:45:00Z'));
      expect(processed.reminder_time).toEqual(new Date('2024-03-14T08:00:00Z'));

      // Verify all properties
      expect(processed.id).toBe(456);
      expect(processed.title).toBe('Complete Test Task');
      expect(processed.description).toBe('A complete description');
      expect(processed.status).toBe('completed');
      expect(processed.priority).toBe('medium');
      expect(processed.is_complete).toBe(true);
      expect(processed.tags).toHaveLength(2);
      expect(processed.card?.title).toBe('Related Card');
      expect(processed.blocked_by).toHaveLength(1);
      expect(processed.blocks).toHaveLength(1);
    });

    it('handles minimal task data', () => {
      const minimalApiResponse: any = {
        id: 999,
        title: 'Minimal Task',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        status: 'todo',
        is_complete: false,
        is_deleted: false,
        reminder_sent: false
        // All other nullable fields omitted or null
      };

      const processed = processTaskFromAPI(minimalApiResponse);

      expect(processed.id).toBe(999);
      expect(processed.title).toBe('Minimal Task');
      expect(processed.scheduled_date).toBeNull();
      expect(processed.due_date).toBeNull();
      expect(processed.completed_at).toBeNull();
      expect(processed.reminder_time).toBeNull();
      expect(processed.description).toBeNull();
      expect(processed.tags).toEqual([]);
      expect(processed.card).toBeNull();
      expect(processed.blocked_by).toEqual([]); // Default empty array
      expect(processed.blocks).toEqual([]); // Default empty array
    });
  });
});