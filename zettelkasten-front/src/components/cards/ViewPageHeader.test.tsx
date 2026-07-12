import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ViewPageHeader } from "./ViewPageHeader";
import { UIStateProvider, useUIState } from "../../contexts/UIStateContext";
import { sampleCards } from "../../tests/data";
import { ViewMode } from "../../pages/cards/ViewPageContainer";

const [card] = sampleCards();

function renderHeader(
  overrides: Partial<React.ComponentProps<typeof ViewPageHeader>> = {},
) {
  const noop = () => {};
  return render(
    <UIStateProvider>
      <ViewPageHeader
        viewingCard={card}
        onEditCard={noop}
        onToggleStar={noop}
        toggleCreateTaskWindow={noop}
        onResummarize={noop}
        onRecategorize={noop}
        viewMode={"normal" as ViewMode}
        onViewModeChange={noop}
        onCreateChildCard={noop}
        {...overrides}
      />
    </UIStateProvider>,
  );
}

describe("ViewPageHeader — info pane toggle", () => {
  it("renders the toggle button by default", () => {
    renderHeader();
    expect(screen.getByTitle("Toggle info pane")).toBeInTheDocument();
  });

  it("reflects the open state with aria-pressed", () => {
    renderHeader();
    const toggle = screen.getByRole("button", { name: "Toggle info pane" });
    expect(toggle).toHaveAttribute("aria-pressed", "true");
  });
});

describe("ViewPageHeader — ＋ Child affordance", () => {
  it("renders the ＋ Child button and not the removed ＋ Link button", () => {
    renderHeader();
    expect(screen.getByText("＋ Child")).toBeInTheDocument();
    // The ＋ Link button was navigation-only and is removed; the Links tab is
    // reachable from the rail tab strip.
    expect(screen.queryByText("＋ Link")).not.toBeInTheDocument();
  });

  it("triggers child creation and opens the rail when ＋ Child is clicked", () => {
    const onCreateChildCard = vi.fn();
    function Harness() {
      const { rightPaneOpen, rightPaneTab } = useUIState();
      return (
        <>
          <ViewPageHeader
            viewingCard={card}
            onEditCard={() => {}}
            onToggleStar={() => {}}
            toggleCreateTaskWindow={() => {}}
            onResummarize={() => {}}
            onRecategorize={() => {}}
            viewMode={"normal" as ViewMode}
            onViewModeChange={() => {}}
            onCreateChildCard={onCreateChildCard}
          />
          <div data-testid="rail-open">{rightPaneOpen ? "open" : "closed"}</div>
          <div data-testid="rail-tab">{rightPaneTab}</div>
        </>
      );
    }
    render(
      <UIStateProvider>
        <Harness />
      </UIStateProvider>,
    );
    fireEvent.click(screen.getByText("＋ Child"));
    expect(onCreateChildCard).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("rail-open").textContent).toBe("open");
    expect(screen.getByTestId("rail-tab").textContent).toBe("links");
  });
});

describe("ViewPageHeader — sibling/parent navigation", () => {
  it("hides the sibling nav cluster when there is no parent or siblings", () => {
    renderHeader();
    expect(screen.queryByText("‹ Prev")).not.toBeInTheDocument();
    expect(screen.queryByText("↑ Up")).not.toBeInTheDocument();
    expect(screen.queryByText("Next ›")).not.toBeInTheDocument();
  });

  it("renders Prev/Up/Next and wires them to the callbacks", () => {
    const onNavigateParent = vi.fn();
    const onNavigatePrev = vi.fn();
    const onNavigateNext = vi.fn();
    // A parent is required for ↑ Up to show (hasParent = parent data present).
    renderHeader({
      onNavigateParent,
      onNavigatePrev,
      onNavigateNext,
      viewingCard: { ...card, parent: { id: 1, card_id: "parent" } as any },
    });
    const prev = screen.getByText("‹ Prev");
    const up = screen.getByText("↑ Up");
    const next = screen.getByText("Next ›");
    fireEvent.click(prev);
    fireEvent.click(up);
    fireEvent.click(next);
    expect(onNavigatePrev).toHaveBeenCalledTimes(1);
    expect(onNavigateParent).toHaveBeenCalledTimes(1);
    expect(onNavigateNext).toHaveBeenCalledTimes(1);
  });
});
