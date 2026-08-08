import "@testing-library/jest-dom";
import { afterAll, afterEach, beforeAll, vi } from "vitest";
import { act } from "@testing-library/react";
import DOMPurify from "dompurify";
import {
  defaultFetchMock,
  mockEndpoint,
  resetMockEndpoints,
} from "./fetchMock";

// Suppress known happy-dom bug: DOMParser triggers an unhandled rejection
// from HTMLIFrameElement internals ("Cannot read properties of null (reading 'console')")
// See: https://github.com/nicedoc/html-encoding-sniffer/issues/1
process.on("unhandledRejection", (reason) => {
  if (
    reason instanceof TypeError &&
    reason.message.includes("reading 'console'") &&
    reason.stack?.includes("HTMLIFrameElement")
  ) {
    return;
  }
  throw reason;
});

// Initialize DOMPurify with the test environment's window
if (typeof window !== "undefined") {
  DOMPurify.addHook("uponSanitizeAttribute", (node, data) => {
    // Additional security hook for attributes
    if (data.attrName === "href" && data.attrValue?.toLowerCase().startsWith("javascript:")) {
      data.attrValue = "";
    }
  });
}

// ---------------------------------------------------------------------------
// Fetch mock
// ---------------------------------------------------------------------------
// Many components mount application contexts which fetch from the backend.
// We provide EXPLICIT defaults for the endpoints the shared provider stack
// (AuthProvider/TaskProvider/TagProvider/StatusProvider) hits on mount.
// Any other request with no registered handler FAILS LOUDLY (see fetchMock.ts)
// so broken API wiring surfaces as a test failure rather than fake success.
const originalFetch = globalThis.fetch;

beforeAll(() => {
  // Defaults for the shared provider stack mounted by renderWithProviders.
  mockEndpoint("/task-statuses", []);
  mockEndpoint("/tags", []);
  mockEndpoint("/tasks?", { tasks: [], total: 0, limit: 100, offset: 0 });

  globalThis.fetch = vi.fn(defaultFetchMock) as unknown as typeof fetch;
});

afterEach(() => {
  resetMockEndpoints();
});

afterAll(() => {
  globalThis.fetch = originalFetch;
});
