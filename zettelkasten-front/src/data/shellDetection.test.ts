/**
 * Shell detection + server-URL resolution (epic Zettelgarden-v5b, Phase 3a —
 * issue c6l.4). Verifies the web app stays a thin client (no shell globals),
 * both native shells are detected, and resolveBaseUrl honors shell settings
 * over the build-time VITE_URL default.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  isDesktopApp,
  isMobileApp,
  isNativeShell,
  waitForShellReady,
} from './tauriStorageAdapter';
import { normalizeServerUrl, resolveBaseUrl } from './syncClient';

describe('shell detection', () => {
  const originalWindow = { ...(globalThis as any).window };

  afterEach(() => {
    // Restore the window shape (delete the injected shell globals).
    const w = globalThis as any;
    delete w.zgDesktop;
    delete w.zgMobile;
    delete w.__TAURI_INTERNALS__;
  });

  it('returns false on plain web (no shell globals)', () => {
    expect(isDesktopApp()).toBe(false);
    expect(isMobileApp()).toBe(false);
    expect(isNativeShell()).toBe(false);
  });

  it('detects the Tauri desktop shell (zgDesktop + internals invoke)', () => {
    const w = globalThis as any;
    w.zgDesktop = { ready: Promise.resolve() };
    w.__TAURI_INTERNALS__ = { invoke: vi.fn() };
    expect(isDesktopApp()).toBe(true);
    expect(isMobileApp()).toBe(false);
    expect(isNativeShell()).toBe(true);
  });

  it('detects the RN mobile shell (zgMobile)', () => {
    (globalThis as any).zgMobile = { ready: Promise.resolve() };
    expect(isMobileApp()).toBe(true);
    expect(isDesktopApp()).toBe(false);
    expect(isNativeShell()).toBe(true);
  });

  it('waitForShellReady resolves without a shell (web)', async () => {
    await expect(waitForShellReady()).resolves.toBeUndefined();
  });

  it('waitForShellReady resolves after the mobile bridge primes', async () => {
    let resolveReady!: () => void;
    (globalThis as any).zgMobile = {
      ready: new Promise<void>((r) => (resolveReady = r)),
    };
    const waited = waitForShellReady();
    resolveReady();
    await expect(waited).resolves.toBeUndefined();
  });
});

describe('normalizeServerUrl', () => {
  it('strips a trailing /api suffix', () => {
    expect(normalizeServerUrl('http://localhost:8079/api')).toBe(
      'http://localhost:8079',
    );
  });

  it('strips a trailing slash', () => {
    expect(normalizeServerUrl('http://10.0.2.2:8079/')).toBe(
      'http://10.0.2.2:8079',
    );
  });

  it('leaves a bare server root untouched', () => {
    expect(normalizeServerUrl('https://notes.example.com')).toBe(
      'https://notes.example.com',
    );
  });
});

describe('resolveBaseUrl', () => {
  beforeEach(() => {
    // Fresh window each test: no shell settings by default.
    const w = globalThis as any;
    delete w.zgMobile;
    delete w.__TAURI_INTERNALS__;
  });

  it('falls back to the build-time VITE_URL (web thin client)', async () => {
    const viteUrl = (import.meta as any).env?.VITE_URL as string | undefined;
    expect(viteUrl).toBeTruthy(); // test harness sets VITE_URL
    await expect(resolveBaseUrl()).resolves.toBe(
      normalizeServerUrl(viteUrl as string),
    );
  });

  it('prefers the mobile shell settings over VITE_URL', async () => {
    (globalThis as any).zgMobile = {
      ready: Promise.resolve(),
      loadSettings: async () => ({
        serverUrl: 'https://notes.example.com/api',
      }),
    };
    await expect(resolveBaseUrl()).resolves.toBe('https://notes.example.com');
  });

  it('normalizes a trailing slash on the settings URL', async () => {
    (globalThis as any).zgMobile = {
      ready: Promise.resolve(),
      loadSettings: async () => ({ serverUrl: 'http://10.0.2.2:8079/' }),
    };
    await expect(resolveBaseUrl()).resolves.toBe('http://10.0.2.2:8079');
  });
});
