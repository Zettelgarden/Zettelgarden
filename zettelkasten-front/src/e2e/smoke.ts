/**
 * Desktop E2E smoke (Zettelgarden-77j): a scripted scenario that runs INSIDE
 * the real Tauri webview when the orchestrator injects the E2E bridge
 * (ZG_E2E=1). Exercises the full shipped stack — real webview, preload shim,
 * keychain/file-keychain IPC, Rust sync_db commands, sync engine, data
 * provider, and the rendered UI (login form, sidebar indicator).
 *
 * Scenarios:
 *   fresh   — register + real login form, fresh-mirror bootstrap, online
 *             create+sync, OFFLINE create/edit/delete card+task (asserting
 *             ZERO window.fetch calls on the hot path), reconnect +
 *             reconciliation, server spot-check, indicator states.
 *   session — relaunch with the same app-data dir + keychain: the app must
 *             boot authenticated from the keychain and render mirror data
 *             instantly (no login form).
 */

import { getSyncClient } from '../data/syncClient';
import { apiClient } from '../api/client';
import { saveNewCard, saveExistingCard, deleteCard } from '../api/cards';
import { saveNewTask, saveExistingTask } from '../api/tasks';

declare global {
  interface Window {
    __zgE2E?: {
      scenario: string;
      fetchCount(): number;
      resetFetchCount(): void;
      report(entry: Record<string, unknown>): Promise<void>;
      done(ok: boolean, summary: Record<string, unknown>): Promise<void>;
    };
  }
}

interface E2E {
  scenario: string;
  report: (entry: Record<string, unknown>) => Promise<void>;
  done: (ok: boolean, summary: Record<string, unknown>) => Promise<void>;
  fetchCount: () => number;
  resetFetchCount: () => void;
}

function getE2E(): E2E {
  const e = window.__zgE2E;
  if (!e) throw new Error('E2E bridge not injected');
  return e;
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** Polls until fn() is truthy or the timeout elapses. */
async function waitFor(
  label: string,
  fn: () => boolean | Promise<boolean>,
  timeoutMs = 60_000,
  intervalMs = 250,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    let v: boolean;
    try {
      v = await fn();
    } catch {
      v = false;
    }
    if (v) return;
    if (Date.now() > deadline) throw new Error(`timeout waiting for ${label}`);
    await sleep(intervalMs);
  }
}

function setInput(selector: string, value: string): void {
  const el = document.querySelector(selector) as HTMLInputElement | null;
  if (!el) throw new Error(`input not found: ${selector}`);
  const setter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    'value',
  )?.set;
  setter?.call(el, value);
  el.dispatchEvent(new Event('input', { bubbles: true }));
  el.dispatchEvent(new Event('change', { bubbles: true }));
}

function indicatorLabel(): string | null {
  const el = document.querySelector('[aria-label]') as HTMLElement | null;
  const all = Array.from(document.querySelectorAll('[aria-label]'));
  const indicator = all.find((n) => {
    const label = n.getAttribute('aria-label') ?? '';
    return (
      label.includes('Synced') ||
      label.includes('Syncing') ||
      label.includes('Offline') ||
      label.includes('pending') ||
      label.includes('Sync error')
    );
  });
  return indicator?.getAttribute('aria-label') ?? null;
}

async function dumpDom(e2e: E2E): Promise<void> {
  try {
    const body = document.body?.innerHTML ?? '';
    await e2e.report({
      step: 'dom-dump',
      ok: true,
      length: body.length,
      snippet: body.slice(0, 600),
      labels: Array.from(document.querySelectorAll('[aria-label]')).map((n) =>
        n.getAttribute('aria-label'),
      ),
      text: (document.body?.innerText ?? '').slice(0, 300),
    });
  } catch (err) {
    await e2e.report({ step: 'dom-dump', ok: false, message: String(err) });
  }
}

async function loginViaForm(email: string, password: string): Promise<void> {
  await waitFor(
    'login form',
    () => !!document.querySelector('input[name="email"]'),
    90_000,
  );
  setInput('input[name="email"]', email);
  setInput('input[name="password"]', password);
  (
    document.querySelector('button[type="submit"]') as HTMLButtonElement | null
  )?.click();
}

async function registerUser(email: string, password: string): Promise<void> {
  const res = await apiClient.post('/users', {
    username: `e2e-${Date.now()}`,
    email,
    password,
    confirmPassword: password,
  });
  if (res.response.status >= 400) {
    throw new Error(`register failed: ${res.response.status}`);
  }
}

function cardData(title: string, body: string) {
  return {
    id: 0,
    card_id: '',
    user_id: 0,
    title,
    body,
    link: '',
    is_deleted: false,
    created_at: new Date(),
    updated_at: new Date(),
    parent_id: 0,
    parent: null as any,
    files: [],
    children: [],
    references: [],
    tags: [],
    tasks: [],
    entities: [],
  } as any;
}

