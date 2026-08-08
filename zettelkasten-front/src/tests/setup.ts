import '@testing-library/jest-dom';
import { afterAll, beforeAll, vi } from 'vitest';
import DOMPurify from 'dompurify';

// Suppress known happy-dom bug: DOMParser triggers an unhandled rejection
// from HTMLIFrameElement internals ("Cannot read properties of null (reading 'console')")
// See: https://github.com/nicedoc/html-encoding-sniffer/issues/1
process.on('unhandledRejection', (reason) => {
  if (
    reason instanceof TypeError &&
    reason.message.includes("reading 'console'") &&
    reason.stack?.includes('HTMLIFrameElement')
  ) {
    return;
  }
  throw reason;
});

// Initialize DOMPurify with the test environment's window
if (typeof window !== 'undefined') {
  DOMPurify.addHook('uponSanitizeAttribute', (node, data) => {
    // Additional security hook for attributes
    if (
      data.attrName === 'href' &&
      data.attrValue?.toLowerCase().startsWith('javascript:')
    ) {
      data.attrValue = '';
    }
  });
}

// Many components mount application contexts which fetch from the backend.
// In tests we provide a safe default fetch mock to avoid cross-origin/network
// failures causing unhandled rejections.
const originalFetch = globalThis.fetch;

beforeAll(() => {
  globalThis.fetch = vi.fn(async (input: any) => {
    const url: string =
      typeof input === 'string'
        ? input
        : input?.url ??
          (typeof input?.toString === 'function' ? input.toString() : '');

    if (url.includes('/task-statuses')) {
      return {
        ok: true,
        status: 200,
        json: async () => [],
        text: async () => '',
      } as any;
    }

    if (url.includes('/tags')) {
      return {
        ok: true,
        status: 200,
        json: async () => [],
        text: async () => '',
      } as any;
    }

    if (url.includes('/tasks?')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ tasks: [], total: 0, limit: 100, offset: 0 }),
        text: async () => '',
      } as any;
    }

    return {
      ok: true,
      status: 200,
      json: async () => ({}),
      text: async () => '',
    } as any;
  });
});

afterAll(() => {
  globalThis.fetch = originalFetch;
});
