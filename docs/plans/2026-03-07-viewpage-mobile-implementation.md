# ViewPage Mobile Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve mobile experience for ViewPage with accordion sections and navigation bottom sheet.

**Architecture:** Create ViewMobileLayout wrapper component that uses existing useViewPageContainer hook. Side panels become collapsible accordion sections. Navigation accessed via bottom sheet on demand.

**Tech Stack:** React, TypeScript, Tailwind CSS, existing MobileBottomSheet component

---

## Task 1: Create ViewMobileAccordion Component

**Files:**
- Create: `zettelkasten-front/src/components/cards/ViewMobileAccordion.tsx`
- Create: `zettelkasten-front/src/components/cards/ViewMobileAccordion.test.tsx`

**Step 1: Write the failing test**

```typescript
// zettelkasten-front/src/components/cards/ViewMobileAccordion.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ViewMobileAccordion } from "./ViewMobileAccordion";

describe("ViewMobileAccordion", () => {
  it("renders title in header", () => {
    render(
      <ViewMobileAccordion title="Tags">
        <div>Content</div>
      </ViewMobileAccordion>
    );
    expect(screen.getByText("Tags")).toBeInTheDocument();
  });

  it("shows content when expanded by default", () => {
    render(
      <ViewMobileAccordion title="Tags" defaultExpanded>
        <div>Test Content</div>
      </ViewMobileAccordion>
    );
    expect(screen.getByText("Test Content")).toBeVisible();
  });

  it("hides content when collapsed", () => {
    render(
      <ViewMobileAccordion title="Tags">
        <div>Test Content</div>
      </ViewMobileAccordion>
    );
    expect(screen.queryByText("Test Content")).not.toBeInTheDocument();
  });

  it("toggles content on header click", () => {
    render(
      <ViewMobileAccordion title="Tags">
        <div>Test Content</div>
      </ViewMobileAccordion>
    );

    // Initially collapsed
    expect(screen.queryByText("Test Content")).not.toBeInTheDocument();

    // Click to expand
    fireEvent.click(screen.getByText("Tags"));
    expect(screen.getByText("Test Content")).toBeVisible();

    // Click to collapse
    fireEvent.click(screen.getByText("Tags"));
    expect(screen.queryByText("Test Content")).not.toBeInTheDocument();
  });

  it("renders right element in header", () => {
    render(
      <ViewMobileAccordion title="Tags" rightElement={<button>Edit</button>}>
        <div>Content</div>
      </ViewMobileAccordion>
    );
    expect(screen.getByText("Edit")).toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- ViewMobileAccordion.test.tsx`
Expected: FAIL with "Cannot find module './ViewMobileAccordion'"

**Step 3: Write minimal implementation**

```typescript
// zettelkasten-front/src/components/cards/ViewMobileAccordion.tsx
import React, { useState } from "react";

interface ViewMobileAccordionProps {
  title: string;
  icon?: React.ReactNode;
  defaultExpanded?: boolean;
  rightElement?: React.ReactNode;
  children: React.ReactNode;
}

export function ViewMobileAccordion({
  title,
  icon,
  defaultExpanded = false,
  rightElement,
  children,
}: ViewMobileAccordionProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  return (
    <div className="border-b border-gray-200">
      <button
        className="w-full sticky top-0 bg-gray-50 px-4 py-3 flex items-center justify-between text-left hover:bg-gray-100 transition-colors z-10"
        onClick={() => setIsExpanded(!isExpanded)}
        aria-expanded={isExpanded}
      >
        <div className="flex items-center gap-2">
          <span className="text-gray-400 text-sm">
            {isExpanded ? "▼" : "►"}
          </span>
          {icon && <span className="text-gray-500">{icon}</span>}
          <span className="font-medium text-gray-900">{title}</span>
        </div>
        {rightElement && (
          <div onClick={(e) => e.stopPropagation()}>
            {rightElement}
          </div>
        )}
      </button>
      {isExpanded && (
        <div className="px-4 py-3 bg-white">
          {children}
        </div>
      )}
    </div>
  );
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- ViewMobileAccordion.test.tsx`
Expected: PASS (4 tests)

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/cards/ViewMobileAccordion.tsx
git add zettelkasten-front/src/components/cards/ViewMobileAccordion.test.tsx
git commit -m "feat(cards): add ViewMobileAccordion component for collapsible sections"
```

---

## Task 2: Create ViewNavigationSheet Component

**Files:**
- Create: `zettelkasten-front/src/components/cards/ViewNavigationSheet.tsx`
- Create: `zettelkasten-front/src/components/cards/ViewNavigationSheet.test.tsx`

**Step 1: Write the failing test**

```typescript
// zettelkasten-front/src/components/cards/ViewNavigationSheet.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ViewNavigationSheet } from "./ViewNavigationSheet";
import { Card, PartialCard, defaultPartialCard } from "../../models/Card";

