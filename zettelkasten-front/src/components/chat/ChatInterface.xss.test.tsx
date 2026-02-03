import React from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ChatInterface } from "./ChatInterface";

function makeChatHook(overrides: Partial<any> = {}): any {
  return {
    messages: [],
    messageInput: "",
    isSending: false,
    error: null,
    selectedModel: "openai/gpt-5",
    collapsedToolResults: new Set<string>(),
    setMessageInput: vi.fn(),
    setSelectedModel: vi.fn(),
    sendMessage: vi.fn(),
    handleCardReference: vi.fn(),
    toggleToolResult: vi.fn(),
    ...overrides,
  };
}

describe("ChatInterface markdown sanitization", () => {
  it("removes javascript: URLs from assistant markdown links", () => {
    const messages = [
      {
        id: "m1",
        conversation_id: "c1",
        role: "assistant",
        content: "[click me](javascript:alert(1))\n\nNormal: [ok](https://example.com)",
        sequence_number: 1,
        status: "completed",
        created_at: "2026-01-19T00:00:00.000Z",
      },
    ];

    render(
      <ChatInterface
        chatHook={makeChatHook({ messages })}
        onCardClick={vi.fn()}
        onTaskClick={vi.fn()}
      />
    );

    const evilAnchor = screen.getByText("click me").closest("a");
    expect(evilAnchor).not.toBeNull();
    const evilHref = evilAnchor?.getAttribute("href") ?? "";
    expect(evilHref.toLowerCase()).not.toContain("javascript:");
    expect(evilAnchor).not.toHaveAttribute("href", expect.stringMatching(/^javascript:/i));
    // If sanitized, the href is typically removed entirely (or otherwise made safe).
    // We only assert that no javascript: URL survives.


    const okLink = screen.getByRole("link", { name: "ok" });
    expect(okLink.getAttribute("href")).toBe("https://example.com");
  });

  it("does not render script tags from assistant content", () => {
    const messages = [
      {
        id: "m2",
        conversation_id: "c1",
        role: "assistant",
        content: "Here is some html: <script>alert('xss')</script>",
        sequence_number: 1,
        status: "completed",
        created_at: "2026-01-19T00:00:00.000Z",
      },
    ];

    const { container } = render(
      <ChatInterface
        chatHook={makeChatHook({ messages })}
        onCardClick={vi.fn()}
        onTaskClick={vi.fn()}
      />
    );

    expect(container.querySelector("script")).toBeNull();
    expect(screen.getByText(/Here is some html/i)).toBeInTheDocument();
  });
});
