// @vitest-environment happy-dom

import React from 'react';
import {
  render,
  cleanup,
  screen,
  fireEvent,
  waitFor,
} from '@testing-library/react';
import { describe, it, vi, expect, beforeEach, afterEach } from 'vitest';
import type { Mock } from 'vitest';
import { SavedSearchesMenu } from './SavedSearchesMenu';
import { ToastProvider } from '../toast/ToastContext';
import * as api from '../../api/taskSavedSearches';
import type { TaskSavedSearch } from '../../models/TaskSavedSearch';

// Mock the API module so the hook never hits the network.
vi.mock('../../api/taskSavedSearches', () => ({
  fetchTaskSavedSearches: vi.fn(),
  createTaskSavedSearch: vi.fn(),
  updateTaskSavedSearch: vi.fn(),
  deleteTaskSavedSearch: vi.fn(),
}));

function renderMenu(
  props: Partial<React.ComponentProps<typeof SavedSearchesMenu>> = {},
) {
  const onApply = vi.fn();
  return {
    onApply,
    ...render(
      <ToastProvider>
        <SavedSearchesMenu
          filterString=""
          sortField="priority"
          sortDirection="asc"
          viewMode="list"
          onApply={onApply}
          {...props}
        />
      </ToastProvider>,
    ),
  };
}

const sample = (over: Partial<TaskSavedSearch> = {}): TaskSavedSearch => ({
  id: 1,
  user_id: 1,
  name: 'Overdue',
  filter_string: 'date:overdue',
  sort_field: 'priority',
  sort_direction: 'desc',
  view_mode: 'list',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...over,
});

describe('SavedSearchesMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (api.fetchTaskSavedSearches as Mock).mockResolvedValue([]);
  });

  afterEach(() => cleanup());

  it('loads and lists saved searches when opened', async () => {
    (api.fetchTaskSavedSearches as Mock).mockResolvedValue([
      sample(),
      sample({ id: 2, name: 'This week', filter_string: 'date:this_week' }),
    ]);

    const { container } = renderMenu();
    fireEvent.click(screen.getByTitle('Saved searches'));

    await waitFor(() => {
      expect(screen.getByText('Overdue')).toBeInTheDocument();
      expect(screen.getByText('This week')).toBeInTheDocument();
    });
    // Empty-state copy should not appear once searches loaded.
    expect(container.textContent).not.toContain('No saved searches yet');
  });

  it('shows the empty state when there are no searches', async () => {
    renderMenu();
    fireEvent.click(screen.getByTitle('Saved searches'));
    await waitFor(() => {
      expect(screen.getByText(/No saved searches yet/)).toBeInTheDocument();
    });
  });

  it('applies a saved search via onApply', async () => {
    const s = sample();
    (api.fetchTaskSavedSearches as Mock).mockResolvedValue([s]);

    const { onApply } = renderMenu();
    fireEvent.click(screen.getByTitle('Saved searches'));
    const button = await screen.findByText('Overdue');
    fireEvent.click(button);

    expect(onApply).toHaveBeenCalledWith(s);
  });

  it('creates a search from the current filter state', async () => {
    (api.createTaskSavedSearch as Mock).mockResolvedValue({ id: 5 });

    renderMenu({
      filterString: '#work date:today',
      sortField: 'title',
      sortDirection: 'asc',
      viewMode: 'kanban',
    });
    fireEvent.click(screen.getByTitle('Saved searches'));

    fireEvent.click(screen.getByText('Save current search'));
    const input = await screen.findByPlaceholderText('Search name');
    fireEvent.change(input, { target: { value: 'Work today' } });
    fireEvent.click(screen.getByText('Save'));

    await waitFor(() => {
      expect(api.createTaskSavedSearch).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'Work today',
          filter_string: '#work date:today',
          sort_field: 'title',
          view_mode: 'kanban',
        }),
      );
    });
  });

  it('deletes a search and refetches', async () => {
    (api.fetchTaskSavedSearches as Mock).mockResolvedValue([sample()]);
    (api.deleteTaskSavedSearch as Mock).mockResolvedValue(undefined);

    renderMenu();
    fireEvent.click(screen.getByTitle('Saved searches'));
    const deleteBtn = await screen.findByTitle('Delete search');
    fireEvent.click(deleteBtn);

    await waitFor(() => {
      expect(api.deleteTaskSavedSearch).toHaveBeenCalledWith(1);
    });
  });
});
