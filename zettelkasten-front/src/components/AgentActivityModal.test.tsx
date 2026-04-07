import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AgentActivityModal } from './AgentActivityModal';
import * as agentsApi from '../api/agents';

// Mock the agents API
vi.mock('../api/agents', () => ({
  getAgentActivity: vi.fn(),
}));

describe('AgentActivityModal', () => {
  const mockOnClose = vi.fn();
  const mockAgentId = 1;
  const mockAgentName = 'Test Agent';

  const mockActivityLogs = [
    {
      id: 1,
      agent_id: 1,
      action: 'read_card',
      target_type: 'card',
      target_id: 123,
      details: { title: 'Test Card' },
      created_at: '2024-01-01T10:00:00Z',
    },
    {
      id: 2,
      agent_id: 1,
      action: 'create_card',
      target_type: 'card',
      target_id: 124,
      details: { title: 'New Card' },
      created_at: '2024-01-01T11:00:00Z',
    },
  ];

  const mockActivityResponse = {
    logs: mockActivityLogs,
    pagination: {
      page: 1,
      per_page: 50,
      total: 100,
      total_pages: 2,
    },
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders nothing when isOpen is false', () => {
    const { container } = render(
      <AgentActivityModal isOpen={false} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />
    );
    expect(container.querySelector('.fixed')).not.toBeInTheDocument();
  });

  it('renders modal when isOpen is true', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(screen.getByText('Agent Activity')).toBeInTheDocument();
    });
  });

  it('loads activity logs on open', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(mockGetAgentActivity).toHaveBeenCalledWith(mockAgentId, 1, 50);
    });
  });

  it('displays activity logs in a table', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(screen.getByText('read_card')).toBeInTheDocument();
      expect(screen.getByText('create_card')).toBeInTheDocument();
      expect(screen.getByText('Test Card')).toBeInTheDocument();
      expect(screen.getByText('New Card')).toBeInTheDocument();
    });
  });

  it('shows loading state while fetching data', () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockImplementation(() => new Promise(() => {})); // Never resolves

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it('shows error message when loading fails', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockRejectedValueOnce(new Error('Failed to load'));

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(screen.getByText(/failed to load activity/i)).toBeInTheDocument();
    });
  });

  it('shows pagination controls', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /previous/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /next/i })).toBeInTheDocument();
    });
  });

  it('disables previous button on first page', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      const prevButton = screen.getByRole('button', { name: /previous/i });
      expect(prevButton).toBeDisabled();
    });
  });

  it('disables next button on last page', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce({
      ...mockActivityResponse,
      pagination: { ...mockActivityResponse.pagination, page: 2, total_pages: 2 },
    });

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      const nextButton = screen.getByRole('button', { name: /next/i });
      expect(nextButton).toBeDisabled();
    });
  });

  it('loads next page when next button is clicked', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(mockGetAgentActivity).toHaveBeenCalledWith(mockAgentId, 1, 50);
    });

    mockGetAgentActivity.mockResolvedValueOnce({
      ...mockActivityResponse,
      pagination: { ...mockActivityResponse.pagination, page: 2 },
      logs: [],
    });

    const nextButton = screen.getByRole('button', { name: /next/i });
    await userEvent.click(nextButton);

    await waitFor(() => {
      expect(mockGetAgentActivity).toHaveBeenCalledWith(mockAgentId, 2, 50);
    });
  });

  it('loads previous page when previous button is clicked', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce({
      ...mockActivityResponse,
      pagination: { ...mockActivityResponse.pagination, page: 2 },
    });

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(mockGetAgentActivity).toHaveBeenCalledWith(mockAgentId, 2, 50);
    });

    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    const prevButton = screen.getByRole('button', { name: /previous/i });
    await userEvent.click(prevButton);

    await waitFor(() => {
      expect(mockGetAgentActivity).toHaveBeenCalledWith(mockAgentId, 1, 50);
    });
  });

  it('shows page information', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(screen.getByText(/page 1 of 2/i)).toBeInTheDocument();
    });
  });

  it('calls onClose when close button is clicked', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(screen.getByText('Agent Activity')).toBeInTheDocument();
    });

    const closeButton = screen.getByRole('button', { name: /close/i });
    await userEvent.click(closeButton);
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('shows empty state when no activity logs', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce({
      logs: [],
      pagination: { page: 1, per_page: 50, total: 0, total_pages: 0 },
    });

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(screen.getByText(/no activity logs/i)).toBeInTheDocument();
    });
  });

  it('formats timestamps correctly', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      // The exact format will depend on the locale, but we can check it's formatted
      expect(screen.getByText(/2024/)).toBeInTheDocument();
    });
  });

  it('displays action, target type, and details in table headers', async () => {
    const mockGetAgentActivity = vi.mocked(agentsApi.getAgentActivity);
    mockGetAgentActivity.mockResolvedValueOnce(mockActivityResponse);

    render(<AgentActivityModal isOpen={true} onClose={mockOnClose} agentId={mockAgentId} agentName={mockAgentName} />);

    await waitFor(() => {
      expect(screen.getByText('Action')).toBeInTheDocument();
      expect(screen.getByText('Target')).toBeInTheDocument();
      expect(screen.getByText('Details')).toBeInTheDocument();
      expect(screen.getByText('Timestamp')).toBeInTheDocument();
    });
  });
});