const mockParentCard: Card = {
  id: 1,
  card_id: "parent-1",
  user_id: "user-1",
  title: "Parent Card",
  body: "Parent body",
  link: "",
  is_deleted: false,
  created_at: new Date(),
  updated_at: new Date(),
  parent_id: null,
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

const mockPrevSibling: PartialCard = {
  id: 2,
  card_id: "prev-1",
  user_id: "user-1",
  title: "Previous Sibling",
  parent_id: 1,
  created_at: new Date(),
  updated_at: new Date(),
};

const mockNextSibling: PartialCard = {
  id: 3,
  card_id: "next-1",
  user_id: "user-1",
  title: "Next Sibling",
  parent_id: 1,
  created_at: new Date(),
  updated_at: new Date(),
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
        viewingCard={mockParentCard}
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
        viewingCard={mockParentCard}
        onNavigate={onNavigate}
      />
    );
    expect(screen.getByText("Navigate")).toBeInTheDocument();
    expect(screen.getByText("Parent Card")).toBeInTheDocument();
    expect(screen.getByText("← Prev")).toBeInTheDocument();
    expect(screen.getByText("Next →")).toBeInTheDocument();
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
        viewingCard={mockParentCard}
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
        viewingCard={mockParentCard}
        onNavigate={onNavigate}
      />
    );
    fireEvent.click(screen.getByText("← Prev"));
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
        viewingCard={mockParentCard}
        onNavigate={onNavigate}
      />
    );
    fireEvent.click(screen.getByText("Next →"));
    expect(onNavigate).toHaveBeenCalledWith(3);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- ViewNavigationSheet.test.tsx`
Expected: FAIL with "Cannot find module './ViewNavigationSheet'"

**Step 3: Write minimal implementation**

```typescript
// zettelkasten-front/src/components/cards/ViewNavigationSheet.tsx
import React from "react";
import { MobileBottomSheet } from "../layout/MobileBottomSheet";
import { Card, PartialCard } from "../../models/Card";

interface ViewNavigationSheetProps {
  isOpen: boolean;
  onClose: () => void;
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  viewingCard: Card;
  onNavigate: (cardId: number) => void;
}

