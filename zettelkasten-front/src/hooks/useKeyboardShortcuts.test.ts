import { renderHook } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useKeyboardShortcuts } from './useKeyboardShortcuts';

describe('useKeyboardShortcuts', () => {
  let addEventListenerSpy: ReturnType<typeof vi.spyOn>;
  let removeEventListenerSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    addEventListenerSpy = vi.spyOn(document, 'addEventListener');
    removeEventListenerSpy = vi.spyOn(document, 'removeEventListener');
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('should add keydown event listener on mount', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    expect(addEventListenerSpy).toHaveBeenCalledWith(
      'keydown',
      expect.any(Function),
    );
  });

  it('should remove keydown event listener on unmount', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    const { unmount } = renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    unmount();

    expect(removeEventListenerSpy).toHaveBeenCalledWith(
      'keydown',
      expect.any(Function),
    );
  });

  it('should call onCreateTask when "t" key is pressed', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    const event = new KeyboardEvent('keydown', { key: 't' });
    document.dispatchEvent(event);

    expect(onCreateTask).toHaveBeenCalled();
  });

  it('should call onQuickSearch when "s" key is pressed', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    const event = new KeyboardEvent('keydown', { key: 's' });
    document.dispatchEvent(event);

    expect(onQuickSearch).toHaveBeenCalled();
  });

  it('should prevent default behavior when shortcut key is pressed', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    const event = new KeyboardEvent('keydown', { key: 't' });
    const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
    document.dispatchEvent(event);

    expect(preventDefaultSpy).toHaveBeenCalled();
  });

  it('should ignore shortcuts when metaKey is pressed', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    const event = new KeyboardEvent('keydown', { key: 't', metaKey: true });
    document.dispatchEvent(event);

    expect(onCreateTask).not.toHaveBeenCalled();
  });

  it('should ignore shortcuts when input element is focused', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    const input = document.createElement('input');
    document.body.appendChild(input);
    input.focus();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    const event = new KeyboardEvent('keydown', { key: 't' });
    document.dispatchEvent(event);

    expect(onCreateTask).not.toHaveBeenCalled();

    document.body.removeChild(input);
  });

  it('should ignore shortcuts when textarea element is focused', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    const textarea = document.createElement('textarea');
    document.body.appendChild(textarea);
    textarea.focus();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    const event = new KeyboardEvent('keydown', { key: 's' });
    document.dispatchEvent(event);

    expect(onQuickSearch).not.toHaveBeenCalled();

    document.body.removeChild(textarea);
  });

  it('should trigger shortcuts when non-input element is focused', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    const div = document.createElement('div');
    document.body.appendChild(div);
    div.focus();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    const event = new KeyboardEvent('keydown', { key: 't' });
    document.dispatchEvent(event);

    expect(onCreateTask).toHaveBeenCalled();

    document.body.removeChild(div);
  });

  it('should ignore unknown keys', () => {
    const onCreateTask = vi.fn();
    const onQuickSearch = vi.fn();

    renderHook(() =>
      useKeyboardShortcuts({
        onCreateTask,
        onQuickSearch,
      }),
    );

    const event = new KeyboardEvent('keydown', { key: 'x' });
    document.dispatchEvent(event);

    expect(onCreateTask).not.toHaveBeenCalled();
    expect(onQuickSearch).not.toHaveBeenCalled();
  });

  it('should update callbacks when they change', () => {
    const onCreateTask1 = vi.fn();
    const onQuickSearch1 = vi.fn();

    const { rerender } = renderHook(
      ({ onCreateTask, onQuickSearch }) =>
        useKeyboardShortcuts({
          onCreateTask,
          onQuickSearch,
        }),
      {
        initialProps: {
          onCreateTask: onCreateTask1,
          onQuickSearch: onQuickSearch1,
        },
      },
    );

    const onCreateTask2 = vi.fn();
    const onQuickSearch2 = vi.fn();

    rerender({
      onCreateTask: onCreateTask2,
      onQuickSearch: onQuickSearch2,
    });

    const event = new KeyboardEvent('keydown', { key: 't' });
    document.dispatchEvent(event);

    expect(onCreateTask1).not.toHaveBeenCalled();
    expect(onCreateTask2).toHaveBeenCalled();
  });
});
