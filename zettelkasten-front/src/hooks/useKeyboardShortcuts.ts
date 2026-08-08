import { useEffect, useCallback } from 'react';

interface KeyboardShortcutCallbacks {
  onCreateTask: () => void;
  onQuickSearch: () => void;
  /** Optional: toggle the card view's right info rail. Fired on Cmd/Ctrl-\. */
  onToggleRightPane?: () => void;
}

/**
 * Custom hook to handle global keyboard shortcuts
 * Supports:
 * - 't' key: create task
 * - 's' key: quick search
 *
 * The hook prevents shortcuts from triggering when input/textarea elements are focused.
 * Meta key combinations are ignored (system shortcuts).
 */
export function useKeyboardShortcuts({
  onCreateTask,
  onQuickSearch,
  onToggleRightPane,
}: KeyboardShortcutCallbacks): void {
  const handleKeyPress = useCallback(
    (event: KeyboardEvent) => {
      // Cmd/Ctrl-\ toggles the card view's right info rail (Obsidian-style).
      // Handled before the meta-ignore below since it requires a modifier.
      if ((event.metaKey || event.ctrlKey) && event.key === '\\') {
        event.preventDefault();
        onToggleRightPane?.();
        return;
      }

      // Ignore if user is holding meta key (system shortcuts)
      if (event.metaKey) {
        return;
      }

      // Only trigger shortcuts when not focusing input/textarea elements
      const focusedElement = document.activeElement;
      if (
        !focusedElement ||
        !focusedElement.tagName.match(/^INPUT|TEXTAREA$/i)
      ) {
        if (event.key === 't') {
          event.preventDefault();
          onCreateTask();
        }
        if (event.key === 's') {
          event.preventDefault();
          onQuickSearch();
        }
      }
    },
    [onCreateTask, onQuickSearch, onToggleRightPane],
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyPress);
    return () => {
      document.removeEventListener('keydown', handleKeyPress);
    };
  }, [handleKeyPress]);
}
