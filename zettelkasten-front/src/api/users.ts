import {
  User,
  CreateUserParams,
  CreateUserResponse,
  EditUserParams,
  UserSubscription,
} from "../models/User";
import { GenericResponse } from "../models/common";
import { apiClient, getData } from "./client";
import { APIError } from "./errors";

export interface MailingListSubscriber {
  id: number;
  email: string;
  welcome_email_sent: boolean;
  subscribed: boolean;
  has_account: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * Create a new user
 */
export async function createUser(
  userData: CreateUserParams
): Promise<CreateUserResponse> {
  return getData(
    apiClient.post<CreateUserResponse>("/users", userData, { skipAuth: true })
  );
}

/**
 * Get user memory
 */
export async function getUserMemory(): Promise<{ memory: string }> {
  return getData(apiClient.get<{ memory: string }>("/user/memory"));
}

/**
 * Get a specific user by ID
 */
export async function getUser(id: string): Promise<User> {
  const encoded = encodeURIComponent(id);
  return getData(apiClient.get<User>(`/users/${encoded}`));
}

/**
 * Query parameters for getUsers
 */
export interface GetUsersParams {
  page?: number;
  per_page?: number;
}

/**
 * Response from getUsers endpoint
 */
export interface GetUsersResponse {
  users: User[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}

/**
 * Get list of users with pagination
 */
export async function getUsers(params?: GetUsersParams): Promise<GetUsersResponse> {
  const requestParams: Record<string, string | number | boolean | undefined> = {};
  if (params?.page) requestParams.page = params.page;
  if (params?.per_page) requestParams.per_page = params.per_page;
  return getData(apiClient.get<GetUsersResponse>("/users", { params: requestParams }));
}

/**
 * Update current user
 */
export async function updateUser(user: User): Promise<User> {
  return editUser(user.id.toString(), {
    username: user.username,
    email: user.email,
    is_admin: user.is_admin,
    dashboard_card_pk: user.dashboard_card_pk,
    has_seen_getting_started: user.has_seen_getting_started,
    timezone: user.timezone,
    caldav_url: user.caldav_url,
  });
}

/**
 * Edit a specific user
 */
export async function editUser(
  userId: string,
  updateData: EditUserParams
): Promise<User> {
  return getData(apiClient.put<User>(`/users/${userId}`, updateData));
}

/**
 * Get current authenticated user
 */
export async function getCurrentUser(): Promise<User> {
  return getData(apiClient.get<User>("/current"));
}

/**
 * Check if current user is an admin
 */
export async function checkAdmin(): Promise<boolean> {
  try {
    const response = await apiClient.fetchResponse("/admin", {
      method: "GET",
    });
    return response.status === 204;
  } catch (error) {
    if (error instanceof APIError && error.status === 401) {
      return false;
    }
    throw error;
  }
}

/**
 * Validate email with token
 */
export async function validateEmail(token: string): Promise<GenericResponse> {
  return getData(
    apiClient.post<GenericResponse>(
      "/email-validate",
      { token },
      { skipAuth: true }
    )
  );
}

/**
 * Resend email validation
 */
export async function resendValidateEmail(): Promise<GenericResponse> {
  return getData(apiClient.get<GenericResponse>("/email-validate"));
}

/**
 * Get user subscription details
 */
export async function getUserSubscription(id: number): Promise<UserSubscription> {
  const encodedId = encodeURIComponent(id);
  return getData(apiClient.get<UserSubscription>(`/users/${encodedId}/subscription`));
}

/**
 * Add email to mailing list
 */
export async function addToMailingList(email: string): Promise<{ email: string }> {
  return getData(
    apiClient.post<{ email: string }>("/mailing-list", { email }, { skipAuth: true })
  );
}

/**
 * Get all mailing list subscribers (admin only)
 */
export async function getMailingListSubscribers(): Promise<MailingListSubscriber[]> {
  return getData(apiClient.get<MailingListSubscriber[]>("/mailing-list"));
}

/**
 * Unsubscribe email from mailing list
 */
export async function unsubscribeMailingList(email: string): Promise<{ message: string }> {
  return getData(
    apiClient.post<{ message: string }>("/mailing-list/unsubscribe", { email })
  );
}

/**
 * Update user memory
 */
export async function updateUserMemory(memory: string): Promise<{ message: string }> {
  return getData(
    apiClient.put<{ message: string }>("/user/memory", { memory })
  );
}