export function ViewNavigationSheet({
  isOpen,
  onClose,
  parentCard,
  prevSibling,
  nextSibling,
  viewingCard,
  onNavigate,
}: ViewNavigationSheetProps) {
  const hasNavigation = parentCard || prevSibling || nextSibling;
  const hasChildren = viewingCard.children && viewingCard.children.length > 0;

  if (!hasNavigation && !hasChildren) {
    return null;
  }

  return (
    <MobileBottomSheet
      isOpen={isOpen}
      onClose={onClose}
      title="Navigate"
    >
      <div className="p-4 space-y-4">
        {/* Parent Card */}
        {parentCard && (
          <button
            onClick={() => {
              onNavigate(parentCard.id);
              onClose();
            }}
            className="w-full p-3 bg-gray-50 rounded-lg text-left hover:bg-gray-100 transition-colors"
          >
            <div className="flex items-center gap-2">
              <span className="text-gray-400">↑</span>
              <div>
                <div className="text-xs text-gray-500">Parent</div>
                <div className="font-medium text-gray-900">{parentCard.title}</div>
              </div>
            </div>
          </button>
        )}

        {/* Sibling Navigation */}
        {(prevSibling || nextSibling) && (
          <div className="flex gap-2">
            {prevSibling && (
              <button
                onClick={() => {
                  onNavigate(prevSibling.id);
                  onClose();
                }}
                className="flex-1 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
              >
                <span className="text-sm font-medium">← Prev</span>
              </button>
            )}
            {nextSibling && (
              <button
                onClick={() => {
                  onNavigate(nextSibling.id);
                  onClose();
                }}
                className="flex-1 p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
              >
                <span className="text-sm font-medium">Next →</span>
              </button>
            )}
          </div>
        )}

        {/* Children */}
        {hasChildren && (
          <div>
            <div className="text-xs text-gray-500 mb-2">Children</div>
            <div className="space-y-1">
              {viewingCard.children.map((child) => (
                <button
                  key={child.id}
                  onClick={() => {
                    onNavigate(child.id);
                    onClose();
                  }}
                  className="w-full p-2 text-left text-sm text-blue-600 hover:bg-gray-50 rounded"
                >
                  {child.title || "Untitled"}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </MobileBottomSheet>
  );
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- ViewNavigationSheet.test.tsx`
Expected: PASS (5 tests)

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/cards/ViewNavigationSheet.tsx
git add zettelkasten-front/src/components/cards/ViewNavigationSheet.test.tsx
git commit -m "feat(cards): add ViewNavigationSheet for mobile hierarchy nav"
```

---

## Task 3: Create ViewMobileLayout Component

**Files:**
- Create: `zettelkasten-front/src/components/cards/ViewMobileLayout.tsx`
- Create: `zettelkasten-front/src/components/cards/ViewMobileLayout.test.tsx`

**Step 1: Write the failing test**

```typescript
// zettelkasten-front/src/components/cards/ViewMobileLayout.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ViewMobileLayout } from "./ViewMobileLayout";
import { Card, defaultPartialCard } from "../../models/Card";

const mockViewingCard: Card = {
  id: 1,
  card_id: "test-1",
  user_id: "user-1",
  title: "Test Card",
  body: "Test body content",
  link: "",
  is_deleted: false,
  created_at: new Date("2024-01-01"),
  updated_at: new Date("2024-01-02"),
  parent_id: null,
  parent: defaultPartialCard,
  files: [],
  children: [],
  references: [],
  tags: [{ name: "test-tag", id: 1 }],
  tasks: [],
  external_events: [],
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
    categorizedReferences: { backlinks: [], references: [] },
    summaries: [],
    latestSummary: null,
    analysis: null,
    relatedCards: null,
    tags: [],
    onEditCard: vi.fn(),
    onCreateChildCard: vi.fn(),
    onToggleStar: vi.fn(),
    onTogglePin: vi.fn(),
    onOpenChatSidebar: vi.fn(),
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
  };

  it("renders card title in top bar", () => {
    render(<ViewMobileLayout {...defaultProps} />);
    expect(screen.getByText("Test Card")).toBeInTheDocument();
  });

  it("renders tags section expanded by default", () => {
    render(<ViewMobileLayout {...defaultProps} />);
    expect(screen.getByText("#test-tag")).toBeVisible();
  });

  it("renders main content section", () => {
    render(<ViewMobileLayout {...defaultProps} />);
    // The main content should be visible
    expect(screen.getByText("Test Card")).toBeInTheDocument();
  });

  it("shows navigation sheet when navigate clicked", () => {
    render(<ViewMobileLayout {...defaultProps} />);
    // Click menu button
    fireEvent.click(screen.getByLabelText("More options"));
    // Click navigate option
    fireEvent.click(screen.getByText("Navigate"));
    expect(screen.getByText("Navigate")).toBeVisible();
  });

  it("renders collapsed navigation section when parent exists", () => {
    const parentCard = { ...mockViewingCard, id: 2, title: "Parent" };
    render(<ViewMobileLayout {...defaultProps} parentCard={parentCard} />);
    expect(screen.getByText("Navigation")).toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd zettelkasten-front && npm test -- ViewMobileLayout.test.tsx`
Expected: FAIL with "Cannot find module './ViewMobileLayout'"

**Step 3: Write minimal implementation**

```typescript
// zettelkasten-front/src/components/cards/ViewMobileLayout.tsx
import React, { useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Card, PartialCard, Entity, CategorizedReferences, Summary, Analysis, RelatedCard } from "../../models/Card";
import { ViewMobileAccordion } from "./ViewMobileAccordion";
import { ViewNavigationSheet } from "./ViewNavigationSheet";
import { ViewCardContentSection } from "./ViewCardContentSection";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { RelatedCards } from "./RelatedCards";
import { HeaderSubSection } from "../Header";
import { linkifyWithDefaultOptions } from "../../utils/strings";
import { PersonIcon } from "../../assets/icons/PersonIcon";
import { CardStructuredDataDisplay } from "../schemas/CardStructuredDataDisplay";
import { RSSArticle } from "../../api/rss";

interface ViewMobileLayoutProps {
  viewingCard: Card;
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  linkedEntities: Entity[];
  categorizedReferences: CategorizedReferences;
  summaries: Summary[];
  latestSummary: Summary | null;
  analysis: Analysis | null;
  relatedCards: RelatedCard[] | null;
  tags: any[];
  sourceArticle?: RSSArticle;
  onEditCard: () => void;
  onCreateChildCard: () => void;
  onToggleStar: () => void;
  onTogglePin: () => void;
  onOpenChatSidebar: () => void;
  toggleCreateTaskWindow: () => void;
  onTagClick: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  onAddBacklink: () => void;
  handleOpenEntity: (entity: Entity) => void;
  onResummarize: () => void;
  onRecategorize: () => void;
  refreshCard: () => void;
  setViewCard: (card: Card) => void;
  setError: (error: string) => void;
  setSelectedFact: (fact: any) => void;
  setShowFactDialog: (show: boolean) => void;
  fileUploadRef: React.RefObject<HTMLInputElement>;
  onSaveCard: (card: Card) => void;
}

type ViewMode = 'normal' | 'tree' | 'summary' | 'analysis';

export function ViewMobileLayout({
  viewingCard,
  parentCard,
  prevSibling,
  nextSibling,
  linkedEntities,
  categorizedReferences,
  summaries,
  latestSummary,
  analysis,
  relatedCards,
  tags,
  sourceArticle,
  onEditCard,
  onCreateChildCard,
  onToggleStar,
  onTogglePin,
  onOpenChatSidebar,
  toggleCreateTaskWindow,
  onTagClick,
  onRemoveTag,
  onAddBacklink,
  handleOpenEntity,
  onResummarize,
  onRecategorize,
  refreshCard,
  setViewCard,
  setError,
  setSelectedFact,
  setShowFactDialog,
  fileUploadRef,
  onSaveCard,
}: ViewMobileLayoutProps) {
  const navigate = useNavigate();
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['tags']));
  const [showNavSheet, setShowNavSheet] = useState(false);
  const [showMenu, setShowMenu] = useState(false);
  const [viewMode, setViewMode] = useState<ViewMode>('normal');

  const toggleSection = (section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(section)) {
        next.delete(section);
      } else {
        next.add(section);
      }
      return next;
    });
  };

  const handleNavigate = (cardId: number) => {
    navigate(`/app/card/${cardId}`);
  };

  const hasNavigation = parentCard || prevSibling || nextSibling;
  const hasEntities = linkedEntities && linkedEntities.length > 0;
  const hasRelatedCards = relatedCards && relatedCards.length > 0;
  const hasChildren = viewingCard.children && viewingCard.children.length > 0;

  return (
    <div className="flex flex-col h-full overflow-hidden md:hidden">
      {/* Top Bar */}
      <div className="sticky top-0 bg-white border-b border-gray-200 z-20">
        <div className="flex items-center justify-between px-4 py-3">
          <h1 className="text-lg font-semibold text-gray-900 truncate">
            {viewingCard.title || "Card"}
          </h1>
          <div className="relative">
            <button
              onClick={() => setShowMenu(!showMenu)}
              className="p-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
              aria-label="More options"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
              </svg>
            </button>
            {showMenu && (
              <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
                <div className="px-3 py-2 text-xs text-gray-500 font-medium">View Mode</div>
                {(['normal', 'tree', 'summary', 'analysis'] as ViewMode[]).map((mode) => (
                  <button
                    key={mode}
                    onClick={() => {
                      setViewMode(mode);
                      setShowMenu(false);
                    }}
                    className={`w-full px-3 py-2 text-left text-sm hover:bg-gray-50 ${
                      viewMode === mode ? 'text-blue-600 font-medium' : 'text-gray-700'
                    }`}
                  >
                    {mode.charAt(0).toUpperCase() + mode.slice(1)}
                  </button>
                ))}
                <hr className="my-1" />
                <button
                  onClick={() => {
                    setShowNavSheet(true);
                    setShowMenu(false);
                  }}
                  className="w-full px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50"
                >
                  Navigate...
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Scrollable Content */}
      <div className="flex-1 overflow-y-auto">
        {/* Main Content */}
        <div className="p-4">
          <ViewCardContentSection
            viewingCard={viewingCard}
            showingSummary={viewMode === 'summary'}
            showingAnalysis={viewMode === 'analysis'}
            latestSummary={latestSummary}
            analysis={analysis}
            onCreateChildCard={onCreateChildCard}
            categorizedReferences={categorizedReferences}
            onAddBacklink={onAddBacklink}
            setViewCard={setViewCard}
            setError={setError}
            handleOpenEntity={handleOpenEntity}
            summaries={summaries}
            setSelectedFact={setSelectedFact}
            setShowFactDialog={setShowFactDialog}
            fileUploadRef={fileUploadRef}
            onSaveCard={onSaveCard}
          />
        </div>

        {/* Accordion Sections */}
        <div className="border-t border-gray-200">
          {/* Tags - expanded by default */}
          <ViewMobileAccordion
            title="Tags"
            defaultExpanded={true}
            rightElement={
              <SearchTagDropdown
                tags={tags}
                handleTagClick={onTagClick}
              />
            }
          >
            <div className="flex flex-wrap gap-1.5">
              {viewingCard.tags.map((tag) => (
                <span
                  key={tag.name}
                  className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
                >
                  <span
                    className="cursor-pointer hover:bg-purple-100"
                    onClick={() => navigate(`/app/search?term=${encodeURIComponent('#' + tag.name)}`)}
                  >
                    #{tag.name}
                  </span>
                  {viewingCard.body.includes(`#${tag.name}`) && (
                    <button
                      onClick={() => onRemoveTag(tag.name)}
                      className="ml-1.5 text-purple-400 hover:text-purple-600"
                    >
                      ×
                    </button>
                  )}
                </span>
              ))}
            </div>
          </ViewMobileAccordion>

          {/* Navigation */}
          {hasNavigation && (
            <ViewMobileAccordion title="Navigation">
              <div className="space-y-2">
                {parentCard && (
                  <button
                    onClick={() => handleNavigate(parentCard.id)}
                    className="w-full p-2 bg-gray-50 rounded text-left hover:bg-gray-100"
                  >
                    <div className="text-xs text-gray-500">Parent</div>
                    <div className="font-medium">{parentCard.title}</div>
                  </button>
                )}
                <div className="flex gap-2">
                  {prevSibling && (
                    <button
                      onClick={() => handleNavigate(prevSibling.id)}
                      className="flex-1 p-2 bg-gray-50 rounded hover:bg-gray-100"
                    >
                      ← Prev
                    </button>
                  )}
                  {nextSibling && (
                    <button
                      onClick={() => handleNavigate(nextSibling.id)}
                      className="flex-1 p-2 bg-gray-50 rounded hover:bg-gray-100"
                    >
                      Next →
                    </button>
                  )}
                </div>
              </div>
            </ViewMobileAccordion>
          )}

          {/* Linked Entities */}
          {hasEntities && (
            <ViewMobileAccordion title="Linked Entities">
              <ul className="space-y-1">
                {linkedEntities.map(entity => (
                  <li
                    key={entity.id}
                    className="py-1 px-2 hover:bg-gray-50 rounded cursor-pointer"
                    onClick={() => handleOpenEntity(entity)}
                  >
                    <div className="flex items-center gap-2 text-xs">
                      <div className="text-gray-400 shrink-0">
                        <PersonIcon />
                      </div>
                      <span className="text-blue-600">{entity.name}</span>
                      <span className="text-gray-300">-</span>
                      <span className="text-gray-500">{entity.type}</span>
                    </div>
                  </li>
                ))}
              </ul>
            </ViewMobileAccordion>
          )}

          {/* Related Cards */}
          {hasRelatedCards && (
            <ViewMobileAccordion title="Related Cards">
              <RelatedCards
                relatedCards={relatedCards!}
                onCardClick={handleNavigate}
              />
            </ViewMobileAccordion>
          )}

          {/* Source Article */}
          {sourceArticle && (
            <ViewMobileAccordion title="Source Article">
              <button
                onClick={() => navigate('/app/rss', { state: { selectedArticleId: sourceArticle.id } })}
                className="w-full text-left p-2 rounded hover:bg-gray-50"
              >
                <div className="flex items-start gap-2">
                  <svg className="w-4 h-4 text-green-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
                  </svg>
                  <div>
                    <p className="text-sm font-medium text-blue-600">{sourceArticle.title}</p>
                    <p className="text-xs text-gray-500 mt-1">
                      RSS Feed • {new Date(sourceArticle.fetched_at).toLocaleDateString()}
                    </p>
                  </div>
                </div>
              </button>
            </ViewMobileAccordion>
          )}

          {/* Structured Data */}
          {viewingCard.schema_id && viewingCard.structured_data && (
            <ViewMobileAccordion title="Data">
              <CardStructuredDataDisplay
                schemaId={viewingCard.schema_id}
                structuredData={viewingCard.structured_data}
              />
            </ViewMobileAccordion>
          )}

          {/* Details */}
          <ViewMobileAccordion title="Details">
            <div className="text-xs text-gray-600 space-y-1">
              {viewingCard.link && (
                <div className="flex items-start">
                  <span className="font-medium w-20">Link:</span>
                  <div
                    className="flex-1 break-all"
                    dangerouslySetInnerHTML={{ __html: linkifyWithDefaultOptions(viewingCard.link) }}
                  />
                </div>
              )}
              <div className="flex items-start">
                <span className="font-medium w-20">Created:</span>
                <span className="flex-1">{viewingCard.created_at.toLocaleDateString()}</span>
              </div>
              <div className="flex items-start">
                <span className="font-medium w-20">Updated:</span>
                <span className="flex-1">{viewingCard.updated_at.toLocaleDateString()}</span>
              </div>
            </div>
          </ViewMobileAccordion>
        </div>
      </div>

      {/* Navigation Bottom Sheet */}
      <ViewNavigationSheet
        isOpen={showNavSheet}
        onClose={() => setShowNavSheet(false)}
        parentCard={parentCard}
        prevSibling={prevSibling}
        nextSibling={nextSibling}
        viewingCard={viewingCard}
        onNavigate={handleNavigate}
      />
    </div>
  );
}
```

**Step 4: Run test to verify it passes**

Run: `cd zettelkasten-front && npm test -- ViewMobileLayout.test.tsx`
Expected: PASS (5 tests)

**Step 5: Commit**

```bash
git add zettelkasten-front/src/components/cards/ViewMobileLayout.tsx
git add zettelkasten-front/src/components/cards/ViewMobileLayout.test.tsx
git commit -m "feat(cards): add ViewMobileLayout with accordion sections"
```

---

## Task 4: Integrate Mobile Layout into ViewPage

**Files:**
- Modify: `zettelkasten-front/src/pages/cards/ViewPage.tsx`

**Step 1: Add mobile detection to ViewPage**

Read the current ViewPage.tsx and add mobile detection state after the existing useState hooks:

```typescript
// Add after line 33 (after viewMode state)
const [isMobile, setIsMobile] = useState(() => {
  if (typeof window !== 'undefined') {
    return window.innerWidth < 768;
  }
  return false;
});

