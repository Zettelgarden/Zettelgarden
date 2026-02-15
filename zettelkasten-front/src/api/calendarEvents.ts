import { apiClient, getData } from './client';

export interface CreateEventRequest {
  title: string;
  description?: string;
  start_time: string;  // ISO 8601
  end_time: string;    // ISO 8601
  all_day: boolean;
  location?: string;
}

export interface CreateEventResponse {
  uid: string;
  message: string;
}

/**
 * Create a new event on an external calendar via CalDAV
 * @param calendarId The ID of the external calendar to create the event on
 * @param event The event details
 * @returns Response with the event UID and message
 */
export async function createEventOnCalendar(
  calendarId: number,
  event: CreateEventRequest
): Promise<CreateEventResponse> {
  return getData(apiClient.post<CreateEventResponse>(
    `/user/external-calendars/${calendarId}/events`,
    event
  ));
}
