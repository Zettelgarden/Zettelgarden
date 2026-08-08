// @vitest-environment happy-dom

import React from 'react';
import {
  render,
  cleanup,
  screen,
  fireEvent,
  waitFor,
} from '@testing-library/react';
import { CreateTaskWindow } from './CreateTaskWindow';
import { saveNewTask } from '../../api/tasks';
import { describe, it, vi, expect, beforeEach, afterEach } from 'vitest';
import { renderWithProviders } from '../../tests/utils';
import type { Mock } from 'vitest';

// Mock the API call
vi.mock('../../api/tasks', () => ({
  saveNewTask: vi.fn(),
}));

// Mock the TaskDateDisplay component to avoid the useAuth dependency
vi.mock('./TaskDateDisplay', () => ({
  TaskDateDisplay: vi.fn(() =>
    React.createElement(
      'div',
      { 'data-testid': 'task-date-display' },
      'Task Date',
    ),
  ),
}));

describe('CreateTaskWindow', () => {
  const mockSetRefresh = vi.fn();
  const mockSetShowTaskWindow = vi.fn();
  const defaultProps = {
    currentCard: null,
    setRefresh: mockSetRefresh,
    setShowTaskWindow: mockSetShowTaskWindow,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it('should render without crashing', () => {
    renderWithProviders(<CreateTaskWindow {...defaultProps} />);
    expect(screen.getByText(/New Task/i));
  });

  it('should call saveNewTask API when the save button is clicked', async () => {
    // Mock the API response
    (saveNewTask as Mock).mockResolvedValue({ id: 1, title: 'Test Task' });

    renderWithProviders(<CreateTaskWindow {...defaultProps} />);

    // Simulate user typing into the input field
    const input = screen.getByPlaceholderText('Enter task title');
    fireEvent.change(input, { target: { value: 'Test Task' } });

    // Simulate clicking the Save button
    const saveButton = screen.getByText('Save');
    fireEvent.click(saveButton);

    // Wait and assert that saveNewTask was called
    await waitFor(() => {
      expect(saveNewTask).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'Test Task',
        }),
      );
    });

    // Check if setRefresh and setShowTaskWindow were called
    await waitFor(() => {
      expect(mockSetShowTaskWindow).toHaveBeenCalledWith(false);
    });
  });

  // it('should handle errors from saveNewTask API', async () => {
  //   // Mock the API to reject
  //   saveNewTask.mockRejectedValue(new Error('Failed to save task'));

  //   renderWithProviders(<CreateTaskWindow {...defaultProps} />);

  //   const input = screen.getByPlaceholderText('Enter new task');
  //   fireEvent.change(input, { target: { value: 'Erroneous Task' } });

  //   const saveButton = screen.getByText('Save');
  //   fireEvent.click(saveButton);

  //   // Wait for error handling (you might display error text or log it)
  //   await waitFor(() => {
  //     // Example: Check if an error message is shown
  //     // Depending on your component's error handling implementation
  //   });
  // });
});
