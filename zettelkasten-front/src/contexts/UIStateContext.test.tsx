import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { UIStateProvider, useUIState } from "./UIStateContext";
import {
  RIGHT_PANE_WIDTH_DEFAULT,
  RIGHT_PANE_WIDTH_MIN,
  RIGHT_PANE_WIDTH_MAX,
} from "./UIStateContext";

const RIGHT_PANE_KEY = "zettelgarden-right-pane-open";
const WIDTH_KEY = "zettelgarden-right-pane-width";

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

describe("UIStateContext — right pane width", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("defaults to the default width when nothing is stored", () => {
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneWidth).toBe(RIGHT_PANE_WIDTH_DEFAULT);
  });

  it("setRightPaneWidth clamps to the allowed range and persists", () => {
    const { result } = renderHook(() => useUIState(), { wrapper });

    act(() => {
      result.current.setRightPaneWidth(450);
    });
    expect(result.current.rightPaneWidth).toBe(450);
    expect(localStorage.getItem(WIDTH_KEY)).toBe("450");

    act(() => {
      result.current.setRightPaneWidth(10);
    });
    expect(result.current.rightPaneWidth).toBe(RIGHT_PANE_WIDTH_MIN);
    expect(localStorage.getItem(WIDTH_KEY)).toBe(String(RIGHT_PANE_WIDTH_MIN));

    act(() => {
      result.current.setRightPaneWidth(99999);
    });
    expect(result.current.rightPaneWidth).toBe(RIGHT_PANE_WIDTH_MAX);
    expect(localStorage.getItem(WIDTH_KEY)).toBe(String(RIGHT_PANE_WIDTH_MAX));
  });

  it("respects a persisted width on mount (within range)", () => {
    localStorage.setItem(WIDTH_KEY, "360");
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneWidth).toBe(360);
  });

  it("clamps an out-of-range persisted value on mount", () => {
    localStorage.setItem(WIDTH_KEY, "50");
    const { result } = renderHook(() => useUIState(), { wrapper });
    expect(result.current.rightPaneWidth).toBe(RIGHT_PANE_WIDTH_MIN);
  });
});
