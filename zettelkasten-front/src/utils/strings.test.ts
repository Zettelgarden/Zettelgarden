import { it, expect, test, describe } from 'vitest';
import { findWordBoundaries } from './strings';

describe('findWordBoundaries', () => {
  it('should find boundaries when index is inside a word', () => {
    const input = 'hello world';
    const index = 6; // Inside "world"
    expect(findWordBoundaries(input, index)).toEqual({ start: 6, end: 11 });
  });

  it('should find boundaries when index is at the start of a word', () => {
    const input = 'hello world';
    const index = 0; // Start of "hello"
    expect(findWordBoundaries(input, index)).toEqual({ start: 0, end: 6 });
  });

  it('should find boundaries when index is at the end of a word', () => {
    const input = 'hello world';
    const index = 4; // End of "hello", character 'o'
    expect(findWordBoundaries(input, index)).toEqual({ start: 0, end: 6 });
  });

  it('should find boundaries for a word surrounded by newlines', () => {
    const input = 'hello\nworld\nagain';
    const index = 7; // Inside "world"
    expect(findWordBoundaries(input, index)).toEqual({ start: 6, end: 12 });
  });

  it('should throw an error if index is out of range (negative)', () => {
    const input = 'hello world';
    expect(() => findWordBoundaries(input, -1)).toThrow(
      'Index is out of bounds',
    );
  });

  it('should throw an error if index is out of range (exceeding length)', () => {
    const input = 'hello world';
    expect(() => findWordBoundaries(input, input.length)).toThrow(
      'Index is out of bounds',
    );
  });

  it('should return correct bounds when the word is a single letter', () => {
    const input = 'a b c';
    const index = 2; // On 'b'
    expect(findWordBoundaries(input, index)).toEqual({ start: 2, end: 4 });
  });
});
