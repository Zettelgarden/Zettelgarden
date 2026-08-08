/**
 * Shared fetch mock handlers for tests.
 *
 * Explicit, per-endpoint handlers that tests can register via
 * `mockEndpoint(...)`. Requests that match NO handler FAIL LOUDLY so that
 * broken API wiring surfaces as a test failure instead of silently
 * succeeding with `{}` (the old behavior, which masked bugs).
 */

import { vi } from 'vitest';

export interface MockResponse {
  ok: boolean;
  status: number;
  json: () => Promise<unknown>;
  text: () => Promise<string>;
}

export function jsonResponse(data: unknown, status = 200): MockResponse {
  return {
    ok: status >= 200 && status < 400,
    status,
    json: async () => data,
    text: async () => JSON.stringify(data),
  };
}

interface EndpointHandler {
  pattern: string | RegExp;
  respond: (url: string) => MockResponse;
}

const handlers: EndpointHandler[] = [];

/**
 * Register a handler that responds to any URL matching `pattern`
 * (substring match for strings, regex match for RegExp).
 */
export function mockEndpoint(
  pattern: string | RegExp,
  data: unknown,
  status = 200,
): void {
  handlers.push({
    pattern,
    respond: () => jsonResponse(data, status),
  });
}

/**
 * Register a custom responder for a URL pattern (for stateful handlers).
 */
export function mockEndpointWith(
  pattern: string | RegExp,
  respond: (url: string) => MockResponse,
): void {
  handlers.push({ pattern, respond });
}

export function resetMockEndpoints(): void {
  handlers.length = 0;
}

function urlMatches(pattern: string | RegExp, url: string): boolean {
  if (pattern instanceof RegExp) {
    return pattern.test(url);
  }
  return url.includes(pattern);
}

/**
 * The default fetch implementation installed by setup.ts.
 *
 * - Matches registered endpoints (see mockEndpoint).
 * - Otherwise FAILS LOUDLY: tests that forget to mock an endpoint the
 *   component actually hits get a clear error instead of fake success.
 */
export async function defaultFetchMock(
  input: RequestInfo | URL,
): Promise<Response> {
  const url: string =
    typeof input === 'string'
      ? input
      : typeof input?.toString === 'function'
      ? input.toString()
      : '';

  const handler = handlers.find((h) => urlMatches(h.pattern, url));

  if (handler) {
    return handler.respond(url) as unknown as Response;
  }

  throw new Error(
    `[test-fetch-mock] No mock registered for URL: ${url}. ` +
      `Register an endpoint with mockEndpoint(pattern, data) in your test, ` +
      `or add an explicit handler in src/tests/setup.ts.`,
  );
}

/**
 * Convenience factory for the fetch mock used in tests that stub fetch
 * directly (returns a vi.fn() pre-wired to defaultFetchMock).
 */
export function createFetchMock() {
  return vi.fn(defaultFetchMock);
}
