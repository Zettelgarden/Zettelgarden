import { describe, it, expect } from 'vitest';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import remarkSchemaTable from '../src/remark-schema-table';
import {
  parseFilterGroups,
  applyFilterGroupsToCard,
} from '../src/utils/schemaFilters';

// Run the markdown -> schemaTable node pipeline and collect the data each
// schemaTable node carries (schemaRef, columns, filters).
function collectSchemaTables(md: string): Array<Record<string, unknown>> {
  const processor = unified().use(remarkParse).use(remarkSchemaTable);
  const tree = processor.runSync(processor.parse(md));
  const results: Array<Record<string, unknown>> = [];
  const walk = (node: any) => {
    if (node.type === 'schemaTable') {
      results.push(node.data);
    }
    if (node.children) node.children.forEach(walk);
  };
  walk(tree as any);
  return results;
}

describe('remarkSchemaTable', () => {
  it('parses a bare schema ref', () => {
    const tables = collectSchemaTables('{{schema:1}}');
    expect(tables).toHaveLength(1);
    expect(tables[0]).toMatchObject({ schemaRef: '1' });
    expect(tables[0].columns).toBeUndefined();
    expect(tables[0].filters).toBeUndefined();
  });

  it('parses columns and AND filters', () => {
    const tables = collectSchemaTables(
      '{{schema:book-review|columns:title,status|filter:status=active,priority=high}}',
    );
    expect(tables).toHaveLength(1);
    expect(tables[0]).toMatchObject({
      schemaRef: 'book-review',
      columns: 'title,status',
      filters: 'status=active,priority=high',
    });
  });

  it('keeps || OR filters intact (not split into options)', () => {
    const tables = collectSchemaTables(
      '{{schema:1|filter:status=active||status=done}}',
    );
    expect(tables).toHaveLength(1);
    expect(tables[0].filters).toBe('status=active||status=done');
    expect(tables[0].columns).toBeUndefined();
  });

  it('keeps percent-encoded pipes inside filter values', () => {
    const tables = collectSchemaTables('{{schema:1|filter:title=a%7Cb}}');
    expect(tables).toHaveLength(1);
    expect(tables[0].filters).toBe('title=a%7Cb');
    expect(tables[0].columns).toBeUndefined();
  });

  it('keeps a literal || in a value when percent-encoded as %7C%7C', () => {
    const tables = collectSchemaTables(
      '{{schema:1|filter:title=a%7C%7Cb||status=done}}',
    );
    expect(tables).toHaveLength(1);
    expect(tables[0].filters).toBe('title=a%7C%7Cb||status=done');
  });

  it('does not treat percent-encoded pipes as option separators', () => {
    const tables = collectSchemaTables(
      '{{schema:1|columns:title|filter:note=a%7Cb}}',
    );
    expect(tables).toHaveLength(1);
    expect(tables[0]).toMatchObject({
      columns: 'title',
      filters: 'note=a%7Cb',
    });
  });

  it('parses old &SCHEMATABLE: syntax', () => {
    const tables = collectSchemaTables('&SCHEMATABLE:1|filter:x=y&');
    expect(tables).toHaveLength(1);
    expect(tables[0]).toMatchObject({ schemaRef: '1', filters: 'x=y' });
  });
});

describe('end-to-end: markdown directive with special characters', () => {
  // Simulates the real pipeline: markdown body -> remark plugin (which runs
  // AFTER CommonMark parsing) -> filter parser -> card matching.
  it('matches values containing | , = through the full markdown pipeline', () => {
    const md = '{{schema:1|filter:title=a%7Cb,status=x%3Dy||note=a%2Cb}}';
    const tables = collectSchemaTables(md);
    expect(tables).toHaveLength(1);

    const groups = parseFilterGroups(tables[0].filters as string);
    const card = {
      title: 'a|b',
      structured_data: { status: 'x=y', note: 'a,b' },
    };
    expect(applyFilterGroupsToCard(card, groups)).toBe(true);

    const noMatch = {
      title: 'a|b',
      structured_data: { status: 'x=y', note: 'other' },
    };
    expect(applyFilterGroupsToCard(noMatch, groups)).toBe(true); // OR group 1
  });
});
