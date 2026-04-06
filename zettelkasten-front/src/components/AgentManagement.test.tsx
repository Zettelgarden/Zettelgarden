import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AgentManagement } from './AgentManagement';
import * as agentsApi from '../api/agents';

// Mock the agents API
vi.mock('../api/agents', () => ({
  listAgents: vi.fn(),
  revokeAgent: vi.fn(),
}));

// Mock the modal components
vi.mock('./CreateAgentModal', () => ({
  CreateAgentModal: ({ isOpen, onClose, onSuccess }: any) => 
    isOpen ? (
      <div data-testid="create-agent-modal">
        <button onClick={() => { onSuccess(); onClose(); }}>Mock Create Modal</button>
      </div>
    ) : null,
}));

vi.mock('./AgentActivityModal', () => ({
  AgentActivityModal: ({ isOpen, onClose, agentId }: any) => 
    isOpen ? (
      <div data-testid="agent-activity-modal">
        <button onClick={onClose}>Mock Activity Modal</button>
      </div>
    ) : null,
}));

describe('AgentManagement', () => {
  const mockAgents = [
    {
      id: 1,
      name: 'Test Agent 1',
      description: 'Description 1',
      created_at: '2024-01-01T00:00:00Z',
      last_used: '2024-01-02T00:00:00Z',
      is_active: true,
    },
    {
      id: 2,
      name: 'Test Agent 2',
      description: 'Description 2',
      created_at: '2024-01-01T00:00:00Z',
      last_used: null,
      is_active: false,
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state initially', () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockImplementation(() => new Promise(() => {})); // Never resolves

    render(<AgentManagement />);

    expect(screen.getByText(/loading agents/i)).toBeInTheDocument();
  });

  it('loads agents on mount', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(mockListAgents).toHaveBeenCalled();
    });
  });

  it('displays agents list after loading', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Test Agent 1')).toBeInTheDocument();
      expect(screen.getByText('Test Agent 2')).toBeInTheDocument();
    });
  });

  it('shows agent status (active/revoked)', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Active')).toBeInTheDocument();
      expect(screen.getByText('Revoked')).toBeInTheDocument();
    });
  });

  it('shows error message when loading fails', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockRejectedValueOnce(new Error('Failed to load'));

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText(/failed to load agents/i)).toBeInTheDocument();
    });
  });

  it('shows empty state when no agents exist', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce([]);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText(/no agents yet/i)).toBeInTheDocument();
    });
  });

  it('opens create modal when "Create Agent" button is clicked', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Test Agent 1')).toBeInTheDocument();
    });

    const createButton = screen.getByRole('button', { name: /create new agent/i });
    await userEvent.click(createButton);

    expect(screen.getByTestId('create-agent-modal')).toBeInTheDocument();
  });

  it('refreshes agents list after creating an agent', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);
    mockListAgents.mockResolvedValueOnce([...mockAgents, {
      id: 3,
      name: 'New Agent',
      created_at: '2024-01-03T00:00:00Z',
      is_active: true,
    }]);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(mockListAgents).toHaveBeenCalledTimes(1);
    });

    const createButton = screen.getByRole('button', { name: /create new agent/i });
    await userEvent.click(createButton);

    const mockCreateButton = screen.getByText('Mock Create Modal');
    await userEvent.click(mockCreateButton);

    await waitFor(() => {
      expect(mockListAgents).toHaveBeenCalledTimes(2);
    });
  });

  it('opens activity modal when "View Activity" button is clicked', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Test Agent 1')).toBeInTheDocument();
    });

    const viewActivityButtons = screen.getAllByRole('button', { name: /view activity/i });
    await userEvent.click(viewActivityButtons[0]);

    expect(screen.getByTestId('agent-activity-modal')).toBeInTheDocument();
  });

  it('revokes agent after confirmation', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    const mockRevokeAgent = vi.mocked(agentsApi.revokeAgent);
    mockListAgents.mockResolvedValueOnce(mockAgents);
    mockListAgents.mockResolvedValueOnce([mockAgents[1]]); // After revoke, only second agent
    mockRevokeAgent.mockResolvedValueOnce(undefined);

    // Mock window.confirm
    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Test Agent 1')).toBeInTheDocument();
    });

    const revokeButtons = screen.getAllByRole('button', { name: /revoke/i });
    await userEvent.click(revokeButtons[0]);

    expect(mockRevokeAgent).toHaveBeenCalledWith(1);
    expect(mockListAgents).toHaveBeenCalledTimes(2); // Initial load + refresh after revoke
  });

  it('does not revoke agent if confirmation is cancelled', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    const mockRevokeAgent = vi.mocked(agentsApi.revokeAgent);
    mockListAgents.mockResolvedValueOnce(mockAgents);
    mockRevokeAgent.mockResolvedValueOnce(undefined);

    // Mock window.confirm to return false
    vi.spyOn(window, 'confirm').mockReturnValueOnce(false);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Test Agent 1')).toBeInTheDocument();
    });

    const revokeButtons = screen.getAllByRole('button', { name: /revoke/i });
    await userEvent.click(revokeButtons[0]);

    expect(mockRevokeAgent).not.toHaveBeenCalled();
  });

  it('shows error when revoking fails', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    const mockRevokeAgent = vi.mocked(agentsApi.revokeAgent);
    mockListAgents.mockResolvedValueOnce(mockAgents);
    mockRevokeAgent.mockRejectedValueOnce(new Error('Revoke failed'));

    vi.spyOn(window, 'confirm').mockReturnValueOnce(true);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Test Agent 1')).toBeInTheDocument();
    });

    const revokeButtons = screen.getAllByRole('button', { name: /revoke/i });
    await userEvent.click(revokeButtons[0]);

    await waitFor(() => {
      expect(screen.getByText(/failed to revoke agent/i)).toBeInTheDocument();
    });
  });

  it('does not show revoke button for inactive agents', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Test Agent 2')).toBeInTheDocument();
    });

    // Find the row for the inactive agent
    const agent2Row = screen.getByText('Test Agent 2').closest('div');
    const revokeButtonsInRow = agent2Row?.querySelectorAll('button');
    
    // Should only have "View Activity" button, not "Revoke"
    expect(revokeButtonsInRow?.length).toBe(1);
  });

  it('formats dates correctly', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText(/created:/i)).toBeInTheDocument();
      expect(screen.getByText(/last used:/i)).toBeInTheDocument();
    });
  });

  it('shows "Never" for agents that have never been used', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Test Agent 2')).toBeInTheDocument();
    });

    // The second agent has last_used: null, should show "Never"
    const agent2Section = screen.getByText('Test Agent 2').parentElement;
    expect(agent2Section).toHaveTextContent('Never');
  });

  it('displays agent descriptions', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Description 1')).toBeInTheDocument();
      expect(screen.getByText('Description 2')).toBeInTheDocument();
    });
  });

  it('shows page header with description', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockResolvedValueOnce(mockAgents);

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText('Agents')).toBeInTheDocument();
      expect(screen.getByText(/create and manage ai agents/i)).toBeInTheDocument();
    });
  });

  it('dismisses error when dismiss button is clicked', async () => {
    const mockListAgents = vi.mocked(agentsApi.listAgents);
    mockListAgents.mockRejectedValueOnce(new Error('Failed to load'));

    render(<AgentManagement />);

    await waitFor(() => {
      expect(screen.getByText(/failed to load agents/i)).toBeInTheDocument();
    });

    const dismissButton = screen.getByRole('button', { name: /dismiss/i });
    await userEvent.click(dismissButton);

    expect(screen.queryByText(/failed to load agents/i)).not.toBeInTheDocument();
  });
});
