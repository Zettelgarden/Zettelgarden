import { PartialCard } from "./Card";

/**
 * External calendar event imported from iCal feed
 */
export interface ExternalEvent {
  id: number;
  user_id: number;
  external_calendar_id?: number;
  title: string;
  description?: string;
  start_time: string;  // ISO 8601
  end_time: string;    // ISO 8601
  all_day: boolean;
  location?: string;
  external_uid?: string;
  external_url?: string;
  recurrence_rule?: string;
  color?: string;
  card_pk?: number;
  created_at: string;
  updated_at: string;
  last_synced_at?: string;
  card?: PartialCard;
}

/**
 * External calendar subscription
 */
export interface ExternalCalendar {
  id: number;
  user_id: number;
  name: string;
  url: string;
  sync_enabled: boolean;
  sync_interval_hours: number;
  color: string;
  last_synced_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Request to create a new external calendar subscription
 */
export interface CreateExternalCalendarRequest {
  name: string;
  url: string;
  color?: string;
}

/**
 * Request to update an external calendar subscription
 */
export interface UpdateExternalCalendarRequest {
  name?: string;
  url?: string;
  color?: string;
  sync_enabled?: boolean;
  sync_interval_hours?: number;
}

/**
 * Request to link an external event to a card
 */
export interface LinkEventToCardRequest {
  card_pk: number;
}

/**
 * Request to create a card from an external event
 */
export interface CreateCardFromEventRequest {
  title?: string;
  body?: string;
}
