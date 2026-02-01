import { describe, it, expect } from 'vitest';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkSpreadsheet from './remark-spreadsheet';
import { visit } from 'unist-util-visit';

describe('remark-spreadsheet', () => {
  it('parses {{spreadsheet:name}} syntax', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet);

    const tree = processor.parse('Check out {{spreadsheet:budget}} for details');
    processor.runSync(tree);

    let foundSpreadsheet = false;
    let spreadsheetName = '';

    visit(tree, 'spreadsheet', (node: any) => {
      foundSpreadsheet = true;
      spreadsheetName = node.data?.name || '';
    });

    expect(foundSpreadsheet).toBe(true);
    expect(spreadsheetName).toBe('budget');
  });

  it('handles default name without colon', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet);

    const tree = processor.parse('Inline {{spreadsheet}} here');
    processor.runSync(tree);

    let foundSpreadsheet = false;
    let spreadsheetName = '';

    visit(tree, 'spreadsheet', (node: any) => {
      foundSpreadsheet = true;
      spreadsheetName = node.data?.name || '';
    });

    expect(foundSpreadsheet).toBe(true);
    expect(spreadsheetName).toBe('sheet1'); // Default name
  });

  it('parses multiple spreadsheets', () => {
    const processor = unified()
      .use(remarkParse)
      .use(remarkSpreadsheet);

    const tree = processor.parse('{{spreadsheet:sales}} and {{spreadsheet:expenses}}');
    processor.runSync(tree);

    const spreadsheetNames: string[] = [];

    visit(tree, 'spreadsheet', (node: any) => {
      spreadsheetNames.push(node.data?.name || '');
    });

    expect(spreadsheetNames).toHaveLength(2);
    expect(spreadsheetNames).toContain('sales');
    expect(spreadsheetNames).toContain('expenses');
  });
});
