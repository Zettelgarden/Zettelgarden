import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateAgentModal } from './CreateAgentModal';
import * as agentsApi from '../api/agents';

// Mock the agents API
vi.mock('../api/agents', () => ({
  createAgent: vi.fn(),
}));

describe('CreateAgentModal', () => {
  const mockOnClose = vi.fn();
  const mockOnSuccess = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders nothing when isOpen is false', () => {
    const { container } = render(
      <CreateAgentModal isOpen={false} onClose={mockOnClose} onSuccess={mockOnSuccess} />
    );
    expect(container.querySelector('.fixed')).not.toBeInTheDocument();
  });

  it('renders modal when isOpen is true', () => {
    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    expect(screen.getByText('Create New Agent')).toBeInTheDocument();
    expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', async () => {
    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    const closeButton = screen.getByRole('button', { name: /close/i });
    await userEvent.click(closeButton);
    expect(mockOnClose).toHaveBeenCalledTimes(1);
  });

  it('submits form with name and description', async () => {
    const mockCreateAgent = vi.mocked(agentsApi.createAgent);
    mockCreateAgent.mockResolvedValueOnce({
      id: 1,
      name: 'Test Agent',
      description: 'Test description',
      api_key: 'test-api-key-123',
      created_at: '2024-01-01T00:00:00Z',
      is_active: true,
    });

    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    
    const nameInput = screen.getByLabelText(/name/i);
    const descriptionInput = screen.getByLabelText(/description/i);
    const submitButton = screen.getByRole('button', { name: /create agent/i });

    await userEvent.type(nameInput, 'Test Agent');
    await userEvent.type(descriptionInput, 'Test description');
    await userEvent.click(submitButton);

    expect(mockCreateAgent).toHaveBeenCalledWith('Test Agent', 'Test description');
  });

  it('shows API key after successful creation', async () => {
    const mockCreateAgent = vi.mocked(agentsApi.createAgent);
    mockCreateAgent.mockResolvedValueOnce({
      id: 1,
      name: 'Test Agent',
      description: 'Test description',
      api_key: 'test-api-key-123',
      created_at: '2024-01-01T00:00:00Z',
      is_active: true,
    });

    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    
    const nameInput = screen.getByLabelText(/name/i);
    const submitButton = screen.getByRole('button', { name: /create agent/i });

    await userEvent.type(nameInput, 'Test Agent');
    await userEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/api key created successfully/i)).toBeInTheDocument();
      expect(screen.getByText('test-api-key-123')).toBeInTheDocument();
    });
  });

  it('shows warning that API key will only be shown once', async () => {
    const mockCreateAgent = vi.mocked(agentsApi.createAgent);
    mockCreateAgent.mockResolvedValueOnce({
      id: 1,
      name: 'Test Agent',
      description: 'Test description',
      api_key: 'test-api-key-123',
      created_at: '2024-01-01T00:00:00Z',
      is_active: true,
    });

    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    
    const nameInput = screen.getByLabelText(/name/i);
    const submitButton = screen.getByRole('button', { name: /create agent/i });

    await userEvent.type(nameInput, 'Test Agent');
    await userEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/only be shown once/i)).toBeInTheDocument();
    });
  });

  it('copies API key to clipboard when copy button is clicked', async () => {
    const mockCreateAgent = vi.mocked(agentsApi.createAgent);
    mockCreateAgent.mockResolvedValueOnce({
      id: 1,
      name: 'Test Agent',
      description: 'Test description',
      api_key: 'test-api-key-123',
      created_at: '2024-01-01T00:00:00Z',
      is_active: true,
    });

    const mockClipboardWrite = vi.fn();
    Object.assign(navigator, {
      clipboard: { writeText: mockClipboardWrite },
    });

    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    
    const nameInput = screen.getByLabelText(/name/i);
    const submitButton = screen.getByRole('button', { name: /create agent/i });

    await userEvent.type(nameInput, 'Test Agent');
    await userEvent.click(submitButton);

    await waitFor(async () => {
      const copyButton = screen.getByRole('button', { name: /copy key/i });
      await userEvent.click(copyButton);
      expect(mockClipboardWrite).toHaveBeenCalledWith('test-api-key-123');
    });
  });

  it('shows error message when creation fails', async () => {
    const mockCreateAgent = vi.mocked(agentsApi.createAgent);
    mockCreateAgent.mockRejectedValueOnce(new Error('Creation failed'));

    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    
    const nameInput = screen.getByLabelText(/name/i);
    const submitButton = screen.getByRole('button', { name: /create agent/i });

    await userEvent.type(nameInput, 'Test Agent');
    await userEvent.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/failed to create agent/i)).toBeInTheDocument();
    });
  });

  it('disables submit button while loading', async () => {
    const mockCreateAgent = vi.mocked(agentsApi.createAgent);
    mockCreateAgent.mockImplementation(() => new Promise(() => {})); // Never resolves

    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    
    const nameInput = screen.getByLabelText(/name/i);
    const submitButton = screen.getByRole('button', { name: /create agent/i });

    await userEvent.type(nameInput, 'Test Agent');
    await userEvent.click(submitButton);

    expect(submitButton).toBeDisabled();
  });

  it('requires name field', async () => {
    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    
    const submitButton = screen.getByRole('button', { name: /create agent/i });
    expect(submitButton).toBeDisabled();
  });

  it('calls onSuccess after successful creation and confirmation', async () => {
    const mockCreateAgent = vi.mocked(agentsApi.createAgent);
    mockCreateAgent.mockResolvedValueOnce({
      id: 1,
      name: 'Test Agent',
      description: 'Test description',
      api_key: 'test-api-key-123',
      created_at: '2024-01-01T00:00:00Z',
      is_active: true,
    });

    render(<CreateAgentModal isOpen={true} onClose={mockOnClose} onSuccess={mockOnSuccess} />);
    
    const nameInput = screen.getByLabelText(/name/i);
    const submitButton = screen.getByRole('button', { name: /create agent/i });

    await userEvent.type(nameInput, 'Test Agent');
    await userEvent.click(submitButton);

    await waitFor(async () => {
      const confirmButton = screen.getByRole('button', { name: /i've copied it/i });
      await userEvent.click(confirmButton);
      expect(mockOnSuccess).toHaveBeenCalled();
      expect(mockOnClose).toHaveBeenCalled();
    });
  });
});