async function runFresh(e2e: E2E): Promise<void> {
  await e2e.report({ step: 'boot', ok: true });

  // Diagnostic probe: raw fetch vs apiClient, and the base URL in play.
  try {
    const r = await fetch((import.meta as any).env?.VITE_URL + '/settings');
    await e2e.report({ step: 'probe-raw-fetch', ok: true, status: r.status });
  } catch (err) {
    await e2e.report({
      step: 'probe-raw-fetch',
      ok: false,
      message: String(err),
    });
  }
  try {
    const r = await fetch('https://example.com');
    await e2e.report({ step: 'probe-https', ok: true, status: r.status });
  } catch (err) {
    await e2e.report({ step: 'probe-https', ok: false, message: String(err) });
  }
  try {
    const xhr = await new Promise<number>((resolve, reject) => {
      const x = new XMLHttpRequest();
      x.open('GET', (import.meta as any).env?.VITE_URL + '/settings');
      x.onload = () => resolve(x.status);
      x.onerror = () => reject(new Error('xhr error'));
      x.send();
    });
    await e2e.report({ step: 'probe-xhr', ok: true, status: xhr });
  } catch (err) {
    await e2e.report({ step: 'probe-xhr', ok: false, message: String(err) });
  }
  try {
    const r = await apiClient.get('/settings');
    await e2e.report({
      step: 'probe-apiclient',
      ok: true,
      status: r.response.status,
    });
  } catch (err) {
    await e2e.report({
      step: 'probe-apiclient',
      ok: false,
      message: String(err),
    });
  }
  await e2e.report({
    step: 'env',
    ok: true,
    online: navigator.onLine,
    baseUrl: (import.meta as any).env?.VITE_URL,
    href: window.location.href,
    secureContext: window.isSecureContext,
  });

  const email = `e2e-${Date.now()}@zettelgarden.test`;
  const password = 'e2e-pass-123!';
  await registerUser(email, password);
  await e2e.report({ step: 'registered', ok: true, email });

  await loginViaForm(email, password);
  await sleep(2500); // let loginUser + keychain write settle
  const tokenAfterLogin = localStorage.getItem('token');
  await e2e.report({
    step: 'login-submitted',
    ok: true,
    tokenPresent: tokenAfterLogin !== null && tokenAfterLogin.length > 0,
    tokenPrefix: tokenAfterLogin ? tokenAfterLogin.slice(0, 12) : null,
    storageLength: localStorage.length,
  });

  // MainApp + the sync indicator must render (desktop mode).
  // Watch the route transition after login (the app can briefly bounce back
  // to /login if the token read races the keychain shim).
  const routeLog: string[] = [];
  const routeWatch = setInterval(() => {
    routeLog.push(
      `${location.hash || '/'}|${!!document.querySelector(
        'input[name="email"]',
      )}|${indicatorLabel() ?? '-'}`,
    );
  }, 3000);
  try {
    await waitFor(
      'app shell',
      () => !!document.querySelector('aside') || indicatorLabel() !== null,
      45_000,
    );
  } catch (err) {
    await e2e.report({
      step: 'route-log',
      ok: true,
      routeLog: routeLog.slice(0, 15),
    });
    await dumpDom(e2e);
    throw err;
  } finally {
    clearInterval(routeWatch);
  }
  await waitFor('sync indicator', () => indicatorLabel() !== null, 30_000);
  await e2e.report({
    step: 'app-rendered',
    ok: true,
    indicator: indicatorLabel(),
  });

  // Fresh-mirror bootstrap through the real IPC: the welcome card the server
  // seeds must appear in the local mirror.
  const client = await getSyncClient();
  if (!client) throw new Error('sync client not available');
  await waitFor(
    'mirror bootstrap',
    async () => (await client.engine.query('cards')).length >= 1,
    60_000,
  );
  const seeded = (await client.engine.query('cards')).length;
  await e2e.report({ step: 'bootstrap-ok', ok: true, seededCards: seeded });

  // Online sanity: create a card and sync it.
  const onlineCard = await saveNewCard(
    cardData('E2E online card', 'online body'),
  );
  await client.syncNow();
  if ((await client.engine.pendingChanges()) !== 0) {
    throw new Error('online card did not sync');
  }
  await e2e.report({
    step: 'online-create-synced',
    ok: true,
    id: onlineCard.id,
  });

  // ---- offline phase ----
  await e2e.report({ step: 'offline-begin', ok: true });
  e2e.resetFetchCount();
  await client.engine.setOnline(false);
  await waitFor(
    'offline indicator',
    () => (indicatorLabel() ?? '').includes('Offline'),
    15_000,
  );

  // Create/edit/delete a card + create/edit a task with NO network.
  const a = await saveNewCard(cardData('E2E offline card A', 'body A'));
  const edited = await saveExistingCard({
    ...a,
    title: 'E2E offline card A (edited)',
  });
  const b = await saveNewCard(cardData('E2E offline card B', 'body B'));
  await deleteCard(b.id);
  const t = await saveNewTask({
    ...emptyTask(),
    title: 'E2E offline task',
    card_pk: 0,
  });
  await saveExistingTask({ ...t, title: 'E2E offline task (edited)' });

  const pending = await client.engine.pendingChanges();
  const fetches = e2e.fetchCount();
  if (fetches !== 0)
    throw new Error(`offline hot path made ${fetches} fetch() calls`);
  // Three distinct rows (edited A, deleted B, edited task T) coalesce to 3.
  if (pending < 3)
    throw new Error(`expected >=3 pending changes, got ${pending}`);

  // Mirror state: edited card present, deleted card gone, edited task present.
  const cards = await client.engine.query('cards');
  const cardAUuid = cards.find(
    (r) => r.data.title === 'E2E offline card A (edited)',
  );
  const cardBGone = !cards.some((r) => r.data.title === 'E2E offline card B');
  const tasks = await client.engine.query('tasks');
  const taskEdited = tasks.some(
    (r) => r.data.title === 'E2E offline task (edited)',
  );
  if (!cardAUuid || !cardBGone || !taskEdited) {
    throw new Error(
      `mirror state mismatch: cards=${cards.map(
        (r) => r.data.title,
      )} tasks=${tasks.map((r) => r.data.title)}`,
    );
  }
  await waitFor(
    'pending indicator',
    () => (indicatorLabel() ?? '').includes('pending'),
    15_000,
  );
  await e2e.report({
    step: 'offline-ok',
    ok: true,
    pending,
    fetchCount: fetches,
    indicator: indicatorLabel(),
  });

  // ---- reconnect + reconcile ----
  await client.engine.setOnline(true);
  await client.syncNow();
  await waitFor(
    'outbox drain',
    async () => (await client.engine.pendingChanges()) === 0,
    60_000,
  );
  await waitFor(
    'synced indicator',
    () => (indicatorLabel() ?? '').includes('Synced'),
    15_000,
  );

  // Server spot-check via REST: the offline-edited card must exist server-side
  // with the edited title (proves the push actually landed). Re-read the
  // mirror (the offline-phase `cards` array above is pre-sync).
  const freshCards = await client.engine.query('cards');
  const serverCard = freshCards.find(
    (r) => r.data.title === 'E2E offline card A (edited)',
  );
  const serverId = serverCard?.data.id;
  if (!serverId) {
    throw new Error('edited card missing from mirror after sync');
  }
  const { data: serverCardData } = await apiClient.get<any>(
    `/cards/${serverId}`,
  );
  if (serverCardData.title !== 'E2E offline card A (edited)') {
    throw new Error(`server card title mismatch: ${serverCardData.title}`);
  }
  await e2e.report({
    step: 'reconcile-ok',
    ok: true,
    serverId,
    indicator: indicatorLabel(),
  });
}

