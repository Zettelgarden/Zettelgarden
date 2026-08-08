import { describe, expect, it } from 'vitest';
import { getMissingRequiredFields, isEmptyValue } from './schemaValidation';
import { FieldDefinition } from '../models/Schema';

const fields: FieldDefinition[] = [
  { name: 'title', type: 'text', required: true },
  { name: 'tags', type: 'multi-select', required: true, options: ['a', 'b'] },
  { name: 'rating', type: 'number', required: true },
  { name: 'active', type: 'boolean', required: true },
  { name: 'note', type: 'text', required: false },
];

describe('getMissingRequiredFields', () => {
  it('returns no missing fields when all required fields are filled', () => {
    const data = { title: 'x', tags: ['a'], rating: 0, active: false };
    expect(getMissingRequiredFields(fields, data)).toEqual([]);
  });

  it('treats zero numbers and false booleans as filled', () => {
    expect(isEmptyValue(0)).toBe(false);
    expect(isEmptyValue(false)).toBe(false);
  });

  it('flags missing keys', () => {
    const data = { title: 'x', tags: ['a'], rating: 1 };
    expect(getMissingRequiredFields(fields, data)).toEqual(['active']);
  });

  it('flags empty strings, whitespace-only strings, null and empty arrays', () => {
    expect(
      getMissingRequiredFields(fields, {
        title: '',
        tags: ['a'],
        rating: 1,
        active: true,
      }),
    ).toEqual(['title']);
    expect(
      getMissingRequiredFields(fields, {
        title: '   ',
        tags: ['a'],
        rating: 1,
        active: true,
      }),
    ).toEqual(['title']);
    expect(
      getMissingRequiredFields(fields, {
        title: null,
        tags: ['a'],
        rating: 1,
        active: true,
      }),
    ).toEqual(['title']);
    expect(
      getMissingRequiredFields(fields, {
        title: 'x',
        tags: [],
        rating: 1,
        active: true,
      }),
    ).toEqual(['tags']);
  });

  it('handles null/undefined data', () => {
    expect(getMissingRequiredFields(fields, null)).toEqual([
      'title',
      'tags',
      'rating',
      'active',
    ]);
    expect(getMissingRequiredFields(fields, undefined)).toEqual([
      'title',
      'tags',
      'rating',
      'active',
    ]);
  });

  it('ignores optional fields', () => {
    const data = { title: 'x', tags: ['a'], rating: 1, active: true };
    expect(getMissingRequiredFields(fields, data)).toEqual([]);
  });
});
