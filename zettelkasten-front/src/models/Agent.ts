/**
 * TypeScript interfaces for Agent API
 */

export interface Agent {
  id: number;
  name: string;
  description?: string;
  created_at: string;
  last_used?: string;
  is_active: boolean;
}

export interface CreateAgentRequest {
  name: string;
  description?: string;
}

export interface CreateAgentResponse extends Agent {
  api_key: string; // Only shown once!
}

export interface AgentActivityLog {
  id: number;
  agent_id: number;
  action: string;
  target_type: string;
  target_id?: number;
  details?: Record<string, any>;
  created_at: string;
}

export interface AgentActivityResponse {
  logs: AgentActivityLog[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}
