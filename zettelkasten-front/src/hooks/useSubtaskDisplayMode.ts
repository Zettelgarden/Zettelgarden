import { useState, useEffect } from 'react';

export type SubtaskDisplayMode = 'nested' | 'flat' | 'hidden';

const VALID_MODES: SubtaskDisplayMode[] = ['nested', 'flat', 'hidden'];
const DEFAULT_MODE: SubtaskDisplayMode = 'nested';

export const STORAGE_KEY = 'subtaskDisplayMode';

export function useSubtaskDisplayMode(): {
  subtaskMode: SubtaskDisplayMode;
  setSubtaskMode: (mode: SubtaskDisplayMode) => void;
} {
  const [subtaskMode, setSubtaskModeState] = useState<SubtaskDisplayMode>(
    () => {
      try {
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved) {
          const parsed = JSON.parse(saved);
          if (VALID_MODES.includes(parsed)) {
            return parsed as SubtaskDisplayMode;
          }
        }
      } catch (e) {
        console.error('Failed to load subtask display mode:', e);
      }
      return DEFAULT_MODE;
    },
  );

  const setSubtaskMode = (mode: SubtaskDisplayMode): void => {
    setSubtaskModeState(mode);
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(mode));
    } catch (e) {
      console.error('Failed to save subtask display mode:', e);
    }
  };

  return { subtaskMode, setSubtaskMode };
}
