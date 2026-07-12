import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import { useRightPaneTab } from "./useRightPaneTab";
import { UIStateProvider, useUIState } from "../contexts/UIStateContext";

/**
 * Reads the active rail tab back out of context so the hook's side effects can
 * be asserted without it returning anything itself.
 */
function Probe({ onTab }: { onTab: (tab: string) => void }) {
  const { rightPaneTab } = useUIState();
  onTab(rightPaneTab);
  return null;
}

function renderHookWithProviders(
  hasRelationships: boolean,
  probe = vi.fn(),
  initialPath = "/",
) {
  window.history.replaceState({}, "", initialPath);
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <BrowserRouter>
      <UIStateProvider>
        {children}
        <Probe onTab={probe} />
      </UIStateProvider>
    </BrowserRouter>
  );
  return renderHook(() => useRightPaneTab({ hasRelationships }), { wrapper });
}

describe("useRightPaneTab", () => {
  beforeEach(() => {
    // URL-param tests mutate the location; reset between tests.
    window.history.replaceState({}, "", "/");
  });

  describe("smart default on mount", () => {
    it("defaults to metadata when there are no relationships", () => {
      const probe = vi.fn();
      renderHookWithProviders(false, probe);
      expect(probe).toHaveBeenLastCalledWith("metadata");
      // …and mirrors it into the URL.
      expect(window.location.search).toContain("pane=metadata");
    });

    it("defaults to links when there are relationships to show", () => {
      const probe = vi.fn();
      renderHookWithProviders(true, probe);
      expect(probe).toHaveBeenLastCalledWith("links");
      expect(window.location.search).toContain("pane=links");
    });
  });

  describe("?pane= mount-read", () => {
    it("honors a valid ?pane= param over the smart default", () => {
      const probe = vi.fn();
      renderHookWithProviders(false, probe, "/?pane=entities");
      // URL wins; would otherwise be metadata.
      expect(probe).toHaveBeenLastCalledWith("entities");
    });

    it("falls back to the smart default for an invalid ?pane= value", () => {
      const probe = vi.fn();
      renderHookWithProviders(false, probe, "/?pane=bogus");
      expect(probe).toHaveBeenLastCalledWith("metadata");
    });

    it("treats an empty ?pane= as absent and uses the smart default", () => {
      const probe = vi.fn();
      renderHookWithProviders(true, probe, "/?pane=");
      expect(probe).toHaveBeenLastCalledWith("links");
    });
  });
});
