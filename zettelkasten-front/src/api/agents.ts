/**
 * API client for Agent management
 */

import { apiClient, getData } from './client';
import {
  Agent,
  CreateAgentRequest,
  CreateAgentResponse,
  AgentActivityResponse,
} from '../models/Agent';

/**
 * Create a new agent
 */
export async function createAgent(
  name: string,
  description?: string,
): Promise<CreateAgentResponse> {
  const request: CreateAgentRequest = { name, description };
  return getData(apiClient.post<CreateAgentResponse>('/api/agents', request));
}

/**
 * List all agents for the current user
 */
export async function listAgents(): Promise<Agent[]> {
  const response = await getData(apiClient.get<{ agents: Agent[] }>('/api/agents'));
  return response.agents;
}

/**
 * Revoke (deactivate) an agent
 */
export async function revokeAgent(agentId: number): Promise<void> {
  await getData(apiClient.delete<void>(`/api/agents/${agentId}`));
}

/**
 * Get activity log for a specific agent
 */
export async function getAgentActivity(
  agentId: number,
  page: number = 1,
  perPage: number = 50,
): Promise<AgentActivityResponse> {
  return getData(
    apiClient.get<AgentActivityResponse>(`/api/agents/${agentId}/activity`, {
      params: { page, per_page: perPage },
    }),
  );
}
