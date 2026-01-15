import { useEffect, useCallback } from 'react';

interface KeyboardShortcutCallbacks {
  onCreateTask: () => void;
  onQuickSearch: () => void;
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
}: KeyboardShortcutCallbacks): void {
  const handleKeyPress = useCallback(
    (event: KeyboardEvent) => {
      // Ignore if user is holding meta key (system shortcuts)
      if (event.metaKey) {
        return;
      }

      // Only trigger shortcuts when not focusing input/textarea elements
      const focusedElement = document.activeElement;
      if (!focusedElement || !focusedElement.tagName.match(/^INPUT|TEXTAREA$/i)) {
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
    [onCreateTask, onQuickSearch],
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyPress);
    return () => {
      document.removeEventListener('keydown', handleKeyPress);
    };
  }, [handleKeyPress]);
}
