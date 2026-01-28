/**
 * Tests for the new API client error handling
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { apiClient, getData } from './client';
import { AuthError, TokenValidationError, NetworkError, ValidationError, NotFoundError } from './errors';

// Store the original fetch to restore after tests
let originalFetch: typeof globalThis.fetch;

describe('API Client', () => {
  let mockFetch: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    // Store the current fetch (which might be mocked by setup.ts)
    originalFetch = globalThis.fetch;

    // Create a fresh mock for each test
    mockFetch = vi.fn();

    // Replace global fetch with our mock
    globalThis.fetch = mockFetch;

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

  describe('GET requests', () => {
    it('should make successful GET request', async () => {
      const mockData = { id: 1, name: 'Test' };
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockData,
        text: async () => JSON.stringify(mockData),
      });

      const result = await apiClient.get('/test');
      expect(result.data).toEqual(mockData);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            Authorization: 'Bearer test-token',
          }),
        })
      );
    });

    it('should handle 401 auth errors', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 401,
        text: async () => 'Unauthorized',
      });

      await expect(apiClient.get('/test')).rejects.toThrow(AuthError);
      await expect(apiClient.get('/test')).rejects.toMatchObject({
        status: 401,
      });
    });

    it('should handle 422 token validation errors', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 422,
        text: async () => 'Invalid token',
      });

      await expect(apiClient.get('/test')).rejects.toThrow(TokenValidationError);
      await expect(apiClient.get('/test')).rejects.toMatchObject({
        status: 422,
      });
    });

    it('should handle 404 not found errors', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        text: async () => 'Not found',
      });

      await expect(apiClient.get('/test')).rejects.toThrow(NotFoundError);
    });

    it('should handle 400 validation errors', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 400,
        text: async () => 'Invalid input',
      });

      await expect(apiClient.get('/test')).rejects.toThrow(ValidationError);
    });

    it('should handle network errors', async () => {
      mockFetch.mockRejectedValue(new TypeError('Failed to fetch'));

      await expect(apiClient.get('/test')).rejects.toThrow(NetworkError);
    });

    it('should include query parameters', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => [],
        text: async () => '[]',
      });

      await apiClient.get('/test', {
        params: { page: 1, limit: 10, search: 'test' },
      });

      expect(mockFetch).toHaveBeenCalled();
      const callArgs = mockFetch.mock.calls[0];
      const url = new URL(callArgs[0]);
      expect(url.searchParams.get('page')).toBe('1');
      expect(url.searchParams.get('limit')).toBe('10');
      expect(url.searchParams.get('search')).toBe('test');
    });

    it('should skip auth when requested', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({}),
        text: async () => '{}',
      });

      await apiClient.get('/test', { skipAuth: true });

      expect(mockFetch).toHaveBeenCalled();
      const callArgs = mockFetch.mock.calls[0];
      expect(callArgs[1].headers).not.toHaveProperty('Authorization');
    });
  });

  describe('POST requests', () => {
    it('should make successful POST request', async () => {
      const mockData = { id: 1, name: 'Test' };
      const requestBody = { name: 'Test' };

      mockFetch.mockResolvedValue({
        ok: true,
        status: 201,
        json: async () => mockData,
        text: async () => JSON.stringify(mockData),
      });

      const result = await apiClient.post('/test', requestBody);
      expect(result.data).toEqual(mockData);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(requestBody),
        })
      );
    });
  });

  describe('PUT requests', () => {
    it('should make successful PUT request', async () => {
      const mockData = { id: 1, name: 'Updated' };
      const requestBody = { name: 'Updated' };

      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockData,
        text: async () => JSON.stringify(mockData),
      });

      const result = await apiClient.put('/test/1', requestBody);
      expect(result.data).toEqual(mockData);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(requestBody),
        })
      );
    });
  });

  describe('DELETE requests', () => {
    it('should handle successful DELETE', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 204,
        json: async () => null,
        text: async () => '',
      });

      const result = await apiClient.delete('/test/1');
      expect(result.response.status).toBe(204);
    });
  });

  describe('getData helper', () => {
    it('should extract data from APIResponse', async () => {
      const mockData = { id: 1, name: 'Test' };
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockData,
        text: async () => JSON.stringify(mockData),
      });

      const result = await getData(apiClient.get('/test'));
      expect(result).toEqual(mockData);
    });
  });

  describe('URL construction', () => {
    it('should properly concatenate base URL path with request path', async () => {
      // This test verifies the fix for the buildURL function
      // Previously, new URL('/login', 'http://localhost:8079/api') would produce
      // 'http://localhost:8079/login' (losing the /api part)
      // Now it correctly produces 'http://localhost:8079/api/login'
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ token: 'test-token' }),
        text: async () => JSON.stringify({ token: 'test-token' }),
      });

      await apiClient.post('/login', { email: 'test@test.com', password: 'password' });

      expect(mockFetch).toHaveBeenCalled();
      const callArgs = mockFetch.mock.calls[0];
      const url = callArgs[0];

      // The URL should contain /api/login not just /login
      expect(url).toContain('/api/login');
    });

    it('should handle paths without leading slash', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => [],
        text: async () => '[]',
      });

      await apiClient.get('test');

      expect(mockFetch).toHaveBeenCalled();
      const callArgs = mockFetch.mock.calls[0];
      const url = callArgs[0];

      // Should still have /api in the path
      expect(url).toContain('/api/test');
    });

    it('should handle full URLs without modification', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ data: 'test' }),
        text: async () => JSON.stringify({ data: 'test' }),
      });

      const fullUrl = 'https://example.com/external-api';
      await apiClient.get(fullUrl);

      expect(mockFetch).toHaveBeenCalled();
      const callArgs = mockFetch.mock.calls[0];
      const url = callArgs[0];

      expect(url).toBe(fullUrl);
    });
  });

  describe('Error message parsing', () => {
    it('should extract error message from JSON response', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 400,
        text: async () => JSON.stringify({ message: 'Custom error message' }),
      });

      await expect(apiClient.get('/test')).rejects.toThrow('Custom error message');
    });

    it('should extract error message from text response', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        text: async () => 'Internal server error',
      });

      await expect(apiClient.get('/test')).rejects.toThrow('Internal server error');
    });

    it('should use default message when response body is empty', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        text: async () => '',
      });

      await expect(apiClient.get('/test')).rejects.toThrow('Request failed with status: 500');
    });
  });
});
