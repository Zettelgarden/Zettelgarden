import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import {
  TagsList,
  DetailsList,
  SourceArticleLink,
} from "./SideMetadataSections";
import { defaultCard } from "../../models/Card";
import type { Card } from "../../models/Card";
import type { RSSArticle } from "../../api/rss";

// ViewPageSidePanels / ViewMobileLayout render these against the real router,
// so we do the same rather than mocking react-router-dom.
function wrap(ui: React.ReactNode) {
  return render(<BrowserRouter>{ui}</BrowserRouter>);
}

function withCard(overrides: Partial<Card>): Card {
  return { ...defaultCard, ...overrides };
}

describe("TagsList", () => {
  it("renders a pill for each tag with a # prefix", () => {
    const card = withCard({
      body: "has #alpha inline",
      tags: [
        { id: 1, name: "alpha", color: "black", user_id: 1 },
        { id: 2, name: "beta", color: "black", user_id: 1 },
      ],
    });
    wrap(<TagsList card={card} onRemoveTag={vi.fn()} />);
    expect(screen.getByText("#alpha")).toBeInTheDocument();
    expect(screen.getByText("#beta")).toBeInTheDocument();
  });

  it("shows the × remove button only for tags present in the body and calls onRemoveTag", () => {
    const onRemoveTag = vi.fn();
    const card = withCard({
      body: "keeps #alpha",
      tags: [
        { id: 1, name: "alpha", color: "black", user_id: 1 },
        { id: 2, name: "beta", color: "black", user_id: 1 },
      ],
    });
    wrap(<TagsList card={card} onRemoveTag={onRemoveTag} />);
    // alpha is in the body → remove button present.
    const removeButtons = screen.getAllByRole("button");
    expect(removeButtons).toHaveLength(1);
    fireEvent.click(removeButtons[0]);
    expect(onRemoveTag).toHaveBeenCalledWith("alpha");
  });

  it("renders nothing when there are no tags", () => {
    const { container } = wrap(
      <TagsList card={withCard({ tags: [] })} onRemoveTag={vi.fn()} />,
    );
    expect(container.querySelector("span")).toBeNull();
  });
});

describe("DetailsList", () => {
  it("renders the ID row and locale-formatted created/updated dates", () => {
    const created = new Date("2024-01-15T10:00:00Z");
    const card = withCard({
      card_id: "A.1",
      created_at: created,
      updated_at: created,
      link: "",
    });
    wrap(<DetailsList card={card} />);
    expect(screen.getByText("[A.1]")).toBeInTheDocument();
    // created + updated share the same date here → both rows render it.
    expect(screen.getAllByText(created.toLocaleDateString())).toHaveLength(2);
  });

  it("shows the link row only when a link is present", () => {
    const { rerender } = render(
      <BrowserRouter>
        <DetailsList card={withCard({ link: "" })} />
      </BrowserRouter>,
    );
    expect(screen.queryByText("Link:")).toBeNull();

    rerender(
      <BrowserRouter>
        <DetailsList card={withCard({ link: "https://example.com" })} />
      </BrowserRouter>,
    );
    expect(screen.getByText("Link:")).toBeInTheDocument();
  });
});

describe("SourceArticleLink", () => {
  const article = {
    id: 42,
    user_id: 1,
    feed_id: 7,
    title: "A Great Article",
    url: "https://example.com/a",
    fetched_at: "2024-03-01T12:00:00Z",
    read: false,
  } as RSSArticle;

  it("renders the title and an RSS Feed line with a locale date", () => {
    wrap(<SourceArticleLink sourceArticle={article} />);
    expect(screen.getByText("A Great Article")).toBeInTheDocument();
    expect(
      screen.getByText(
        `RSS Feed • ${new Date(article.fetched_at).toLocaleDateString()}`,
      ),
    ).toBeInTheDocument();
  });
});
