import { describe, it, expect, vi } from 'vitest';
import { renderHook } from '@testing-library/react';
import { TagProvider, useTagContext } from './TagContext';
import { Tag } from '../models/Tags';

describe('TagContext', () => {
  const mockTags: Tag[] = [
    { id: 1, name: 'Work', color: '#3b82f6', user_id: 1 },
    { id: 2, name: 'Personal', color: '#10b981', user_id: 1 },
  ];

  it('provides tags from test data when in testing mode', () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <TagProvider testing={true} testTags={mockTags}>
        {children}
      </TagProvider>
    );

    const { result } = renderHook(() => useTagContext(), { wrapper });

    expect(result.current.tags).toEqual(mockTags);
  });

  it('provides empty tags array when no test tags provided', () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <TagProvider testing={true} testTags={[]}>
        {children}
      </TagProvider>
    );

    const { result } = renderHook(() => useTagContext(), { wrapper });

    expect(result.current.tags).toEqual([]);
  });

  it('provides setRefreshTags function', () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <TagProvider testing={true} testTags={mockTags}>
        {children}
      </TagProvider>
    );

    const { result } = renderHook(() => useTagContext(), { wrapper });

    expect(result.current.setRefreshTags).toBeDefined();
    expect(typeof result.current.setRefreshTags).toBe('function');
  });

  it('throws error when used outside TagProvider', () => {
    expect(() => {
      renderHook(() => useTagContext());
    }).toThrow('useTagContext must be used wtihin a TagProvider');
  });
});
