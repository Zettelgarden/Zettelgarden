import { renderHook, act } from '@testing-library/react';
import { useSubtaskDisplayMode } from './useSubtaskDisplayMode';

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
    expect(localStorage.getItem('subtaskDisplayMode')).toBe('"flat"');
  });

  it('should load saved mode from localStorage', () => {
    localStorage.setItem('subtaskDisplayMode', '"hidden"');

    const { result } = renderHook(() => useSubtaskDisplayMode());
    expect(result.current.subtaskMode).toBe('hidden');
  });
});
