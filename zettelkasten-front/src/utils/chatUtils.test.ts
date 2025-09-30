import { describe, it, expect } from 'vitest';
import { parseCardReferences } from './chatUtils';

describe('parseCardReferences', () => {
  it('parses single card reference', () => {
    const text = 'Check out [Card: 123 | My Card Title] for more info';
    const refs = parseCardReferences(text);

    expect(refs).toHaveLength(1);
    expect(refs[0].cardId).toBe('123');
    expect(refs[0].title).toBe('My Card Title');
    expect(refs[0].fullMatch).toBe('[Card: 123 | My Card Title]');
    expect(refs[0].startIndex).toBe(10);
  });

  it('parses multiple card references', () => {
    const text = 'See [Card: 1 | First] and [Card: 2 | Second]';
    const refs = parseCardReferences(text);

    expect(refs).toHaveLength(2);
    expect(refs[0].cardId).toBe('1');
    expect(refs[1].cardId).toBe('2');
  });

  it('handles extra whitespace in references', () => {
    const text = '[Card:  456  |  Spaced Title  ]';
    const refs = parseCardReferences(text);

    expect(refs).toHaveLength(1);
    expect(refs[0].cardId).toBe('456');
    expect(refs[0].title).toBe('Spaced Title');
  });

  it('returns empty array when no references found', () => {
    const text = 'Just plain text with no references';
    const refs = parseCardReferences(text);

    expect(refs).toHaveLength(0);
  });

  it('handles card IDs with special characters', () => {
    const text = '[Card: 10/A.2/B | Complex ID]';
    const refs = parseCardReferences(text);

    expect(refs).toHaveLength(1);
    expect(refs[0].cardId).toBe('10/A.2/B');
  });
});