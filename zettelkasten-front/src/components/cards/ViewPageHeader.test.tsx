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

describe("ViewPageHeader — ＋ Child / ＋ Link affordances", () => {
  it("renders both creation buttons on the view-mode row", () => {
    renderHeader();
    expect(screen.getByText("＋ Child")).toBeInTheDocument();
    expect(screen.getByText("＋ Link")).toBeInTheDocument();
  });

  it("opens the rail to the Links tab when ＋ Link is clicked", () => {
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
            onCreateChildCard={() => {}}
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
    fireEvent.click(screen.getByText("＋ Link"));
    expect(screen.getByTestId("rail-open").textContent).toBe("open");
    expect(screen.getByTestId("rail-tab").textContent).toBe("links");
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
