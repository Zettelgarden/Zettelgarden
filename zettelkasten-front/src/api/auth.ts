import { ResetPasswordResponse } from '../models/Auth';
import { GenericResponse } from '../models/common';
import { LoginResponse } from '../models/Auth';
import { apiClient, getData } from './client';

/**
 * Login with email and password
 */
export async function login(
  email: string,
  password: string,
): Promise<LoginResponse> {
  return getData(
    apiClient.post<LoginResponse>(
      '/login',
      { email, password },
      { skipAuth: true },
    ),
  );
}

/**
 * Request password reset email
 */
export async function requestPasswordReset(
  email: string,
): Promise<GenericResponse> {
  return getData(
    apiClient.post<GenericResponse>(
      '/request-reset',
      { email },
      { skipAuth: true },
    ),
  );
}

/**
 * Reset password with token
 */
export async function resetPassword(
  token: string,
  new_password: string,
): Promise<ResetPasswordResponse> {
  return getData(
    apiClient.post<ResetPasswordResponse>(
      '/reset-password',
      { token, new_password },
      { skipAuth: true },
    ),
  );
}
