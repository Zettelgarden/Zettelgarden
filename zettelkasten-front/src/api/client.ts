/**
 * API Client - A centralized fetch wrapper with proper error handling
 *
 * This client provides a clean interface for making API requests without
 * side effects. All auth errors are returned as error objects, allowing
 * the calling code or UI layer to decide how to handle them.
 */

import {
  APIError,
  AuthError,
  TokenValidationError,
  NetworkError,
  ValidationError,
  NotFoundError,
  ServerError,
} from './errors';

const BASE_URL = import.meta.env.VITE_URL;

/**
 * Configuration options for API requests
 */
export interface RequestConfig extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
  skipAuth?: boolean;
}

/**
 * Response wrapper that includes the raw Response object
 */
export interface APIResponse<T> {
  data: T;
  response: Response;
}

/**
 * Build URL with query parameters
 *
 * Handles proper path concatenation to avoid losing parts of the base URL path.
 * For example, if base is "http://localhost:8079/api" and path is "/login",
 * this correctly produces "http://localhost:8079/api/login" instead of
 * "http://localhost:8079/login" (which is what new URL(path, base) does).
 */
export function buildURL(
  base: string,
  path: string,
  params?: Record<string, string | number | boolean | undefined>,
): string {
  // If path is already a full URL, use it as-is
  if (path.startsWith('http://') || path.startsWith('https://')) {
    const url = new URL(path);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          url.searchParams.append(key, String(value));
        }
      });
    }
    return url.toString();
  }

  // For relative paths, properly concatenate with base URL path
  const baseUrl = new URL(base);
  // Get the base path (e.g., "/api" from "http://localhost:8079/api")
  const basePath = baseUrl.pathname;

  // Clean up the path to concatenate
  const cleanPath = path.startsWith('/') ? path.slice(1) : path;

  // Build full path: "/api" + "/" + "login" = "/api/login"
  const fullPath = basePath.endsWith('/')
    ? basePath + cleanPath
    : basePath + '/' + cleanPath;

  // Create the URL with the correct path
  const url = new URL(fullPath, base);

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        url.searchParams.append(key, String(value));
      }
    });
  }
  return url.toString();
}

/**
 * Get auth token from localStorage
 */
function getAuthToken(): string | null {
  return localStorage.getItem('token');
}

/**
 * Parse error message from response
 */
async function parseErrorMessage(response: Response): Promise<string> {
  try {
    const text = await response.text();
    if (text) {
      try {
        const json = JSON.parse(text);
        return json.message || json.error || text;
      } catch {
        return text;
      }
    }
    return `Request failed with status: ${response.status}`;
  } catch {
    return `Request failed with status: ${response.status}`;
  }
}

/**
 * Convert HTTP status to appropriate error type
 */
function createErrorFromStatus(status: number, message: string): APIError {
  switch (status) {
    case 401:
      return new AuthError(message);
    case 422:
      return new TokenValidationError(message);
    case 400:
      return new ValidationError(message, 400);
    case 404:
      return new NotFoundError(message);
    default:
      if (status >= 500) {
        return new ServerError(message);
      }
      return new APIError(message, status);
  }
}

/**
 * Check response and return appropriate error or continue
 * This is the NEW version that returns errors instead of throwing
 */
export async function checkStatus(response: Response): Promise<Response> {
  // Always return the response - let the caller handle errors
  // This is a CHANGE from the old common.ts which threw errors directly
  return response;
}

/**
 * Parse response as JSON
 */
async function parseResponse<T>(response: Response): Promise<T> {
  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  if (!text) {
    return undefined as T;
  }

  try {
    return JSON.parse(text) as T;
  } catch (error) {
    throw new APIError(
      `Failed to parse response as JSON: ${text}`,
      response.status,
    );
  }
}

/**
 * Core fetch wrapper with error handling
 */
async function fetchWithErrorHandling(
  path: string,
  config: RequestConfig = {},
): Promise<Response> {
  const { params, skipAuth = false, ...requestConfig } = config;

  // Build URL with params
  const url = buildURL(BASE_URL, path, params);

  // Add auth header
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...requestConfig.headers,
  };

  if (!skipAuth) {
    const token = getAuthToken();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  }

  // Make request
  try {
    const response = await fetch(url, {
      ...requestConfig,
      headers,
    });

    return response;
  } catch (error) {
    if (error instanceof TypeError) {
      // Network error (failed to fetch, timeout, etc.)
      throw new NetworkError(
        'Network request failed. Please check your connection.',
      );
    }
    throw error;
  }
}

/**
 * Process response and handle errors
 * This is where we convert HTTP errors to typed error objects
 */
async function processResponse<T>(response: Response): Promise<APIResponse<T>> {
  // If response is not OK, create and return an error
  if (!response.ok) {
    const message = await parseErrorMessage(response);
    throw createErrorFromStatus(response.status, message);
  }

  // Parse response data
  const data = await parseResponse<T>(response);

  return {
    data,
    response,
  };
}

/**
 * API Client - Public interface
 */
export const apiClient = {
  /**
   * GET request
   */
  async get<T>(path: string, config?: RequestConfig): Promise<APIResponse<T>> {
    const response = await fetchWithErrorHandling(path, {
      ...config,
      method: 'GET',
    });
    return processResponse<T>(response);
  },

  /**
   * POST request
   */
  async post<T>(
    path: string,
    body?: unknown,
    config?: RequestConfig,
  ): Promise<APIResponse<T>> {
    const response = await fetchWithErrorHandling(path, {
      ...config,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
    return processResponse<T>(response);
  },

  /**
   * PUT request
   */
  async put<T>(
    path: string,
    body?: unknown,
    config?: RequestConfig,
  ): Promise<APIResponse<T>> {
    const response = await fetchWithErrorHandling(path, {
      ...config,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    });
    return processResponse<T>(response);
  },

  /**
   * PATCH request
   */
  async patch<T>(
    path: string,
    body?: unknown,
    config?: RequestConfig,
  ): Promise<APIResponse<T>> {
    const response = await fetchWithErrorHandling(path, {
      ...config,
      method: 'PATCH',
      body: body ? JSON.stringify(body) : undefined,
    });
    return processResponse<T>(response);
  },

  /**
   * DELETE request
   */
  async delete<T>(
    path: string,
    config?: RequestConfig,
  ): Promise<APIResponse<T>> {
    const response = await fetchWithErrorHandling(path, {
      ...config,
      method: 'DELETE',
    });
    return processResponse<T>(response);
  },

  /**
   * Raw response fetch (for when you need the Response object directly)
   */
  async fetchResponse(path: string, config?: RequestConfig): Promise<Response> {
    return fetchWithErrorHandling(path, config);
  },
};

/**
 * Helper function for backward compatibility
 * Extract just the data from APIResponse
 */
export async function getData<T>(promise: Promise<APIResponse<T>>): Promise<T> {
  const { data } = await promise;
  return data;
}