useEffect(() => {
  const handleResize = () => {
    setIsMobile(window.innerWidth < 768);
  };

  window.addEventListener('resize', handleResize);
  return () => window.removeEventListener('resize', handleResize);
}, []);
```

**Step 2: Add import for ViewMobileLayout**

```typescript
// Add to imports at top of file
import { ViewMobileLayout } from "../../components/cards/ViewMobileLayout";
```

**Step 3: Update return statement to conditionally render**

Replace the return statement to check for mobile:

```typescript
// Mobile layout
if (isMobile && viewingCard) {
  return (
    <ViewMobileLayout
      viewingCard={viewingCard}
      parentCard={parentCard}
      prevSibling={prevSibling}
      nextSibling={nextSibling}
      linkedEntities={linkedEntities}
      categorizedReferences={categorizedReferences}
      summaries={summaries}
      latestSummary={latestSummary}
      analysis={analysis}
      relatedCards={relatedCards}
      tags={tags}
      sourceArticle={viewingCard.source_article}
      onEditCard={onEditCard}
      onCreateChildCard={onCreateChildCard}
      onToggleStar={onToggleStar}
      onTogglePin={onTogglePin}
      onOpenChatSidebar={onOpenChatSidebar}
      toggleCreateTaskWindow={toggleCreateTaskWindow}
      onTagClick={onTagClick}
      onRemoveTag={onRemoveTag}
      onAddBacklink={onAddBacklink}
      handleOpenEntity={handleOpenEntity}
      onResummarize={onResummarize}
      onRecategorize={onRecategorize}
      refreshCard={refreshCard}
      setViewCard={setViewCard}
      setError={setError}
      setSelectedFact={setSelectedFact}
      setShowFactDialog={setShowFactDialog}
      fileUploadRef={fileUploadRef}
      onSaveCard={handleSaveCard}
    />
  );
}

