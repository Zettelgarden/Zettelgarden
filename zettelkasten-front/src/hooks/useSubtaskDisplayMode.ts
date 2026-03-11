import { useState, useEffect } from 'react';

export type SubtaskDisplayMode = 'nested' | 'flat' | 'hidden';

const STORAGE_KEY = 'subtaskDisplayMode';

export function useSubtaskDisplayMode() {
  const [subtaskMode, setSubtaskModeState] = useState<SubtaskDisplayMode>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        return JSON.parse(saved) as SubtaskDisplayMode;
      }
    } catch (e) {
      console.error('Failed to load subtask display mode:', e);
    }
    return 'nested';
  });

  const setSubtaskMode = (mode: SubtaskDisplayMode) => {
    setSubtaskModeState(mode);
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(mode));
    } catch (e) {
      console.error('Failed to save subtask display mode:', e);
    }
  };

  return { subtaskMode, setSubtaskMode };
}
