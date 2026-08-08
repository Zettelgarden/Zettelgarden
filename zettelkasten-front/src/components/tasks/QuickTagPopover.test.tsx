// @vitest-environment happy-dom

import { describe, it, expect } from 'vitest';
import {
  applyQuickTagSelection,
  filterAndSortTagNames,
  getQuickTagTrigger,
} from './QuickTagPopover';

describe('getQuickTagTrigger', () => {
  it("triggers on '#' at start", () => {
    expect(getQuickTagTrigger('#', 1)).toEqual({ start: 0, end: 1, query: '' });
    expect(getQuickTagTrigger('#pro', 4)).toEqual({
      start: 0,
      end: 4,
      query: 'pro',
    });
  });

  it("triggers on '#' after whitespace", () => {
    expect(getQuickTagTrigger('do #', 4)).toEqual({
      start: 3,
      end: 4,
      query: '',
    });
    expect(getQuickTagTrigger('do #Pro', 7)).toEqual({
      start: 3,
      end: 7,
      query: 'Pro',
    });
  });

  it("does not trigger when '#' is not word-start", () => {
    expect(getQuickTagTrigger('abc#', 4)).toBeNull();
    expect(getQuickTagTrigger('abc#tag', 7)).toBeNull();
  });

  it("does not trigger on '##'", () => {
    expect(getQuickTagTrigger('##', 2)).toBeNull();
    expect(getQuickTagTrigger(' ##', 3)).toBeNull();
  });

  it('supports hyphens in query', () => {
    expect(getQuickTagTrigger('#project-alpha', 14)).toEqual({
      start: 0,
      end: 14,
      query: 'project-alpha',
    });
  });
});

describe('filterAndSortTagNames', () => {
  const tags = [
    { name: 'project-alpha' },
    { name: 'project' },
    { name: 'alpha' },
    { name: 'Work' },
  ];

  it('filters case-insensitively with includes()', () => {
    expect(filterAndSortTagNames(tags, 'ALP')).toEqual([
      'alpha',
      'project-alpha',
    ]);
  });

  it('orders prefix matches first, then alphabetical', () => {
    expect(filterAndSortTagNames(tags, 'pro')).toEqual([
      'project',
      'project-alpha',
    ]);
  });
});

describe('applyQuickTagSelection', () => {
  it('replaces the typed token and adds trailing space', () => {
    const trigger = getQuickTagTrigger('do #pro', 7);
    expect(trigger).not.toBeNull();

    const res = applyQuickTagSelection({
      title: 'do #pro',
      trigger: trigger!,
      selectedTagName: 'project-alpha',
    });

    expect(res.didInsert).toBe(true);
    expect(res.nextTitle).toBe('do #project-alpha ');
    expect(res.nextCursor).toBe('do #project-alpha '.length);
  });

  it('prevents duplicates', () => {
    const trigger = getQuickTagTrigger('do #', 4);
    expect(trigger).not.toBeNull();

    const res = applyQuickTagSelection({
      title: 'do #work',
      trigger: trigger!,
      selectedTagName: 'work',
    });

    expect(res.didInsert).toBe(false);
    expect(res.nextTitle).toBe('do #work');
  });
});