// Desktop layout (existing code)
return (
  // ... existing return content
);
```

**Step 4: Run tests to verify nothing broke**

Run: `cd zettelkasten-front && npm test`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add zettelkasten-front/src/pages/cards/ViewPage.tsx
git commit -m "feat(cards): integrate ViewMobileLayout into ViewPage"
```

---

## Task 5: Manual Testing & Polish

**Files:**
- None (manual testing)

**Step 1: Start development server**

Run: `cd zettelkasten-front && npm start`

**Step 2: Test mobile view in browser**

1. Open browser dev tools
2. Toggle device toolbar (mobile emulation)
3. Navigate to a card page
4. Verify:
   - Top bar shows card title and menu
   - Main content is visible
   - Tags section is expanded by default
   - Other sections are collapsed
   - Tapping sections expands/collapses them
   - Menu shows view mode options
   - Navigate option opens bottom sheet

**Step 3: Test navigation sheet**

1. Open a card with parent/siblings
2. Click menu → Navigate
3. Verify:
   - Bottom sheet opens
   - Parent card button works
   - Prev/Next buttons work
   - Children list appears if card has children
   - Swipe down closes sheet

**Step 4: Test view modes**

1. Open menu
2. Select different view modes (Tree, Summary, Analysis)
3. Verify content changes appropriately

**Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix(cards): polish ViewMobileLayout interactions"
```

---

## Task 6: Push to Remote

**Step 1: Verify all changes**

Run: `git status`
Expected: All changes committed

**Step 2: Push to remote**

Run: `git push origin master`

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | ViewMobileAccordion component | 2 new |
| 2 | ViewNavigationSheet component | 2 new |
| 3 | ViewMobileLayout component | 2 new |
| 4 | Integrate into ViewPage | 1 modified |
| 5 | Manual testing | - |
| 6 | Push to remote | - |

**Total:** 6 new files, 1 modified file
