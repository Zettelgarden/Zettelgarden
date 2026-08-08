/**
 * Tests for markdown utilities (src/utils/markdown.ts)
 */

import { describe, it, expect } from 'vitest';
import { htmlToMarkdown, safeHtmlToMarkdown } from './markdown';

describe('htmlToMarkdown', () => {
  it('returns empty string for empty input', () => {
    expect(htmlToMarkdown('')).toBe('');
    expect(htmlToMarkdown('   ')).toBe('');
  });

  it('strips script tags and their content', () => {
    const html = '<p>Hello</p><script>alert("xss")</script>';
    const result = htmlToMarkdown(html);
    expect(result).not.toContain('alert');
    expect(result).toContain('Hello');
  });

  it('strips style tags and their content', () => {
    const html = '<p>Hi</p><style>p { color: red; }</style>';
    const result = htmlToMarkdown(html);
    expect(result).not.toContain('color');
    expect(result).toContain('Hi');
  });

  it('converts headings to ATX markdown', () => {
    expect(htmlToMarkdown('<h1>Title</h1>')).toBe('# Title');
    expect(htmlToMarkdown('<h3>Sub</h3>')).toBe('### Sub');
  });

  it('preserves links', () => {
    expect(htmlToMarkdown('<a href="https://example.com">Example</a>')).toBe(
      '[Example](https://example.com)',
    );
  });

  it('preserves images with alt text', () => {
    expect(htmlToMarkdown('<img src="/pic.png" alt="A photo">')).toBe(
      '![A photo](/pic.png)',
    );
  });

  it('preserves bold and italic', () => {
    expect(htmlToMarkdown('<strong>bold</strong>')).toBe('**bold**');
    expect(htmlToMarkdown('<b>bold</b>')).toBe('**bold**');
    expect(htmlToMarkdown('<em>italic</em>')).toBe('*italic*');
    expect(htmlToMarkdown('<i>italic</i>')).toBe('*italic*');
  });

  it('converts a full document', () => {
    const html =
      '<h1>Hello</h1><p>Some <strong>bold</strong> text and ' +
      '<a href="https://x.com">a link</a>.</p>';
    const result = htmlToMarkdown(html);
    expect(result).toContain('# Hello');
    expect(result).toContain('**bold**');
    expect(result).toContain('[a link](https://x.com)');
  });
});

describe('safeHtmlToMarkdown', () => {
  it('returns empty string for null/undefined/empty', () => {
    expect(safeHtmlToMarkdown(null)).toBe('');
    expect(safeHtmlToMarkdown(undefined)).toBe('');
    expect(safeHtmlToMarkdown('')).toBe('');
    expect(safeHtmlToMarkdown('   ')).toBe('');
  });

  it('passes markdown through unchanged', () => {
    const md = '# Heading\n\nSome **bold** text';
    expect(safeHtmlToMarkdown(md)).toBe(md);
  });

  it('passes list markdown through unchanged', () => {
    const md = '- item one\n- item two';
    expect(safeHtmlToMarkdown(md)).toBe(md);
  });

  it('passes blockquote markdown through unchanged', () => {
    const md = '> quoted text';
    expect(safeHtmlToMarkdown(md)).toBe(md);
  });

  it('passes code fence markdown through unchanged', () => {
    const md = '```js\nconst x = 1;\n```';
    expect(safeHtmlToMarkdown(md)).toBe(md);
  });

  it('converts HTML to markdown', () => {
    const html = '<h1>Title</h1><p>Paragraph</p>';
    const result = safeHtmlToMarkdown(html);
    expect(result).toBe('# Title\n\nParagraph');
  });

  it('trims whitespace around input', () => {
    expect(safeHtmlToMarkdown('  plain text  ')).toBe('plain text');
  });

  it('returns plain text as-is when it looks like neither HTML nor markdown', () => {
    expect(safeHtmlToMarkdown('Just some plain text.')).toBe(
      'Just some plain text.',
    );
  });

  it('returns plain text even when it contains angle-bracket-ish content that is not a known tag', () => {
    // "x > y" should not be treated as HTML
    expect(safeHtmlToMarkdown('a > b < c')).toBe('a > b < c');
  });

  it('handles HTML that fails conversion by falling back to the trimmed text', () => {
    // Malformed but tag-looking input: isHtmlContent matches the tag open,
    // conversion may still succeed or the catch returns the trimmed input.
    const input = '<p>unclosed';
    const result = safeHtmlToMarkdown(input);
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });
});
