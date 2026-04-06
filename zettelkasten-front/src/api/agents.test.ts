/**
 * Tests for the agents API client
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createAgent, listAgents, revokeAgent, getAgentActivity } from './agents';
import { NotFoundError, ValidationError } from './errors';

// Store the original fetch to restore after tests
let originalFetch: typeof globalThis.fetch;

// Mock data
const mockAgent = {
  id: 1,
  name: 'Test Agent',
  description: 'Test Description',
  created_at: '2024-01-01T00:00:00Z',
  last_used: null,
  is_active: true,
};

const mockAgentWithKey = {
  ...mockAgent,
  api_key: 'zg_live_test_key_123456789',
};

const mockActivityLog = {
  id: 1,
  agent_id: 1,
  action: 'create_card',
  target_type: 'card',
  target_id: 123,
  details: { title: 'Test Card' },
  created_at: '2024-01-01T12:00:00Z',
};

describe('Agents API Client', () => {
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

  describe('createAgent', () => {
    it('should create an agent successfully', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => mockAgentWithKey,
        text: async () => JSON.stringify(mockAgentWithKey),
      });
      globalThis.fetch = mockFetch;

      const result = await createAgent('Test Agent', 'Test Description');

      expect(result.name).toBe('Test Agent');
      expect(result.description).toBe('Test Description');
      expect(result.api_key).toBeDefined();
      expect(result.api_key).toMatch(/^zg_live_/);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/agents'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ name: 'Test Agent', description: 'Test Description' }),
          headers: expect.objectContaining({
            Authorization: 'Bearer test-token',
          }),
        }),
      );
    });

    it('should create an agent without description', async () => {
      const agentWithoutDescription = {
        ...mockAgentWithKey,
        description: undefined,
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => agentWithoutDescription,
        text: async () => JSON.stringify(agentWithoutDescription),
      });
      globalThis.fetch = mockFetch;

      const result = await createAgent('Test Agent');

      expect(result.name).toBe('Test Agent');
      expect(result.description).toBeUndefined();
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/agents'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ name: 'Test Agent' }),
        }),
      );
    });

    it('should handle validation error when name is missing', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        text: async () => JSON.stringify({ error: 'Name is required' }),
      });
      globalThis.fetch = mockFetch;

      await expect(createAgent('', 'Test Description')).rejects.toThrow();
    });
  });

  describe('listAgents', () => {
    it('should list all agents', async () => {
      const mockResponse = {
        agents: [mockAgent],
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockResponse,
        text: async () => JSON.stringify(mockResponse),
      });
      globalThis.fetch = mockFetch;

      const result = await listAgents();

      expect(result).toBeInstanceOf(Array);
      expect(result).toHaveLength(1);
      expect(result[0].id).toBe(1);
      expect(result[0].name).toBe('Test Agent');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/agents'),
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            Authorization: 'Bearer test-token',
          }),
        }),
      );
    });

    it('should return empty array when no agents exist', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ agents: [] }),
        text: async () => JSON.stringify({ agents: [] }),
      });
      globalThis.fetch = mockFetch;

      const result = await listAgents();

      expect(result).toEqual([]);
    });
  });

  describe('revokeAgent', () => {
    it('should revoke an agent successfully', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 204,
        json: async () => null,
        text: async () => '',
      });
      globalThis.fetch = mockFetch;

      await revokeAgent(1);

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/agents/1'),
        expect.objectContaining({
          method: 'DELETE',
          headers: expect.objectContaining({
            Authorization: 'Bearer test-token',
          }),
        }),
      );
    });

    it('should handle 404 when agent does not exist', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        text: async () => 'Agent not found',
      });
      globalThis.fetch = mockFetch;

      await expect(revokeAgent(999)).rejects.toThrow(NotFoundError);
    });
  });

  describe('getAgentActivity', () => {
    it('should get agent activity log', async () => {
      const mockResponse = {
        logs: [mockActivityLog],
        pagination: {
          page: 1,
          per_page: 50,
          total: 1,
          total_pages: 1,
        },
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockResponse,
        text: async () => JSON.stringify(mockResponse),
      });
      globalThis.fetch = mockFetch;

      const result = await getAgentActivity(1, 1, 50);

      expect(result.logs).toBeInstanceOf(Array);
      expect(result.logs).toHaveLength(1);
      expect(result.logs[0].action).toBe('create_card');
      expect(result.logs[0].target_type).toBe('card');
      expect(result.logs[0].target_id).toBe(123);
      expect(result.pagination).toBeDefined();
      expect(result.pagination.page).toBe(1);
      expect(result.pagination.total).toBe(1);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/agents/1/activity'),
        expect.objectContaining({
          method: 'GET',
        }),
      );
    });

    it('should use default pagination values', async () => {
      const mockResponse = {
        logs: [],
        pagination: {
          page: 1,
          per_page: 50,
          total: 0,
          total_pages: 0,
        },
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockResponse,
        text: async () => JSON.stringify(mockResponse),
      });
      globalThis.fetch = mockFetch;

      const result = await getAgentActivity(1);

      expect(result.pagination.page).toBe(1);
      expect(result.pagination.per_page).toBe(50);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('page=1'),
        expect.any(Object),
      );
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('per_page=50'),
        expect.any(Object),
      );
    });

    it('should handle custom pagination', async () => {
      const mockResponse = {
        logs: [],
        pagination: {
          page: 2,
          per_page: 20,
          total: 30,
          total_pages: 2,
        },
      };

      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockResponse,
        text: async () => JSON.stringify(mockResponse),
      });
      globalThis.fetch = mockFetch;

      const result = await getAgentActivity(1, 2, 20);

      expect(result.pagination.page).toBe(2);
      expect(result.pagination.per_page).toBe(20);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('page=2'),
        expect.any(Object),
      );
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('per_page=20'),
        expect.any(Object),
      );
    });

    it('should handle 404 when agent does not exist', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 404,
        text: async () => 'Agent not found',
      });
      globalThis.fetch = mockFetch;

      await expect(getAgentActivity(999)).rejects.toThrow(NotFoundError);
    });
  });

  describe('error handling', () => {
    it('should handle network errors', async () => {
      const mockFetch = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'));
      globalThis.fetch = mockFetch;

      await expect(listAgents()).rejects.toThrow('Network request failed');
    });

    it('should handle server errors', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        text: async () => 'Internal server error',
      });
      globalThis.fetch = mockFetch;

      await expect(listAgents()).rejects.toThrow();
    });
  });
});
