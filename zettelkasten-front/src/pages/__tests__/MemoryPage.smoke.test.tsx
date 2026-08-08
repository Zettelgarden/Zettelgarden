/**
 * Smoke test: MemoryPage renders memory content and enters edit mode (a5q.6).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { MemoryPage } from '../MemoryPage';

vi.mock('../../api/users', () => ({
  getUserMemory: vi.fn(),
  updateUserMemory: vi.fn(),
}));

const { getUserMemory, updateUserMemory } = await import('../../api/users');

describe('MemoryPage smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders stored memory content', async () => {
    vi.mocked(getUserMemory).mockResolvedValue({
      memory: 'I prefer concise answers.',
    });

    renderWithProviders(<MemoryPage />);

    await waitFor(() =>
      expect(screen.getByText('I prefer concise answers.')).toBeInTheDocument(),
    );
    expect(screen.getByText('Edit Memory')).toBeInTheDocument();
  });

  it('renders the empty state when no memory exists', async () => {
    vi.mocked(getUserMemory).mockResolvedValue({ memory: '' });

    renderWithProviders(<MemoryPage />);

    await waitFor(() =>
      expect(screen.getByText(/No memory content yet/)).toBeInTheDocument(),
    );
  });

  it('enters edit mode and saves updates', async () => {
    vi.mocked(getUserMemory).mockResolvedValue({
      memory: 'Original memory',
    });
    vi.mocked(updateUserMemory).mockResolvedValue(undefined as any);

    renderWithProviders(<MemoryPage />);

    await waitFor(() =>
      expect(screen.getByText('Original memory')).toBeInTheDocument(),
    );

    fireEvent.click(screen.getByText('Edit Memory'));

    const textarea = screen.getByPlaceholderText(/Enter your memory content/i);
    fireEvent.change(textarea, { target: { value: 'Updated memory' } });
    fireEvent.click(screen.getByText('Save'));

    await waitFor(() =>
      expect(updateUserMemory).toHaveBeenCalledWith('Updated memory'),
    );
    // Back in view mode: the Edit button is visible again and the
    // textarea is gone; the saved memory renders as markdown.
    await waitFor(() => {
      expect(screen.getByText('Edit Memory')).toBeInTheDocument();
      expect(
        screen.queryByPlaceholderText(/Enter your memory content/i),
      ).toBeNull();
    });
    expect(screen.getByText('Updated memory')).toBeInTheDocument();
  });
});
