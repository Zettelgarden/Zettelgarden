import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmailDetailPage } from "./EmailDetailPage";
import { MemoryRouter, Route, Routes } from "react-router-dom";
// Mock the auth context
vi.mock("../contexts/AuthContext", () => ({
  useAuth: () => ({
    isAuthenticated: true,
    isLoading: false,
    isAdmin: false,
    hasSubscription: false,
    loginUser: vi.fn(),
    loginUserFromToken: vi.fn(),
    logoutUser: vi.fn(),
    currentUser: { id: 1, username: "testuser" },
    user: { id: 1, username: "testuser" },
    updateUser: vi.fn(),
  }),
}));

// Mock the email API
vi.mock("../api/email", () => ({
  getEmail: vi.fn(() => Promise.resolve({
    id: 1,
    subject: "Test Email",
    from_address: "test@example.com",
    from_name: "Test Sender",
    to_addresses: "recipient@example.com",
    body_html: null,
    body_text: "Plain text email",
    received_at: "2026-02-26T00:00:00.000Z",
    status: "unprocessed",
    is_read: false,
  })),
  updateEmailStatus: vi.fn(() => Promise.resolve({
    id: 1,
    status: "archived",
  })),
  Email: {},
}));

// Mock the document title utility
vi.mock("../utils/title", () => ({
  setDocumentTitle: vi.fn(),
}));

// Mock the emailHtml utilities for component tests
vi.mock("../utils/emailHtml", () => ({
  processEmailHtml: vi.fn((html: string) => html),
}));

// Mock the CreateTaskWindow component
vi.mock("../components/tasks/CreateTaskWindow", () => ({
  CreateTaskWindow: () => <div data-testid="create-task-window">Create Task Window</div>,
}));

// Mock the EmailConvertDialog component
vi.mock("../components/email/EmailConvertDialog", () => ({
  EmailConvertDialog: () => <div data-testid="email-convert-dialog">Email Convert Dialog</div>,
}));

function renderWithRouter(component: React.ReactElement) {
  return render(
    <MemoryRouter initialEntries={["/app/emails/1"]}>
      <Routes>
        <Route path="/app/emails/:id" element={component} />
      </Routes>
    </MemoryRouter>
  );
}

describe("EmailDetailPage", () => {
  it("renders email loading state", () => {
    renderWithRouter(<EmailDetailPage />);
    expect(screen.getByText("Loading email...")).toBeInTheDocument();
  });

  it("renders email content after loading", async () => {
    renderWithRouter(<EmailDetailPage />);

    // Wait for the email to load
    const emailContent = await screen.findByText("Plain text email");
    expect(emailContent).toBeInTheDocument();
  });

  it("displays back button", async () => {
    renderWithRouter(<EmailDetailPage />);

    // Wait for loading to complete and back button to appear
    const backButton = await screen.findByText(/Back to Inbox/);
    expect(backButton).toBeInTheDocument();
  });

  it("displays email metadata", async () => {
    renderWithRouter(<EmailDetailPage />);

    expect(await screen.findByText("Test Email")).toBeInTheDocument();
    expect(await screen.findByText(/Test Sender/)).toBeInTheDocument();
    expect(await screen.findByText(/test@example.com/)).toBeInTheDocument();
  });
});

// Note: HTML sanitization tests are now in src/utils/emailHtml.test.ts
// to properly test the emailHtml utilities without module mocking issues
