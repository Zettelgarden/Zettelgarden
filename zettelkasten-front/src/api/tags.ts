import { Tag } from '../models/Tags';
import { apiClient, getData } from './client';

export function fetchUserTags(): Promise<Tag[]> {
  return getData(apiClient.get<Tag[]>(`/tags`));
}

export function deleteTag(id: number): Promise<Tag | null> {
  return getData(apiClient.delete<Tag>(`/tags/id/${id}`));
}

export interface CreateTagParams {
  name: string;
  color: string;
}

export function createTag(params: CreateTagParams): Promise<Tag> {
  return getData(apiClient.post<Tag>(`/tags`, params));
}