async function runSession(e2e: E2E): Promise<void> {
  await e2e.report({ step: 'boot', ok: true });

  // Must boot AUTHENTICATED from the keychain: no login form, app shell shows.
  await sleep(2000);
  const loginVisible = !!document.querySelector('input[name="email"]');
  if (loginVisible)
    throw new Error('relaunch showed the login form (keychain session lost)');
  await waitFor(
    'app shell',
    () => !!document.querySelector('aside') || indicatorLabel() !== null,
    90_000,
  );

  const client = await getSyncClient();
  if (!client) throw new Error('sync client not available');
  await waitFor(
    'mirror data',
    async () => (await client.engine.query('cards')).length >= 1,
    60_000,
  );
  const cards = await client.engine.query('cards');
  const titles = cards.map((r) => r.data.title);
  const sawOfflineEdit = titles.includes('E2E offline card A (edited)');
  await e2e.report({
    step: 'session-ok',
    ok: true,
    cardCount: cards.length,
    sawOfflineEdit,
    indicator: indicatorLabel(),
  });
  if (!sawOfflineEdit)
    throw new Error('mirror missing the offline-created card');
}

function emptyTask(): any {
  return {
    id: 0,
    card_pk: 0,
    user_id: 0,
    title: '',
    scheduled_date: null,
    due_date: null,
    created_at: new Date(),
    updated_at: new Date(),
    completed_at: null,
    description: null,
    priority: null,
    status: 'todo',
    is_complete: false,
    is_deleted: false,
    reminder_time: null,
    reminder_sent: false,
    parent_task_id: null,
    sort_order: null,
  };
}

export async function runE2E(): Promise<void> {
  const e2e = getE2E();
  const started = Date.now();
  try {
    if (e2e.scenario === 'session') {
      await runSession(e2e);
    } else {
      await runFresh(e2e);
    }
    await e2e.done(true, {
      ok: true,
      scenario: e2e.scenario,
      ms: Date.now() - started,
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    await e2e.report({ step: 'error', ok: false, message });
    await e2e.done(false, {
      ok: false,
      scenario: e2e.scenario,
      ms: Date.now() - started,
      message,
    });
  }
}
