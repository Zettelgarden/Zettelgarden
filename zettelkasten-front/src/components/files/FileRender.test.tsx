import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { FileRender } from './FileRender';
import { File } from '../../models/File';

vi.mock('../../api/files', () => ({
  downloadFile: vi.fn(),
  renderFile: vi.fn(),
}));

const { downloadFile } = await import('../../api/files');

function baseFile(): File {
  return {
    id: 7,
    name: 'notes.md',
    filetype: 'text/markdown',
    path: '/files/7',
    filename: 'notes.md',
    size: 100,
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
  };
}

describe('FileRender', () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(() => vi.unstubAllGlobals());

  it('renders extracted text for a markdown file without downloading', async () => {
    const file = {
      ...baseFile(),
      extracted_text: '# Hello\n\nSome **markdown** content.',
    };
    renderWithProviders(<FileRender file={file} />);

    expect(
      await screen.findByRole('heading', { name: 'Hello' }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Some/)).toBeInTheDocument();
    expect(downloadFile).not.toHaveBeenCalled();
  });

  it('renders plain text files in a pre block with copy affordance', async () => {
    const file = {
      ...baseFile(),
      name: 'config.yaml',
      filetype: 'application/x-yaml',
      extracted_text: 'theme: dark\nport: 8080',
    };
    const clipboard = { writeText: vi.fn().mockResolvedValue(undefined) };
    Object.defineProperty(navigator, 'clipboard', {
      value: clipboard,
      configurable: true,
    });

    renderWithProviders(<FileRender file={file} />);

    expect(await screen.findByText(/theme: dark/)).toBeInTheDocument();
    fireEvent.click(screen.getByText('Copy'));
    await waitFor(() =>
      expect(clipboard.writeText).toHaveBeenCalledWith(
        'theme: dark\nport: 8080',
      ),
    );
  });

  it('reads the downloaded blob as text when no extracted_text exists', async () => {
    const file = { ...baseFile(), extracted_text: undefined };
    vi.mocked(downloadFile).mockResolvedValue('blob:mock-url');
    const realFetch = globalThis.fetch.bind(globalThis);
    vi.stubGlobal(
      'fetch',
      vi.fn((url: RequestInfo | URL) =>
        String(url).startsWith('blob:')
          ? Promise.resolve(
              new Response('downloaded file contents', { status: 200 }),
            )
          : realFetch(url),
      ),
    );

    renderWithProviders(<FileRender file={file} />);

    expect(
      await screen.findByText(/downloaded file contents/),
    ).toBeInTheDocument();
  });

  it('shows "Preview not available" for non-text blobs', async () => {
    const file = {
      ...baseFile(),
      name: 'data.bin',
      filetype: 'application/octet-stream',
    };
    vi.mocked(downloadFile).mockResolvedValue('blob:mock-url');
    // Binary-ish content: NUL bytes / replacement chars are not previewable.
    const realFetch = globalThis.fetch.bind(globalThis);
    vi.stubGlobal(
      'fetch',
      vi.fn((url: RequestInfo | URL) =>
        String(url).startsWith('blob:')
          ? Promise.resolve(new Response('bad\u0000bytes', { status: 200 }))
          : realFetch(url),
      ),
    );

    renderWithProviders(<FileRender file={file} />);

    expect(
      await screen.findByText(/Preview not available/),
    ).toBeInTheDocument();
  });
});
