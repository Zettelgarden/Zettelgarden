import {
  ExternalEvent,
  ExternalCalendar,
  CreateExternalCalendarRequest,
  UpdateExternalCalendarRequest,
} from "src/models/ExternalEvent";
import { apiClient, getData } from "./client";

/**
 * Fetch all external calendar subscriptions for the current user
 */
export async function getExternalCalendars(): Promise<ExternalCalendar[]> {
  return getData(apiClient.get<ExternalCalendar[]>("/user/external-calendars"));
}

/**
 * Create a new external calendar subscription
 */
export async function createExternalCalendar(
  data: CreateExternalCalendarRequest
): Promise<ExternalCalendar> {
  return getData(apiClient.post<ExternalCalendar>("/user/external-calendars", data));
}

/**
 * Update an existing external calendar subscription
 */
export async function updateExternalCalendar(
  id: number,
  data: UpdateExternalCalendarRequest
): Promise<ExternalCalendar> {
  return getData(apiClient.put<ExternalCalendar>(`/user/external-calendars/${id}`, data));
}

/**
 * Delete an external calendar subscription
 */
export async function deleteExternalCalendar(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/user/external-calendars/${id}`));
}

/**
 * Trigger a manual sync of an external calendar
 */
export async function syncExternalCalendar(id: number): Promise<ExternalCalendar> {
  return getData(apiClient.post<ExternalCalendar>(`/user/external-calendars/${id}/sync`, {}));
}

/**
 * Fetch external events within a date range
 */
export async function getExternalEvents(start: Date, end: Date): Promise<ExternalEvent[]> {
  return getData(
    apiClient.get<ExternalEvent[]>("/user/external-events", {
      params: {
        start: start.toISOString(),
        end: end.toISOString(),
      },
    })
  );
}
