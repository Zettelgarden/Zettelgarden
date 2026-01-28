/**
 * React Query hooks for authentication and user data
 *
 * This module provides hooks for auth operations with:
 * - Automatic token management
 * - Proper error handling for 401/422 responses
 * - Subscription status tracking
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../../api/queryClient';
import {
  getCurrentUser,
  checkAdmin,
  updateUser as apiUpdateUser,
  getUserSubscription,
} from '../../api/users';
import { User, UserSubscription } from '../../models/User';

/**
 * Hook to fetch current user data
 *
 * Automatically handles authentication state and redirects
 * on 401/422 errors via the checkStatus function in the API layer.
 *
 * @returns Query result with current user
 *
 * @example
 * ```tsx
 * function UserProfile() {
 *   const { data: user, isLoading, error } = useCurrentUser();
 *
 *   if (isLoading) return <div>Loading...</div>;
 *   if (error) return <div>Error: {error.message}</div>;
 *
 *   return <div>Welcome, {user?.username}</div>;
 * }
 * ```
 */
export function useCurrentUser() {
  return useQuery({
    queryKey: queryKeys.auth.current(),
    queryFn: getCurrentUser,
    retry: false, // Don't retry auth failures
    staleTime: 10 * 60 * 1000, // 10 minutes - user data changes infrequently
  });
}

/**
 * Hook to check if current user is an admin
 *
 * @returns Query result with admin status
 */
export function useIsAdmin() {
  return useQuery({
    queryKey: queryKeys.auth.admin(),
    queryFn: checkAdmin,
    retry: false,
    staleTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to fetch user subscription status
 *
 * @param userId - User ID
 * @returns Query result with subscription data
 */
export function useUserSubscription(userId: number) {
  return useQuery({
    queryKey: queryKeys.auth.subscription(userId),
    queryFn: () => getUserSubscription(userId),
    enabled: !!userId,
    staleTime: 5 * 60 * 1000, // 5 minutes - subscription status changes infrequently
  });
}

/**
 * Mutation hook to update user data
 *
 * @returns Mutation with status and handlers
 */
export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (user: User) => apiUpdateUser(user),

    onSuccess: (updatedUser) => {
      // Update the current user cache
      queryClient.setQueryData(queryKeys.auth.current(), updatedUser);
    },
  });
}
