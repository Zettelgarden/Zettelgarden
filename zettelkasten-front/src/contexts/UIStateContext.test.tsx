import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { UIStateProvider, useUIState } from "./UIStateContext";

const RIGHT_PANE_KEY = "zettelgarden-right-pane-open";

function wrapper({ children }: { children: React.ReactNode }) {
  return <UIStateProvider>{children}</UIStateProvider>;
}

describe("UIStateContext — right pane", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("defaults the right pane to open", () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneOpen).toBe(true);
  });

  it("toggleRightPane flips the state and persists it", () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneOpen).toBe(true);

    act(() => {
      result.current.toggleRightPane();
    });
    expect(result.current.rightPaneOpen).toBe(false);
    expect(localStorage.getItem(RIGHT_PANE_KEY)).toBe("false");

    act(() => {
      result.current.toggleRightPane();
    });
    expect(result.current.rightPaneOpen).toBe(true);
    expect(localStorage.getItem(RIGHT_PANE_KEY)).toBe("true");
  });

  it("setRightPaneOpen persists the given value", () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    act(() => {
      result.current.setRightPaneOpen(false);
    });
    expect(result.current.rightPaneOpen).toBe(false);
    expect(localStorage.getItem(RIGHT_PANE_KEY)).toBe("false");
  });

  it('respects a persisted "false" on mount', () => {
    localStorage.setItem(RIGHT_PANE_KEY, "false");
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneOpen).toBe(false);
  });
});
