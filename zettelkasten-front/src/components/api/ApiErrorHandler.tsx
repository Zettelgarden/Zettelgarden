/**
 * ApiErrorHandler Component
 *
 * This component provides a React hook for handling API errors at the edge of the app.
 * It handles auth errors (401, 422) by triggering logout and showing appropriate toasts.
 *
 * Usage:
 *   const { handleApiError } = useApiErrorHandler();
 *
 *   try {
 *     await apiClient.get('/some-endpoint');
 *   } catch (error) {
 *     handleApiError(error);
 *   }
 */

import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../contexts/AuthContext';
import { useToast } from '../toast/ToastContext';
import { APIError, isAuthError, isNetworkError } from '../../api/errors';

interface ErrorHandlingOptions {
  showToast?: boolean;
  redirectOnError?: boolean;
  customMessage?: string;
}

/**
 * Hook for handling API errors
 *
 * This hook provides a centralized way to handle API errors with:
 * - Automatic auth error handling (logout + redirect)
 * - Toast notifications for user-facing errors
 * - Optional custom error messages
 */
export function useApiErrorHandler() {
  const { logoutUser } = useAuth();
  const { showToast } = useToast();
  const navigate = useNavigate();

  const handleApiError = useCallback(
    (error: unknown, options: ErrorHandlingOptions = {}) => {
      const {
        showToast: shouldShowToast = true,
        redirectOnError = true,
        customMessage,
      } = options;

      // Handle auth errors (401, 422)
      if (isAuthError(error)) {
        // Clear the invalid token (safe to call even if key doesn't exist)
        localStorage.removeItem('token');

        // Logout from auth context
        logoutUser();

        // Show toast notification
        if (shouldShowToast) {
          showToast(
            'error',
            'Authentication Failed',
            error.message || 'Please log in to continue'
          );
        }

        // Redirect to login page
        if (redirectOnError) {
          navigate('/login', { replace: true });
        }

        return true; // Error was handled
      }

      // Handle network errors
      if (isNetworkError(error)) {
        if (shouldShowToast) {
          showToast(
            'error',
            'Network Error',
            customMessage || error.message || 'Please check your connection and try again'
          );
        }
        return true;
      }

      // Handle other API errors
      if (error instanceof APIError) {
        if (shouldShowToast) {
          const title = getErrorTitle(error.status);
          showToast(
            'error',
            title,
            customMessage || error.message || 'An error occurred'
          );
        }
        return true;
      }

      // Handle unknown errors
      if (error instanceof Error) {
        if (shouldShowToast) {
          showToast(
            'error',
            'Error',
            customMessage || error.message || 'An unexpected error occurred'
          );
        }
        return true;
      }

      // Log unknown error types
      console.error('Unknown error type:', error);
      return false;
    },
    [logoutUser, showToast, navigate]
  );

  return { handleApiError };
}

/**
 * Get user-friendly error title from status code
 */
function getErrorTitle(status?: number): string {
  if (!status) return 'Error';

  switch (status) {
    case 400:
      return 'Invalid Request';
    case 401:
      return 'Authentication Required';
    case 403:
      return 'Access Denied';
    case 404:
      return 'Not Found';
    case 409:
      return 'Conflict';
    case 422:
      return 'Validation Error';
    case 429:
      return 'Too Many Requests';
    case 500:
      return 'Server Error';
    case 502:
      return 'Bad Gateway';
    case 503:
      return 'Service Unavailable';
    default:
      if (status >= 400 && status < 500) {
        return 'Client Error';
      }
      if (status >= 500) {
        return 'Server Error';
      }
      return 'Error';
  }
}

/**
 * HOC wrapper for components that need automatic API error handling
 *
 * This wraps a component and provides error handling for async operations.
 * Use this when you want to automatically handle API errors in a component.
 */
export function withApiErrorHandler<P extends { onError?: (error: unknown) => void }>(
  Component: React.ComponentType<P>
): React.ComponentType<Omit<P, 'onError'>> {
  return function WrappedComponent(props: Omit<P, 'onError'>) {
    const { handleApiError } = useApiErrorHandler();

    const onError = useCallback(
      (error: unknown) => {
        handleApiError(error);
        const p = props as P;
        if (p.onError) {
          p.onError(error);
        }
      },
      [handleApiError, props]
    );

    return <Component {...(props as P)} onError={onError} />;
  };
}
