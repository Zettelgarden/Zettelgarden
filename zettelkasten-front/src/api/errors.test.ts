/**
 * Tests for API error types
 */

import { describe, it, expect } from 'vitest';
import {
  APIError,
  AuthError,
  TokenValidationError,
  NetworkError,
  ValidationError,
  NotFoundError,
  ServerError,
  isAuthError,
  isNetworkError,
  isAPIError,
} from './errors';

describe('API Error Types', () => {
  describe('Error Classes', () => {
    it('should create APIError with correct properties', () => {
      const error = new APIError('Test error', 500, 'Response text');
      expect(error.message).toBe('Test error');
      expect(error.status).toBe(500);
      expect(error.responseText).toBe('Response text');
      expect(error.name).toBe('APIError');
    });

    it('should create AuthError with default message', () => {
      const error = new AuthError();
      expect(error.message).toBe('Authentication failed');
      expect(error.status).toBe(401);
      expect(error.name).toBe('AuthError');
    });

    it('should create AuthError with custom message', () => {
      const error = new AuthError('Custom auth error');
      expect(error.message).toBe('Custom auth error');
      expect(error.status).toBe(401);
    });

    it('should create TokenValidationError with default message', () => {
      const error = new TokenValidationError();
      expect(error.message).toBe('Token validation failed');
      expect(error.status).toBe(422);
      expect(error.name).toBe('TokenValidationError');
    });

    it('should create NetworkError with default message', () => {
      const error = new NetworkError();
      expect(error.message).toBe('Network request failed');
      expect(error.status).toBe(0);
      expect(error.name).toBe('NetworkError');
    });

    it('should create ValidationError', () => {
      const error = new ValidationError('Invalid input', 400);
      expect(error.message).toBe('Invalid input');
      expect(error.status).toBe(400);
      expect(error.name).toBe('ValidationError');
    });

    it('should create NotFoundError', () => {
      const error = new NotFoundError('Resource not found');
      expect(error.message).toBe('Resource not found');
      expect(error.status).toBe(404);
      expect(error.name).toBe('NotFoundError');
    });

    it('should create ServerError', () => {
      const error = new ServerError('Server exploded');
      expect(error.message).toBe('Server exploded');
      expect(error.status).toBe(500);
      expect(error.name).toBe('ServerError');
    });
  });

  describe('Type Guards', () => {
    it('should identify AuthError with isAuthError', () => {
      const authError = new AuthError();
      const tokenError = new TokenValidationError();
      const otherError = new Error('Regular error');

      expect(isAuthError(authError)).toBe(true);
      expect(isAuthError(tokenError)).toBe(true);
      expect(isAuthError(otherError)).toBe(false);
      expect(isAuthError(null)).toBe(false);
      expect(isAuthError(undefined)).toBe(false);
    });

    it('should identify NetworkError with isNetworkError', () => {
      const networkError = new NetworkError();
      const otherError = new Error('Regular error');

      expect(isNetworkError(networkError)).toBe(true);
      expect(isNetworkError(otherError)).toBe(false);
      expect(isNetworkError(null)).toBe(false);
    });

    it('should identify APIError with isAPIError', () => {
      const apiError = new APIError('API error', 500);
      const authError = new AuthError();
      const regularError = new Error('Regular error');

      expect(isAPIError(apiError)).toBe(true);
      expect(isAPIError(authError)).toBe(true);
      expect(isAPIError(regularError)).toBe(false);
      expect(isAPIError(null)).toBe(false);
    });
  });

  describe('Error Instance Checks', () => {
    it('should allow instanceof checks for all error types', () => {
      expect(new APIError('test') instanceof APIError).toBe(true);
      expect(new AuthError() instanceof AuthError).toBe(true);
      expect(new TokenValidationError() instanceof TokenValidationError).toBe(true);
      expect(new NetworkError() instanceof NetworkError).toBe(true);
      expect(new ValidationError('test') instanceof ValidationError).toBe(true);
      expect(new NotFoundError() instanceof NotFoundError).toBe(true);
      expect(new ServerError() instanceof ServerError).toBe(true);

      // All should be instance of APIError
      expect(new AuthError() instanceof APIError).toBe(true);
      expect(new TokenValidationError() instanceof APIError).toBe(true);
      expect(new NetworkError() instanceof APIError).toBe(true);
    });
  });
});
