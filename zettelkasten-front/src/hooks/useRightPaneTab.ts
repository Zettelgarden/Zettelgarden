import { useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useUIState, RightPaneTab } from "../contexts/UIStateContext";

const VALID_TAB_IDS: RightPaneTab[] = ["links", "metadata", "entities"];

/**
 * Keeps `rightPaneTab` (from `UIStateContext`) in sync with the `?pane=`
 * query parameter so a rail tab is shareable/bookmarkable. Shared by the view
 * and edit pages so a deep link to a tab works on either.
 *
 * On mount: if `?pane=` is a valid tab id, adopt it; otherwise pick a smart
 * default — Links when there's relationship data to show, Metadata otherwise.
 * Afterwards, every tab change is mirrored back into the URL with `replace`
 * (no history pollution).
 *
 * @param hasRelationships whether the host page has children/references to
 *   show, which decides the Links-vs-Metadata smart default.
 */
export function useRightPaneTab({ hasRelationships }: { hasRelationships: boolean }) {
  const { rightPaneTab, setRightPaneTab } = useUIState();
  const [searchParams, setSearchParams] = useSearchParams();

  // On mount: pick the initial tab from ?pane= if present and valid, otherwise
  // fall back to the smart default. Runs once; explicit tab clicks (and the
  // URL sync below) take over afterwards.
  useEffect(() => {
    const pane = searchParams.get("pane");
    if (pane && VALID_TAB_IDS.includes(pane as RightPaneTab)) {
      setRightPaneTab(pane as RightPaneTab);
      return;
    }
    setRightPaneTab(hasRelationships ? "links" : "metadata");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Reflect the active tab in ?pane= so a view is shareable/bookmarkable.
  // Uses replace to avoid polluting browser history on every tab switch.
  useEffect(() => {
    setSearchParams(
      (prev) => {
        if (prev.get("pane") === rightPaneTab) return prev;
        const next = new URLSearchParams(prev);
        next.set("pane", rightPaneTab);
        return next;
      },
      { replace: true },
    );
  }, [rightPaneTab, setSearchParams]);
}
