import { describe, it, expect, vi, afterEach } from 'vitest';
import { formatCsvValue, schemaCardsToCsv, downloadCsv } from './schemaCsv';
import { Card } from '../models/Card';
import { FieldDefinition } from '../models/Schema';

function makeCard(overrides: Partial<Card>): Card {
  return {
    id: 1,
    card_id: '20240101120000',
    user_id: 1,
    title: 'A title',
    body: '',
    link: '',
    is_deleted: false,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-02'),
    parent_id: -1,
    parent: {
      id: -1,
      card_id: '',
      user_id: -1,
      title: '',
      parent_id: -1,
      created_at: new Date(0),
      updated_at: new Date(0),
      tags: [],
    },
    files: [],
    children: [],
    references: [],
    tags: [],
    tasks: [],
    entities: [],
    ...overrides,
  };
}

const fields: FieldDefinition[] = [
  { name: 'Author', type: 'text', required: true },
  { name: 'Rating', type: 'number', required: false },
  { name: 'Tags', type: 'multi-select', required: false },
];

describe('formatCsvValue', () => {
  it('renders plain values without quoting', () => {
    expect(formatCsvValue('hello')).toBe('hello');
    expect(formatCsvValue(5)).toBe('5');
    expect(formatCsvValue(true)).toBe('true');
  });

  it('renders null/undefined as empty cells', () => {
    expect(formatCsvValue(null)).toBe('');
    expect(formatCsvValue(undefined)).toBe('');
    expect(formatCsvValue('')).toBe('');
  });

  it('joins arrays for multi-select values (quoted when they contain commas)', () => {
    expect(formatCsvValue(['a', 'b'])).toBe('"a, b"');
    expect(formatCsvValue(['a'])).toBe('a');
  });

  it('neutralizes spreadsheet formula injection (OWASP CSV Injection)', () => {
    // Contains quotes, so RFC 4180 quoting wraps it and doubles the quotes
    // on top of the apostrophe guard.
    expect(formatCsvValue('=HYPERLINK("http://evil","x")')).toBe(
      '"\'=HYPERLINK(""http://evil"",""x"")"',
    );
    expect(formatCsvValue('+SUM(1,2)')).toBe('"\'+SUM(1,2)"');
    expect(formatCsvValue('-1+1')).toBe("'-1+1");
    expect(formatCsvValue('@cmd')).toBe("'@cmd");
    expect(formatCsvValue('\t=2')).toBe("'\t=2");
    // Innocent values are untouched
    expect(formatCsvValue('normal')).toBe('normal');
    expect(formatCsvValue('5')).toBe('5');
    // The guard composes with quoting (comma + formula prefix)
    expect(formatCsvValue('=1,2')).toBe('"\'=1,2"');
  });

  it('quotes values containing commas, quotes, or newlines', () => {
    expect(formatCsvValue('a,b')).toBe('"a,b"');
    expect(formatCsvValue('say "hi"')).toBe('"say ""hi"""');
    expect(formatCsvValue('line1\nline2')).toBe('"line1\nline2"');
  });
});

describe('schemaCardsToCsv', () => {
  it('produces a header of card_id, title, then fields in order', () => {
    const csv = schemaCardsToCsv([], fields);
    const lines = csv.split('\n');
    expect(lines).toHaveLength(1);
    expect(lines[0]).toBe('card_id,title,Author,Rating,Tags');
  });

  it('serializes structured field values per card', () => {
    const cards = [
      makeCard({
        card_id: '20240101120000',
        title: 'Dune',
        structured_data: {
          Author: 'Frank Herbert',
          Rating: 5,
          Tags: ['sci-fi', 'classic'],
        },
      }),
    ];
    const csv = schemaCardsToCsv(cards, fields);
    const lines = csv.split('\n');
    expect(lines[0]).toBe('card_id,title,Author,Rating,Tags');
    expect(lines[1]).toBe(
      '20240101120000,Dune,Frank Herbert,5,"sci-fi, classic"',
    );
  });

  it('escapes quotes and commas inside titles and values', () => {
    const cards = [
      makeCard({
        card_id: '20240101120000',
        title: 'Quoted, "title"',
        structured_data: { Author: 'Doe, Jane' },
      }),
    ];
    const csv = schemaCardsToCsv(cards, fields);
    const lines = csv.split('\n');
    expect(lines[1]).toBe('20240101120000,"Quoted, ""title""","Doe, Jane",,');
  });

  it('exports empty structured fields as empty cells', () => {
    const cards = [
      makeCard({
        card_id: '20240101120000',
        title: 'No data yet',
        structured_data: {},
      }),
    ];
    const csv = schemaCardsToCsv(cards, fields);
    const lines = csv.split('\n');
    expect(lines[1]).toBe('20240101120000,No data yet,,,');
  });

  it('only exports the cards passed in (caller applies filters)', () => {
    const cards = [
      makeCard({ card_id: '1', title: 'Keep' }),
      makeCard({ card_id: '2', title: 'Filtered out' }),
    ];
    const csv = schemaCardsToCsv([cards[0]], fields);
    expect(csv.split('\n')).toHaveLength(2);
    expect(csv).toContain('Keep');
    expect(csv).not.toContain('Filtered out');
  });
});

describe('downloadCsv', () => {
  const originalCreateObjectURL = URL.createObjectURL;
  const originalRevokeObjectURL = URL.revokeObjectURL;
  const clickSpy = vi.fn();

  afterEach(() => {
    vi.restoreAllMocks();
    URL.createObjectURL = originalCreateObjectURL;
    URL.revokeObjectURL = originalRevokeObjectURL;
  });

  it('creates a BOM-prefixed blob download with the given filename', () => {
    let capturedBlob: Blob | null = null;
    URL.createObjectURL = vi.fn((blob: Blob) => {
      capturedBlob = blob;
      return 'blob:mock';
    });
    URL.revokeObjectURL = vi.fn();
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(clickSpy);

    downloadCsv('cards.csv', 'card_id,title\n1,Dune');

    expect(URL.createObjectURL).toHaveBeenCalledTimes(1);
    expect(capturedBlob).not.toBeNull();
    // The blob carries a UTF-8 BOM so Excel decodes non-ASCII correctly.
    expect(capturedBlob!.text()).resolves.toBe('\uFEFFcard_id,title\n1,Dune');
    expect(clickSpy).toHaveBeenCalledTimes(1);
  });

  it('deferred revoke of the object URL', () => {
    URL.createObjectURL = vi.fn(() => 'blob:mock');
    const revoke = vi.fn();
    URL.revokeObjectURL = revoke;
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});

    downloadCsv('cards.csv', 'a\n1');

    // Not revoked synchronously; freed on a later tick.
    expect(revoke).not.toHaveBeenCalled();
    vi.waitFor(() => expect(revoke).toHaveBeenCalledWith('blob:mock'));
  });
});
