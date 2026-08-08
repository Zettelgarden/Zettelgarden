import { describe, it, expect, beforeEach } from 'vitest';
import { setDocumentTitle } from './title';

describe('setDocumentTitle', () => {
  beforeEach(() => {
    document.title = '';
  });

  it('sets base title when no page title provided', () => {
    setDocumentTitle();
    expect(document.title).toMatch(/^Zettelgarden/);
  });

  it('sets page title with base when page title provided', () => {
    setDocumentTitle('Search');
    expect(document.title).toContain('Search -');
    expect(document.title).toContain('Zettelgarden');
  });

  it('handles different page titles', () => {
    setDocumentTitle('Edit Card');
    expect(document.title).toContain('Edit Card -');
    expect(document.title).toContain('Zettelgarden');

    setDocumentTitle('Dashboard');
    expect(document.title).toContain('Dashboard -');
    expect(document.title).toContain('Zettelgarden');
  });
});
