import { apiClient } from './client';
import { GraphData, NetworkStats } from '../models/Graph';

/**
 * Fetch the user's knowledge graph. Optional types filter: comma-separated
 * subset of 'card', 'entity', 'tag' (default all).
 */
export async function getGraphData(types?: string): Promise<GraphData> {
  const params = types ? `?types=${encodeURIComponent(types)}` : '';
  const { data } = await apiClient.get<GraphData>(`/graph${params}`);
  return data;
}

/**
 * Fetch network health metrics for the user's vault.
 */
export async function getNetworkStats(): Promise<NetworkStats> {
  const { data } = await apiClient.get<NetworkStats>('/graph/stats');
  return data;
}
