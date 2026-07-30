import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { SchemaPage } from "./SchemaPage";
import { SchemaDefinition } from "../models/Schema";

vi.mock("../api/schemas", () => ({
  fetchSchemas: vi.fn(),
  deleteSchema: vi.fn(),
}));

const { fetchSchemas } = await import("../api/schemas");

const { mockNavigate } = vi.hoisted(() => ({ mockNavigate: vi.fn() }));
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const now = new Date();

function makeSchema(overrides: Partial<SchemaDefinition>): SchemaDefinition {
  return {
    id: 1,
    name: "Schema",
    slug: "schema",
    owner_id: 1,
    fields: [{ name: "title", type: "text", required: true }],
    created_at: now,
    updated_at: now,
    is_deleted: false,
    card_count: 0,
    ...overrides,
  };
}

const schemas: SchemaDefinition[] = [
  makeSchema({
    id: 1,
    name: "Book Review",
    slug: "book-review",
    fields: [
      { name: "Author", type: "text", required: true },
      { name: "Rating", type: "number", required: false },
    ],
    card_count: 5,
  }),
  makeSchema({
    id: 2,
    name: "Movies",
    slug: "movies",
    fields: [{ name: "Director", type: "text", required: false }],
    card_count: 0,
  }),
];

function renderPage() {
  return render(
    <MemoryRouter>
      <SchemaPage />
    </MemoryRouter>,
  );
}

describe("SchemaPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (fetchSchemas as ReturnType<typeof vi.fn>).mockResolvedValue(schemas);
  });

  it("renders schemas with their card counts", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Book Review")).toBeInTheDocument());
    expect(screen.getByText("5 cards")).toBeInTheDocument();
    expect(screen.getByText("0 cards")).toBeInTheDocument();
  });

  it("filters schemas by name search", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Book Review")).toBeInTheDocument());

    fireEvent.change(screen.getByPlaceholderText(/search schemas/i), {
      target: { value: "movie" },
    });

    expect(screen.queryByText("Book Review")).not.toBeInTheDocument();
    expect(screen.getByText("Movies")).toBeInTheDocument();
  });

  it("navigates to the table view when a row is clicked", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText("Book Review")).toBeInTheDocument());

    fireEvent.click(screen.getByText("Book Review"));

    expect(mockNavigate).toHaveBeenCalledWith("/app/schemas/1/table");
  });

  it("shows the empty state with the explanatory blurb when there are no schemas", async () => {
    (fetchSchemas as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(/create your first schema/i)).toBeInTheDocument(),
    );
    expect(screen.getByText(/define custom data structures/i)).toBeInTheDocument();
  });
});
