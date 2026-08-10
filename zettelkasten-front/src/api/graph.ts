import { apiClient } from './client';
import { GraphData } from '../models/Graph';

/**
 * Fetch the user's knowledge graph. Optional types filter: comma-separated
 * subset of 'card', 'entity', 'tag' (default all).
 */
export async function getGraphData(types?: string): Promise<GraphData> {
  const params = types ? `?types=${encodeURIComponent(types)}` : '';
  const { data } = await apiClient.get<GraphData>(`/graph${params}`);
  return data;
}
