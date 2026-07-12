// zettelkasten-front/src/components/cards/ViewMobileLayout.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ViewMobileLayout } from "./ViewMobileLayout";
import { Card, defaultPartialCard } from "../../models/Card";

// Mock react-router-dom
vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

// Mock child components
vi.mock("./ViewCardContentSection", () => ({
  ViewCardContentSection: ({ viewingCard }: any) => (
    <div data-testid="view-card-content-section">Content Section</div>
  ),
}));

vi.mock("../tags/SearchTagDropdown", () => ({
  SearchTagDropdown: ({ tags, handleTagClick }: any) => (
    <div data-testid="search-tag-dropdown">Add Tag</div>
  ),
}));

vi.mock("./RelatedCards", () => ({
  RelatedCards: ({ relatedCards, onCardClick }: any) => (
    <div data-testid="related-cards">Related Cards ({relatedCards.length})</div>
  ),
}));

vi.mock("../schemas/CardStructuredDataDisplay", () => ({
  CardStructuredDataDisplay: () => (
    <div data-testid="structured-data-display">Structured Data</div>
  ),
}));

vi.mock("../../utils/strings", () => ({
  linkifyWithDefaultOptions: (str: string) => str,
}));

vi.mock("../../assets/icons/PersonIcon", () => ({
  PersonIcon: () => <span data-testid="person-icon">P</span>,
}));

vi.mock("./ViewNavigationSheet", () => ({
  ViewNavigationSheet: ({ isOpen, onClose, title }: any) =>
    isOpen ? <div data-testid="view-navigation-sheet">Navigate</div> : null,
}));

const mockViewingCard: Card = {
  id: 1,
  card_id: "test-1",
  user_id: 1,
  title: "Test Card",
  body: "Test body content",
  link: "",
  is_deleted: false,
  created_at: new Date("2024-01-01"),
  updated_at: new Date("2024-01-02"),
  parent_id: -1,
  parent: defaultPartialCard,
  files: [],
  children: [],
  references: [],
  tags: [{ name: "test-tag", id: 1, color: "#3b82f6", user_id: 1 }],
  tasks: [],
  entities: [],
  is_starred: false,
};

describe("ViewMobileLayout", () => {
  const defaultProps = {
    viewingCard: mockViewingCard,
    parentCard: null,
    prevSibling: null,
    nextSibling: null,
    linkedEntities: [],
    categorizedReferences: { bidirectional: [], outgoing: [], incoming: [] },
    summaries: [],
    latestSummary: null,
    relatedCards: null,
    tags: [],
    onEditCard: vi.fn(),
    onCreateChildCard: vi.fn(),
    onToggleStar: vi.fn(),
    toggleCreateTaskWindow: vi.fn(),
    onTagClick: vi.fn(),
    onRemoveTag: vi.fn(),
    onAddBacklink: vi.fn(),
    handleOpenEntity: vi.fn(),
    onResummarize: vi.fn(),
    onRecategorize: vi.fn(),
    refreshCard: vi.fn(),
    setViewCard: vi.fn(),
    setError: vi.fn(),
    setSelectedFact: vi.fn(),
    setShowFactDialog: vi.fn(),
    fileUploadRef: { current: null },
    onSaveCard: vi.fn(),
    viewMode: "normal" as const,
    onViewModeChange: vi.fn(),
  };

  it("renders card title in top bar", () => {
    render(<ViewMobileLayout {...defaultProps} />);
    // The title appears in the h1 element in the top bar
    expect(
      screen.getByRole("heading", { name: "Test Card" }),
    ).toBeInTheDocument();
  });

  it("renders tags section expanded by default", () => {
    render(<ViewMobileLayout {...defaultProps} />);
    expect(screen.getByText("#test-tag")).toBeVisible();
  });

  it("renders main content section", () => {
    render(<ViewMobileLayout {...defaultProps} />);
    expect(screen.getByTestId("view-card-content-section")).toBeInTheDocument();
  });

  it("shows navigation sheet when navigate clicked", () => {
    render(<ViewMobileLayout {...defaultProps} />);
    fireEvent.click(screen.getByLabelText("More options"));
    fireEvent.click(screen.getByText("Navigate..."));
    expect(screen.getByText("Navigate")).toBeVisible();
  });

  it("renders collapsed navigation section when parent exists", () => {
    const parentCard = { ...mockViewingCard, id: 2, title: "Parent" };
    render(<ViewMobileLayout {...defaultProps} parentCard={parentCard} />);
    expect(screen.getByText("Navigation")).toBeInTheDocument();
  });
});
