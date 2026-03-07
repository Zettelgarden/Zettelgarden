// zettelkasten-front/src/components/cards/ViewNavigationSheet.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ViewNavigationSheet } from "./ViewNavigationSheet";
import { Card, PartialCard, defaultPartialCard } from "../../models/Card";

const mockParentCard: Card = {
  id: 1,
  card_id: "parent-1",
  user_id: 1,
  title: "Parent Card",
  body: "Parent body",
  link: "",
  is_deleted: false,
  created_at: new Date(),
  updated_at: new Date(),
  parent_id: -1,
  parent: defaultPartialCard,
  files: [],
  children: [],
  references: [],
  tags: [],
  tasks: [],
  external_events: [],
  entities: [],
  is_starred: false,
};

const mockViewingCard: Card = {
  ...mockParentCard,
  id: 4,
  card_id: "viewing-1",
  title: "Viewing Card",
  parent_id: 1,
  children: [],
};

const mockPrevSibling: PartialCard = {
  id: 2,
  card_id: "prev-1",
  user_id: 1,
  title: "Previous Sibling",
  parent_id: 1,
  created_at: new Date(),
  updated_at: new Date(),
  tags: [],
};

const mockNextSibling: PartialCard = {
  id: 3,
  card_id: "next-1",
  user_id: 1,
  title: "Next Sibling",
  parent_id: 1,
  created_at: new Date(),
  updated_at: new Date(),
  tags: [],
};

describe("ViewNavigationSheet", () => {
  it("renders nothing when closed", () => {
    const onNavigate = vi.fn();
    render(
      <ViewNavigationSheet
        isOpen={false}
        onClose={() => {}}
        parentCard={null}
        prevSibling={null}
        nextSibling={null}
        viewingCard={mockViewingCard}
        onNavigate={onNavigate}
      />
    );
    expect(screen.queryByText("Navigate")).not.toBeInTheDocument();
  });

  it("renders navigation options when open", () => {
    const onNavigate = vi.fn();
    render(
      <ViewNavigationSheet
        isOpen={true}
        onClose={() => {}}
        parentCard={mockParentCard}
        prevSibling={mockPrevSibling}
        nextSibling={mockNextSibling}
        viewingCard={mockViewingCard}
        onNavigate={onNavigate}
      />
    );
    expect(screen.getByText("Navigate")).toBeInTheDocument();
    expect(screen.getByText("Parent Card")).toBeInTheDocument();
    expect(screen.getByText("Previous Sibling")).toBeInTheDocument();
    expect(screen.getByText("Next Sibling")).toBeInTheDocument();
  });

  it("calls onNavigate when parent card clicked", () => {
    const onNavigate = vi.fn();
    render(
      <ViewNavigationSheet
        isOpen={true}
        onClose={() => {}}
        parentCard={mockParentCard}
        prevSibling={null}
        nextSibling={null}
        viewingCard={mockViewingCard}
        onNavigate={onNavigate}
      />
    );
    fireEvent.click(screen.getByText("Parent Card"));
    expect(onNavigate).toHaveBeenCalledWith(1);
  });

  it("calls onNavigate when prev sibling clicked", () => {
    const onNavigate = vi.fn();
    render(
      <ViewNavigationSheet
        isOpen={true}
        onClose={() => {}}
        parentCard={null}
        prevSibling={mockPrevSibling}
        nextSibling={null}
        viewingCard={mockViewingCard}
        onNavigate={onNavigate}
      />
    );
    fireEvent.click(screen.getByText("Previous Sibling"));
    expect(onNavigate).toHaveBeenCalledWith(2);
  });

  it("calls onNavigate when next sibling clicked", () => {
    const onNavigate = vi.fn();
    render(
      <ViewNavigationSheet
        isOpen={true}
        onClose={() => {}}
        parentCard={null}
        prevSibling={null}
        nextSibling={mockNextSibling}
        viewingCard={mockViewingCard}
        onNavigate={onNavigate}
      />
    );
    fireEvent.click(screen.getByText("Next Sibling"));
    expect(onNavigate).toHaveBeenCalledWith(3);
  });
});
