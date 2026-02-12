import { describe, it, expect } from 'vitest';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkSpreadsheet from './remark-spreadsheet';
import { visit } from 'unist-util-visit';

describe('remark-spreadsheet', () => {
  it('parses {{spreadsheet:123}} syntax with numeric ID', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet);

    const tree = processor.parse('Check out {{spreadsheet:123}} for details');
    processor.runSync(tree);

    let foundSpreadsheet = false;
    let spreadsheetId = '';
    let dataSpreadsheetId = '';

    visit(tree, 'spreadsheet', (node: any) => {
      foundSpreadsheet = true;
      spreadsheetId = node.data?.id || '';
      dataSpreadsheetId = node.data?.hProperties?.['data-spreadsheet-id'] || '';
    });

    expect(foundSpreadsheet).toBe(true);
    expect(spreadsheetId).toBe('123');
    expect(dataSpreadsheetId).toBe('123');
  });

  it('handles empty string for syntax without colon (legacy support)', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet);

    const tree = processor.parse('Inline {{spreadsheet}} here');
    processor.runSync(tree);

    let foundSpreadsheet = false;
    let spreadsheetId = '';
    let dataSpreadsheetId = '';

    visit(tree, 'spreadsheet', (node: any) => {
      foundSpreadsheet = true;
      spreadsheetId = node.data?.id || '';
      dataSpreadsheetId = node.data?.hProperties?.['data-spreadsheet-id'] || '';
    });

    expect(foundSpreadsheet).toBe(true);
    expect(spreadsheetId).toBe(''); // Empty string for legacy support
    expect(dataSpreadsheetId).toBe('');
  });

  it('parses multiple spreadsheets with numeric IDs', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet);

    const tree = processor.parse('{{spreadsheet:456}} and {{spreadsheet:789}}');
    processor.runSync(tree);

    const spreadsheetIds: string[] = [];

    visit(tree, 'spreadsheet', (node: any) => {
      spreadsheetIds.push(node.data?.id || '');
    });

    expect(spreadsheetIds).toHaveLength(2);
    expect(spreadsheetIds).toContain('456');
    expect(spreadsheetIds).toContain('789');
  });

  it('does not match non-numeric values after colon', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet);

    const tree = processor.parse('{{spreadsheet:budget}} should not match');
    processor.runSync(tree);

    let foundSpreadsheet = false;

    visit(tree, 'spreadsheet', (node: any) => {
      foundSpreadsheet = true;
    });

    expect(foundSpreadsheet).toBe(false);
  });
});
