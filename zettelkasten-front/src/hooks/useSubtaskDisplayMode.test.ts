import { renderHook, act } from '@testing-library/react';
import { useSubtaskDisplayMode, STORAGE_KEY } from './useSubtaskDisplayMode';

describe('useSubtaskDisplayMode', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('should default to nested mode', () => {
    const { result } = renderHook(() => useSubtaskDisplayMode());
    expect(result.current.subtaskMode).toBe('nested');
  });

  it('should persist mode to localStorage', () => {
    const { result } = renderHook(() => useSubtaskDisplayMode());

    act(() => {
      result.current.setSubtaskMode('flat');
    });

    expect(result.current.subtaskMode).toBe('flat');
    expect(localStorage.getItem(STORAGE_KEY)).toBe('"flat"');
  });

  it('should load saved mode from localStorage', () => {
    localStorage.setItem(STORAGE_KEY, '"hidden"');

    const { result } = renderHook(() => useSubtaskDisplayMode());
    expect(result.current.subtaskMode).toBe('hidden');
  });

  it('should default to nested when localStorage contains invalid mode', () => {
    localStorage.setItem(STORAGE_KEY, '"invalid"');
    const { result } = renderHook(() => useSubtaskDisplayMode());
    expect(result.current.subtaskMode).toBe('nested');
  });
});
