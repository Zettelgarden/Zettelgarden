import { describe, it, expect, beforeEach } from 'vitest';
import { render, renderHook, act } from '@testing-library/react';
import { UIStateProvider, useUIState } from './UIStateContext';
import { sampleCards } from '../tests/data';

const RIGHT_PANE_KEY = 'zettelgarden-right-pane-open';
const PINNED_CARD_KEY = 'zettelgarden-pinned-card';

function wrapper({ children }: { children: React.ReactNode }) {
  return <UIStateProvider>{children}</UIStateProvider>;
}

describe('UIStateContext — right pane', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('defaults the right pane to open', () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneOpen).toBe(true);
  });

  it('toggleRightPane flips the state and persists it', () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneOpen).toBe(true);

    act(() => {
      result.current.toggleRightPane();
    });
    expect(result.current.rightPaneOpen).toBe(false);
    expect(localStorage.getItem(RIGHT_PANE_KEY)).toBe('false');

    act(() => {
      result.current.toggleRightPane();
    });
    expect(result.current.rightPaneOpen).toBe(true);
    expect(localStorage.getItem(RIGHT_PANE_KEY)).toBe('true');
  });

  it('setRightPaneOpen persists the given value', () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    act(() => {
      result.current.setRightPaneOpen(false);
    });
    expect(result.current.rightPaneOpen).toBe(false);
    expect(localStorage.getItem(RIGHT_PANE_KEY)).toBe('false');
  });

  it('respects a persisted "false" on mount', () => {
    localStorage.setItem(RIGHT_PANE_KEY, 'false');
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneOpen).toBe(false);
  });

  it('collapses the rail when a card is pinned and restores it on unpin', () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    // Start open
    expect(result.current.rightPaneOpen).toBe(true);

    const [card] = sampleCards();
    act(() => {
      result.current.setPinnedCard(card);
    });
    // Pinning collapses the rail
    expect(result.current.rightPaneOpen).toBe(false);

    act(() => {
      result.current.setPinnedCard(null);
    });
    // Unpinning restores the prior state (was open)
    expect(result.current.rightPaneOpen).toBe(true);
  });

  it('restores a closed rail to closed after a pin cycle', () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    // User closes the rail first
    act(() => {
      result.current.setRightPaneOpen(false);
    });
    expect(result.current.rightPaneOpen).toBe(false);

    const [card] = sampleCards();
    act(() => {
      result.current.setPinnedCard(card);
    });
    expect(result.current.rightPaneOpen).toBe(false);

    act(() => {
      result.current.setPinnedCard(null);
    });
    // Was closed before pinning, stays closed after unpin
    expect(result.current.rightPaneOpen).toBe(false);
  });

  it('does not persist the forced pin-collapsed state to localStorage', () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    const [card] = sampleCards();
    act(() => {
      result.current.setPinnedCard(card);
    });
    // The auto-collapse should not have written 'false' to localStorage —
    // the user's saved preference (default open) must survive a reload.
    expect(localStorage.getItem(RIGHT_PANE_KEY)).toBeNull();

    // Clear the pinned card from storage to reset the pin effect for a clean
    // remount, then confirm a fresh mount still sees the pane as open.
    localStorage.removeItem(PINNED_CARD_KEY);
    const { result: remounted } = renderHook(() => useUIState(), { wrapper });
    expect(remounted.current.rightPaneOpen).toBe(true);
  });
});
