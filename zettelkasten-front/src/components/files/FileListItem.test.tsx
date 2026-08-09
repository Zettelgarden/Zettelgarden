import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '../../tests/utils';
import { FileListItem } from './FileListItem';
import { File } from '../../models/File';

vi.mock('../../api/files', () => ({
  renderFile: vi.fn(),
  deleteFile: vi.fn(),
  editFile: vi.fn(),
  downloadThumbnail: vi.fn(),
  downloadFile: vi.fn(),
  importEpub: vi.fn(),
}));

// A result shaped like the old Typesense search response: card_pk is set but
// the card object is missing entirely (Zettelgarden-72f.2).
function typesenseShapedFile(): File {
  return {
    id: 42,
    name: 'report.pdf',
    filetype: 'application/pdf',
    path: '/files/42',
    filename: 'report.pdf',
    size: 2048,
    created_by: 1,
    updated_by: 1,
    card_pk: 7, // linked to a card...
    is_deleted: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    thumbnail_path: null,
    // ...but no card object came back with the result
    card: undefined as unknown as File['card'],
  };
}

describe('FileListItem', () => {
  it('renders a Typesense-shaped result (card missing) without crashing', () => {
    renderWithProviders(
      <FileListItem
        file={typesenseShapedFile()}
        onDelete={() => {}}
        setRefreshFiles={() => {}}
        filterString=""
        setFilterString={() => {}}
      />,
    );

    expect(screen.getByText('report.pdf')).toBeInTheDocument();
    // No card link is rendered when the card object is missing
    expect(screen.queryByText(/^\[/)).toBeNull();
  });

  it('renders the card link when the file includes a card', () => {
    const linkedFile = typesenseShapedFile();
    linkedFile.card = {
      id: 7,
      card_id: 'card-7',
      user_id: 1,
      title: 'A linked card',
      parent_id: 0,
      created_at: new Date(),
      updated_at: new Date(),
      tags: [],
    };

    renderWithProviders(
      <FileListItem
        file={linkedFile}
        onDelete={() => {}}
        setRefreshFiles={() => {}}
        filterString=""
        setFilterString={() => {}}
      />,
    );

    expect(screen.getByText('[card-7]')).toBeInTheDocument();
  });

  it('renders a snippet with the matching field when present', () => {
    const file = typesenseShapedFile();
    file.snippet = '…quarterly budget review…';
    file.snippet_field = 'content';

    renderWithProviders(
      <FileListItem
        file={file}
        onDelete={() => {}}
        setRefreshFiles={() => {}}
        filterString=""
        setFilterString={() => {}}
      />,
    );

    expect(screen.getByText('Content')).toBeInTheDocument();
    expect(screen.getByText('…quarterly budget review…')).toBeInTheDocument();
  });
});
