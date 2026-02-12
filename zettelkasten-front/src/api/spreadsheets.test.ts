/**
 * Tests for the spreadsheets API client
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import {
  fetchSpreadsheets,
  fetchSpreadsheet,
  createSpreadsheet,
  updateSpreadsheet,
  deleteSpreadsheet,
} from './spreadsheets';
import { apiClient } from './client';
import { NotFoundError } from './errors';

// Store the original fetch to restore after tests
let originalFetch: typeof globalThis.fetch;

// Mock data
const mockSpreadsheetData = {
  rows: 5,
  cols: 5,
  data: {
    A1: { value: '123', formula: '' },
    B1: { value: '456', formula: '' },
    A2: { value: '579', formula: 'A1+B1', computed: 579 },
  },
};

const mockSpreadsheet = {
  id: 1,
  user_id: 1,
  card_id: 10,
  name: 'sheet1',
  rows: 5,
  cols: 5,
  data: mockSpreadsheetData,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const mockSpreadsheet2 = {
  id: 2,
  user_id: 1,
  card_id: 10,
  name: 'sheet2',
  rows: 3,
  cols: 3,
  data: {
    rows: 3,
    cols: 3,
    data: {
      A1: { value: 'test', formula: '' },
    },
  },
  created_at: '2024-01-02T00:00:00Z',
  updated_at: '2024-01-02T00:00:00Z',
};

describe('Spreadsheets API Client', () => {
  beforeEach(() => {
    // Store the current fetch (which might be mocked by setup.ts)
    originalFetch = globalThis.fetch;

    // Mock localStorage
    vi.stubGlobal('localStorage', {
      getItem: vi.fn(() => 'test-token'),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    });
  });

  afterEach(() => {
    // Restore original fetch
    globalThis.fetch = originalFetch;

    // Unmock localStorage
    vi.unstubAllGlobals();

    vi.clearAllMocks();
  });

  describe('fetchSpreadsheets', () => {
    it('should fetch all spreadsheets for a card', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => [mockSpreadsheet, mockSpreadsheet2],
        text: async () => JSON.stringify([mockSpreadsheet, mockSpreadsheet2]),
      });
      globalThis.fetch = mockFetch;

      const result = await fetchSpreadsheets(10);

      expect(result).toHaveLength(2);
      expect(result[0].id).toBe(1);
      expect(result[0].name).toBe('sheet1');
      expect(result[1].id).toBe(2);
      expect(result[1].name).toBe('sheet2');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/cards/10/spreadsheets'),
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            Authorization: 'Bearer test-token',
          }),
        }),
      );
    });

    it('should return empty array when no spreadsheets exist', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => [],
        text: async () => '[]',
      });
      globalThis.fetch = mockFetch;

      const result = await fetchSpreadsheets(10);

      expect(result).toEqual([]);
    });

    it('should handle 404 errors when card is not found', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        text: async () => 'Card not found',
      });
      globalThis.fetch = mockFetch;

      await expect(fetchSpreadsheets(999)).rejects.toThrow(NotFoundError);
    });
  });

  describe('fetchSpreadsheet', () => {
    it('should fetch a single spreadsheet by ID', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockSpreadsheet,
        text: async () => JSON.stringify(mockSpreadsheet),
      });
      globalThis.fetch = mockFetch;

      const result = await fetchSpreadsheet(1);

      expect(result.id).toBe(1);
      expect(result.name).toBe('sheet1');
      expect(result.data.rows).toBe(5);
      expect(result.data.cols).toBe(5);
      expect(result.data.data.A1).toEqual({ value: '123', formula: '' });
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/spreadsheets/1'),
        expect.objectContaining({
          method: 'GET',
        }),
      );
    });

    it('should handle 404 when spreadsheet does not exist', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        text: async () => 'Spreadsheet not found',
      });
      globalThis.fetch = mockFetch;

      await expect(fetchSpreadsheet(999)).rejects.toThrow(NotFoundError);
    });
  });

  describe('createSpreadsheet', () => {
    it('should create a new spreadsheet with default name', async () => {
      const newSpreadsheet = {
        ...mockSpreadsheet,
        id: 3,
        name: 'sheet1',
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => newSpreadsheet,
        text: async () => JSON.stringify(newSpreadsheet),
      });
      globalThis.fetch = mockFetch;

      const result = await createSpreadsheet(10, 'sheet1');

      expect(result.id).toBe(3);
      expect(result.name).toBe('sheet1');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/cards/10/spreadsheets'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ name: 'sheet1' }),
        }),
      );
    });

    it('should create a spreadsheet with custom name', async () => {
      const newSpreadsheet = {
        ...mockSpreadsheet,
        id: 4,
        name: 'budget',
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => newSpreadsheet,
        text: async () => JSON.stringify(newSpreadsheet),
      });
      globalThis.fetch = mockFetch;

      const result = await createSpreadsheet(10, 'budget');

      expect(result.name).toBe('budget');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/cards/10/spreadsheets'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ name: 'budget' }),
        }),
      );
    });
  });

  describe('updateSpreadsheet', () => {
    it('should update spreadsheet data', async () => {
      const updatedData = {
        rows: 10,
        cols: 10,
        data: {
          A1: { value: 'new value', formula: '' },
          C5: { value: '100', formula: '' },
        },
      };

      const updatedSpreadsheet = {
        ...mockSpreadsheet,
        data: updatedData,
        updated_at: '2024-01-03T00:00:00Z',
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => updatedSpreadsheet,
        text: async () => JSON.stringify(updatedSpreadsheet),
      });
      globalThis.fetch = mockFetch;

      const result = await updateSpreadsheet(1, updatedData);

      expect(result.data.rows).toBe(10);
      expect(result.data.cols).toBe(10);
      expect(result.data.data.A1.value).toBe('new value');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/spreadsheets/1'),
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(updatedData),
        }),
      );
    });

    it('should handle partial updates', async () => {
      const partialData = {
        rows: 5,
        cols: 5,
        data: {
          A1: { value: 'updated', formula: '' },
        },
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockSpreadsheet,
        text: async () => JSON.stringify(mockSpreadsheet),
      });
      globalThis.fetch = mockFetch;

      await updateSpreadsheet(1, partialData);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/spreadsheets/1'),
        expect.objectContaining({
          method: 'PUT',
        }),
      );
    });

    it('should handle 404 when spreadsheet does not exist', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        text: async () => 'Spreadsheet not found',
      });
      globalThis.fetch = mockFetch;

      await expect(
        updateSpreadsheet(999, mockSpreadsheetData),
      ).rejects.toThrow(NotFoundError);
    });
  });

  describe('deleteSpreadsheet', () => {
    it('should delete a spreadsheet', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
        json: async () => null,
        text: async () => '',
      });
      globalThis.fetch = mockFetch;

      await deleteSpreadsheet(1);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/spreadsheets/1'),
        expect.objectContaining({
          method: 'DELETE',
        }),
      );
    });

    it('should handle 404 when spreadsheet does not exist', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        text: async () => 'Spreadsheet not found',
      });
      globalThis.fetch = mockFetch;

      await expect(deleteSpreadsheet(999)).rejects.toThrow(NotFoundError);
    });
  });

  describe('date parsing', () => {
    it('should parse date strings into Date objects', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockSpreadsheet,
        text: async () => JSON.stringify(mockSpreadsheet),
      });
      globalThis.fetch = mockFetch;

      const result = await fetchSpreadsheet(1);

      expect(result.created_at).toBeInstanceOf(Date);
      expect(result.updated_at).toBeInstanceOf(Date);
      expect(result.created_at.toISOString()).toBe('2024-01-01T00:00:00.000Z');
    });
  });
});
