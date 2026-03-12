import { useState, useRef, useCallback, ChangeEvent } from 'react';
import {
  type QuickTagTrigger,
  getQuickTagTrigger,
  applyQuickTagSelection,
} from '../components/tasks/QuickTagPopover';

interface UseFilterInputResult {
  /** Ref to the filter input element */
  filterInputRef: React.RefObject<HTMLInputElement>;
  /** Current cursor position in the filter input */
  cursorPosition: number;
  /** Current quick tag trigger state (null if no trigger active) */
  filterTrigger: QuickTagTrigger | null;
  /** Whether the filter input is currently focused */
  isFilterFocused: boolean;
  /** Set whether the filter input is focused */
  setIsFilterFocused: (focused: boolean) => void;
  /** Handler for filter input change events */
  handleFilterChange: (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  /** Handler for quick tag selection */
  handleSelectQuickTag: (selectedTagName: string) => void;
  /** Refresh the filter trigger state from the input element */
  refreshFilterTriggerFromInput: (input: HTMLInputElement) => void;
  /** Set the filter trigger state directly */
  setFilterTrigger: (trigger: QuickTagTrigger | null) => void;
}

interface UseFilterInputOptions {
  /** The current filter string value */
  filterString: string;
  /** Function to update the filter string */
  setFilterString: (value: string) => void;
}

/**
 * Custom hook to manage filter input state and quick tag autocomplete.
 * 
 * This hook encapsulates all the logic for handling filter input with
 * quick tag autocomplete functionality, including cursor position tracking
 * and tag selection.
 * 
 * @example
 * ```tsx
 * const {
 *   filterInputRef,
 *   cursorPosition,
 *   filterTrigger,
 *   isFilterFocused,
 *   setIsFilterFocused,
 *   handleFilterChange,
 *   handleSelectQuickTag,
 *   refreshFilterTriggerFromInput,
 *   setFilterTrigger,
 * } = useFilterInput({
 *   filterString,
 *   setFilterString,
 * });
 * ```
 */
export function useFilterInput({
  filterString,
  setFilterString,
}: UseFilterInputOptions): UseFilterInputResult {
  const filterInputRef = useRef<HTMLInputElement>(null);
  const [cursorPosition, setCursorPosition] = useState(0);
  const [filterTrigger, setFilterTrigger] = useState<QuickTagTrigger | null>(null);
  const [isFilterFocused, setIsFilterFocused] = useState(false);

  const handleFilterChange = useCallback(
    (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      const nextValue = e.target.value;
      const nextCursor = (e.target as HTMLInputElement).selectionStart ?? nextValue.length;

      setCursorPosition(nextCursor);
      setFilterTrigger(getQuickTagTrigger(nextValue, nextCursor));
      setFilterString(nextValue);
    },
    [setFilterString]
  );

  const refreshFilterTriggerFromInput = useCallback((input: HTMLInputElement) => {
    const cursor = input.selectionStart ?? 0;
    setCursorPosition(cursor);
    setFilterTrigger(getQuickTagTrigger(input.value, cursor));
  }, []);

  const handleSelectQuickTag = useCallback(
    (selectedTagName: string) => {
      if (!filterTrigger) return;

      const res = applyQuickTagSelection({
        title: filterString,
        trigger: filterTrigger,
        selectedTagName,
      });

      setCursorPosition(res.nextCursor);

      if (!res.didInsert) {
        setFilterTrigger(null);
        return;
      }

      setFilterString(res.nextTitle);
      setFilterTrigger(null);

      // Restore focus + cursor after React updates the controlled input.
      requestAnimationFrame(() => {
        const input = filterInputRef.current;
        if (!input) return;
        input.focus();
        input.setSelectionRange(res.nextCursor, res.nextCursor);
      });
    },
    [filterString, filterTrigger, setFilterString]
  );

  return {
    filterInputRef,
    cursorPosition,
    filterTrigger,
    isFilterFocused,
    setIsFilterFocused,
    handleFilterChange,
    handleSelectQuickTag,
    refreshFilterTriggerFromInput,
    setFilterTrigger,
  };
}
