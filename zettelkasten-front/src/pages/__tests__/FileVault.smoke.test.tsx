/**
 * Smoke test: FileVault renders the file list and storage quota (a5q.6).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { FileVault } from '../FileVault';
import { File } from '../../models/File';

vi.mock('../../api/files', () => ({
  getAllFiles: vi.fn(),
  deleteFile: vi.fn(),
  editFile: vi.fn(),
  uploadFile: vi.fn(),
  renderFile: vi.fn(),
  downloadFile: vi.fn(),
  downloadThumbnail: vi.fn(),
  importEpub: vi.fn(),
}));

const { getAllFiles, deleteFile, renderFile } = await import('../../api/files');

const files: File[] = [
  {
    id: 1,
    name: 'zettelkasten-notes.md',
    filetype: 'md',
    path: '/files/1',
    filename: 'zettelkasten-notes.md',
    size: 2048,
    created_by: 1,
    updated_by: 1,
    card_pk: 0,
    is_deleted: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    thumbnail_path: null,
    card: {
      id: 0,
      card_id: '',
      user_id: 1,
      title: '',
      parent_id: 0,
      created_at: new Date(),
      updated_at: new Date(),
      tags: [],
    },
  },
];

describe('FileVault smoke', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the file list with mocked data', async () => {
    vi.mocked(getAllFiles).mockResolvedValue({
      files,
      page: 1,
      per_page: 20,
      total: 1,
      total_pages: 1,
      storage_used: 2048,
      max_storage: 104857600,
    });

    renderWithProviders(<FileVault />);

    // The "Files" header renders during loading too; wait for the row.
    await waitFor(() =>
      expect(screen.getByText('zettelkasten-notes.md')).toBeInTheDocument(),
    );
    // Storage quota indicator renders when max_storage > 0
    expect(screen.getByText(/Storage/)).toBeInTheDocument();
    // Exactly one GET /files on mount (72f.7 — no duplicate fetches)
    expect(getAllFiles).toHaveBeenCalledTimes(1);
  });

  it('renders an empty state when there are no files', async () => {
    vi.mocked(getAllFiles).mockResolvedValue({
      files: [],
      page: 1,
      per_page: 20,
      total: 0,
      total_pages: 1,
      storage_used: 0,
      max_storage: 0,
    });

    renderWithProviders(<FileVault />);

    await waitFor(() =>
      expect(screen.getByText('No files yet')).toBeInTheDocument(),
    );
    expect(screen.queryByText('zettelkasten-notes.md')).toBeNull();
  });

  it('highlights a row with ArrowDown and deletes it with Delete', async () => {
    vi.mocked(getAllFiles).mockResolvedValue({
      files,
      page: 1,
      per_page: 20,
      total: 1,
      total_pages: 1,
      storage_used: 2048,
      max_storage: 104857600,
    });
    vi.mocked(deleteFile).mockResolvedValue({
      message: 'deleted',
      error: false,
    });

    renderWithProviders(<FileVault />);
    await waitFor(() =>
      expect(screen.getByText('zettelkasten-notes.md')).toBeInTheDocument(),
    );

    // ArrowDown highlights the first row
    fireEvent.keyDown(document, { key: 'ArrowDown' });
    const row = document.querySelector('[data-file-id="1"]');
    expect(row?.className).toContain('ring-2');

    // Delete on the highlighted row deletes just that row (after confirm)
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
    fireEvent.keyDown(document, { key: 'Delete' });
    await waitFor(() => expect(deleteFile).toHaveBeenCalledWith(1));
    confirmSpy.mockRestore();
  });

  it('opens the file preview with Enter on the highlighted row', async () => {
    vi.mocked(getAllFiles).mockResolvedValue({
      files,
      page: 1,
      per_page: 20,
      total: 1,
      total_pages: 1,
      storage_used: 2048,
      max_storage: 104857600,
    });
    vi.mocked(renderFile).mockResolvedValue(undefined);

    renderWithProviders(<FileVault />);
    await waitFor(() =>
      expect(screen.getByText('zettelkasten-notes.md')).toBeInTheDocument(),
    );

    fireEvent.keyDown(document, { key: 'ArrowDown' });
    fireEvent.keyDown(document, { key: 'Enter' });

    // .md is not an image/PDF, so the download/render path is used
    await waitFor(() =>
      expect(renderFile).toHaveBeenCalledWith(1, 'zettelkasten-notes.md'),
    );
  });
});
