import { describe, it, expect } from 'vitest';
import { parseFiletypeFilter } from './filetypeSearch';

describe('parseFiletypeFilter', () => {
  it('extracts a bare filetype: token', () => {
    expect(parseFiletypeFilter('filetype:pdf')).toEqual({
      searchText: '',
      filetype: 'pdf',
    });
  });

  it('keeps MIME types with slashes intact', () => {
    expect(parseFiletypeFilter('filetype:image/png')).toEqual({
      searchText: '',
      filetype: 'image/png',
    });
  });

  it('combines filetype: with a search term', () => {
    expect(parseFiletypeFilter('filetype:pdf quarterly')).toEqual({
      searchText: 'quarterly',
      filetype: 'pdf',
    });
  });

  it('returns plain search text when no filetype: token exists', () => {
    expect(parseFiletypeFilter('quarterly report')).toEqual({
      searchText: 'quarterly report',
      filetype: null,
    });
  });

  it('uses the first token when multiple are present', () => {
    expect(parseFiletypeFilter('filetype:pdf filetype:docx')).toEqual({
      searchText: '',
      filetype: 'pdf',
    });
  });

  it('handles an empty input', () => {
    expect(parseFiletypeFilter('')).toEqual({ searchText: '', filetype: null });
  });
});
