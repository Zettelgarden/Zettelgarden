/**
 * @deprecated This file is deprecated. Please use the new API client from './client'
 *
 * The old checkStatus function had problematic side effects (direct page reloads).
 * The new apiClient from './client' properly separates concerns:
 * - API layer: Returns error objects instead of triggering side effects
 * - UI layer: Uses useApiErrorHandler hook to handle errors appropriately
 *
 * Migration guide:
 * 1. Replace: import { checkStatus } from './common'
 *    With: import { apiClient, getData } from './client'
 *
 * 2. Replace: fetch(url).then(checkStatus).then(r => r.json())
 *    With: getData(apiClient.get(url))
 *
 * 3. Add error handling in your component:
 *    const { handleApiError } = useApiErrorHandler();
 *    try {
 *      await getData(apiClient.get('/endpoint'));
 *    } catch (error) {
 *      handleApiError(error);
 *    }
 */

import { APIError, AuthError, TokenValidationError } from './errors';

/**
 * @deprecated Use apiClient from './client' instead
 *
 * This function is kept for backward compatibility but should not be used in new code.
 * It now throws proper error types instead of triggering side effects.
 */
export async function checkStatus(response: Response): Promise<Response> {
  // For 401 and 422, throw specific auth errors instead of triggering side effects
  if (response.status === 401) {
    throw new AuthError('Authentication failed');
  }

  if (response.status === 422) {
    throw new TokenValidationError('Token validation failed');
  }

  // If the response is ok, return the response to continue the promise chain
  if (response.ok) {
    return response;
  }

  // Try to extract error message from response body
  let errorText: string;
  try {
    errorText = await response.text();
  } catch (e) {
    // If we can't read the body, fall back to status code
    errorText = `Request failed with status: ${response.status}`;
  }

  // Throw the error with the extracted message
  throw new APIError(errorText || `Request failed with status: ${response.status}`, response.status);
}
