/**
 * Custom error types for API error handling.
 * These errors allow the API layer to return error information without
 * triggering side effects like page reloads.
 */

/**
 * Base API error class
 */
export class APIError extends Error {
  constructor(
    message: string,
    public status?: number,
    public responseText?: string
  ) {
    super(message);
    this.name = 'APIError';
  }
}

/**
 * Authentication error (401)
 */
export class AuthError extends APIError {
  constructor(message: string = 'Authentication failed') {
    super(message, 401);
    this.name = 'AuthError';
  }
}

/**
 * Token validation error (422)
 */
export class TokenValidationError extends APIError {
  constructor(message: string = 'Token validation failed') {
    super(message, 422);
    this.name = 'TokenValidationError';
  }
}

/**
 * Network error (failed to connect, timeout, etc.)
 */
export class NetworkError extends APIError {
  constructor(message: string = 'Network request failed') {
    super(message, 0);
    this.name = 'NetworkError';
  }
}

/**
 * Validation error (400, 422)
 */
export class ValidationError extends APIError {
  constructor(message: string, status: number = 400) {
    super(message, status);
    this.name = 'ValidationError';
  }
}

/**
 * Not found error (404)
 */
export class NotFoundError extends APIError {
  constructor(message: string = 'Resource not found') {
    super(message, 404);
    this.name = 'NotFoundError';
  }
}

/**
 * Server error (5xx)
 */
export class ServerError extends APIError {
  constructor(message: string = 'Server error occurred') {
    super(message, 500);
    this.name = 'ServerError';
  }
}

/**
 * Check if an error is an auth-related error that should trigger logout
 */
export function isAuthError(error: unknown): error is AuthError | TokenValidationError {
  return error instanceof AuthError || error instanceof TokenValidationError;
}

/**
 * Check if an error is a network error (no response from server)
 */
export function isNetworkError(error: unknown): error is NetworkError {
  return error instanceof NetworkError;
}

/**
 * Type guard to check if error is an APIError
 */
export function isAPIError(error: unknown): error is APIError {
  return error instanceof APIError;
}
