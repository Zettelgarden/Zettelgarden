import { Tag } from '../models/Tags';
import { apiClient, getData } from './client';
import { getDataProvider } from '../data/provider';

/**
 * Fetch all user tags. Desktop: from the local mirror (instant, offline).
 */
export function fetchUserTags(): Promise<Tag[]> {
  return getDataProvider().fetchUserTags();
}

/**
 * Delete a tag. Desktop: queues a local delete, reconciles on reconnect.
 */
export function deleteTag(id: number): Promise<Tag | null> {
  return getDataProvider().deleteTag(id);
}

export interface CreateTagParams {
  name: string;
  color: string;
}

/**
 * Create a tag. Desktop: writes the local mirror + outbox; the server
 * name-merges on push (two devices creating "Work" converge to one row).
 */
export function createTag(params: CreateTagParams): Promise<Tag> {
  return getDataProvider().createTag(params);
}
