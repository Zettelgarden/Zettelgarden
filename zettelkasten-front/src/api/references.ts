import { ParseResult } from '../models/Reference';
import { apiClient, getData } from './client';

export function parseURL(url: string): Promise<ParseResult> {
  return getData(apiClient.post<ParseResult>('/url/parse', { url }));
}
