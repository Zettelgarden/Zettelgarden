import { describe, it, expect } from 'vitest';
import { formatCsvValue, schemaCardsToCsv } from './schemaCsv';
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
