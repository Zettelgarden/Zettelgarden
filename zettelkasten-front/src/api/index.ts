/**
 * API Module - Public API exports
 *
 * This file provides a convenient way to import API-related functionality.
 * For new code, prefer importing from specific modules:
 * - import { apiClient } from './client'
 * - import { AuthError } from './errors'
 * - import { useApiErrorHandler } from '../components/api/ApiErrorHandler'
 */

// Core API client
export {
  apiClient,
  getData,
  type RequestConfig,
  type APIResponse,
} from './client';

// Error types
export {
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

// Re-export commonly used API functions
export { getCurrentUser, updateUser, checkAdmin } from './users';
export {
  getCard,
  saveNewCard,
  saveExistingCard,
  deleteCard,
  semanticSearchCards,
  semanticSearchCardsPaginated,
} from './cards';
export {
  fetchTasks,
  fetchTask,
  saveNewTask,
  saveExistingTask,
  deleteTask,
} from './tasks';
